package slider

import "fmt"

func (m Model) renderRightPane(width int) string {
	if m.isSelectMode() && len(m.words) > 0 {
		return m.renderWordSelectPanel(width)
	}
	if m.transcript != nil {
		return m.renderTranscriptPanel(width)
	}
	return ""
}

func (m Model) renderSearchInput() string {
	matchInfo := ""
	if m.searchBuffer != "" {
		matchInfo = fmt.Sprintf("  (%d matches)", len(m.searchResults))
	}
	return fmt.Sprintf(" %s %s%s%s",
		accentStyle.Render("/"),
		m.searchBuffer,
		faintStyle.Render("█"),
		faintStyle.Render(matchInfo))
}
