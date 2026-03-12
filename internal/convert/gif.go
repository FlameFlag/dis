package convert

import (
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// ExportGIF converts a video to GIF using FFmpeg (frame extraction) + gifski (encoding).
func ExportGIF(ctx context.Context, inputPath string, s *config.Settings, trimSettings *config.TrimSettings, uploadDate string) error {
	if _, err := exec.LookPath("gifski"); err != nil {
		return fmt.Errorf("gifski not found: install it: brew install gifski (macOS) or cargo install gifski")
	}

	info, err := ProbeMedia(ctx, inputPath)
	if err != nil {
		return fmt.Errorf("failed to probe media: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "dis-gif-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	duration := info.Duration
	if trimSettings != nil {
		duration = trimSettings.Duration
	}

	framePattern := filepath.Join(tmpDir, "frame%05d.png")
	ffmpegArgs := buildFrameExtractionArgs(inputPath, framePattern, s, trimSettings)

	log.Info("Extracting frames...")
	err = tui.RunWithProgress(ctx, "Extracting frames...", tui.ProgressModeBar, func(onProgress func(tui.ProgressInfo)) error {
		return RunFFmpeg(ctx, ffmpegArgs, duration, func(pct int) {
			onProgress(tui.ProgressInfo{Percent: float64(pct)})
		})
	})
	if err != nil {
		return fmt.Errorf("frame extraction failed: %w", err)
	}

	frames, err := filepath.Glob(filepath.Join(tmpDir, "frame*.png"))
	if err != nil || len(frames) == 0 {
		return fmt.Errorf("no frames extracted")
	}

	outputPath := ConstructOutputPathWithExt(inputPath, s, ".gif")

	gifskiArgs := []string{
		"--fps", fmt.Sprintf("%d", s.GIFFps),
		"--quality", fmt.Sprintf("%d", s.GIFQuality),
		"--width", fmt.Sprintf("%d", s.GIFWidth),
		"--quiet",
		"-o", outputPath,
	}
	gifskiArgs = append(gifskiArgs, frames...)

	log.Info("Encoding GIF with gifski...")
	cmd := exec.CommandContext(ctx, "gifski", gifskiArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gifski encoding failed: %w", err)
	}

	if uploadDate != "" {
		setFileTimestamps(outputPath, uploadDate)
	}

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

	log.Info("GIF saved", "path", outputPath)
	return nil
}

func buildFrameExtractionArgs(input, framePattern string, s *config.Settings, trimSettings *config.TrimSettings) []string {
	var args []string
	if trimSettings != nil {
		args = append(args, trimSettings.FFmpegArgs()...)
	}
	args = append(args, "-i", input)
	vf := fmt.Sprintf("fps=%d,scale=%d:-2", s.GIFFps, s.GIFWidth)
	args = append(args, "-vf", vf)
	args = append(args, framePattern)
	return args
}
