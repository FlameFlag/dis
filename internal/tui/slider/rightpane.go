package slider

import "fmt"

func (m Model) renderRightPane(width int) string {
	if m.isSelectMode() && len(m.words) > 0 {
		return m.renderWordSelectPanel(width, 0)
	}
	if m.transcript != nil {
		return m.renderTranscriptPanel(width, 0)
	}
	return ""
}

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
	matchInfo := ""
	if m.searchInput.Value() != "" {
		matchInfo = fmt.Sprintf("  (%d matches)", len(m.searchResults))
	}
	return " " + accentStyle.Render("/") + " " + m.searchInput.View() + faintStyle.Render(matchInfo)
}
