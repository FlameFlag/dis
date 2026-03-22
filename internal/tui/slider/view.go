package slider

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width < 20 {
		return ""
	}

	helpBar := m.renderHelpBar()

	if m.isTwoPane() {
		leftW := m.leftPaneWidth()
		rightW := m.rightPaneWidth()
		left := m.renderLeftPane(leftW)
		right := m.renderRightPane(rightW)
		body := m.renderBorderedLayout(left, leftW, right, rightW)
		return body + "\n" + helpBar + "\n"
	}

	// Single-column fallback
	left := m.renderLeftPane(m.width - 2)
	body := m.renderSingleColumnLayout(left, m.width-2)
	return body + "\n" + helpBar + "\n"
}

func (m Model) renderBorderedLayout(left string, leftW int, right string, rightW int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Equalize heights
	maxH := max(len(leftLines), len(rightLines))
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	// Right pane border color: peach in select mode, default otherwise
	divColor := borderStyle
	if m.isSelectMode() {
		divColor = accentStyle
	}

	// Build right pane title
	rightTitle := m.rightPaneTitle()

	var b strings.Builder

	// Top border: ╭─ Timeline ─────────────┬─ Transcript ──────────╮
	leftTitle := " Timeline "
	topLeft := "─" + borderStyle.Render(leftTitle) + borderStyle.Render(strings.Repeat("─", max(leftW-lipgloss.Width(leftTitle)-1, 0)))
	topRight := "─" + divColor.Render(rightTitle) + divColor.Render(strings.Repeat("─", max(rightW-lipgloss.Width(rightTitle)-1, 0)))
	b.WriteString(borderStyle.Render("╭") + borderStyle.Render(topLeft) + divColor.Render("┬") + divColor.Render(topRight) + divColor.Render("╮") + "\n")

	// Body rows
	leftPad := lipgloss.NewStyle().Width(leftW)
	rightPad := lipgloss.NewStyle().Width(rightW)
	for i := range maxH {
		ll := leftPad.Render(leftLines[i])
		rl := rightPad.Render(rightLines[i])
		b.WriteString(borderStyle.Render("│") + ll + divColor.Render("│") + rl + divColor.Render("│") + "\n")
	}

	// Bottom border: ╰─────────────────────────┴──────────────────────╯
	// Search input sits above the bottom border if active
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(m.width-2-lipgloss.Width(searchLine), 0)
		b.WriteString(borderStyle.Render("│") + searchLine + strings.Repeat(" ", searchPad) + borderStyle.Render("│") + "\n")
	}

	b.WriteString(borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", leftW)) + borderStyle.Render("┴") + divColor.Render(strings.Repeat("─", rightW)) + divColor.Render("╯"))

	return b.String()
}

func (m Model) renderSingleColumnLayout(content string, innerW int) string {
	lines := strings.Split(content, "\n")

	var b strings.Builder

	// Top border
	b.WriteString(borderStyle.Render("╭") + borderStyle.Render("─ Timeline "+strings.Repeat("─", max(innerW-11, 0))) + borderStyle.Render("╮") + "\n")

	innerPad := lipgloss.NewStyle().Width(innerW)
	for _, line := range lines {
		b.WriteString(borderStyle.Render("│") + innerPad.Render(line) + borderStyle.Render("│") + "\n")
	}

	// Search input
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(innerW-lipgloss.Width(searchLine), 0)
		b.WriteString(borderStyle.Render("│") + searchLine + strings.Repeat(" ", searchPad) + borderStyle.Render("│") + "\n")
	}

	b.WriteString(borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", innerW)) + borderStyle.Render("╯"))

	return b.String()
}

func (m Model) rightPaneTitle() string {
	if m.isSelectMode() && len(m.words) > 0 {
		selCount := m.selectedWordCount()
		return fmt.Sprintf(" Select Words %d/%d ", selCount, len(m.words))
	}
	return " Transcript "
}
