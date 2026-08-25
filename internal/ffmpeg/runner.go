// Package ffmpeg owns FFmpeg and ffprobe process execution.
package ffmpeg

import (
	"bufio"
	"context"
	"encoding/json/v2"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/4evy/dis/internal/procgroup"
	"github.com/4evy/dis/internal/progress"
)

const (
	percentComplete  = 100
	probeGracePeriod = 3 * time.Second
	stderrTailLines  = 8
)

// Runner executes resolved FFmpeg and ffprobe binaries.
type Runner struct {
	Executable      string
	ProbeExecutable string
}

// New returns a runner configured with resolved executable paths.
func New(executable, probeExecutable string) *Runner {
	return &Runner{Executable: executable, ProbeExecutable: probeExecutable}
}

// Run executes FFmpeg and emits monotonic progress updates.
func (r *Runner) Run(
	ctx context.Context,
	args []string,
	totalDuration float64,
	sink progress.Sink,
) error {
	fullArgs := slices.Concat([]string{"-y"}, args)
	cmd := exec.CommandContext(ctx, r.Executable, fullArgs...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	reportLine := ProgressLineSink(totalDuration, sink)
	var tail lineTail
	err = procgroup.Run(cmd, procgroup.DefaultGracePeriod, func() error {
		scanner := bufio.NewScanner(stderr)
		scanner.Split(ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			tail.Add(line)
			reportLine(line)
		}
		return scanner.Err()
	})
	if err != nil {
		if detail := tail.String(); detail != "" {
			return fmt.Errorf("ffmpeg exited with error: %w: %s", err, detail)
		}
		return fmt.Errorf("ffmpeg exited with error: %w", err)
	}
	if sink != nil {
		sink.Report(progress.Update{Percent: percentComplete})
	}
	return nil
}

// FormatCommand returns a readable representation of an FFmpeg invocation.
func (r *Runner) FormatCommand(args []string) string {
	return r.Executable + " " + strings.Join(args, " ")
}

type lineTail struct {
	lines []string
}

func (t *lineTail) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "frame=") {
		return
	}
	t.lines = append(t.lines, line)
	if len(t.lines) > stderrTailLines {
		t.lines = t.lines[len(t.lines)-stderrTailLines:]
	}
}

func (t *lineTail) String() string { return strings.Join(t.lines, "\n") }

const (
	timeHoursCapture = iota + 1
	timeMinutesCapture
	timeSecondsCapture
	timeFractionCapture
	timeFloatBits = 64
)

var timePattern = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d+)`)

// ParseTime extracts seconds from an FFmpeg progress line.
func ParseTime(line string) float64 {
	matches := timePattern.FindStringSubmatch(line)
	if matches == nil {
		return 0
	}
	hours, _ := strconv.ParseFloat(matches[timeHoursCapture], timeFloatBits)
	minutes, _ := strconv.ParseFloat(matches[timeMinutesCapture], timeFloatBits)
	seconds, _ := strconv.ParseFloat(matches[timeSecondsCapture], timeFloatBits)
	fraction, _ := strconv.ParseFloat(
		"0."+matches[timeFractionCapture],
		timeFloatBits,
	)
	return hours*time.Hour.Seconds() + minutes*time.Minute.Seconds() +
		seconds + fraction
}

// ProgressLineSink returns a concurrency-safe FFmpeg stderr line consumer.
func ProgressLineSink(totalDuration float64, sink progress.Sink) func(string) {
	if sink == nil || totalDuration <= 0 {
		return func(string) {}
	}
	var mu sync.Mutex
	var maximum float64
	return func(line string) {
		elapsed := ParseTime(line)
		if elapsed <= 0 {
			return
		}
		percent := min(elapsed/totalDuration*percentComplete, percentComplete)
		mu.Lock()
		maximum = max(maximum, percent)
		update := progress.Update{Percent: maximum}
		mu.Unlock()
		sink.Report(update)
	}
}

// ScanLines splits FFmpeg output on either a newline or carriage return.
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

// Info contains media facts returned by ffprobe.
type Info struct {
	Duration   float64
	Width      int
	Height     int
	Framerate  float64
	VideoCodec string
	AudioCodec string
	HasVideo   bool
	HasAudio   bool
}

type probeOutput struct {
	Format  probeFormat   `json:"format"`
	Streams []probeStream `json:"streams"`
}

type probeFormat struct {
	Duration string `json:"duration"`
}

type probeStream struct {
	CodecType    string `json:"codec_type"`
	CodecName    string `json:"codec_name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	RFrameRate   string `json:"r_frame_rate"`
	AvgFrameRate string `json:"avg_frame_rate"`
	Duration     string `json:"duration"`
}

// Probe returns stream and container metadata for a local file.
func (r *Runner) Probe(ctx context.Context, path string) (Info, error) {
	cmd := exec.CommandContext(
		ctx,
		r.ProbeExecutable,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	procgroup.Setup(cmd, probeGracePeriod)
	out, err := cmd.Output()
	if err != nil {
		return Info{}, fmt.Errorf("ffprobe failed: %w", err)
	}

	var raw probeOutput
	if err := json.Unmarshal(out, &raw); err != nil {
		return Info{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}
	var info Info
	if raw.Format.Duration != "" {
		info.Duration, _ = strconv.ParseFloat(raw.Format.Duration, timeFloatBits)
	}
	for _, stream := range raw.Streams {
		switch stream.CodecType {
		case "video":
			info.HasVideo = true
			info.VideoCodec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height
			info.Framerate = parseFramerate(stream.RFrameRate)
			if info.Framerate == 0 {
				info.Framerate = parseFramerate(stream.AvgFrameRate)
			}
		case "audio":
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
		}
	}
	for _, stream := range raw.Streams {
		if stream.CodecType != "video" || stream.Duration == "" {
			continue
		}
		duration, parseErr := strconv.ParseFloat(stream.Duration, timeFloatBits)
		if parseErr == nil && duration > 0 &&
			(info.Duration == 0 || duration < info.Duration) {
			info.Duration = duration
		}
	}
	return info, nil
}

// ProbeDuration returns only the media duration.
func (r *Runner) ProbeDuration(ctx context.Context, path string) (float64, error) {
	info, err := r.Probe(ctx, path)
	return info.Duration, err
}

func parseFramerate(rate string) float64 {
	if rate == "" || rate == "0/0" {
		return 0
	}
	var numerator, denominator float64
	if n, err := fmt.Sscanf(
		rate,
		"%f/%f",
		&numerator,
		&denominator,
	); n == 2 && err == nil && denominator != 0 {
		return numerator / denominator
	}
	value, _ := strconv.ParseFloat(rate, timeFloatBits)
	return value
}
