package slider

import "strings"

// renderIntegratedSlider renders the slider track with silence brackets.
// Returns two rows: top = track chars, bottom = silence brackets.
func (m Model) renderIntegratedSlider(width int) (topRow, bottomRow string) {
	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	// Pre-pass: find silence run boundaries for bracket markers.
	silenceStart := make(map[int]bool, width)
	silenceEnd := make(map[int]bool, width)
	prevSilence := false
	for i := range width {
		s := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		if s && !prevSilence {
			silenceStart[i] = true
		}
		if !s && prevSilence {
			silenceEnd[i-1] = true
		}
		prevSilence = s
	}
	if prevSilence {
		silenceEnd[width-1] = true
	}

	var top, bot strings.Builder

	for i := range width {
		silence := m.isSilenceAt(float64(i) / float64(width) * m.duration)

		// Handle positions - same vertical bar in both rows.
		if i == startIdx {
			if m.adjustingStart {
				top.WriteString(handleActiveStyle.Render("┃"))
				bot.WriteString(handleActiveStyle.Render("┃"))
			} else {
				top.WriteString(handleInactiveStyle.Render("│"))
				bot.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}
		if i == endIdx {
			if !m.adjustingStart {
				top.WriteString(handleActiveStyle.Render("┃"))
				bot.WriteString(handleActiveStyle.Render("┃"))
			} else {
				top.WriteString(handleInactiveStyle.Render("│"))
				bot.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}

		inRange := i > startIdx && i < endIdx

		if silence {
			// Top row: blank space in silence style.
			if inRange {
				top.WriteString(silenceInStyle.Render(" "))
			} else {
				top.WriteString(silenceOutStyle.Render(" "))
			}
			// Bottom row: bracket markers.
			var bracket string
			switch {
			case silenceStart[i] && silenceEnd[i]:
				bracket = "⌊"
			case silenceStart[i]:
				bracket = "⌊"
			case silenceEnd[i]:
				bracket = "⌋"
			default:
				bracket = "╌"
			}
			if inRange {
				bot.WriteString(silenceInStyle.Render(bracket))
			} else {
				bot.WriteString(silenceOutStyle.Render(bracket))
			}
			continue
		}

		// Simple track chars in both rows.
		if inRange {
			top.WriteString(selectedTrack.Render("━"))
			bot.WriteString(selectedTrack.Render("━"))
		} else {
			top.WriteString(unselectedTrack.Render("─"))
			bot.WriteString(unselectedTrack.Render("─"))
		}
	}

	return top.String(), bot.String()
}

// renderSliderWithSegments renders the slider showing multiple selected segments.
// Returns two rows: top = track chars, bottom = silence brackets.
func (m Model) renderSliderWithSegments(width int) (topRow, bottomRow string) {
	segments := m.selectedSegments()
	if len(segments) == 0 {
		return m.renderIntegratedSlider(width)
	}

	cols := make([]byte, width)
	for i := range cols {
		cols[i] = 'u'
	}

	for _, seg := range segments {
		si := int(seg.Start / m.duration * float64(width))
		ei := int(seg.End() / m.duration * float64(width))
		if si < 0 {
			si = 0
		}
		if ei >= width {
			ei = width - 1
		}
		for i := si; i <= ei && i < width; i++ {
			cols[i] = 's'
		}
	}

	cursorCol := -1
	if m.isSelectMode() && m.cursor >= 0 && m.cursor < len(m.words) {
		cursorCol = int(m.words[m.cursor].Start / m.duration * float64(width))
		if cursorCol >= width {
			cursorCol = width - 1
		}
	}

	// Pre-pass: silence run boundaries.
	silenceStart := make(map[int]bool, width)
	silenceEnd := make(map[int]bool, width)
	prevSilence := false
	for i := range width {
		s := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		if s && !prevSilence {
			silenceStart[i] = true
		}
		if !s && prevSilence {
			silenceEnd[i-1] = true
		}
		prevSilence = s
	}
	if prevSilence {
		silenceEnd[width-1] = true
	}

	var top, bot strings.Builder
	for i := range width {
		if i == cursorCol {
			top.WriteString(handleActiveStyle.Render("┃"))
			bot.WriteString(handleActiveStyle.Render("┃"))
			continue
		}
		silence := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		selected := cols[i] == 's'

		if silence {
			// Top: blank.
			if selected {
				top.WriteString(silenceInStyle.Render(" "))
			} else {
				top.WriteString(silenceOutStyle.Render(" "))
			}
			// Bottom: bracket markers.
			var bracket string
			switch {
			case silenceStart[i]:
				bracket = "⌊"
			case silenceEnd[i]:
				bracket = "⌋"
			default:
				bracket = "╌"
			}
			if selected {
				bot.WriteString(silenceInStyle.Render(bracket))
			} else {
				bot.WriteString(silenceOutStyle.Render(bracket))
			}
		} else {
			if selected {
				top.WriteString(selectedTrack.Render("━"))
				bot.WriteString(selectedTrack.Render("━"))
			} else {
				top.WriteString(unselectedTrack.Render("─"))
				bot.WriteString(unselectedTrack.Render("─"))
			}
		}
	}
	return top.String(), bot.String()
}
