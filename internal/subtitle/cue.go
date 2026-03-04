package subtitle

import (
	"dis/internal/util"
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	arrowRe = regexp.MustCompile(`\s*-->\s*`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)
)

func stripTags(s string) string {
	return html.UnescapeString(tagRe.ReplaceAllString(s, ""))
}

// Cue is a single subtitle entry with timing and plain text.
type Cue struct {
	Start float64
	End   float64
	Text  string // plain text, HTML tags stripped

	// WordTimings holds per-word timestamps extracted from VTT <c> tags.
	// nil if word-level timing is not available for this cue.
	WordTimings []WordTiming
}

// WordTiming is a word with an exact timestamp from VTT <c> tags.
type WordTiming struct {
	Text  string
	Start float64
}

// Transcript is an ordered list of cues.
type Transcript []Cue

// Word is a single word with timing derived from its parent cue.
type Word struct {
	Text     string
	Start    float64
	End      float64
	CueIndex int
}

// CueAt returns the index of the cue containing the given time, or -1.
func (t Transcript) CueAt(seconds float64) int {
	for i, c := range t {
		if seconds >= c.Start && seconds < c.End {
			return i
		}
	}
	return -1
}

// NearestCue returns the index of the cue closest to the given time.
func (t Transcript) NearestCue(seconds float64) int {
	if len(t) == 0 {
		return -1
	}
	return util.NearestIndex(t, seconds, func(c Cue) float64 { return c.Start })
}

// NextCueStart returns the start time of the next cue after the given time.
// Returns -1 if there is no next cue.
func (t Transcript) NextCueStart(after float64) float64 {
	for _, c := range t {
		if c.Start > after+0.001 {
			return c.Start
		}
	}
	return -1
}

// PrevCueStart returns the start time of the previous cue before the given time.
// Returns -1 if there is no previous cue.
func (t Transcript) PrevCueStart(before float64) float64 {
	result := -1.0
	for _, c := range t {
		if c.Start < before-0.001 {
			result = c.Start
		} else {
			break
		}
	}
	return result
}

// Search returns indices of cues whose text contains the query (case-insensitive).
func (t Transcript) Search(query string) []int {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var results []int
	for i, c := range t {
		if strings.Contains(strings.ToLower(c.Text), q) {
			results = append(results, i)
		}
	}
	return results
}

// Words flattens all cues into a slice of individually-timed words.
// Timing comes from VTT <c> word-level timestamps when available,
// otherwise linearly interpolated within the cue.
func (t Transcript) Words() []Word {
	var words []Word
	for i, c := range t {
		cueWords := splitWords(c.Text)
		if len(cueWords) == 0 {
			continue
		}

		if len(c.WordTimings) > 0 {
			// Use exact word-level timing from VTT <c> tags
			words = append(words, wordsFromTimings(c, i)...)
		} else {
			// Linear interpolation
			words = append(words, wordsInterpolated(c, cueWords, i)...)
		}
	}
	return words
}

// SearchWords returns indices into the word slice where the query matches (case-insensitive).
func (t Transcript) SearchWords(words []Word, query string) []int {
	if query == "" || len(words) == 0 {
		return nil
	}
	q := strings.ToLower(query)
	var results []int
	for i, w := range words {
		if strings.Contains(strings.ToLower(w.Text), q) {
			results = append(results, i)
		}
	}
	return results
}

func wordsFromTimings(c Cue, cueIndex int) []Word {
	var words []Word
	for j, wt := range c.WordTimings {
		end := c.End
		if j+1 < len(c.WordTimings) {
			end = c.WordTimings[j+1].Start
		}
		text := strings.TrimSpace(wt.Text)
		if text == "" {
			continue
		}

		// Split multi-word entries (e.g. VTT prefix text before first <c> tag)
		subwords := splitWords(text)
		if len(subwords) > 1 {
			subDur := (end - wt.Start) / float64(len(subwords))
			for k, sw := range subwords {
				words = append(words, Word{
					Text:     sw,
					Start:    wt.Start + float64(k)*subDur,
					End:      wt.Start + float64(k+1)*subDur,
					CueIndex: cueIndex,
				})
			}
		} else {
			words = append(words, Word{
				Text:     text,
				Start:    wt.Start,
				End:      end,
				CueIndex: cueIndex,
			})
		}
	}
	return words
}

func wordsInterpolated(c Cue, texts []string, cueIndex int) []Word {
	n := len(texts)
	cueDur := c.End - c.Start
	wordDur := cueDur / float64(n)

	words := make([]Word, 0, n)
	for j, text := range texts {
		words = append(words, Word{
			Text:     text,
			Start:    c.Start + float64(j)*wordDur,
			End:      c.Start + float64(j+1)*wordDur,
			CueIndex: cueIndex,
		})
	}
	return words
}

func splitWords(text string) []string {
	var words []string
	for _, w := range strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r)
	}) {
		w = strings.TrimSpace(w)
		if w != "" {
			words = append(words, w)
		}
	}
	return words
}
