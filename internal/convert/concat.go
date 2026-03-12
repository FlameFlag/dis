package convert

import (
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

// ConcatSegments extracts multiple segments from inputPath and concatenates them
// into a single output video using ffmpeg's trim/concat filters.
func ConcatSegments(ctx context.Context, inputPath string, s *config.Settings, segments []config.TrimSettings, uploadDate string) error {
	info, err := ProbeMedia(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe media: %w", err)
	}

	if !info.HasVideo && !info.HasAudio {
		return fmt.Errorf("no video or audio stream found in file")
	}

	outputPath := ConstructOutputPath(inputPath, s)
	args := buildConcatArgs(inputPath, outputPath, s, info, segments)

	var totalDur float64
	for _, seg := range segments {
		totalDur += seg.Duration
	}

	log.Info("Concatenating segments...", "segments", len(segments), "input", inputPath)

	err = tui.RunWithProgress(ctx, "Concatenating...", tui.ProgressModeBar, func(onProgress func(tui.ProgressInfo)) error {
		return RunFFmpeg(ctx, args, totalDur, func(pct int) {
			onProgress(tui.ProgressInfo{Percent: float64(pct)})
		})
	})
	if err != nil {
		log.Error("Concatenation failed", "args", FFmpegArgsString(args))
		return fmt.Errorf("concatenation failed: %w", err)
	}

	if uploadDate != "" {
		setFileTimestamps(outputPath, uploadDate)
	}

	log.Info("Concatenated video saved", "path", outputPath)

	originalSize := fileSize(inputPath)
	compressedSize := fileSize(outputPath)
	tui.PrintResultsTable(originalSize, compressedSize)

	if s.Copy {
		if err := util.CopyToClipboard(outputPath); err != nil {
			log.Warn("Could not copy to clipboard", "err", err)
		} else {
			log.Info("Copied to clipboard", "path", outputPath)
		}
	}

	return nil
}

// buildConcatArgs builds ffmpeg args using the trim+concat filter approach.
// Each segment is extracted with trim/atrim filters and concatenated.
func buildConcatArgs(input, output string, s *config.Settings, info *MediaInfo, segments []config.TrimSettings) []string {
	codec := config.ParseCodec(s.VideoCodec)
	n := len(segments)

	// Build filter_complex
	var fc strings.Builder
	var concatInputs strings.Builder

	for i, seg := range segments {
		start := seg.Start
		end := seg.End()

		if info.HasVideo {
			fmt.Fprintf(&fc, "[0:v]trim=start=%g:end=%g,setpts=PTS-STARTPTS[v%d];", start, end, i)
		}
		if info.HasAudio {
			fmt.Fprintf(&fc, "[0:a]atrim=start=%g:end=%g,asetpts=PTS-STARTPTS[a%d];", start, end, i)
		}

		if info.HasVideo {
			fmt.Fprintf(&concatInputs, "[v%d]", i)
		}
		if info.HasAudio {
			fmt.Fprintf(&concatInputs, "[a%d]", i)
		}
	}

	// Concat
	vOut := 0
	aOut := 0
	if info.HasVideo {
		vOut = 1
	}
	if info.HasAudio {
		aOut = 1
	}

	fmt.Fprintf(&fc, "%sconcat=n=%d:v=%d:a=%d", concatInputs.String(), n, vOut, aOut)
	if info.HasVideo {
		fc.WriteString("[outv]")
	}
	if info.HasAudio {
		fc.WriteString("[outa]")
	}

	args := []string{
		"-fflags", "+genpts",
		"-i", input,
		"-filter_complex", fc.String(),
	}

	if info.HasVideo {
		args = append(args, "-map", "[outv]")
	}
	if info.HasAudio {
		args = append(args, "-map", "[outa]")
	}

	// Strip metadata
	args = append(args, "-map_metadata", "-1")

	// Video encoding settings
	if info.HasVideo {
		args = append(args, "-crf", fmt.Sprintf("%d", s.Crf))

		args = append(args, "-pix_fmt", codec.PixelFormat())
		args = append(args, "-preset", "veryslow")
		args = append(args, "-c:v", codec.FFmpegCodecName())
		args = append(args, codecParams(codec, s.MultiThread, info.Framerate)...)

		// Target size constraint
		if s.TargetSize != "" {
			targetBytes, _ := config.ParseSize(s.TargetSize)
			if targetBytes > 0 {
				var totalDur float64
				for _, seg := range segments {
					totalDur += seg.Duration
				}
				audioBitrate := s.AudioBitrate
				if audioBitrate == 0 {
					audioBitrate = 128
				}
				videoBitrateKbps := config.CalculateVideoBitrate(targetBytes, totalDur, audioBitrate)
				if videoBitrateKbps > 0 {
					args = append(args, targetSizeArgs(videoBitrateKbps)...)
				}
			}
		}

		if s.Resolution != "" {
			args = append(args, resolutionArgs(s.Resolution, info.Width, info.Height)...)
		}
	}

	// Audio encoding
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
