package download

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// downloadTrimmed runs a trimmed download using go-ytdlp with process group
// cleanup. Parses ffmpeg's time= progress from stderr during
// --force-keyframes-at-cuts re-encoding.
func downloadTrimmed(ctx context.Context, rawURL string, s *config.Settings, trim *config.TrimSettings, tempDir string, onProgress func(tui.ProgressInfo)) error {
	outputTmpl := fmt.Sprintf("%%(display_id)s-%s.%%(ext)s", trim.FilenamePart())

	dl := baseCommand(s, rawURL)
	dl.Output(filepath.Join(tempDir, outputTmpl))
	dl.DownloadSections(trim.DownloadSection())
	dl.ForceKeyframesAtCuts()

	log.Info("Downloading video section", "section", trim.DownloadSection())

	_, err := runInProcessGroup(ctx, dl, rawURL, convert.MakeProgressCallback(trim.Duration, onProgress))
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	return nil
}
