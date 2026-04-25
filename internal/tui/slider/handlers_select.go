package slider

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.sel.anchor = -1
		m.search.results = nil
		m.updateSliderFromSelection()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Enter):
		m.mode = modeNormal
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		if m.sel.cursor > 0 {
			m.sel.cursor--
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Right):
		if m.sel.cursor < len(m.words)-1 {
			m.sel.cursor++
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.ShiftLeft):
		if m.sel.cursor > 0 {
			if m.sel.anchor < 0 {
				m.sel.anchor = m.sel.cursor
				m.sel.anchorSelecting = !m.sel.selected[m.sel.cursor]
			}
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
			m.sel.cursor--
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.ShiftRight):
		if m.sel.cursor < len(m.words)-1 {
			if m.sel.anchor < 0 {
				m.sel.anchor = m.sel.cursor
				m.sel.anchorSelecting = !m.sel.selected[m.sel.cursor]
			}
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
			m.sel.cursor++
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		// Jump to first word of previous cue
		if m.sel.cursor > 0 {
			curCue := m.words[m.sel.cursor].CueIndex
			i := m.sel.cursor - 1
			// Skip remaining words in current cue
			for i > 0 && m.words[i].CueIndex == curCue {
				i--
			}
			// Find first word of that cue
			prevCue := m.words[i].CueIndex
			for i > 0 && m.words[i-1].CueIndex == prevCue {
				i--
			}
			m.sel.cursor = i
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		// Jump to first word of next cue
		if m.sel.cursor < len(m.words)-1 {
			curCue := m.words[m.sel.cursor].CueIndex
			i := m.sel.cursor + 1
			for i < len(m.words) && m.words[i].CueIndex == curCue {
				i++
			}
			if i < len(m.words) {
				m.sel.cursor = i
			}
		}
		return m, nil

	case key.Matches(msg, keys.Space):
		m.sel.selected[m.sel.cursor] = !m.sel.selected[m.sel.cursor]
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.ParagraphSelect):
		m.toggleParagraph()
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.SelectTrimRange):
		if len(m.splits) > 0 {
			m.selectWordsInRanges(m.splits)
		} else {
			m.selectWordsInRanges([]trimRange{{start: m.startPos, end: m.endPos}})
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Deselect):
		for i := range m.sel.selected {
			m.sel.selected[i] = false
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Search):
		m.mode = modeSearchSelect
		m.search.input.Reset()
		m.search.results = nil
		m.search.index = 0
		return m, m.search.input.Focus()

	case key.Matches(msg, keys.NextMatch):
		if len(m.search.results) > 0 {
			m.search.index = (m.search.index + 1) % len(m.search.results)
			m.sel.cursor = m.search.results[m.search.index]
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if len(m.search.results) > 0 {
			m.search.index = (m.search.index - 1 + len(m.search.results)) % len(m.search.results)
			m.sel.cursor = m.search.results[m.search.index]
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}
