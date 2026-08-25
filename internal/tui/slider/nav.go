package slider

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Step-adjust keys: all follow the same pattern
	for _, s := range navigationSteps {
		if key.Matches(msg, s.binding) {
			m.adjustValue(s.step)
			m.viewportLocked = true
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, Escape):
		m.cancelled = true
		return m, tea.Quit

	case key.Matches(msg, Enter):
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, SelectStart):
		m.adjustingStart = true
		return m, nil

	case key.Matches(msg, SelectEnd):
		m.adjustingStart = false
		return m, nil

	case key.Matches(msg, Tab):
		m.adjustingStart = !m.adjustingStart
		return m, nil

	case key.Matches(msg, Space):
		m.mode = modeInput
		m.timeInput.Reset()
		return m, m.timeInput.Focus()

	case key.Matches(msg, PageUp):
		if m.transcript != nil {
			m.viewportLocked = false
			m.transcriptOffset = max(m.transcriptOffset-TranscriptVisibleCues, 0)
		}
		return m, nil

	case key.Matches(msg, PageDown):
		if m.transcript != nil {
			m.viewportLocked = false
			maxOffset := max(len(m.transcript)-TranscriptVisibleCues, 0)
			m.transcriptOffset = min(m.transcriptOffset+TranscriptVisibleCues, maxOffset)
		}
		return m, nil

	case key.Matches(msg, Search):
		if m.transcript != nil {
			m.mode = modeSearch
			m.search.input.Reset()
			m.search.results = nil
			m.search.index = 0
			return m, m.search.input.Focus()
		}
		return m, nil

	case key.Matches(msg, NextCue):
		if m.transcript != nil {
			m.snapToNextCue()
			m.viewportLocked = true
			return m, nil
		}
		return m, nil

	case key.Matches(msg, PrevCue):
		if m.transcript != nil {
			m.snapToPrevCue()
			m.viewportLocked = true
			return m, nil
		}
		return m, nil

	case key.Matches(msg, NextMatch):
		if m.transcript != nil && len(m.search.results) > 0 {
			m.search.index = (m.search.index + 1) % len(m.search.results)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, PrevMatch):
		if m.transcript != nil && len(m.search.results) > 0 {
			m.search.index = (m.search.index - 1 + len(m.search.results)) % len(m.search.results)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, TranscriptSelect):
		if m.transcript != nil && len(m.words) > 0 {
			m.mode = modeSelect
			m.sel.cursor = m.nearestWordIndex(m.activePos())
		}
		return m, nil

	case key.Matches(msg, Split):
		// Save current range as a split (guard: end > start)
		if m.endPos > m.startPos {
			m.splits = append(m.splits, trimRange{start: m.startPos, end: m.endPos})
			m.startPos = 0
			m.endPos = m.duration
		}
		return m, nil

	case key.Matches(msg, DeleteSplit):
		// Pop last saved split
		if len(m.splits) > 0 {
			m.splits = m.splits[:len(m.splits)-1]
		}
		return m, nil

	case key.Matches(msg, GIFToggle):
		if !m.gifAvailable {
			m.warning = "GIF export needs gifski. Install it with: brew install gifski"
			return m, expireWarning(m.warning)
		} else {
			m.gifMode = !m.gifMode
		}
		return m, nil

	case key.Matches(msg, SpeedToggle):
		index := slices.Index(playbackSpeeds[:], m.speedMultiplier)
		m.speedMultiplier = playbackSpeeds[(index+1)%len(playbackSpeeds)]
		return m, nil

	case key.Matches(msg, Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}
