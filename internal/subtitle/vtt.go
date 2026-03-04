package subtitle

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	vttTimestampRe = regexp.MustCompile(`(\d{1,2}:)?(\d{2}):(\d{2})\.(\d{3})`)
	vttCTagRe      = regexp.MustCompile(`<(\d{2}:\d{2}:\d{2}\.\d{3})>`)
)

// ParseVTT parses a WebVTT subtitle string into a Transcript.
func ParseVTT(data string) (Transcript, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))

	// Skip BOM if present
	var firstLine = true

	type state int
	const (
		stateHeader state = iota
		stateSeeking
		stateCueText
	)

	var (
		transcript Transcript
		current    = stateHeader
		cueStart   float64
		cueEnd     float64
		textLines  []string
	)

	flushCue := func() {
		if len(textLines) == 0 {
			return
		}
		rawText := strings.Join(textLines, " ")
		wordTimings := extractWordTimings(rawText, cueStart)
		plainText := stripTags(rawText)
		plainText = strings.TrimSpace(plainText)
		if plainText == "" {
			textLines = nil
			return
		}
		transcript = append(transcript, Cue{
			Start:       cueStart,
			End:         cueEnd,
			Text:        plainText,
			WordTimings: wordTimings,
		})
		textLines = nil
	}

	for scanner.Scan() {
		line := scanner.Text()

		if firstLine {
			// Strip BOM
			line = strings.TrimPrefix(line, "\ufeff")
			firstLine = false
			if strings.HasPrefix(line, "WEBVTT") {
				current = stateSeeking
				continue
			}
			// Not a valid VTT file but try to parse anyway
			current = stateSeeking
		}

		switch current {
		case stateHeader:
			// Skip header lines until blank
			if strings.TrimSpace(line) == "" {
				current = stateSeeking
			}

		case stateSeeking:
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Check if this is a timestamp line
			if arrowRe.MatchString(line) {
				flushCue()
				start, end, err := parseVTTTimestampLine(line)
				if err != nil {
					continue
				}
				cueStart = start
				cueEnd = end
				current = stateCueText
				continue
			}
			// Otherwise it's a cue identifier or note, skip

		case stateCueText:
			if strings.TrimSpace(line) == "" {
				flushCue()
				current = stateSeeking
				continue
			}
			textLines = append(textLines, line)
		}
	}

	// Flush final cue
	flushCue()

	transcript = deduplicateVTTCues(transcript)

	if len(transcript) == 0 {
		return nil, fmt.Errorf("no cues found in VTT data")
	}

	return transcript, nil
}

func parseVTTTimestampLine(line string) (float64, float64, error) {
	parts := arrowRe.Split(line, 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid timestamp line")
	}

	start, err := parseVTTTimestamp(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}

	// End may have positioning info after it, take only the timestamp
	endPart := strings.TrimSpace(parts[1])
	endFields := strings.Fields(endPart)
	if len(endFields) == 0 {
		return 0, 0, fmt.Errorf("missing end timestamp")
	}

	end, err := parseVTTTimestamp(endFields[0])
	if err != nil {
		return 0, 0, err
	}

	return start, end, nil
}

func parseVTTTimestamp(ts string) (float64, error) {
	m := vttTimestampRe.FindStringSubmatch(ts)
	if m == nil {
		return 0, fmt.Errorf("invalid timestamp: %s", ts)
	}

	var hours int
	if m[1] != "" {
		h, err := strconv.Atoi(strings.TrimSuffix(m[1], ":"))
		if err != nil {
			return 0, err
		}
		hours = h
	}

	minutes, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(m[3])
	if err != nil {
		return 0, err
	}

	millis, err := strconv.Atoi(m[4])
	if err != nil {
		return 0, err
	}

	return float64(hours)*3600 + float64(minutes)*60 + float64(seconds) + float64(millis)/1000, nil
}

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

// deduplicateVTTCues removes overlapping YouTube auto-caption cues.
// YouTube often emits rolling cues where each new cue repeats the previous text
// plus adds new words. We keep only the cue with the most text for each time range.
func deduplicateVTTCues(cues Transcript) Transcript {
	if len(cues) <= 1 {
		return cues
	}

	var result Transcript
	for i := 0; i < len(cues); i++ {
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
