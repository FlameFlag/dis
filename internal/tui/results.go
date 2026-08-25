package tui

import (
	"fmt"
	"math"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/dustin/go-humanize"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorText).Padding(0, 1)
	cellStyle   = lipgloss.NewStyle().Padding(0, 1)
	greenStyle  = lipgloss.NewStyle().Foreground(ColorGreen).Padding(0, 1)
	redStyle    = lipgloss.NewStyle().Foreground(ColorRed).Padding(0, 1)
)

const percentComplete = 100.0

const (
	compressedColumn = iota + 1
	savedColumn
)

// PrintResultsTable prints a styled comparison table of original vs compressed size.
func PrintResultsTable(originalSize, compressedSize int64) {
	saved := originalSize - compressedSize
	var savedPct float64
	if originalSize != 0 {
		savedPct = float64(saved) / float64(originalSize) * percentComplete
	}

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

	columnStyles := map[int]lipgloss.Style{
		compressedColumn: compStyle,
		savedColumn:      savedColorStyle,
	}
	results := table.New().
		Headers("Original", "Compressed", savedLabel).
		Row(origStr, compStr, savedStr).
		BorderColumn(false).
		BorderHeader(false).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorSurface1)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if columnStyle, ok := columnStyles[col]; ok {
				return columnStyle
			}
			return cellStyle
		})

	fmt.Println(results)
}
