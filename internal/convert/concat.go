package convert

import (
	"context"
	"dis/internal/config"
	"dis/internal/tui"
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
	if s.Speed > 1.0 {
		totalDur /= s.Speed
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

	reportResults(inputPath, outputPath, s)

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
			vf := fmt.Sprintf("[0:v]trim=start=%g:end=%g,setpts=PTS-STARTPTS", start, end)
			if vsf := videoSpeedFilter(s.Speed); vsf != "" {
				vf += "," + vsf
			}
			fmt.Fprintf(&fc, "%s[v%d];", vf, i)
		}
		if info.HasAudio {
			af := fmt.Sprintf("[0:a]atrim=start=%g:end=%g,asetpts=PTS-STARTPTS", start, end)
			if asf := audioSpeedFilter(s.Speed); asf != "" {
				af += "," + asf
			}
			fmt.Fprintf(&fc, "%s[a%d];", af, i)
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
		var totalDur float64
		for _, seg := range segments {
			totalDur += seg.Duration
		}
		args = appendVideoEncoderArgs(args, s, codec, info, totalDur)

		if s.Resolution != "" {
			args = append(args, resolutionArgs(s.Resolution, info.Width, info.Height)...)
		}
	}

	// Audio encoding
	if info.HasAudio {
		args = appendAudioEncoderArgs(args, s, codec)
	}

	// Faststart for MP4
	if !codec.IsWebM() {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, output)
	return args
}
