package slider

import (
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderWordSelectPanel(width int, targetHeight int) string {
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
		if m.sel.cursor >= g.startIdx && m.sel.cursor <= g.endIdx {
			cursorGroup = i
			break
		}
	}

	visibleCues := WordSelectVisibleCues
	if targetHeight > 0 {
		visibleCues = max(targetHeight-2, WordSelectVisibleCues)
	}
	pinOffset := visibleCues / 3

	startGroup := max(cursorGroup-pinOffset, 0)
	endGroup := startGroup + visibleCues
	if endGroup > len(groups) {
		endGroup = len(groups)
		startGroup = max(endGroup-visibleCues, 0)
	}

	selectedStyle := lipgloss.NewStyle().Foreground(tui.ColorPeach)
	cursorSelectedStyle := lipgloss.NewStyle().Reverse(true).Bold(true).Foreground(tui.ColorPeach)

	searchSet := make(map[int]bool, len(m.search.results))
	for _, idx := range m.search.results {
		searchSet[idx] = true
	}

	var lines []string

	// Scroll indicator above
	if startGroup > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▲ %d more", startGroup)))
	}

	cursorTime := m.words[m.sel.cursor].Start

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

			isCursor := i == m.sel.cursor
			isSelected := m.sel.selected[i]
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
