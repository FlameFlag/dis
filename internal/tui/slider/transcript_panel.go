package slider

import (
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderTranscriptPanel(width int) string {
	if len(m.transcript) == 0 {
		return ""
	}

	var lines []string

	pos := m.activePos()
	activeCue := m.transcript.NearestCue(pos)

	var startCue int
	if m.viewportLocked {
		startCue = max(activeCue-TranscriptPinOffset, 0)
		maxOffset := max(len(m.transcript)-TranscriptVisibleCues, 0)
		if startCue > maxOffset {
			startCue = maxOffset
		}
	} else {
		startCue = m.transcriptOffset
	}

	endCue := min(startCue+TranscriptVisibleCues, len(m.transcript))

	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	// Scroll indicator above
	if startCue > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▲ %d more", startCue)))
	}

	activeBg := lipgloss.NewStyle().Background(tui.ColorSurface1)
	textWidth := max(width-10, 10) // timestamp + padding

	for i := startCue; i < endCue; i++ {
		cue := m.transcript[i]
		timeStr := util.FormatDurationShort(cue.Start)
		isActive := i == activeCue

		text := cue.Text
		if lipgloss.Width(text) > textWidth && textWidth > 3 {
			text = truncateVisual(text, textWidth-1) + "…"
		}

		sponsorCat := m.sponsorCategoryAt(cue.Start)
		styledText := text
		if isActive {
			styledText = activeBg.Render(accentStyle.Render(text))
		} else if searchSet[i] {
			styledText = warmStyle.Render(text)
		} else if sponsorCat != "" {
			if sc, ok := sponsorCategories[sponsorCat]; ok {
				styledText = sc.Color.Render(text)
			}
		} else if cue.End <= pos {
			styledText = faintStyle.Render(text)
		}

		// Active indicator on right side
		indicator := "  "
		if isActive {
			indicator = accentStyle.Render(" ◀")
		}

		line := fmt.Sprintf("  %s  %s%s", faintStyle.Render(timeStr), styledText, indicator)
		lines = append(lines, line)
	}

	// Scroll indicator below
	remaining := len(m.transcript) - endCue
	if remaining > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▼ %d more", remaining)))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderWordSelectPanel(width int) string {
	if len(m.words) == 0 {
		return ""
	}

	markerCol := 3
	timestampCol := 6
	textWidth := max(width-markerCol-timestampCol, 20)

	type cueGroup struct {
		cueIndex int
		startIdx int
		endIdx   int
	}
	var groups []cueGroup
	if len(m.words) > 0 {
		cur := cueGroup{cueIndex: m.words[0].CueIndex, startIdx: 0}
		for i := 1; i < len(m.words); i++ {
			if m.words[i].CueIndex != cur.cueIndex {
				cur.endIdx = i - 1
				groups = append(groups, cur)
				cur = cueGroup{cueIndex: m.words[i].CueIndex, startIdx: i}
			}
		}
		cur.endIdx = len(m.words) - 1
		groups = append(groups, cur)
	}

	cursorGroup := 0
	for i, g := range groups {
		if m.cursor >= g.startIdx && m.cursor <= g.endIdx {
			cursorGroup = i
			break
		}
	}

	startGroup := max(cursorGroup-WordSelectPinOffset, 0)
	endGroup := startGroup + WordSelectVisibleCues
	if endGroup > len(groups) {
		endGroup = len(groups)
		startGroup = max(endGroup-WordSelectVisibleCues, 0)
	}

	selectedStyle := lipgloss.NewStyle().Foreground(tui.ColorPeach)
	cursorSelectedStyle := lipgloss.NewStyle().Reverse(true).Bold(true).Foreground(tui.ColorPeach)

	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	var lines []string

	// Scroll indicator above
	if startGroup > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▲ %d more", startGroup)))
	}

	cursorTime := m.words[m.cursor].Start

	for gi := startGroup; gi < endGroup; gi++ {
		g := groups[gi]
		timestamp := util.FormatDurationShort(m.words[g.startIdx].Start)
		tsPrefix := dimStyle.Render(fmt.Sprintf("%-5s ", timestamp))

		marker := "   "
		if gi == cursorGroup {
			marker = accentStyle.Render(" › ")
		}

		groupPassed := m.words[g.endIdx].End <= cursorTime

		var line strings.Builder
		lineLen := 0
		firstLine := true

		for i := g.startIdx; i <= g.endIdx; i++ {
			wordText := m.words[i].Text
			wordText = strings.TrimPrefix(wordText, ">>")
			wordText = strings.TrimSpace(wordText)
			if wordText == "" {
				continue
			}
			displayLen := len(wordText)

			if lineLen > 0 && lineLen+1+displayLen > textWidth {
				if firstLine {
					lines = append(lines, marker+tsPrefix+line.String())
					firstLine = false
				} else {
					lines = append(lines, strings.Repeat(" ", markerCol+timestampCol)+line.String())
				}
				line.Reset()
				lineLen = 0
			}

			if lineLen > 0 {
				line.WriteByte(' ')
				lineLen++
			}

			isCursor := i == m.cursor
			isSelected := m.selected[i]
			isSearchMatch := searchSet[i]

			switch {
			case isCursor && isSelected:
				line.WriteString(cursorSelectedStyle.Render(wordText))
			case isCursor:
				line.WriteString(reverseStyle.Render(wordText))
			case isSelected:
				line.WriteString(selectedStyle.Render(wordText))
			case isSearchMatch:
				line.WriteString(warmStyle.Render(wordText))
			case groupPassed:
				line.WriteString(faintStyle.Render(wordText))
			default:
				if cat := m.sponsorCategoryAt(m.words[i].Start); cat != "" {
					if sc, ok := sponsorCategories[cat]; ok {
						line.WriteString(sc.Color.Render(wordText))
					} else {
						line.WriteString(wordText)
					}
				} else {
					line.WriteString(wordText)
				}
			}
			lineLen += displayLen
		}

		if lineLen > 0 {
			if firstLine {
				lines = append(lines, marker+tsPrefix+line.String())
			} else {
				lines = append(lines, strings.Repeat(" ", markerCol+timestampCol)+line.String())
			}
		}
	}

	// Scroll indicator below
	if endGroup < len(groups) {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▼ %d more", len(groups)-endGroup)))
	}

	return strings.Join(lines, "\n")
}
