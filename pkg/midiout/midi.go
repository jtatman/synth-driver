package midiout

import (
	"bytes"
	"encoding/binary"

	"github.com/james-see/imagination-engine/pkg/theory"
)

const ppqn = 480 // Ticks per quarter note (16th note = 120 ticks)

type midiEvent struct {
	absTick int
	data    []byte
}

// orderedSlicesForExport returns the full 9-stage Song Journey sequence of slices
// if SongChain is present, or the 5 base slices otherwise.
func orderedSlicesForExport(arr theory.Arrangement) []theory.SectionSlice {
	if len(arr.SongChain) > 0 && len(arr.Slices) > 0 {
		var out []theory.SectionSlice
		for _, stage := range arr.SongChain {
			if stage.SliceIndex >= 0 && stage.SliceIndex < len(arr.Slices) {
				sl := arr.Slices[stage.SliceIndex]
				sl.Name = stage.Label
				out = append(out, sl)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return arr.Slices
}

// EncodeMultiTrackMIDI builds a Format-1 Standard MIDI File (.mid) with 5 tracks:
//   - Track 0: Conductor (Tempo, 4/4 Time Signature, Song Journey Stage & Chord Markers)
//   - Track 1 (Ch 1): Behringer Crave Sub/Bass Sequence (+ MIDI CC#1 Mod & CC#74 Cutoff automation)
//   - Track 2 (Ch 2): Cre8audio East Beast High/Midrange Harmonic Line (+ MIDI CC#1 & CC#74 automation)
//   - Track 3 (Ch 3): Behringer Edge Constant-Pitch Quarter-Note Beat Pulse
//   - Track 4 (Ch 4 / Custom): 2-to-3 Note Voice-Led Polyphonic Pad Chords (+ MIDI CC#1 & CC#74 automation)
func EncodeMultiTrackMIDI(arr theory.Arrangement) []byte {
	bpm := arr.BPM
	if bpm < 40 {
		bpm = 86
	}

	vc := arr.VoiceConfig
	if vc.CraveChannel == 0 && vc.EastBeastChannel == 0 && vc.EdgeChannel == 0 && vc.PolyChannel == 0 {
		vc = theory.DefaultVoiceSettings()
		vc.PolyChannel = 3 // Ch 4 for DAW multi-track separation unless customized
	}

	exportSlices := orderedSlicesForExport(arr)

	tracks := [][]byte{
		buildConductorTrack(arr, exportSlices, bpm),
		buildMonoSynthTrack("Behringer Crave (Sub/Bass Anchor + CC#1/CC#74 Mod)", byte(vc.CraveChannel&0x0F), exportSlices),
		buildMonoSynthTrack("East Beast (High/Mid Counter-Voice + CC#1/CC#74 Mod)", byte(vc.EastBeastChannel&0x0F), extractEastBeastSlices(exportSlices)),
		buildEdgeTrack("Behringer Edge (Constant-Pitch Beat Pulse)", byte(vc.EdgeChannel&0x0F), exportSlices),
		buildPolyChordTrack("Polyphonic 2-3 Note Pad Shells (Vital / Surge XT)", byte(vc.PolyChannel&0x0F), arr, exportSlices),
	}

	var out bytes.Buffer
	out.WriteString("MThd")
	binary.Write(&out, binary.BigEndian, uint32(6))
	binary.Write(&out, binary.BigEndian, uint16(1)) // Format 1 (multi-track)
	binary.Write(&out, binary.BigEndian, uint16(len(tracks)))
	binary.Write(&out, binary.BigEndian, uint16(ppqn))

	for _, trk := range tracks {
		out.WriteString("MTrk")
		binary.Write(&out, binary.BigEndian, uint32(len(trk)))
		out.Write(trk)
	}
	return out.Bytes()
}

func buildConductorTrack(arr theory.Arrangement, slices []theory.SectionSlice, bpm int) []byte {
	var events []midiEvent

	events = append(events, metaTextEvent(0, 0x03, arr.Title+" ("+arr.Genre+")"))

	usPerQN := 60000000 / bpm
	events = append(events, midiEvent{
		absTick: 0,
		data: []byte{
			0xFF, 0x51, 0x03,
			byte((usPerQN >> 16) & 0xFF),
			byte((usPerQN >> 8) & 0xFF),
			byte(usPerQN & 0xFF),
		},
	})

	// 4/4 Time Signature
	events = append(events, midiEvent{
		absTick: 0,
		data:    []byte{0xFF, 0x58, 0x04, 0x04, 0x02, 0x18, 0x08},
	})

	stepTicks := ppqn / 4 // 120 ticks per step
	curTick := 0
	for _, sl := range slices {
		events = append(events, metaTextEvent(curTick, 0x06, "["+sl.Name+"]"))
		stepsPerChord := len(sl.CraveSteps) / maxInt(1, len(arr.Chords))
		for cIdx, ch := range arr.Chords {
			chordTick := curTick + cIdx*stepsPerChord*stepTicks
			events = append(events, metaTextEvent(chordTick, 0x05, ch.Symbol+" ("+ch.Roman+" - "+ch.Function+")"))
		}
		curTick += len(sl.CraveSteps) * stepTicks
	}

	return serializeEvents(events)
}

func buildMonoSynthTrack(trackName string, channel byte, slices []theory.SectionSlice) []byte {
	var events []midiEvent
	events = append(events, metaTextEvent(0, 0x03, trackName))

	stepTicks := ppqn / 4
	curTick := 0
	lastCC1 := -1
	lastCC74 := -1

	for _, sl := range slices {
		for _, st := range sl.CraveSteps {
			// Write MIDI CC#1 (Mod Wheel) and CC#74 (Filter Cutoff) automation when changed
			if st.ModCC1 != lastCC1 {
				events = append(events, midiEvent{
					absTick: curTick,
					data:    []byte{0xB0 | (channel & 0x0F), 1, byte(st.ModCC1 & 0x7F)},
				})
				lastCC1 = st.ModCC1
			}
			if st.CutoffCC74 != lastCC74 {
				events = append(events, midiEvent{
					absTick: curTick,
					data:    []byte{0xB0 | (channel & 0x0F), 74, byte(st.CutoffCC74 & 0x7F)},
				})
				lastCC74 = st.CutoffCC74
			}

			if st.Rest {
				curTick += stepTicks
				continue
			}

			ratchetCount := st.Ratchet + 1
			if ratchetCount < 1 {
				ratchetCount = 1
			}
			if ratchetCount > 4 {
				ratchetCount = 4
			}

			if ratchetCount > 1 {
				subTick := stepTicks / ratchetCount
				dur := (subTick * 3) / 4
				if dur < 10 {
					dur = 10
				}
				for r := 0; r < ratchetCount; r++ {
					onTick := curTick + r*subTick
					offTick := onTick + dur
					events = append(events,
						midiEvent{absTick: onTick, data: []byte{0x90 | (channel & 0x0F), byte(st.MIDINote), byte(st.Velocity)}},
						midiEvent{absTick: offTick, data: []byte{0x80 | (channel & 0x0F), byte(st.MIDINote), 0x00}},
					)
				}
			} else {
				gateRatio := st.Gate + 1
				dur := (stepTicks * gateRatio) / 8
				if st.Glide {
					dur = stepTicks + 15
				} else if dur >= stepTicks {
					dur = stepTicks - 6
				}
				if dur < 15 {
					dur = 15
				}
				events = append(events,
					midiEvent{absTick: curTick, data: []byte{0x90 | (channel & 0x0F), byte(st.MIDINote), byte(st.Velocity)}},
					midiEvent{absTick: curTick + dur, data: []byte{0x80 | (channel & 0x0F), byte(st.MIDINote), 0x00}},
				)
			}

			curTick += stepTicks
		}
	}

	return serializeEvents(events)
}

func extractEastBeastSlices(slices []theory.SectionSlice) []theory.SectionSlice {
	out := make([]theory.SectionSlice, len(slices))
	for i, s := range slices {
		cp := s
		cp.CraveSteps = s.EastBeastSteps
		out[i] = cp
	}
	return out
}

func buildEdgeTrack(trackName string, channel byte, slices []theory.SectionSlice) []byte {
	var events []midiEvent
	events = append(events, metaTextEvent(0, 0x03, trackName))

	stepTicks := ppqn / 4
	curTick := 0
	for _, sl := range slices {
		if len(sl.EdgeSteps) == 0 {
			curTick += len(sl.CraveSteps) * stepTicks
			continue
		}
		for i := 0; i < len(sl.CraveSteps); i++ {
			st := sl.EdgeSteps[i%len(sl.EdgeSteps)]
			if !st.Rest {
				dur := stepTicks / 2
				events = append(events,
					midiEvent{absTick: curTick, data: []byte{0x90 | (channel & 0x0F), byte(st.MIDINote), byte(st.Velocity)}},
					midiEvent{absTick: curTick + dur, data: []byte{0x80 | (channel & 0x0F), byte(st.MIDINote), 0x00}},
				)
			}
			curTick += stepTicks
		}
	}
	return serializeEvents(events)
}

func buildPolyChordTrack(trackName string, channel byte, arr theory.Arrangement, slices []theory.SectionSlice) []byte {
	var events []midiEvent
	events = append(events, metaTextEvent(0, 0x03, trackName))

	stepTicks := ppqn / 4
	curTick := 0
	lastCC1 := -1
	lastCC74 := -1

	for _, sl := range slices {
		// Write step-level CC#1 and CC#74 automation for the pad synth too
		for sIdx, st := range sl.CraveSteps {
			t := curTick + sIdx*stepTicks
			if st.ModCC1 != lastCC1 {
				events = append(events, midiEvent{
					absTick: t,
					data:    []byte{0xB0 | (channel & 0x0F), 1, byte(st.ModCC1 & 0x7F)},
				})
				lastCC1 = st.ModCC1
			}
			if st.CutoffCC74 != lastCC74 {
				events = append(events, midiEvent{
					absTick: t,
					data:    []byte{0xB0 | (channel & 0x0F), 74, byte(st.CutoffCC74 & 0x7F)},
				})
				lastCC74 = st.CutoffCC74
			}
		}

		totalTicks := len(sl.CraveSteps) * stepTicks
		if len(arr.Chords) == 0 {
			curTick += totalTicks
			continue
		}
		chordDur := totalTicks / len(arr.Chords)
		for _, ch := range arr.Chords {
			for _, note := range ch.VoicingMIDI {
				events = append(events,
					midiEvent{absTick: curTick, data: []byte{0x90 | (channel & 0x0F), byte(note), 84}},
					midiEvent{absTick: curTick + chordDur - 12, data: []byte{0x80 | (channel & 0x0F), byte(note), 0x00}},
				)
			}
			curTick += chordDur
		}
	}

	return serializeEvents(events)
}

func metaTextEvent(absTick int, metaType byte, text string) midiEvent {
	b := []byte(text)
	var data []byte
	data = append(data, 0xFF, metaType)
	data = append(data, encodeVarLen(len(b))...)
	data = append(data, b...)
	return midiEvent{absTick: absTick, data: data}
}

func serializeEvents(events []midiEvent) []byte {
	for i := 1; i < len(events); i++ {
		for j := i; j > 0; j-- {
			a := events[j-1]
			b := events[j]
			swap := false
			if b.absTick < a.absTick {
				swap = true
			} else if b.absTick == a.absTick && len(b.data) > 0 && len(a.data) > 0 {
				if (b.data[0]&0xF0) == 0x80 && (a.data[0]&0xF0) == 0x90 {
					swap = true
				}
			}
			if swap {
				events[j-1], events[j] = events[j], events[j-1]
			} else {
				break
			}
		}
	}

	var buf bytes.Buffer
	prevTick := 0
	for _, ev := range events {
		delta := ev.absTick - prevTick
		if delta < 0 {
			delta = 0
		}
		buf.Write(encodeVarLen(delta))
		buf.Write(ev.data)
		prevTick = ev.absTick
	}
	buf.Write([]byte{0x00, 0xFF, 0x2F, 0x00})
	return buf.Bytes()
}

func encodeVarLen(v int) []byte {
	if v < 0 {
		v = 0
	}
	var bytesRev []byte
	bytesRev = append(bytesRev, byte(v&0x7F))
	v >>= 7
	for v > 0 {
		bytesRev = append(bytesRev, byte((v&0x7F)|0x80))
		v >>= 7
	}
	for i, j := 0, len(bytesRev)-1; i < j; i, j = i+1, j-1 {
		bytesRev[i], bytesRev[j] = bytesRev[j], bytesRev[i]
	}
	return bytesRev
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
