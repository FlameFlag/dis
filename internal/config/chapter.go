package config

import (
	"dis/internal/util"
	"fmt"
)

// ChapterMode determines how selected chapters are downloaded.
type ChapterMode int

const (
	ChapterModeCombined ChapterMode = iota
	ChapterModeSeparate
)

// Chapter represents a single video chapter.
type Chapter struct {
	Index     int
	Title     string
	StartTime float64
	EndTime   float64
}

// Duration returns the chapter length in seconds.
func (c Chapter) Duration() float64 {
	return c.EndTime - c.StartTime
}

// DownloadSection returns the yt-dlp download section string: "*start-end".
func (c Chapter) DownloadSection() string {
	return fmt.Sprintf("*%s-%s", formatFloat(c.StartTime), formatFloat(c.EndTime))
}

// Label returns a human-readable label like "1. Title (0:23 - 1:45)".
func (c Chapter) Label() string {
	return fmt.Sprintf("%d. %s (%s - %s)", c.Index+1, c.Title,
		util.FormatDurationShort(c.StartTime), util.FormatDurationShort(c.EndTime))
}
