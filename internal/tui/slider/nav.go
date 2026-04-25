package slider

import (
	"dis/internal/tui/slider/keys"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Step-adjust keys: all follow the same pattern
	for _, s := range navigationSteps {
		if key.Matches(msg, s.binding) {
			m.adjustValue(s.step)
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
	}

	switch {
	case key.Matches(msg, keys.Escape):
		m.cancelled = true
		return m, tea.Quit

	case key.Matches(msg, keys.Enter):
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.SelectStart):
		m.adjustingStart = true
		return m, nil

	case key.Matches(msg, keys.SelectEnd):
		m.adjustingStart = false
		return m, nil

	case key.Matches(msg, keys.Tab):
		m.adjustingStart = !m.adjustingStart
		return m, nil

	case key.Matches(msg, keys.Space):
		m.mode = modeInput
		m.timeInput.Reset()
		return m, m.timeInput.Focus()

	case key.Matches(msg, keys.PageUp):
		if m.transcript != nil {
			m.viewportLocked = false
			m.transcriptOffset -= TranscriptVisibleCues
			if m.transcriptOffset < 0 {
				m.transcriptOffset = 0
			}
		}
		return m, nil

	case key.Matches(msg, keys.PageDown):
		if m.transcript != nil {
			m.viewportLocked = false
			m.transcriptOffset += TranscriptVisibleCues
			maxOffset := max(len(m.transcript)-TranscriptVisibleCues, 0)
			if m.transcriptOffset > maxOffset {
				m.transcriptOffset = maxOffset
			}
		}
		return m, nil

	case key.Matches(msg, keys.Search):
		if m.transcript != nil {
			m.mode = modeSearch
			m.search.input.Reset()
			m.search.results = nil
			m.search.index = 0
			return m, m.search.input.Focus()
		}
		return m, nil

	case key.Matches(msg, keys.NextCue):
		if m.transcript != nil {
			m.snapToNextCue()
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
		return m, nil

	case key.Matches(msg, keys.PrevCue):
		if m.transcript != nil {
			m.snapToPrevCue()
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
		return m, nil

	case key.Matches(msg, keys.NextMatch):
		if m.transcript != nil && len(m.search.results) > 0 {
			m.search.index = (m.search.index + 1) % len(m.search.results)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if m.transcript != nil && len(m.search.results) > 0 {
			m.search.index = (m.search.index - 1 + len(m.search.results)) % len(m.search.results)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.TranscriptSelect):
		if m.transcript != nil && len(m.words) > 0 {
			m.mode = modeSelect
			m.sel.cursor = m.nearestWordIndex(m.activePos())
		}
		return m, nil

	case key.Matches(msg, keys.Split):
		// Save current range as a split (guard: end > start)
		if m.endPos > m.startPos {
			m.splits = append(m.splits, trimRange{start: m.startPos, end: m.endPos})
			m.startPos = 0
			m.endPos = m.duration
		}
		return m, m.triggerAnim()

	case key.Matches(msg, keys.DeleteSplit):
		// Pop last saved split
		if len(m.splits) > 0 {
			m.splits = m.splits[:len(m.splits)-1]
		}
		return m, nil

	case key.Matches(msg, keys.GIFToggle):
		if !m.gifAvailable {
			m.warning = "gifski not found - install: brew install gifski"
			m.warningExpiry = time.Now().Add(2 * time.Second)
		} else {
			m.gifMode = !m.gifMode
		}
		return m, nil

	case key.Matches(msg, keys.SpeedToggle):
		switch m.speedMultiplier {
		case 1.0:
			m.speedMultiplier = 1.5
		case 1.5:
			m.speedMultiplier = 2.0
		default:
			m.speedMultiplier = 1.0
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}
