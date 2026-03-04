package config

import (
	"fmt"
	"math"
	"time"
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
		"-ss", formatTimeFFmpeg(t.Start),
		"-t", formatTimeFFmpeg(t.Duration),
	}
}

// FilenamePart returns a filename-safe representation: "SS_cs-SS_cs".
func (t TrimSettings) FilenamePart() string {
	return fmt.Sprintf("%s-%s", formatTimeFilename(t.Start), formatTimeFilename(t.End()))
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	return fmt.Sprintf("%g", f)
}

func formatTimeFFmpeg(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func formatTimeFilename(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	wholeSecs := int(d.Seconds())
	centisecs := (int(d.Milliseconds()) % 1000) / 10
	return fmt.Sprintf("%02d_%02d", wholeSecs, centisecs)
}
