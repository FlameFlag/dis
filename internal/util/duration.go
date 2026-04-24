package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(math.Abs(seconds) * float64(time.Second))
}

// splitHMS breaks a duration into hours, minutes, seconds, and milliseconds.
func splitHMS(d time.Duration) (h, m, s, ms int) {
	h = int(d.Hours())
	m = int(d.Minutes()) % 60
	s = int(d.Seconds()) % 60
	ms = int(d.Milliseconds()) % 1000
	return
}

// FormatDurationShort formats seconds as "M:SS".
func FormatDurationShort(seconds float64) string {
	_, m, s, _ := splitHMS(secondsToDuration(seconds))
	return fmt.Sprintf("%d:%02d", m, s)
}

// FormatDurationMillis formats seconds as "M:SS.mmm" with millisecond precision.
// The ".mmm" suffix is omitted when milliseconds are zero.
func FormatDurationMillis(seconds float64) string {
	_, m, s, ms := splitHMS(secondsToDuration(seconds))
	if ms == 0 {
		return fmt.Sprintf("%d:%02d", m, s)
	}
	return fmt.Sprintf("%d:%02d.%03d", m, s, ms)
}

// FormatTimeHMS formats seconds as "HH:MM:SS.mmm" — the form FFmpeg expects for -ss/-t.
func FormatTimeHMS(seconds float64) string {
	h, m, s, ms := splitHMS(secondsToDuration(seconds))
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// FormatTimeFilename formats seconds as "SS_cs" (seconds and centiseconds), safe for filenames.
func FormatTimeFilename(seconds float64) string {
	d := secondsToDuration(seconds)
	wholeSecs := int(d.Seconds())
	centisecs := (int(d.Milliseconds()) % 1000) / 10
	return fmt.Sprintf("%02d_%02d", wholeSecs, centisecs)
}

// FormatETAShort returns a short ETA string like "4s" or "1m12s".
func FormatETAShort(d time.Duration) string {
	d = max(d.Round(time.Second), 0)
	s := int(math.Round(d.Seconds()))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

// ParseTimeValue parses "MM:SS", "MM:SS.cs", or a plain float string into seconds.
func ParseTimeValue(input string) (float64, error) {
	input = strings.TrimSpace(input)

	if f, err := strconv.ParseFloat(input, 64); err == nil {
		return f, nil
	}

	minStr, secStr, ok := strings.Cut(input, ":")
	if !ok {
		return 0, fmt.Errorf("unrecognized time format")
	}

	minutes, err := strconv.Atoi(minStr)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %w", err)
	}

	secs, err := strconv.ParseFloat(secStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %w", err)
	}

	return float64(minutes)*60 + secs, nil
}
