package slider

import (
	"math"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type animTickMsg struct{}

func animTick() tea.Cmd {
	return tea.Tick(time.Second/AnimFPS, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
}

func (m *Model) triggerAnim() tea.Cmd {
	if !m.animating {
		m.animating = true
		return animTick()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case StoryboardReadyMsg:
		m.storyboard = msg.Data
		m.storyboardCh = nil
		return m, nil
	case TranscriptReadyMsg:
		m.transcript = msg.Transcript
		m.transcriptCh = nil
		if len(m.transcript) > 0 {
			m.words = m.transcript.Words()
			m.selected = make([]bool, len(m.words))
		}
		return m, nil
	case SponsorSegsReadyMsg:
		m.sponsorSegments = msg.Segments
		m.sponsorSegsCh = nil
		return m, nil
	case spinner.TickMsg:
		if m.isLoading() {
			var cmd tea.Cmd
			m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case animTickMsg:
		if m.warning != "" && time.Now().After(m.warningExpiry) {
			m.warning = ""
		}
		if !m.animating {
			return m, nil
		}
		m.animStartPos, m.animStartVel = m.animSpring.Update(m.animStartPos, m.animStartVel, m.startPos)
		m.animEndPos, m.animEndVel = m.animSpring.Update(m.animEndPos, m.animEndVel, m.endPos)
		// Settle threshold: half a column width in seconds
		threshold := m.duration / float64(m.sliderWidth()) / 2
		startSettled := math.Abs(m.animStartPos-m.startPos) < threshold && math.Abs(m.animStartVel) < threshold
		endSettled := math.Abs(m.animEndPos-m.endPos) < threshold && math.Abs(m.animEndVel) < threshold
		if startSettled && endSettled {
			m.animStartPos = m.startPos
			m.animEndPos = m.endPos
			m.animStartVel = 0
			m.animEndVel = 0
			m.animating = false
			return m, nil
		}
		return m, animTick()
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch, modeSearchSelect:
			return m.handleSearchMode(msg)
		case modeInput:
			return m.handleInputMode(msg)
		case modeSelect:
			return m.handleSelectMode(msg)
		default:
			return m.handleNavigation(msg)
		}
	}
	// Route cursor blink messages to active textinput
	switch {
	case m.isSearchMode():
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	case m.mode == modeInput:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	}
	return m, nil
}
