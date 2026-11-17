package midiout

import (
	"strings"
	"sync"
	"time"

	"github.com/james-see/imagination-engine/pkg/theory"
)

// TransportState holds the live configuration for the Go backend hardware/soft-synth sequencer loop.
type TransportState struct {
	Playing     bool                 `json:"playing"`
	PortID      string               `json:"port_id"`    // "all" or specific port ID
	SongMode    string               `json:"song_mode"`  // "loop" (single section) or "journey" (9-stage song chain)
	SliceIdx    int                  `json:"slice_idx"`  // 0..4
	StageIdx    int                  `json:"stage_idx"`  // 0..8 when SongMode == "journey"
	BPM         int                  `json:"bpm"`        // 40..240 (default 80..95)
	SendClock   bool                 `json:"send_clock"` // 24 PPQN 0xF8 MIDI Clock (drives Behringer Edge natively)
	ClockDiv    int                  `json:"clock_div"`  // 1 = 24 PPQN, 2 = 12 PPQN (Half), 4 = 6 PPQN (Quarter/Beat)
	SendModCC   bool                 `json:"send_mod_cc"`
	EnablePoly  bool                 `json:"enable_poly"`
	EnableCrave bool                 `json:"enable_crave"`
	EnableEast  bool                 `json:"enable_east"`
	EnableEdge  bool                 `json:"enable_edge"`
	VoiceConfig theory.VoiceSettings `json:"voice_config"`
	Arrangement theory.Arrangement   `json:"arrangement"`
}

// TransportStatus is returned to the UI for low-CPU playhead, section, and CC modulation display.
type TransportStatus struct {
	Running    bool   `json:"running"`
	Step       int    `json:"step"`
	SliceIdx   int    `json:"slice_idx"`
	StageIdx   int    `json:"stage_idx"`
	StageLabel string `json:"stage_label"`
	ModCC1     int    `json:"mod_cc1"`
	CutoffCC74 int    `json:"cutoff_cc74"`
}

// Engine drives native OS MIDI output (ALSA Sequencer / CoreMIDI) from a high-precision Go goroutine
// so browser UI threads (Chrome/Firefox/Safari) use near-zero CPU and never cause MIDI lag.
type Engine struct {
	mu          sync.Mutex
	state       TransportState
	ports       []PortInfo
	stopCh      chan struct{}
	running     bool
	currentStep int
	activeSlice int
	activeStage int
	stageLabel  string
	lastCC1     int
	lastCC74    int
}

func NewEngine() *Engine {
	return &Engine{
		lastCC1:  -1,
		lastCC74: -1,
	}
}

func (e *Engine) UpdateAndStart(st TransportState) {
	discovered, _ := ListHardwarePorts()

	e.mu.Lock()
	wasRunning := e.running
	oldPorts := e.ports
	e.ports = discovered
	e.state = st

	if !st.Playing || st.PortID == "none" {
		if wasRunning {
			close(e.stopCh)
			e.running = false
			e.mu.Unlock()
			sendStopToAll(oldPorts)
			return
		}
		e.mu.Unlock()
		return
	}

	if !wasRunning {
		e.stopCh = make(chan struct{})
		e.running = true
		e.currentStep = 0
		e.lastCC1 = -1
		e.lastCC74 = -1

		chain := st.Arrangement.SongChain
		if len(chain) == 0 {
			chain = theory.DefaultSongJourneyChain()
		}
		if st.SongMode == "journey" {
			stg := st.StageIdx
			if stg < 0 || stg >= len(chain) {
				stg = 0
			}
			e.activeStage = stg
			e.activeSlice = chain[stg].SliceIndex
			e.stageLabel = chain[stg].Label
		} else {
			e.activeStage = 0
			e.activeSlice = st.SliceIdx
			e.stageLabel = "Looping Section"
		}

		stopCh := e.stopCh
		targetPorts := resolveTargetPorts(st.PortID, discovered)
		e.mu.Unlock()
		for _, pid := range targetPorts {
			_ = SendHardwareMIDI(pid, []byte{0xFA}) // MIDI Start
		}
		go e.runLoop(stopCh)
		return
	}

	// If already running and user clicked a section tab in "loop" mode, update immediately
	if st.SongMode != "journey" {
		e.activeSlice = st.SliceIdx
		e.stageLabel = "Looping Section"
	}
	e.mu.Unlock()
}

func (e *Engine) Stop() {
	e.mu.Lock()
	if e.running {
		ports := e.ports
		close(e.stopCh)
		e.running = false
		e.state.Playing = false
		e.mu.Unlock()
		sendStopToAll(ports)
		return
	}
	e.mu.Unlock()
}

func (e *Engine) CurrentStep() (bool, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running, e.currentStep
}

func (e *Engine) Status() TransportStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	cc1 := e.lastCC1
	if cc1 < 0 {
		cc1 = 12
	}
	cc74 := e.lastCC74
	if cc74 < 0 {
		cc74 = 48
	}
	return TransportStatus{
		Running:    e.running,
		Step:       e.currentStep,
		SliceIdx:   e.activeSlice,
		StageIdx:   e.activeStage,
		StageLabel: e.stageLabel,
		ModCC1:     cc1,
		CutoffCC74: cc74,
	}
}

func resolveTargetPorts(portSetting string, available []PortInfo) []string {
	if portSetting == "none" {
		return nil
	}
	if portSetting == "" || portSetting == "all" {
		out := make([]string, 0, len(available))
		for _, p := range available {
			out = append(out, p.ID)
		}
		return out
	}
	return []string{portSetting}
}

func isVitalOmniPort(p PortInfo) bool {
	if p.ID == "coremidi:virtual_vital" {
		return true
	}
	return strings.Contains(strings.ToLower(p.Name), "vital")
}

func resolveVoicePorts(voicePortID, globalPortID string, available []PortInfo, isPolyPad bool) []string {
	if voicePortID == "none" {
		return nil
	}
	if voicePortID != "" && voicePortID != "all" && voicePortID != "global" {
		return []string{voicePortID}
	}
	if globalPortID == "none" {
		return nil
	}
	if globalPortID != "" && globalPortID != "all" {
		return []string{globalPortID}
	}
	// Smart "all" routing:
	// - Poly Pad goes to ALL ports (including "Imagine Pad Only (Vital Omni-Safe)" and "Surge XT / Multi-Ch").
	// - Monophonic Crave (Ch 1) & East Beast (Ch 2) are automatically ABSORBED/FILTERED OUT of Vital Omni ports
	//   so Vital (in Omni mode) only plays the Pad Chords while Hardware + Surge XT receive all channels!
	out := make([]string, 0, len(available))
	for _, p := range available {
		if !isPolyPad && isVitalOmniPort(p) {
			continue
		}
		out = append(out, p.ID)
	}
	return out
}

func sendToPorts(portIDs []string, msg []byte) {
	for _, pid := range portIDs {
		if pid != "" {
			_ = SendHardwareMIDI(pid, msg)
		}
	}
}

func sendStopToAll(ports []PortInfo) {
	for _, p := range ports {
		_ = SendHardwareMIDI(p.ID, []byte{0xFC}) // MIDI Stop
		for ch := byte(0); ch < 16; ch++ {
			_ = SendHardwareMIDI(p.ID, []byte{0xB0 | ch, 123, 0}) // All Notes Off
		}
	}
}

func (e *Engine) runLoop(stopCh chan struct{}) {
	subTick := 0 // 0..5 (6 MIDI clock pulses per step = 24 PPQN)
	totalPulses := 0
	step := 0

	for {
		e.mu.Lock()
		st := e.state
		ports := e.ports
		e.currentStep = step
		sIdx := e.activeSlice
		e.mu.Unlock()

		bpm := st.BPM
		if bpm < 30 {
			bpm = 86
		}
		if bpm > 260 {
			bpm = 260
		}

		stepDur := time.Duration(float64(time.Minute) / float64(bpm) / 4.0)
		clockDur := stepDur / 6

		if st.SendClock {
			cDiv := st.ClockDiv
			if cDiv != 2 && cDiv != 4 {
				cDiv = 1
			}
			if totalPulses%cDiv == 0 {
				clockPorts := resolveTargetPorts(st.PortID, ports)
				sendToPorts(clockPorts, []byte{0xF8})
			}
		}

		if subTick == 0 {
			e.triggerStep(st, ports, sIdx, step, stepDur)
		}

		select {
		case <-stopCh:
			return
		case <-time.After(clockDur):
		}

		totalPulses++
		subTick++
		if subTick >= 6 {
			subTick = 0
			step++
			if step >= 32 {
				step = 0
				e.advanceSectionIfJourney()
			}
		}
	}
}

func (e *Engine) advanceSectionIfJourney() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.SongMode != "journey" {
		e.activeSlice = e.state.SliceIdx
		e.stageLabel = "Looping Section"
		return
	}
	chain := e.state.Arrangement.SongChain
	if len(chain) == 0 {
		chain = theory.DefaultSongJourneyChain()
	}
	e.activeStage = (e.activeStage + 1) % len(chain)
	e.activeSlice = chain[e.activeStage].SliceIndex
	e.stageLabel = chain[e.activeStage].Label
}

func (e *Engine) triggerStep(st TransportState, availablePorts []PortInfo, sIdx, step int, stepDur time.Duration) {
	arr := st.Arrangement
	if len(arr.Slices) == 0 {
		return
	}
	if sIdx < 0 || sIdx >= len(arr.Slices) {
		sIdx = 0
	}
	sl := arr.Slices[sIdx]
	vc := st.VoiceConfig

	polyPorts := resolveVoicePorts(vc.PolyPortID, st.PortID, availablePorts, true)
	cravePorts := resolveVoicePorts(vc.CravePortID, st.PortID, availablePorts, false)
	eastPorts := resolveVoicePorts(vc.EastBeastPortID, st.PortID, availablePorts, false)
	edgePorts := resolveVoicePorts(vc.EdgePortID, st.PortID, availablePorts, false)

	// Send MIDI CC#1 (Mod Wheel) and CC#74 (Filter Cutoff) on Build-Up / Drop / Transitions
	if step < len(sl.CraveSteps) {
		modCC1 := sl.CraveSteps[step].ModCC1
		cutoffCC74 := sl.CraveSteps[step].CutoffCC74

		e.mu.Lock()
		ccChanged := (modCC1 != e.lastCC1 || cutoffCC74 != e.lastCC74)
		e.lastCC1 = modCC1
		e.lastCC74 = cutoffCC74
		e.mu.Unlock()

		if st.SendModCC && ccChanged {
			if st.EnablePoly && len(polyPorts) > 0 {
				ch := byte(vc.PolyChannel & 0x0F)
				sendToPorts(polyPorts, []byte{0xB0 | ch, 1, byte(modCC1 & 0x7F)})
				sendToPorts(polyPorts, []byte{0xB0 | ch, 74, byte(cutoffCC74 & 0x7F)})
			}
			if st.EnableCrave && len(cravePorts) > 0 {
				ch := byte(vc.CraveChannel & 0x0F)
				sendToPorts(cravePorts, []byte{0xB0 | ch, 1, byte(modCC1 & 0x7F)})
				sendToPorts(cravePorts, []byte{0xB0 | ch, 74, byte(cutoffCC74 & 0x7F)})
			}
			if st.EnableEast && len(eastPorts) > 0 {
				ch := byte(vc.EastBeastChannel & 0x0F)
				sendToPorts(eastPorts, []byte{0xB0 | ch, 1, byte(modCC1 & 0x7F)})
				sendToPorts(eastPorts, []byte{0xB0 | ch, 74, byte(cutoffCC74 & 0x7F)})
			}
		}
	}

	// 1. Polyphonic 2-to-3 Note Pad Shells (e.g., Vital / Surge XT)
	if st.EnablePoly && len(arr.Chords) > 0 && len(polyPorts) > 0 {
		stepsPerChord := 32 / len(arr.Chords)
		if stepsPerChord < 1 {
			stepsPerChord = 1
		}
		if step%stepsPerChord == 0 {
			cIdx := (step / stepsPerChord) % len(arr.Chords)
			ch := arr.Chords[cIdx]
			chordDur := time.Duration(float64(stepDur) * float64(stepsPerChord) * 0.90)
			midiCh := byte(vc.PolyChannel & 0x0F)
			for _, n := range ch.VoicingMIDI {
				scheduleMultiPortNote(polyPorts, midiCh, byte(n&0x7F), 82, 0, chordDur)
			}
		}
	}

	// 2. Crave Sub/Bass Sequence (default -1 Oct Sub)
	if st.EnableCrave && step < len(sl.CraveSteps) && len(cravePorts) > 0 {
		ev := sl.CraveSteps[step]
		if !ev.Rest {
			triggerMonoStep(cravePorts, byte(vc.CraveChannel&0x0F), ev, stepDur)
		}
	}

	// 3. East Beast High/Mid Counter-Voice (default +1 Oct High)
	if st.EnableEast && step < len(sl.EastBeastSteps) && len(eastPorts) > 0 {
		ev := sl.EastBeastSteps[step]
		if !ev.Rest {
			triggerMonoStep(eastPorts, byte(vc.EastBeastChannel&0x0F), ev, stepDur)
		}
	}

	// 4. Behringer Edge Beat Pulse (single constant-pitch pulse every beat: step % 4 == 0)
	if st.EnableEdge && len(sl.EdgeSteps) > 0 && len(edgePorts) > 0 {
		ev := sl.EdgeSteps[step%len(sl.EdgeSteps)]
		if !ev.Rest {
			scheduleMultiPortNote(edgePorts, byte(vc.EdgeChannel&0x0F), byte(ev.MIDINote&0x7F), byte(ev.Velocity&0x7F), 0, stepDur/2)
		}
	}
}

func triggerMonoStep(portIDs []string, midiCh byte, ev theory.StepEvent, stepDur time.Duration) {
	ratchets := ev.Ratchet + 1
	if ratchets < 1 {
		ratchets = 1
	}
	if ratchets > 4 {
		ratchets = 4
	}
	subDur := stepDur / time.Duration(ratchets)
	gateFrac := float64(ev.Gate+1) / 8.0
	if ev.Glide && ratchets == 1 {
		gateFrac = 1.05 // slight legato overlap for hardware portamento
	} else if gateFrac > 0.92 {
		gateFrac = 0.92
	}
	noteDur := time.Duration(float64(subDur) * gateFrac)
	if noteDur < 18*time.Millisecond {
		noteDur = 18 * time.Millisecond
	}
	for r := 0; r < ratchets; r++ {
		delay := time.Duration(r) * subDur
		scheduleMultiPortNote(portIDs, midiCh, byte(ev.MIDINote&0x7F), byte(ev.Velocity&0x7F), delay, noteDur)
	}
}

func scheduleMultiPortNote(portIDs []string, ch, note, vel byte, delay, dur time.Duration) {
	targets := append([]string(nil), portIDs...)
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		sendToPorts(targets, []byte{0x90 | (ch & 0x0F), note, vel})
		time.Sleep(dur)
		sendToPorts(targets, []byte{0x80 | (ch & 0x0F), note, 0})
	}()
}
