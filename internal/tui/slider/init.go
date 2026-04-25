package slider

import (
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/subtitle"

	tea "github.com/charmbracelet/bubbletea"
)

// StoryboardReadyMsg is sent when background storyboard fetch completes.
type StoryboardReadyMsg struct {
	Data *storyboard.StoryboardData
}

// TranscriptReadyMsg is sent when background transcript fetch completes.
type TranscriptReadyMsg struct {
	Transcript subtitle.Transcript
}

// SponsorSegsReadyMsg is sent when background SponsorBlock fetch completes.
type SponsorSegsReadyMsg struct {
	Segments []sponsorblock.Segment
}

// waitForChan returns a tea.Cmd that blocks on ch and wraps the received
// value with wrap. A closed channel produces wrap's zero-value message.
func waitForChan[T any, M tea.Msg](ch <-chan T, wrap func(T) M) tea.Cmd {
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			var zero T
			return wrap(zero)
		}
		return wrap(v)
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.storyboardCh != nil {
		cmds = append(cmds, waitForChan(m.storyboardCh, func(d *storyboard.StoryboardData) StoryboardReadyMsg {
			return StoryboardReadyMsg{Data: d}
		}))
	}
	if m.transcriptCh != nil {
		cmds = append(cmds, waitForChan(m.transcriptCh, func(t subtitle.Transcript) TranscriptReadyMsg {
			return TranscriptReadyMsg{Transcript: t}
		}))
	}
	if m.sponsorSegsCh != nil {
		cmds = append(cmds, waitForChan(m.sponsorSegsCh, func(s []sponsorblock.Segment) SponsorSegsReadyMsg {
			return SponsorSegsReadyMsg{Segments: s}
		}))
	}

	if m.isLoading() {
		cmds = append(cmds, m.loadingSpinner.Tick)
	}

	return tea.Batch(cmds...)
}
