package storyboard

import (
	"image"
	"strings"

	ytdlp "github.com/lrstanley/go-ytdlp"
)

// StoryboardInfo holds metadata about a YouTube storyboard sprite sheet.
type StoryboardInfo struct {
	Rows      int
	Columns   int
	CellW     int
	CellH     int
	Fragments []FragmentInfo
}

// FragmentInfo describes a single storyboard sprite sheet image.
type FragmentInfo struct {
	URL      string
	Duration float64
}

// StoryboardData holds downloaded and decoded storyboard images.
type StoryboardData struct {
	Info   StoryboardInfo
	Images map[int]image.Image // fragment index -> decoded image
}

// ExtractStoryboard finds a storyboard format in yt-dlp metadata and returns its info.
// Prefers the highest-resolution storyboard (largest total area). Returns nil if none found.
func ExtractStoryboard(info *ytdlp.ExtractedInfo) *StoryboardInfo {
	if info == nil || len(info.Formats) == 0 {
		return nil
	}

	// Collect storyboard formats (FormatID starting with "sb")
	var sbFormats []*ytdlp.ExtractedFormat
	for _, f := range info.Formats {
		if f.FormatID != nil && strings.HasPrefix(*f.FormatID, "sb") {
			sbFormats = append(sbFormats, f)
		}
	}
	if len(sbFormats) == 0 {
		return nil
	}

	// Prefer the highest-resolution storyboard (largest total area)
	chosen := sbFormats[0]
	bestArea := float64(0)
	for _, f := range sbFormats {
		if f.Width != nil && f.Height != nil {
			area := *f.Width * *f.Height
			if area > bestArea {
				bestArea = area
				chosen = f
			}
		}
	}

	rows, cols := 0, 0
	if chosen.Rows != nil {
		rows = *chosen.Rows
	}
	if chosen.Columns != nil {
		cols = *chosen.Columns
	}
	if rows == 0 || cols == 0 {
		return nil
	}

	// Cell dimensions are computed from actual image data in FetchStoryboardData.
	cellW := 0
	cellH := 0

	var fragments []FragmentInfo
	for _, frag := range chosen.Fragments {
		if frag == nil {
			continue
		}
		u := frag.URL
		if u == "" && frag.Path != nil && chosen.FragmentBaseURL != nil {
			u = *chosen.FragmentBaseURL + *frag.Path
		}
		if u == "" {
			continue
		}
		fragments = append(fragments, FragmentInfo{
			URL:      u,
			Duration: frag.Duration,
		})
	}
	if len(fragments) == 0 {
		return nil
	}

	return &StoryboardInfo{
		Rows:      rows,
		Columns:   cols,
		CellW:     cellW,
		CellH:     cellH,
		Fragments: fragments,
	}
}
