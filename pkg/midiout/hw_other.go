//go:build !darwin && !linux

package midiout

import "fmt"

func ListHardwarePorts() ([]PortInfo, error) {
	return nil, nil
}

func SendHardwareMIDI(portID string, data []byte) error {
	return fmt.Errorf("native hardware MIDI not implemented on this OS")
}
