package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/download"
	"dis/internal/tui"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
)

func runChapterMode(ctx context.Context, s *config.Settings, links []string) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return err
		}

		log.Info("Fetching metadata...", "url", link)
		info, err := download.FetchMetadata(ctx, link)
		if err != nil {
			log.Error("Failed to fetch metadata", "url", link, "err", err)
			continue
		}

		chapters := download.ExtractChapters(info)
		if len(chapters) == 0 {
			log.Error("No chapters found in video", "url", link)
			continue
		}
		log.Info("Found chapters", "count", len(chapters))

		selection, err := tui.SelectChapters(chapters)
		if err != nil {
			return err
		}
		if selection == nil {
			log.Info("No chapters selected, skipping", "url", link)
			continue
		}

		switch selection.Mode {
		case config.ChapterModeCombined:
			result, err := downloadChaptersCombined(ctx, s, link, selection)
			if errors.Is(err, tui.ErrUserCancelled) {
				return nil
			}
			if err != nil {
				log.Error("Failed to download chapters", "url", link, "err", err)
				continue
			}
			tempDirs = append(tempDirs, result.TempDir)
			log.Info("Downloaded combined chapters", "path", result.OutputPath)
			if err := convertDownloaded(ctx, s, result); err != nil {
				log.Error("Failed to convert", "path", result.OutputPath, "err", err)
			}

		case config.ChapterModeSeparate:
			dirs, err := downloadChaptersSeparate(ctx, s, link, selection)
			tempDirs = append(tempDirs, dirs...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func downloadChaptersCombined(ctx context.Context, s *config.Settings, link string, selection *tui.ChapterSelection) (*download.DownloadResult, error) {
	var result *download.DownloadResult
	err := tui.RunWithProgress(ctx, "Downloading chapters...", tui.ProgressModeSparkline, func(onProgress func(tui.ProgressInfo)) error {
		var dlErr error
		result, dlErr = download.DownloadChaptersCombined(ctx, link, s, selection.Chapters, onProgress)
		return dlErr
	})
	return result, err
}

func downloadChaptersSeparate(ctx context.Context, s *config.Settings, link string, selection *tui.ChapterSelection) (tempDirs []string, _ error) {
	for _, ch := range selection.Chapters {
		if err := ctx.Err(); err != nil {
			return tempDirs, err
		}

		var result *download.DownloadResult
		err := tui.RunWithProgress(ctx, fmt.Sprintf("Downloading %q...", ch.Title), tui.ProgressModeSparkline, func(onProgress func(tui.ProgressInfo)) error {
			var dlErr error
			result, dlErr = download.DownloadChapterSeparate(ctx, link, s, ch, onProgress)
			return dlErr
		})
		if errors.Is(err, tui.ErrUserCancelled) {
			return tempDirs, nil
		}
		if err != nil {
			log.Error("Failed to download chapter", "title", ch.Title, "err", err)
			continue
		}
		tempDirs = append(tempDirs, result.TempDir)
		log.Info("Downloaded chapter", "path", result.OutputPath)
		if err := convertDownloaded(ctx, s, result); err != nil {
			log.Error("Failed to convert", "path", result.OutputPath, "err", err)
		}
	}
	return tempDirs, nil
}
