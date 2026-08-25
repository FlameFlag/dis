package subtitle

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	timestampHoursCapture = iota + 1
	timestampMinutesCapture
	timestampSecondsCapture
	timestampMillisCapture
)

// cueFormat captures the differences between subtitle formats consumed by the
// shared parser in parseCues, only timestamp shape and a couple of small
// VTT-specific behaviours actually vary.
type cueFormat struct {
	label       string         // shown in error messages
	timestampRe *regexp.Regexp // matches a single timestamp; groups: HH MM SS mmm (HH may be empty)
	splitEnd    bool           // VTT may carry positioning info after the end timestamp; take the first field
	withWords   bool           // run extractWordTimings on the raw cue text
}

// parseCues drives the cue state machine that VTT and SRT share. The format
// argument supplies the format-specific timestamp regex and a couple of small
// behavioural switches.
func parseCues(data string, f cueFormat) (Transcript, error) {
	type state int
	const (
		stateSeeking state = iota
		stateCueText
	)

	var (
		transcript Transcript
		current    = stateSeeking
		cueStart   float64
		cueEnd     float64
		textLines  []string
		firstLine  = true
	)

	flushCue := func() {
		if len(textLines) == 0 {
			return
		}
		rawText := strings.Join(textLines, " ")
		var wordTimings []WordTiming
		if f.withWords {
			wordTimings = extractWordTimings(rawText, cueStart)
		}
		plainText := strings.TrimSpace(stripTags(rawText))
		textLines = nil
		if plainText == "" {
			return
		}
		transcript = append(transcript, Cue{
			Start:       cueStart,
			End:         cueEnd,
			Text:        plainText,
			WordTimings: wordTimings,
		})
	}

	for line := range strings.Lines(data) {
		line = strings.TrimRight(line, "\r\n")
		if firstLine {
			line = strings.TrimPrefix(line, "\ufeff")
			firstLine = false
		}

		switch current {
		case stateSeeking:
			line = strings.TrimSpace(line)
			if line == "" || !strings.Contains(line, "-->") {
				// Skip blanks, sequence numbers, cue identifiers, NOTE blocks.
				continue
			}
			flushCue()
			start, end, err := parseTimestampLine(line, f)
			if err != nil {
				continue
			}
			cueStart, cueEnd = start, end
			current = stateCueText

		case stateCueText:
			if strings.TrimSpace(line) == "" {
				flushCue()
				current = stateSeeking
				continue
			}
			textLines = append(textLines, line)
		}
	}

	flushCue()

	if len(transcript) == 0 {
		return nil, fmt.Errorf("no cues found in %s data", f.label)
	}
	return transcript, nil
}

func parseTimestampLine(line string, f cueFormat) (float64, float64, error) {
	startText, endText, ok := strings.Cut(line, "-->")
	if !ok {
		return 0, 0, errors.New("invalid timestamp line")
	}

	start, err := parseTimestamp(strings.TrimSpace(startText), f.timestampRe)
	if err != nil {
		return 0, 0, err
	}

	endStr := strings.TrimSpace(endText)
	if f.splitEnd {
		fields := strings.Fields(endStr)
		if len(fields) == 0 {
			return 0, 0, errors.New("missing end timestamp")
		}
		endStr = fields[0]
	}

	end, err := parseTimestamp(endStr, f.timestampRe)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

// parseTimestamp expects a regex that captures HH (possibly empty), MM, SS, mmm.
func parseTimestamp(ts string, re *regexp.Regexp) (float64, error) {
	m := re.FindStringSubmatch(ts)
	if m == nil {
		return 0, fmt.Errorf("invalid timestamp: %s", ts)
	}
	return hmsToSeconds(
		m[timestampHoursCapture],
		m[timestampMinutesCapture],
		m[timestampSecondsCapture],
		m[timestampMillisCapture],
	)
}
