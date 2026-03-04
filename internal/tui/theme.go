package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Catppuccin Macchiato palette.
const (
	ColorPeach    = lipgloss.Color("#f5a97f")
	ColorYellow   = lipgloss.Color("#eed49f")
	ColorTeal     = lipgloss.Color("#8bd5ca")
	ColorGreen    = lipgloss.Color("#a6da95")
	ColorRed      = lipgloss.Color("#ed8796")
	ColorText     = lipgloss.Color("#cad3f5")
	ColorSubtext0 = lipgloss.Color("#a5adcb")
	ColorSurface2 = lipgloss.Color("#5b6078")
	ColorSurface1 = lipgloss.Color("#494d64")
	ColorOverlay0 = lipgloss.Color("#6e738d")
	ColorBase     = lipgloss.Color("#24273a")
)

// ConfigureLogger styles the charmbracelet/log default logger with Catppuccin colors.
func ConfigureLogger() {
	styles := log.DefaultStyles()

	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("INFO").
		Foreground(ColorGreen).
		Bold(true)
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("WARN").
		Foreground(ColorYellow).
		Bold(true)
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("ERRO").
		Foreground(ColorRed).
		Bold(true)
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("DEBU").
		Foreground(ColorOverlay0)

	styles.Key = lipgloss.NewStyle().Foreground(ColorSubtext0)
	styles.Value = lipgloss.NewStyle().Foreground(ColorText)
	styles.Separator = lipgloss.NewStyle().Foreground(ColorSurface2).SetString("=")

	log.SetStyles(styles)
}
