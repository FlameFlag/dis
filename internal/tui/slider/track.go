package slider

import (
	"dis/internal/tui/slider/style"
	"dis/internal/util"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderIntegratedSlider renders the slider track with gradient edges on the
// selected region and sponsor-block colors baked directly into the track.
func (m Model) renderIntegratedSlider(width int) string {
	startIdx := int(m.anim.startPos / m.duration * float64(width))
	endIdx := int(m.anim.endPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	// Build sponsor color map for the track (#7: colored region highlighting)
	sponsorColor := m.sponsorColorMap(width)

	var out strings.Builder

	for i := range width {
		// Handles (#3: chunkier handles)
		if i == startIdx {
			if m.adjustingStart {
				out.WriteString(style.HandleActive.Render("▌"))
			} else {
				out.WriteString(style.HandleInactive.Render("▌"))
			}
			continue
		}
		if i == endIdx {
			if !m.adjustingStart {
				out.WriteString(style.HandleActive.Render("▐"))
			} else {
				out.WriteString(style.HandleInactive.Render("▐"))
			}
			continue
		}

		if i > startIdx && i < endIdx {
			// #2: gradient edges + #7: sponsor colors
			out.WriteString(m.renderSelectedCol(i, startIdx, endIdx, sponsorColor))
		} else {
			out.WriteString(style.UnselectedTrack.Render("─"))
		}
	}

	return out.String()
}

// renderSelectedCol renders a single selected-region column, applying
// a fade-in/fade-out gradient at the edges and sponsor-block color overrides.
func (m Model) renderSelectedCol(col, startIdx, endIdx int, sponsorColor []lipgloss.Color) string {
	// Check for sponsor color override (#7)
	if sponsorColor[col] != "" {
		return lipgloss.NewStyle().Foreground(sponsorColor[col]).Render("━")
	}

	distFromEdge := min(col-startIdx, endIdx-col)
	if distFromEdge <= len(style.Track) {
		return style.Track[max(distFromEdge-1, 0)].Render("━")
	}
	return style.SelectedTrack.Render("━")
}

// renderStartLabel renders the start-handle timestamp ABOVE the track.
func (m Model) renderStartLabel(width int) string {
	startIdx := max(int(m.anim.startPos/m.duration*float64(width)), 0)
	if startIdx >= width {
		startIdx = width - 1
	}

	label := util.FormatDurationMillis(m.startPos)
	pos := startIdx - len(label)/2
	pos = max(pos, 0)
	if pos+len(label) > width {
		pos = width - len(label)
	}

	s := style.Faint
	if m.adjustingStart {
		s = style.AccentBold
	}

	return strings.Repeat(" ", pos) + s.Render(label)
}

// renderEndLabel renders the end-handle timestamp BELOW the track.
func (m Model) renderEndLabel(width int) string {
	endIdx := max(int(m.anim.endPos/m.duration*float64(width)), 0)
	if endIdx >= width {
		endIdx = width - 1
	}

	label := util.FormatDurationMillis(m.endPos)
	pos := endIdx - len(label)/2
	pos = max(pos, 0)
	if pos+len(label) > width {
		pos = width - len(label)
	}

	s := style.Faint
	if !m.adjustingStart {
		s = style.AccentBold
	}

	return strings.Repeat(" ", pos) + s.Render(label)
}

// sponsorColorMap returns per-column sponsor colors for the track.
func (m Model) sponsorColorMap(width int) []lipgloss.Color {
	colors := make([]lipgloss.Color, width)
	if m.duration <= 0 {
		return colors
	}
	for _, seg := range m.sponsorSegments {
		sc, ok := style.SponsorCategories[seg.Category]
		if !ok {
			continue
		}
		si := int(seg.Start / m.duration * float64(width))
		ei := int(seg.End / m.duration * float64(width))
		if si < 0 {
			si = 0
		}
		if ei >= width {
			ei = width - 1
		}
		for i := si; i <= ei; i++ {
			colors[i] = sc.HexColor
		}
	}
	return colors
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
	if m.isSelectMode() && m.sel.cursor >= 0 && m.sel.cursor < len(m.words) {
		cursorCol = int(m.words[m.sel.cursor].Start / m.duration * float64(width))
		if cursorCol >= width {
			cursorCol = width - 1
		}
	}

	var out strings.Builder
	for i := range width {
		if i == cursorCol {
			out.WriteString(style.HandleActive.Render("▌"))
			continue
		}
		if cols[i] == 's' {
			out.WriteString(style.SelectedTrack.Render("━"))
		} else {
			out.WriteString(style.UnselectedTrack.Render("─"))
		}
	}
	return out.String()
}
