package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/download"
	"dis/internal/tui"
	"dis/internal/util"
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
	return convert.ConvertVideo(ctx, result.OutputPath, s, nil, result.UploadDate)
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
