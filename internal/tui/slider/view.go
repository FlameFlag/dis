package slider

import (
	"dis/internal/tui/slider/style"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width < 20 {
		return ""
	}

	helpBar := m.renderHelpBar()
	helpH := strings.Count(helpBar, "\n") + 1

	// Available content height: terminal height minus border chrome
	// top border (1) + bottom border (1) + help lines + final newline (1)
	contentHeight := 0
	if m.height > 0 {
		overhead := 1 + 1 + helpH + 1
		if m.isSearchMode() {
			overhead++
		}
		contentHeight = max(m.height-overhead, 0)
	}

	if m.isTwoPane() {
		leftW := m.leftPaneWidth()
		rightW := m.rightPaneWidth()
		left := m.renderLeftPaneWithHeight(leftW, contentHeight)
		leftHeight := strings.Count(left, "\n") + 1
		targetHeight := max(leftHeight, contentHeight)
		right := m.renderRightPaneWithHeight(rightW, targetHeight)
		body := m.renderBorderedLayout(left, leftW, right, rightW, helpBar, contentHeight)
		return body + "\n"
	}

	// Single-column fallback
	innerW := m.width - 2
	left := m.renderLeftPaneWithHeight(innerW, contentHeight)
	body := m.renderSingleColumnLayout(left, innerW, helpBar, contentHeight)
	return body + "\n"
}

func (m Model) renderBorderedLayout(left string, leftW int, right string, rightW int, helpBar string, contentHeight int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Equalize heights, ensuring we fill contentHeight
	maxH := max(len(leftLines), len(rightLines), contentHeight)
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	// Right pane border color: peach in select mode, default otherwise
	divColor := style.Border
	if m.isSelectMode() {
		divColor = style.Accent
	}

	// Build right pane title
	rightTitle := m.rightPaneTitle()

	var b strings.Builder

	// Top border: ┌─ Timeline ─────────────┬─ Transcript ──────────┐
	leftTitle := " Timeline "
	topLeft := "─" + style.Border.Render(leftTitle) + style.Border.Render(strings.Repeat("─", max(leftW-lipgloss.Width(leftTitle)-1, 0)))
	topRight := "─" + divColor.Render(rightTitle) + divColor.Render(strings.Repeat("─", max(rightW-lipgloss.Width(rightTitle)-1, 0)))
	b.WriteString(style.Border.Render("┌") + style.Border.Render(topLeft) + divColor.Render("┬") + divColor.Render(topRight) + divColor.Render("┐") + "\n")

	// Body rows
	leftPad := lipgloss.NewStyle().Width(leftW)
	rightPad := lipgloss.NewStyle().Width(rightW)
	for i := range maxH {
		ll := leftPad.Render(leftLines[i])
		rl := rightPad.Render(rightLines[i])
		b.WriteString(style.Border.Render("│") + ll + divColor.Render("│") + rl + divColor.Render("│") + "\n")
	}

	// Search input sits above the bottom border if active
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(m.width-2-lipgloss.Width(searchLine), 0)
		b.WriteString(style.Border.Render("│") + searchLine + strings.Repeat(" ", searchPad) + style.Border.Render("│") + "\n")
	}

	// Bottom border
	b.WriteString(style.Border.Render("└") + style.Border.Render(strings.Repeat("─", leftW)) + style.Border.Render("┴") + style.Border.Render(strings.Repeat("─", rightW)) + style.Border.Render("┘") + "\n")

	// Help bar below the box
	b.WriteString(helpBar)

	return b.String()
}

func (m Model) renderSingleColumnLayout(content string, innerW int, helpBar string, contentHeight int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	var b strings.Builder

	// Top border
	b.WriteString(style.Border.Render("┌") + style.Border.Render("─ Timeline "+strings.Repeat("─", max(innerW-11, 0))) + style.Border.Render("┐") + "\n")

	innerPad := lipgloss.NewStyle().Width(innerW)
	for _, line := range lines {
		b.WriteString(style.Border.Render("│") + innerPad.Render(line) + style.Border.Render("│") + "\n")
	}

	// Search input
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(innerW-lipgloss.Width(searchLine), 0)
		b.WriteString(style.Border.Render("│") + searchLine + strings.Repeat(" ", searchPad) + style.Border.Render("│") + "\n")
	}

	// Bottom border
	b.WriteString(style.Border.Render("└") + style.Border.Render(strings.Repeat("─", innerW)) + style.Border.Render("┘") + "\n")

	// Help bar below the box
	b.WriteString(helpBar)

	return b.String()
}

func (m Model) rightPaneTitle() string {
	if m.isSelectMode() && len(m.words) > 0 {
		selCount := m.selectedWordCount()
		return fmt.Sprintf(" Select Words %d/%d ", selCount, len(m.words))
	}
	return " Transcript "
}
