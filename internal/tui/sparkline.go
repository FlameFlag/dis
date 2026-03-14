package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// brailleLevels maps 0–7 to braille chars filling bottom-to-top.
var brailleLevels = []rune{'⠀', '⡀', '⣀', '⣤', '⣴', '⣶', '⣷', '⣿'}

func sparkColor(level int) lipgloss.Color {
	switch {
	case level <= 2:
		return ColorTeal
	case level <= 4:
		return ColorYellow
	default:
		return ColorPeach
	}
}

func (m progressModel) renderBrailleWave(barW int) string {
	filled := max(min(int(m.displayPct/100.0*float64(barW)), barW), 0)

	var b strings.Builder
	for i := range filled {
		level := waveCenter + waveAmplitude*math.Sin(float64(i)*waveFrequency+m.wavePhase)
		li := max(0, min(int(math.Round(level)), maxSparkLevel))
		styled := lipgloss.NewStyle().Foreground(sparkColor(li)).Render(string(brailleLevels[li]))
		b.WriteString(styled)
	}
	if filled < barW {
		empty := strings.Repeat(" ", barW-filled)
		b.WriteString(lipgloss.NewStyle().Foreground(ColorSurface1).Render(empty))
	}
	return b.String()
}

var (
	progressMsgStyle = lipgloss.NewStyle().Foreground(ColorText)
	progressETAStyle = lipgloss.NewStyle().Foreground(ColorOverlay0)
	progressPctStyle = lipgloss.NewStyle().Foreground(ColorTeal)
)

// formatETAShort returns a short ETA string like "4s" or "1m12s".
func formatETAShort(d time.Duration) string {
	d = max(d.Round(time.Second), 0)
	s := int(math.Round(d.Seconds()))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}
