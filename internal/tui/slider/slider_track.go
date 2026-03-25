package slider

import "strings"

// renderIntegratedSlider renders the slider track.
func (m Model) renderIntegratedSlider(width int) string {
	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	var out strings.Builder

	for i := range width {
		if i == startIdx {
			if m.adjustingStart {
				out.WriteString(handleActiveStyle.Render("┃"))
			} else {
				out.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}
		if i == endIdx {
			if !m.adjustingStart {
				out.WriteString(handleActiveStyle.Render("┃"))
			} else {
				out.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}

		if i > startIdx && i < endIdx {
			out.WriteString(selectedTrack.Render("━"))
		} else {
			out.WriteString(unselectedTrack.Render("─"))
		}
	}

	return out.String()
}

// renderSliderWithSegments renders the slider showing multiple selected segments.
func (m Model) renderSliderWithSegments(width int) string {
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

	var out strings.Builder
	for i := range width {
		if i == cursorCol {
			out.WriteString(handleActiveStyle.Render("┃"))
			continue
		}
		if cols[i] == 's' {
			out.WriteString(selectedTrack.Render("━"))
		} else {
			out.WriteString(unselectedTrack.Render("─"))
		}
	}
	return out.String()
}
