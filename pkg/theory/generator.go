package theory

import (
	"fmt"
	"math/rand"
	"strings"
)

// UIKeyNames matches the 12 key buttons in the Studio UI and CLI ("Eb", "Ab", "Bb" for flat keys).
var UIKeyNames = []string{"C", "C#", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}

// TransposeChord shifts a ChordSpec by semitones and returns a newly voice-led ChordSpec.
func TransposeChord(ch ChordSpec, semitones int) ChordSpec {
	newRootPC := (ch.RootPC + semitones%12 + 12) % 12
	newRootName := UIKeyNames[newRootPC]

	qual := ch.Quality
	if qual == "maj" {
		qual = ""
	}
	newSym := newRootName + qual
	return ParseChordSymbol(newSym, ch.Roman, ch.Function, ch.TransitionTip)
}

// TransposeArrangement transposes an entire Arrangement to a target key (e.g., "C", "G", "F#", "Bb")
// and rebuilds all polyphonic voicings, titles, transition tips, and hardware slices using arr.Profile.
func TransposeArrangement(arr Arrangement, targetKey string, seed int64) Arrangement {
	if arr.Profile == "" {
		arr.Profile = DefaultEngineProfile()
	}
	arr.BPM = NormalizeSlowBPM(arr.BPM)
	arr.SongChain = DefaultSongJourneyChain()

	// Determine the base pitch class from RootKey (or first chord root if RootKey is missing)
	oldPC := ParsePitchClass(arr.RootKey)
	if arr.RootKey == "" && len(arr.Chords) > 0 {
		oldPC = arr.Chords[0].RootPC
	}
	arr.RootKey = UIKeyNames[oldPC]

	targetKey = strings.TrimSpace(targetKey)
	if targetKey == "" {
		arr.Chords = VoiceLeadProgressionWithProfile(arr.Chords, arr.Profile, arr.ColorDensity)
		arr.Slices = BuildArrangementSlicesWithProfile(arr.Chords, arr.Genre, arr.Complexity, arr.ColorDensity, seed, arr.Profile)
		return arr
	}

	newPC := ParsePitchClass(targetKey)
	diff := (newPC - oldPC + 12) % 12

	if diff != 0 {
		newChords := make([]ChordSpec, len(arr.Chords))
		for i, ch := range arr.Chords {
			newChords[i] = TransposeChord(ch, diff)
		}
		// Update transition tips so they reflect the newly transposed chord symbols and roots
		for i := range newChords {
			nextCh := newChords[(i+1)%len(newChords)]
			newChords[i].TransitionTip = fmt.Sprintf("[%s -> %s] Root %s anchors bass while guide tones lead into %s (%s)",
				newChords[i].Symbol, nextCh.Symbol, newChords[i].RootName, nextCh.Symbol, nextCh.Function)
		}
		arr.Chords = newChords
		arr.RootKey = UIKeyNames[newPC]

		// If the title embeds the chord progression after "—", update it to show the transposed chords!
		if idx := strings.Index(arr.Title, "—"); idx != -1 {
			prefix := strings.TrimSpace(arr.Title[:idx])
			syms := make([]string, len(arr.Chords))
			for i, c := range arr.Chords {
				syms[i] = c.Symbol
			}
			arr.Title = prefix + " — " + strings.Join(syms, " – ")
		}
	}

	arr.Chords = VoiceLeadProgressionWithProfile(arr.Chords, arr.Profile, arr.ColorDensity)
	arr.Slices = BuildArrangementSlicesWithProfile(arr.Chords, arr.Genre, arr.Complexity, arr.ColorDensity, seed, arr.Profile)
	return arr
}

// GenerateHybridProgression synthesizes a brand-new, anti-slop progression by blending two genres
// or applying Neo-Riemannian / modal-interchange rules.
func GenerateHybridProgression(primaryGenre, secondaryInfluence, rootKey string, complexity, colorDensity float64, seed int64) Arrangement {
	return GenerateHybridProgressionWithProfile(primaryGenre, secondaryInfluence, rootKey, complexity, colorDensity, seed, DefaultEngineProfile())
}

// GenerateHybridProgressionWithProfile synthesizes a hybrid progression using the selected engine profile ("v1", "v2", "v3").
func GenerateHybridProgressionWithProfile(primaryGenre, secondaryInfluence, rootKey string, complexity, colorDensity float64, seed int64, profile string) Arrangement {
	if profile == "" {
		profile = DefaultEngineProfile()
	}
	rng := rand.New(rand.NewSource(seed))
	if rootKey == "" {
		rootKey = UIKeyNames[rng.Intn(12)]
	}
	rootPC := ParsePitchClass(rootKey)
	rootKey = UIKeyNames[rootPC]
	n := func(offset int) string {
		pc := (rootPC + offset + 12) % 12
		return UIKeyNames[pc]
	}

	type blueprint struct {
		title       string
		mode        string
		bpm         int
		chords      []ChordSpec
		theoryNotes string
		patchTip    string
	}

	pLower := strings.ToLower(primaryGenre + " " + secondaryInfluence)

	var bp blueprint
	switch {
	case strings.Contains(pLower, "squarepusher") || strings.Contains(pLower, "idm"):
		bp = blueprint{
			title: fmt.Sprintf("%s Dorian / Altered IDM Fusion", rootKey),
			mode:  "Dorian + Altered Dominant + Lydian bVI",
			bpm:   92,
			chords: []ChordSpec{
				ParseChordSymbol(n(0)+"m11", "i11", "Tonic Dorian Anchor", "Hold the 11th as a common tone while bass drops to bVI"),
				ParseChordSymbol(n(8)+"maj7#11", "bVImaj7#11", "Lydian Submediant Lift", "Use the #11 to glide chromatically into the ii chord"),
				ParseChordSymbol(n(2)+"m9", "ii9", "Dorian Brightness", "Stepwise bass walk into the tritone substitute"),
				ParseChordSymbol(n(1)+"7#9", "bII7#9", "Tritone Sub / Altered Break", "Modulated #9/#11 turnaround before resolving by half-step to root"),
			},
			theoryNotes: "Combines a warm Dorian minor-11th anchor with a borrowed bVImaj7#11 pad chord, then replaces the standard V7 turnaround with a bII7#9 tritone substitution that resolves down by a single semitone.",
			patchTip:    "Crave (-1 Oct Sub): Pulse wave w/ PWM, VCF cutoff at 40%, Resonance 60%. East Beast (+1 Oct High): Filter FM / CC#74 modulated guide tones.",
		}
	case strings.Contains(pLower, "dnb") || strings.Contains(pLower, "drum"):
		bp = blueprint{
			title: fmt.Sprintf("%s Liquid Atmospheric Horizon", rootKey),
			mode:  "Aeolian / Dorian Hybrid",
			bpm:   88,
			chords: []ChordSpec{
				ParseChordSymbol(n(0)+"m9", "im9", "Deep Liquid Tonic", "Let the 9th ring on East Beast while Crave holds the sub root"),
				ParseChordSymbol(n(5)+"m11", "iv11", "Subdominant Depth", "Keep the root of chord 1 as the 5th of iv11 (zero-motion upper voice)"),
				ParseChordSymbol(n(8)+"maj9", "bVImaj9", "Panoramic Lift", "Step up two semitones to bVII or drop into V7b9b13"),
				ParseChordSymbol(n(7)+"7b9", "V7b9", "Harmonic Minor Resolve", "Use the diminished guide-tone pull between Crave and East Beast to snap back to Bar 1"),
			},
			theoryNotes: "Classic LTJ Bukem / Calibre liquid architecture at half-time breathing speed (88 BPM): open 2-to-3 note pad shells let Vital/Surge XT overtones shine while Crave anchors the sub.",
			patchTip:    "Crave (-1 Oct Sub): Sawtooth sub foundation. East Beast (+1 Oct High): Triangle + slow LFO on filter cutoff playing sparse 9th/7th intervals.",
		}
	case strings.Contains(pLower, "lofi") || strings.Contains(pLower, "jazz"):
		bp = blueprint{
			title: fmt.Sprintf("%s Late-Night Neo-Soul & Modular Tape", rootKey),
			mode:  "Major / Borrowed Minor Mixture",
			bpm:   84,
			chords: []ChordSpec{
				ParseChordSymbol(n(2)+"m9", "ii9", "Warm Subdominant Entry", "Start away from the tonic so the loop never feels static"),
				ParseChordSymbol(n(7)+"13", "V13", "Lush Dominant", "Drop the 13th down a half-step to become the 9th of Imaj9"),
				ParseChordSymbol(n(0)+"maj9", "Imaj9", "Tonic Arrival", "Introduce a secondary dominant VI7b13 on the second half of the bar"),
				ParseChordSymbol(n(9)+"7b13", "VI7b13", "Secondary Dominant Turnaround", "b13 resolves smoothly down by semitone to the 9th of ii9"),
			},
			theoryNotes: "Avoids the 'stuck in a major scale' trap by starting on ii9, resolving briefly to Imaj9, and immediately pivoting through an altered secondary dominant (VI7b13) so the loop perpetually rolls forward.",
			patchTip:    "Set swing to 55%. Crave (-1 Oct Sub): gentle low-pass pluck. East Beast (+1 Oct High): soft band-pass lead playing the 3rd and 7th guide tones.",
		}
	default:
		bp = blueprint{
			title: fmt.Sprintf("%s Hypnotic Berlin-Trance Pedal Journey", rootKey),
			mode:  "Phrygian Dominant / Aeolian Modal Shift",
			bpm:   86,
			chords: []ChordSpec{
				ParseChordSymbol(n(0)+"m9", "im9", "Hypnotic Minor Center", "Anchor the 5th as an unbroken pedal tone across all 4 bars"),
				ParseChordSymbol(n(8)+"maj7#11", "bVImaj7#11", "Cosmic Expansion", "Keep the 5th pedal ringing—it becomes the major 7th of bVI!"),
				ParseChordSymbol(n(10)+"6", "bVII6", "Dorian/Aeolian Bridge", "Chromatic bass climb from bVI -> bVII -> bII"),
				ParseChordSymbol(n(1)+"maj7#11", "bIImaj7#11", "Neapolitan / Phrygian Resolve", "Slide down one semitone from bII root straight into the tonic downbeat"),
			},
			theoryNotes: "Solves modular monotony using a common-tone pedal point plus a bIImaj7#11 (Neapolitan Lydian) turnaround chord at Bar 4, using MIDI CC#1 & CC#74 filter sweeps on the Drop instead of note clutter.",
			patchTip:    "Crave (-1 Oct Sub): sparse sub anchor with portamento on turnarounds. Edge: steady 1-pulse-per-beat clock trigger at constant pitch.",
		}
	}

	chords := VoiceLeadProgressionWithProfile(bp.chords, profile, colorDensity)
	slices := BuildArrangementSlicesWithProfile(chords, primaryGenre+" / "+secondaryInfluence, complexity, colorDensity, seed, profile)

	return Arrangement{
		ID:              0,
		Title:           bp.title,
		Genre:           strings.TrimSpace(primaryGenre + " × " + secondaryInfluence),
		RootKey:         rootKey,
		Mode:            bp.mode,
		BPM:             NormalizeSlowBPM(bp.bpm),
		Profile:         profile,
		Complexity:      complexity,
		ColorDensity:    colorDensity,
		Tags:            []string{strings.ToLower(primaryGenre), strings.ToLower(secondaryInfluence), "hybrid", "anti-slop"},
		TheoryNotes:     bp.theoryNotes,
		PatchSuggestion: bp.patchTip,
		Chords:          chords,
		Slices:          slices,
		SongChain:       DefaultSongJourneyChain(),
	}
}
