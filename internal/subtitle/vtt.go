package subtitle

import (
	"regexp"
	"strings"
)

// vttTimestampRe captures HH (possibly empty), MM, SS, mmm. The hours group
// uses a non-capturing colon so the captured field is just the digits, same
// shape as srtTimestampRe so both can share parseTimestamp.
var vttTimestampRe = regexp.MustCompile(`(?:(\d{1,2}):)?(\d{2}):(\d{2})\.(\d{3})`)

var vttFormat = cueFormat{
	label:       "VTT",
	timestampRe: vttTimestampRe,
	splitEnd:    true,
	withWords:   true,
}

// ParseVTT parses a WebVTT subtitle string into a Transcript.
func ParseVTT(data string) (Transcript, error) {
	transcript, err := parseCues(data, vttFormat)
	if err != nil {
		return nil, err
	}
	return deduplicateVTTCues(transcript), nil
}

// deduplicateVTTCues removes overlapping YouTube auto-caption cues.
// YouTube often emits rolling cues where each new cue repeats the previous text
// plus adds new words. We keep only the cue with the most text for each time range.
func deduplicateVTTCues(cues Transcript) Transcript {
	if len(cues) <= 1 {
		return cues
	}

	var result Transcript
	for i := range cues {
		// If the next cue starts at the same time or overlaps significantly
		// and contains this cue's text, skip this cue
		if i+1 < len(cues) {
			next := cues[i+1]
			curr := cues[i]
			if next.Start <= curr.Start+0.1 && strings.Contains(next.Text, curr.Text) {
				continue
			}
		}
		result = append(result, cues[i])
	}
	return result
}
