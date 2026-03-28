package slider

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func helpPill(key, desc string) string {
	if strings.HasPrefix(key, "[") && strings.HasSuffix(key, "]") {
		return helpPillStyle.Render(desc + " " + key)
	}
	return helpPillStyle.Render(desc + " [" + key + "]")
}

func (m Model) renderHelpBar() string {
	if m.isSearchMode() {
		pills := []string{
			helpPill("type", "search"),
			helpPill("⏎", "snap"),
			helpPill("esc", "cancel"),
		}
		return joinPillRows(pills, m.helpBarWidth())
	}

	if m.isSelectMode() {
		pills := []string{
			helpPill("←→", "word"),
			helpPill("↑↓", "cue"),
			helpPill("␣", "toggle"),
			helpPill("shift+← shift+→", "range"),
			helpPill("p", "sentence"),
			helpPill("a", "trim range"),
			helpPill("d", "clear"),
			helpPill("/", "search"),
			helpPill("esc", "back"),
			helpPill("⏎", "done"),
		}
		return joinPillRows(pills, m.helpBarWidth())
	}

	if m.transcript != nil {
		pills := []string{
			helpPill("tab", "switch"),
			helpPill("←→", "1s"),
			helpPill("↑↓", "1m"),
			helpPill("[]", "snap"),
			helpPill("/", "search"),
			helpPill("s", "split"),
			helpPill("d", "undo"),
			helpPill("g", "gif"),
			helpPill("v", "speed"),
			helpPill("t", "words"),
			helpPill("⏎", "done"),
		}
		return joinPillRows(pills, m.helpBarWidth())
	}

	pills := []string{
		helpPill("tab", "switch"),
		helpPill("←→", "1s"),
		helpPill("↑↓", "1m"),
		helpPill("shift", "10ms"),
		helpPill("space", "type"),
		helpPill("s", "split"),
		helpPill("d", "undo"),
		helpPill("g", "gif"),
		helpPill("v", "speed"),
		helpPill("⏎", "done"),
	}
	return joinPillRows(pills, m.helpBarWidth())
}

// helpBarWidth returns the available inner width for the help bar.
func (m Model) helpBarWidth() int {
	if m.isTwoPane() {
		return m.leftPaneWidth() + m.rightPaneWidth() + 1
	}
	return m.width - 2
}

// joinPillRows lays out pills horizontally, wrapping to a second row if needed.
func joinPillRows(pills []string, maxWidth int) string {
	var rows []string
	var current []string
	lineW := 0

	for _, p := range pills {
		pw := lipgloss.Width(p)
		needed := pw
		if len(current) > 0 {
			needed += 1 // space separator
		}
		if lineW+needed > maxWidth && len(current) > 0 {
			rows = append(rows, strings.Join(current, " "))
			current = nil
			lineW = 0
		}
		current = append(current, p)
		if lineW == 0 {
			lineW = pw
		} else {
			lineW += 1 + pw
		}
	}
	if len(current) > 0 {
		rows = append(rows, strings.Join(current, " "))
	}

	return strings.Join(rows, "\n")
}
