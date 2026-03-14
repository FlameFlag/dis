package subtitle

import (
	"bufio"
	"context"
	"dis/internal/util"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	"github.com/lrstanley/go-ytdlp"
)

// SilenceInterval represents a detected silence region.
type SilenceInterval struct {
	Start float64
	End   float64
}

const (
	silenceThreshold   = "-30dB"
	silenceMinDuration = "1.5"
)

var (
	silenceStartRe = regexp.MustCompile(`silence_start:\s*([\d.]+)`)
	silenceEndRe   = regexp.MustCompile(`silence_end:\s*([\d.]+)`)
)

// DetectSilence downloads the worst-quality audio from a URL and runs
// FFmpeg silencedetect to find pauses. Designed to run in a background goroutine.
func DetectSilence(ctx context.Context, rawURL string) ([]SilenceInterval, error) {
	if store, ok := openCache(); ok {
		defer store.Close()
		store.DeleteExpired()
		if data, ok := store.GetSilence(rawURL); ok {
			var intervals []SilenceInterval
			if json.Unmarshal(data, &intervals) == nil {
				log.Debug("Silence cache hit", "url", rawURL)
				return intervals, nil
			}
		}
	}

	intervals, err := detectSilenceFromAudio(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	if store, ok := openCache(); ok {
		defer store.Close()
		if blob, err := json.Marshal(intervals); err == nil {
			store.SetSilence(rawURL, blob)
		}
	}
	return intervals, nil
}

func detectSilenceFromAudio(ctx context.Context, rawURL string) ([]SilenceInterval, error) {
	tmpDir, err := os.MkdirTemp("", "dis-silence-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	audioPath, err := downloadAudio(ctx, rawURL, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("downloading audio: %w", err)
	}

	intervals, err := runSilenceDetect(ctx, audioPath)
	if err != nil {
		return nil, fmt.Errorf("silence detection: %w", err)
	}

	return intervals, nil
}

func downloadAudio(ctx context.Context, rawURL, tmpDir string) (string, error) {
	dl := ytdlp.New()
	dl.Output(filepath.Join(tmpDir, "audio.%(ext)s"))
	dl.Format("worstaudio")
	dl.ExtractAudio()

	_, err := dl.Run(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("yt-dlp audio download failed: %w", err)
	}

	// Find downloaded file
	return util.FindFirstFile(tmpDir)
}

func runSilenceDetect(ctx context.Context, audioPath string) ([]SilenceInterval, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", audioPath,
		"-af", fmt.Sprintf("silencedetect=n=%s:d=%s", silenceThreshold, silenceMinDuration),
		"-f", "null", "-",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	var intervals []SilenceInterval
	var currentStart float64
	hasStart := false

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()

		if m := silenceStartRe.FindStringSubmatch(line); m != nil {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				currentStart = v
				hasStart = true
			}
		}
		if m := silenceEndRe.FindStringSubmatch(line); m != nil {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil && hasStart {
				intervals = append(intervals, SilenceInterval{
					Start: currentStart,
					End:   v,
				})
				hasStart = false
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		// ffmpeg returns non-zero for some formats but still outputs valid data
		if len(intervals) == 0 {
			return nil, fmt.Errorf("ffmpeg silencedetect failed: %w", err)
		}
	}

	log.Debug("Silence detection complete", "intervals", len(intervals), "elapsed", time.Since(start).Round(time.Millisecond))
	return intervals, nil
}
