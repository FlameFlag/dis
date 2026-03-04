package subtitle

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type json3File struct {
	Events []json3Event `json:"events"`
}

type json3Event struct {
	TStartMs    int        `json:"tStartMs"`
	DDurationMs int        `json:"dDurationMs"`
	Segs        []json3Seg `json:"segs"`
	AAppend     int        `json:"aAppend"`
	WpWinPosId  *int       `json:"wpWinPosId"`
}

type json3Seg struct {
	Utf8      string `json:"utf8"`
	TOffsetMs int    `json:"tOffsetMs"`
	AcAsrConf *int   `json:"acAsrConf"`
}

var bracketRe = regexp.MustCompile(`^\[.+\]$`)

// ParseJSON3 parses YouTube's json3 subtitle format into a Transcript.
// json3 provides per-word millisecond timing via events[].segs[].
func ParseJSON3(data string) (Transcript, error) {
	var f json3File
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		return nil, fmt.Errorf("parsing json3: %w", err)
	}

	var transcript Transcript
	for _, ev := range f.Events {
		// Skip newline markers and window setup events
		if ev.AAppend == 1 || ev.WpWinPosId != nil {
			continue
		}
		if len(ev.Segs) == 0 {
			continue
		}

		// Build concatenated text and word timings
		var textParts []string
		var timings []WordTiming
		for _, seg := range ev.Segs {
			text := strings.TrimSpace(seg.Utf8)
			if text == "" {
				continue
			}
			textParts = append(textParts, text)
			timings = append(timings, WordTiming{
				Text:  text,
				Start: float64(ev.TStartMs+seg.TOffsetMs) / 1000.0,
			})
		}

		fullText := strings.Join(textParts, " ")
		if fullText == "" {
			continue
		}

		// Skip music/applause markers like [Music], [Applause]
		if bracketRe.MatchString(strings.TrimSpace(fullText)) {
			continue
		}

		cue := Cue{
			Start:       float64(ev.TStartMs) / 1000.0,
			End:         float64(ev.TStartMs+ev.DDurationMs) / 1000.0,
			Text:        fullText,
			WordTimings: timings,
		}
		transcript = append(transcript, cue)
	}

	if len(transcript) == 0 {
		return nil, fmt.Errorf("json3: no cues found")
	}

	return transcript, nil
}
