package config

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
)

// ParseSize parses a human-readable size string like "10MB", "2GB", "50MiB"
// and returns the number of bytes.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	b, err := humanize.ParseBytes(s)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %q (expected e.g. 10MB, 2GB, 50MiB)", s)
	}
	if b == 0 {
		return 0, fmt.Errorf("size must be positive: %s", s)
	}
	return int64(b), nil
}

// CalculateVideoBitrate computes the video bitrate (in kbit/s) needed to fit
// a file within targetBytes, given the duration and audio bitrate.
// Returns 0 if the calculation would produce a non-positive result.
func CalculateVideoBitrate(targetBytes int64, durationSecs float64, audioBitrateKbps int) int {
	if durationSecs <= 0 {
		return 0
	}

	// Total bitrate in kbit/s, with 5% overhead margin for container/muxing
	totalBitrateKbps := float64(targetBytes) * 8 / durationSecs / 1000 * 0.95
	videoBitrateKbps := totalBitrateKbps - float64(audioBitrateKbps)

	if videoBitrateKbps <= 0 {
		return 0
	}

	return int(videoBitrateKbps)
}
