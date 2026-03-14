package download

import (
	"bufio"
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
)

// downloadTrimmedRaw runs yt-dlp as a raw process for trimmed downloads.
// With --force-keyframes-at-cuts, yt-dlp feeds URLs directly to ffmpeg which
// downloads and re-encodes in one pass. All progress comes from ffmpeg's
// stderr output (frame=... time=HH:MM:SS.ms ...) which go-ytdlp can't expose.
func downloadTrimmedRaw(ctx context.Context, rawURL string, s *config.Settings, trim *config.TrimSettings, tempDir string, onProgress func(tui.ProgressInfo)) error {
	outputTmpl := fmt.Sprintf("%%(display_id)s-%s.%%(ext)s", trim.FilenamePart())

	args := []string{
		"--format-sort", "res,vcodec:h264,ext:mp4:m4a",
		"--merge-output-format", "mp4",
		"--remux-video", "mp4",
		"--embed-metadata",
		"--newline",
		"-o", filepath.Join(tempDir, outputTmpl),
		"--download-sections", trim.DownloadSection(),
		"--force-keyframes-at-cuts",
	}
	if s.Sponsor && util.IsYouTube(rawURL) {
		args = append(args, "--sponsorblock-remove", "all")
	}
	args = append(args, rawURL)

	log.Info("Downloading video section", "section", trim.DownloadSection())
	log.Debug("yt-dlp raw command", "args", args)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("yt-dlp: %w", err)
	}

	var (
		mu     sync.Mutex
		maxPct float64
		wg     sync.WaitGroup
		// Collect stderr lines for error reporting
		stderrLines []string
	)

	emit := func(pct float64) {
		if onProgress == nil {
			return
		}
		mu.Lock()
		if pct > maxPct {
			maxPct = pct
		}
		p := maxPct
		mu.Unlock()
		onProgress(tui.ProgressInfo{Percent: p})
	}

	// Drain stdout (yt-dlp prints [download], [info] etc. here - not much useful progress)
	wg.Go(func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Just drain; no useful incremental progress on stdout
		}
	})

	// Parse stderr for ffmpeg's time= progress
	wg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Split(util.ScanFFmpegLines)

		for scanner.Scan() {
			line := scanner.Text()

			mu.Lock()
			stderrLines = append(stderrLines, line)
			mu.Unlock()

			if trim.Duration > 0 {
				if t := util.ParseFFmpegTime(line); t > 0 {
					pct := min(t/trim.Duration*100, 100)
					emit(pct)
				}
			}
		}
	})

	wg.Wait()
	err = cmd.Wait()
	if err != nil {
		mu.Lock()
		errOutput := strings.Join(stderrLines, "\n")
		mu.Unlock()
		return fmt.Errorf("yt-dlp: %w: %s", err, errOutput)
	}
	return nil
}
