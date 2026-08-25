package subtitle

import (
	"html"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const cueTimeEpsilon = 0.001

var tagRe = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return html.UnescapeString(tagRe.ReplaceAllString(s, ""))
}

// hmsToSeconds converts numeric HH/MM/SS/mmm strings (milliseconds 0–999)
// into seconds. Empty hours is treated as 0.
func hmsToSeconds(h, m, s, ms string) (float64, error) {
	var hours int
	if h != "" {
		v, err := strconv.Atoi(h)
		if err != nil {
			return 0, err
		}
		hours = v
	}
	minutes, err := strconv.Atoi(m)
	if err != nil {
		return 0, err
	}
	seconds, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	millis, err := strconv.Atoi(ms)
	if err != nil {
		return 0, err
	}
	return float64(hours)*time.Hour.Seconds() +
		float64(minutes)*time.Minute.Seconds() +
		float64(seconds) +
		float64(millis)/millisecondsPerSecond, nil
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
	return slices.IndexFunc(t, func(c Cue) bool {
		return seconds >= c.Start && seconds < c.End
	})
}

// NearestCue returns the index of the cue closest to the given time.
func (t Transcript) NearestCue(seconds float64) int {
	if len(t) == 0 {
		return -1
	}
	index := sort.Search(len(t), func(index int) bool {
		return t[index].Start >= seconds
	})
	if index == 0 {
		return 0
	}
	if index == len(t) {
		return len(t) - 1
	}
	if seconds-t[index-1].Start <= t[index].Start-seconds {
		return index - 1
	}
	return index
}

// NextCueStart returns the start time of the next cue after the given time.
// Returns -1 if there is no next cue.
func (t Transcript) NextCueStart(after float64) float64 {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Start > after+cueTimeEpsilon
	})
	if i < len(t) {
		return t[i].Start
	}
	return -1
}

// PrevCueStart returns the start time of the previous cue before the given time.
// Returns -1 if there is no previous cue.
func (t Transcript) PrevCueStart(before float64) float64 {
	i := sort.Search(len(t), func(i int) bool {
		return t[i].Start >= before-cueTimeEpsilon
	})
	if i > 0 {
		return t[i-1].Start
	}
	return -1
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
	return strings.Fields(text)
}
