//go:build darwin

package midiout

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const kCFStringEncodingUTF8 uint32 = 0x08000100

var (
	darwinOnce sync.Once
	darwinErr  error
	sendMu     sync.Mutex

	cfStringCreateWithCString func(alloc uintptr, cStr *byte, encoding uint32) uintptr
	cfStringGetCString        func(theString uintptr, buffer *byte, bufferSize int64, encoding uint32) bool
	cfRelease                 func(cf uintptr)

	midiGetNumberOfDestinations func() uintptr
	midiGetDestination          func(destIndex0 uintptr) uint32
	midiObjectGetStringProperty func(obj uint32, propertyID uintptr, str *uintptr) int32
	midiClientCreate            func(name uintptr, notifyProc uintptr, notifyRefCon uintptr, outClient *uint32) int32
	midiOutputPortCreate        func(client uint32, portName uintptr, outPort *uint32) int32
	midiPacketListInit          func(pktlist *byte) uintptr
	midiPacketListAdd           func(pktlist *byte, listSize uintptr, curPacket uintptr, time uint64, nData uintptr, data *byte) uintptr
	midiSend                    func(port uint32, dest uint32, pktlist *byte) int32
	midiSourceCreate            func(client uint32, name uintptr, outSrc *uint32) int32
	midiReceived                func(src uint32, pktlist *byte) int32

	propDisplayName uintptr
	propName        uintptr
	propModel       uintptr

	midiClient          uint32
	midiOutPort         uint32
	midiVirtualSrc      uint32 // Multi-Channel Virtual Out (for Surge XT / Logic / Cubase)
	midiVirtualVitalSrc uint32 // Pre-Filtered Pad-Only Virtual Out (for Vital Omni mode!)
)

func initCoreMIDI() error {
	darwinOnce.Do(func() {
		cfLib, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			darwinErr = fmt.Errorf("dlopen CoreFoundation: %w", err)
			return
		}
		cmLib, err := purego.Dlopen("/System/Library/Frameworks/CoreMIDI.framework/CoreMIDI", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			darwinErr = fmt.Errorf("dlopen CoreMIDI: %w", err)
			return
		}

		purego.RegisterLibFunc(&cfStringCreateWithCString, cfLib, "CFStringCreateWithCString")
		purego.RegisterLibFunc(&cfStringGetCString, cfLib, "CFStringGetCString")
		purego.RegisterLibFunc(&cfRelease, cfLib, "CFRelease")

		purego.RegisterLibFunc(&midiGetNumberOfDestinations, cmLib, "MIDIGetNumberOfDestinations")
		purego.RegisterLibFunc(&midiGetDestination, cmLib, "MIDIGetDestination")
		purego.RegisterLibFunc(&midiObjectGetStringProperty, cmLib, "MIDIObjectGetStringProperty")
		purego.RegisterLibFunc(&midiClientCreate, cmLib, "MIDIClientCreate")
		purego.RegisterLibFunc(&midiOutputPortCreate, cmLib, "MIDIOutputPortCreate")
		purego.RegisterLibFunc(&midiPacketListInit, cmLib, "MIDIPacketListInit")
		purego.RegisterLibFunc(&midiPacketListAdd, cmLib, "MIDIPacketListAdd")
		purego.RegisterLibFunc(&midiSend, cmLib, "MIDISend")
		purego.RegisterLibFunc(&midiSourceCreate, cmLib, "MIDISourceCreate")
		purego.RegisterLibFunc(&midiReceived, cmLib, "MIDIReceived")

		propDisplayName = loadCFStringConstant(cmLib, "kMIDIPropertyDisplayName", "displayName")
		propName = loadCFStringConstant(cmLib, "kMIDIPropertyName", "name")
		propModel = loadCFStringConstant(cmLib, "kMIDIPropertyModel", "model")

		clientName := makeCFString("ImaginationEngineClient")
		if st := midiClientCreate(clientName, 0, 0, &midiClient); st != 0 {
			darwinErr = fmt.Errorf("MIDIClientCreate failed with OSStatus %d", st)
			return
		}
		portName := makeCFString("ImaginationEngineOutPort")
		if st := midiOutputPortCreate(midiClient, portName, &midiOutPort); st != 0 {
			darwinErr = fmt.Errorf("MIDIOutputPortCreate failed with OSStatus %d", st)
			return
		}

		// 1. Pre-filtered Pad-Only Virtual Port so Vital (in Omni mode) never hears Crave or East Beast!
		vitalName := makeCFString("Imagine Pad Only (Vital Omni-Safe)")
		_ = midiSourceCreate(midiClient, vitalName, &midiVirtualVitalSrc)

		// 2. Full Multi-Channel Virtual Port for Surge XT (channel-selectable) & DAWs
		virtName := makeCFString("Imagine Multi-Ch (Surge XT / DAW)")
		_ = midiSourceCreate(midiClient, virtName, &midiVirtualSrc)
	})
	return darwinErr
}

func loadCFStringConstant(lib uintptr, symName, fallback string) uintptr {
	addr, err := purego.Dlsym(lib, symName)
	if err == nil && addr != 0 {
		ptr := *(*uintptr)(unsafe.Pointer(addr))
		if ptr != 0 {
			return ptr
		}
	}
	return makeCFString(fallback)
}

func makeCFString(s string) uintptr {
	b := append([]byte(s), 0)
	return cfStringCreateWithCString(0, &b[0], kCFStringEncodingUTF8)
}

func cfStringToGo(cfStr uintptr) string {
	if cfStr == 0 {
		return ""
	}
	buf := make([]byte, 256)
	if !cfStringGetCString(cfStr, &buf[0], int64(len(buf)), kCFStringEncodingUTF8) {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

func getEndpointProperty(ep uint32, prop uintptr) string {
	if prop == 0 {
		return ""
	}
	var cfStr uintptr
	st := midiObjectGetStringProperty(ep, prop, &cfStr)
	if st != 0 || cfStr == 0 {
		return ""
	}
	defer cfRelease(cfStr)
	return strings.TrimSpace(cfStringToGo(cfStr))
}

// ListHardwarePorts returns all active CoreMIDI output destinations plus the two Virtual Sources on macOS:
//  1. "Imagine Pad Only (Vital Omni-Safe)" — pre-filtered so Vital only receives Poly Pad chords
//  2. "Imagine Multi-Ch (Surge XT / DAW)" — carries all channels for Surge XT / Logic / Cubase
func ListHardwarePorts() ([]PortInfo, error) {
	if err := initCoreMIDI(); err != nil {
		return nil, err
	}
	n := int(midiGetNumberOfDestinations())
	var ports []PortInfo
	for i := 0; i < n; i++ {
		ep := midiGetDestination(uintptr(i))
		if ep == 0 {
			continue
		}
		disp := getEndpointProperty(ep, propDisplayName)
		name := getEndpointProperty(ep, propName)
		model := getEndpointProperty(ep, propModel)

		label := disp
		if label == "" {
			label = name
		}
		if model != "" && !strings.Contains(strings.ToLower(label), strings.ToLower(model)) {
			label = model + " - " + label
		}
		if label == "" {
			label = fmt.Sprintf("CoreMIDI Destination #%d", i+1)
		}
		ports = append(ports, PortInfo{
			ID:      fmt.Sprintf("coremidi:%d", i),
			Name:    label,
			Backend: "CoreMIDI (macOS)",
		})
	}
	if midiVirtualVitalSrc != 0 {
		ports = append(ports, PortInfo{
			ID:      "coremidi:virtual_vital",
			Name:    "Imagine Pad Only (Vital Omni-Safe — Absorbs Ch 1 & 2)",
			Backend: "CoreMIDI Virtual Filter (macOS)",
		})
	}
	if midiVirtualSrc != 0 {
		ports = append(ports, PortInfo{
			ID:      "coremidi:virtual",
			Name:    "Imagine Multi-Ch (Surge XT / Logic / GarageBand)",
			Backend: "CoreMIDI Virtual (macOS)",
		})
	}
	return ports, nil
}

// SendHardwareMIDI transmits raw MIDI bytes (Note On/Off, CC, Clock, Start/Stop) to a CoreMIDI destination or virtual source.
func SendHardwareMIDI(portID string, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if err := initCoreMIDI(); err != nil {
		return err
	}

	sendMu.Lock()
	defer sendMu.Unlock()

	pktListBuf := make([]byte, 512)
	curPkt := midiPacketListInit(&pktListBuf[0])
	curPkt = midiPacketListAdd(&pktListBuf[0], uintptr(len(pktListBuf)), curPkt, 0, uintptr(len(data)), &data[0])
	if curPkt == 0 {
		return fmt.Errorf("MIDIPacketListAdd failed")
	}

	if portID == "coremidi:virtual_vital" {
		if midiVirtualVitalSrc == 0 {
			return fmt.Errorf("CoreMIDI Vital virtual source not available")
		}
		if st := midiReceived(midiVirtualVitalSrc, &pktListBuf[0]); st != 0 {
			return fmt.Errorf("MIDIReceived (vital) failed with OSStatus %d", st)
		}
		return nil
	}

	if portID == "coremidi:virtual" {
		if midiVirtualSrc == 0 {
			return fmt.Errorf("CoreMIDI virtual source not available")
		}
		if st := midiReceived(midiVirtualSrc, &pktListBuf[0]); st != 0 {
			return fmt.Errorf("MIDIReceived failed with OSStatus %d", st)
		}
		return nil
	}

	idxStr := strings.TrimPrefix(portID, "coremidi:")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return fmt.Errorf("invalid CoreMIDI port ID %q: %w", portID, err)
	}

	n := int(midiGetNumberOfDestinations())
	if idx < 0 || idx >= n {
		return fmt.Errorf("CoreMIDI destination index %d out of range (0..%d)", idx, n-1)
	}
	dest := midiGetDestination(uintptr(idx))
	if dest == 0 {
		return fmt.Errorf("CoreMIDI destination %d is unavailable", idx)
	}

	if st := midiSend(midiOutPort, dest, &pktListBuf[0]); st != 0 {
		return fmt.Errorf("MIDISend failed with OSStatus %d", st)
	}
	return nil
}
