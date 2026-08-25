package storyboard

import (
	"image"
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
	Images []image.Image
}
