package util

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FormatDurationShort formats seconds as "M:SS".
func FormatDurationShort(seconds float64) string {
	seconds = math.Abs(seconds)
	d := time.Duration(seconds * float64(time.Second))
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

// FormatDurationMillis formats seconds as "M:SS.mmm" with millisecond precision.
// The ".mmm" suffix is omitted when milliseconds are zero.
func FormatDurationMillis(seconds float64) string {
	seconds = math.Abs(seconds)
	d := time.Duration(seconds * float64(time.Second))
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	if ms == 0 {
		return fmt.Sprintf("%d:%02d", mins, secs)
	}
	return fmt.Sprintf("%d:%02d.%03d", mins, secs, ms)
}

// ParseTimeValue parses "MM:SS", "MM:SS.cs", or a plain float string into seconds.
func ParseTimeValue(input string) (float64, error) {
	input = strings.TrimSpace(input)

	// Try as plain number (seconds, possibly with centiseconds)
	if f, err := strconv.ParseFloat(input, 64); err == nil {
		return f, nil
	}

	// Try as MM:SS or MM:SS.CS
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

// FileExists checks if path exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// IsValidURL checks for a valid URL with scheme and host.
func IsValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// FindFirstFile returns the path to the first non-directory entry in dir.
func FindFirstFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no file found in %s", dir)
}

// NearestIndex returns the index of the item closest to target by the given key.
// Returns -1 if items is empty.
func NearestIndex[T any](items []T, target float64, key func(T) float64) int {
	if len(items) == 0 {
		return -1
	}
	best := 0
	bestDist := math.Abs(target - key(items[0]))
	for i := 1; i < len(items); i++ {
		if d := math.Abs(target - key(items[i])); d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}

// IsYouTube returns true if the URL appears to be a YouTube link.
func IsYouTube(rawURL string) bool {
	return strings.Contains(rawURL, "youtu")
}

// ShortGUID returns a short random hex string (4 chars).
func ShortGUID() string {
	return fmt.Sprintf("%04x", rand.N(0x10000))
}

// ShortID returns a 6-char random hex string for temp directories.
func ShortID() string {
	return fmt.Sprintf("%06x", rand.N(0x1000000))
}
