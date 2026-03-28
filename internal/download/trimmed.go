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

// downloadTrimmed runs a trimmed download using go-ytdlp with process group
// cleanup. Parses ffmpeg's time= progress from stderr during
// --force-keyframes-at-cuts re-encoding.
func downloadTrimmed(ctx context.Context, rawURL string, s *config.Settings, trim *config.TrimSettings, tempDir string, onProgress func(tui.ProgressInfo)) error {
	outputTmpl := fmt.Sprintf("%%(display_id)s-%s.%%(ext)s", trim.FilenamePart())

	dl := baseCommand(s, rawURL)
	dl.Output(filepath.Join(tempDir, outputTmpl))
	dl.DownloadSections(trim.DownloadSection())
	dl.ForceKeyframesAtCuts()

	var mu sync.Mutex
	var maxPct float64

	stderrFn := func(line string) {
		if onProgress == nil || trim.Duration <= 0 {
			return
		}
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
	}

	log.Info("Downloading video section", "section", trim.DownloadSection())

	_, err := runInProcessGroup(ctx, dl, rawURL, stderrFn)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	return nil
}
