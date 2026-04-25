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
	if !m.anim.active {
		m.anim.active = true
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
			m.sel.selected = make([]bool, len(m.words))
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
		if !m.anim.active {
			return m, nil
		}
		m.anim.startPos, m.anim.startVel = m.anim.spring.Update(m.anim.startPos, m.anim.startVel, m.startPos)
		m.anim.endPos, m.anim.endVel = m.anim.spring.Update(m.anim.endPos, m.anim.endVel, m.endPos)
		// Settle threshold: half a column width in seconds
		threshold := m.duration / float64(m.sliderWidth()) / 2
		startSettled := math.Abs(m.anim.startPos-m.startPos) < threshold && math.Abs(m.anim.startVel) < threshold
		endSettled := math.Abs(m.anim.endPos-m.endPos) < threshold && math.Abs(m.anim.endVel) < threshold
		if startSettled && endSettled {
			m.anim.startPos = m.startPos
			m.anim.endPos = m.endPos
			m.anim.startVel = 0
			m.anim.endVel = 0
			m.anim.active = false
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
		m.search.input, cmd = m.search.input.Update(msg)
		return m, cmd
	case m.mode == modeInput:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	}
	return m, nil
}
