package convert

import (
	"cmp"
	"dis/internal/config"
	"dis/internal/validate"
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

	// Target size constraint
	duration := info.Duration
	if trimSettings != nil {
		duration = trimSettings.Duration
	}
	args = append(args, targetBitrateArgs(s, duration)...)

	// Video filter chain (resolution + speed)
	var vFilters []string
	if s.Speed > 1.0 {
		vFilters = append(vFilters, fmt.Sprintf("setpts=PTS/%.4g", s.Speed))
	}
	if s.Resolution != "" && info.HasVideo {
		if sf := scaleFilter(s.Resolution, info.Width, info.Height); sf != "" {
			vFilters = append(vFilters, sf)
		}
	}
	if len(vFilters) > 0 {
		args = append(args, "-vf", strings.Join(vFilters, ","))
	}

	// Audio codec
	if info.HasAudio {
		args = append(args, "-c:a", codec.AudioCodecName())
		if s.AudioBitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", s.AudioBitrate))
		}
		if s.Speed > 1.0 {
			args = append(args, "-af", fmt.Sprintf("atempo=%.4g", s.Speed))
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

// targetSizeArgs returns FFmpeg arguments to constrain the video bitrate.
func targetSizeArgs(videoBitrateKbps int) []string {
	return []string{
		"-maxrate", fmt.Sprintf("%dk", videoBitrateKbps),
		"-bufsize", fmt.Sprintf("%dk", videoBitrateKbps*2),
	}
}

// targetBitrateArgs resolves the target-size setting into -maxrate/-bufsize
// args, or returns nil when unset, unparseable, or the duration is zero.
func targetBitrateArgs(s *config.Settings, duration float64) []string {
	if s.TargetSize == "" {
		return nil
	}
	targetBytes, _ := config.ParseSize(s.TargetSize)
	if targetBytes <= 0 {
		return nil
	}
	audioBitrate := cmp.Or(s.AudioBitrate, validate.DefaultAudioBitrate)
	kbps := config.CalculateVideoBitrate(targetBytes, duration, audioBitrate)
	if kbps <= 0 {
		return nil
	}
	return targetSizeArgs(kbps)
}

// scaleFilter returns a "scale=W:H" filter string for the given resolution,
// preserving aspect ratio and ensuring even dimensions. Returns "" on invalid input.
func scaleFilter(resolution string, origWidth, origHeight int) string {
	cleaned := strings.TrimSuffix(strings.ToLower(resolution), "p")
	resInt, err := strconv.Atoi(cleaned)
	if err != nil {
		return ""
	}

	aspectRatio := float64(origWidth) / float64(origHeight)
	outWidth := int(math.Round(float64(resInt) * aspectRatio))
	outHeight := resInt

	// Ensure even dimensions
	outWidth -= outWidth % 2
	outHeight -= outHeight % 2

	return fmt.Sprintf("scale=%d:%d", outWidth, outHeight)
}

func resolutionArgs(resolution string, origWidth, origHeight int) []string {
	if sf := scaleFilter(resolution, origWidth, origHeight); sf != "" {
		return []string{"-vf", sf}
	}
	return nil
}
