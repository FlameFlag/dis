package convert

import (
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"dis/internal/util"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
)

// ConvertVideo converts a video file with the given settings and optional trim.
func ConvertVideo(ctx context.Context, inputPath string, s *config.Settings, trimSettings *config.TrimSettings, uploadDate string) error {
	info, err := ProbeMedia(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe media: %w", err)
	}

	if !info.HasVideo && !info.HasAudio {
		return fmt.Errorf("no video or audio stream found in file")
	}

	// Warn if duration exceeds max duration from preset
	if s.MaxDuration > 0 && info.Duration > s.MaxDuration {
		log.Warn("Video duration exceeds platform limit",
			"duration", fmt.Sprintf("%.0fs", info.Duration),
			"max", fmt.Sprintf("%.0fs", s.MaxDuration))
	}

	if skip := checkSkipConversion(s, trimSettings); skip {
		return copyWithoutConversion(inputPath, s, uploadDate)
	}

	for {
		outputPath := ConstructOutputPath(inputPath, s)
		args := BuildFFmpegArgs(inputPath, outputPath, s, info, trimSettings)

		// Determine duration for progress
		duration := info.Duration
		if trimSettings != nil {
			duration = trimSettings.Duration
		}

		log.Info("Starting conversion...", "input", filepath.Base(inputPath))

		err = tui.RunWithProgress(ctx, "Converting...", tui.ProgressModeBar, func(onProgress func(tui.ProgressInfo)) error {
			return RunFFmpeg(ctx, args, duration, func(pct int) {
				onProgress(tui.ProgressInfo{Percent: float64(pct)})
			})
		})
		if errors.Is(err, tui.ErrUserCancelled) {
			return nil
		}
		if err != nil {
			log.Error("Conversion failed", "args", FFmpegArgsString(args))
			return fmt.Errorf("conversion failed: %w", err)
		}

		// Set file timestamps if we have upload date
		if uploadDate != "" {
			setFileTimestamps(outputPath, uploadDate)
		}

		log.Info("Converted video saved", "path", outputPath)

		// Results table
		originalSize := fileSize(inputPath)
		compressedSize := fileSize(outputPath)
		tui.PrintResultsTable(originalSize, compressedSize)


		// Warn if target size is set and output exceeds target
		if s.TargetSize != "" {
			targetBytes, _ := config.ParseSize(s.TargetSize)
			if targetBytes > 0 && compressedSize > targetBytes {
				log.Warn("Output file exceeds target size",
					"target", s.TargetSize,
					"actual", humanize.Bytes(uint64(compressedSize)))
			}
		}

		// Copy to clipboard if enabled
		if s.Copy {
			if err := util.CopyToClipboard(outputPath); err != nil {
				log.Warn("Could not copy to clipboard", "err", err)
			} else {
				log.Info("Copied to clipboard", "path", outputPath)
			}
		}

		// Retry if output is larger
		if compressedSize > originalSize && info.HasVideo {
			retry, err := shouldRetry(s, outputPath, info)
			if err != nil {
				return err
			}
			if retry {
				continue
			}
		}

		return nil
	}
}

func shouldRetry(s *config.Settings, outputPath string, info *MediaInfo) (bool, error) {
	log.Warn("The resulting file is larger than the original.")

	deleteAndRetry, err := confirmPrompt("Delete and try again with better settings?")
	if err != nil || !deleteAndRetry {
		return false, err
	}

	_ = os.Remove(outputPath)
	log.Info("Deleted the oversized converted video.")

	changed := false

	if wantRes, err := confirmPrompt("Would you like to change the resolution?"); err != nil {
		return false, err
	} else if wantRes {
		changed = changeResolution(s, info) || changed
	}

	if wantCrf, err := confirmPrompt("Would you like to enter a new CRF value?"); err != nil {
		return false, err
	} else if wantCrf {
		changed = changeCrfValue(s) || changed
	}

	return changed, nil
}

// confirmPrompt shows a yes/no confirmation and returns the result.
func confirmPrompt(title string) (bool, error) {
	var result bool
	err := huh.NewConfirm().Title(title).Value(&result).Run()
	return result, err
}

func changeResolution(s *config.Settings, info *MediaInfo) bool {
	maxDim := max(info.Width, info.Height)

	// Find current resolution bucket
	currentRes := closestResolution(maxDim)
	var lowerRes []string
	for _, r := range config.ValidResolutions {
		if r < currentRes {
			lowerRes = append(lowerRes, fmt.Sprintf("%dp", r))
		}
	}

	if len(lowerRes) == 0 {
		log.Warn("No lower resolutions available.")
		return false
	}

	var chosen string
	opts := make([]huh.Option[string], len(lowerRes))
	for i, r := range lowerRes {
		opts[i] = huh.NewOption(r, r)
	}

	err := huh.NewSelect[string]().
		Title("Select a lower resolution:").
		Options(opts...).
		Value(&chosen).
		Run()
	if err != nil {
		return false
	}

	s.Resolution = chosen
	return true
}

func closestResolution(dimension int) int {
	resolutions := config.ValidResolutions
	closest := resolutions[0]
	for _, r := range resolutions {
		if r <= dimension {
			closest = r
		}
	}
	return closest
}

func changeCrfValue(s *config.Settings) bool {
	for {
		var input string
		err := huh.NewInput().
			Title("Enter new CRF value (higher = smaller file):").
			Value(&input).
			Run()
		if err != nil {
			return false
		}

		val, err := strconv.Atoi(input)
		if err != nil {
			log.Warn("Invalid CRF value, please enter a number.")
			continue
		}

		if val <= s.Crf {
			log.Warn("Please enter a value higher than the current CRF.", "current", s.Crf)
			continue
		}

		s.Crf = val
		return true
	}
}

func setFileTimestamps(path, uploadDate string) {
	for _, layout := range []string{"20060102", time.RFC3339} {
		if t, err := time.Parse(layout, uploadDate); err == nil {
			_ = os.Chtimes(path, t, t)
			return
		}
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// checkSkipConversion asks the user whether to skip conversion.
// Returns true if conversion should be skipped.
func checkSkipConversion(s *config.Settings, trimSettings *config.TrimSettings) bool {
	if s.NoConvert {
		if trimSettings != nil {
			log.Warn("--no-convert: trimming will not be applied (requires FFmpeg)")
		}
		return true
	}

	var skip bool
	err := huh.NewConfirm().
		Title("Skip conversion?").
		Description("The file will be copied as-is without re-encoding.").
		Value(&skip).
		Run()
	if err != nil {
		return false
	}

	if skip && trimSettings != nil {
		log.Warn("Skipping conversion will also skip trimming for local files")
	}

	return skip
}

// copyWithoutConversion copies the input file to the output directory without re-encoding.
func copyWithoutConversion(inputPath string, s *config.Settings, uploadDate string) error {
	ext := filepath.Ext(inputPath)
	outputPath := ConstructOutputPathWithExt(inputPath, s, ext)

	src, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if uploadDate != "" {
		setFileTimestamps(outputPath, uploadDate)
	}

	log.Info("Copied without conversion", "path", outputPath)

	if s.Copy {
		if err := util.CopyToClipboard(outputPath); err != nil {
			log.Warn("Could not copy to clipboard", "err", err)
		} else {
			log.Info("Copied to clipboard", "path", outputPath)
		}
	}

	return nil
}
