package midiout

// PortInfo describes a hardware or virtual MIDI output destination discovered by the Go backend.
type PortInfo struct {
	ID      string `json:"id"`      // e.g., "coremidi:0" or "alsa:/dev/snd/midiC1D0"
	Name    string `json:"name"`    // e.g., "CRAVE" or "Behringer CRAVE MIDI 1"
	Backend string `json:"backend"` // "CoreMIDI (macOS)" or "ALSA (Linux)"
}
