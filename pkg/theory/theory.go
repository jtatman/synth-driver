package theory

import (
	"fmt"
	"math/rand"
	"regexp"
	"runtime"
	"strings"
)

// NoteNames maps pitch class (0-11) to sharp note names.
var NoteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// FlatNoteNames maps pitch class (0-11) to flat-friendly names for display.
var FlatNoteNames = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}

// DefaultEngineProfile returns "v3" (Mac Ultra-Sparse Sketch Mode) on darwin,
// and "v2" (Refined Sparse + CC Modulation) on Linux and other platforms.
func DefaultEngineProfile() string {
	if runtime.GOOS == "darwin" {
		return "v3"
	}
	return "v2"
}

// NormalizeSlowBPM maps any genre/seed BPM into a relaxed, clear 80..95 BPM studio range by default.
func NormalizeSlowBPM(bpm int) int {
	if bpm >= 80 && bpm <= 95 {
		return bpm
	}
	if bpm <= 0 {
		return 86
	}
	return 80 + ((bpm * 7) % 16) // deterministically lands in 80..95 BPM
}

// ChordSpec represents a parsed chord with its harmonic function, intervals, and color extensions.
type ChordSpec struct {
	Symbol            string   `json:"symbol"`
	Roman             string   `json:"roman"`
	RootPC            int      `json:"root_pc"`            // 0=C .. 11=B
	RootName          string   `json:"root_name"`          // e.g. "A", "F#"
	Quality           string   `json:"quality"`            // min9, maj7#11, dom7b9, etc.
	Function          string   `json:"function"`           // Tonic, Subdominant, Dominant, Modal/Borrowed, Turnaround
	CoreIntervals     []int    `json:"core_intervals"`     // Semitones from root: [0, 3, 7, 10]
	ColorIntervals    []int    `json:"color_intervals"`    // Extensions: [2 (9th), 5 (11th), 6 (#11), 9 (13th)]
	ColorDescriptions []string `json:"color_descriptions"` // Human-readable color labels
	VoicingMIDI       []int    `json:"voicing_midi"`       // Smoothly voice-led 2-to-3 note pad voicing
	TransitionTip     string   `json:"transition_tip"`     // Musical explanation of how to exit this chord
}

// StepEvent represents a single step on a monophonic synthesizer (Crave, East Beast, Edge)
// including MIDI CC#1 (Mod Wheel) and CC#74 (Filter Cutoff) modulation values.
type StepEvent struct {
	Step       int    `json:"step"`        // 1-indexed step number (1..32)
	MIDINote   int    `json:"midi_note"`   // Standard MIDI note (e.g. 36 = C2, 48 = C3)
	NoteName   string `json:"note_name"`   // e.g. "A2", "F#3"
	ChordSym   string `json:"chord_sym"`   // Active chord symbol on this step
	Role       string `json:"role"`        // "Root", "3rd", "5th", "7th", "Color:9th", "Approach", "Pedal"
	Gate       int    `json:"gate"`        // Crave gate 0..7 (3=50%, 5=75%, 7=tie)
	Ratchet    int    `json:"ratchet"`     // Crave ratchet 0..3 (0=1x, 1=2x, 2=3x, 3=4x)
	Velocity   int    `json:"velocity"`    // 1..127
	Glide      bool   `json:"glide"`       // Portamento / slide
	Accent     bool   `json:"accent"`      // Filter/VCA accent
	Rest       bool   `json:"rest"`        // True if silent step
	ModCC1     int    `json:"mod_cc1"`     // MIDI CC#1 Modulation Wheel (0..127)
	CutoffCC74 int    `json:"cutoff_cc74"` // MIDI CC#74 Brightness / Filter Cutoff (0..127)
}

// SectionSlice represents a 32-step hardware-ready song section of the arrangement.
type SectionSlice struct {
	Name           string      `json:"name"`             // e.g., "1. Intro", "2. Build-Up", "3. Drop", "4. Breakdown", "5. Outro"
	Slug           string      `json:"slug"`             // e.g., "part_1_intro"
	Description    string      `json:"description"`      // Musical explanation of this song section
	Length         int         `json:"length"`           // 32 steps
	Swing          int         `json:"swing"`            // 50..75
	CraveSteps     []StepEvent `json:"crave_steps"`      // Ch 1: Sub/Bass Anchor Voice
	EastBeastSteps []StepEvent `json:"east_beast_steps"` // Ch 2: High/Midrange Harmonic Counter-Voice
	EdgeSteps      []StepEvent `json:"edge_steps"`       // Ch 3: Quarter-Note Beat Pulse (constant pitch)
}

// SongChainStage represents one stage in the automated Full Song Journey player.
type SongChainStage struct {
	StageIndex int    `json:"stage_index"`
	SliceIndex int    `json:"slice_index"`
	Label      string `json:"label"`
}

// DefaultSongJourneyChain returns the 9-stage song progression:
// Alternating Intro/Outro -> Build-Up -> Modulated Drop -> Simplified Breakdown -> Intro/Outro Ending.
func DefaultSongJourneyChain() []SongChainStage {
	return []SongChainStage{
		{StageIndex: 0, SliceIndex: 0, Label: "1/9: Intro (Opening Pulse)"},
		{StageIndex: 1, SliceIndex: 4, Label: "2/9: Outro (Hypnotic Answer)"},
		{StageIndex: 2, SliceIndex: 0, Label: "3/9: Intro (Deepen Groove)"},
		{StageIndex: 3, SliceIndex: 4, Label: "4/9: Outro (Breathing Space)"},
		{StageIndex: 4, SliceIndex: 1, Label: "5/9: Build-Up (+Approach Notes & Rising Mod)"},
		{StageIndex: 5, SliceIndex: 2, Label: "6/9: Drop (Peak CC#1/CC#74 Modulation)"},
		{StageIndex: 6, SliceIndex: 3, Label: "7/9: Breakdown (Lush Pedal Space)"},
		{StageIndex: 7, SliceIndex: 0, Label: "8/9: Intro (Return to Anchor)"},
		{StageIndex: 8, SliceIndex: 4, Label: "9/9: Outro (Final Wind-Down)"},
	}
}

// VoiceSettings holds custom octave offsets, MIDI channel assignments (0..15),
// and optional per-voice target MIDI Port IDs ("all" = broadcast to all active ports).
type VoiceSettings struct {
	CraveOctave     int `json:"crave_octave"`      // -3 .. +3 (Default -1 = Sub)
	EastBeastOctave int `json:"east_beast_octave"` // -3 .. +3 (Default +1 = High)
	EdgeOctave      int `json:"edge_octave"`       // -3 .. +3 (Default 0)
	PolyOctave      int `json:"poly_octave"`       // -3 .. +3 (Default 0)

	CraveChannel     int `json:"crave_channel"`      // 0..15 (default 0 = Ch 1)
	EastBeastChannel int `json:"east_beast_channel"` // 0..15 (default 1 = Ch 2)
	EdgeChannel      int `json:"edge_channel"`       // 0..15 (default 2 = Ch 3)
	PolyChannel      int `json:"poly_channel"`       // 0..15 (default 0 = Ch 1/Omni for Vital/Surge XT)

	PolyPortID      string `json:"poly_port_id"`       // "all" or specific port ID
	CravePortID     string `json:"crave_port_id"`      // "all" or specific port ID
	EastBeastPortID string `json:"east_beast_port_id"` // "all" or specific port ID
	EdgePortID      string `json:"edge_port_id"`       // "all" or specific port ID
}

func DefaultVoiceSettings() VoiceSettings {
	return VoiceSettings{
		CraveOctave:      -1, // Default -1 Oct (Sub)
		EastBeastOctave:  1,  // Default +1 Oct (High)
		EdgeOctave:       0,  // Default 0
		PolyOctave:       0,  // Default 0
		CraveChannel:     0,  // MIDI Ch 1
		EastBeastChannel: 1,  // MIDI Ch 2
		EdgeChannel:      2,  // MIDI Ch 3
		PolyChannel:      3,  // MIDI Ch 4 (Isolated for Surge XT + passed to Vital Omni-Safe port)
		PolyPortID:       "all",
		CravePortID:      "all",
		EastBeastPortID:  "all",
		EdgePortID:       "all",
	}
}

// Arrangement holds the complete expanded musical structure for a progression.
type Arrangement struct {
	ID              int              `json:"id"`
	Title           string           `json:"title"`
	Genre           string           `json:"genre"`
	RootKey         string           `json:"root_key"`
	Mode            string           `json:"mode"`
	BPM             int              `json:"bpm"`
	Profile         string           `json:"profile"` // "v1", "v2", or "v3"
	Complexity      float64          `json:"complexity"`
	ColorDensity    float64          `json:"color_density"`
	Tags            []string         `json:"tags"`
	TheoryNotes     string           `json:"theory_notes"`
	PatchSuggestion string           `json:"patch_suggestion"`
	Chords          []ChordSpec      `json:"chords"`
	Slices          []SectionSlice   `json:"slices"`
	SongChain       []SongChainStage `json:"song_chain"`
	VoiceConfig     VoiceSettings    `json:"voice_config"`
}

// ApplyVoiceSettings shifts each track's MIDI notes by its configured octave offset (-3..+3)
// and updates all note labels and MIDI channel assignments.
func ApplyVoiceSettings(arr Arrangement, vs VoiceSettings) Arrangement {
	arr.VoiceConfig = vs
	if len(arr.SongChain) == 0 {
		arr.SongChain = DefaultSongJourneyChain()
	}

	if vs.PolyOctave != 0 {
		shift := vs.PolyOctave * 12
		for i := range arr.Chords {
			voiced := make([]int, len(arr.Chords[i].VoicingMIDI))
			for j, m := range arr.Chords[i].VoicingMIDI {
				voiced[j] = clampMIDI(m + shift)
			}
			arr.Chords[i].VoicingMIDI = voiced
		}
	}

	for sIdx := range arr.Slices {
		if vs.CraveOctave != 0 {
			shift := vs.CraveOctave * 12
			for i := range arr.Slices[sIdx].CraveSteps {
				n := clampMIDI(arr.Slices[sIdx].CraveSteps[i].MIDINote + shift)
				arr.Slices[sIdx].CraveSteps[i].MIDINote = n
				arr.Slices[sIdx].CraveSteps[i].NoteName = MIDIToNoteName(n)
			}
		}
		if vs.EastBeastOctave != 0 {
			shift := vs.EastBeastOctave * 12
			for i := range arr.Slices[sIdx].EastBeastSteps {
				n := clampMIDI(arr.Slices[sIdx].EastBeastSteps[i].MIDINote + shift)
				arr.Slices[sIdx].EastBeastSteps[i].MIDINote = n
				arr.Slices[sIdx].EastBeastSteps[i].NoteName = MIDIToNoteName(n)
			}
		}
		if vs.EdgeOctave != 0 {
			shift := vs.EdgeOctave * 12
			for i := range arr.Slices[sIdx].EdgeSteps {
				n := clampMIDI(arr.Slices[sIdx].EdgeSteps[i].MIDINote + shift)
				arr.Slices[sIdx].EdgeSteps[i].MIDINote = n
				arr.Slices[sIdx].EdgeSteps[i].NoteName = MIDIToNoteName(n)
			}
		}
	}
	return arr
}

// ParsePitchClass converts a note name ("C", "F#", "Bb", etc.) to 0..11.
func ParsePitchClass(name string) int {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0
	}
	base := strings.ToUpper(name[:1])
	pc := 0
	switch base {
	case "C":
		pc = 0
	case "D":
		pc = 2
	case "E":
		pc = 4
	case "F":
		pc = 5
	case "G":
		pc = 7
	case "A":
		pc = 9
	case "B":
		pc = 11
	}
	rest := name[1:]
	for _, ch := range rest {
		if ch == '#' {
			pc = (pc + 1) % 12
		} else if ch == 'b' || ch == 'B' {
			pc = (pc + 11) % 12
		}
	}
	return pc
}

// MIDIToNoteName converts a MIDI note number (e.g., 60 -> "C4") to a string.
func MIDIToNoteName(midi int) string {
	if midi < 0 {
		midi = 0
	}
	pc := midi % 12
	oct := (midi / 12) - 1
	return fmt.Sprintf("%s%d", NoteNames[pc], oct)
}

var chordRegex = regexp.MustCompile(`^([A-G][#b]?)(.*)$`)

// ParseChordSymbol parses a rich chord symbol (like "Am9", "Fmaj7#11", "E7b9", "G7alt", "Dm11")
// into its core intervals and color extensions.
func ParseChordSymbol(sym, roman, functionHint, transitionTip string) ChordSpec {
	sym = strings.TrimSpace(sym)
	matches := chordRegex.FindStringSubmatch(sym)
	rootName := "C"
	qual := ""
	if len(matches) == 3 {
		rootName = matches[1]
		qual = matches[2]
	}
	rootPC := ParsePitchClass(rootName)

	core := []int{0, 4, 7} // Default major triad
	color := []int{}
	colorDesc := []string{}

	qLower := strings.ToLower(qual)

	isMinor := strings.HasPrefix(qLower, "m") && !strings.HasPrefix(qLower, "maj")
	isDim := strings.Contains(qLower, "dim") || strings.Contains(qLower, "m7b5")
	isAug := strings.Contains(qLower, "aug") || strings.Contains(qLower, "+")
	isSus4 := strings.Contains(qLower, "sus4") || strings.HasSuffix(qLower, "sus")
	isSus2 := strings.Contains(qLower, "sus2")

	if isDim {
		core = []int{0, 3, 6}
	} else if isMinor {
		core = []int{0, 3, 7}
	} else if isAug {
		core = []int{0, 4, 8}
	} else if isSus4 {
		core = []int{0, 5, 7}
		colorDesc = append(colorDesc, "sus4 tension")
	} else if isSus2 {
		core = []int{0, 2, 7}
		colorDesc = append(colorDesc, "sus2 open air")
	}

	if strings.Contains(qLower, "maj7") || strings.Contains(qLower, "maj9") || strings.Contains(qLower, "maj13") {
		core = append(core, 11)
		colorDesc = append(colorDesc, "maj7 shimmer")
	} else if strings.Contains(qLower, "dim7") {
		core = append(core, 9)
	} else if strings.Contains(qLower, "6") && !strings.Contains(qLower, "13") {
		core = append(core, 9)
		colorDesc = append(colorDesc, "6th Dorian/lofi color")
	} else if strings.Contains(qLower, "7") || strings.Contains(qLower, "9") || strings.Contains(qLower, "11") || strings.Contains(qLower, "13") || strings.Contains(qLower, "alt") {
		core = append(core, 10)
	}

	if strings.Contains(qLower, "b9") {
		color = append(color, 1)
		colorDesc = append(colorDesc, "b9 Phrygian/dark dominant bite")
	} else if strings.Contains(qLower, "#9") {
		color = append(color, 3)
		colorDesc = append(colorDesc, "#9 altered tension")
	} else if strings.Contains(qLower, "9") || strings.Contains(qLower, "11") || strings.Contains(qLower, "13") {
		color = append(color, 2)
		colorDesc = append(colorDesc, "9th panoramic warmth")
	}

	if strings.Contains(qLower, "#11") {
		color = append(color, 6)
		colorDesc = append(colorDesc, "#11 Lydian float")
	} else if strings.Contains(qLower, "11") {
		color = append(color, 5)
		colorDesc = append(colorDesc, "11th suspended modal depth")
	}

	if strings.Contains(qLower, "b13") {
		color = append(color, 8)
		colorDesc = append(colorDesc, "b13 bittersweet tension")
	} else if strings.Contains(qLower, "13") {
		color = append(color, 9)
		colorDesc = append(colorDesc, "13th extension")
	}

	if strings.Contains(qLower, "alt") {
		color = append(color, 1, 6, 8)
		colorDesc = append(colorDesc, "altered dominant (b9/#11/b13) pull")
	}

	if len(color) == 0 {
		if isMinor {
			color = append(color, 2, 5)
			colorDesc = append(colorDesc, "implied 9th/11th minor color")
		} else {
			color = append(color, 2, 11)
			colorDesc = append(colorDesc, "implied maj7/9th harmonic air")
		}
	}

	if qual == "" {
		qual = "maj"
	}

	return ChordSpec{
		Symbol:            sym,
		Roman:             roman,
		RootPC:            rootPC,
		RootName:          rootName,
		Quality:           qual,
		Function:          functionHint,
		CoreIntervals:     core,
		ColorIntervals:    color,
		ColorDescriptions: colorDesc,
		TransitionTip:     transitionTip,
	}
}

// VoiceLeadProgression computes clean, open 2-to-3 note pad voicings (Root + 5th/Guide + optional single color tone)
// so wavetable/pad synths (Vital, Surge XT) let their overtones and filters breathe without 5-note mud.
func VoiceLeadProgression(chords []ChordSpec) []ChordSpec {
	return VoiceLeadProgressionWithProfile(chords, DefaultEngineProfile(), 0.65)
}

// VoiceLeadProgressionWithProfile builds polyphonic pad voicings:
// - "v3" (Mac Sketch Mode): 2-note open dyads (Root + 5th or Guide 3rd/7th)
// - "v2" (Refined): 2 to 3 notes max (Root + 5th + 1 voice-led guide/color tone)
// - "v1" (Legacy): 4 to 5 notes
func VoiceLeadProgressionWithProfile(chords []ChordSpec, profile string, colorDensity float64) []ChordSpec {
	out := make([]ChordSpec, len(chords))
	prevUpper := 55 // G3 midrange anchor

	for i, ch := range chords {
		rootMIDI := 36 + ch.RootPC // C2..B2 pad root
		if rootMIDI > 45 {
			rootMIDI -= 12
		}

		thirdIv := 4
		if len(ch.CoreIntervals) > 1 {
			thirdIv = ch.CoreIntervals[1]
		}
		fifthIv := 7
		if len(ch.CoreIntervals) > 2 {
			fifthIv = ch.CoreIntervals[2]
		}

		colorIv := thirdIv
		if colorDensity > 0.55 && len(ch.ColorIntervals) > 0 {
			colorIv = ch.ColorIntervals[0]
		} else if len(ch.CoreIntervals) > 3 {
			colorIv = ch.CoreIntervals[3]
		}

		if profile == "v1" {
			// Legacy 5-note voicing
			intervals := append([]int{}, ch.CoreIntervals...)
			if len(ch.ColorIntervals) > 0 {
				intervals = append(intervals, ch.ColorIntervals[0])
			}
			voicing := []int{rootMIDI}
			for _, iv := range intervals {
				pc := (ch.RootPC + iv) % 12
				n := 48 + pc
				if n > 64 {
					n -= 12
				}
				voicing = append(voicing, n)
			}
			ch.VoicingMIDI = uniqueSortedInts(voicing)
			out[i] = ch
			continue
		}

		// For v2 and v3: Strictly 2 or 3 notes max!
		// Note 1: Deep Root anchor (lets pad overtones bloom naturally)
		// Note 2: Smoothly voice-led 3rd/color guide tone near prevUpper (48..60)
		guidePC := (ch.RootPC + colorIv) % 12
		guideNote := 48 + guidePC
		if guideNote-prevUpper > 6 {
			guideNote -= 12
		} else if prevUpper-guideNote > 6 {
			guideNote += 12
		}
		if guideNote < 46 {
			guideNote += 12
		}
		if guideNote > 62 {
			guideNote -= 12
		}
		prevUpper = guideNote

		if profile == "v3" || colorDensity <= 0.35 {
			// 2-note open dyad (Root + Guide/5th) — ultra-clean for Vital / Surge XT pads on Mac
			ch.VoicingMIDI = uniqueSortedInts([]int{rootMIDI, guideNote})
		} else {
			// 3-note open triad/shell (Root + 5th + Guide/Color tone) — max 3 notes!
			fifthNote := rootMIDI + fifthIv
			if fifthNote >= guideNote {
				fifthNote -= 12
			}
			if fifthNote <= rootMIDI {
				fifthNote = rootMIDI + 7
			}
			voicing := uniqueSortedInts([]int{rootMIDI, fifthNote, guideNote})
			if len(voicing) > 3 {
				voicing = voicing[:3]
			}
			ch.VoicingMIDI = voicing
		}
		out[i] = ch
	}
	return out
}

func uniqueSortedInts(in []int) []int {
	seen := map[int]bool{}
	var res []int
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			res = append(res, v)
		}
	}
	for i := 1; i < len(res); i++ {
		for j := i; j > 0 && res[j] < res[j-1]; j-- {
			res[j], res[j-1] = res[j-1], res[j]
		}
	}
	return res
}

// BuildArrangementSlices generates the 5-Part Song Structure using the default engine profile.
func BuildArrangementSlices(chords []ChordSpec, genre string, complexity, colorDensity float64, seed int64) []SectionSlice {
	return BuildArrangementSlicesWithProfile(chords, genre, complexity, colorDensity, seed, DefaultEngineProfile())
}

// BuildArrangementSlicesWithProfile generates the 5-Part Song Structure tailored to the selected profile:
//   - "v3": Mac Ultra-Sparse Sketch Mode (1 note every 4 beats, occasionally 2 on turnarounds + CC modulation)
//   - "v2": Refined Anti-Crowding Mode (Intro/Outro spaciousness + sparse Build/Drop with MIDI CC#1 & CC#74 sweeps)
//   - "v1": Legacy Dense 32-step patterns
func BuildArrangementSlicesWithProfile(chords []ChordSpec, genre string, complexity, colorDensity float64, seed int64, profile string) []SectionSlice {
	if len(chords) == 0 {
		return nil
	}
	if profile == "" {
		profile = DefaultEngineProfile()
	}
	rng := rand.New(rand.NewSource(seed))

	swing := 50
	gLower := strings.ToLower(genre)
	if strings.Contains(gLower, "lofi") || strings.Contains(gLower, "jazz") || strings.Contains(gLower, "deep house") {
		swing = 55
	}

	return []SectionSlice{
		buildSlice(chords, "1. Intro: Minimal Pulse & Bass", "part_1_intro",
			"Spacious modular entry: Edge sends 1 steady beat pulse per quarter-note while Crave plays a sparse root-fifth anchor and East Beast whispers on downbeats.",
			0, swing, complexity, colorDensity, profile, rng),
		buildSlice(chords, "2. Build-Up: Sparse Groove + Approach & Mod Ramp", "part_2_buildup",
			"Adds just 1–2 transition approach notes onto the Intro skeleton while ramping MIDI CC#1 (Mod Wheel) and CC#74 (Filter Cutoff) upward.",
			1, swing, complexity, colorDensity, profile, rng),
		buildSlice(chords, "3. Drop: Modulated Anthem (Sparse Notes + CC#1/CC#74 Sweep)", "part_3_drop",
			"Peak energy through timbre rather than note clutter: punchy, breathable bass/lead anchors driven by high MIDI CC#1 & CC#74 filter/distortion modulation.",
			2, swing, complexity, colorDensity, profile, rng),
		buildSlice(chords, "4. Breakdown: Lush Space & Modal Pedal", "part_4_breakdown",
			"Simplified breathing centerpiece: long tied notes separated by multi-beat silences around a shared modal pedal tone.",
			3, swing, complexity, colorDensity, profile, rng),
		buildSlice(chords, "5. Outro: Mellow Resolve & Wind-Down", "part_5_outro",
			"Warm modular resolve: strips away upper tensions, cools down CC modulation, and settles onto the tonic and fifth.",
			4, swing, complexity, colorDensity, profile, rng),
	}
}

func buildSlice(chords []ChordSpec, name, slug, desc string, sectionType int, swing int, complexity, colorDensity float64, profile string, rng *rand.Rand) SectionSlice {
	numSteps := 32
	stepsPerChord := numSteps / len(chords)
	if stepsPerChord < 1 {
		stepsPerChord = 1
	}

	crave := make([]StepEvent, numSteps)
	eastBeast := make([]StepEvent, numSteps)
	edge := make([]StepEvent, numSteps)

	pedalPC := (chords[0].RootPC + 7) % 12 // 5th of tonic as warm midrange pedal
	edgeConstantPitch := clampMIDI(36 + chords[0].RootPC)
	edgeConstantName := MIDIToNoteName(edgeConstantPitch)

	for stepIdx := 0; stepIdx < numSteps; stepIdx++ {
		chordIdx := (stepIdx / stepsPerChord) % len(chords)
		nextChordIdx := (chordIdx + 1) % len(chords)
		posInChord := stepIdx % stepsPerChord

		ch := chords[chordIdx]
		nextCh := chords[nextChordIdx]

		modCC1, cutoffCC74 := computeStepModulation(sectionType, stepIdx, numSteps)

		var cNote int
		var cRole string
		var cGate, cRatchet, cVel int
		var cGlide, cAccent, cRest bool

		var ebNote int
		var ebRole string
		var ebGate, ebRatchet, ebVel int
		var ebGlide, ebAccent, ebRest bool

		switch profile {
		case "v3":
			cNote, cRole, cGate, cRatchet, cVel, cGlide, cAccent, cRest = pickCraveStepV3Sketch(
				ch, nextCh, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, complexity,
			)
			ebNote, ebRole, ebGate, ebRatchet, ebVel, ebGlide, ebAccent, ebRest = pickEastBeastStepV3Sketch(
				ch, nextCh, cNote, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, colorDensity,
			)
		case "v1":
			cNote, cRole, cGate, cRatchet, cVel, cGlide, cAccent, cRest = pickCraveStepV1Legacy(
				ch, nextCh, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, complexity, colorDensity, rng,
			)
			ebNote, ebRole, ebGate, ebRatchet, ebVel, ebGlide, ebAccent, ebRest = pickEastBeastStepV1Legacy(
				ch, nextCh, cNote, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, complexity, colorDensity, rng,
			)
		default: // "v2" Refined Sparse + CC Modulation
			cNote, cRole, cGate, cRatchet, cVel, cGlide, cAccent, cRest = pickCraveStepV2Refined(
				ch, nextCh, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, complexity, colorDensity,
			)
			ebNote, ebRole, ebGate, ebRatchet, ebVel, ebGlide, ebAccent, ebRest = pickEastBeastStepV2Refined(
				ch, nextCh, cNote, posInChord, stepsPerChord, stepIdx, sectionType, pedalPC, complexity, colorDensity,
			)
		}

		crave[stepIdx] = StepEvent{
			Step:       stepIdx + 1,
			MIDINote:   cNote,
			NoteName:   MIDIToNoteName(cNote),
			ChordSym:   ch.Symbol,
			Role:       cRole,
			Gate:       cGate,
			Ratchet:    cRatchet,
			Velocity:   cVel,
			Glide:      cGlide,
			Accent:     cAccent,
			Rest:       cRest,
			ModCC1:     modCC1,
			CutoffCC74: cutoffCC74,
		}

		eastBeast[stepIdx] = StepEvent{
			Step:       stepIdx + 1,
			MIDINote:   ebNote,
			NoteName:   MIDIToNoteName(ebNote),
			ChordSym:   ch.Symbol,
			Role:       ebRole,
			Gate:       ebGate,
			Ratchet:    ebRatchet,
			Velocity:   ebVel,
			Glide:      ebGlide,
			Accent:     ebAccent,
			Rest:       ebRest,
			ModCC1:     modCC1,
			CutoffCC74: cutoffCC74,
		}

		// Behringer Edge: Single constant-pitch pulse once per beat (every 4 steps: steps 1, 5, 9, 13, 17, 21, 25, 29).
		// Never changes pitch so the Edge's internal 8-step pitch/velocity knobs work cleanly without mud!
		isBeatPulse := (stepIdx%4 == 0)
		// In Breakdown (sectionType == 3), half-time pulse (every 8 steps) for extra breathing room
		if sectionType == 3 {
			isBeatPulse = (stepIdx%8 == 0)
		}
		edge[stepIdx] = StepEvent{
			Step:       stepIdx + 1,
			MIDINote:   edgeConstantPitch,
			NoteName:   edgeConstantName,
			ChordSym:   ch.Symbol,
			Role:       "Beat Pulse",
			Gate:       2,
			Ratchet:    0,
			Velocity:   100,
			Glide:      false,
			Accent:     stepIdx%8 == 0,
			Rest:       !isBeatPulse,
			ModCC1:     modCC1,
			CutoffCC74: cutoffCC74,
		}
	}

	return SectionSlice{
		Name:           name,
		Slug:           slug,
		Description:    desc,
		Length:         numSteps,
		Swing:          swing,
		CraveSteps:     crave,
		EastBeastSteps: eastBeast,
		EdgeSteps:      edge,
	}
}

// computeStepModulation calculates MIDI CC#1 (Mod Wheel) and CC#74 (Filter Cutoff) values (0..127)
// so Build-Up and Drop create timbral distortion/filter movement without crowding notes.
func computeStepModulation(sectionType, stepIdx, numSteps int) (int, int) {
	progress := float64(stepIdx) / float64(numSteps-1)
	switch sectionType {
	case 0: // 1. Intro: Clean, warm low-mid filter
		return 12, 48
	case 1: // 2. Build-Up: Smoothly ramp CC#1 (24 -> 92) and CC#74 (52 -> 104) across the 32 steps
		cc1 := 24 + int(progress*68.0)
		cc74 := 52 + int(progress*52.0)
		return cc1, cc74
	case 2: // 3. Drop: High modulation energy with rhythmic filter movement (CC#1 85..122, CC#74 82..124)
		wave := (stepIdx % 8)
		boost := wave * 4
		cc1 := clampCC(90 + boost)
		cc74 := clampCC(86 + boost)
		if stepIdx >= 24 {
			cc1 = clampCC(110 + (stepIdx-24)*2)
			cc74 = clampCC(106 + (stepIdx-24)*2)
		}
		return cc1, cc74
	case 3: // 4. Breakdown: Deep, airy filter pad breathe
		cc1 := 20 + (stepIdx%8)*2
		cc74 := 44 + (stepIdx%8)*3
		return clampCC(cc1), clampCC(cc74)
	default: // 5. Outro: Cool down modulation back to calm tonic (28 -> 8)
		cc1 := 28 - int(progress*20.0)
		cc74 := 54 - int(progress*22.0)
		return clampCC(cc1), clampCC(cc74)
	}
}

func clampCC(v int) int {
	if v < 0 {
		return 0
	}
	if v > 127 {
		return 127
	}
	return v
}

// ============================================================================
// VERSION 3: MAC ULTRA-SPARSE SKETCH MODE (1 note per 4 steps, occasionally 2)
// ============================================================================

func pickCraveStepV3Sketch(ch, nextCh ChordSpec, pos, span, globalStep, sectionType, pedalPC int, complexity float64) (int, string, int, int, int, bool, bool, bool) {
	rootBase := 36 + ch.RootPC
	if rootBase > 45 {
		rootBase -= 12
	}
	fifthIv := 7
	if len(ch.CoreIntervals) > 2 {
		fifthIv = ch.CoreIntervals[2]
	}

	note := rootBase
	role := "Sketch Root"
	gate := 6
	ratchet := 0
	vel := 100
	glide := false
	accent := false
	rest := true

	switch sectionType {
	case 0: // 1. INTRO SKETCH: 1 note per 4 steps (Root on 0, 5th on 4)
		if pos == 0 {
			rest = false
			note = rootBase
			role = "Intro Root"
			accent = true
			vel = 108
			gate = 6
		} else if pos == 4 {
			rest = false
			note = rootBase + fifthIv
			role = "Sparse 5th"
			vel = 92
			gate = 5
		}
	case 1: // 2. BUILD-UP SKETCH: 1 note per 4 steps + occasional approach on step 6
		if pos == 0 {
			rest = false
			note = rootBase
			role = "Build Root"
			accent = true
			vel = 110
			gate = 6
		} else if pos == 4 {
			rest = false
			note = rootBase + fifthIv
			role = "Build 5th"
			gate = 5
		} else if pos == span-2 && globalStep%8 == 6 {
			rest = false
			nextRoot := 36 + nextCh.RootPC
			if nextRoot > 45 {
				nextRoot -= 12
			}
			note = nextRoot - 1
			role = "Sketch Approach -> " + nextCh.RootName
			glide = true
			gate = 6
			vel = 104
		}
	case 2: // 3. DROP SKETCH: Sparse punchy anchor (pos 0, pos 4, and turnaround pos 6) + heavy CC mod
		if pos == 0 {
			rest = false
			note = rootBase
			role = "Drop Sub Anchor"
			accent = true
			vel = 118
			gate = 6
		} else if pos == 4 {
			rest = false
			note = rootBase + 12
			role = "Drop Octave"
			accent = true
			vel = 112
			gate = 5
		} else if pos == 6 && complexity > 0.4 {
			rest = false
			note = rootBase + fifthIv
			role = "Drop 5th"
			glide = true
			vel = 105
			gate = 5
		}
	case 3: // 4. BREAKDOWN SKETCH: Ultra-spacious — only 1 long sustained note per 8-step bar!
		if pos == 0 {
			rest = false
			note = rootBase
			role = "Breakdown Root Hold"
			gate = 7
			vel = 96
		}
	default: // 5. OUTRO SKETCH: 1 note on pos 0, pedal on pos 4 in first half only
		if pos == 0 {
			rest = false
			note = rootBase
			role = "Outro Root"
			gate = 6
			vel = 94
		} else if pos == 4 && globalStep < 16 {
			rest = false
			note = 36 + pedalPC
			role = "Outro Pedal"
			glide = true
			gate = 5
			vel = 86
		}
	}

	return clampMIDI(note), role, gate, ratchet, vel, glide, accent, rest
}

func pickEastBeastStepV3Sketch(ch, nextCh ChordSpec, craveMIDI, pos, span, globalStep, sectionType, pedalPC int, colorDensity float64) (int, string, int, int, int, bool, bool, bool) {
	midBase := 48 + ch.RootPC
	if midBase > 55 {
		midBase -= 12
	}
	thirdIv := 4
	if len(ch.CoreIntervals) > 1 {
		thirdIv = ch.CoreIntervals[1]
	}
	seventhIv := 10
	if len(ch.CoreIntervals) > 3 {
		seventhIv = ch.CoreIntervals[3]
	}
	color1 := 2
	if len(ch.ColorIntervals) > 0 {
		color1 = ch.ColorIntervals[0]
	}

	note := midBase + thirdIv
	role := "Sketch Guide"
	gate := 6
	ratchet := 0
	vel := 88
	glide := false
	accent := false
	rest := true

	switch sectionType {
	case 0: // 1. INTRO SKETCH: Only 1 note every 8 steps (pos 2)
		if pos == 2 {
			rest = false
			note = midBase + thirdIv
			role = "Intro Guide 3rd"
			gate = 6
			vel = 84
		}
	case 1: // 2. BUILD-UP SKETCH: Counter-note on pos 2 and guide lead on pos 6
		if pos == 2 {
			rest = false
			note = midBase + thirdIv
			role = "Build 3rd"
			gate = 5
			vel = 90
		} else if pos == 6 {
			rest = false
			note = midBase + seventhIv
			role = "Build 7th Lead"
			glide = true
			gate = 6
			vel = 96
		}
	case 2: // 3. DROP SKETCH: Call-and-response offbeat color (pos 2 and pos 6 only, + pos 7 on turnaround)
		if pos == 2 {
			rest = false
			note = midBase + thirdIv
			role = "Drop 3rd Call"
			accent = true
			vel = 104
			gate = 5
		} else if pos == 6 {
			rest = false
			note = midBase + color1 + 12
			role = "Drop Color Response"
			glide = true
			vel = 100
			gate = 6
		}
	case 3: // 4. BREAKDOWN SKETCH: 1 lush color note on pos 4 (answering Crave on pos 0)
		if pos == 4 {
			rest = false
			note = midBase + color1 + 12
			role = "Breakdown Color 9th"
			glide = true
			gate = 7
			vel = 90
		}
	default: // 5. OUTRO SKETCH: 1 mellow pedal/3rd on pos 2
		if pos == 2 && globalStep%16 < 8 {
			rest = false
			note = midBase + thirdIv
			role = "Outro Warm 3rd"
			gate = 6
			vel = 82
		}
	}

	for note > 64 {
		note -= 12
	}
	for note < 43 {
		note += 12
	}
	return clampMIDI(note), role, gate, ratchet, vel, glide, accent, rest
}

// ============================================================================
// VERSION 2: REFINED SPARSE + CC MODULATION (No 32-step "rotary organ" clutter)
// ============================================================================

func pickCraveStepV2Refined(ch, nextCh ChordSpec, pos, span, globalStep, sectionType, pedalPC int, complexity, colorDensity float64) (int, string, int, int, int, bool, bool, bool) {
	rootBase := 36 + ch.RootPC
	if rootBase > 45 {
		rootBase -= 12
	}
	thirdIv := 4
	if len(ch.CoreIntervals) > 1 {
		thirdIv = ch.CoreIntervals[1]
	}
	fifthIv := 7
	if len(ch.CoreIntervals) > 2 {
		fifthIv = ch.CoreIntervals[2]
	}
	colorIv := 2
	if len(ch.ColorIntervals) > 0 {
		colorIv = ch.ColorIntervals[0]
	}

	note := rootBase
	role := "Root"
	gate := 5
	ratchet := 0
	vel := 98
	glide := false
	accent := false
	rest := true

	switch sectionType {
	case 0: // 1. INTRO: Spacious pulse (steps 0, 4, 6; 5 rests per 8-step bar)
		switch pos {
		case 0:
			rest = false
			note = rootBase
			role = "Intro Root"
			accent = true
			vel = 110
			gate = 5
		case 4:
			rest = false
			note = rootBase
			role = "Root Pulse"
			gate = 4
		case 6:
			rest = false
			note = rootBase + fifthIv
			role = "5th Pulse"
			gate = 4
		default:
			rest = true
			role = "Intro Space"
		}

	case 1: // 2. BUILD-UP: Intro skeleton + chromatic approach glide on step 7 (still 4 full rests per 8 steps!)
		switch pos {
		case 0:
			rest = false
			note = rootBase
			role = "Root Anchor"
			accent = true
			vel = 112
			gate = 5
		case 3:
			rest = false
			note = rootBase + fifthIv
			role = "Syncopated 5th"
			gate = 4
		case 4:
			rest = false
			note = rootBase
			role = "Root"
			gate = 4
		case 7:
			rest = false
			nextRoot := 36 + nextCh.RootPC
			if nextRoot > 45 {
				nextRoot -= 12
			}
			if nextRoot > rootBase {
				note = nextRoot - 1
			} else {
				note = nextRoot + 1
			}
			role = "Approach -> " + nextCh.RootName
			glide = true
			accent = true
			vel = 114
			gate = 6
		default:
			rest = true
			role = "Build Breathing Space"
		}

	case 2: // 3. DROP: Punchy groove with rests on 1, 3, 5 + CC#1/CC#74 filter distortion
		switch pos {
		case 0:
			rest = false
			note = rootBase
			role = "Drop Sub Root"
			accent = true
			vel = 120
			gate = 5
		case 2:
			rest = false
			note = rootBase + 12
			role = "Octave Lift"
			accent = true
			vel = 114
			gate = 4
		case 4:
			rest = false
			note = rootBase + fifthIv
			role = "5th Drive"
			vel = 108
			gate = 5
		case 6:
			rest = false
			note = rootBase + thirdIv
			role = "Drop 3rd"
			glide = true
			vel = 106
			gate = 5
		case 7:
			if globalStep%16 == 15 {
				rest = false
				nextRoot := 36 + nextCh.RootPC
				if nextRoot > 45 {
					nextRoot -= 12
				}
				note = nextRoot - 1
				role = "Turnaround Glide -> " + nextCh.RootName
				glide = true
				accent = true
				vel = 116
				gate = 6
			} else {
				rest = true
				role = "Drop Pocket"
			}
		default:
			rest = true
			role = "Drop Pocket"
		}

	case 3: // 4. BREAKDOWN: Simplified long sustains with silence in between (pos 0 and pos 4 only)
		switch pos {
		case 0:
			rest = false
			note = rootBase
			role = "Sustained Root"
			gate = 7
			vel = 100
		case 4:
			rest = false
			note = 36 + pedalPC
			role = "Modal Pedal"
			glide = true
			gate = 7
			vel = 92
		case 6:
			if globalStep >= 16 && colorDensity > 0.5 {
				rest = false
				note = rootBase + colorIv + 12
				role = "Sparse Color"
				glide = true
				gate = 6
				vel = 88
			} else {
				rest = true
				role = "Breakdown Air"
			}
		default:
			rest = true
			role = "Breakdown Air"
		}

	default: // 5. OUTRO: Mellow resolve & wind-down
		switch pos {
		case 0:
			rest = false
			note = rootBase
			role = "Outro Root"
			vel = 98
			gate = 6
		case 4:
			rest = false
			note = 36 + pedalPC
			role = "Tonic Fifth Pedal"
			glide = true
			gate = 6
			vel = 88
		case 6:
			if globalStep < 16 {
				rest = false
				note = rootBase + thirdIv
				role = "Warm 3rd"
				gate = 4
				vel = 84
			} else {
				rest = true
				role = "Outro Silence"
			}
		default:
			rest = true
			role = "Outro Space"
		}
	}

	return clampMIDI(note), role, gate, ratchet, vel, glide, accent, rest
}

func pickEastBeastStepV2Refined(ch, nextCh ChordSpec, craveMIDI, pos, span, globalStep, sectionType, pedalPC int, complexity, colorDensity float64) (int, string, int, int, int, bool, bool, bool) {
	midBase := 48 + ch.RootPC
	if midBase > 55 {
		midBase -= 12
	}
	thirdIv := 4
	if len(ch.CoreIntervals) > 1 {
		thirdIv = ch.CoreIntervals[1]
	}
	fifthIv := 7
	if len(ch.CoreIntervals) > 2 {
		fifthIv = ch.CoreIntervals[2]
	}
	seventhIv := 10
	if len(ch.CoreIntervals) > 3 {
		seventhIv = ch.CoreIntervals[3]
	}
	color1 := 2
	if len(ch.ColorIntervals) > 0 {
		color1 = ch.ColorIntervals[0]
	}

	note := midBase + thirdIv
	role := "Guide 3rd"
	gate := 5
	ratchet := 0
	vel := 90
	glide := false
	accent := false
	rest := true

	switch sectionType {
	case 0: // 1. INTRO: Very sparse — step 0 and step 4 in second half
		if pos == 0 {
			rest = false
			note = midBase + thirdIv
			role = "Intro Guide 3rd"
			gate = 6
			vel = 84
		} else if pos == 4 && globalStep >= 16 {
			rest = false
			note = midBase + seventhIv
			role = "Intro 7th"
			gate = 5
			vel = 80
		}

	case 1: // 2. BUILD-UP: Sparse counter-line (only steps 2, 5, 7 — leaves 5 steps of rest per bar!)
		switch pos {
		case 2:
			rest = false
			note = midBase + thirdIv
			role = "3rd Guide"
			gate = 5
			vel = 92
		case 5:
			rest = false
			note = midBase + color1 + 12
			role = "Color 9th Lift"
			glide = true
			gate = 5
			vel = 96
		case 7:
			if globalStep%16 == 7 || globalStep%16 == 15 {
				rest = false
				nextThird := (nextCh.RootPC + 4) % 12
				if len(nextCh.CoreIntervals) > 1 {
					nextThird = (nextCh.RootPC + nextCh.CoreIntervals[1]) % 12
				}
				note = 48 + nextThird
				role = "Guide Lead -> " + nextCh.Symbol
				glide = true
				accent = true
				gate = 6
				vel = 102
			}
		}

	case 2: // 3. DROP: Interlocking sparse motif (plays on steps 1, 3, 5 where Crave rests!)
		switch pos {
		case 1:
			rest = false
			note = midBase + thirdIv
			role = "Offbeat 3rd"
			accent = true
			gate = 4
			vel = 104
		case 3:
			rest = false
			note = midBase + fifthIv
			role = "Offbeat 5th"
			gate = 4
			vel = 98
		case 5:
			rest = false
			note = midBase + color1 + 12
			role = "Modulated Color"
			glide = true
			accent = true
			gate = 5
			vel = 106
		case 7:
			if globalStep == 15 || globalStep == 31 {
				rest = false
				note = midBase + seventhIv
				role = "Turnaround 7th"
				glide = true
				gate = 6
				vel = 102
			}
		}

	case 3: // 4. BREAKDOWN: Answers Crave in the open spaces (steps 2 and 6 only)
		switch pos {
		case 2:
			rest = false
			note = midBase + color1 + 12
			role = "Lush Color 9th"
			glide = true
			gate = 7
			vel = 94
		case 6:
			rest = false
			note = midBase + seventhIv
			role = "Warm 7th Breathe"
			gate = 6
			vel = 88
		}

	default: // 5. OUTRO: Mellow sparse counter-notes (steps 0 and 4)
		if pos == 0 {
			rest = false
			note = midBase + thirdIv
			role = "Mellow 3rd"
			gate = 6
			vel = 84
		} else if pos == 4 {
			rest = false
			note = 48 + pedalPC
			role = "Outro Pedal"
			glide = true
			gate = 6
			vel = 78
		}
	}

	for note > 64 {
		note -= 12
	}
	for note < 43 {
		note += 12
	}
	if !rest && note == craveMIDI {
		if note+7 <= 64 {
			note += 7
		} else {
			note += 12
		}
	}
	return clampMIDI(note), role, gate, ratchet, vel, glide, accent, rest
}

// ============================================================================
// VERSION 1: LEGACY DENSE PATTERNS (Preserved for comparison)
// ============================================================================

func pickCraveStepV1Legacy(ch, nextCh ChordSpec, pos, span, globalStep, sectionType, pedalPC int, complexity, colorDensity float64, rng *rand.Rand) (int, string, int, int, int, bool, bool, bool) {
	rootBase := 36 + ch.RootPC
	if rootBase > 45 {
		rootBase -= 12
	}
	thirdIv := 4
	if len(ch.CoreIntervals) > 1 {
		thirdIv = ch.CoreIntervals[1]
	}
	fifthIv := 7
	if len(ch.CoreIntervals) > 2 {
		fifthIv = ch.CoreIntervals[2]
	}
	seventhIv := 10
	if len(ch.CoreIntervals) > 3 {
		seventhIv = ch.CoreIntervals[3]
	}
	colorIv := 2
	if len(ch.ColorIntervals) > 0 {
		colorIv = ch.ColorIntervals[pos%len(ch.ColorIntervals)]
	}

	note := rootBase
	role := "Root"
	gate := 4
	ratchet := 0
	vel := 95
	glide := false
	accent := false
	rest := false

	switch sectionType {
	case 0:
		switch pos {
		case 0, 2, 4:
			note = rootBase
			role = "Intro Root"
		case 6:
			note = rootBase + fifthIv
			role = "5th Pulse"
		default:
			rest = true
			role = "Intro Space"
		}
	case 1:
		if pos == span-1 {
			nextRoot := 36 + nextCh.RootPC
			if nextRoot > 45 {
				nextRoot -= 12
			}
			note = nextRoot - 1
			role = "Approach -> " + nextCh.RootName
			glide = true
			gate = 7
			accent = true
			break
		}
		notes := []int{rootBase, rootBase + fifthIv, rootBase + 12, rootBase + colorIv + 12, rootBase, rootBase + thirdIv, rootBase + seventhIv}
		note = notes[pos%len(notes)]
		role = "Legacy Build"
	case 2:
		notes := []int{rootBase, rootBase + 12, rootBase, rootBase + 12, rootBase + fifthIv, rootBase + 12, rootBase + thirdIv + 12, rootBase + seventhIv}
		note = notes[pos%len(notes)]
		role = "Legacy Drop"
	case 3:
		notes := []int{rootBase, rootBase, rootBase + fifthIv, rootBase + fifthIv, 36 + pedalPC, 36 + pedalPC, rootBase + colorIv + 12, rootBase + seventhIv}
		note = notes[pos%len(notes)]
		role = "Legacy Breakdown"
		gate = 7
	default:
		if pos%2 == 0 {
			note = rootBase
			role = "Outro"
		} else {
			rest = true
			role = "Outro Space"
		}
	}
	return clampMIDI(note), role, gate, ratchet, vel, glide, accent, rest
}

func pickEastBeastStepV1Legacy(ch, nextCh ChordSpec, craveMIDI, pos, span, globalStep, sectionType, pedalPC int, complexity, colorDensity float64, rng *rand.Rand) (int, string, int, int, int, bool, bool, bool) {
	midBase := 48 + ch.RootPC
	if midBase > 55 {
		midBase -= 12
	}
	thirdIv := 4
	if len(ch.CoreIntervals) > 1 {
		thirdIv = ch.CoreIntervals[1]
	}
	fifthIv := 7
	if len(ch.CoreIntervals) > 2 {
		fifthIv = ch.CoreIntervals[2]
	}
	seventhIv := 10
	if len(ch.CoreIntervals) > 3 {
		seventhIv = ch.CoreIntervals[3]
	}
	color1 := 2
	if len(ch.ColorIntervals) > 0 {
		color1 = ch.ColorIntervals[0]
	}

	arpNotes := []int{
		midBase + thirdIv,
		midBase + fifthIv,
		midBase + seventhIv,
		midBase + color1 + 12,
		midBase + seventhIv,
		midBase + fifthIv,
		midBase + thirdIv,
		48 + pedalPC,
	}
	note := arpNotes[pos%len(arpNotes)]
	rest := (sectionType == 0 && pos != 0) || (sectionType == 4 && pos%4 != 0)
	for note > 64 {
		note -= 12
	}
	return clampMIDI(note), "Legacy Arp", 4, 0, 95, false, false, rest
}

func clampMIDI(n int) int {
	if n < 24 {
		return 24
	}
	if n > 84 {
		return 84
	}
	return n
}
