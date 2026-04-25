package slider

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		m.searchInput.Blur()
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
			if len(m.searchResults) > 0 {
				m.cursor = m.searchResults[m.searchIndex]
			}
			return m, nil
		}
		m.mode = modeNormal
		m.snapToCueSearchResult()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.searchInput.Blur()
		m.searchInput.Reset()
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
		} else {
			m.mode = modeNormal
		}
		m.searchResults = nil
		return m, nil

	default:
		prevVal := m.searchInput.Value()
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		if m.searchInput.Value() != prevVal {
			m.updateSearchResults()
		}
		return m, cmd
	}
}
