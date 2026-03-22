package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var brailleSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

var spinnerMsgStyle = lipgloss.NewStyle().Foreground(ColorText)

type spinnerModel struct {
	message   string
	spinner   spinner.Model
	done      bool
	cancelled bool
	err       error
	doneCh    chan struct{}
}

type spinnerDoneMsg struct{ err error }

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitDone())
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
	case spinnerDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return " " + m.spinner.View() + " " + spinnerMsgStyle.Render(m.message) + "\n"
}

// RunWithSpinner displays an animated braille spinner while fn runs.
// Returns ErrUserCancelled if the user presses Ctrl+C.
func RunWithSpinner(ctx context.Context, message string, fn func() error) error {
	doneCh := make(chan struct{})
	s := spinner.New(spinner.WithSpinner(brailleSpinner), spinner.WithStyle(lipgloss.NewStyle().Foreground(ColorTeal)))
	m := spinnerModel{
		message: message,
		spinner: s,
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
