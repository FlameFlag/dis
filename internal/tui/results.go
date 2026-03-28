package tui

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Padding(0, 1)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	greenStyle  = lipgloss.NewStyle().Foreground(ColorGreen).Padding(0, 1)
	redStyle    = lipgloss.NewStyle().Foreground(ColorRed).Padding(0, 1)
	borderStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(ColorSurface1).Padding(0, 1)
)

// PrintResultsTable prints a styled comparison table of original vs compressed size.
func PrintResultsTable(originalSize, compressedSize int64) {
	saved := originalSize - compressedSize
	savedPct := float64(saved) / float64(originalSize) * 100

	origStr := humanize.IBytes(uint64(originalSize))

	compStyle := greenStyle
	if compressedSize > originalSize {
		compStyle = redStyle
	}
	compStr := humanize.IBytes(uint64(compressedSize))

	savedLabel := "Saved"
	savedSymbol := "-"
	savedColorStyle := greenStyle
	if saved < 0 {
		savedLabel = "Increased"
		savedSymbol = "+"
		savedColorStyle = redStyle
	}

	pctStr := fmt.Sprintf("%s%.2f%%", savedSymbol, math.Abs(savedPct))
	savedStr := fmt.Sprintf("%s%s (%s)", savedSymbol, humanize.IBytes(uint64(int64(math.Abs(float64(saved))))), pctStr)

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		headerStyle.Render("Original"),
		headerStyle.Render("Compressed"),
		headerStyle.Render(savedLabel),
	)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cellStyle.Render(origStr),
		compStyle.Render(compStr),
		savedColorStyle.Render(savedStr),
	)

	table := lipgloss.JoinVertical(lipgloss.Left, header, row)
	fmt.Println(borderStyle.Render(table))
}
