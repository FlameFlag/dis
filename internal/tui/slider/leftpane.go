package slider

import (
	"bytes"
	"dis/internal/tui/slider/style"
	"dis/internal/tui/slider/textbuf"
	"dis/internal/util"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderLeftPaneWithHeight(width int, targetHeight int) string {
	var lines []string
	w := max(width-2, MinSliderWidth) // inner padding

	// Header: "✂ Trim" ... right-aligned M:SS
	header := style.Bold.Render("✂ Trim")
	durStr := style.Faint.Render(util.FormatDurationShort(m.duration))
	pad := max(width-1-lipgloss.Width(header)-lipgloss.Width(durStr), 1)
	lines = append(lines, " "+header+strings.Repeat(" ", pad)+durStr)

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

	// SponsorBlock segments row (kept for highlights ★ and as legend)
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
		lines = append(lines, m.renderSplitsPanelLines(w, MaxVisibleSplits)...)
	}

	// Collect bottom-pinned elements (thumbnail, warning, format badge)
	var bottomLines []string

	if thumb := m.renderThumbnail(w); thumb != "" {
		bottomLines = append(bottomLines, "")
		for tl := range strings.SplitSeq(thumb, "\n") {
			bottomLines = append(bottomLines, " "+tl)
		}
	}

	if m.warning != "" {
		bottomLines = append(bottomLines, "")
		bottomLines = append(bottomLines, " "+style.Warn.Render(m.warning))
	}

	formatBadge := m.renderFormatBadge()
	if formatBadge != "" {
		bottomLines = append(bottomLines, strings.Repeat(" ", max(width-lipgloss.Width(formatBadge)-1, 0))+formatBadge)
	}

	// Insert padding between main content and bottom elements to fill height
	if targetHeight > 0 {
		usedLines := len(lines) + len(bottomLines)
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
		return strings.Repeat(" ", width), style.Dim.Render(strings.Repeat("┈", width))
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

	labelBuf := bytes.Repeat([]byte{' '}, width)

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
		if textbuf.HasOverlap(labelBuf, start, lblLen) {
			continue
		}
		textbuf.Place(labelBuf, start, lbl)
	}

	labels = style.Faint.Render(string(labelBuf))

	// Build tick row with handle position markers (#4: playhead indicator)
	startIdx := int(m.anim.startPos / m.duration * float64(width))
	endIdx := int(m.anim.endPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	var tickBuf strings.Builder
	for i := range width {
		switch i {
		case startIdx:
			if m.adjustingStart {
				tickBuf.WriteString(style.AccentBold.Render("▼"))
			} else {
				tickBuf.WriteString(style.Faint.Render("▼"))
			}
		case endIdx:
			if !m.adjustingStart {
				tickBuf.WriteString(style.AccentBold.Render("▼"))
			} else {
				tickBuf.WriteString(style.Faint.Render("▼"))
			}
		default:
			tickBuf.WriteString(style.Dim.Render("┈"))
		}
	}
	ticks = tickBuf.String()
	return
}
