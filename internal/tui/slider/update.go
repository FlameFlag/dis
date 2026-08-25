package slider

import (
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

type warningExpiredMsg string

func expireWarning(warning string) tea.Cmd {
	return tea.Tick(warningDisplayDuration, func(time.Time) tea.Msg {
		return warningExpiredMsg(warning)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.keyHelp.SetWidth(max(msg.Width-keyHelpHorizontalInset, 0))
		return m, nil
	case warningExpiredMsg:
		if m.warning == string(msg) {
			m.warning = ""
		}
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
	case tea.KeyMsg:
		// Ctrl-C is unconditional, including while a text field or help is open.
		if key.Matches(msg, Cancel) {
			m.cancelled = true
			return m, tea.Quit
		}

		if m.helpVisible {
			switch {
			case key.Matches(msg, Help, Escape):
				m.helpVisible = false
				return m, nil
			case key.Matches(msg, Up):
				m.helpScroll = max(m.helpScroll-1, 0)
				return m, nil
			case key.Matches(msg, Down):
				m.helpScroll = min(m.helpScroll+1, m.helpMaxOffset())
				return m, nil
			case key.Matches(msg, PageUp):
				m.helpScroll = max(m.helpScroll-helpPageStep, 0)
				return m, nil
			case key.Matches(msg, PageDown):
				m.helpScroll = min(m.helpScroll+helpPageStep, m.helpMaxOffset())
				return m, nil
			case key.Matches(msg, Quit):
				m.cancelled = true
				return m, tea.Quit
			default:
				return m, nil
			}
		}

		// A question mark remains ordinary text while an input has focus.
		if !m.isSearchMode() && m.mode != modeInput && key.Matches(msg, Help) {
			m.helpVisible = true
			m.helpScroll = 0
			return m, nil
		}

		if !m.isSearchMode() && m.mode != modeInput && key.Matches(msg, Quit) {
			m.cancelled = true
			return m, tea.Quit
		}

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
