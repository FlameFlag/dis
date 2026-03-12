package convert

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var timeRegex = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2,3})`)

// ProgressFunc is called with progress percentage (0-100).
type ProgressFunc func(percent int)

// RunFFmpeg executes FFmpeg with the given args and reports progress.
func RunFFmpeg(ctx context.Context, args []string, totalDuration float64, onProgress ProgressFunc) error {
	// Add -y to overwrite without asking
	fullArgs := slices.Concat([]string{"-y"}, args)

	cmd := exec.CommandContext(ctx, "ffmpeg", fullArgs...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(scanFFmpegLines)

	for scanner.Scan() {
		line := scanner.Text()
		if totalDuration > 0 && onProgress != nil {
			if t := parseTimeProgress(line); t > 0 {
				pct := min(int(t/totalDuration*100), 100)
				if pct > 0 {
					onProgress(pct)
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}

	return nil
}

// parseTimeProgress extracts seconds from "time=HH:MM:SS.ms" in FFmpeg output.
func parseTimeProgress(line string) float64 {
	matches := timeRegex.FindStringSubmatch(line)
	if matches == nil {
		return 0
	}

	hours, _ := strconv.ParseFloat(matches[1], 64)
	minutes, _ := strconv.ParseFloat(matches[2], 64)
	seconds, _ := strconv.ParseFloat(matches[3], 64)
	ms, _ := strconv.ParseFloat(matches[4], 64)

	// Normalize ms (could be 2 or 3 digits)
	if len(matches[4]) == 2 {
		ms /= 100
	} else {
		ms /= 1000
	}

	return hours*3600 + minutes*60 + seconds + ms
}

// scanFFmpegLines is a custom split function for FFmpeg's stderr output,
// which uses \r for progress updates.
func scanFFmpegLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for \r or \n
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

// FFmpegArgsString returns a readable representation of the ffmpeg command.
func FFmpegArgsString(args []string) string {
	return "ffmpeg " + strings.Join(args, " ")
}
