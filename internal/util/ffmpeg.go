package util

import (
	"regexp"
	"strconv"
)

// FFmpegTimeRegex matches "time=HH:MM:SS.ms" in FFmpeg output.
var FFmpegTimeRegex = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d+)`)

// ParseFFmpegTime extracts seconds from a "time=HH:MM:SS.ms" line in FFmpeg output.
// Returns 0 if no match is found.
func ParseFFmpegTime(line string) float64 {
	matches := FFmpegTimeRegex.FindStringSubmatch(line)
	if matches == nil {
		return 0
	}

	hours, _ := strconv.ParseFloat(matches[1], 64)
	minutes, _ := strconv.ParseFloat(matches[2], 64)
	seconds, _ := strconv.ParseFloat(matches[3], 64)
	frac, _ := strconv.ParseFloat("0."+matches[4], 64)

	return hours*3600 + minutes*60 + seconds + frac
}

// ScanFFmpegLines is a bufio.SplitFunc that splits on \n or \r,
// handling FFmpeg's \r-based progress updates.
func ScanFFmpegLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}
