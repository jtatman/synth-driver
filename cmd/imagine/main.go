package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/james-see/imagination-engine/pkg/db"
	"github.com/james-see/imagination-engine/pkg/midiout"
	"github.com/james-see/imagination-engine/pkg/obsidian"
	"github.com/james-see/imagination-engine/pkg/seq"
	"github.com/james-see/imagination-engine/pkg/theory"
	"github.com/james-see/imagination-engine/pkg/web"
)

func main() {
	dbPath := envOr("IMAGINE_DB", "imagination.db")

	// If launched with no args, or "play" / "tui", or a song number directly (e.g. "./imagine 9" or "./imagine 9 Bb"),
	// launch the lightweight zero-browser TUI player!
	if len(os.Args) < 2 {
		runTUIPlayer(dbPath, os.Args[1:])
		return
	}

	cmd := os.Args[1]
	if _, err := strconv.Atoi(cmd); err == nil {
		// User ran "./imagine 9" or "./imagine 9 Bb" directly!
		runTUIPlayer(dbPath, os.Args[1:])
		return
	}

	switch cmd {
	case "play", "tui":
		runTUIPlayer(dbPath, os.Args[2:])

	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		genre := fs.String("genre", "", "Filter by genre (trance, dnb, squarepusher, modular, lofi)")
		search := fs.String("q", "", "Search query")
		key := fs.String("key", "", "Transpose listed progressions to target key (e.g. G, Bb, F#)")
		verbose := fs.Bool("v", false, "Show full theory notes")
		fs.Parse(os.Args[2:])

		store, err := db.Open(dbPath)
		must(err)
		defer store.Close()

		list, err := store.ListProgressions(*genre, *search)
		must(err)

		printCompactSongList(list, *key, *verbose)

	case "export", "generate":
		fs := flag.NewFlagSet("export", flag.ExitOnError)
		id := fs.Int("id", 1, "Progression ID from SQLite database")
		hybrid := fs.String("hybrid", "", "Synthesize a hybrid progression, e.g. 'Trance,Squarepusher'")
		key := fs.String("key", "", "Transpose to target key (e.g. C, G, F#, Bb)")
		profile := fs.String("profile", theory.DefaultEngineProfile(), "Engine density profile: v3 (Mac Sketch), v2 (Refined Sparse), v1 (Legacy)")
		outDir := fs.String("out", "./output", "Output directory for .mid, .seq, and .svg files")
		complexity := fs.Float64("complexity", 0.60, "Transition complexity (0.0 - 1.0)")
		colorD := fs.Float64("color", 0.70, "Harmonic color density (0.0 - 1.0)")
		bpm := fs.Int("bpm", 0, "Override tempo in BPM (40 - 200)")
		craveOct := fs.Int("crave-oct", -1, "Octave offset for Crave (-3 to +3, default -1 Sub)")
		eastOct := fs.Int("east-oct", 1, "Octave offset for East Beast (-3 to +3, default +1 High)")
		edgeOct := fs.Int("edge-oct", 0, "Octave offset for Edge (-3 to +3)")
		polyOct := fs.Int("poly-oct", 0, "Octave offset for Poly Chords (-3 to +3)")
		fs.Parse(os.Args[2:])

		store, err := db.Open(dbPath)
		must(err)
		defer store.Close()

		var arr theory.Arrangement
		seed := time.Now().UnixNano()
		if *hybrid != "" {
			parts := strings.Split(*hybrid, ",")
			g1 := parts[0]
			g2 := "Esoteric Modular"
			if len(parts) > 1 {
				g2 = parts[1]
			}
			arr = theory.GenerateHybridProgressionWithProfile(g1, g2, *key, *complexity, *colorD, seed, *profile)
		} else {
			arr, err = store.GetProgression(*id)
			must(err)
			arr.Profile = *profile
			arr.Complexity = *complexity
			arr.ColorDensity = *colorD
			arr = theory.TransposeArrangement(arr, *key, seed)
		}
		if *bpm >= 30 && *bpm <= 300 {
			arr.BPM = *bpm
		}
		vs := theory.DefaultVoiceSettings()
		vs.CraveOctave = *craveOct
		vs.EastBeastOctave = *eastOct
		vs.EdgeOctave = *edgeOct
		vs.PolyOctave = *polyOct
		vs.PolyChannel = 3 // Ch 4 for DAW/soft-synth separation
		arr = theory.ApplyVoiceSettings(arr, vs)

		must(os.MkdirAll(*outDir, 0755))
		baseSlug := cleanSlug(arr.Title + "_" + arr.RootKey + "_" + arr.Profile)

		midiPath := filepath.Join(*outDir, baseSlug+".mid")
		must(os.WriteFile(midiPath, midiout.EncodeMultiTrackMIDI(arr), 0644))

		svgPath := filepath.Join(*outDir, baseSlug+"_chart.svg")
		must(os.WriteFile(svgPath, []byte(web.RenderArrangementSVG(arr, 0)), 0644))

		var seqFiles []string
		for _, sl := range arr.Slices {
			craveSeqPath := filepath.Join(*outDir, fmt.Sprintf("%s_%s.crave.seq", baseSlug, sl.Slug))
			must(os.WriteFile(craveSeqPath, seq.EncodeCraveSeq(sl.CraveSteps, sl.Swing), 0644))
			seqFiles = append(seqFiles, craveSeqPath)

			ebSeqPath := filepath.Join(*outDir, fmt.Sprintf("%s_%s.eastbeast.seq", baseSlug, sl.Slug))
			must(os.WriteFile(ebSeqPath, seq.EncodeCraveSeq(sl.EastBeastSteps, sl.Swing), 0644))
			seqFiles = append(seqFiles, ebSeqPath)

			td3SeqPath := filepath.Join(*outDir, fmt.Sprintf("%s_%s.td3.seq", baseSlug, sl.Slug))
			must(os.WriteFile(td3SeqPath, seq.EncodeTD3Seq(sl.CraveSteps), 0644))
			seqFiles = append(seqFiles, td3SeqPath)
		}

		printTerminalArrangement(arr, midiPath, svgPath, seqFiles)

	case "inspect":
		if len(os.Args) < 3 {
			fmt.Println("Usage: imagine inspect <pattern.crave.seq>")
			os.Exit(1)
		}
		data, err := os.ReadFile(os.Args[2])
		must(err)
		summary, err := seq.InspectCraveSeq(data)
		must(err)
		fmt.Print(summary)

	case "import-obsidian":
		fs := flag.NewFlagSet("import-obsidian", flag.ExitOnError)
		requireTags := fs.Bool("require-tags", true, "Only import notes with both 'music' and 'chords' in frontmatter tags")
		fs.Parse(os.Args[2:])
		vaultPath := "/home/james/brain"
		if fs.NArg() >= 1 {
			vaultPath = fs.Arg(0)
		}
		store, err := db.Open(dbPath)
		must(err)
		defer store.Close()

		beforeCount, _ := store.Count()
		_, err = obsidian.ImportDirectory(store, vaultPath, *requireTags)
		must(err)
		afterCount, _ := store.Count()
		fmt.Printf("Scanned Obsidian vault (%s) for notes tagged [music, chords]:\n", vaultPath)
		fmt.Printf("  Added %d new progressions (Total in %s: %d progressions)\n", afterCount-beforeCount, dbPath, afterCount)

	case "export-seed":
		outPath := "pkg/db/obsidian_seed.json"
		if len(os.Args) >= 3 {
			outPath = os.Args[2]
		}
		store, err := db.Open(dbPath)
		must(err)
		defer store.Close()

		n, err := store.ExportObsidianSeedJSON(outPath)
		must(err)
		fmt.Printf("Exported %d Obsidian progressions to %s (embedded for cross-platform builds)\n", n, outPath)

	case "midi-ports", "devices":
		ports, err := midiout.ListHardwarePorts()
		must(err)
		fmt.Printf("Detected %d Native OS MIDI Output Port(s):\n", len(ports))
		for _, p := range ports {
			fmt.Printf("  - [%s] ID=%s  Name=%q\n", p.Backend, p.ID, p.Name)
		}

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.Int("port", 8090, "HTTP port for interactive studio")
		fs.Parse(os.Args[2:])

		store, err := db.Open(dbPath)
		must(err)
		defer store.Close()

		srv := web.NewServer(store)
		addr := fmt.Sprintf("127.0.0.1:%d", *port)
		fmt.Printf("IMAGINATION ENGINE Studio running at http://%s\n", addr)
		fmt.Println("Features: Live Multi-Port Native MIDI + Song Journey Chain + .MID / .SEQ / .SVG / .PNG Export")
		must(http.ListenAndServe(addr, srv.Handler()))

	default:
		printUsage()
	}
}

// runTUIPlayer starts the lightweight, zero-browser-overhead Terminal Player & Menu.
// It auto-connects to all active MIDI ports (Ch 1: Crave Sub, Ch 2: East Beast High, Ch 3: Edge Beat, Ch 4: Poly Pad)
// and lets you play any song by its reference number (1..65) and transpose keys on the fly.
func runTUIPlayer(dbPath string, args []string) {
	fs := flag.NewFlagSet("play", flag.ExitOnError)
	idFlag := fs.Int("id", 0, "Song reference number (1..65) to start playing immediately")
	keyFlag := fs.String("key", "", "Target key (e.g. C, G, Bb, F#, Eb)")
	bpmFlag := fs.Int("bpm", 0, "Override BPM (default 80..95)")
	profileFlag := fs.String("profile", theory.DefaultEngineProfile(), "Engine mode: v3 (Mac Sketch), v2 (Refined), v1 (Legacy)")
	polyChFlag := fs.Int("poly-ch", 4, "MIDI Channel (1..16) for Poly Pad Shells (default 4; set 1 for Ch 1)")
	fs.Parse(args)

	// Allow positional arguments: e.g. `imagine play 9` or `imagine play 9 Bb`
	rem := fs.Args()
	if len(rem) >= 1 {
		if v, err := strconv.Atoi(rem[0]); err == nil && v > 0 {
			*idFlag = v
		} else if isValidKeyName(rem[0]) {
			*keyFlag = rem[0]
		}
	}
	if len(rem) >= 2 && isValidKeyName(rem[1]) {
		*keyFlag = rem[1]
	}

	store, err := db.Open(dbPath)
	must(err)
	defer store.Close()

	allList, err := store.ListProgressions("", "")
	must(err)

	ports, _ := midiout.ListHardwarePorts()
	engine := midiout.NewEngine()

	// Ensure Ctrl+C stops all MIDI notes cleanly
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nStopping MIDI Sequencer & sending All-Notes-Off...")
		engine.Stop()
		os.Exit(0)
	}()

	currentID := *idFlag
	if currentID <= 0 {
		currentID = 1
	}
	targetKey := strings.TrimSpace(*keyFlag)
	customBPM := *bpmFlag
	profile := *profileFlag
	songMode := "journey" // Full 9-Stage Song Journey by default
	sliceIdx := 0
	playing := true

	polyCh := (*polyChFlag - 1) & 0x0F
	vs := theory.DefaultVoiceSettings()
	vs.CraveChannel = 0     // MIDI Ch 1 (Crave Sub -1 Oct)
	vs.EastBeastChannel = 1 // MIDI Ch 2 (East Beast High +1 Oct)
	vs.EdgeChannel = 2      // MIDI Ch 3 (Edge Constant Beat Pulse)
	vs.PolyChannel = polyCh // MIDI Ch 4 (or Ch 1) for Soft Synths / Omni

	enablePoly := true
	enableCrave := true
	enableEast := true
	enableEdge := false // Edge runs natively off channel-less MIDI Clock (0xF8 + Start/Stop)!
	sendClock := true
	clockDiv := 1

	fmt.Println(strings.Repeat("=", 88))
	fmt.Printf("IMAGINATION ENGINE // LIGHTWEIGHT CLI & TUI PLAYER (Mode: %s | Zero Browser Overhead)\n", strings.ToUpper(profile))
	if len(ports) == 0 {
		fmt.Println("MIDI PORTS : None detected yet (type 'r' to rescan once synths are connected)")
	} else {
		var pNames []string
		for _, p := range ports {
			pNames = append(pNames, p.Name)
		}
		fmt.Printf("MIDI PORTS : Auto-Started ALL %d Active Port(s): %s\n", len(ports), strings.Join(pNames, " + "))
	}
	fmt.Printf("ROUTING    : Ch1=Crave(-1 Sub) | Ch2=EastBeast(+1 High) | Ch%d=PolyPad(2-3 Notes) | Clock(0xF8)->Edge\n", vs.PolyChannel+1)
	fmt.Println(strings.Repeat("=", 88))

	// If user didn't specify a song ID on the CLI, show compact menu first, then auto-play #1
	if *idFlag == 0 {
		printCompactSongList(allList, targetKey, false)
	}

	var currentArr theory.Arrangement

	syncPlayback := func() {
		arr, err := store.GetProgression(currentID)
		if err != nil {
			fmt.Printf("Song #%d not found (valid range: 1..%d)\n", currentID, len(allList))
			return
		}
		arr.Profile = profile
		arr = theory.TransposeArrangement(arr, targetKey, int64(arr.ID*101+7))
		if customBPM >= 40 && customBPM <= 200 {
			arr.BPM = customBPM
		}
		arr = theory.ApplyVoiceSettings(arr, vs)
		currentArr = arr

		engine.UpdateAndStart(midiout.TransportState{
			Playing:     playing,
			PortID:      "all",
			SongMode:    songMode,
			SliceIdx:    sliceIdx,
			StageIdx:    0,
			BPM:         arr.BPM,
			SendClock:   sendClock,
			ClockDiv:    clockDiv,
			SendModCC:   true,
			EnablePoly:  enablePoly,
			EnableCrave: enableCrave,
			EnableEast:  enableEast,
			EnableEdge:  enableEdge,
			VoiceConfig: vs,
			Arrangement: arr,
		})
		printMinimalNowPlaying(arr, playing, songMode, sliceIdx, len(ports), enablePoly, enableCrave, enableEast, sendClock, clockDiv)
	}

	syncPlayback()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nimagine [Song# | Key (C,G,Bb..) | b <BPM> | s <1-5> | j | l | p | q]> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			st := engine.Status()
			printMinimalNowPlaying(currentArr, playing, songMode, st.SliceIdx, len(ports), enablePoly, enableCrave, enableEast, sendClock, clockDiv)
			continue
		}

		fields := strings.Fields(line)
		cmdTok := strings.ToLower(fields[0])

		// 1. Direct song number (e.g. "9" or "9 Bb" or "15 G")
		if num, err := strconv.Atoi(fields[0]); err == nil && num >= 1 && num <= len(allList) {
			currentID = num
			if len(fields) >= 2 && isValidKeyName(fields[1]) {
				targetKey = normalizeKeyInput(fields[1])
			}
			playing = true
			syncPlayback()
			continue
		}

		// 2. Direct key name (e.g. "G", "Bb", "F#", "Eb", "C")
		if isValidKeyName(fields[0]) && len(fields) == 1 {
			targetKey = normalizeKeyInput(fields[0])
			syncPlayback()
			continue
		}

		switch cmdTok {
		case "q", "quit", "exit":
			fmt.Println("Stopping MIDI Sequencer & sending All-Notes-Off...")
			engine.Stop()
			return

		case "p", "pause", "play", "stop":
			playing = !playing
			if !playing {
				engine.Stop()
				fmt.Println("⏸ Sequencer Paused (All Notes Off).")
			} else {
				syncPlayback()
			}

		case "k", "key":
			if len(fields) >= 2 {
				if strings.ToLower(fields[1]) == "orig" || strings.ToLower(fields[1]) == "native" {
					targetKey = ""
				} else if isValidKeyName(fields[1]) {
					targetKey = normalizeKeyInput(fields[1])
				}
				syncPlayback()
			} else {
				fmt.Println("Usage: k <Key> (e.g. 'k G', 'k Bb', 'k F#', or 'k orig')")
			}

		case "orig", "native":
			targetKey = ""
			syncPlayback()

		case "b", "bpm", "tempo":
			if len(fields) >= 2 {
				if v, err := strconv.Atoi(fields[1]); err == nil && v >= 40 && v <= 200 {
					customBPM = v
					syncPlayback()
				}
			} else {
				fmt.Println("Usage: b <BPM> (e.g. 'b 84' or 'b 92')")
			}

		case "s", "slice", "section":
			if len(fields) >= 2 {
				if v, err := strconv.Atoi(fields[1]); err == nil && v >= 1 && v <= 5 {
					songMode = "loop"
					sliceIdx = v - 1
					syncPlayback()
				}
			} else {
				fmt.Println("Usage: s <1..5> (1=Intro, 2=Build, 3=Drop, 4=Breakdown, 5=Outro)")
			}

		case "j", "journey", "chain":
			songMode = "journey"
			syncPlayback()

		case "v1", "v2", "v3":
			profile = cmdTok
			syncPlayback()

		case "c", "clock":
			if len(fields) >= 2 {
				switch strings.ToLower(fields[1]) {
				case "off", "0":
					sendClock = false
				case "on":
					sendClock = true
				case "1", "1x":
					sendClock = true
					clockDiv = 1
				case "2", "1/2", "half":
					sendClock = true
					clockDiv = 2
				case "4", "1/4", "beat":
					sendClock = true
					clockDiv = 4
				}
			} else {
				sendClock = !sendClock
			}
			syncPlayback()

		case "m", "mute":
			if len(fields) >= 2 {
				switch strings.ToLower(fields[1]) {
				case "1", "crave":
					enableCrave = !enableCrave
				case "2", "east":
					enableEast = !enableEast
				case "3", "clock", "edge":
					sendClock = !sendClock
				case "4", "poly", "pad":
					enablePoly = !enablePoly
				}
				syncPlayback()
			} else {
				fmt.Println("Usage: m <1|2|3|4> (toggle 1=Crave, 2=EastBeast, 3=Edge Clock, 4=PolyPad)")
			}

		case "ch1", "omni":
			vs.PolyChannel = 0 // Put Poly Pad on Ch 1 (Omni)
			syncPlayback()

		case "ch4":
			vs.PolyChannel = 3 // Put Poly Pad on Ch 4
			syncPlayback()

		case "r", "rescan":
			ports, _ = midiout.ListHardwarePorts()
			syncPlayback()

		case "l", "list":
			gFilter := ""
			if len(fields) >= 2 {
				gFilter = strings.Join(fields[1:], " ")
			}
			filtered, _ := store.ListProgressions(gFilter, "")
			printCompactSongList(filtered, targetKey, false)

		default:
			fmt.Println("Commands: <1..65> play song # | <Key> transpose (C, G, Bb, F#, orig) | b <BPM> | s <1..5> section | j journey | v3/v2/v1 mode | m <1..4> mute voice | l [genre] list | p pause | q quit")
		}
	}
}

// printCompactSongList prints the streamlined song table showing only Reference #, Key, BPM, Chord Pattern, and Root Notes.
func printCompactSongList(list []theory.Arrangement, targetKey string, verbose bool) {
	keyLabel := "Original Keys"
	if targetKey != "" {
		keyLabel = "Transposed to Key: " + targetKey
	}
	fmt.Printf("\nSONG CORPUS (%d songs | %s) — Type any number (1..%d) to play:\n", len(list), keyLabel, len(list))
	fmt.Println(strings.Repeat("-", 96))
	fmt.Printf("%-6s | %-5s | %-7s | %-38s | %-28s\n", "REF #", "KEY", "TEMPO", "CHORD PATTERN", "ROOT NOTES")
	fmt.Println(strings.Repeat("-", 96))
	for _, a := range list {
		if targetKey != "" {
			a = theory.TransposeArrangement(a, targetKey, int64(a.ID*101+7))
		}
		var syms []string
		var roots []string
		for _, c := range a.Chords {
			syms = append(syms, c.Symbol)
			roots = append(roots, c.RootName)
		}
		fmt.Printf("[#%02d]  | %-5s | %3d BPM | %-38s | Roots: %-21s (%s)\n",
			a.ID, a.RootKey, a.BPM,
			truncateCLI(strings.Join(syms, " -> "), 38),
			strings.Join(roots, " - "),
			a.Genre)
		if verbose {
			fmt.Printf("       Theory: %s\n", a.TheoryNotes)
		}
	}
	fmt.Println(strings.Repeat("-", 96))
}

// printMinimalNowPlaying shows only the essential musical info: Song #, Key, Chords, Root Notes, and Active Voices.
func printMinimalNowPlaying(arr theory.Arrangement, playing bool, songMode string, sliceIdx, portCount int, enPoly, enCrave, enEast, sendClock bool, clockDiv int) {
	status := "▶ PLAYING"
	if !playing {
		status = "⏸ PAUSED"
	}

	var chordSyms []string
	var rootNotes []string
	var padVoicings []string

	for _, c := range arr.Chords {
		chordSyms = append(chordSyms, fmt.Sprintf("%-9s", c.Symbol))
		subOct := 2 + arr.VoiceConfig.CraveOctave
		if c.RootPC >= 9 {
			subOct--
		}
		rootNotes = append(rootNotes, fmt.Sprintf("%-9s", fmt.Sprintf("%s (%s%d)", c.RootName, c.RootName, subOct)))

		var vNames []string
		for _, m := range c.VoicingMIDI {
			vNames = append(vNames, theory.MIDIToNoteName(m))
		}
		padVoicings = append(padVoicings, fmt.Sprintf("[%s]", strings.Join(vNames, " ")))
	}

	modeDesc := "Full 9-Stage Song Journey (Auto-Chain)"
	if songMode != "journey" && sliceIdx >= 0 && sliceIdx < len(arr.Slices) {
		modeDesc = "Looping " + arr.Slices[sliceIdx].Name
	}

	clockLabel := "OFF"
	if sendClock {
		switch clockDiv {
		case 2:
			clockLabel = "ON (1/2x Half-Time 12 PPQN)"
		case 4:
			clockLabel = "ON (1/4x Beat Pulse 6 PPQN)"
		default:
			clockLabel = "ON (1x 24 PPQN)"
		}
	}

	fmt.Println(strings.Repeat("-", 88))
	fmt.Printf("%s [#%02d] %s  |  KEY: %s  |  %d BPM  |  Engine: %s  |  MIDI Ports: %d\n",
		status, arr.ID, arr.Title, arr.RootKey, arr.BPM, strings.ToUpper(arr.Profile), portCount)
	fmt.Printf("  CHORD PATTERN :  %s\n", strings.Join(chordSyms, " -> "))
	fmt.Printf("  ROOT NOTES    :  %s\n", strings.Join(rootNotes, " -> "))
	fmt.Printf("  PAD SHELLS    :  %s\n", strings.Join(padVoicings, " -> "))
	fmt.Printf("  SONG MODE     :  %s\n", modeDesc)
	fmt.Printf("  VOICES & SYNC :  Ch%d Crave(%s) | Ch%d EastBeast(%s) | Ch%d PolyPad(%s) | Edge Clock: %s\n",
		arr.VoiceConfig.CraveChannel+1, onOff(enCrave),
		arr.VoiceConfig.EastBeastChannel+1, onOff(enEast),
		arr.VoiceConfig.PolyChannel+1, onOff(enPoly),
		clockLabel)
	fmt.Println(strings.Repeat("-", 88))
}

func isValidKeyName(s string) bool {
	s = strings.TrimSpace(s)
	valid := []string{
		"C", "C#", "Db", "D", "D#", "Eb", "E", "F",
		"F#", "Gb", "G", "G#", "Ab", "A", "A#", "Bb", "B",
	}
	for _, k := range valid {
		if strings.EqualFold(s, k) {
			return true
		}
	}
	return false
}

func normalizeKeyInput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	pc := theory.ParsePitchClass(s)
	return theory.UIKeyNames[pc]
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "MUTE"
}

func truncateCLI(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
}

func printTerminalArrangement(arr theory.Arrangement, midiPath, svgPath string, seqFiles []string) {
	fmt.Println(strings.Repeat("=", 96))
	fmt.Printf("ARRANGEMENT : %s\n", arr.Title)
	fmt.Printf("GENRE / KEY : %s  |  Key: %s (%s)  |  %d BPM  |  Profile: %s\n", arr.Genre, arr.RootKey, arr.Mode, arr.BPM, strings.ToUpper(arr.Profile))
	fmt.Println(strings.Repeat("-", 96))
	fmt.Println("CHORD PROGRESSION & ROOT NOTES:")
	for i, ch := range arr.Chords {
		var vNames []string
		for _, m := range ch.VoicingMIDI {
			vNames = append(vNames, theory.MIDIToNoteName(m))
		}
		fmt.Printf("  Bar %d: %-10s [Root: %-2s] %-24s | Pad Shell: %-18s\n",
			i+1, ch.Symbol, ch.RootName, ch.Function, strings.Join(vNames, " "))
	}
	fmt.Println(strings.Repeat("-", 96))
	fmt.Printf("EXPORTED FILES:\n")
	fmt.Printf("  [MIDI 5-Track] %s\n", midiPath)
	fmt.Printf("  [SVG Chart]    %s\n", svgPath)
	for _, sf := range seqFiles {
		fmt.Printf("  [SynthTribe]   %s\n", sf)
	}
	fmt.Println(strings.Repeat("=", 96))
}

func cleanSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prev := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prev = false
		} else if !prev {
			b.WriteByte('_')
			prev = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`IMAGINATION ENGINE (imagine) - Modular Harmony, Multi-Port MIDI & Behringer Crave .SEQ Engine

Usage:
  imagine play [song#] [key]            # Zero-browser CLI/TUI player (auto-starts all MIDI ports & channels)
  imagine <song#> [key]                 # Shortcut to immediately play song # (e.g. 'imagine 9 Bb' or 'imagine 1 G')
  imagine list [--key G] [--genre dnb]  # Compact numbered list of Chords & Root Notes
  imagine export [--id 9] [--key Bb]    # Export .mid, .crave.seq, .eastbeast.seq, .td3.seq, and .svg
  imagine midi-ports                    # List detected CoreMIDI / ALSA hardware & virtual ports
  imagine serve [--port 8090]           # Launch web studio`)
}
