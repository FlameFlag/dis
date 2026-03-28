package download

import (
	"bufio"
	"bytes"
	"context"
	"dis/internal/convert"
	"dis/internal/procgroup"
	"fmt"
	"time"

	"github.com/lrstanley/go-ytdlp"
)

// runInProcessGroup runs a yt-dlp command with proper process group cleanup.
// Unlike dl.Run(), this uses BuildCommand so we can set up process group
// management before the command starts — ensuring ffmpeg grandchildren are
// killed on cancellation.
func runInProcessGroup(ctx context.Context, dl *ytdlp.Command, url string, onStderrLine func(string)) (string, error) {
	cmd := dl.BuildCommand(ctx, url)
	procgroup.Setup(cmd, 5*time.Second)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start yt-dlp: %w", err)
	}
	procgroup.Track(cmd)
	defer procgroup.Untrack(cmd)

	scanner := bufio.NewScanner(stderrPipe)
	scanner.Split(convert.ScanFFmpegLines)
	for scanner.Scan() {
		if onStderrLine != nil {
			onStderrLine(scanner.Text())
		}
	}

	if err := cmd.Wait(); err != nil {
		return stdout.String(), fmt.Errorf("yt-dlp: %w", err)
	}

	return stdout.String(), nil
}
