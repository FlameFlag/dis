package slider

import (
	"bytes"
	"image/color"
	"strings"

	"github.com/4evy/dis/internal/timecode"

	"charm.land/lipgloss/v2"
)

// renderIntegratedSlider renders the slider track with gradient edges on the
// selected region and sponsor-block colors baked directly into the track.
func (m Model) renderIntegratedSlider(width int) string {
	startIdx := max(int(m.startPos/m.duration*float64(width)), 0)
	endIdx := min(int(m.endPos/m.duration*float64(width)), width-1)

	// Build sponsor color map for the track (#7: colored region highlighting)
	sponsorColor := m.sponsorColorMap(width)

	var out strings.Builder

	for i := range width {
		// Handles (#3: chunkier handles)
		if i == startIdx {
			if m.adjustingStart {
				out.WriteString(HandleActive.Render("▌"))
			} else {
				out.WriteString(HandleInactive.Render("▌"))
			}
			continue
		}
		if i == endIdx {
			if !m.adjustingStart {
				out.WriteString(HandleActive.Render("▐"))
			} else {
				out.WriteString(HandleInactive.Render("▐"))
			}
			continue
		}

		if i > startIdx && i < endIdx {
			// #2: gradient edges + #7: sponsor colors
			out.WriteString(m.renderSelectedCol(i, startIdx, endIdx, sponsorColor))
		} else {
			out.WriteString(UnselectedTrack.Render("─"))
		}
	}

	return out.String()
}

// renderSelectedCol renders a single selected-region column, applying
// a fade-in/fade-out gradient at the edges and sponsor-block color overrides.
func (m Model) renderSelectedCol(col, startIdx, endIdx int, sponsorColor []color.Color) string {
	// Check for sponsor color override (#7)
	if sponsorColor[col] != nil {
		return lipgloss.NewStyle().Foreground(sponsorColor[col]).Render("━")
	}

	distFromEdge := min(col-startIdx, endIdx-col)
	if distFromEdge <= len(Track) {
		return Track[max(distFromEdge-1, 0)].Render("━")
	}
	return SelectedTrack.Render("━")
}

// renderStartLabel renders the start-handle timestamp ABOVE the track.
func (m Model) renderStartLabel(width int) string {
	startIdx := min(max(int(m.startPos/m.duration*float64(width)), 0), width-1)

	label := timecode.FormatMillis(m.startPos)
	labelWidth := lipgloss.Width(label)
	pos := startIdx - labelWidth/labelCenterDivisor
	pos = max(pos, 0)
	if pos+labelWidth > width {
		pos = width - labelWidth
	}

	s := Faint
	if m.adjustingStart {
		s = AccentBold
	}

	return strings.Repeat(" ", pos) + s.Render(label)
}

// renderEndLabel renders the end-handle timestamp BELOW the track.
func (m Model) renderEndLabel(width int) string {
	endIdx := min(max(int(m.endPos/m.duration*float64(width)), 0), width-1)

	label := timecode.FormatMillis(m.endPos)
	labelWidth := lipgloss.Width(label)
	pos := endIdx - labelWidth/labelCenterDivisor
	pos = max(pos, 0)
	if pos+labelWidth > width {
		pos = width - labelWidth
	}

	s := Faint
	if !m.adjustingStart {
		s = AccentBold
	}

	return strings.Repeat(" ", pos) + s.Render(label)
}

// sponsorColorMap returns per-column sponsor colors for the track.
func (m Model) sponsorColorMap(width int) []color.Color {
	colors := make([]color.Color, width)
	if m.duration <= 0 {
		return colors
	}
	for _, seg := range m.sponsorSegments {
		sc, ok := SponsorCategories[seg.Category]
		if !ok {
			continue
		}
		si := max(int(seg.Start/m.duration*float64(width)), 0)
		ei := min(int(seg.End/m.duration*float64(width)), width-1)
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

	cols := bytes.Repeat([]byte{'u'}, width)

	for _, seg := range segments {
		si := max(int(seg.Start/m.duration*float64(width)), 0)
		ei := min(int(seg.End()/m.duration*float64(width)), width-1)
		for i := si; i <= ei && i < width; i++ {
			cols[i] = 's'
		}
	}

	cursorCol := -1
	if m.isSelectMode() && m.sel.cursor >= 0 && m.sel.cursor < len(m.words) {
		cursorCol = min(int(m.words[m.sel.cursor].Start/m.duration*float64(width)), width-1)
	}

	var out strings.Builder
	for i := range width {
		if i == cursorCol {
			out.WriteString(HandleActive.Render("▌"))
			continue
		}
		if cols[i] == 's' {
			out.WriteString(SelectedTrack.Render("━"))
		} else {
			out.WriteString(UnselectedTrack.Render("─"))
		}
	}
	return out.String()
}
