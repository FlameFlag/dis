// Package media defines values shared across application and adapter boundaries.
package media

import (
	"errors"
	"fmt"
	"math"
)

// Clip is an ordered media range measured in seconds.
type Clip struct {
	Start    float64
	Duration float64
}

// NewClip validates and returns a media clip.
func NewClip(start, duration float64) (Clip, error) {
	if math.IsNaN(start) || math.IsInf(start, 0) || start < 0 {
		return Clip{}, errors.New("clip start must be a non-negative finite value")
	}
	if math.IsNaN(duration) || math.IsInf(duration, 0) || duration <= 0 {
		return Clip{}, errors.New("clip duration must be a positive finite value")
	}
	return Clip{Start: start, Duration: duration}, nil
}

// NewClipBounds validates and returns a clip from ordered bounds.
func NewClipBounds(start, end float64) (Clip, error) {
	if end <= start {
		return Clip{}, fmt.Errorf(
			"end time (%.2f) must be greater than start time (%.2f)",
			end,
			start,
		)
	}
	return NewClip(start, end-start)
}

// End returns the exclusive end of the clip in seconds.
func (c Clip) End() float64 { return c.Start + c.Duration }

// Chapter is a named clip from media metadata.
type Chapter struct {
	Index int
	Title string
	Clip  Clip
}

// SubtitleSource describes one downloadable subtitle representation.
type SubtitleSource struct {
	URL     string
	Headers map[string]string
}

// StoryboardFragment describes one storyboard sprite sheet.
type StoryboardFragment struct {
	URL      string
	Duration float64
}

// StoryboardFormat describes a storyboard representation from metadata.
type StoryboardFormat struct {
	Rows      int
	Columns   int
	Width     float64
	Height    float64
	Fragments []StoryboardFragment
}

// Metadata contains only application-facing data returned by a downloader.
type Metadata struct {
	ID                string
	DisplayID         string
	Extractor         string
	Duration          float64
	UploadDate        string
	Chapters          []Chapter
	Subtitles         map[string][]SubtitleSource
	AutomaticCaptions map[string][]SubtitleSource
	Storyboards       []StoryboardFormat
}
