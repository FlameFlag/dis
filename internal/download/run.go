package download

import (
	"bufio"
	"bytes"
	"context"
	"dis/internal/convert"
	"dis/internal/procgroup"
	"fmt"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
)

// runInProcessGroup runs a yt-dlp command with proper process group cleanup.
// Unlike dl.Run(), this uses BuildCommand so we can set up process group
// management before the command starts, ensuring ffmpeg grandchildren are
// killed on cancellation.
func runInProcessGroup(ctx context.Context, dl *ytdlp.Command, url string, onStderrLine func(string)) (string, error) {
	cmd := dl.BuildCommand(ctx, url)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe: %w", err)
	}

	var stderrTail lineTail
	err = procgroup.Run(cmd, 5*time.Second, func() error {
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Split(convert.ScanFFmpegLines)
		for scanner.Scan() {
			line := scanner.Text()
			stderrTail.add(line)
			if onStderrLine != nil {
				onStderrLine(line)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		if stderr := stderrTail.String(); stderr != "" {
			return stdout.String(), fmt.Errorf("yt-dlp: %w: %s", err, stderr)
		}
		return stdout.String(), fmt.Errorf("yt-dlp: %w", err)
	}
	return stdout.String(), nil
}

type lineTail struct {
	lines []string
}

func (t *lineTail) add(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "frame=") {
		return
	}
	const maxLines = 8
	t.lines = append(t.lines, line)
	if len(t.lines) > maxLines {
		t.lines = t.lines[len(t.lines)-maxLines:]
	}
}

func (t *lineTail) String() string {
	return strings.Join(t.lines, "\n")
}
