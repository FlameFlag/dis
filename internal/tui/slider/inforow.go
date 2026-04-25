package slider

import (
	"dis/internal/tui/slider/style"
	"dis/internal/util"
	"fmt"
	"strings"
)

func (m Model) renderInfoRow() string {
	startStr := util.FormatDurationMillis(m.startPos)
	endStr := util.FormatDurationMillis(m.endPos)
	length := m.endPos - m.startPos
	lengthStr := util.FormatDurationMillis(length)

	var styledStart, styledEnd string
	if m.adjustingStart {
		styledStart = style.AccentBold.Render(startStr)
		styledEnd = style.Value.Render(endStr)
	} else {
		styledStart = style.Value.Render(startStr)
		styledEnd = style.AccentBold.Render(endStr)
	}

	info := fmt.Sprintf(" %s %s  %s %s  %s %s",
		style.Faint.Render("start"), styledStart,
		style.Faint.Render("end"), styledEnd,
		style.Faint.Render("length"), style.Faint.Render(lengthStr))
	return info
}

func (m Model) renderSelectInfo() string {
	segs := m.selectedSegments()
	selCount := m.selectedWordCount()
	totalWords := len(m.words)

	var totalDur float64
	for _, seg := range segs {
		totalDur += seg.Duration
	}

	if len(segs) == 0 {
		return style.Faint.Render(" No words selected")
	}

	segText := "segment"
	if len(segs) != 1 {
		segText = "segments"
	}

	return fmt.Sprintf(" %s %s · %s · %s",
		style.Value.Render(fmt.Sprintf("%d %s", len(segs), segText)),
		style.Faint.Render(util.FormatDurationShort(totalDur)),
		style.Faint.Render("total"),
		style.Faint.Render(fmt.Sprintf("%d/%d", selCount, totalWords)))
}

func (m Model) renderInlineInput() string {
	inputView := m.timeInput.View()

	startStr := util.FormatDurationMillis(m.startPos)
	endStr := util.FormatDurationMillis(m.endPos)

	if m.adjustingStart {
		return fmt.Sprintf(" %s %s  %s %s  %s %s",
			style.Faint.Render("start"), inputView,
			style.Faint.Render("end"), style.Value.Render(endStr),
			style.Faint.Render("length"), style.Faint.Render("--:--.---"))
	}
	return fmt.Sprintf(" %s %s  %s %s  %s %s",
		style.Faint.Render("start"), style.Value.Render(startStr),
		style.Faint.Render("end"), inputView,
		style.Faint.Render("length"), style.Faint.Render("--:--.---"))
}

func (m Model) renderFormatBadge() string {
	if m.gifMode {
		badge := style.AccentBold.Render("GIF")
		if m.speedMultiplier > 1.0 {
			badge += " " + style.AccentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
		}
		return badge
	}
	badge := style.Faint.Render("MP4")
	if m.speedMultiplier > 1.0 {
		badge += " " + style.AccentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
	}
	return badge
}

func (m Model) renderLoadingStatus() string {
	if !m.isLoading() {
		return ""
	}
	spinner := m.loadingSpinner.View()
	var items []string
	if m.storyboardCh != nil {
		items = append(items, "storyboard")
	}
	return style.Faint.Render(spinner + " " + strings.Join(items, " · "))
}
