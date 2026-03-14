package slider

import (
	"dis/internal/util"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderVerticalWaveform(width int) string {
	if len(m.waveform) == 0 {
		return ""
	}

	barChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
	maxBarWidth := max(width-8, 4) // leave space for position indicator

	// Determine visible height (we'll use the available space)
	visibleRows := min(16, len(m.waveform))

	// Find which row the active handle corresponds to
	activePos := m.activePos()
	activeRow := int(activePos / m.duration * float64(visibleRows))
	if activeRow >= visibleRows {
		activeRow = visibleRows - 1
	}

	var lines []string

	for row := range visibleRows {
		// Map row to waveform sample
		sampleIdx := row * len(m.waveform) / visibleRows
		if sampleIdx >= len(m.waveform) {
			sampleIdx = len(m.waveform) - 1
		}
		amp := m.waveform[sampleIdx].Amplitude

		// Calculate bar width
		fullBlocks := int(amp * float64(maxBarWidth))
		fractional := amp*float64(maxBarWidth) - float64(fullBlocks)

		var bar strings.Builder
		for i := 0; i < fullBlocks && i < maxBarWidth; i++ {
			bar.WriteString("█")
		}
		if fullBlocks < maxBarWidth && fractional > 0.1 {
			level := int(fractional * float64(len(barChars)-1))
			if level >= len(barChars) {
				level = len(barChars) - 1
			}
			bar.WriteString(barChars[level])
		}

		barStr := bar.String()
		barWidth := lipgloss.Width(barStr)

		// Determine if this row is in the selected range
		rowTime := float64(row) / float64(visibleRows) * m.duration
		inRange := rowTime >= m.animStartPos && rowTime <= m.animEndPos

		var styledBar string
		if inRange {
			styledBar = accentStyle.Render(barStr)
		} else {
			styledBar = dimStyle.Render(barStr)
		}

		// Position indicator
		indicator := ""
		if row == activeRow {
			timeStr := util.FormatDurationShort(activePos)
			indicator = " " + accentBold.Render("◀ "+timeStr)
		}

		padding := max(maxBarWidth-barWidth, 0)
		line := "  " + styledBar + strings.Repeat(" ", padding) + indicator
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
