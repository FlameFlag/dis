package config

import (
	"dis/internal/util"
	"fmt"
	"math"
)

// TrimSettings holds the start time and duration for trimming.
type TrimSettings struct {
	Start    float64 // Start time in seconds
	Duration float64 // Duration in seconds
}

// End returns the end time in seconds.
func (t TrimSettings) End() float64 {
	return t.Start + t.Duration
}

// DownloadSection returns the yt-dlp download section string: "*start-end".
func (t TrimSettings) DownloadSection() string {
	return fmt.Sprintf("*%s-%s", formatFloat(t.Start), formatFloat(t.End()))
}

// FFmpegArgs returns the FFmpeg trim arguments: ["-ss", "HH:MM:SS.mmm", "-t", "HH:MM:SS.mmm"].
func (t TrimSettings) FFmpegArgs() []string {
	return []string{
		"-ss", util.FormatTimeHMS(t.Start),
		"-t", util.FormatTimeHMS(t.Duration),
	}
}

// FilenamePart returns a filename-safe representation: "SS_cs-SS_cs".
func (t TrimSettings) FilenamePart() string {
	return fmt.Sprintf("%s-%s", util.FormatTimeFilename(t.Start), util.FormatTimeFilename(t.End()))
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%g", f)
}
