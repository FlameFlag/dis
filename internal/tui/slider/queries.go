package slider

import (
	"dis/internal/sponsorblock"
	"dis/internal/util"
	"fmt"
	"math"
	"unicode"
)

func (m Model) isSelectMode() bool {
	return m.mode == modeSelect || m.mode == modeSearchSelect
}

func (m Model) isSearchMode() bool {
	return m.mode == modeSearch || m.mode == modeSearchSelect
}

func (m Model) sliderWidth() int {
	w := max(m.leftPaneWidth()-2, MinSliderWidth) // 2 for inner padding
	return w
}

func (m Model) isTwoPane() bool {
	return m.width >= MinTwoPaneWidth && m.transcript != nil
}

func (m Model) leftPaneWidth() int {
	if m.isTwoPane() {
		return m.width * LeftPaneRatio / 100
	}
	return m.width - 2 // single column: 1 border each side
}

func (m Model) rightPaneWidth() int {
	if !m.isTwoPane() {
		return 0
	}
	return m.width - m.leftPaneWidth() - 3 // 3 for border chars (│ left border + │ divider + │ right border)
}

// sponsorCategoryAt returns the SponsorBlock category for a given timestamp, or empty string.
func (m Model) sponsorCategoryAt(seconds float64) sponsorblock.Category {
	for _, seg := range m.sponsorSegments {
		if seconds >= seg.Start && seconds < seg.End {
			return seg.Category
		}
	}
	return ""
}

func (m Model) isSilenceAt(seconds float64) bool {
	for _, si := range m.silenceIntervals {
		if seconds >= si.Start && seconds <= si.End {
			return true
		}
	}
	return false
}

func (m Model) activePos() float64 {
	if m.adjustingStart {
		return m.startPos
	}
	return m.endPos
}

func (m *Model) adjustValue(step float64) {
	if m.adjustingStart {
		newStart := m.startPos + step
		m.startPos = math.Max(0, math.Min(m.endPos-MillisecondStep, newStart))
	} else {
		newEnd := m.endPos + step
		m.endPos = math.Max(m.startPos+MillisecondStep, math.Min(m.duration, newEnd))
	}
	m.roundPositions()
}

func (m *Model) roundPositions() {
	m.startPos = math.Round(m.startPos*100) / 100
	m.endPos = math.Round(m.endPos*100) / 100
}

func (m *Model) processTimeInput() {
	seconds, err := util.ParseTimeValue(m.timeInput.Value())
	if err != nil {
		return
	}

	if m.adjustingStart {
		if seconds >= 0 && seconds <= m.endPos-MillisecondStep {
			m.startPos = seconds
		}
	} else {
		if seconds >= m.startPos+MillisecondStep && seconds <= m.duration {
			m.endPos = seconds
		}
	}
	m.roundPositions()
}

func validateTimeInput(s string) error {
	for _, c := range s {
		if !unicode.IsDigit(c) && c != ':' && c != '.' {
			return fmt.Errorf("invalid character: %c", c)
		}
	}
	return nil
}
