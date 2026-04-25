package slider

import (
	"dis/internal/tui/slider/keys"
	"math"

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

func (m *Model) updateSearchResults() {
	if m.search.input.Value() == "" {
		m.search.results = nil
		m.search.index = 0
		return
	}
	if m.mode == modeSearchSelect {
		m.search.results = m.transcript.SearchWords(m.words, m.search.input.Value())
	} else {
		m.search.results = m.transcript.Search(m.search.input.Value())
	}
	m.search.index = 0
}

func (m *Model) snapToCueSearchResult() {
	if len(m.search.results) == 0 || m.transcript == nil {
		return
	}
	idx := m.search.results[m.search.index]
	if idx >= 0 && idx < len(m.transcript) {
		cueStart := m.transcript[idx].Start
		if m.adjustingStart {
			m.startPos = math.Max(0, math.Min(m.endPos-MillisecondStep, cueStart))
		} else {
			cueEnd := m.transcript[idx].End
			m.endPos = math.Max(m.startPos+MillisecondStep, math.Min(m.duration, cueEnd))
		}
		m.roundPositions()
	}
}

func (m *Model) snapToNextCue() {
	pos := m.activePos()
	next := m.transcript.NextCueStart(pos)
	if next < 0 {
		return
	}
	// If rounding would produce the same position, skip to the next cue
	if math.Round(next*100)/100 <= math.Round(pos*100)/100 {
		next = m.transcript.NextCueStart(next + 0.001)
		if next < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = math.Min(m.endPos-MillisecondStep, next)
	} else {
		m.endPos = math.Min(m.duration, next)
	}
	m.roundPositions()
}

func (m *Model) snapToPrevCue() {
	pos := m.activePos()
	prev := m.transcript.PrevCueStart(pos)
	if prev < 0 {
		return
	}
	// If rounding would produce the same position, skip to the previous cue
	if math.Round(prev*100)/100 >= math.Round(pos*100)/100 {
		prev = m.transcript.PrevCueStart(prev - 0.001)
		if prev < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = math.Max(0, prev)
	} else {
		m.endPos = math.Max(m.startPos+MillisecondStep, prev)
	}
	m.roundPositions()
}
