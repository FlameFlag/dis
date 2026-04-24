package util

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

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
