package slider

import (
	"math"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Enter):
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
		return m, nil

	case key.Matches(msg, Escape):
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
			m.startPos = max(0, min(m.endPos-MillisecondStep, cueStart))
		} else {
			cueEnd := m.transcript[idx].End
			m.endPos = max(m.startPos+MillisecondStep, min(m.duration, cueEnd))
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
	if math.Round(next*positionRoundingScale)/positionRoundingScale <=
		math.Round(pos*positionRoundingScale)/positionRoundingScale {
		next = m.transcript.NextCueStart(next + searchCueEpsilon)
		if next < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = min(m.endPos-MillisecondStep, next)
	} else {
		m.endPos = min(m.duration, next)
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
	if math.Round(prev*positionRoundingScale)/positionRoundingScale >=
		math.Round(pos*positionRoundingScale)/positionRoundingScale {
		prev = m.transcript.PrevCueStart(prev - searchCueEpsilon)
		if prev < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = max(0, prev)
	} else {
		m.endPos = max(m.startPos+MillisecondStep, prev)
	}
	m.roundPositions()
}
