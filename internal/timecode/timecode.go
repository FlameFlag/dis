// Package timecode parses and formats media times.
package timecode

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	floatBitSize = 64
	centisecond  = 10 * time.Millisecond
)

func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(math.Abs(seconds) * float64(time.Second))
}

func splitHMS(d time.Duration) (h, m, s, ms int) {
	h = int(d / time.Hour)
	d %= time.Hour
	m = int(d / time.Minute)
	d %= time.Minute
	s = int(d / time.Second)
	ms = int(d % time.Second / time.Millisecond)
	return h, m, s, ms
}

// FormatShort formats seconds as "M:SS".
func FormatShort(seconds float64) string {
	duration := secondsToDuration(seconds)
	minutes := int(duration / time.Minute)
	secs := int(duration % time.Minute / time.Second)
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// FormatMillis formats seconds as "M:SS.mmm" and omits zero milliseconds.
func FormatMillis(seconds float64) string {
	duration := secondsToDuration(seconds)
	minutes := int(duration / time.Minute)
	secs := int(duration % time.Minute / time.Second)
	ms := int(duration % time.Second / time.Millisecond)
	if ms == 0 {
		return fmt.Sprintf("%d:%02d", minutes, secs)
	}
	return fmt.Sprintf("%d:%02d.%03d", minutes, secs, ms)
}

// FormatHMS formats seconds as "HH:MM:SS.mmm".
func FormatHMS(seconds float64) string {
	h, m, s, ms := splitHMS(secondsToDuration(seconds))
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

// FormatFilename formats seconds as filename-safe seconds and centiseconds.
func FormatFilename(seconds float64) string {
	d := secondsToDuration(seconds)
	wholeSecs := int(d.Seconds())
	centisecs := int(d % time.Second / centisecond)
	return fmt.Sprintf("%02d_%02d", wholeSecs, centisecs)
}

// FormatETA returns a rounded compact duration.
func FormatETA(d time.Duration) string {
	return max(d.Round(time.Second), 0).String()
}

// Parse parses "MM:SS", "MM:SS.cs", or plain seconds.
func Parse(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if value, err := strconv.ParseFloat(input, floatBitSize); err == nil {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, errors.New("time must be finite")
		}
		return value, nil
	}

	minStr, secStr, ok := strings.Cut(input, ":")
	if !ok {
		return 0, errors.New("unrecognized time format")
	}
	minutes, err := strconv.Atoi(minStr)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %w", err)
	}
	seconds, err := strconv.ParseFloat(secStr, floatBitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %w", err)
	}
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, errors.New("seconds must be finite")
	}
	return float64(minutes)*time.Minute.Seconds() + seconds, nil
}
