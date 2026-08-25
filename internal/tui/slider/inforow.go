package slider

import (
	"fmt"
	"strings"

	"github.com/4evy/dis/internal/timecode"

	"charm.land/lipgloss/v2"
)

func (m Model) renderInfoRow() string {
	startStr := timecode.FormatMillis(m.startPos)
	endStr := timecode.FormatMillis(m.endPos)
	length := m.endPos - m.startPos
	lengthStr := timecode.FormatMillis(length)

	startMarker := "○"
	endMarker := "○"
	startLabel := Faint.Render("Start")
	endLabel := Faint.Render("End")
	var styledStart, styledEnd string
	if m.adjustingStart {
		startMarker = AccentBold.Render("●")
		startLabel = AccentBold.Render("Start")
		styledStart = AccentBold.Render(startStr)
		styledEnd = Value.Render(endStr)
	} else {
		endMarker = AccentBold.Render("●")
		endLabel = AccentBold.Render("End")
		styledStart = Value.Render(startStr)
		styledEnd = AccentBold.Render(endStr)
	}

	if m.leftPaneWidth() < compactInfoWidth {
		compact := fmt.Sprintf(" %s %s  %s %s  %s %s",
			startMarker, styledStart,
			endMarker, styledEnd,
			Faint.Render("Keep"), Faint.Render(lengthStr))
		if lipgloss.Width(compact) > m.leftPaneWidth() {
			return fmt.Sprintf(" %s %s  %s %s\n   %s %s",
				startMarker, styledStart,
				endMarker, styledEnd,
				Faint.Render("Keep"), Faint.Render(lengthStr))
		}
		return compact
	}
	return fmt.Sprintf(" %s %s %s  %s %s %s  %s %s",
		startMarker, startLabel, styledStart,
		endMarker, endLabel, styledEnd,
		Faint.Render("Keep"), Faint.Render(lengthStr))
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
		return Faint.Render(" No words selected")
	}

	segText := "segment"
	if len(segs) != 1 {
		segText = "segments"
	}

	return fmt.Sprintf(" %s %s · %s · %s",
		Value.Render(fmt.Sprintf("%d %s", len(segs), segText)),
		Faint.Render(timecode.FormatShort(totalDur)),
		Faint.Render("total"),
		Faint.Render(fmt.Sprintf("%d/%d", selCount, totalWords)))
}

func (m Model) renderInlineInput() string {
	inputView := m.timeInput.View()

	startStr := timecode.FormatMillis(m.startPos)
	endStr := timecode.FormatMillis(m.endPos)

	if m.leftPaneWidth() < compactInfoWidth {
		if m.adjustingStart {
			return fmt.Sprintf(" %s %s  %s %s\n   %s %s",
				AccentBold.Render("●"), inputView,
				Faint.Render("○"), Value.Render(endStr),
				Faint.Render("Keep"), Faint.Render("--:--.---"))
		}
		return fmt.Sprintf(" %s %s  %s %s\n   %s %s",
			Faint.Render("○"), Value.Render(startStr),
			AccentBold.Render("●"), inputView,
			Faint.Render("Keep"), Faint.Render("--:--.---"))
	}
	if m.adjustingStart {
		return fmt.Sprintf(" %s %s  %s %s  %s %s",
			Faint.Render("Start"), inputView,
			Faint.Render("End"), Value.Render(endStr),
			Faint.Render("Keep"), Faint.Render("--:--.---"))
	}
	return fmt.Sprintf(" %s %s  %s %s  %s %s",
		Faint.Render("Start"), Value.Render(startStr),
		Faint.Render("End"), inputView,
		Faint.Render("Keep"), Faint.Render("--:--.---"))
}

func (m Model) renderFormatBadge() string {
	if m.gifMode {
		badge := AccentBold.Render("GIF")
		if m.speedMultiplier > playbackSpeeds[0] {
			badge += " " + AccentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
		}
		return badge
	}
	badge := Faint.Render("MP4")
	if m.speedMultiplier > playbackSpeeds[0] {
		badge += " " + AccentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
	}
	return badge
}

func (m Model) renderLoadingStatus() string {
	if !m.isLoading() {
		return ""
	}
	spinner := m.loadingSpinner.View()
	var items []string
	if m.transcriptCh != nil {
		items = append(items, "transcript")
	}
	if m.storyboardCh != nil {
		items = append(items, "storyboard")
	}
	if m.sponsorSegsCh != nil {
		items = append(items, "SponsorBlock")
	}
	return Faint.Render(spinner + " " + strings.Join(items, " · "))
}
