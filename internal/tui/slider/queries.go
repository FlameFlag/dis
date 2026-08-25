package slider

import (
	"fmt"
	"math"
	"unicode"

	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/timecode"
)

func (m Model) isSelectMode() bool {
	return m.mode == modeSelect || m.mode == modeSearchSelect
}

func (m Model) isSearchMode() bool {
	return m.mode == modeSearch || m.mode == modeSearchSelect
}

func (m Model) isTwoPane() bool {
	return m.width >= MinTwoPaneWidth && m.transcript != nil
}

func (m Model) isStacked() bool {
	return !m.isTwoPane() && m.transcript != nil
}

func (m Model) leftPaneWidth() int {
	if m.isTwoPane() {
		return m.width * LeftPaneRatio / percentageBase
	}
	return m.width - singlePaneBorderCells
}

func (m Model) rightPaneWidth() int {
	if !m.isTwoPane() {
		return 0
	}
	return m.width - m.leftPaneWidth() - twoPaneBorderCells
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

func (m Model) activePos() float64 {
	if m.adjustingStart {
		return m.startPos
	}
	return m.endPos
}

func (m *Model) adjustValue(step float64) {
	if m.adjustingStart {
		newStart := m.startPos + step
		m.startPos = max(0, min(m.endPos-MillisecondStep, newStart))
	} else {
		newEnd := m.endPos + step
		m.endPos = max(m.startPos+MillisecondStep, min(m.duration, newEnd))
	}
	m.roundPositions()
}

func (m *Model) roundPositions() {
	m.startPos = math.Round(m.startPos*positionRoundingScale) / positionRoundingScale
	m.endPos = math.Round(m.endPos*positionRoundingScale) / positionRoundingScale
}

func (m *Model) processTimeInput() error {
	seconds, err := timecode.Parse(m.timeInput.Value())
	if err != nil {
		return err
	}

	if m.adjustingStart {
		if seconds < 0 || seconds > m.endPos-MillisecondStep {
			return fmt.Errorf("start must be before %s", timecode.FormatMillis(m.endPos))
		}
		m.startPos = seconds
	} else {
		if seconds < m.startPos+MillisecondStep || seconds > m.duration {
			return fmt.Errorf("end must be between %s and %s",
				timecode.FormatMillis(m.startPos),
				timecode.FormatMillis(m.duration))
		}
		m.endPos = seconds
	}
	m.roundPositions()
	return nil
}

func validateTimeInput(s string) error {
	for _, c := range s {
		if !unicode.IsDigit(c) && c != ':' && c != '.' {
			return fmt.Errorf("invalid character: %c", c)
		}
	}
	return nil
}
