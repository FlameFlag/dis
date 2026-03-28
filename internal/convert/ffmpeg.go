package convert

import (
	"bufio"
	"context"
	"dis/internal/procgroup"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// ProgressFunc is called with progress percentage (0-100).
type ProgressFunc func(percent int)

// RunFFmpeg executes FFmpeg with the given args and reports progress.
func RunFFmpeg(ctx context.Context, args []string, totalDuration float64, onProgress ProgressFunc) error {
	// Add -y to overwrite without asking
	fullArgs := slices.Concat([]string{"-y"}, args)

	cmd := exec.CommandContext(ctx, "ffmpeg", fullArgs...)
	procgroup.Setup(cmd, 5*time.Second)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	procgroup.Track(cmd)
	defer procgroup.Untrack(cmd)

	scanner := bufio.NewScanner(stderr)
	scanner.Split(ScanFFmpegLines)

	for scanner.Scan() {
		line := scanner.Text()
		if totalDuration > 0 && onProgress != nil {
			if t := ParseFFmpegTime(line); t > 0 {
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

// FFmpegArgsString returns a readable representation of the ffmpeg command.
func FFmpegArgsString(args []string) string {
	return "ffmpeg " + strings.Join(args, " ")
}
