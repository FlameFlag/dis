package slider

import (
	"dis/internal/config"
	"dis/internal/subtitle"
	"dis/internal/tui/slider/keys"
	"dis/internal/util"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

func (m Model) handleSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.sel.anchor = -1
		m.search.results = nil
		m.updateSliderFromSelection()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Enter):
		m.mode = modeNormal
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		if m.sel.cursor > 0 {
			m.sel.cursor--
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Right):
		if m.sel.cursor < len(m.words)-1 {
			m.sel.cursor++
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.ShiftLeft):
		if m.sel.cursor > 0 {
			if m.sel.anchor < 0 {
				m.sel.anchor = m.sel.cursor
				m.sel.anchorSelecting = !m.sel.selected[m.sel.cursor]
			}
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
			m.sel.cursor--
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.ShiftRight):
		if m.sel.cursor < len(m.words)-1 {
			if m.sel.anchor < 0 {
				m.sel.anchor = m.sel.cursor
				m.sel.anchorSelecting = !m.sel.selected[m.sel.cursor]
			}
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
			m.sel.cursor++
			m.sel.selected[m.sel.cursor] = m.sel.anchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		// Jump to first word of previous cue
		if m.sel.cursor > 0 {
			curCue := m.words[m.sel.cursor].CueIndex
			i := m.sel.cursor - 1
			// Skip remaining words in current cue
			for i > 0 && m.words[i].CueIndex == curCue {
				i--
			}
			// Find first word of that cue
			prevCue := m.words[i].CueIndex
			for i > 0 && m.words[i-1].CueIndex == prevCue {
				i--
			}
			m.sel.cursor = i
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		// Jump to first word of next cue
		if m.sel.cursor < len(m.words)-1 {
			curCue := m.words[m.sel.cursor].CueIndex
			i := m.sel.cursor + 1
			for i < len(m.words) && m.words[i].CueIndex == curCue {
				i++
			}
			if i < len(m.words) {
				m.sel.cursor = i
			}
		}
		return m, nil

	case key.Matches(msg, keys.Space):
		m.sel.selected[m.sel.cursor] = !m.sel.selected[m.sel.cursor]
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.ParagraphSelect):
		m.toggleParagraph()
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.SelectTrimRange):
		if len(m.splits) > 0 {
			m.selectWordsInRanges(m.splits)
		} else {
			m.selectWordsInRanges([]trimRange{{start: m.startPos, end: m.endPos}})
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Deselect):
		for i := range m.sel.selected {
			m.sel.selected[i] = false
		}
		m.sel.anchor = -1
		return m, nil

	case key.Matches(msg, keys.Search):
		m.mode = modeSearchSelect
		m.search.input.Reset()
		m.search.results = nil
		m.search.index = 0
		return m, m.search.input.Focus()

	case key.Matches(msg, keys.NextMatch):
		if len(m.search.results) > 0 {
			m.search.index = (m.search.index + 1) % len(m.search.results)
			m.sel.cursor = m.search.results[m.search.index]
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if len(m.search.results) > 0 {
			m.search.index = (m.search.index - 1 + len(m.search.results)) % len(m.search.results)
			m.sel.cursor = m.search.results[m.search.index]
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

// isSentenceEnd returns true if the word ends with sentence-ending punctuation.
func isSentenceEnd(word string) bool {
	trimmed := strings.TrimRight(word, `"')]}`)
	return strings.HasSuffix(trimmed, ".") ||
		strings.HasSuffix(trimmed, "?") ||
		strings.HasSuffix(trimmed, "!")
}

// toggleParagraph selects/deselects a sentence around the cursor, crossing cue boundaries.
func (m *Model) toggleParagraph() {
	if len(m.words) == 0 {
		return
	}

	// Search backward across all words for sentence-ending punctuation
	lo := 0
	for i := m.sel.cursor - 1; i >= 0; i-- {
		if isSentenceEnd(m.words[i].Text) {
			lo = i + 1
			break
		}
	}

	// Search forward across all words for sentence-ending punctuation
	hi := len(m.words) - 1
	for i := m.sel.cursor; i < len(m.words); i++ {
		if isSentenceEnd(m.words[i].Text) {
			hi = i
			break
		}
	}

	if lo > hi {
		return
	}

	// Majority toggle
	sel := 0
	for i := lo; i <= hi; i++ {
		if m.sel.selected[i] {
			sel++
		}
	}
	newState := sel <= (hi-lo+1)/2
	for i := lo; i <= hi; i++ {
		m.sel.selected[i] = newState
	}
}

func (m *Model) updateSliderFromSelection() {
	segs := m.selectedSegments()
	if len(segs) > 0 {
		m.startPos = segs[0].Start
		m.endPos = segs[len(segs)-1].End()
		m.roundPositions()
	}
}

// selectedSegments derives contiguous time ranges from the word selection.
func (m Model) selectedSegments() []config.TrimSettings {
	if len(m.words) == 0 {
		return nil
	}

	var segments []config.TrimSettings
	inSegment := false
	var segStart float64

	for i, sel := range m.sel.selected {
		if sel && !inSegment {
			segStart = m.words[i].Start
			inSegment = true
		} else if !sel && inSegment {
			segments = append(segments, config.TrimSettings{
				Start:    segStart,
				Duration: m.words[i-1].End - segStart,
			})
			inSegment = false
		}
	}
	if inSegment {
		lastIdx := len(m.words) - 1
		for lastIdx >= 0 && !m.sel.selected[lastIdx] {
			lastIdx--
		}
		if lastIdx >= 0 {
			segments = append(segments, config.TrimSettings{
				Start:    segStart,
				Duration: m.words[lastIdx].End - segStart,
			})
		}
	}

	for i, seg := range segments {
		log.Debug("Word selection segment",
			"idx", i, "start", seg.Start, "end", seg.End(),
			"duration", seg.Duration, "section", seg.DownloadSection())
	}

	return segments
}

// selectedWordCount returns the number of selected words.
func (m Model) selectedWordCount() int {
	n := 0
	for _, s := range m.sel.selected {
		if s {
			n++
		}
	}
	return n
}

func (m Model) hasWordSelection() bool {
	return slices.Contains(m.sel.selected, true)
}

func (m Model) nearestWordIndex(seconds float64) int {
	return util.NearestIndex(m.words, seconds, func(w subtitle.Word) float64 { return w.Start })
}

// selectWordsInRanges pre-selects words that overlap any of the given time ranges.
func (m *Model) selectWordsInRanges(ranges []trimRange) {
	for i, word := range m.words {
		for _, r := range ranges {
			if word.Start < r.end && word.End > r.start {
				m.sel.selected[i] = true
				break
			}
		}
	}
}
