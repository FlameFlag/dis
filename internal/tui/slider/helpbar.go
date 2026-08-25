package slider

import (
	"fmt"
	"strings"

	"github.com/4evy/dis/internal/tui"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

type shortcutGroup struct {
	title    string
	bindings []key.Binding
}

func displayBinding(keyText, description string) key.Binding {
	return key.NewBinding(
		key.WithKeys(keyText),
		key.WithHelp(keyText, description),
	)
}

func (m Model) renderHelpBar() string {
	bindings := m.shortHelpBindings()
	help := m.keyHelp
	help.SetWidth(max(m.helpBarWidth()-1, 0))
	return " " + help.ShortHelpView(bindings)
}

func (m Model) shortHelpBindings() []key.Binding {
	switch {
	case m.isSearchMode():
		return []key.Binding{
			displayBinding("type", "search"),
			displayBinding("enter", "go to match"),
			displayBinding("esc", "cancel"),
		}
	case m.mode == modeInput:
		return []key.Binding{
			displayBinding("type", "set time"),
			displayBinding("enter", "accept"),
			displayBinding("esc", "cancel"),
		}
	case m.isSelectMode():
		return []key.Binding{
			Help,
			displayBinding("←/→", "move by word"),
			displayBinding("space", "toggle word"),
			displayBinding("enter", "apply selection"),
			displayBinding("esc", "back to range"),
		}
	default:
		bindings := []key.Binding{
			Help,
			Left,
			Tab,
			Enter,
			Split,
		}
		if len(m.words) > 0 {
			bindings = append(bindings, TranscriptSelect)
		}
		return bindings
	}
}

// helpBarWidth returns the available inner width for the help bar.
func (m Model) helpBarWidth() int {
	if m.isTwoPane() {
		return m.leftPaneWidth() + m.rightPaneWidth() + 1
	}
	return m.width - singlePaneBorderCells
}

func (m Model) fullHelpGroups() []shortcutGroup {
	if m.isSelectMode() {
		moveBindings := []key.Binding{
			displayBinding("←/→", "previous/next word"),
			displayBinding("↑/↓", "previous/next cue"),
			displayBinding("shift+←/→", "extend selection"),
		}
		if len(m.search.results) > 0 {
			moveBindings = append(moveBindings, NextMatch)
		}
		return []shortcutGroup{
			{
				title:    "Move",
				bindings: moveBindings,
			},
			{
				title: "Select",
				bindings: []key.Binding{
					displayBinding("space", "toggle word"),
					ParagraphSelect,
					SelectTrimRange,
					Deselect,
					Search,
				},
			},
			{
				title: "Finish",
				bindings: []key.Binding{
					displayBinding("enter", "apply selection"),
					displayBinding("esc", "back to range"),
					Quit,
					Cancel,
				},
			},
		}
	}

	navigation := []key.Binding{
		SelectStart,
		SelectEnd,
		Tab,
		Left,
		ShiftLeft,
		Up,
	}
	transcript := []key.Binding{
		NextCue,
		PageUp,
		Search,
	}
	if len(m.search.results) > 0 {
		transcript = append(transcript, NextMatch)
	}
	if len(m.words) > 0 {
		transcript = append(transcript, TranscriptSelect)
	}
	actions := []key.Binding{
		Space,
		Split,
	}
	if len(m.splits) > 0 {
		actions = append(actions, DeleteSplit)
	}
	actions = append(actions,
		GIFToggle,
		SpeedToggle,
		Enter,
		Escape,
		Quit,
		Cancel,
	)

	groups := []shortcutGroup{{title: "Range", bindings: navigation}}
	if m.transcript != nil {
		groups = append(groups, shortcutGroup{title: "Transcript", bindings: transcript})
	}
	groups = append(groups, shortcutGroup{title: "Actions", bindings: actions})
	return groups
}

func (m Model) helpContext() string {
	if m.isSelectMode() {
		return "Word selection"
	}
	return "Range editing"
}

func (m Model) renderShortcutGroup(group shortcutGroup) string {
	help := m.keyHelp
	help.SetWidth(0)
	return Bold.Render(group.title) + "\n" +
		help.FullHelpView([][]key.Binding{group.bindings})
}

func (m Model) renderHelpGroups(maxWidth int) string {
	groups := m.fullHelpGroups()
	columns := make([]string, len(groups))
	for i, group := range groups {
		columns[i] = lipgloss.NewStyle().
			MarginRight(helpColumnSpacing).
			Render(m.renderShortcutGroup(group))
	}
	if row := lipgloss.JoinHorizontal(lipgloss.Top, columns...); lipgloss.Width(row) <= maxWidth {
		return row
	}
	for i, group := range groups {
		columns[i] = m.renderShortcutGroup(group)
	}
	return strings.Join(columns, "\n\n")
}

func (m Model) helpViewport() (content string, viewportHeight int) {
	innerWidth := max(m.width-helpHorizontalInset, helpMinimumWidth)
	content = m.renderHelpGroups(innerWidth)
	viewportHeight = max(m.height-helpVerticalInset, 1)
	return content, viewportHeight
}

func (m Model) helpMaxOffset() int {
	content, viewportHeight := m.helpViewport()
	return max(lipgloss.Height(content)-viewportHeight, 0)
}

func (m Model) renderFullHelpScreen() string {
	content, viewportHeight := m.helpViewport()
	lines := strings.Split(content, "\n")
	maxOffset := max(len(lines)-viewportHeight, 0)
	offset := min(m.helpScroll, maxOffset)
	end := min(offset+viewportHeight, len(lines))
	visible := strings.Join(lines[offset:end], "\n")

	position := ""
	if maxOffset > 0 {
		position = fmt.Sprintf(" · %d–%d of %d", offset+1, end, len(lines))
	}
	footer := HelpDesc.Render("↑/↓ scroll · ?/esc close" + position)
	body := AccentBold.Render("Keyboard shortcuts") + "\n" +
		Faint.Render(m.helpContext()) + "\n\n" + visible + "\n\n" + footer
	panel := Modal.Width(max(m.width-helpHorizontalInset, helpMinimumWidth)).Render(body)
	background := lipgloss.NewStyle().Background(tui.ColorBase)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		panel,
		lipgloss.WithWhitespaceStyle(background),
	)
}
