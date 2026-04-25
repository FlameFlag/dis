package subtitle

import (
	"regexp"
	"strings"
)

var vttCTagRe = regexp.MustCompile(`<(\d{2}:\d{2}:\d{2}\.\d{3})>`)

// extractWordTimings extracts <c> tag timestamps from VTT cue text.
// Returns nil if no <c> tags are found.
func extractWordTimings(rawText string, cueStart float64) []WordTiming {
	matches := vttCTagRe.FindAllStringSubmatchIndex(rawText, -1)
	if len(matches) == 0 {
		return nil
	}

	var timings []WordTiming

	// Text before the first <c> tag belongs to cueStart
	firstTagStart := matches[0][0]
	prefix := stripTags(rawText[:firstTagStart])
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		timings = append(timings, WordTiming{
			Text:  prefix,
			Start: cueStart,
		})
	}

	for i, match := range matches {
		// match[2]:match[3] is the timestamp capture group
		ts, err := parseVTTTimestamp(rawText[match[2]:match[3]])
		if err != nil {
			continue
		}

		// Text runs from after this tag to the start of the next tag (or end)
		textStart := match[1] // end of the <timestamp> tag
		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1][0]
		} else {
			textEnd = len(rawText)
		}

		text := stripTags(rawText[textStart:textEnd])
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		timings = append(timings, WordTiming{
			Text:  text,
			Start: ts,
		})
	}

	if len(timings) == 0 {
		return nil
	}
	return timings
}
