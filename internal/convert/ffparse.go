package convert

import (
	"regexp"
	"strconv"
	"sync"

	"dis/internal/tui"
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

// MakeProgressCallback builds a stderr-line handler that parses FFmpeg's
// time= field and reports monotonically-increasing percent against totalDuration.
// Safe to call from a goroutine. A no-op is returned if onProgress is nil
// or totalDuration is non-positive.
func MakeProgressCallback(totalDuration float64, onProgress func(tui.ProgressInfo)) func(string) {
	if onProgress == nil || totalDuration <= 0 {
		return func(string) {}
	}
	var mu sync.Mutex
	var maxPct float64
	return func(line string) {
		t := ParseFFmpegTime(line)
		if t <= 0 {
			return
		}
		pct := min(t/totalDuration*100, 100)
		mu.Lock()
		if pct > maxPct {
			maxPct = pct
		}
		p := maxPct
		mu.Unlock()
		onProgress(tui.ProgressInfo{Percent: p})
	}
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
