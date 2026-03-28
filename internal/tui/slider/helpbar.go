package slider

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func helpEntry(key, desc string) string {
	return helpKeyStyle.Render(key) + " " + faintStyle.Render(desc)
}

// alignColumns pads entries at each column position so separators align vertically.
func alignColumns(row1, row2 []string) {
	n := min(len(row1), len(row2))
	for i := range n {
		w1 := lipgloss.Width(row1[i])
		w2 := lipgloss.Width(row2[i])
		colW := max(w1, w2)
		if w1 < colW {
			row1[i] += strings.Repeat(" ", colW-w1)
		}
		if w2 < colW {
			row2[i] += strings.Repeat(" ", colW-w2)
		}
	}
}

func (m Model) renderHelpBar() string {
	sep := dimStyle.Render(" │ ")

	if m.isSearchMode() {
		return "  " + strings.Join([]string{
			faintStyle.Render("type to search"),
			helpEntry("⏎", "snap"),
			helpEntry("esc", "cancel"),
		}, sep)
	}

	if m.isSelectMode() {
		row1 := []string{
			helpEntry("←→", "word"),
			helpEntry("↑↓", "cue"),
			helpEntry("␣", "toggle"),
			helpEntry("shift+← shift+→", "range"),
			helpEntry("p", "sentence"),
		}
		row2 := []string{
			helpEntry("d", "clear"),
			helpEntry("/", "search"),
			helpEntry("esc", "back"),
			helpEntry("⏎", "done"),
		}
		alignColumns(row1, row2)
		return "  " + strings.Join(row1, sep) + "\n" + "  " + strings.Join(row2, sep)
	}

	if m.transcript != nil {
		row1 := []string{
			helpEntry("tab", "switch"),
			helpEntry("←→", "1s"),
			helpEntry("↑↓", "1m"),
			helpEntry("[]", "snap"),
			helpEntry("/", "search"),
		}
		row2 := []string{
			helpEntry("s", "split"),
			helpEntry("d", "undo"),
			helpEntry("g", "gif"),
			helpEntry("v", "speed"),
			helpEntry("t", "words"),
			helpEntry("⏎", "done"),
		}
		alignColumns(row1, row2)
		return "  " + strings.Join(row1, sep) + "\n" + "  " + strings.Join(row2, sep)
	}

	row1 := []string{
		helpEntry("tab", "switch"),
		helpEntry("←→", "1s"),
		helpEntry("↑↓", "1m"),
		helpEntry("shift", "10ms"),
		helpEntry("space", "type"),
	}
	row2 := []string{
		helpEntry("s", "split"),
		helpEntry("d", "undo"),
		helpEntry("g", "gif"),
		helpEntry("v", "speed"),
		helpEntry("⏎", "done"),
	}
	alignColumns(row1, row2)
	return "  " + strings.Join(row1, sep) + "\n" + "  " + strings.Join(row2, sep)
}
