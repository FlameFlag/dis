package convert

import (
	"context"
	"dis/internal/procgroup"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
)

// Stream codec types from ffprobe output.
const (
	codecTypeVideo = "video"
	codecTypeAudio = "audio"
)

// MediaInfo holds information about a media file.
type MediaInfo struct {
	Duration   float64
	Width      int
	Height     int
	Framerate  float64
	VideoCodec string
	AudioCodec string
	HasVideo   bool
	HasAudio   bool
}

type ffprobeOutput struct {
	Format  ffprobeFormat   `json:"format"`
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
	Duration     string `json:"duration"`
}

// ProbeMedia runs ffprobe and returns media information.
func ProbeMedia(ctx context.Context, path string) (*MediaInfo, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	procgroup.Setup(cmd, 3*time.Second)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &MediaInfo{}

	// Parse duration
	if probe.Format.Duration != "" {
		d, err := strconv.ParseFloat(probe.Format.Duration, 64)
		if err != nil {
			log.Warn("Could not parse ffprobe duration", "value", probe.Format.Duration, "err", err)
		}
		info.Duration = d
	}

	// Parse streams
	for _, s := range probe.Streams {
		switch s.CodecType {
		case codecTypeVideo:
			info.HasVideo = true
			info.VideoCodec = s.CodecName
			info.Width = s.Width
			info.Height = s.Height
			info.Framerate = parseFramerate(s.RFrameRate)
			if info.Framerate == 0 {
				info.Framerate = parseFramerate(s.AvgFrameRate)
			}
		case codecTypeAudio:
			info.HasAudio = true
			info.AudioCodec = s.CodecName
		}
	}

	// Cross-check format duration against video stream duration - use the shorter one.
	// Trimmed downloads often have incorrect container-level duration metadata.
	for _, s := range probe.Streams {
		if s.CodecType == codecTypeVideo && s.Duration != "" {
			if streamDur, err := strconv.ParseFloat(s.Duration, 64); err == nil && streamDur > 0 {
				if info.Duration == 0 || streamDur < info.Duration {
					info.Duration = streamDur
				}
			}
		}
	}

	return info, nil
}

// ProbeDuration is a convenience wrapper that returns just the duration.
func ProbeDuration(ctx context.Context, path string) (float64, error) {
	info, err := ProbeMedia(ctx, path)
	if err != nil {
		return 0, err
	}
	return info.Duration, nil
}

func parseFramerate(rate string) float64 {
	if rate == "" || rate == "0/0" {
		return 0
	}

	// Try parsing as fraction "num/den"
	var num, den float64
	if n, err := fmt.Sscanf(rate, "%f/%f", &num, &den); n == 2 && err == nil && den != 0 {
		return num / den
	}

	// Try parsing as plain float
	f, _ := strconv.ParseFloat(rate, 64)
	return f
}
