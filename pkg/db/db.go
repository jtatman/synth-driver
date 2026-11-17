package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/james-see/imagination-engine/pkg/theory"
	_ "modernc.org/sqlite"
)

//go:embed obsidian_seed.json
var embeddedObsidianSeed []byte

type EmbeddedProgression struct {
	Title           string        `json:"title"`
	Genre           string        `json:"genre"`
	RootKey         string        `json:"root_key"`
	Mode            string        `json:"mode"`
	BPM             int           `json:"bpm"`
	Complexity      float64       `json:"complexity"`
	ColorDensity    float64       `json:"color_density"`
	Tags            []string      `json:"tags"`
	TheoryNotes     string        `json:"theory_notes"`
	PatchSuggestion string        `json:"patch_suggestion"`
	Chords          []StoredChord `json:"chords"`
	Source          string        `json:"source"`
}

type Store struct {
	db *sql.DB
}

type StoredChord struct {
	Symbol        string `json:"symbol"`
	Roman         string `json:"roman"`
	Function      string `json:"function"`
	TransitionTip string `json:"transition_tip"`
}

func Open(dbPath string) (*Store, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: conn}
	if err := s.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}
	if err := s.seedIfEmpty(); err != nil {
		conn.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS progressions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		genre TEXT NOT NULL,
		root_key TEXT NOT NULL,
		mode TEXT NOT NULL,
		bpm INTEGER NOT NULL,
		complexity REAL NOT NULL,
		color_density REAL NOT NULL,
		tags TEXT NOT NULL,
		theory_notes TEXT NOT NULL,
		patch_suggestion TEXT NOT NULL,
		chords_json TEXT NOT NULL,
		source TEXT DEFAULT 'seed'
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) Count() (int, error) {
	var c int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM progressions`).Scan(&c)
	return c, err
}

func (s *Store) InsertProgression(arr theory.Arrangement, source string) (int64, error) {
	stored := make([]StoredChord, len(arr.Chords))
	for i, c := range arr.Chords {
		stored[i] = StoredChord{
			Symbol:        c.Symbol,
			Roman:         c.Roman,
			Function:      c.Function,
			TransitionTip: c.TransitionTip,
		}
	}
	b, err := json.Marshal(stored)
	if err != nil {
		return 0, err
	}

	// Idempotency check: avoid inserting duplicate title + chords_json
	var existingID int64
	err = s.db.QueryRow(`SELECT id FROM progressions WHERE title = ? AND chords_json = ? LIMIT 1`, arr.Title, string(b)).Scan(&existingID)
	if err == nil && existingID > 0 {
		return existingID, nil
	}

	tagsStr := strings.Join(arr.Tags, ",")
	res, err := s.db.Exec(`
		INSERT INTO progressions (title, genre, root_key, mode, bpm, complexity, color_density, tags, theory_notes, patch_suggestion, chords_json, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, arr.Title, arr.Genre, arr.RootKey, arr.Mode, arr.BPM, arr.Complexity, arr.ColorDensity, tagsStr, arr.TheoryNotes, arr.PatchSuggestion, string(b), source)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListProgressions(genreFilter, searchQuery string) ([]theory.Arrangement, error) {
	query := `SELECT id, title, genre, root_key, mode, bpm, complexity, color_density, tags, theory_notes, patch_suggestion, chords_json FROM progressions WHERE 1=1`
	var args []any

	if genreFilter != "" && strings.ToLower(genreFilter) != "all" {
		query += ` AND (LOWER(genre) LIKE ? OR LOWER(tags) LIKE ?)`
		pat := "%" + strings.ToLower(genreFilter) + "%"
		args = append(args, pat, pat)
	}
	if searchQuery != "" {
		query += ` AND (LOWER(title) LIKE ? OR LOWER(theory_notes) LIKE ? OR LOWER(tags) LIKE ? OR LOWER(chords_json) LIKE ?)`
		pat := "%" + strings.ToLower(searchQuery) + "%"
		args = append(args, pat, pat, pat, pat)
	}
	query += ` ORDER BY id ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []theory.Arrangement
	for rows.Next() {
		var arr theory.Arrangement
		var tagsStr, chordsJSON string
		if err := rows.Scan(&arr.ID, &arr.Title, &arr.Genre, &arr.RootKey, &arr.Mode, &arr.BPM, &arr.Complexity, &arr.ColorDensity, &tagsStr, &arr.TheoryNotes, &arr.PatchSuggestion, &chordsJSON); err != nil {
			return nil, err
		}
		if tagsStr != "" {
			arr.Tags = strings.Split(tagsStr, ",")
		}
		arr.BPM = theory.NormalizeSlowBPM(arr.BPM)
		arr.Profile = theory.DefaultEngineProfile()
		arr.SongChain = theory.DefaultSongJourneyChain()
		var sc []StoredChord
		if err := json.Unmarshal([]byte(chordsJSON), &sc); err == nil {
			specs := make([]theory.ChordSpec, len(sc))
			for i, item := range sc {
				specs[i] = theory.ParseChordSymbol(item.Symbol, item.Roman, item.Function, item.TransitionTip)
			}
			arr.Chords = theory.VoiceLeadProgressionWithProfile(specs, arr.Profile, arr.ColorDensity)
			arr.Slices = theory.BuildArrangementSlicesWithProfile(arr.Chords, arr.Genre, arr.Complexity, arr.ColorDensity, int64(arr.ID*101+7), arr.Profile)
		}
		result = append(result, arr)
	}
	return result, nil
}

func (s *Store) GetProgression(id int) (theory.Arrangement, error) {
	all, err := s.ListProgressions("", "")
	if err != nil {
		return theory.Arrangement{}, err
	}
	for _, a := range all {
		if a.ID == id {
			return a, nil
		}
	}
	if len(all) > 0 {
		return all[0], nil
	}
	return theory.Arrangement{}, fmt.Errorf("progression %d not found", id)
}

func (s *Store) seedIfEmpty() error {
	count, err := s.Count()
	if err != nil || count > 0 {
		return err
	}

	seeds := []theory.Arrangement{
		{
			Title:           "Hypnotic Dorian Pedal & Phrygian Turnaround",
			Genre:           "Trance",
			RootKey:         "F#",
			Mode:            "Dorian / Phrygian Mixture",
			BPM:             136,
			Complexity:      0.55,
			ColorDensity:    0.78,
			Tags:            []string{"trance", "hypnotic", "pedal-point", "dorian", "modular"},
			TheoryNotes:     "Instead of plain minor triads, this keeps a high C#5 (5th) pedal ringing across all 4 bars. Bar 2 lifts to the Dorian IV9 (B9), Bar 3 floats on Dmaj7#11, and Bar 4 borrows the Phrygian bIImaj7 (Gmaj7#11) to slide down by one half-step into F#m9.",
			PatchSuggestion: "Crave: Sawtooth, Cutoff 35%, Env Mod 60%, Decay 45%, Glide ON for steps 8/16/24/32. East Beast: Square + Sub playing the C# pedal and upper 9th/3rd guide tones.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("F#m9", "im9", "Tonic Dorian Anchor", "Keep C# (5th) ringing while sliding bass up a 4th to B"),
				theory.ParseChordSymbol("B9", "IV9", "Dorian Characteristic Lift", "The D# (major 6th of F#m) breaks the standard sad minor scale instantly"),
				theory.ParseChordSymbol("Dmaj7#11", "bVImaj7#11", "Lydian Submediant Float", "Common tones F# and C# stay frozen while bass drops to D"),
				theory.ParseChordSymbol("Gmaj7#11", "bIImaj7#11", "Phrygian Neapolitan Resolve", "Slide down 1 semitone from G -> F# with 2x ratchet on step 32"),
			},
		},
		{
			Title:           "LTJ / Calibre Atmospheric Liquid Horizon",
			Genre:           "Drum & Bass",
			RootKey:         "C",
			Mode:            "Aeolian / Altered Dominant",
			BPM:             174,
			Complexity:      0.65,
			ColorDensity:    0.90,
			Tags:            []string{"dnb", "liquid", "atmospheric", "jazz-chords", "9th-chords"},
			TheoryNotes:     "Built around lush minor-9th and major-9th voicings where the top voice barely moves. Bar 4 uses a G7b9b13 (altered dominant) to inject dark jazz tension before melting back into Cm9.",
			PatchSuggestion: "Crave: Low-pass sub/bass voice on Ch 1 playing roots + 7th leaps. East Beast: Ch 2 playing warm Rhodes-like filtered triangle 3rds & 9ths. Edge: 174 BPM syncopated break percussion.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Cm9", "im9", "Liquid Minor Tonic", "Glide from D (9th) down to C before dropping bass to Fm11"),
				theory.ParseChordSymbol("Fm11", "iv11", "Deep Subdominant Cushion", "Bb (11th of Fm) prepares the Abmaj9 in the next bar"),
				theory.ParseChordSymbol("Abmaj7#11", "bVImaj7#11", "Panoramic Atmospheric Pad", "D natural (#11) gives that weightless sci-fi jungle shimmer"),
				theory.ParseChordSymbol("G7alt", "V7(b9,b13)", "Altered Jazz Turnaround", "Ab (b9) and Eb (b13) resolve inward by semitone to G and D of Cm9"),
			},
		},
		{
			Title:           "Squarepusher Acid-Jazz Fusion & Ratchet Break",
			Genre:           "IDM / Squarepusher",
			RootKey:         "E",
			Mode:            "Dorian / Chromatic Mediant / Lydian Dominant",
			BPM:             168,
			Complexity:      0.92,
			ColorDensity:    0.95,
			Tags:            []string{"squarepusher", "idm", "acid-jazz", "ratchets", "fusion"},
			TheoryNotes:     "Inspired by Tom Jenkinson's fretless bass + 303 + breakcore writing: opens on Em11, jumps by chromatic mediant to Cmaj9#11, pivots through F#m7b5, and detonates into a B7#9b13 turnaround with 3x-4x ratchets and wide 2-octave leaps.",
			PatchSuggestion: "Crave: Resonance 75% (acid squelch), Ratchets active on Part C, patch Velocity -> VCF Cutoff. East Beast: Fast arpeggiated counter-line on upper extensions (#9, #11, 13).",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Em11", "im11", "Fusion Tonic Groove", "Use wide 10th intervals (E2 -> G3 -> F#4) with syncopated rests"),
				theory.ParseChordSymbol("Cmaj7#11", "bVImaj7#11", "Lydian Mediant Contrast", "Play F# (#11) against C bass for instant Squarepusher harmonic bite"),
				theory.ParseChordSymbol("F#m7b5", "iiø7", "Half-Diminished Pivot", "C natural stays held from Cmaj7#11 as the b5 of F#m7b5!"),
				theory.ParseChordSymbol("B7#9", "V7#9", "Hendrix/Altered Break Resolve", "4x ratchet burst on D natural (#9) + D# (3rd) cross-relation"),
			},
		},
		{
			Title:           "Esoteric Modular Hexatonic Pole Journey (Neo-Riemannian)",
			Genre:           "Esoteric Modular",
			RootKey:         "A",
			Mode:            "Hexatonic / Neo-Riemannian PLP Cycle",
			BPM:             118,
			Complexity:      0.60,
			ColorDensity:    0.85,
			Tags:            []string{"modular", "esoteric", "berlin-school", "neo-riemannian", "evolving"},
			TheoryNotes:     "Breaks out of diatonic scales completely using Neo-Riemannian voice leading: each chord moves only 1 or 2 voices by a single semitone (Am9 -> Fmaj7#11 -> Fm9 -> C#maj7 -> Am9), creating a cinematic, endlessly evolving modular journey.",
			PatchSuggestion: "Long gate lengths (5-7) with slow LFO modulating Crave pulse width and East Beast wavefolder. Let the Edge run a 7-step polymeter against the 32-step harmonic cycle.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Am9", "im9", "Home Center", "Move E up 1 semitone to F while keeping A and C stationary"),
				theory.ParseChordSymbol("Fmaj7#11", "bVImaj7#11", "Submediant Major", "Transform F major to F minor (Modal Mixture): A drops 1 semitone to Ab!"),
				theory.ParseChordSymbol("Fm9", "bvi9", "Dark Borrowed MinorSubmediant", "Ab and C stay held while F drops 1 semitone to E (or C#maj7)"),
				theory.ParseChordSymbol("Dbmaj7#11", "bIVmaj7#11", "Hexatonic Pole Arrival", "F and C stay common tones as Db slides back up to A minor"),
			},
		},
		{
			Title:           "Late-Night Lo-Fi Guitar & Tape Tritone Walk",
			Genre:           "Lo-Fi / Neo-Soul",
			RootKey:         "D",
			Mode:            "Major / Chromatic Tritone Sub",
			BPM:             84,
			Complexity:      0.50,
			ColorDensity:    0.88,
			Tags:            []string{"lofi", "guitar", "neo-soul", "tritone-sub", "chill"},
			TheoryNotes:     "Translates jazz-guitar shell+extension voicings into synth voices: Em9 -> Eb7#9 (tritone sub) -> Dmaj9 -> B7b13. Notice the bassline walks chromatically down: E -> Eb -> D -> B!",
			PatchSuggestion: "Swing 57%. Crave plays the smooth chromatic bassline (E -> Eb -> D -> B) with glide on step 7-8; East Beast plays the warm upper 3rd/7th/9th chord shells.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Em9", "ii9", "Warm Subdominant Start", "Bass steps down 1 semitone from E to Eb"),
				theory.ParseChordSymbol("Eb7#9", "bII7#9", "Tritone Substitution", "Replaces A7; F# and C# (or G/Db) slide down by half-step into Dmaj9"),
				theory.ParseChordSymbol("Dmaj9", "Imaj9", "Sunlight Tonic Arrival", "Hold F# as the 5th of the upcoming B7b13 turnaround"),
				theory.ParseChordSymbol("B7b13", "VI7b13", "Secondary Dominant to ii", "G natural (b13) resolves smoothly to F# (9th of Em9)"),
			},
		},
		{
			Title:           "Berlin School Phrygian-Dominant Sequencer Odyssey",
			Genre:           "Trance",
			RootKey:         "D",
			Mode:            "Phrygian Dominant / Harmonic Minor",
			BPM:             130,
			Complexity:      0.70,
			ColorDensity:    0.80,
			Tags:            []string{"trance", "berlin-school", "phrygian", "psy", "sequencer"},
			TheoryNotes:     "Uses the exotic tension between F# (major 3rd of D) and Eb (flat 2nd) to create hypnotic Goa/Berlin-School modular energy without ever sounding like a generic major/minor loop.",
			PatchSuggestion: "Crave: 16th-note ratcheted sequence with Accent patched to VCF Cutoff. East Beast: slow sweeping band-pass drone + counter-melody on the exotic b2 and maj3 intervals.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("D7b9", "I7b9", "Phrygian Dominant Center", "Emphasize the D - F# - Eb interval triangle"),
				theory.ParseChordSymbol("Gm9", "iv9", "Dark Subdominant Release", "Resolves the F# up to G while keeping D as a common-tone 5th"),
				theory.ParseChordSymbol("Ebmaj7#11", "bIImaj7#11", "Exotic Phrygian Lift", "A natural (#11) links directly back to the D tonic"),
				theory.ParseChordSymbol("Cm11", "bvii11", "Modal Subtonic Bridge", "Walk bass C -> C# -> D into Bar 1"),
			},
		},
		{
			Title:           "Quartal Poly-Rhythmic Modular Drift",
			Genre:           "Esoteric Modular",
			RootKey:         "G",
			Mode:            "Dorian Quartal (Stacked 4ths)",
			BPM:             124,
			Complexity:      0.75,
			ColorDensity:    0.85,
			Tags:            []string{"modular", "quartal", "esoteric", "dorian", "west-coast"},
			TheoryNotes:     "Built from stacked 4ths (sus4/11th voicings) like McCoy Tyner and Buchla/East-Beast modular patches: ambiguous, open, and immune to 'happy major scale' fatigue.",
			PatchSuggestion: "East Beast wavefolder + FM at 25% playing stacked 4th intervals (G-C-F-Bb) while Crave anchors the shifting modal bass roots.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Gm11", "im11", "Quartal Dorian Stack", "Voice in 4ths: G - C - F - Bb - D"),
				theory.ParseChordSymbol("F6", "bVII6", "Whole-Step Subtonic Shift", "Shift the entire quartal stack down a whole step"),
				theory.ParseChordSymbol("Ebmaj7#11", "bVImaj7#11", "Lydian Warmth", "Keep D and G ringing across the bass change"),
				theory.ParseChordSymbol("Am11", "ii11", "Dorian Supertonic Turnaround", "Stepwise climb A -> Bb -> G back to the root"),
			},
		},
		{
			Title:           "Photek / Source Direct Dark Jazz Jungle",
			Genre:           "Drum & Bass",
			RootKey:         "F",
			Mode:            "Minor / Diminished Passing / Lydian bIII",
			BPM:             170,
			Complexity:      0.85,
			ColorDensity:    0.92,
			Tags:            []string{"dnb", "jungle", "dark-jazz", "photek", "breaks"},
			TheoryNotes:     "Moody 90s sci-fi jungle harmony: alternates between deep Fm11 and Abmaj9#11 before dropping into a tense E°7 / C7b9 turnaround with syncopated bass drops.",
			PatchSuggestion: "Crave: Deep sine-filtered triangle/saw sub with long portamento glides. East Beast: metallic FM-tinged chord stabs on offbeats. Edge: fast 170 BPM polyrhythmic hats/toms.",
			Chords: []theory.ChordSpec{
				theory.ParseChordSymbol("Fm11", "im11", "Noir Minor Foundation", "Long sub sustain on F1 with upper Bb/G tension"),
				theory.ParseChordSymbol("Abmaj7#11", "bIIImaj7#11", "Sci-Fi Relative Major", "D natural (#11) prevents relative major from sounding happy"),
				theory.ParseChordSymbol("Dbmaj9", "bVImaj9", "Deep Submediant Descent", "Smooth voice-leading: C and Eb stay fixed"),
				theory.ParseChordSymbol("C7b9", "V7b9", "Diminished Pull", "Db (b9) and E (3rd) create a tritone-diminished claw back to Fm"),
			},
		},
	}

	for _, seed := range seeds {
		if _, err := s.InsertProgression(seed, "seed"); err != nil {
			return err
		}
	}

	// Also seed embedded Obsidian progressions (so standalone macOS / go install builds include them automatically)
	var embedded []EmbeddedProgression
	if len(embeddedObsidianSeed) > 0 && json.Unmarshal(embeddedObsidianSeed, &embedded) == nil {
		for _, ep := range embedded {
			specs := make([]theory.ChordSpec, len(ep.Chords))
			for i, sc := range ep.Chords {
				specs[i] = theory.ParseChordSymbol(sc.Symbol, sc.Roman, sc.Function, sc.TransitionTip)
			}
			arr := theory.Arrangement{
				Title:           ep.Title,
				Genre:           ep.Genre,
				RootKey:         ep.RootKey,
				Mode:            ep.Mode,
				BPM:             ep.BPM,
				Complexity:      ep.Complexity,
				ColorDensity:    ep.ColorDensity,
				Tags:            ep.Tags,
				TheoryNotes:     ep.TheoryNotes,
				PatchSuggestion: ep.PatchSuggestion,
				Chords:          specs,
			}
			src := ep.Source
			if src == "" {
				src = "obsidian-embedded"
			}
			_, _ = s.InsertProgression(arr, src)
		}
	}
	return nil
}

// ExportObsidianSeedJSON writes all non-seed (Obsidian-imported) progressions to a JSON file
// so they can be embedded into the Go binary via //go:embed for portable macOS/Linux builds.
func (s *Store) ExportObsidianSeedJSON(outPath string) (int, error) {
	rows, err := s.db.Query(`SELECT title, genre, root_key, mode, bpm, complexity, color_density, tags, theory_notes, patch_suggestion, chords_json, source FROM progressions WHERE source != 'seed' ORDER BY id ASC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var list []EmbeddedProgression
	for rows.Next() {
		var ep EmbeddedProgression
		var tagsStr, chordsJSON string
		if err := rows.Scan(&ep.Title, &ep.Genre, &ep.RootKey, &ep.Mode, &ep.BPM, &ep.Complexity, &ep.ColorDensity, &tagsStr, &ep.TheoryNotes, &ep.PatchSuggestion, &chordsJSON, &ep.Source); err != nil {
			return 0, err
		}
		if tagsStr != "" {
			ep.Tags = strings.Split(tagsStr, ",")
		}
		_ = json.Unmarshal([]byte(chordsJSON), &ep.Chords)
		list = append(list, ep)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(outPath, b, 0644); err != nil {
		return 0, err
	}
	return len(list), nil
}
