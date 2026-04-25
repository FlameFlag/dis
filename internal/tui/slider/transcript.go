package slider

import (
	"dis/internal/tui"
	"dis/internal/tui/slider/style"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) renderTranscriptPanel(width int, targetHeight int) string {
	if len(m.transcript) == 0 {
		return ""
	}

	var lines []string

	pos := m.activePos()
	activeCue := m.transcript.NearestCue(pos)

	// Use targetHeight to determine visible cues, reserving lines for scroll indicators
	visibleCues := TranscriptVisibleCues
	if targetHeight > 0 {
		// Reserve up to 2 lines for scroll indicators (top + bottom)
		visibleCues = max(targetHeight-2, TranscriptVisibleCues)
	}
	pinOffset := visibleCues / 3

	var startCue int
	if m.viewportLocked {
		startCue = max(activeCue-pinOffset, 0)
		maxOffset := max(len(m.transcript)-visibleCues, 0)
		if startCue > maxOffset {
			startCue = maxOffset
		}
	} else {
		startCue = m.transcriptOffset
	}

	endCue := min(startCue+visibleCues, len(m.transcript))

	searchSet := make(map[int]bool, len(m.search.results))
	for _, idx := range m.search.results {
		searchSet[idx] = true
	}

	// Scroll indicator above
	if startCue > 0 {
		lines = append(lines, style.Faint.Render(fmt.Sprintf("  ▲ %d more", startCue)))
	}

	activeBg := lipgloss.NewStyle().Background(tui.ColorSurface1)
	textWidth := max(width-10, 10) // timestamp + padding

	// Count how many cues are below the active one for fade calculation
	cuesBelowActive := max(endCue-activeCue-1, 1)

	for i := startCue; i < endCue; i++ {
		cue := m.transcript[i]
		timeStr := util.FormatDurationShort(cue.Start)
		isActive := i == activeCue

		text := cue.Text
		if lipgloss.Width(text) > textWidth && textWidth > 3 {
			text = ansi.Truncate(text, textWidth, "…")
		}

		sponsorCat := m.sponsorCategoryAt(cue.Start)
		styledText := text
		timeStyle := style.Faint
		if isActive {
			styledText = activeBg.Render(style.Accent.Render(text))
		} else if searchSet[i] {
			styledText = style.Warm.Render(text)
		} else if sponsorCat != "" {
			if sc, ok := style.SponsorCategories[sponsorCat]; ok {
				styledText = sc.Color.Render(text)
			}
		} else if cue.End <= pos {
			styledText = style.Faint.Render(text)
		} else if i > activeCue {
			// Apply fade gradient based on distance below active cue
			dist := i - activeCue - 1
			// Map distance to gradient index based on proportion of remaining cues
			gradIdx := dist * len(style.Fade) / cuesBelowActive
			if gradIdx >= len(style.Fade) {
				gradIdx = len(style.Fade) - 1
			}
			fade := style.Fade[gradIdx]
			styledText = fade.Render(text)
			timeStyle = fade
		}

		// Active indicator on right side
		indicator := "  "
		if isActive {
			indicator = style.Accent.Render(" ◀")
		}

		line := fmt.Sprintf("  %s  %s%s", timeStyle.Render(timeStr), styledText, indicator)
		lines = append(lines, line)
	}

	// Scroll indicator below
	remaining := len(m.transcript) - endCue
	if remaining > 0 {
		lines = append(lines, style.Faint.Render(fmt.Sprintf("  ▼ %d more", remaining)))
	}

	return strings.Join(lines, "\n")
}
