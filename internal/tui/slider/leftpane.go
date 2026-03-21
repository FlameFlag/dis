package slider

import (
	"dis/internal/util"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderLeftPane(width int) string {
	var lines []string
	w := max(width-2, MinSliderWidth) // inner padding

	// Header: "✂ Trim" ... right-aligned M:SS
	header := boldStyle.Render("✂ Trim")
	durStr := faintStyle.Render(util.FormatDurationShort(m.duration))
	pad := max(width-1-lipgloss.Width(header)-lipgloss.Width(durStr), 1)
	lines = append(lines, " "+header+strings.Repeat(" ", pad)+durStr)

	// Blank line
	lines = append(lines, "")

	// Time ruler
	labels, ticks := m.renderTimeRuler(w)
	lines = append(lines, " "+labels)
	lines = append(lines, " "+ticks)

	// Slider track - two rows: track top, silence brackets bottom
	if m.isSelectMode() && m.hasWordSelection() {
		top, bot := m.renderSliderWithSegments(w)
		lines = append(lines, " "+top)
		lines = append(lines, " "+bot)
	} else {
		top, bot := m.renderIntegratedSlider(w)
		lines = append(lines, " "+top)
		lines = append(lines, " "+bot)
	}

	// SponsorBlock segments row (no legend)
	if len(m.sponsorSegments) > 0 {
		lines = append(lines, " "+m.renderSponsorSegments(w))
	}

	// Chapter labels (no connector row)
	if len(m.chapters) > 0 {
		if lbl := m.renderChapterLabels(w); lbl != "" {
			lines = append(lines, " "+lbl)
		}
	}

	// Blank line
	lines = append(lines, "")

	// Info row / inline input / select info
	if m.isSelectMode() {
		lines = append(lines, m.renderSelectInfo())
	} else if m.mode == modeInput {
		lines = append(lines, m.renderInlineInput())
	} else {
		lines = append(lines, m.renderInfoRow())
	}

	// Loading status row
	if loading := m.renderLoadingStatus(); loading != "" {
		lines = append(lines, " "+loading)
	}

	// Splits panel
	if len(m.splits) > 0 && !m.isSelectMode() {
		lines = append(lines, "")
		lines = append(lines, m.renderSplitsPanelLines(w)...)
	}

	// Thumbnail preview
	if thumb := m.renderThumbnail(w); thumb != "" {
		lines = append(lines, "")
		for tl := range strings.SplitSeq(thumb, "\n") {
			lines = append(lines, " "+tl)
		}
	}

	// Warning (if any)
	if m.warning != "" {
		lines = append(lines, "")
		lines = append(lines, " "+warnStyle.Render(m.warning))
	}

	// Format badge at bottom-right
	formatBadge := m.renderFormatBadge()
	if formatBadge != "" {
		// Pad the last line or add a new one with right-aligned badge
		lines = append(lines, strings.Repeat(" ", max(width-lipgloss.Width(formatBadge)-1, 0))+formatBadge)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderTimeRuler(width int) (labels string, ticks string) {
	if m.duration <= 0 {
		return strings.Repeat(" ", width), dimStyle.Render(strings.Repeat("┈", width))
	}

	pixelsPerSecond := float64(width) / m.duration

	intervals := []float64{10, 15, 30, 60, 120, 300, 600}
	interval := intervals[len(intervals)-1]
	for _, iv := range intervals {
		if iv*pixelsPerSecond >= 10 {
			interval = iv
			break
		}
	}

	labelBuf := make([]byte, width)
	for i := range labelBuf {
		labelBuf[i] = ' '
	}

	for t := 0.0; t <= m.duration; t += interval {
		pos := int(t / m.duration * float64(width-1))
		if pos >= width {
			pos = width - 1
		}
		lbl := util.FormatDurationShort(t)
		lblLen := len(lbl)
		start := max(pos-lblLen/2, 0)
		if start+lblLen > width {
			start = width - lblLen
		}
		if start < 0 {
			continue
		}
		if bufHasOverlap(labelBuf, start, lblLen) {
			continue
		}
		bufPlace(labelBuf, start, lbl)
	}

	labels = faintStyle.Render(string(labelBuf))
	ticks = dimStyle.Render(strings.Repeat("┈", width))
	return
}
