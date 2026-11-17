package web

import (
	"fmt"
	"html"
	"strings"

	"github.com/james-see/imagination-engine/pkg/theory"
)

// RenderArrangementSVG generates a standalone, high-resolution SVG chart of the chord progression,
// voice-leading graph, transition tips, and 32-step Crave + East Beast hardware sequencer grids.
func RenderArrangementSVG(arr theory.Arrangement, sliceIdx int) string {
	if sliceIdx < 0 || sliceIdx >= len(arr.Slices) {
		sliceIdx = 0
	}
	var sl theory.SectionSlice
	if len(arr.Slices) > 0 {
		sl = arr.Slices[sliceIdx]
	}

	width := 1280
	height := 860

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" style="background:#0b0f19;font-family:'JetBrains Mono','Fira Code',monospace;">`, width, height, width, height)

	// Definitions & gradients
	sb.WriteString(`
	<defs>
		<linearGradient id="bgGrad" x1="0" y1="0" x2="1" y2="1">
			<stop offset="0%" stop-color="#0b0f19"/>
			<stop offset="100%" stop-color="#111827"/>
		</linearGradient>
		<linearGradient id="cardGrad" x1="0" y1="0" x2="0" y2="1">
			<stop offset="0%" stop-color="#1e293b"/>
			<stop offset="100%" stop-color="#0f172a"/>
		</linearGradient>
		<marker id="arrow" viewBox="0 0 10 10" refX="6" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
			<path d="M 0 1 L 10 5 L 0 9 z" fill="#f59e0b"/>
		</marker>
	</defs>
	<rect width="100%" height="100%" fill="url(#bgGrad)"/>
	`)

	// Header Banner
	fmt.Fprintf(&sb, `<rect x="24" y="20" width="1232" height="92" rx="10" fill="#111827" stroke="#334155" stroke-width="1.5"/>`)
	fmt.Fprintf(&sb, `<text x="44" y="54" fill="#f8fafc" font-size="22" font-weight="bold">%s</text>`, html.EscapeString(arr.Title))
	fmt.Fprintf(&sb, `<text x="44" y="80" fill="#38bdf8" font-size="13">KEY: %s (%s)  |  GENRE: %s  |  BPM: %d  |  ACTIVE SLICE: %s</text>`,
		html.EscapeString(arr.RootKey), html.EscapeString(arr.Mode), html.EscapeString(arr.Genre), arr.BPM, html.EscapeString(sl.Name))
	fmt.Fprintf(&sb, `<text x="44" y="100" fill="#94a3b8" font-size="11.5">%s</text>`, html.EscapeString(truncateStr(arr.TheoryNotes, 145)))

	// Section 1: Chord Progression & Harmonic Transition Graph (y=130..355)
	fmt.Fprintf(&sb, `<text x="28" y="146" fill="#e2e8f0" font-size="14" font-weight="bold">1. HARMONIC PROGRESSION, VOICE-LED EXTENSIONS &amp; TRANSITION BRIDGES</text>`)

	numChords := len(arr.Chords)
	if numChords > 0 {
		cardW := (1232 - (numChords-1)*36) / numChords
		for i, ch := range arr.Chords {
			cx := 24 + i*(cardW+36)
			cy := 160
			chH := 185

			fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="url(#cardGrad)" stroke="#475569" stroke-width="1.5"/>`, cx, cy, cardW, chH)
			// Top accent bar
			accentColors := []string{"#38bdf8", "#a855f7", "#10b981", "#f59e0b"}
			ac := accentColors[i%len(accentColors)]
			fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="%d" height="6" rx="3" fill="%s"/>`, cx, cy, cardW, ac)

			// Chord Symbol & Roman
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#ffffff" font-size="24" font-weight="bold">%s</text>`, cx+16, cy+38, html.EscapeString(ch.Symbol))
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="%s" font-size="14" font-weight="bold">%s</text>`, cx+cardW-16, cy+36, ac, html.EscapeString(ch.Roman))
			// Function
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#cbd5e1" font-size="11.5">%s</text>`, cx+16, cy+58, html.EscapeString(ch.Function))

			// Voicing MIDI Notes
			var noteLabels []string
			for _, m := range ch.VoicingMIDI {
				noteLabels = append(noteLabels, theory.MIDIToNoteName(m))
			}
			fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="%d" height="26" rx="5" fill="#0f172a" stroke="#1e293b"/>`, cx+14, cy+70, cardW-28)
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#38bdf8" font-size="11.5">Voicing: %s</text>`, cx+22, cy+87, html.EscapeString(strings.Join(noteLabels, " · ")))

			// Color Extensions
			colorText := "9th / Modal Color"
			if len(ch.ColorDescriptions) > 0 {
				colorText = strings.Join(ch.ColorDescriptions, " + ")
			}
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#fbbf24" font-size="11">★ %s</text>`, cx+16, cy+115, html.EscapeString(truncateStr(colorText, 36)))

			// Transition Tip wrapped in two lines
			l1, l2 := splitTwoLines(ch.TransitionTip, 38)
			fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#94a3b8" font-size="10.5">→ %s</text>`, cx+16, cy+142, html.EscapeString(l1))
			if l2 != "" {
				fmt.Fprintf(&sb, `<text x="%d" y="%d" fill="#94a3b8" font-size="10.5">  %s</text>`, cx+16, cy+158, html.EscapeString(l2))
			}

			// Draw transition arrow to next chord
			if i < numChords-1 {
				ax1 := cx + cardW + 4
				ax2 := cx + cardW + 30
				ay := cy + chH/2
				fmt.Fprintf(&sb, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#f59e0b" stroke-width="2.5" marker-end="url(#arrow)"/>`, ax1, ay, ax2, ay)
			}
		}
	}

	// Section 2: Behringer Crave 32-Step Monophonic Sequence Grid (Ch 1)
	drawStepGridSVG(&sb, "2. BEHRINGER CRAVE 32-STEP PATTERN (CH 1: BASS / IMPLIED CHORDS / TRANSITION GLIDES)", sl.CraveSteps, 24, 375, 1232, 195, "#f59e0b")

	// Section 3: Cre8audio East Beast 32-Step Harmonic Counter-Voice (Ch 2)
	drawStepGridSVG(&sb, "3. CRE8AUDIO EAST BEAST 32-STEP COUNTER-VOICE (CH 2: COMPLEMENTARY 3RDs / 7THs / 9THs / #11THs)", sl.EastBeastSteps, 24, 595, 1232, 195, "#38bdf8")

	// Footer: Hardware Patch Suggestion
	fmt.Fprintf(&sb, `<rect x="24" y="805" width="1232" height="38" rx="6" fill="#0f172a" stroke="#334155"/>`)
	fmt.Fprintf(&sb, `<text x="38" y="829" fill="#a7f3d0" font-size="11.5">PATCH &amp; ROUTING TIP: %s</text>`, html.EscapeString(truncateStr(arr.PatchSuggestion, 140)))

	sb.WriteString(`</svg>`)
	return sb.String()
}

func drawStepGridSVG(sb *strings.Builder, title string, steps []theory.StepEvent, x, y, w, h int, primaryColor string) {
	fmt.Fprintf(sb, `<text x="%d" y="%d" fill="#e2e8f0" font-size="13.5" font-weight="bold">%s</text>`, x+4, y, html.EscapeString(title))
	boxY := y + 12
	boxH := h - 16
	fmt.Fprintf(sb, `<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="#0f172a" stroke="#334155" stroke-width="1.2"/>`, x, boxY, w, boxH)

	if len(steps) == 0 {
		return
	}
	n := len(steps)
	cellW := float64(w-24) / float64(n)

	// Find min/max MIDI for vertical scaling
	minM, maxM := 127, 0
	for _, st := range steps {
		if !st.Rest {
			if st.MIDINote < minM {
				minM = st.MIDINote
			}
			if st.MIDINote > maxM {
				maxM = st.MIDINote
			}
		}
	}
	if minM >= maxM {
		minM = 36
		maxM = 60
	}
	spanM := float64(maxM - minM + 6)

	for i, st := range steps {
		cx := float64(x+12) + float64(i)*cellW
		// Beat divider every 4 steps, bar divider every 8 steps
		if i > 0 && i%8 == 0 {
			fmt.Fprintf(sb, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#475569" stroke-width="1.5"/>`, cx, boxY+4, cx, boxY+boxH-4)
		} else if i > 0 && i%4 == 0 {
			fmt.Fprintf(sb, `<line x1="%.1f" y1="%d" x2="%.1f" y2="%d" stroke="#1e293b" stroke-width="1"/>`, cx, boxY+4, cx, boxY+boxH-4)
		}

		// Step number at bottom
		fmt.Fprintf(sb, `<text x="%.1f" y="%d" fill="#64748b" font-size="9.5" text-anchor="middle">%02d</text>`, cx+cellW/2, boxY+boxH-8, st.Step)

		if st.Rest {
			fmt.Fprintf(sb, `<text x="%.1f" y="%d" fill="#475569" font-size="10" text-anchor="middle">-</text>`, cx+cellW/2, boxY+boxH/2)
			continue
		}

		// Color by musical role
		fill := primaryColor
		rLower := strings.ToLower(st.Role)
		if strings.Contains(rLower, "approach") || strings.Contains(rLower, "turnaround") || strings.Contains(rLower, "lead ->") {
			fill = "#f43f5e" // Coral/Red for transition & approach notes
		} else if strings.Contains(rLower, "color") || strings.Contains(rLower, "9th") || strings.Contains(rLower, "tension") || strings.Contains(rLower, "burst") {
			fill = "#a855f7" // Purple for color extensions
		} else if strings.Contains(rLower, "pedal") {
			fill = "#10b981" // Emerald for modal pedal points
		}

		norm := float64(st.MIDINote-minM+2) / spanM
		barMaxH := float64(boxH - 48)
		barY := float64(boxY+boxH-26) - norm*barMaxH
		gateFrac := float64(st.Gate+1) / 8.0
		barW := (cellW - 4) * gateFrac
		if barW < 8 {
			barW = 8
		}

		// Note bar
		fmt.Fprintf(sb, `<rect x="%.1f" y="%.1f" width="%.1f" height="18" rx="4" fill="%s" opacity="0.9"/>`, cx+2, barY, barW, fill)

		// Glide connector line
		if st.Glide {
			fmt.Fprintf(sb, `<path d="M %.1f %.1f Q %.1f %.1f %.1f %.1f" stroke="#fbbf24" stroke-width="2" fill="none"/>`,
				cx+barW-2, barY+9, cx+cellW*0.8, barY-4, cx+cellW+4, barY+9)
		}

		// Note Name inside/above bar
		fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" fill="#ffffff" font-size="10" font-weight="bold" text-anchor="middle">%s</text>`, cx+cellW/2, barY+13, html.EscapeString(st.NoteName))

		// Flags (Accent 'A', Glide 'G', Ratchet 'x2..x4')
		var badges []string
		if st.Accent {
			badges = append(badges, "A")
		}
		if st.Glide {
			badges = append(badges, "G")
		}
		if st.Ratchet > 0 {
			badges = append(badges, fmt.Sprintf("x%d", st.Ratchet+1))
		}
		if len(badges) > 0 {
			fmt.Fprintf(sb, `<text x="%.1f" y="%.1f" fill="#fde047" font-size="8.5" text-anchor="middle">%s</text>`, cx+cellW/2, barY-3, strings.Join(badges, ""))
		}
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func splitTwoLines(s string, width int) (string, string) {
	if len(s) <= width {
		return s, ""
	}
	words := strings.Fields(s)
	var l1, l2 []string
	cur := 0
	onFirst := true
	for _, w := range words {
		if onFirst && cur+len(w)+1 <= width {
			l1 = append(l1, w)
			cur += len(w) + 1
		} else {
			onFirst = false
			l2 = append(l2, w)
		}
	}
	return strings.Join(l1, " "), truncateStr(strings.Join(l2, " "), width)
}
