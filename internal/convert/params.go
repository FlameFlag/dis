package convert

import (
	"dis/internal/config"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
)

// BuildFFmpegArgs constructs the full FFmpeg argument list.
func BuildFFmpegArgs(input string, output string, s *config.Settings, info *MediaInfo, trimSettings *config.TrimSettings) []string {
	codec := config.ParseCodec(s.VideoCodec)
	args := []string{}

	// Trim args go first (before -i)
	if trimSettings != nil {
		args = append(args, trimSettings.FFmpegArgs()...)
	}

	// Regenerate timestamps to fix inherited metadata from trimmed downloads
	args = append(args, "-fflags", "+genpts")

	args = append(args, "-i", input)

	// Strip inherited metadata so ffmpeg generates fresh container duration
	args = append(args, "-map_metadata", "-1")

	// CRF
	args = append(args, "-crf", strconv.Itoa(s.Crf))

	// Pixel format
	args = append(args, "-pix_fmt", codec.PixelFormat())

	// Preset
	args = append(args, "-preset", "veryslow")

	// Video codec
	args = append(args, "-c:v", codec.FFmpegCodecName())

	// Codec-specific params
	args = append(args, codecParams(codec, s.MultiThread, info.Framerate)...)

	// Resolution scaling
	if s.Resolution != "" && info.HasVideo {
		args = append(args, resolutionArgs(s.Resolution, info.Width, info.Height)...)
	}

	// Audio codec
	if info.HasAudio {
		args = append(args, "-c:a", codec.AudioCodecName())
		if s.AudioBitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", s.AudioBitrate))
		}
	}

	// Faststart for MP4
	if !codec.IsWebM() {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, output)
	return args
}

func codecParams(codec config.Codec, multiThread bool, framerate float64) []string {
	switch codec {
	case config.CodecH264, config.CodecHEVC:
		if multiThread {
			return []string{"-threads", strconv.Itoa(runtime.NumCPU())}
		}
		return nil

	case config.CodecVP9:
		return []string{
			"-row-mt", "1",
			"-lag-in-frames", "25",
			"-cpu-used", "4",
			"-auto-alt-ref", "1",
			"-arnr-maxframes", "7",
			"-arnr-strength", "4",
			"-aq-mode", "0",
			"-enable-tpl", "1",
		}

	case config.CodecAV1:
		cpuUsed := "4"
		if framerate < 24 {
			cpuUsed = "2"
		} else if framerate > 60 {
			cpuUsed = "6"
		}
		return []string{
			"-lag-in-frames", "48",
			"-row-mt", "1",
			"-tile-rows", "0",
			"-tile-columns", "1",
			"-cpu-used", cpuUsed,
		}

	default:
		return nil
	}
}

func resolutionArgs(resolution string, origWidth, origHeight int) []string {
	cleaned := strings.TrimSuffix(strings.ToLower(resolution), "p")
	resInt, err := strconv.Atoi(cleaned)
	if err != nil {
		return nil
	}

	aspectRatio := float64(origWidth) / float64(origHeight)
	outWidth := int(math.Round(float64(resInt) * aspectRatio))
	outHeight := resInt

	// Ensure even dimensions
	outWidth -= outWidth % 2
	outHeight -= outHeight % 2

	return []string{"-vf", fmt.Sprintf("scale=%d:%d", outWidth, outHeight)}
}
