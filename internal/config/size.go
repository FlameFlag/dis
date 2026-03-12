package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var sizePattern = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(B|KB|MB|GB|TB|KiB|MiB|GiB|TiB)$`)

// sizeMultipliers maps unit suffixes (lowercased) to byte multipliers.
var sizeMultipliers = map[string]float64{
	"b":   1,
	"kb":  1000,
	"mb":  1000 * 1000,
	"gb":  1000 * 1000 * 1000,
	"tb":  1000 * 1000 * 1000 * 1000,
	"kib": 1024,
	"mib": 1024 * 1024,
	"gib": 1024 * 1024 * 1024,
	"tib": 1024 * 1024 * 1024 * 1024,
}

// ParseSize parses a human-readable size string like "10MB", "2GB", "50MiB"
// and returns the number of bytes.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	matches := sizePattern.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %q (expected e.g. 10MB, 2GB, 50MiB)", s)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value: %q", matches[1])
	}

	unit := strings.ToLower(matches[2])
	multiplier, ok := sizeMultipliers[unit]
	if !ok {
		return 0, fmt.Errorf("unknown size unit: %q", matches[2])
	}

	bytes := int64(value * multiplier)
	if bytes <= 0 {
		return 0, fmt.Errorf("size must be positive: %s", s)
	}

	return bytes, nil
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
