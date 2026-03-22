package slider

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		if m.timeInput.Value() != "" {
			m.processTimeInput()
		}
		m.timeInput.Blur()
		m.mode = modeNormal
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.timeInput.Blur()
		m.timeInput.Reset()
		m.mode = modeNormal
		return m, nil

	default:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.selectAnchor = -1
		m.searchResults = nil
		m.updateSliderFromSelection()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Enter):
		m.mode = modeNormal
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		if m.cursor > 0 {
			m.cursor--
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Right):
		if m.cursor < len(m.words)-1 {
			m.cursor++
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.ShiftLeft):
		if m.cursor > 0 {
			if m.selectAnchor < 0 {
				m.selectAnchor = m.cursor
				m.selectAnchorSelecting = !m.selected[m.cursor]
			}
			m.selected[m.cursor] = m.selectAnchorSelecting
			m.cursor--
			m.selected[m.cursor] = m.selectAnchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.ShiftRight):
		if m.cursor < len(m.words)-1 {
			if m.selectAnchor < 0 {
				m.selectAnchor = m.cursor
				m.selectAnchorSelecting = !m.selected[m.cursor]
			}
			m.selected[m.cursor] = m.selectAnchorSelecting
			m.cursor++
			m.selected[m.cursor] = m.selectAnchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		// Jump to first word of previous cue
		if m.cursor > 0 {
			curCue := m.words[m.cursor].CueIndex
			i := m.cursor - 1
			// Skip remaining words in current cue
			for i > 0 && m.words[i].CueIndex == curCue {
				i--
			}
			// Find first word of that cue
			prevCue := m.words[i].CueIndex
			for i > 0 && m.words[i-1].CueIndex == prevCue {
				i--
			}
			m.cursor = i
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		// Jump to first word of next cue
		if m.cursor < len(m.words)-1 {
			curCue := m.words[m.cursor].CueIndex
			i := m.cursor + 1
			for i < len(m.words) && m.words[i].CueIndex == curCue {
				i++
			}
			if i < len(m.words) {
				m.cursor = i
			}
		}
		return m, nil

	case key.Matches(msg, keys.Space):
		m.selected[m.cursor] = !m.selected[m.cursor]
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.ParagraphSelect):
		m.toggleParagraph()
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Deselect):
		for i := range m.selected {
			m.selected[i] = false
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Search):
		m.mode = modeSearchSelect
		m.searchInput.Reset()
		m.searchResults = nil
		m.searchIndex = 0
		return m, m.searchInput.Focus()

	case key.Matches(msg, keys.NextMatch):
		if len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchResults)
			m.cursor = m.searchResults[m.searchIndex]
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchResults)) % len(m.searchResults)
			m.cursor = m.searchResults[m.searchIndex]
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

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
