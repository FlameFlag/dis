package slider

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		m.search.input.Blur()
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
			if len(m.search.results) > 0 {
				m.sel.cursor = m.search.results[m.search.index]
			}
			return m, nil
		}
		m.mode = modeNormal
		m.snapToCueSearchResult()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.search.input.Blur()
		m.search.input.Reset()
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
		} else {
			m.mode = modeNormal
		}
		m.search.results = nil
		return m, nil

	default:
		prevVal := m.search.input.Value()
		var cmd tea.Cmd
		m.search.input, cmd = m.search.input.Update(msg)
		if m.search.input.Value() != prevVal {
			m.updateSearchResults()
		}
		return m, cmd
	}
}
