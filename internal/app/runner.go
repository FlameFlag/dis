package app

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/download"
	"github.com/4evy/dis/internal/ffmpeg"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/storyboard"
	"github.com/4evy/dis/internal/subtitle"
)

const gifSpeedPromptMinimum = 4

// Runner owns the complete user workflow.
type Runner struct {
	downloader  downloader
	converter   converter
	interaction Interaction
	progress    ProgressRunner
	reporter    Reporter
	subtitles   subtitleLoader
	storyboards storyboardLoader
	sponsors    sponsorLoader
}

// New constructs an application runner from consumer-owned dependencies.
func New(
	downloader downloader,
	converter converter,
	interaction Interaction,
	progressRunner ProgressRunner,
	reporter Reporter,
	subtitles subtitleLoader,
	storyboards storyboardLoader,
	sponsors sponsorLoader,
) *Runner {
	if progressRunner == nil {
		progressRunner = directProgress{}
	}
	if reporter == nil {
		reporter = discardReporter{}
	}
	return &Runner{
		downloader: downloader, converter: converter, interaction: interaction,
		progress: progressRunner, reporter: reporter, subtitles: subtitles,
		storyboards: storyboards, sponsors: sponsors,
	}
}

// Run validates options, resolves interaction, and processes every valid input.
func (r *Runner) Run(ctx context.Context, options Options) error {
	_, err := options.Validate()
	if err != nil {
		return err
	}
	inputs := r.classifyInputs(options.Inputs)
	if len(inputs) == 0 {
		return errors.New("no valid input links or local files were provided")
	}

	if options.Chapter {
		err = r.runChapterMode(ctx, options, inputs)
	} else {
		options, err = r.resolveTrim(ctx, options, inputs)
		if err == nil {
			err = r.runClips(ctx, options, inputs)
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ErrCancelled) {
		return nil
	}
	return err
}

type inputKind int

const (
	inputFile inputKind = iota
	inputURL
)

type input struct {
	value string
	kind  inputKind
}

func (r *Runner) classifyInputs(values []string) []input {
	inputs := make([]input, 0, len(values))
	for _, value := range values {
		info, err := os.Stat(value)
		if err == nil && !info.IsDir() {
			extension := filepath.Ext(value)
			mediaType := mime.TypeByExtension(extension)
			if mediaType == "" {
				r.reporter.Warning(
					"Could not determine content type for file " + value,
				)
			} else if !strings.HasPrefix(mediaType, "video/") &&
				!strings.HasPrefix(mediaType, "audio/") {
				r.reporter.Failure(value, fmt.Errorf(
					"input file is not a recognized video/audio type: %s (type: %s)",
					value,
					mediaType,
				))
				continue
			}
			inputs = append(inputs, input{value: value, kind: inputFile})
			continue
		}
		if IsURL(value) {
			inputs = append(inputs, input{value: value, kind: inputURL})
			continue
		}
		r.reporter.Failure(value, fmt.Errorf("invalid input file or link: %s", value))
	}
	return inputs
}

func (r *Runner) runClips(
	ctx context.Context,
	options Options,
	inputs []input,
) error {
	clips := options.Trim.Clips
	if len(clips) <= 1 {
		var clip *media.Clip
		if len(clips) == 1 {
			clip = &clips[0]
		}
		return r.processInputs(ctx, options, inputs, clip)
	}
	strategy, err := r.interaction.SelectSegmentStrategy(ctx, len(clips))
	if err != nil {
		return err
	}
	switch strategy {
	case SegmentSplit:
		for _, clip := range clips {
			if err := r.processInputs(ctx, options, inputs, &clip); err != nil {
				return err
			}
		}
		return nil
	case SegmentCombine:
		return r.combineInputs(ctx, options, inputs, clips)
	case SegmentSpan:
		span, err := spanClips(clips)
		if err != nil {
			return err
		}
		return r.processInputs(ctx, options, inputs, &span)
	default:
		return fmt.Errorf("unknown segment strategy: %d", strategy)
	}
}

func (r *Runner) processInputs(
	ctx context.Context,
	options Options,
	inputs []input,
	clip *media.Clip,
) error {
	for _, item := range inputs {
		if err := ctx.Err(); err != nil {
			return err
		}
		var err error
		if item.kind == inputURL {
			err = r.processURL(ctx, options, item.value, clip)
		} else {
			err = r.processFile(ctx, options, item.value, "", clip)
		}
		if err := r.reportInputError(ctx, item.value, err); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) processURL(
	ctx context.Context,
	options Options,
	rawURL string,
	clip *media.Clip,
) error {
	artifact, err := progressValue(
		ctx,
		r.progress,
		"Downloading...",
		func(runCtx context.Context, sink progress.Sink) (*download.Artifact, error) {
			return r.downloader.Download(runCtx, download.Request{
				URL: rawURL, Clip: clip,
				RemoveSponsors: options.Download.RemoveSponsors,
			}, sink)
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := artifact.Close(); closeErr != nil {
			r.reporter.Failure(artifact.Path, closeErr)
		}
	}()
	return r.processFile(
		ctx,
		options,
		artifact.Path,
		artifact.UploadDate,
		nil,
	)
}

func (r *Runner) processFile(
	ctx context.Context,
	options Options,
	path string,
	uploadDate string,
	clip *media.Clip,
) error {
	if options.GIF {
		result, err := progressValue(
			ctx,
			r.progress,
			"Exporting GIF...",
			func(runCtx context.Context, sink progress.Sink) (convert.Result, error) {
				return r.converter.ExportGIF(runCtx, convert.GIFJob{
					InputPath: path, Clip: clip, UploadDate: uploadDate,
					Options: options.Conversion, GIF: options.GIFOptions,
				}, sink)
			},
		)
		if err != nil {
			return err
		}
		r.reportResult(options, result)
		return nil
	}

	convertFile := !options.NoConvert
	if convertFile {
		var err error
		convertFile, err = r.interaction.ConfirmConversion(ctx, path)
		if err != nil {
			return err
		}
	}
	if !convertFile {
		if clip != nil {
			r.reporter.Warning("Skipping conversion will also skip trimming for local files")
		}
		result, err := progressValue(
			ctx,
			r.progress,
			"Copying...",
			func(runCtx context.Context, sink progress.Sink) (convert.Result, error) {
				return r.converter.Copy(runCtx, convert.Job{
					InputPath: path, UploadDate: uploadDate, Options: options.Conversion,
				}, sink)
			},
		)
		if err != nil {
			return err
		}
		r.reportResult(options, result)
		return nil
	}
	return r.convertWithRetry(ctx, options, path, uploadDate, clip)
}

func (r *Runner) convertWithRetry(
	ctx context.Context,
	options Options,
	path string,
	uploadDate string,
	clip *media.Clip,
) error {
	info, err := r.converter.Probe(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to probe media: %w", err)
	}
	if options.Conversion.MaxDuration > 0 &&
		info.Duration > options.Conversion.MaxDuration {
		r.reporter.Warning(fmt.Sprintf(
			"Video duration %.0fs exceeds platform limit %.0fs",
			info.Duration,
			options.Conversion.MaxDuration,
		))
	}
	conversionOptions := options.Conversion
	for {
		result, convertErr := progressValue(
			ctx,
			r.progress,
			"Converting...",
			func(runCtx context.Context, sink progress.Sink) (convert.Result, error) {
				return r.converter.Convert(runCtx, convert.Job{
					InputPath: path, Clip: clip, UploadDate: uploadDate,
					Options: conversionOptions,
				}, sink)
			},
		)
		if convertErr != nil {
			return convertErr
		}
		if result.OutputSize <= result.InputSize || !info.HasVideo {
			options.Conversion = conversionOptions
			r.reportResult(options, result)
			return nil
		}
		decision, decisionErr := r.interaction.RetryOversized(ctx, RetryRequest{
			Result: result, Info: info, Options: conversionOptions,
		})
		if decisionErr != nil {
			return decisionErr
		}
		if !decision.Retry {
			options.Conversion = conversionOptions
			r.reportResult(options, result)
			return nil
		}
		if err := os.Remove(result.OutputPath); err != nil {
			return fmt.Errorf("delete oversized output: %w", err)
		}
		conversionOptions = decision.Options
		if _, validationErr := conversionOptions.Validate(); validationErr != nil {
			return validationErr
		}
	}
}

func (r *Runner) reportResult(options Options, result convert.Result) {
	r.reporter.Result(ResultReport{
		Result: result, TargetBytes: options.Conversion.TargetBytes,
		CopyPath: options.Copy,
	})
}

func (r *Runner) reportInputError(
	ctx context.Context,
	inputValue string,
	err error,
) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ErrCancelled) {
		return ErrCancelled
	}
	r.reporter.Failure(inputValue, err)
	return nil
}

func progressValue[T any](
	ctx context.Context,
	runner ProgressRunner,
	label string,
	task func(context.Context, progress.Sink) (T, error),
) (T, error) {
	var result T
	err := runner.Run(ctx, label, func(runCtx context.Context, sink progress.Sink) error {
		var taskErr error
		result, taskErr = task(runCtx, sink)
		return taskErr
	})
	return result, err
}

type directProgress struct{}

func (directProgress) Run(
	ctx context.Context,
	_ string,
	task func(context.Context, progress.Sink) error,
) error {
	return task(ctx, progress.Discard)
}

type discardReporter struct{}

func (discardReporter) Info(string)           {}
func (discardReporter) Warning(string)        {}
func (discardReporter) Failure(string, error) {}
func (discardReporter) Result(ResultReport)   {}

var (
	_ downloader       = (*download.Client)(nil)
	_ converter        = (*convert.Converter)(nil)
	_ subtitleLoader   = (*subtitle.Client)(nil)
	_ storyboardLoader = (*storyboard.Client)(nil)
	_ sponsorLoader    = (*sponsorblock.Client)(nil)
	_                  = ffmpeg.Info{}
)
