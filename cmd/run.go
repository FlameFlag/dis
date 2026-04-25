package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"dis/internal/validate"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func validateAll(s *config.Settings, cfg *config.FileConfig) error {
	errs := errors.Join(
		validate.Inputs(s.Input),
		validate.Output(s.Output),
		validate.Crf(s.Crf),
		validate.AudioBitrate(s.AudioBitrate),
		validate.Resolution(s.Resolution),
		validate.VideoCodec(s.VideoCodec),
		validate.TargetSize(s.TargetSize),
		validate.Preset(s.Preset, cfg.Presets),
		validate.Speed(s.Speed),
	)
	if s.GIF {
		errs = errors.Join(errs,
			validate.GIFFps(s.GIFFps),
			validate.GIFWidth(s.GIFWidth),
			validate.GIFQuality(s.GIFQuality),
			validate.GIFLossyQuality(s.GIFLossyQuality),
			validate.GIFMotionQuality(s.GIFMotionQuality),
			validate.Speed(s.GIFSpeed),
		)
	}
	return errs
}

func run(ctx context.Context, s *config.Settings) error {
	for _, dep := range []string{"ffmpeg", "yt-dlp"} {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s not found, please install it and ensure it is in your PATH", dep)
		}
	}

	if s.GIF {
		if _, err := exec.LookPath("gifski"); err != nil {
			return fmt.Errorf("gifski not found - install it: brew install gifski (macOS) or cargo install gifski")
		}
	}

	if err := resolveOutput(s); err != nil {
		return err
	}

	links, localFiles := categorizeInputs(s.Input)
	if len(links) == 0 && len(localFiles) == 0 {
		log.Warn("No valid input links or local files were provided.")
		return nil
	}

	if s.Chapter {
		if len(links) == 0 {
			return fmt.Errorf("--chapter requires a URL input")
		}
		return runChapterMode(ctx, s, links)
	}

	trimSegments, err := resolveTrimWithSpeedPrompt(ctx, s, links, localFiles)
	if errors.Is(err, tui.ErrUserCancelled) {
		return nil
	}
	if err != nil {
		return err
	}

	if len(trimSegments) > 1 {
		return runMultiSegmentDownload(ctx, s, links, localFiles, trimSegments)
	}

	var trimSettings *config.TrimSettings
	if len(trimSegments) == 1 {
		trimSettings = &trimSegments[0]
	}

	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	downloaded, cancelled := downloadLinks(ctx, s, links, trimSettings, "Downloading...", &tempDirs)
	if cancelled {
		return nil
	}

	for _, r := range downloaded {
		if err := convertDownloaded(ctx, s, r); err != nil {
			log.Error("Failed to convert video", "path", r.OutputPath, "err", err)
		}
	}

	for _, path := range localFiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.GIF {
			if err := convert.ExportGIF(ctx, path, s, trimSettings, ""); err != nil {
				log.Error("Failed to export GIF", "path", path, "err", err)
			}
		} else {
			if err := convert.ConvertVideo(ctx, path, s, trimSettings, ""); err != nil {
				log.Error("Failed to convert video", "path", path, "err", err)
			}
		}
	}

	return nil
}

func resolveOutput(s *config.Settings) error {
	if s.Output == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		s.Output = cwd
	}
	if abs, err := filepath.Abs(s.Output); err == nil {
		s.Output = abs
	}
	return nil
}
