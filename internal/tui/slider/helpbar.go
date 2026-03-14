package slider

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func helpEntry(key, desc string) string {
	return helpKeyStyle.Render(key) + " " + faintStyle.Render(desc)
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
		line1 := "  " + strings.Join([]string{
			helpEntry("←→", "word"),
			helpEntry("↑↓", "cue"),
			helpEntry("␣", "toggle"),
			helpEntry("⇧←⇧→", "range"),
			helpEntry("p", "sentence"),
		}, sep)
		line2 := "  " + strings.Join([]string{
			helpEntry("d", "clear"),
			helpEntry("/", "search"),
			helpEntry("esc", "back"),
			helpEntry("⏎", "done"),
		}, sep)
		return line1 + "\n" + line2
	}

	if m.transcript != nil {
		line1 := "  " + strings.Join([]string{
			helpEntry("tab", "switch"),
			helpEntry("←→", "1s"),
			helpEntry("↑↓", "1m"),
			helpEntry("[]", "snap"),
			helpEntry("/", "search"),
		}, sep)
		line2 := "  " + strings.Join([]string{
			helpEntry("s", "split"),
			helpEntry("d", "undo"),
			helpEntry("g", "gif"),
			helpEntry("v", "speed"),
			helpEntry("t", "words"),
			helpEntry("⏎", "done"),
		}, sep)
		return line1 + "\n" + line2
	}

	line1 := "  " + strings.Join([]string{
		helpEntry("tab", "switch"),
		helpEntry("←→", "1s"),
		helpEntry("↑↓", "1m"),
		helpEntry("shift", "10ms"),
		helpEntry("space", "type"),
	}, sep)
	line2 := "  " + strings.Join([]string{
		helpEntry("s", "split"),
		helpEntry("d", "undo"),
		helpEntry("g", "gif"),
		helpEntry("v", "speed"),
		helpEntry("⏎", "done"),
	}, sep)
	return line1 + "\n" + line2
}

// padRight pads a string with spaces to reach the target visual width.
func padRight(s string, targetWidth int) string {
	w := lipgloss.Width(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
}

// truncateVisual truncates a string to at most maxWidth visual columns.
func truncateVisual(s string, maxWidth int) string {
	w := 0
	for i, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > maxWidth {
			return s[:i]
		}
		w += rw
	}
	return s
}
