package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var brailleSpinner = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

var (
	spinnerCharStyle = lipgloss.NewStyle().Foreground(ColorTeal)
	spinnerMsgStyle  = lipgloss.NewStyle().Foreground(ColorText)
)

type spinnerModel struct {
	message   string
	frame     int
	done      bool
	cancelled bool
	err       error
	doneCh    chan struct{}
}

type (
	spinnerTickMsg struct{}
	spinnerDoneMsg struct{ err error }
)

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.tick(), m.waitDone())
}

func (m spinnerModel) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

func (m spinnerModel) waitDone() tea.Cmd {
	return func() tea.Msg {
		<-m.doneCh
		return nil
	}
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}
	case spinnerTickMsg:
		m.frame = (m.frame + 1) % len(brailleSpinner)
		return m, m.tick()
	case spinnerDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	char := spinnerCharStyle.Render(string(brailleSpinner[m.frame]))
	return " " + char + " " + spinnerMsgStyle.Render(m.message) + "\n"
}

// RunWithSpinner displays an animated braille spinner while fn runs.
// Returns ErrUserCancelled if the user presses Ctrl+C.
func RunWithSpinner(ctx context.Context, message string, fn func() error) error {
	doneCh := make(chan struct{})
	m := spinnerModel{
		message: message,
		doneCh:  doneCh,
	}

	p := tea.NewProgram(m, tea.WithContext(ctx))

	go func() {
		fnErr := fn()
		close(doneCh)
		p.Send(spinnerDoneMsg{err: fnErr})
	}()

	result, err := p.Run()
	if err != nil {
		return err
	}
	if final, ok := result.(spinnerModel); ok {
		if final.cancelled || !final.done {
			return ErrUserCancelled
		}
		return final.err
	}
	return nil
}

// RunWithSpinnerResult is like RunWithSpinner but returns a value from fn.
func RunWithSpinnerResult[T any](ctx context.Context, message string, fn func() (T, error)) (T, error) {
	var result T
	err := RunWithSpinner(ctx, message, func() error {
		var fnErr error
		result, fnErr = fn()
		return fnErr
	})
	return result, err
}
