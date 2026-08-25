package convert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/4evy/dis/internal/ffmpeg"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/procgroup"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/timecode"
)

const (
	uploadDateLayout        = "20060102"
	shortUUIDLength         = 4
	automaticFrameDimension = -2
	progressComplete        = 100
	encoderDimensionFactor  = 2
	gifskiFixedArgCount     = 13
)

type runner interface {
	Run(context.Context, []string, float64, progress.Sink) error
	Probe(context.Context, string) (ffmpeg.Info, error)
	FormatCommand([]string) string
}

// Converter converts, copies, and concatenates local media.
type Converter struct {
	runner runner
	gifski string
}

// New creates a converter from resolved process adapters.
func New(runner runner, gifski string) *Converter {
	return &Converter{runner: runner, gifski: gifski}
}

// Job describes one ordinary conversion or copy.
type Job struct {
	InputPath  string
	Clip       *media.Clip
	UploadDate string
	Options    Options
}

// GIFJob describes one GIF export.
type GIFJob struct {
	InputPath  string
	Clip       *media.Clip
	UploadDate string
	Options    Options
	GIF        GIFOptions
}

// ConcatJob describes noncontiguous clips to concatenate.
type ConcatJob struct {
	InputPath  string
	Clips      []media.Clip
	UploadDate string
	Options    Options
}

// Result contains output facts used by application policy and reporting.
type Result struct {
	OutputPath string
	InputSize  int64
	OutputSize int64
}

// Probe returns neutral media facts needed by the application.
func (c *Converter) Probe(ctx context.Context, path string) (ffmpeg.Info, error) {
	return c.runner.Probe(ctx, path)
}

// Convert re-encodes a media file.
func (c *Converter) Convert(
	ctx context.Context,
	job Job,
	sink progress.Sink,
) (Result, error) {
	info, err := c.runner.Probe(ctx, job.InputPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to probe media: %w", err)
	}
	if !info.HasVideo && !info.HasAudio {
		return Result{}, errors.New("no video or audio stream found in file")
	}
	outputPath := constructOutputPath(job.InputPath, job.Options, "")
	args := buildArgs(job.InputPath, outputPath, job.Options, info, job.Clip)
	duration := outputDuration(jobDuration(info.Duration, job.Clip), job.Options.Speed)
	if err := c.runner.Run(ctx, args, duration, sink); err != nil {
		_ = os.Remove(outputPath)
		return Result{}, fmt.Errorf(
			"conversion failed: %w (command: %s)",
			err,
			c.runner.FormatCommand(args),
		)
	}
	applyTimestamps(outputPath, job.UploadDate)
	return resultFor(job.InputPath, outputPath), nil
}

// Copy copies a media artifact without re-encoding it.
func (c *Converter) Copy(
	ctx context.Context,
	job Job,
	sink progress.Sink,
) (Result, error) {
	extension := filepath.Ext(job.InputPath)
	outputPath := constructOutputPath(job.InputPath, job.Options, extension)
	source, err := os.Open(job.InputPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() { _ = source.Close() }()

	total := statSize(job.InputPath)
	destination, err := os.Create(outputPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = destination.Close() }()

	writer := io.Writer(destination)
	if sink != nil {
		writer = &copyProgressWriter{writer: destination, sink: sink, total: total}
	}
	if _, err := copyContext(ctx, writer, source); err != nil {
		_ = destination.Close()
		_ = os.Remove(outputPath)
		return Result{}, fmt.Errorf("failed to copy file: %w", err)
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(outputPath)
		return Result{}, fmt.Errorf("failed to close output file: %w", err)
	}
	applyTimestamps(outputPath, job.UploadDate)
	return resultFor(job.InputPath, outputPath), nil
}

// ExportGIF extracts frames with FFmpeg and encodes them with gifski.
func (c *Converter) ExportGIF(
	ctx context.Context,
	job GIFJob,
	sink progress.Sink,
) (Result, error) {
	if c.gifski == "" {
		return Result{}, errors.New(
			"gifski not found: install it: brew install gifski (macOS) or cargo install gifski",
		)
	}
	info, err := c.runner.Probe(ctx, job.InputPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to probe media: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "dis-gif-*")
	if err != nil {
		return Result{}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	framePattern := filepath.Join(tempDir, "frame%05d.png")
	args := buildFrameArgs(job, framePattern)
	duration := jobDuration(info.Duration, job.Clip)
	if err := c.runner.Run(ctx, args, duration, sink); err != nil {
		return Result{}, fmt.Errorf("frame extraction failed: %w", err)
	}
	frames, err := filepath.Glob(filepath.Join(tempDir, "frame*.png"))
	if err != nil || len(frames) == 0 {
		return Result{}, errors.New("no frames extracted")
	}

	outputPath := constructOutputPath(job.InputPath, job.Options, ".gif")
	gifskiArgs := make([]string, 0, gifskiFixedArgCount+len(frames))
	gifskiArgs = append(gifskiArgs,
		"--fps", strconv.Itoa(job.GIF.FPS),
		"--quality", strconv.Itoa(job.GIF.Quality),
		"--lossy-quality", strconv.Itoa(job.GIF.LossyQuality),
		"--motion-quality", strconv.Itoa(job.GIF.MotionQuality),
		"--width", strconv.Itoa(job.GIF.Width),
		"--quiet", "-o", outputPath,
	)
	gifskiArgs = append(gifskiArgs, frames...)
	command := exec.CommandContext(ctx, c.gifski, gifskiArgs...)
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := procgroup.Run(command, procgroup.DefaultGracePeriod, nil); err != nil {
		_ = os.Remove(outputPath)
		return Result{}, fmt.Errorf(
			"gifski encoding failed: %w: %s",
			err,
			strings.TrimSpace(output.String()),
		)
	}
	if sink != nil {
		sink.Report(progress.Update{Percent: progressComplete})
	}
	applyTimestamps(outputPath, job.UploadDate)
	return resultFor(job.InputPath, outputPath), nil
}

// Concat extracts and concatenates noncontiguous clips.
func (c *Converter) Concat(
	ctx context.Context,
	job ConcatJob,
	sink progress.Sink,
) (Result, error) {
	info, err := c.runner.Probe(ctx, job.InputPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to probe media: %w", err)
	}
	if !info.HasVideo && !info.HasAudio {
		return Result{}, errors.New("no video or audio stream found in file")
	}
	outputPath := constructOutputPath(job.InputPath, job.Options, "")
	args := buildConcatArguments(job.InputPath, outputPath, job.Options, info, job.Clips)
	var duration float64
	for _, clip := range job.Clips {
		duration += clip.Duration
	}
	duration = outputDuration(duration, job.Options.Speed)
	if err := c.runner.Run(ctx, args, duration, sink); err != nil {
		_ = os.Remove(outputPath)
		return Result{}, fmt.Errorf(
			"concatenation failed: %w (command: %s)",
			err,
			c.runner.FormatCommand(args),
		)
	}
	applyTimestamps(outputPath, job.UploadDate)
	return resultFor(job.InputPath, outputPath), nil
}

type copyProgressWriter struct {
	writer      io.Writer
	sink        progress.Sink
	total       int64
	transferred int64
}

func (w *copyProgressWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	w.transferred += int64(n)
	var percent float64
	if w.total > 0 {
		percent = float64(w.transferred) / float64(w.total) * progressComplete
	}
	w.sink.Report(progress.Update{
		Percent: percent, Transferred: w.transferred, Total: w.total,
	})
	return n, err
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	return io.Copy(destination, contextReader{Context: ctx, Reader: source})
}

type contextReader struct {
	context.Context
	io.Reader
}

func (r contextReader) Read(data []byte) (int, error) {
	if err := r.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(data)
}

func resultFor(inputPath, outputPath string) Result {
	return Result{
		OutputPath: outputPath,
		InputSize:  statSize(inputPath),
		OutputSize: statSize(outputPath),
	}
}

func statSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func applyTimestamps(path, uploadDate string) {
	if uploadDate == "" {
		return
	}
	for _, layout := range []string{uploadDateLayout, time.RFC3339} {
		if value, err := time.Parse(layout, uploadDate); err == nil {
			_ = os.Chtimes(path, value, value)
			return
		}
	}
}

func constructOutputPath(inputPath string, options Options, extension string) string {
	if extension == "" {
		extension = options.Codec.config().extension
	}
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	if options.RandomName {
		baseName = shortID()
	}
	outputPath := filepath.Join(options.OutputDir, baseName+extension)
	if _, err := os.Stat(outputPath); err == nil {
		outputPath = filepath.Join(
			options.OutputDir,
			baseName+"-"+shortID()+extension,
		)
	}
	return outputPath
}

func shortID() string { return uuid.New().String()[:shortUUIDLength] }

func jobDuration(duration float64, clip *media.Clip) float64 {
	if clip != nil {
		return clip.Duration
	}
	return duration
}

func outputDuration(duration, speed float64) float64 {
	if speed > DefaultPlaybackSpeed {
		return duration / speed
	}
	return duration
}

func buildFrameArgs(job GIFJob, framePattern string) []string {
	var args []string
	args = append(args, clipArguments(job.Clip)...)
	args = append(args, "-i", job.InputPath)
	filter := fmt.Sprintf(
		"fps=%d,scale=%d:%d",
		job.GIF.FPS,
		job.GIF.Width,
		automaticFrameDimension,
	)
	if speed := videoSpeedFilterV2(job.GIF.Speed); speed != "" {
		filter = speed + "," + filter
	}
	return append(args, "-vf", filter, framePattern)
}

func buildArgs(
	inputPath string,
	outputPath string,
	options Options,
	info ffmpeg.Info,
	clip *media.Clip,
) []string {
	args := clipArguments(clip)
	args = append(args, "-fflags", "+genpts", "-i", inputPath, "-map_metadata", "-1")
	args = appendVideoArguments(args, options, info, jobDuration(info.Duration, clip))

	var videoFilters []string
	if speed := videoSpeedFilterV2(options.Speed); speed != "" {
		videoFilters = append(videoFilters, speed)
	}
	if options.Resolution != 0 && info.HasVideo {
		videoFilters = append(videoFilters, scaleFilterV2(
			options.Resolution,
			info.Width,
			info.Height,
		))
	}
	if len(videoFilters) > 0 {
		args = append(args, "-vf", strings.Join(videoFilters, ","))
	}
	if info.HasAudio {
		args = appendAudioArguments(args, options)
		if speed := audioSpeedFilterV2(options.Speed); speed != "" {
			args = append(args, "-af", speed)
		}
	}
	if !options.Codec.config().webm {
		args = append(args, "-movflags", "+faststart")
	}
	return append(args, outputPath)
}

func clipArguments(clip *media.Clip) []string {
	if clip == nil {
		return nil
	}
	return []string{
		"-ss", timecode.FormatHMS(clip.Start),
		"-t", timecode.FormatHMS(clip.Duration),
	}
}

func appendVideoArguments(
	args []string,
	options Options,
	info ffmpeg.Info,
	duration float64,
) []string {
	codec := options.Codec.config()
	args = append(
		args,
		"-crf", strconv.Itoa(options.CRF),
		"-pix_fmt", codec.pixelFormat,
		"-preset", "veryslow",
		"-c:v", codec.name,
	)
	args = append(args, codecParameters(
		options.Codec,
		options.MultiThread,
		info.Framerate,
	)...)
	return append(args, targetSizeArguments(options, duration)...)
}

func appendAudioArguments(args []string, options Options) []string {
	args = append(args, "-c:a", options.Codec.config().audioCodec)
	if options.AudioBitrate > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", options.AudioBitrate))
	}
	return args
}

func videoSpeedFilterV2(speed float64) string {
	if speed <= DefaultPlaybackSpeed {
		return ""
	}
	return fmt.Sprintf("setpts=PTS/%.4g", speed)
}

func audioSpeedFilterV2(speed float64) string {
	if speed <= DefaultPlaybackSpeed {
		return ""
	}
	return fmt.Sprintf("atempo=%.4g", speed)
}

func scaleFilterV2(resolution, width, height int) string {
	aspectRatio := float64(width) / float64(height)
	outputWidth := int(math.Round(float64(resolution) * aspectRatio))
	outputWidth -= outputWidth % encoderDimensionFactor
	resolution -= resolution % encoderDimensionFactor
	return fmt.Sprintf("scale=%d:%d", outputWidth, resolution)
}

func buildConcatArguments(
	inputPath string,
	outputPath string,
	options Options,
	info ffmpeg.Info,
	clips []media.Clip,
) []string {
	var filters []string
	var inputs strings.Builder
	for index, clip := range clips {
		if info.HasVideo {
			filter := fmt.Sprintf(
				"[0:v]trim=start=%g:end=%g,setpts=PTS-STARTPTS",
				clip.Start,
				clip.End(),
			)
			if speed := videoSpeedFilterV2(options.Speed); speed != "" {
				filter += "," + speed
			}
			filters = append(filters, fmt.Sprintf("%s[v%d]", filter, index))
			fmt.Fprintf(&inputs, "[v%d]", index)
		}
		if info.HasAudio {
			filter := fmt.Sprintf(
				"[0:a]atrim=start=%g:end=%g,asetpts=PTS-STARTPTS",
				clip.Start,
				clip.End(),
			)
			if speed := audioSpeedFilterV2(options.Speed); speed != "" {
				filter += "," + speed
			}
			filters = append(filters, fmt.Sprintf("%s[a%d]", filter, index))
			fmt.Fprintf(&inputs, "[a%d]", index)
		}
	}

	videoOutput := 0
	audioOutput := 0
	if info.HasVideo {
		videoOutput = 1
	}
	if info.HasAudio {
		audioOutput = 1
	}
	concat := inputs.String() + fmt.Sprintf(
		"concat=n=%d:v=%d:a=%d",
		len(clips),
		videoOutput,
		audioOutput,
	)
	if info.HasVideo {
		concat += "[outv]"
	}
	if info.HasAudio {
		concat += "[outa]"
	}
	filters = append(filters, concat)

	args := []string{
		"-fflags", "+genpts", "-i", inputPath,
		"-filter_complex", strings.Join(filters, ";"),
	}
	if info.HasVideo {
		args = append(args, "-map", "[outv]")
	}
	if info.HasAudio {
		args = append(args, "-map", "[outa]")
	}
	args = append(args, "-map_metadata", "-1")
	if info.HasVideo {
		var duration float64
		for _, clip := range clips {
			duration += clip.Duration
		}
		args = appendVideoArguments(args, options, info, duration)
		if options.Resolution != 0 {
			args = append(args, "-vf", scaleFilterV2(
				options.Resolution,
				info.Width,
				info.Height,
			))
		}
	}
	if info.HasAudio {
		args = appendAudioArguments(args, options)
	}
	if !options.Codec.config().webm {
		args = append(args, "-movflags", "+faststart")
	}
	return append(args, outputPath)
}
