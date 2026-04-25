package slider

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		if m.timeInput.Value() != "" {
			m.processTimeInput()
		}
		m.timeInput.Blur()
		m.mode = modeNormal
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.timeInput.Blur()
		m.timeInput.Reset()
		m.mode = modeNormal
		return m, nil

	default:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	}
}
