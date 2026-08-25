package slider

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Enter):
		if m.timeInput.Value() != "" {
			if err := m.processTimeInput(); err != nil {
				m.warning = "Invalid time: " + err.Error()
				return m, expireWarning(m.warning)
			}
		}
		m.timeInput.Blur()
		m.mode = modeNormal
		return m, nil

	case key.Matches(msg, Escape):
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
