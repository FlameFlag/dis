package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/download"
	"dis/internal/tui"
	"dis/internal/util"
	"errors"
	"os"

	"github.com/charmbracelet/log"
)

// categorizeInputs splits inputs into URLs and local file paths.
func categorizeInputs(inputs []string) (urls []string, files []string) {
	for _, input := range inputs {
		if util.FileExists(input) {
			files = append(files, input)
		} else if util.IsValidURL(input) {
			urls = append(urls, input)
		}
	}
	return
}

// downloadWithProgress downloads a video and displays a progress bar.
func downloadWithProgress(ctx context.Context, msg, link string, s *config.Settings, trim *config.TrimSettings) (*download.DownloadResult, error) {
	var result *download.DownloadResult
	err := tui.RunWithProgress(ctx, msg, tui.ProgressModeSparkline, func(onProgress func(tui.ProgressInfo)) error {
		var dlErr error
		result, dlErr = download.DownloadVideo(ctx, link, s, trim, onProgress)
		return dlErr
	})
	return result, err
}

// convertDownloaded converts a downloaded video with the given settings.
func convertDownloaded(ctx context.Context, s *config.Settings, result *download.DownloadResult) error {
	if s.GIF {
		return convert.ExportGIF(ctx, result.OutputPath, s, nil, result.UploadDate)
	}
	return convert.ConvertVideo(ctx, result.OutputPath, s, nil, result.UploadDate)
}

// downloadEach drives the per-link download loop: progress bar, temp-dir
// tracking, error logging, and the user-cancel / context-cancel exits that
// every caller had been duplicating. It returns true when the loop aborted
// (cancellation), letting the caller bail in turn.
func downloadEach(
	ctx context.Context,
	s *config.Settings,
	links []string,
	trim *config.TrimSettings,
	msg string,
	tempDirs *[]string,
	onResult func(*download.DownloadResult),
) bool {
	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return true
		}
		result, err := downloadWithProgress(ctx, msg, link, s, trim)
		if errors.Is(err, tui.ErrUserCancelled) {
			return true
		}
		if err != nil {
			log.Error("Failed to download video", "url", link, "err", err)
			continue
		}
		*tempDirs = append(*tempDirs, result.TempDir)
		onResult(result)
	}
	return false
}

func downloadLinks(ctx context.Context, s *config.Settings, links []string, trim *config.TrimSettings, tempDirs *[]string) []*download.DownloadResult {
	if len(links) == 0 {
		return nil
	}

	log.Info("Starting download", "count", len(links))
	var results []*download.DownloadResult
	downloadEach(ctx, s, links, trim, "Downloading...", tempDirs, func(result *download.DownloadResult) {
		log.Info("Downloaded video", "path", result.OutputPath)
		results = append(results, result)
	})
	return results
}

// cleanupDirs removes all temporary directories in the slice.
func cleanupDirs(dirs *[]string) {
	for _, d := range *dirs {
		if err := os.RemoveAll(d); err != nil {
			log.Error("Failed to delete temporary directory", "dir", d, "err", err)
		} else {
			log.Info("Deleted temp dir", "dir", d)
		}
	}
}
