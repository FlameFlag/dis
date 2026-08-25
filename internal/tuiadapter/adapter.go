package tuiadapter

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/4evy/dis/internal/app"
	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/timecode"
	"github.com/4evy/dis/internal/tui"
	"github.com/4evy/dis/internal/tui/slider"

	"charm.land/huh/v2"
	"charm.land/log/v2"
	"github.com/atotto/clipboard"
	"github.com/dustin/go-humanize"
)

const (
	longGIFWarningSeconds  = 6
	trimAgainOptionSeconds = 12
	mediumGIFSpeed         = 1.5
	fastGIFSpeed           = 2.0
)

// Adapter implements application interaction, progress, and reporting.
type Adapter struct{}

// NewAdapter returns a terminal adapter.
func NewAdapter() *Adapter { return &Adapter{} }

// Run executes a task under the terminal progress UI.
func (*Adapter) Run(
	ctx context.Context,
	label string,
	task func(context.Context, progress.Sink) error,
) error {
	err := tui.RunWithProgress(
		ctx,
		label,
		tui.ProgressModeDownload,
		func(runCtx context.Context, report func(tui.ProgressInfo)) error {
			return task(runCtx, progress.SinkFunc(func(update progress.Update) {
				report(tui.ProgressInfo{
					Percent:    update.Percent,
					Speed:      update.Speed,
					Downloaded: update.Transferred,
					Total:      update.Total,
					ETA:        update.ETA,
				})
			}))
		},
	)
	return mapCancellation(ctx, err)
}

// SelectChapters presents chapter and ordering choices.
func (*Adapter) SelectChapters(
	ctx context.Context,
	chapters []media.Chapter,
) (*app.ChapterSelection, error) {
	options := make([]huh.Option[int], len(chapters))
	for index, chapter := range chapters {
		label := fmt.Sprintf(
			"%d. %s (%s - %s)",
			chapter.Index+1,
			chapter.Title,
			timecode.FormatShort(chapter.Clip.Start),
			timecode.FormatShort(chapter.Clip.End()),
		)
		options[index] = huh.NewOption(label, index)
	}
	var selected []int
	if err := tui.RunPrompt(ctx, huh.NewMultiSelect[int]().
		Title("Select chapters to download").
		Options(options...).
		Value(&selected)); err != nil {
		return nil, mapCancellation(ctx, err)
	}
	if len(selected) == 0 {
		return nil, nil
	}
	picked := make([]media.Chapter, 0, len(selected))
	for _, index := range selected {
		picked = append(picked, chapters[index])
	}
	mode := app.ChapterSeparate
	if len(picked) > 1 {
		var combine bool
		if err := tui.RunPrompt(ctx, huh.NewConfirm().
			Title("Combine into single video?").
			Value(&combine)); err != nil {
			return nil, mapCancellation(ctx, err)
		}
		if combine {
			mode = app.ChapterCombined
			var order string
			if err := tui.RunPrompt(ctx, huh.NewSelect[string]().
				Title("Chapter order").
				Options(
					huh.NewOption("Original order (recommended)", "original"),
					huh.NewOption("Reverse order", "reverse"),
				).
				Value(&order)); err != nil {
				return nil, mapCancellation(ctx, err)
			}
			if order == "reverse" {
				slices.Reverse(picked)
			}
		}
	}
	return &app.ChapterSelection{Chapters: picked, Mode: mode}, nil
}

// SelectTrim runs the full-screen slider.
func (*Adapter) SelectTrim(
	ctx context.Context,
	data app.TrimData,
) (*app.TrimSelection, error) {
	clips, state, err := slider.Run(ctx, slider.Options{
		Duration: data.Duration, Chapters: data.Chapters,
		Transcript: data.Transcript, Storyboard: data.Storyboard,
		SponsorSegments: data.SponsorSegments,
		GIFAvailable:    data.GIFAvailable, GIF: data.GIF,
	})
	if err != nil {
		return nil, mapCancellation(ctx, err)
	}
	if clips == nil {
		return nil, app.ErrCancelled
	}
	return &app.TrimSelection{Clips: clips, GIF: state.GIF, Speed: state.Speed}, nil
}

// SelectSegmentStrategy asks how to process noncontiguous clips.
func (*Adapter) SelectSegmentStrategy(
	ctx context.Context,
	count int,
) (app.SegmentStrategy, error) {
	var choice app.SegmentStrategy
	err := tui.RunPrompt(ctx, huh.NewSelect[app.SegmentStrategy]().
		Title(fmt.Sprintf(
			"Your selection has %d separate segments. How should they be handled?",
			count,
		)).
		Options(
			huh.NewOption(
				fmt.Sprintf("Split into %d separate videos", count),
				app.SegmentSplit,
			),
			huh.NewOption(
				"Combine into one video (skip gaps)",
				app.SegmentCombine,
			),
			huh.NewOption("One video including gaps", app.SegmentSpan),
		).
		Value(&choice))
	return choice, mapCancellation(ctx, err)
}

// SelectGIFSpeed asks how to reduce a long GIF's size.
func (*Adapter) SelectGIFSpeed(
	ctx context.Context,
	request app.GIFSpeedRequest,
) (app.GIFSpeedChoice, error) {
	description := fmt.Sprintf(
		"GIF is %.0fs - speeding up reduces file size",
		request.Duration,
	)
	if request.Duration >= longGIFWarningSeconds {
		description = fmt.Sprintf(
			"GIF is %.0fs - long GIFs produce large files, speeding up helps",
			request.Duration,
		)
	}
	type choice struct {
		speed  float64
		goBack bool
	}
	options := []huh.Option[choice]{
		huh.NewOption("1x (no change)", choice{speed: 1}),
		huh.NewOption("1.5x", choice{speed: mediumGIFSpeed}),
		huh.NewOption("2x", choice{speed: fastGIFSpeed}),
	}
	if request.CanGoBack && request.Duration >= trimAgainOptionSeconds {
		options = append(options, huh.NewOption(
			"Go back and trim shorter",
			choice{goBack: true},
		))
	}
	var selected choice
	err := tui.RunPrompt(ctx, huh.NewSelect[choice]().
		Title("Speed up GIF playback?").
		Description(description).
		Options(options...).
		Value(&selected))
	return app.GIFSpeedChoice{
		Speed: selected.speed, GoBack: selected.goBack,
	}, mapCancellation(ctx, err)
}

// ConfirmConversion asks whether to re-encode or copy a file.
func (*Adapter) ConfirmConversion(ctx context.Context, _ string) (bool, error) {
	var confirmed bool
	err := tui.RunPrompt(ctx, huh.NewConfirm().
		Title("Convert this file?").
		Description(
			"Re-encode with your current settings. Choose No to copy the file as-is.",
		).
		Value(&confirmed))
	return confirmed, mapCancellation(ctx, err)
}

// RetryOversized gathers safer settings for one retry.
func (*Adapter) RetryOversized(
	ctx context.Context,
	request app.RetryRequest,
) (app.RetryDecision, error) {
	log.Warn("The resulting file is larger than the original.")
	var retry bool
	if err := tui.RunPrompt(ctx, huh.NewConfirm().
		Title("Delete and try again with better settings?").
		Value(&retry)); err != nil || !retry {
		return app.RetryDecision{}, mapCancellation(ctx, err)
	}
	options := request.Options
	changed := false
	var changeResolution bool
	if err := tui.RunPrompt(ctx, huh.NewConfirm().
		Title("Would you like to change the resolution?").
		Value(&changeResolution)); err != nil {
		return app.RetryDecision{}, mapCancellation(ctx, err)
	}
	if changeResolution {
		available := lowerResolutions(request.Info.Height)
		if len(available) == 0 {
			log.Warn("No lower resolutions available.")
		} else {
			resolutionOptions := make([]huh.Option[int], len(available))
			for index, resolution := range available {
				resolutionOptions[index] = huh.NewOption(
					fmt.Sprintf("%dp", resolution),
					resolution,
				)
			}
			var resolution int
			if err := tui.RunPrompt(ctx, huh.NewSelect[int]().
				Title("Select a lower resolution:").
				Options(resolutionOptions...).
				Value(&resolution)); err != nil {
				return app.RetryDecision{}, mapCancellation(ctx, err)
			}
			options.Resolution = resolution
			changed = true
		}
	}
	var changeCRF bool
	if err := tui.RunPrompt(ctx, huh.NewConfirm().
		Title("Would you like to enter a new CRF value?").
		Value(&changeCRF)); err != nil {
		return app.RetryDecision{}, mapCancellation(ctx, err)
	}
	if changeCRF {
		var value string
		if err := tui.RunPrompt(ctx, huh.NewInput().
			Title("Enter new CRF value (higher = smaller file):").
			Value(&value).
			Validate(func(input string) error {
				parsed, err := strconv.Atoi(input)
				if err != nil {
					return fmt.Errorf("enter a number: %w", err)
				}
				if parsed <= options.CRF {
					return fmt.Errorf("enter a value higher than %d", options.CRF)
				}
				return nil
			})); err != nil {
			return app.RetryDecision{}, mapCancellation(ctx, err)
		}
		parsed, _ := strconv.Atoi(value)
		options.CRF = parsed
		changed = true
	}
	return app.RetryDecision{Retry: changed, Options: options}, nil
}

func lowerResolutions(height int) []int {
	resolutions := convert.Resolutions()
	cutoff, _ := slices.BinarySearch(resolutions, height)
	return resolutions[:cutoff]
}

// Info reports an informational event.
func (*Adapter) Info(message string) { log.Info(message) }

// Warning reports a warning event.
func (*Adapter) Warning(message string) { log.Warn(message) }

// Failure reports a scoped failure.
func (*Adapter) Failure(scope string, err error) {
	log.Error("Failed to process input", "input", scope, "err", err)
}

// Result renders output facts and optionally copies the path.
func (*Adapter) Result(report app.ResultReport) {
	tui.PrintResultsTable(report.Result.InputSize, report.Result.OutputSize)
	log.Info("Output saved", "path", report.Result.OutputPath)
	if report.TargetBytes > 0 && report.Result.OutputSize > report.TargetBytes {
		log.Warn(
			"Output file exceeds target size",
			"target", humanize.Bytes(uint64(report.TargetBytes)),
			"actual", humanize.Bytes(uint64(report.Result.OutputSize)),
		)
	}
	if !report.CopyPath {
		return
	}
	if err := clipboard.WriteAll(report.Result.OutputPath); err != nil {
		log.Warn("Could not copy to clipboard", "err", err)
	} else {
		log.Info("Copied to clipboard", "path", report.Result.OutputPath)
	}
}

func mapCancellation(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, tui.ErrUserCancelled) || errors.Is(err, context.Canceled) {
		return app.ErrCancelled
	}
	return err
}
