package download

import (
	"bufio"
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
)

var ffmpegTimeRegex = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d+)`)

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
	if s.Sponsor && isYouTube(rawURL) {
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

	// Drain stdout (yt-dlp prints [download], [info] etc. here — not much useful progress)
	wg.Go(func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Just drain; no useful incremental progress on stdout
		}
	})

	// Parse stderr for ffmpeg's time= progress
	wg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		// ffmpeg writes long lines with \r for in-place updates; split on both
		scanner.Split(splitCRLF)

		for scanner.Scan() {
			line := scanner.Text()

			mu.Lock()
			stderrLines = append(stderrLines, line)
			mu.Unlock()

			if trim.Duration > 0 {
				if m := ffmpegTimeRegex.FindStringSubmatch(line); m != nil {
					h, _ := strconv.ParseFloat(m[1], 64)
					mins, _ := strconv.ParseFloat(m[2], 64)
					sec, _ := strconv.ParseFloat(m[3], 64)
					frac, _ := strconv.ParseFloat("0."+m[4], 64)
					t := h*3600 + mins*60 + sec + frac
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

// splitCRLF is a bufio.SplitFunc that splits on \n or \r (ffmpeg uses \r for progress).
func splitCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
