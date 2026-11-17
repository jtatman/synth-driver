package seq

import (
	"bytes"
	"strings"
	"testing"

	"github.com/james-see/imagination-engine/pkg/theory"
)

func TestCraveSeqRoundTrip(t *testing.T) {
	chords := []theory.ChordSpec{
		theory.ParseChordSymbol("Em11", "im11", "Tonic", "Bridge 1"),
		theory.ParseChordSymbol("Cmaj7#11", "bVImaj7#11", "Submediant", "Bridge 2"),
		theory.ParseChordSymbol("F#m7b5", "iiø7", "Supertonic", "Bridge 3"),
		theory.ParseChordSymbol("B7#9", "V7#9", "Dominant", "Bridge 4"),
	}
	slices := theory.BuildArrangementSlices(chords, "Trance", 0.9, 0.9, 42)
	if len(slices) != 5 {
		t.Fatalf("expected 5 slices, got %d", len(slices))
	}

	raw := EncodeCraveSeq(slices[2].CraveSteps, slices[2].Swing)
	// 32-step Crave .seq file should be 42 header/info bytes + 32*8 step bytes = 298 bytes
	if len(raw) != 298 {
		t.Fatalf("expected 298 bytes for 32-step Crave .seq, got %d", len(raw))
	}
	if !bytes.Equal(raw[:4], []byte{0x23, 0x98, 0x54, 0x76}) {
		t.Fatalf("invalid magic header")
	}

	summary, err := InspectCraveSeq(raw)
	if err != nil {
		t.Fatalf("InspectCraveSeq failed: %v", err)
	}
	if !strings.Contains(summary, "Length: 32 steps") {
		t.Fatalf("expected 32 steps in summary, got:\n%s", summary)
	}

	td3Raw := EncodeTD3Seq(slices[0].CraveSteps)
	if len(td3Raw) != 146 {
		t.Fatalf("expected 146 bytes for TD-3 .seq, got %d", len(td3Raw))
	}
}
