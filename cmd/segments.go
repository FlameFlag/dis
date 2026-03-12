package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"dis/internal/util"
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
)

// runMultiSegmentDownload handles downloads when the user selected non-contiguous segments.
func runMultiSegmentDownload(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	choice, err := promptSegmentChoice(len(segments))
	if err != nil {
		return err
	}

	switch choice {
	case "split":
		return runSplitSegments(ctx, s, links, localFiles, segments)
	case "combine":
		return runCombineSegments(ctx, s, links, localFiles, segments)
	case "span":
		return runSpanSegments(ctx, s, links, localFiles, segments)
	}
	return nil
}

func promptSegmentChoice(count int) (string, error) {
	var choice string
	err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Your selection has %d separate segments. How should they be handled?", count)).
		Options(
			huh.NewOption(fmt.Sprintf("Split into %d separate videos", count), "split"),
			huh.NewOption("Combine into one video (skip gaps)", "combine"),
			huh.NewOption("One video including gaps", "span"),
		).
		Value(&choice).
		Run()
	return choice, err
}

func runSplitSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return err
		}

		trimSettings := seg
		log.Info("Processing segment", "index", i+1, "start", util.FormatDurationShort(seg.Start), "end", util.FormatDurationShort(seg.End()))

		for _, link := range links {
			result, err := downloadWithProgress(ctx, fmt.Sprintf("Downloading segment %d...", i+1), link, s, &trimSettings)
			if errors.Is(err, tui.ErrUserCancelled) {
				return nil
			}
			if err != nil {
				log.Error("Failed to download video", "url", link, "err", err)
				continue
			}
			tempDirs = append(tempDirs, result.TempDir)
			if err := convertDownloaded(ctx, s, result); err != nil {
				log.Error("Failed to convert video", "path", result.OutputPath, "err", err)
			}
		}

		for _, path := range localFiles {
			if err := convert.ConvertVideo(ctx, path, s, &trimSettings, ""); err != nil {
				log.Error("Failed to convert video", "path", path, "err", err)
			}
		}
	}
	return nil
}

func runCombineSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	spanTrim := spanFromSegments(segments)

	relativeSegments := make([]config.TrimSettings, len(segments))
	for i, seg := range segments {
		relativeSegments[i] = config.TrimSettings{
			Start:    seg.Start - spanTrim.Start,
			Duration: seg.Duration,
		}
	}

	for _, link := range links {
		result, err := downloadWithProgress(ctx, "Downloading...", link, s, spanTrim)
		if errors.Is(err, tui.ErrUserCancelled) {
			return nil
		}
		if err != nil {
			log.Error("Failed to download video", "url", link, "err", err)
			continue
		}
		tempDirs = append(tempDirs, result.TempDir)
		if err := convert.ConcatSegments(ctx, result.OutputPath, s, relativeSegments, result.UploadDate); err != nil {
			log.Error("Failed to concatenate segments", "err", err)
		}
	}

	for _, path := range localFiles {
		if err := convert.ConcatSegments(ctx, path, s, segments, ""); err != nil {
			log.Error("Failed to concatenate segments", "path", path, "err", err)
		}
	}
	return nil
}

func runSpanSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	spanTrim := spanFromSegments(segments)

	for _, link := range links {
		result, err := downloadWithProgress(ctx, "Downloading...", link, s, spanTrim)
		if errors.Is(err, tui.ErrUserCancelled) {
			return nil
		}
		if err != nil {
			log.Error("Failed to download video", "url", link, "err", err)
			continue
		}
		tempDirs = append(tempDirs, result.TempDir)
		if err := convertDownloaded(ctx, s, result); err != nil {
			log.Error("Failed to convert video", "path", result.OutputPath, "err", err)
		}
	}

	for _, path := range localFiles {
		if err := convert.ConvertVideo(ctx, path, s, spanTrim, ""); err != nil {
			log.Error("Failed to convert video", "path", path, "err", err)
		}
	}
	return nil
}

func spanFromSegments(segments []config.TrimSettings) *config.TrimSettings {
	return &config.TrimSettings{
		Start:    segments[0].Start,
		Duration: segments[len(segments)-1].End() - segments[0].Start,
	}
}
