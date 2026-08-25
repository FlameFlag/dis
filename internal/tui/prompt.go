package tui

import (
	"context"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

const promptButtonHorizontalPadding = 2

var promptTheme = huh.ThemeFunc(func(isDark bool) *huh.Styles {
	theme := huh.ThemeBase(isDark)

	theme.Focused.Base = theme.Focused.Base.BorderForeground(ColorSurface2)
	theme.Focused.Card = theme.Focused.Base
	theme.Focused.Title = lipgloss.NewStyle().Foreground(ColorPeach).Bold(true)
	theme.Focused.NoteTitle = theme.Focused.Title
	theme.Focused.Description = lipgloss.NewStyle().Foreground(ColorOverlay0)
	theme.Focused.ErrorIndicator = lipgloss.NewStyle().Foreground(ColorRed).SetString(" !")
	theme.Focused.ErrorMessage = lipgloss.NewStyle().Foreground(ColorRed)
	theme.Focused.SelectSelector = lipgloss.NewStyle().Foreground(ColorPeach).SetString("› ")
	theme.Focused.NextIndicator = lipgloss.NewStyle().Foreground(ColorPeach).SetString("↓")
	theme.Focused.PrevIndicator = lipgloss.NewStyle().Foreground(ColorPeach).SetString("↑")
	theme.Focused.Option = lipgloss.NewStyle().Foreground(ColorText)
	theme.Focused.MultiSelectSelector = theme.Focused.SelectSelector
	theme.Focused.SelectedOption = lipgloss.NewStyle().Foreground(ColorGreen)
	theme.Focused.SelectedPrefix = lipgloss.NewStyle().
		Foreground(ColorGreen).
		SetString("[✓] ")
	theme.Focused.UnselectedOption = lipgloss.NewStyle().Foreground(ColorText)
	theme.Focused.UnselectedPrefix = lipgloss.NewStyle().
		Foreground(ColorOverlay0).
		SetString("[ ] ")
	theme.Focused.FocusedButton = lipgloss.NewStyle().
		Foreground(ColorBase).
		Background(ColorPeach).
		Bold(true).
		Padding(0, promptButtonHorizontalPadding).
		MarginRight(1)
	theme.Focused.BlurredButton = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorSurface1).
		Padding(0, promptButtonHorizontalPadding).
		MarginRight(1)
	theme.Focused.TextInput.Cursor = lipgloss.NewStyle().Foreground(ColorPeach)
	theme.Focused.TextInput.Placeholder = lipgloss.NewStyle().Foreground(ColorOverlay0)
	theme.Focused.TextInput.Prompt = lipgloss.NewStyle().Foreground(ColorPeach)
	theme.Focused.TextInput.Text = lipgloss.NewStyle().Foreground(ColorText)

	theme.Blurred = theme.Focused
	theme.Blurred.Base = theme.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	theme.Blurred.Card = theme.Blurred.Base
	theme.Blurred.Title = lipgloss.NewStyle().Foreground(ColorOverlay0)
	theme.Blurred.NextIndicator = lipgloss.NewStyle()
	theme.Blurred.PrevIndicator = lipgloss.NewStyle()

	theme.Group.Title = theme.Focused.Title
	theme.Group.Description = theme.Focused.Description
	theme.Help.ShortKey = lipgloss.NewStyle().Foreground(ColorPeach).Bold(true)
	theme.Help.ShortDesc = lipgloss.NewStyle().Foreground(ColorOverlay0)
	theme.Help.ShortSeparator = lipgloss.NewStyle().Foreground(ColorSurface2)
	theme.Help.Ellipsis = theme.Help.ShortSeparator

	return theme
})

// RunPrompt runs a Huh field with cancellation support and adaptive key help.
func RunPrompt(ctx context.Context, field huh.Field) error {
	return huh.NewForm(huh.NewGroup(field)).
		WithTheme(promptTheme).
		RunWithContext(ctx)
}
