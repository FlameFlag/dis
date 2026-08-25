package slider

import (
	"slices"
	"strings"

	"github.com/4evy/dis/internal/timecode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) renderLeftPaneWithHeight(width int, targetHeight int) string {
	var lines []string
	w := max(width-singlePaneBorderCells, 1)

	// Source and output metadata stay secondary to the editable range.
	source := Faint.Render("Source") + " " +
		Value.Render(timecode.FormatShort(m.duration))
	format := m.renderFormatBadge()
	pad := max(width-1-lipgloss.Width(source)-lipgloss.Width(format), 1)
	lines = append(lines, " "+source+strings.Repeat(" ", pad)+format)

	// Blank line
	lines = append(lines, "")

	// Time ruler with handle markers (#4)
	labels, ticks := m.renderTimeRuler(w)
	lines = append(lines, " "+labels)
	lines = append(lines, " "+ticks)

	// Slider track with inline timestamps (#8) and all enhancements
	if m.isSelectMode() && m.hasWordSelection() {
		lines = append(lines, " "+m.renderSliderWithSegments(w))
	} else {
		// Start timestamp above track (#8)
		lines = append(lines, " "+m.renderStartLabel(w))

		// Slider track with gradient edges (#2) and sponsor colors (#7)
		lines = append(lines, " "+m.renderIntegratedSlider(w))

		// End timestamp below track (#8)
		lines = append(lines, " "+m.renderEndLabel(w))
	}

	// Keep current state and feedback next to the control they describe.
	hasStatus := false
	if m.warning != "" {
		lines = append(lines,
			" "+ansi.Truncate(Warn.Render(m.warning), w, "…"))
		hasStatus = true
	}
	if loading := m.renderLoadingStatus(); loading != "" {
		lines = append(lines, " "+ansi.Truncate(loading, w, "…"))
		hasStatus = true
	}
	if !hasStatus {
		lines = append(lines, "")
	}

	// Info row / inline input / select info
	switch {
	case m.isSelectMode():
		lines = append(lines, m.renderSelectInfo())
	case m.mode == modeInput:
		lines = append(lines, m.renderInlineInput())
	default:
		lines = append(lines, m.renderInfoRow())
	}

	// Secondary timeline context follows the editable state so small terminals
	// retain the range and any feedback before decorative detail.
	if len(m.sponsorSegments) > 0 {
		lines = append(lines, " "+m.renderSponsorSegments(w))
		if legend := m.renderSponsorLegend(w); legend != "" {
			lines = append(lines, " "+legend)
		}
	}
	if len(m.chapters) > 0 {
		if lbl := m.renderChapterLabels(w); lbl != "" {
			lines = append(lines, " "+lbl)
		}
	}

	// Splits panel
	if len(m.splits) > 0 && !m.isSelectMode() {
		lines = append(lines, "")
		lines = append(lines, m.renderSplitsPanelLines(w, MaxVisibleSplits)...)
	}

	// Keep the preview at the bottom and scale it to the remaining space.
	var bottomLines []string
	mainHeight := lipgloss.Height(strings.Join(lines, "\n"))
	availableThumbnailHeight := -1
	if targetHeight > 0 {
		availableThumbnailHeight = max(targetHeight-mainHeight-1, 0)
	}
	if thumb := m.renderThumbnail(w, availableThumbnailHeight); thumb != "" {
		bottomLines = append(bottomLines, "")
		for tl := range strings.SplitSeq(thumb, "\n") {
			bottomLines = append(bottomLines, " "+tl)
		}
	}

	// Insert padding between main content and bottom elements to fill height
	if targetHeight > 0 {
		usedLines := lipgloss.Height(strings.Join(lines, "\n")) + len(bottomLines)
		for usedLines < targetHeight {
			lines = append(lines, "")
			usedLines++
		}
	}

	lines = append(lines, bottomLines...)
	return strings.Join(lines, "\n")
}

func (m Model) renderTimeRuler(width int) (labels string, ticks string) {
	if m.duration <= 0 {
		return strings.Repeat(" ", width), Dim.Render(strings.Repeat("┈", width))
	}

	pixelsPerSecond := float64(width) / m.duration

	interval := rulerIntervals[len(rulerIntervals)-1]
	for _, iv := range rulerIntervals {
		if iv*pixelsPerSecond >= rulerMinimumTickSpacing {
			interval = iv
			break
		}
	}

	occupied := make([]bool, width)
	var labelLayers []*lipgloss.Layer

	for t := 0.0; t <= m.duration; t += interval {
		pos := int(t / m.duration * float64(width-1))
		if pos >= width {
			pos = width - 1
		}
		lbl := timecode.FormatShort(t)
		lblLen := lipgloss.Width(lbl)
		start := max(pos-lblLen/labelCenterDivisor, 0)
		if start+lblLen > width {
			start = width - lblLen
		}
		if start < 0 {
			continue
		}
		if slices.Contains(occupied[start:start+lblLen], true) {
			continue
		}
		for i := start; i < start+lblLen; i++ {
			occupied[i] = true
		}
		labelLayers = append(labelLayers, lipgloss.NewLayer(Faint.Render(lbl)).X(start))
	}

	labelsCanvas := lipgloss.NewCanvas(width, 1)
	labelsCanvas.Compose(lipgloss.NewCompositor(labelLayers...))
	labels = labelsCanvas.Render()

	// Build tick row with handle position markers (#4: playhead indicator)
	startIdx := max(int(m.startPos/m.duration*float64(width)), 0)
	endIdx := min(int(m.endPos/m.duration*float64(width)), width-1)

	var tickBuf strings.Builder
	for i := range width {
		switch i {
		case startIdx:
			if m.adjustingStart {
				tickBuf.WriteString(AccentBold.Render("▼"))
			} else {
				tickBuf.WriteString(Faint.Render("▼"))
			}
		case endIdx:
			if !m.adjustingStart {
				tickBuf.WriteString(AccentBold.Render("▼"))
			} else {
				tickBuf.WriteString(Faint.Render("▼"))
			}
		default:
			tickBuf.WriteString(Dim.Render("┈"))
		}
	}
	ticks = tickBuf.String()
	return
}
