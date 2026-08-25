package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	huhspinner "charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
)

var spinnerTheme = huhspinner.ThemeFunc(func(bool) *huhspinner.Styles {
	return &huhspinner.Styles{
		Spinner: lipgloss.NewStyle().Foreground(ColorTeal),
		Title:   lipgloss.NewStyle().Foreground(ColorText),
	}
})

// RunWithSpinner displays an animated braille spinner while fn runs.
// Returns ErrUserCancelled if the user presses Ctrl+C.
func RunWithSpinner(ctx context.Context, message string, fn func(context.Context) error) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := huhspinner.New().
		Type(huhspinner.MiniDot).
		Title(" " + message).
		WithTheme(spinnerTheme).
		Context(runCtx).
		ActionWithErr(fn).
		Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return ErrUserCancelled
	}
	if err != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// RunWithSpinnerResult is like RunWithSpinner but returns a value from fn.
func RunWithSpinnerResult[T any](ctx context.Context, message string, fn func(context.Context) (T, error)) (T, error) {
	var result T
	err := RunWithSpinner(ctx, message, func(runCtx context.Context) error {
		var fnErr error
		result, fnErr = fn(runCtx)
		return fnErr
	})
	return result, err
}
