package slider

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func (m Model) renderRightPaneWithHeight(width int, height int) string {
	if m.isSelectMode() && len(m.words) > 0 {
		return m.renderWordSelectPanel(width, height)
	}
	if m.transcript != nil {
		return m.renderTranscriptPanel(width, height)
	}
	return ""
}

func (m Model) renderSearchInput() string {
	matchText := "Type a title or phrase"
	matchStyle := Faint
	if m.search.input.Value() != "" && len(m.search.results) == 0 {
		matchText = "No matches"
		matchStyle = Warn
	} else if len(m.search.results) > 0 {
		matchText = fmt.Sprintf(
			"Match %d of %d",
			m.search.index+1,
			len(m.search.results),
		)
	}
	label := AccentBold.Render("Search")
	availableWidth := max(m.width-singlePaneBorderCells, 1)
	if availableWidth < compactSearchWidth && m.search.input.Value() == "" {
		matchText = "Type to search"
	}
	matchInfo := matchStyle.Render(matchText)
	prefix := " " + label + "  "
	suffix := "  " + matchInfo
	fixedWidth := lipgloss.Width(prefix + suffix)
	inputWidth := max(
		availableWidth-fixedWidth-1, // textinput reserves a cell for its cursor
		minimumSearchInputWidth,
	)
	input := m.search.input
	input.SetWidth(inputWidth)
	input.SetCursor(input.Position())
	return prefix + input.View() + suffix
}
