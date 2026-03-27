package download

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/charmbracelet/log"
)

// downloadTrimmed runs a trimmed download using go-ytdlp with StderrFunc to
// capture ffmpeg's time= progress during --force-keyframes-at-cuts.
func downloadTrimmed(ctx context.Context, rawURL string, s *config.Settings, trim *config.TrimSettings, tempDir string, onProgress func(tui.ProgressInfo)) error {
	outputTmpl := fmt.Sprintf("%%(display_id)s-%s.%%(ext)s", trim.FilenamePart())

	dl := baseCommand(s, rawURL)
	dl.Output(filepath.Join(tempDir, outputTmpl))
	dl.DownloadSections(trim.DownloadSection())
	dl.ForceKeyframesAtCuts()

	if onProgress != nil && trim.Duration > 0 {
		var mu sync.Mutex
		var maxPct float64

		dl.StderrFunc(func(line string) {
			if t := convert.ParseFFmpegTime(line); t > 0 {
				pct := min(t/trim.Duration*100, 100)
				mu.Lock()
				if pct > maxPct {
					maxPct = pct
				}
				p := maxPct
				mu.Unlock()
				onProgress(tui.ProgressInfo{Percent: p})
			}
		})
	}

	log.Info("Downloading video section", "section", trim.DownloadSection())

	result, err := dl.Run(ctx, rawURL)
	if err != nil {
		if result != nil && result.Stderr != "" {
			return fmt.Errorf("yt-dlp: %w: %s", err, result.Stderr)
		}
		return fmt.Errorf("yt-dlp: %w", err)
	}
	return nil
}
