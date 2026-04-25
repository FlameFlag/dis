package convert

import (
	"dis/internal/config"
	"fmt"
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

	args = appendVideoEncoderArgs(args, s, codec, info, clipDuration(info, trimSettings))

	// Video filter chain (resolution + speed)
	var vFilters []string
	if vsf := videoSpeedFilter(s.Speed); vsf != "" {
		vFilters = append(vFilters, vsf)
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
		args = appendAudioEncoderArgs(args, s, codec)
		if asf := audioSpeedFilter(s.Speed); asf != "" {
			args = append(args, "-af", asf)
		}
	}

	// Faststart for MP4
	if !codec.IsWebM() {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, output)
	return args
}

// appendVideoEncoderArgs emits CRF, pixel format, preset, codec, codec-specific
// params, and the target-size bitrate cap, the block that both BuildFFmpegArgs
// and buildConcatArgs need verbatim.
func appendVideoEncoderArgs(args []string, s *config.Settings, codec config.Codec, info *MediaInfo, duration float64) []string {
	args = append(args,
		"-crf", strconv.Itoa(s.Crf),
		"-pix_fmt", codec.PixelFormat(),
		"-preset", "veryslow",
		"-c:v", codec.FFmpegCodecName(),
	)
	args = append(args, codecParams(codec, s.MultiThread, info.Framerate)...)
	args = append(args, targetBitrateArgs(s, duration)...)
	return args
}

// appendAudioEncoderArgs emits -c:a and -b:a (when a bitrate override is set).
func appendAudioEncoderArgs(args []string, s *config.Settings, codec config.Codec) []string {
	args = append(args, "-c:a", codec.AudioCodecName())
	if s.AudioBitrate > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", s.AudioBitrate))
	}
	return args
}
