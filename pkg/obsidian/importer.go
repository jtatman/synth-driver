package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/james-see/imagination-engine/pkg/db"
	"github.com/james-see/imagination-engine/pkg/theory"
)

var (
	// Matches a standard chord symbol like Am, Am7, Fmaj7, Gsus4, Esus4, Fadd9, F/A, Bb, Eb, Db, etc.
	chordSymRe = regexp.MustCompile(`^[A-G][#b♯♭]?(?:maj|min|m|dim|aug|sus|add|alt|[0-9]|[#b][0-9])*(?:/[A-G][#b]?)?$`)

	// Format A: Zynthion style: **Am7** *i7* **D7** *IV7*
	zynthionPairRe = regexp.MustCompile(`\*\*([A-G][^*]+)\*\*\s*\*([iIvV]+[^*]*)\*`)

	// Format C: NameOnTheBeat style: ### 1\. The euphoric minor loop: i, VI, III, VII (Am, F, C, G)
	nameOnBeatRe = regexp.MustCompile(`^###\s*\d+\\?\.\s*([^:]+):\s*([^(]+)\(([^)]+)\)`)

	// Format D & E: Bold chord list in numbered item: 1. **Cmaj7 - Gmaj7 - Am7 - Fmaj7** or 1. **Am – F – C – G** — desc
	boldChordListRe = regexp.MustCompile(`^\d+\.\s*\*\*([^*]+)\*\*(?:\s*(?:—|--|-|\(([^)]+)\))\s*(.*))?$`)

	// Format F: Unison Roman numeral headings: ### #6. vi-ii-V-I
	unisonHeadingRe = regexp.MustCompile(`^###\s*#(\d+)\.\s*([IivV]+(?:-[IivV]+){2,4})\s*$`)

	// Splitting chord lists by dash, en-dash, em-dash, comma, or spaces
	chordSplitRe = regexp.MustCompile(`\s*(?:–|—|-|,|\s+)\s*`)
)

type frontmatter struct {
	Title       string
	Source      string
	Description string
	Tags        []string
}

// ImportDirectory scans an Obsidian vault or directory, filters for markdown files whose frontmatter
// tags include both "music" and "chords", extracts all chord progressions, and inserts them into SQLite.
func ImportDirectory(store *db.Store, rootPath string, requireMusicChordsTags bool) (int, error) {
	info, err := os.Stat(rootPath)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return importMarkdownFile(store, rootPath, false)
	}

	totalImported := 0
	err = filepath.WalkDir(rootPath, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir // skip .obsidian, .smart-env, .venv, .git
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			n, _ := importMarkdownFile(store, p, requireMusicChordsTags)
			totalImported += n
		}
		return nil
	})
	return totalImported, err
}

func importMarkdownFile(store *db.Store, filePath string, requireTags bool) (int, error) {
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	content := strings.ReplaceAll(string(rawBytes), "\r\n", "\n")

	fm, body := parseFrontmatter(content)
	if requireTags && (!hasTag(fm.Tags, "music") || !hasTag(fm.Tags, "chords")) {
		return 0, nil
	}

	genre := inferGenreFromNote(fm, filepath.Base(filePath))
	bpm := inferBPMFromGenre(genre)
	lines := strings.Split(body, "\n")

	imported := 0
	var currentSection string

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Check Format C: NameOnTheBeat heading: ### 1\. The euphoric minor loop: i, VI, III, VII (Am, F, C, G)
		if m := nameOnBeatRe.FindStringSubmatch(line); len(m) == 4 {
			subTitle := strings.TrimSpace(m[1])
			romans := splitTokens(m[2])
			chords := filterValidChords(splitTokens(m[3]))
			desc := collectParagraphs(lines, i+1, 2)
			if len(chords) >= 2 {
				if insertExtracted(store, fm, genre, bpm, subTitle, chords, romans, desc, filePath) {
					imported++
				}
				continue
			}
		}

		// Check Format F: Unison Roman-numeral heading: ### #6. vi-ii-V-I
		if m := unisonHeadingRe.FindStringSubmatch(line); len(m) == 3 {
			numStr := m[1]
			romanSeq := m[2]
			romans := strings.Split(romanSeq, "-")
			chords := realizeRomansInCOrAm(romans)
			desc := collectParagraphs(lines, i+1, 4)
			subTitle := fmt.Sprintf("Trance Archetype #%s (%s)", numStr, romanSeq)
			if len(chords) >= 3 {
				if insertExtracted(store, fm, genre, bpm, subTitle, chords, romans, desc, filePath) {
					imported++
				}
				continue
			}
		}

		// Track section headings (## ...)
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			heading = strings.ReplaceAll(heading, `\.`, `.`)
			currentSection = heading

			// Format B: Esencial style:
			// ## 1\. i–VI–III–VII: The Open, Anthemic Progression
			// followed within 6 lines by Roman line and Chord line (Am – F – C – G)
			romans, chords, desc, nextIdx := tryParseEsencialBlock(lines, i+1)
			if len(chords) >= 3 {
				if insertExtracted(store, fm, genre, bpm, heading, chords, romans, desc, filePath) {
					imported++
				}
				i = nextIdx
				continue
			}
			continue
		}

		// Check Format A: Zynthion style: **Am7** *i7* **D7** *IV7*
		if zMatches := zynthionPairRe.FindAllStringSubmatch(line, -1); len(zMatches) >= 2 {
			var chords, romans []string
			for _, zm := range zMatches {
				c := cleanChordToken(zm[1])
				if c != "" {
					chords = append(chords, c)
					romans = append(romans, strings.TrimSpace(zm[2]))
				}
			}
			if len(chords) >= 2 {
				desc := collectParagraphs(lines, i+1, 1)
				title := currentSection
				if title == "" {
					title = strings.Join(chords, " → ")
				}
				if insertExtracted(store, fm, genre, bpm, title, chords, romans, desc, filePath) {
					imported++
				}
				continue
			}
		}

		// Check Format D & E: 1. **Cmaj7 - Gmaj7 - Am7 - Fmaj7** or 1. **Am – F – C – G** — desc
		if m := boldChordListRe.FindStringSubmatch(line); len(m) >= 2 {
			chords := filterValidChords(splitTokens(m[1]))
			if len(chords) >= 3 {
				parenHint := strings.TrimSpace(m[2])
				inlineDesc := strings.TrimSpace(m[3])
				if inlineDesc == "" {
					inlineDesc = collectParagraphs(lines, i+1, 1)
				}
				if parenHint != "" {
					inlineDesc = "(" + parenHint + ") " + inlineDesc
				}
				subTitle := strings.Join(chords, " – ")
				if currentSection != "" && !strings.Contains(strings.ToLower(currentSection), "progressions") {
					subTitle = currentSection + ": " + subTitle
				}
				if insertExtracted(store, fm, genre, bpm, subTitle, chords, nil, inlineDesc, filePath) {
					imported++
				}
				continue
			}
		}
	}

	return imported, nil
}

func insertExtracted(store *db.Store, fm frontmatter, genre string, bpm int, subTitle string, rawChords, romans []string, desc, filePath string) bool {
	if len(rawChords) < 2 {
		return false
	}
	// If it's a 2-chord loop (like Deep House Am7 -> D7), expand it to a 4-bar phrase with color variation on bars 3-4!
	if len(rawChords) == 2 {
		rawChords = []string{rawChords[0], rawChords[1], enrichChordForModular(rawChords[0], 1), enrichChordForModular(rawChords[1], 3)}
		if len(romans) == 2 {
			romans = []string{romans[0], romans[1], romans[0] + "(9)", romans[1] + "(alt)"}
		}
	} else if len(rawChords) == 3 {
		// 3-chord cadence (e.g. F/A -> G/A -> Am): repeat tonic or turnaround on bar 4
		rawChords = append(rawChords, enrichChordForModular(rawChords[len(rawChords)-1], 3))
		if len(romans) == 3 {
			romans = append(romans, romans[2])
		}
	}

	specs := make([]theory.ChordSpec, len(rawChords))
	for i, c := range rawChords {
		baseChord, slashBass := splitSlashChord(c)
		enriched := enrichChordForModular(baseChord, i)
		r := ""
		if i < len(romans) {
			r = strings.TrimSpace(romans[i])
		}
		fn := fmt.Sprintf("Bar %d (%s)", i+1, c)
		tip := "Hold shared guide tones; step into next root on beat 4"
		if slashBass != "" {
			fn = fmt.Sprintf("Pedal Bass / %s over %s", baseChord, slashBass)
			tip = fmt.Sprintf("Keep %s anchored in the Crave bass while East Beast shifts to %s", slashBass, baseChord)
		} else if i == len(rawChords)-1 {
			tip = "Turnaround bar: inject chromatic approach or ratchet fill back to Bar 1"
		}
		specs[i] = theory.ParseChordSymbol(enriched, r, fn, tip)
	}

	rootKey := specs[0].RootName
	mode := "Aeolian / Modal"
	if strings.Contains(strings.ToLower(desc), "dorian") || strings.Contains(strings.ToLower(subTitle), "dorian") {
		mode = "Dorian"
	} else if strings.Contains(strings.ToLower(desc), "phrygian") {
		mode = "Phrygian / Modal Tension"
	} else if !strings.HasPrefix(strings.ToLower(specs[0].Quality), "m") || strings.HasPrefix(strings.ToLower(specs[0].Quality), "maj") {
		mode = "Ionian / Uplifting Major"
	}

	fullTitle := fmt.Sprintf("%s — %s", shortNoteName(fm.Title), subTitle)
	if len(fullTitle) > 68 {
		fullTitle = subTitle + " (" + shortNoteName(fm.Title) + ")"
	}
	if desc == "" {
		desc = fm.Description
	}

	tags := append([]string{"obsidian"}, fm.Tags...)
	arr := theory.Arrangement{
		Title:           fullTitle,
		Genre:           genre,
		RootKey:         rootKey,
		Mode:            mode,
		BPM:             bpm,
		Complexity:      0.68,
		ColorDensity:    0.84,
		Tags:            tags,
		TheoryNotes:     fmt.Sprintf("[Obsidian: %s] %s", fm.Title, strings.TrimSpace(desc)),
		PatchSuggestion: "Crave (Ch 1): Root + 5th + Step 8/16/24/32 chromatic glides. East Beast (Ch 2): 3rd/7th/9th upper voicing.",
		Chords:          theory.VoiceLeadProgression(specs),
	}

	_, err := store.InsertProgression(arr, "obsidian:"+filepath.Base(filePath))
	return err == nil
}

// enrichChordForModular preserves existing extensions (like Am7, Cmaj7, Gsus4, Fadd9)
// and gently upgrades plain triads (Am -> Am9, F -> Fmaj7#11, E -> E7b9) so they don't sound flat.
func enrichChordForModular(sym string, barIdx int) string {
	sym = strings.TrimSpace(sym)
	lower := strings.ToLower(sym)
	// Already has 7, 9, 11, 13, sus, dim, add
	if strings.ContainsAny(lower, "7913") || strings.Contains(lower, "sus") || strings.Contains(lower, "add") || strings.Contains(lower, "dim") {
		return sym
	}
	// Plain minor triad (e.g., Am, Dm, Em, Fm, Gm, Bm)
	if strings.HasSuffix(sym, "m") {
		if barIdx%2 == 1 {
			return sym + "11"
		}
		return sym + "9"
	}
	// Major triad on turnaround bar (often V in harmonic minor like E in A minor)
	if barIdx == 3 && (sym == "E" || sym == "B" || sym == "A" || sym == "D" || sym == "G") {
		return sym + "7sus4"
	}
	// Plain major triad -> maj7 or add9
	if barIdx%2 == 1 {
		return sym + "maj7"
	}
	return sym + "add9"
}

func splitSlashChord(sym string) (string, string) {
	parts := strings.Split(sym, "/")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return sym, ""
}

func tryParseEsencialBlock(lines []string, startIdx int) ([]string, []string, string, int) {
	var romans, chords []string
	endIdx := minInt(len(lines), startIdx+8)
	descStart := startIdx
	for j := startIdx; j < endIdx; j++ {
		l := strings.TrimSpace(lines[j])
		if l == "" || regexp.MustCompile(`^\d+$`).MatchString(l) {
			continue
		}
		if strings.HasPrefix(l, "#") {
			break
		}
		tokens := splitTokens(l)
		validChords := filterValidChords(tokens)
		if len(validChords) >= 3 && len(validChords) == len(tokens) {
			chords = validChords
			descStart = j + 1
			break
		}
		if isRomanLine(tokens) {
			romans = tokens
		}
	}
	if len(chords) >= 3 {
		desc := collectParagraphs(lines, descStart, 2)
		return romans, chords, desc, descStart
	}
	return nil, nil, "", startIdx
}

func isRomanLine(tokens []string) bool {
	if len(tokens) < 3 {
		return false
	}
	romanRe := regexp.MustCompile(`^[b#]?[iIvV]+(?:7|maj7|sus4|add9|dim)?$`)
	for _, t := range tokens {
		if !romanRe.MatchString(t) {
			return false
		}
	}
	return true
}

func realizeRomansInCOrAm(romans []string) []string {
	// Map Roman numerals to A minor / C major chords
	amMap := map[string]string{
		"I": "Cmaj7", "ii": "Dm9", "iii": "Em7", "III": "Cmaj7",
		"IV": "Fmaj7", "iv": "Dm9", "V": "G7sus4", "v": "Em7",
		"vi": "Am9", "VI": "Fmaj7", "VII": "G6", "i": "Am9",
	}
	var out []string
	for _, r := range romans {
		rClean := strings.TrimSpace(r)
		if ch, ok := amMap[rClean]; ok {
			out = append(out, ch)
		} else {
			out = append(out, "Am9")
		}
	}
	return out
}

func collectParagraphs(lines []string, startIdx, maxParas int) string {
	var paras []string
	for j := startIdx; j < len(lines) && len(paras) < maxParas; j++ {
		l := strings.TrimSpace(lines[j])
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "#") || strings.HasPrefix(l, "[Open in") || strings.HasPrefix(l, "![") || regexp.MustCompile(`^\d+\.\s*\*\*`).MatchString(l) {
			break
		}
		// Strip markdown links
		l = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(l, "$1")
		l = strings.ReplaceAll(l, "**", "")
		paras = append(paras, l)
	}
	return strings.Join(paras, " ")
}

func splitTokens(s string) []string {
	raw := chordSplitRe.Split(strings.TrimSpace(s), -1)
	var out []string
	for _, r := range raw {
		c := cleanChordToken(r)
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}

func filterValidChords(tokens []string) []string {
	var out []string
	for _, t := range tokens {
		c := cleanChordToken(t)
		if c != "" && chordSymRe.MatchString(c) {
			out = append(out, c)
		}
	}
	return out
}

func cleanChordToken(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "*_.,;()[]")
	s = strings.ReplaceAll(s, "♭", "b")
	s = strings.ReplaceAll(s, "♯", "#")
	return s
}

func parseFrontmatter(content string) (frontmatter, string) {
	var fm frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return fm, content
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return fm, content
	}
	fmBlock := content[4 : 4+end]
	body := content[4+end+4:]

	lines := strings.Split(fmBlock, "\n")
	inTags := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "title:") {
			fm.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "title:")), `"'`)
			inTags = false
		} else if strings.HasPrefix(trimmed, "source:") {
			fm.Source = strings.TrimSpace(strings.TrimPrefix(trimmed, "source:"))
			inTags = false
		} else if strings.HasPrefix(trimmed, "description:") {
			fm.Description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "description:")), `"'`)
			inTags = false
		} else if strings.HasPrefix(trimmed, "tags:") {
			inTags = true
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "tags:"))
			if rest != "" {
				for _, t := range strings.Split(strings.Trim(rest, "[]"), ",") {
					fm.Tags = append(fm.Tags, strings.ToLower(strings.TrimSpace(t)))
				}
			}
		} else if inTags && strings.HasPrefix(trimmed, "-") {
			tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			tag = strings.Trim(tag, `"'`)
			if tag != "" {
				fm.Tags = append(fm.Tags, tag)
			}
		} else if strings.Contains(trimmed, ":") && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
			inTags = false
		}
	}
	return fm, body
}

func hasTag(tags []string, target string) bool {
	target = strings.ToLower(target)
	for _, t := range tags {
		if strings.ToLower(t) == target {
			return true
		}
	}
	return false
}

func inferGenreFromNote(fm frontmatter, filename string) string {
	combined := strings.ToLower(fm.Title + " " + strings.Join(fm.Tags, " ") + " " + filename)
	switch {
	case strings.Contains(combined, "deep house") || strings.Contains(combined, "housemusic"):
		return "Deep House / Dorian"
	case strings.Contains(combined, "progressive house"):
		return "Progressive House & Trance"
	case strings.Contains(combined, "trance"):
		return "Trance"
	case strings.Contains(combined, "e minor"):
		return "Dark Electronic / E Minor"
	default:
		return "Obsidian Modular"
	}
}

func inferBPMFromGenre(genre string) int {
	g := strings.ToLower(genre)
	switch {
	case strings.Contains(g, "deep house"):
		return 122
	case strings.Contains(g, "progressive"):
		return 128
	case strings.Contains(g, "trance"):
		return 138
	default:
		return 130
	}
}

func shortNoteName(title string) string {
	switch {
	case strings.Contains(title, "Deep House"):
		return "Deep House 7ths"
	case strings.Contains(title, "7 Uplifting"):
		return "Uplifting Trance"
	case strings.Contains(title, "6 Euphoric"):
		return "Euphoric Trance"
	case strings.Contains(title, "20 Inspiring"):
		return "Prog House/Trance"
	case strings.Contains(title, "E Minor"):
		return "E Minor Electronic"
	case strings.Contains(title, "9 That Actually Work"):
		return "Armada/FSOE Trance"
	case strings.Contains(title, "Powerful & Exhilarating"):
		return "Classic Trance"
	default:
		if len(title) > 24 {
			return title[:24]
		}
		return title
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
