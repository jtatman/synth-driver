//go:build linux

package midiout

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	linuxMu     sync.Mutex
	openHandles = map[string]*os.File{}

	alsaOnce sync.Once
	alsaErr  error
	seqHandle uintptr
	midiDev   uintptr
	myPortID  int32

	sndSeqOpen              func(handle *uintptr, name *byte, streams int32, mode int32) int32
	sndSeqSetClientName     func(seq uintptr, name *byte) int32
	sndSeqCreateSimplePort  func(seq uintptr, name *byte, caps uint32, typeFlags uint32) int32
	sndMidiEventNew         func(bufsize uintptr, rdev *uintptr) int32
	sndMidiEventResetEncode func(dev uintptr)
	sndMidiEventEncodeByte  func(dev uintptr, c int32, ev *byte) int32
	sndSeqEventOutputDirect func(seq uintptr, ev *byte) int32
)

func initALSASeq() error {
	alsaOnce.Do(func() {
		lib, err := purego.Dlopen("libasound.so.2", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			alsaErr = fmt.Errorf("dlopen libasound.so.2: %w", err)
			return
		}
		purego.RegisterLibFunc(&sndSeqOpen, lib, "snd_seq_open")
		purego.RegisterLibFunc(&sndSeqSetClientName, lib, "snd_seq_set_client_name")
		purego.RegisterLibFunc(&sndSeqCreateSimplePort, lib, "snd_seq_create_simple_port")
		purego.RegisterLibFunc(&sndMidiEventNew, lib, "snd_midi_event_new")
		purego.RegisterLibFunc(&sndMidiEventResetEncode, lib, "snd_midi_event_reset_encode")
		purego.RegisterLibFunc(&sndMidiEventEncodeByte, lib, "snd_midi_event_encode_byte")
		purego.RegisterLibFunc(&sndSeqEventOutputDirect, lib, "snd_seq_event_output_direct")

		defName := []byte("default\x00")
		// SND_SEQ_OPEN_DUPLEX = 3 (or SND_SEQ_OPEN_OUTPUT = 1)
		if rc := sndSeqOpen(&seqHandle, &defName[0], 1, 0); rc < 0 {
			alsaErr = fmt.Errorf("snd_seq_open failed: %d", rc)
			return
		}
		cName := []byte("ImaginationEngine\x00")
		sndSeqSetClientName(seqHandle, &cName[0])

		pName := []byte("Imagine MIDI Out\x00")
		// SND_SEQ_PORT_CAP_READ (1) | SND_SEQ_PORT_CAP_SUBS_READ (32) | SND_SEQ_PORT_CAP_WRITE (2)
		myPortID = sndSeqCreateSimplePort(seqHandle, &pName[0], 1|32|2, 2|0x100000)

		if rc := sndMidiEventNew(256, &midiDev); rc < 0 {
			alsaErr = fmt.Errorf("snd_midi_event_new failed: %d", rc)
			return
		}
	})
	return alsaErr
}

var (
	clientLineRe = regexp.MustCompile(`^Client\s+(\d+)\s*:\s*"([^"]+)"`)
	portLineRe   = regexp.MustCompile(`^\s*Port\s+(\d+)\s*:\s*"([^"]+)"\s*\(([^)]+)\)`)
)

// ListHardwarePorts discovers both ALSA Sequencer clients (like Vital, Midi Through, USB synths)
// from /proc/asound/seq/clients AND raw ALSA MIDI devices (/dev/snd/midiC*D*).
func ListHardwarePorts() ([]PortInfo, error) {
	_ = initALSASeq()
	var ports []PortInfo

	// 1. Parse /proc/asound/seq/clients for ALSA Sequencer ports (includes Vital, Midi Through, hardware USB-MIDI)
	if data, err := os.ReadFile("/proc/asound/seq/clients"); err == nil {
		lines := strings.Split(string(data), "\n")
		curClient := -1
		curClientName := ""
		for _, line := range lines {
			if m := clientLineRe.FindStringSubmatch(line); len(m) == 3 {
				curClient, _ = strconv.Atoi(m[1])
				curClientName = m[2]
				continue
			}
			if curClient <= 0 || curClientName == "ImaginationEngine" || strings.HasPrefix(curClientName, "PipeWire-") {
				continue
			}
			if m := portLineRe.FindStringSubmatch(line); len(m) == 4 {
				portNum, _ := strconv.Atoi(m[1])
				portName := m[2]
				flags := m[3]
				// Include any port that accepts writes ('w' or 'W')
				if strings.ContainsAny(flags, "wW") {
					label := fmt.Sprintf("%s : %s (%d:%d)", curClientName, portName, curClient, portNum)
					ports = append(ports, PortInfo{
						ID:      fmt.Sprintf("seq:%d:%d", curClient, portNum),
						Name:    label,
						Backend: "ALSA Sequencer (Linux)",
					})
				}
			}
		}
	}

	// 2. Also include any direct /dev/snd/midiC*D* RawMIDI hardware character devices
	matches, _ := filepath.Glob("/dev/snd/midiC*D*")
	for _, devPath := range matches {
		base := filepath.Base(devPath)
		cardName := resolveALSACardName(base)
		label := base
		if cardName != "" {
			label = fmt.Sprintf("%s (%s)", cardName, base)
		}
		ports = append(ports, PortInfo{
			ID:      "alsa:" + devPath,
			Name:    label,
			Backend: "ALSA RawMIDI (Linux)",
		})
	}
	return ports, nil
}

func resolveALSACardName(midiBase string) string {
	if !strings.HasPrefix(midiBase, "midiC") {
		return ""
	}
	rest := strings.TrimPrefix(midiBase, "midiC")
	dIdx := strings.IndexByte(rest, 'D')
	if dIdx <= 0 {
		return ""
	}
	cardNum := rest[:dIdx]
	idBytes, err := os.ReadFile(fmt.Sprintf("/proc/asound/card%s/id", cardNum))
	if err == nil {
		return strings.TrimSpace(string(idBytes))
	}
	return ""
}

// SendHardwareMIDI transmits raw MIDI bytes to either an ALSA Sequencer client:port ("seq:128:0")
// or an ALSA RawMIDI device ("alsa:/dev/snd/midiC1D0").
func SendHardwareMIDI(portID string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if strings.HasPrefix(portID, "seq:") {
		if err := initALSASeq(); err != nil {
			return err
		}
		parts := strings.Split(strings.TrimPrefix(portID, "seq:"), ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid ALSA seq port ID %q", portID)
		}
		destClient, _ := strconv.Atoi(parts[0])
		destPort, _ := strconv.Atoi(parts[1])

		linuxMu.Lock()
		defer linuxMu.Unlock()

		ev := make([]byte, 64)
		sndMidiEventResetEncode(midiDev)
		for _, b := range data {
			rc := sndMidiEventEncodeByte(midiDev, int32(b), &ev[0])
			if rc == 1 {
				// Set SND_SEQ_QUEUE_DIRECT (253), source port, and destination client:port
				ev[3] = 253
				ev[13] = byte(myPortID)
				ev[14] = byte(destClient)
				ev[15] = byte(destPort)
				sndSeqEventOutputDirect(seqHandle, &ev[0])
			}
		}
		return nil
	}

	devPath := strings.TrimPrefix(portID, "alsa:")
	linuxMu.Lock()
	defer linuxMu.Unlock()

	f, ok := openHandles[devPath]
	if !ok || f == nil {
		var err error
		f, err = os.OpenFile(devPath, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("open ALSA MIDI device %s: %w", devPath, err)
		}
		openHandles[devPath] = f
	}
	_, err := f.Write(data)
	if err != nil {
		f.Close()
		delete(openHandles, devPath)
		return err
	}
	return nil
}
