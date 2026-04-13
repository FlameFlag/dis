package tui

import (
	"dis/internal/tui/palette"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	ColorPeach    = palette.Resolved.Accent
	ColorYellow   = palette.Resolved.Warm
	ColorTeal     = palette.Resolved.Info
	ColorGreen    = palette.Resolved.Success
	ColorRed      = palette.Resolved.Error
	ColorText     = palette.Resolved.Text
	ColorSubtext0 = palette.Resolved.Subtext0
	ColorSurface0 = palette.Resolved.Surface0
	ColorSurface2 = palette.Resolved.Surface2
	ColorSurface1 = palette.Resolved.Surface1
	ColorOverlay0 = palette.Resolved.Overlay0
	ColorBase     = palette.Resolved.Base

	ColorFadeEnd   = palette.Resolved.FadeEnd
	ColorTrackDim  = palette.Resolved.TrackDim
	ColorTrackMid  = palette.Resolved.TrackMid
	ColorTrackWarm = palette.Resolved.TrackWarm
)

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
