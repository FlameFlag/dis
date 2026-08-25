package slider

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() tea.View {
	if m.width < MinTerminalWidth || (m.height > 0 && m.height < MinTerminalHeight) {
		return newView(m.renderSizeWarning())
	}
	if m.helpVisible {
		return newView(m.renderFullHelpScreen() + "\n")
	}

	helpBar := m.renderHelpBar()
	helpHeight := lipgloss.Height(helpBar)

	switch {
	case m.isTwoPane():
		contentHeight := m.availableContentHeight(helpHeight, standardBorderRows)
		leftWidth := m.leftPaneWidth()
		rightWidth := m.rightPaneWidth()
		left := m.renderLeftPaneWithHeight(leftWidth, contentHeight)
		right := m.renderRightPaneWithHeight(rightWidth, contentHeight)
		body := m.renderBorderedLayout(
			left,
			leftWidth,
			right,
			rightWidth,
			helpBar,
			contentHeight,
		)
		return newView(body + "\n")

	case m.isStacked():
		contentHeight := m.availableContentHeight(helpHeight, stackedBorderRows)
		innerWidth := m.width - singlePaneBorderCells
		naturalTimeline := m.renderLeftPaneWithHeight(innerWidth, 0)
		timelineHeight := lipgloss.Height(naturalTimeline)
		transcriptHeight := 0
		if contentHeight > 0 {
			minTranscriptHeight := min(stackedMaxTranscriptHeight,
				max(contentHeight/stackedTranscriptDivisor, 1))
			timelineHeight = min(
				timelineHeight,
				max(contentHeight-minTranscriptHeight, 1),
			)
			transcriptHeight = max(contentHeight-timelineHeight, 1)
		}
		timeline := m.renderLeftPaneWithHeight(innerWidth, timelineHeight)
		transcript := m.renderRightPaneWithHeight(innerWidth, transcriptHeight)
		body := m.renderStackedLayout(
			timeline,
			transcript,
			innerWidth,
			timelineHeight,
			transcriptHeight,
			helpBar,
		)
		return newView(body + "\n")

	default:
		contentHeight := m.availableContentHeight(helpHeight, standardBorderRows)
		innerWidth := m.width - singlePaneBorderCells
		content := m.renderLeftPaneWithHeight(innerWidth, contentHeight)
		body := m.renderSingleColumnLayout(
			content,
			innerWidth,
			helpBar,
			contentHeight,
		)
		return newView(body + "\n")
	}
}

func newView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m Model) availableContentHeight(helpHeight, borderRows int) int {
	if m.height <= 0 {
		return 0
	}
	overhead := borderRows + helpHeight + 1 // final newline
	if m.isSearchMode() {
		overhead++
	}
	return max(m.height-overhead, 1)
}

func (m Model) renderSizeWarning() string {
	width := max(m.width, 1)
	height := max(m.height, 1)
	if height < minimumDetailedWarningRows {
		message := AccentBold.Render(ansi.Truncate("Resize terminal", width, ""))
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, message)
	}
	titleText := ansi.Truncate("dis · Trim", width, "")
	title := AccentBold.Render(titleText)
	detail := fmt.Sprintf(
		"Resize to at least %d × %d",
		MinTerminalWidth,
		MinTerminalHeight,
	)
	if width < lipgloss.Width(detail) {
		detail = ansi.Truncate("Resize terminal", width, "")
	}
	separator := "\n\n"
	if height == minimumDetailedWarningRows {
		separator = "\n"
	}
	message := title + separator + Faint.Render(detail)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, message)
}

func fitLines(content string, height int) []string {
	lines := strings.Split(content, "\n")
	if height <= 0 {
		return lines
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func renderTitledBorder(
	leftCorner string,
	rightCorner string,
	width int,
	title string,
	borderStyle lipgloss.Style,
	titleStyle lipgloss.Style,
) string {
	title = " " + title + " "
	fill := max(width-lipgloss.Width(title)-1, 0)
	return borderStyle.Render(leftCorner+"─") +
		titleStyle.Render(title) +
		borderStyle.Render(strings.Repeat("─", fill)+rightCorner)
}

func (m Model) renderBorderedLayout(
	left string,
	leftWidth int,
	right string,
	rightWidth int,
	helpBar string,
	contentHeight int,
) string {
	leftLines := fitLines(left, contentHeight)
	rightLines := fitLines(right, contentHeight)
	maxHeight := max(len(leftLines), len(rightLines))
	for len(leftLines) < maxHeight {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxHeight {
		rightLines = append(rightLines, "")
	}

	rightBorder := Border
	if m.isSelectMode() || m.isSearchMode() {
		rightBorder = Accent
	}

	leftTitle := m.leftPaneTitle()
	rightTitle := m.rightPaneTitle()
	leftTitleWidth := lipgloss.Width(" " + leftTitle + " ")
	rightTitleWidth := lipgloss.Width(" " + rightTitle + " ")
	topLeftFill := max(leftWidth-leftTitleWidth-1, 0)
	topRightFill := max(rightWidth-rightTitleWidth-1, 0)

	var b strings.Builder
	b.WriteString(Border.Render("┌─") +
		AccentBold.Render(" "+leftTitle+" ") +
		Border.Render(strings.Repeat("─", topLeftFill)+"┬─") +
		rightBorder.Render(" "+rightTitle+" ") +
		rightBorder.Render(strings.Repeat("─", topRightFill)+"┐") + "\n")

	leftPad := lipgloss.NewStyle().Width(leftWidth)
	rightPad := lipgloss.NewStyle().Width(rightWidth)
	for i := range maxHeight {
		b.WriteString(Border.Render("│") +
			leftPad.Render(leftLines[i]) +
			rightBorder.Render("│") +
			rightPad.Render(rightLines[i]) +
			rightBorder.Render("│") + "\n")
	}

	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(m.width-singlePaneBorderCells-lipgloss.Width(searchLine), 0)
		b.WriteString(Border.Render("│") + searchLine +
			strings.Repeat(" ", searchPad) + Border.Render("│") + "\n")
	}

	b.WriteString(Border.Render("└"+strings.Repeat("─", leftWidth)+"┴"+
		strings.Repeat("─", rightWidth)+"┘") + "\n")
	b.WriteString(helpBar)
	return b.String()
}

func (m Model) renderStackedLayout(
	timeline string,
	transcript string,
	innerWidth int,
	timelineHeight int,
	transcriptHeight int,
	helpBar string,
) string {
	timelineLines := fitLines(timeline, timelineHeight)
	transcriptLines := fitLines(transcript, transcriptHeight)
	innerPad := lipgloss.NewStyle().Width(innerWidth)
	rightBorder := Border
	if m.isSelectMode() || m.isSearchMode() {
		rightBorder = Accent
	}

	var b strings.Builder
	b.WriteString(renderTitledBorder(
		"┌",
		"┐",
		innerWidth,
		m.leftPaneTitle(),
		Border,
		AccentBold,
	) + "\n")
	for _, line := range timelineLines {
		b.WriteString(Border.Render("│") + innerPad.Render(line) +
			Border.Render("│") + "\n")
	}
	b.WriteString(renderTitledBorder(
		"├",
		"┤",
		innerWidth,
		m.rightPaneTitle(),
		rightBorder,
		rightBorder,
	) + "\n")
	for _, line := range transcriptLines {
		b.WriteString(rightBorder.Render("│") + innerPad.Render(line) +
			rightBorder.Render("│") + "\n")
	}

	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(innerWidth-lipgloss.Width(searchLine), 0)
		b.WriteString(rightBorder.Render("│") + searchLine +
			strings.Repeat(" ", searchPad) + rightBorder.Render("│") + "\n")
	}

	b.WriteString(rightBorder.Render("└"+strings.Repeat("─", innerWidth)+"┘") + "\n")
	b.WriteString(helpBar)
	return b.String()
}

func (m Model) renderSingleColumnLayout(
	content string,
	innerWidth int,
	helpBar string,
	contentHeight int,
) string {
	lines := fitLines(content, contentHeight)
	innerPad := lipgloss.NewStyle().Width(innerWidth)

	var b strings.Builder
	b.WriteString(renderTitledBorder(
		"┌",
		"┐",
		innerWidth,
		m.leftPaneTitle(),
		Border,
		AccentBold,
	) + "\n")
	for _, line := range lines {
		b.WriteString(Border.Render("│") + innerPad.Render(line) +
			Border.Render("│") + "\n")
	}

	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := max(innerWidth-lipgloss.Width(searchLine), 0)
		b.WriteString(Border.Render("│") + searchLine +
			strings.Repeat(" ", searchPad) + Border.Render("│") + "\n")
	}

	b.WriteString(Border.Render("└"+strings.Repeat("─", innerWidth)+"┘") + "\n")
	b.WriteString(helpBar)
	return b.String()
}

func (m Model) leftPaneTitle() string {
	activeEdge := "Start edge"
	if !m.adjustingStart {
		activeEdge = "End edge"
	}
	switch {
	case m.mode == modeInput:
		return "Trim · Enter " + strings.ToLower(activeEdge)
	case m.isSelectMode():
		return "Trim · Selection preview"
	default:
		return "Trim · " + activeEdge
	}
}

func (m Model) rightPaneTitle() string {
	switch {
	case m.mode == modeSearchSelect:
		return "Words · Search"
	case m.isSelectMode() && len(m.words) > 0:
		return fmt.Sprintf("Words · %d selected", m.selectedWordCount())
	case m.mode == modeSearch:
		return "Transcript · Search"
	case !m.viewportLocked:
		return "Transcript · Browsing"
	default:
		return "Transcript · Following"
	}
}
