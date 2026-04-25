package slider

import "fmt"

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
	if m.search.input.Value() != "" {
		matchInfo = fmt.Sprintf("  (%d matches)", len(m.search.results))
	}
	return " " + accentStyle.Render("/") + " " + m.search.input.View() + faintStyle.Render(matchInfo)
}
