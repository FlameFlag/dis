package subtitle

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var srtTimestampRe = regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2}),(\d{3})`)

// ParseSRT parses an SRT subtitle string into a Transcript.
func ParseSRT(data string) (Transcript, error) {
	type state int
	const (
		stateSeeking state = iota
		stateTimestamp
		stateCueText
	)

	var (
		transcript Transcript
		current    = stateSeeking
		cueStart   float64
		cueEnd     float64
		textLines  []string
	)

	flushCue := func() {
		if len(textLines) == 0 {
			return
		}
		rawText := strings.Join(textLines, " ")
		plainText := stripTags(rawText)
		plainText = strings.TrimSpace(plainText)
		if plainText == "" {
			textLines = nil
			return
		}
		transcript = append(transcript, Cue{
			Start: cueStart,
			End:   cueEnd,
			Text:  plainText,
		})
		textLines = nil
	}

	for line := range strings.Lines(data) {
		line = strings.TrimRight(line, "\r\n")
		// Strip BOM
		line = strings.TrimPrefix(line, "\ufeff")

		switch current {
		case stateSeeking:
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Check if it's a sequence number (digits only)
			if _, err := strconv.Atoi(line); err == nil {
				current = stateTimestamp
				continue
			}
			// Could also be a timestamp line directly
			if arrowRe.MatchString(line) {
				flushCue()
				start, end, err := parseSRTTimestampLine(line)
				if err != nil {
					continue
				}
				cueStart = start
				cueEnd = end
				current = stateCueText
			}

		case stateTimestamp:
			line = strings.TrimSpace(line)
			if arrowRe.MatchString(line) {
				flushCue()
				start, end, err := parseSRTTimestampLine(line)
				if err != nil {
					current = stateSeeking
					continue
				}
				cueStart = start
				cueEnd = end
				current = stateCueText
			} else {
				current = stateSeeking
			}

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

	if len(transcript) == 0 {
		return nil, fmt.Errorf("no cues found in SRT data")
	}

	return transcript, nil
}

func parseSRTTimestampLine(line string) (float64, float64, error) {
	parts := arrowRe.Split(line, 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid timestamp line")
	}

	start, err := parseSRTTimestamp(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}

	end, err := parseSRTTimestamp(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}

	return start, end, nil
}

func parseSRTTimestamp(ts string) (float64, error) {
	m := srtTimestampRe.FindStringSubmatch(ts)
	if m == nil {
		return 0, fmt.Errorf("invalid SRT timestamp: %s", ts)
	}
	return hmsToSeconds(m[1], m[2], m[3], m[4])
}
