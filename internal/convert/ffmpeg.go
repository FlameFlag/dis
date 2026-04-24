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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	err = procgroup.Run(ctx, cmd, 5*time.Second, func() error {
		scanner := bufio.NewScanner(stderr)
		scanner.Split(ScanFFmpegLines)
		for scanner.Scan() {
			if totalDuration > 0 && onProgress != nil {
				if t := ParseFFmpegTime(scanner.Text()); t > 0 {
					pct := min(int(t/totalDuration*100), 100)
					if pct > 0 {
						onProgress(pct)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}
	return nil
}

// FFmpegArgsString returns a readable representation of the ffmpeg command.
func FFmpegArgsString(args []string) string {
	return "ffmpeg " + strings.Join(args, " ")
}
