package slider

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		if m.inputBuffer != "" {
			m.processTimeInput()
		}
		m.mode = modeNormal
		m.inputBuffer = ""
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.inputBuffer = ""
		return m, nil

	case key.Matches(msg, keys.Backspace):
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil

	default:
		ch := msg.String()
		if len(ch) == 1 && isValidTimeChar(rune(ch[0])) {
			m.inputBuffer += ch
		}
		return m, nil
	}
}

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
		m.inputBuffer = ""
		return m, nil

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
			m.searchBuffer = ""
			m.searchResults = nil
			m.searchIndex = 0
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
		if m.transcript != nil && len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchResults)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if m.transcript != nil && len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchResults)) % len(m.searchResults)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.TranscriptSelect):
		if m.transcript != nil && len(m.words) > 0 {
			m.mode = modeSelect
			m.cursor = m.nearestWordIndex(m.activePos())
		}
		return m, nil

	case key.Matches(msg, keys.Split):
		// Save current range as a split (guard: end > start)
		if m.endPos > m.startPos {
			m.splits = append(m.splits, trimRange{start: m.startPos, end: m.endPos})
		}
		return m, nil

	case key.Matches(msg, keys.DeleteSplit):
		// Pop last saved split
		if len(m.splits) > 0 {
			m.splits = m.splits[:len(m.splits)-1]
		}
		return m, nil

	case key.Matches(msg, keys.GIFToggle):
		if !m.gifAvailable {
			m.warning = "gifski not found — install: brew install gifski"
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

func (m Model) handleSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.selectAnchor = -1
		m.searchResults = nil
		m.updateSliderFromSelection()
		return m, nil

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
		m.searchBuffer = ""
		m.searchResults = nil
		m.searchIndex = 0
		return m, nil

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
		// Confirm search and snap to current match
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
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
		} else {
			m.mode = modeNormal
		}
		m.searchBuffer = ""
		m.searchResults = nil
		return m, nil

	case key.Matches(msg, keys.Backspace):
		if len(m.searchBuffer) > 0 {
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
			m.updateSearchResults()
		}
		return m, nil

	default:
		ch := msg.String()
		if len(ch) == 1 {
			m.searchBuffer += ch
			m.updateSearchResults()
		}
		return m, nil
	}
}
