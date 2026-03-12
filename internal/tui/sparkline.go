package tui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// brailleLevels maps 0–7 to braille chars filling bottom-to-top.
var brailleLevels = []rune{'⠀', '⡀', '⣀', '⣤', '⣴', '⣶', '⣷', '⣿'}

// sparkline characters and heat-map colors.
var sparkChars = []rune("▁▂▃▄▅▆▇█")

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

func (m progressModel) renderSparkline(w int) string {
	if w < 1 || m.ringLen == 0 {
		return ""
	}

	// Collect last w samples from ring buffer
	n := min(w, m.ringLen)

	samples := make([]float64, n)
	start := (m.ringHead - n + len(m.speedRing)) % len(m.speedRing)
	for i := range n {
		samples[i] = m.speedRing[(start+i)%len(m.speedRing)]
	}

	maxSpeed := slices.Max(samples)

	var b strings.Builder
	for _, s := range samples {
		level := 0
		if maxSpeed > 0 {
			level = int(s / maxSpeed * maxSparkLevel)
		}
		level = max(0, min(level, maxSparkLevel))
		styled := lipgloss.NewStyle().Foreground(sparkColor(level)).Render(string(sparkChars[level]))
		b.WriteString(styled)
	}
	return b.String()
}

func (m progressModel) renderBrailleWave(barW int) string {
	filled := max(min(int(m.displayPct/100.0*float64(barW)), barW), 0)

	var b strings.Builder
	for i := 0; i < filled; i++ {
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
	progressMsgStyle   = lipgloss.NewStyle().Foreground(ColorText)
	progressTealStyle  = lipgloss.NewStyle().Foreground(ColorTeal)
	progressSpeedStyle = lipgloss.NewStyle().Foreground(ColorSubtext0)
	progressETAStyle   = lipgloss.NewStyle().Foreground(ColorOverlay0)
	progressPctStyle   = lipgloss.NewStyle().Foreground(ColorTeal)
)

// formatScaled formats a value using unit scaling with the given suffixes.
func formatScaled(value float64, unit float64, suffixes []string) string {
	exp := 0
	val := value
	for val >= unit && exp < len(suffixes)-1 {
		val /= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", val, suffixes[exp])
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(b int64) string {
	if b <= 0 {
		return "? MiB"
	}
	if b < int64(binaryKilo) {
		return fmt.Sprintf("%d B", b)
	}
	return formatScaled(float64(b), binaryKilo, []string{"B", "KiB", "MiB", "GiB", "TiB"})
}

// formatSpeed formats bytes/sec into a human-readable string.
func formatSpeed(bps float64) string {
	if bps <= 0 {
		return "0 B/s"
	}
	return formatScaled(bps, binaryKilo, []string{"B/s", "KiB/s", "MiB/s", "GiB/s"})
}

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
