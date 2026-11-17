package seq

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/james-see/imagination-engine/pkg/theory"
)

// Fixed 32-byte SynthTribe header for Behringer CRAVE (v1.1.1)
var craveHeader = []byte{
	0x23, 0x98, 0x54, 0x76, // Magic bytes
	0x00, 0x00, 0x00, 0x0a, // Device name length (10 bytes UTF-16BE)
	0x00, 0x43, 0x00, 0x52, 0x00, 0x41, 0x00, 0x56, 0x00, 0x45, // "CRAVE"
	0x00, 0x00, 0x00, 0x0a, // Version length (10 bytes UTF-16BE)
	0x00, 0x31, 0x00, 0x2e, 0x00, 0x31, 0x00, 0x2e, 0x00, 0x31, // "1.1.1"
}

// Fixed 36-byte SynthTribe header for Behringer TD-3 (v1.3.7) matching synthtribe2midi
var td3Header = []byte{
	0x23, 0x98, 0x54, 0x76, // Magic bytes
	0x00, 0x00, 0x00, 0x08, // Device name length (8 bytes)
	0x00, 0x54, 0x00, 0x44, 0x00, 0x2d, 0x00, 0x33, // "TD-3"
	0x00, 0x00, 0x00, 0x0a, // Version length (10 bytes)
	0x00, 0x31, 0x00, 0x2e, 0x00, 0x33, 0x00, 0x2e, 0x00, 0x37, 0x00, 0x00, // "1.3.7\0"
	0x00, 0x70, 0x00, 0x00, // Remaining payload length (0x70 = 112 bytes)
}

// EncodeCraveSeq encodes a SectionSlice's CraveSteps (or EastBeastSteps) into a native
// Behringer Crave SynthTribe .seq file (up to 32 steps with gate, ratchet, velocity, glide, accent).
func EncodeCraveSeq(steps []theory.StepEvent, swing int) []byte {
	n := len(steps)
	if n == 0 {
		n = 1
		steps = []theory.StepEvent{{Step: 1, MIDINote: 48, Gate: 4, Velocity: 100}}
	}
	if n > 32 {
		n = 32
		steps = steps[:32]
	}
	if swing < 50 {
		swing = 50
	}
	if swing > 75 {
		swing = 75
	}

	var buf bytes.Buffer
	buf.Write(craveHeader)

	// 2 padding bytes (0x00, 0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)

	// Byte length field: 0x0e + (seqLength - 1) * 8
	bytesLength := 0x0e + (n-1)*8
	buf.WriteByte(byte(bytesLength >> 8))
	buf.WriteByte(byte(bytesLength & 0xff))

	// Swing field (2 bytes, nibble-encoded offset from 50%)
	swingOffset := swing - 50
	buf.WriteByte(byte(swingOffset / 16))
	buf.WriteByte(byte(swingOffset % 16))

	// Sequence length (4 bytes: 0x00, (n-1)/8, 0x00, (n-1)%8)
	seqLen := n - 1
	buf.WriteByte(0x00)
	buf.WriteByte(byte(seqLen / 8))
	buf.WriteByte(0x00)
	buf.WriteByte(byte(seqLen % 8))

	// Write 8 bytes per step
	for i := 0; i < n; i++ {
		st := steps[i]
		// Crave note value: MIDI note clamped to valid Crave range (24..84)
		noteVal := st.MIDINote
		if noteVal < 12 {
			noteVal = 12
		}
		if noteVal > 96 {
			noteVal = 96
		}
		buf.WriteByte(byte(noteVal / 16))
		buf.WriteByte(byte(noteVal % 16))

		gate := st.Gate
		if gate < 0 {
			gate = 0
		}
		if gate > 7 {
			gate = 7
		}
		buf.WriteByte(byte(gate))

		ratchet := st.Ratchet
		if ratchet < 0 {
			ratchet = 0
		}
		if ratchet > 3 {
			ratchet = 3
		}
		buf.WriteByte(byte(ratchet))

		vel := st.Velocity
		if vel < 1 {
			vel = 64
		}
		if vel > 127 {
			vel = 127
		}
		buf.WriteByte(byte(vel / 16))
		buf.WriteByte(byte(vel % 16))

		// Effects byte:
		// 0x01 = Glide
		// 0x04 = Accent
		// 0x08 = Step Enabled (cleared when Rest is true)
		var effects byte
		if st.Glide {
			effects |= 0x01
		}
		if st.Accent {
			effects |= 0x04
		}
		if !st.Rest {
			effects |= 0x08
		}
		buf.WriteByte(effects)
		buf.WriteByte(0x00)
	}

	return buf.Bytes()
}

// EncodeTD3Seq encodes up to 16 steps into the 146-byte TD-3 .seq format (compatible with synthtribe2midi).
func EncodeTD3Seq(steps []theory.StepEvent) []byte {
	if len(steps) > 16 {
		steps = steps[:16]
	}
	for len(steps) < 16 {
		steps = append(steps, theory.StepEvent{Step: len(steps) + 1, MIDINote: 36, Gate: 4, Velocity: 100, Rest: true})
	}

	out := make([]byte, 146)
	copy(out[0:36], td3Header)

	var tieMask uint16
	var restMask uint16

	for i := 0; i < 16; i++ {
		st := steps[i]
		// TD-3 note offset: MIDI note - 24 (clamped 0..48)
		noteVal := st.MIDINote - 24
		if noteVal < 0 {
			noteVal += 12
		}
		if noteVal < 0 {
			noteVal = 0
		}
		if noteVal > 48 {
			noteVal = 48
		}

		// Notes at 0x24 (32 bytes)
		out[0x24+i*2] = byte(noteVal / 16)
		out[0x24+i*2+1] = byte(noteVal % 16)

		// Accents at 0x44 (32 bytes)
		if st.Accent {
			out[0x44+i*2+1] = 0x01
		}

		// Slides at 0x64 (32 bytes)
		if st.Glide {
			out[0x64+i*2+1] = 0x01
		}

		// In TD-3 format, bit=1 in tieMask means trigger new pitch (not tied sustain),
		// and bit=1 in restMask means note enabled (not rest)
		tieMask |= (1 << i)
		if !st.Rest {
			restMask |= (1 << i)
		}
	}

	// 0x84: Triplet flag (0x00, 0x00)
	// 0x86: Sequence length = 16 (0x01, 0x00)
	out[0x86] = 0x01
	out[0x87] = 0x00

	// 0x8A: Tie bitmask (4 nibble bytes)
	out[0x8A] = byte((tieMask >> 4) & 0x0F)
	out[0x8B] = byte(tieMask & 0x0F)
	out[0x8C] = byte((tieMask >> 12) & 0x0F)
	out[0x8D] = byte((tieMask >> 8) & 0x0F)

	// 0x8E: Rest/Enable bitmask (4 nibble bytes)
	out[0x8E] = byte((restMask >> 4) & 0x0F)
	out[0x8F] = byte(restMask & 0x0F)
	out[0x90] = byte((restMask >> 12) & 0x0F)
	out[0x91] = byte((restMask >> 8) & 0x0F)

	return out
}

// InspectCraveSeq parses a binary Crave .seq file and returns a formatted ASCII step table.
func InspectCraveSeq(data []byte) (string, error) {
	if len(data) < 42 {
		return "", fmt.Errorf("file too short (%d bytes) for Crave .seq", len(data))
	}
	if !bytes.Equal(data[:4], []byte{0x23, 0x98, 0x54, 0x76}) {
		return "", fmt.Errorf("invalid SynthTribe magic bytes")
	}

	swing := 50 + int(data[36])*16 + int(data[37])
	seqLen := int(data[39])*8 + int(data[41]) + 1

	var sb strings.Builder
	fmt.Fprintf(&sb, "Behringer CRAVE .seq Pattern | Length: %d steps | Swing: %d%%\n", seqLen, swing)
	sb.WriteString("-------------------------------------------------------------\n")
	sb.WriteString("Step | Note | Vel | Ratchet | Flags | Gate\n")
	sb.WriteString("-------------------------------------------------------------\n")

	offset := 42
	for i := 0; i < seqLen && offset+8 <= len(data); i++ {
		st := data[offset : offset+8]
		noteVal := int(st[0])*16 + int(st[1])
		gate := int(st[2])
		ratchet := int(st[3]) + 1
		vel := int(st[4])*16 + int(st[5])
		eff := st[6]

		glide := (eff & 0x01) != 0
		accent := (eff & 0x04) != 0
		enabled := (eff & 0x08) != 0

		gChar := ' '
		if glide {
			gChar = 'G'
		}
		aChar := ' '
		if accent {
			aChar = 'A'
		}
		rChar := ' '
		if !enabled {
			rChar = 'R'
		}

		gateBar := strings.Repeat("#", gate+1) + strings.Repeat(".", 7-gate)
		fmt.Fprintf(&sb, " %02d  | %-4s | %3d |   x%d    | %c %c %c | [%s]\n",
			i+1, theory.MIDIToNoteName(noteVal), vel, ratchet, gChar, aChar, rChar, gateBar)
		offset += 8
	}
	return sb.String(), nil
}
