package app

import (
	"context"
	"errors"

	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/storyboard"
	"github.com/4evy/dis/internal/subtitle"
)

func (r *Runner) resolveTrim(
	ctx context.Context,
	options Options,
	inputs []input,
) (Options, error) {
	if !options.Trim.Interactive {
		return r.resolveGIFSpeed(ctx, options, inputs, false)
	}
	source, err := r.loadTrimSource(ctx, inputs)
	if err != nil {
		return options, err
	}
	if source.duration <= 0 {
		r.reporter.Warning(
			"Could not determine a valid video duration. Skipping trim.",
		)
		options.Trim = TrimOptions{}
		return options, nil
	}
	for {
		selection, err := r.interaction.SelectTrim(
			ctx,
			r.newTrimData(ctx, options, source),
		)
		if err != nil {
			return options, err
		}
		if selection == nil {
			return options, ErrCancelled
		}
		options.Trim.Clips = selection.Clips
		options.GIF = selection.GIF
		if options.GIF {
			options.GIFOptions.Speed = selection.Speed
		} else {
			options.Conversion.Speed = selection.Speed
		}
		options, err = r.resolveGIFSpeed(ctx, options, inputs, true)
		if err == nil {
			return options, nil
		}
		if !errors.Is(err, errGoBack) {
			return options, err
		}
		options.GIFOptions.Speed = convert.DefaultPlaybackSpeed
	}
}

var errGoBack = errors.New("go back to trim selection")

func (r *Runner) resolveGIFSpeed(
	ctx context.Context,
	options Options,
	inputs []input,
	canGoBack bool,
) (Options, error) {
	if !options.GIF || options.GIFOptions.Speed > convert.DefaultPlaybackSpeed {
		return options, nil
	}
	duration := clipsDuration(options.Trim.Clips)
	if duration <= 0 {
		for _, item := range inputs {
			if item.kind == inputFile {
				info, err := r.converter.Probe(ctx, item.value)
				if err == nil {
					duration = info.Duration
				}
				break
			}
		}
	}
	if duration < gifSpeedPromptMinimum {
		return options, nil
	}
	choice, err := r.interaction.SelectGIFSpeed(ctx, GIFSpeedRequest{
		Duration: duration, CanGoBack: canGoBack,
	})
	if err != nil {
		return options, err
	}
	if choice.GoBack {
		return options, errGoBack
	}
	if choice.Speed > convert.DefaultPlaybackSpeed {
		options.GIFOptions.Speed = choice.Speed
	}
	return options, nil
}

type trimSource struct {
	duration     float64
	metadata     media.Metadata
	sponsorBlock bool
}

func (r *Runner) loadTrimSource(
	ctx context.Context,
	inputs []input,
) (trimSource, error) {
	for _, item := range inputs {
		if item.kind != inputFile {
			continue
		}
		info, err := r.converter.Probe(ctx, item.value)
		if err == nil && info.Duration > 0 {
			return trimSource{duration: info.Duration}, nil
		}
		if err != nil {
			r.reporter.Failure(item.value, err)
		}
		break
	}
	for _, item := range inputs {
		if item.kind != inputURL {
			continue
		}
		metadata, err := progressValue(
			ctx,
			r.progress,
			"Fetching metadata...",
			func(runCtx context.Context, _ progress.Sink) (media.Metadata, error) {
				return r.downloader.Metadata(runCtx, item.value)
			},
		)
		if err != nil {
			return trimSource{}, err
		}
		return trimSource{
			duration:     metadata.Duration,
			metadata:     metadata,
			sponsorBlock: sponsorblock.SupportsExtractor(metadata.Extractor),
		}, nil
	}
	return trimSource{}, nil
}

func (r *Runner) newTrimData(
	ctx context.Context,
	options Options,
	source trimSource,
) TrimData {
	data := TrimData{
		Duration: source.duration, Chapters: source.metadata.Chapters,
		GIFAvailable: options.GIFAvailable, GIF: options.GIF,
	}
	if source.metadata.ID == "" {
		return data
	}
	if r.subtitles != nil {
		channel := make(chan subtitle.Transcript, 1)
		data.Transcript = channel
		go func() {
			defer close(channel)
			transcript, err := r.subtitles.Fetch(ctx, source.metadata)
			if err == nil && len(transcript) > 0 {
				channel <- transcript
			}
		}()
	}
	if r.storyboards != nil && len(source.metadata.Storyboards) > 0 {
		channel := make(chan *storyboard.StoryboardData, 1)
		data.Storyboard = channel
		go func() {
			defer close(channel)
			storyboardData, err := r.storyboards.Fetch(ctx, source.metadata)
			if err == nil && storyboardData != nil {
				channel <- storyboardData
			}
		}()
	}
	if r.sponsors != nil && source.sponsorBlock {
		channel := make(chan []sponsorblock.Segment, 1)
		data.SponsorSegments = channel
		go func() {
			defer close(channel)
			segments, err := r.sponsors.Segments(ctx, source.metadata.ID)
			if err == nil && len(segments) > 0 {
				channel <- segments
			}
		}()
	}
	return data
}

func clipsDuration(clips []media.Clip) float64 {
	var duration float64
	for _, clip := range clips {
		duration += clip.Duration
	}
	return duration
}
