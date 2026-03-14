package slider

import (
	"dis/internal/config"
	"dis/internal/subtitle"
	"dis/internal/util"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
)

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
	for i := m.cursor - 1; i >= 0; i-- {
		if isSentenceEnd(m.words[i].Text) {
			lo = i + 1
			break
		}
	}

	// Search forward across all words for sentence-ending punctuation
	hi := len(m.words) - 1
	for i := m.cursor; i < len(m.words); i++ {
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
		if m.selected[i] {
			sel++
		}
	}
	newState := sel <= (hi-lo+1)/2
	for i := lo; i <= hi; i++ {
		m.selected[i] = newState
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

	for i, sel := range m.selected {
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
		for lastIdx >= 0 && !m.selected[lastIdx] {
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
	for _, s := range m.selected {
		if s {
			n++
		}
	}
	return n
}

func (m Model) hasWordSelection() bool {
	return slices.Contains(m.selected, true)
}

func (m Model) nearestWordIndex(seconds float64) int {
	return util.NearestIndex(m.words, seconds, func(w subtitle.Word) float64 { return w.Start })
}

// selectWordsInRanges pre-selects words that overlap any of the given time ranges.
func (m *Model) selectWordsInRanges(ranges []trimRange) {
	for i, word := range m.words {
		for _, r := range ranges {
			if word.Start < r.end && word.End > r.start {
				m.selected[i] = true
				break
			}
		}
	}
}
