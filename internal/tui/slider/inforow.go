package slider

import (
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
		styledStart = accentBold.Render(startStr)
		styledEnd = valueStyle.Render(endStr)
	} else {
		styledStart = valueStyle.Render(startStr)
		styledEnd = accentBold.Render(endStr)
	}

	info := fmt.Sprintf(" %s %s  %s %s  %s %s",
		faintStyle.Render("start"), styledStart,
		faintStyle.Render("end"), styledEnd,
		faintStyle.Render("length"), faintStyle.Render(lengthStr))
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
		return faintStyle.Render(" No words selected")
	}

	segText := "segment"
	if len(segs) != 1 {
		segText = "segments"
	}

	return fmt.Sprintf(" %s %s · %s · %s",
		valueStyle.Render(fmt.Sprintf("%d %s", len(segs), segText)),
		faintStyle.Render(util.FormatDurationShort(totalDur)),
		faintStyle.Render("total"),
		faintStyle.Render(fmt.Sprintf("%d/%d", selCount, totalWords)))
}

func (m Model) renderInlineInput() string {
	inputView := m.timeInput.View()

	startStr := util.FormatDurationMillis(m.startPos)
	endStr := util.FormatDurationMillis(m.endPos)

	if m.adjustingStart {
		return fmt.Sprintf(" %s %s  %s %s  %s %s",
			faintStyle.Render("start"), inputView,
			faintStyle.Render("end"), valueStyle.Render(endStr),
			faintStyle.Render("length"), faintStyle.Render("--:--.---"))
	}
	return fmt.Sprintf(" %s %s  %s %s  %s %s",
		faintStyle.Render("start"), valueStyle.Render(startStr),
		faintStyle.Render("end"), inputView,
		faintStyle.Render("length"), faintStyle.Render("--:--.---"))
}

func (m Model) renderFormatBadge() string {
	if m.gifMode {
		badge := accentBold.Render("GIF")
		if m.speedMultiplier > 1.0 {
			badge += " " + accentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
		}
		return badge
	}
	badge := faintStyle.Render("MP4")
	if m.speedMultiplier > 1.0 {
		badge += " " + accentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
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
	return faintStyle.Render(spinner + " " + strings.Join(items, " · "))
}
