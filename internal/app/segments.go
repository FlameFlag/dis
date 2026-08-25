package app

import (
	"context"
	"errors"

	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/download"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
)

func spanClips(clips []media.Clip) (media.Clip, error) {
	if len(clips) == 0 {
		return media.Clip{}, errors.New("cannot span an empty clip selection")
	}
	return media.NewClipBounds(clips[0].Start, clips[len(clips)-1].End())
}

func (r *Runner) combineInputs(
	ctx context.Context,
	options Options,
	inputs []input,
	clips []media.Clip,
) error {
	span, err := spanClips(clips)
	if err != nil {
		return err
	}
	for _, item := range inputs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.kind == inputURL {
			err = r.combineURL(ctx, options, item.value, span, clips)
		} else {
			err = r.concatFile(ctx, options, item.value, "", clips)
		}
		if err := r.reportInputError(ctx, item.value, err); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) combineURL(
	ctx context.Context,
	options Options,
	rawURL string,
	span media.Clip,
	clips []media.Clip,
) error {
	artifact, err := progressValue(
		ctx,
		r.progress,
		"Downloading...",
		func(runCtx context.Context, sink progress.Sink) (*download.Artifact, error) {
			return r.downloader.Download(runCtx, download.Request{
				URL: rawURL, Clip: &span,
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
	relative := make([]media.Clip, len(clips))
	for index, clip := range clips {
		relative[index] = media.Clip{
			Start: clip.Start - span.Start, Duration: clip.Duration,
		}
	}
	return r.concatFile(
		ctx,
		options,
		artifact.Path,
		artifact.UploadDate,
		relative,
	)
}

func (r *Runner) concatFile(
	ctx context.Context,
	options Options,
	path string,
	uploadDate string,
	clips []media.Clip,
) error {
	result, err := progressValue(
		ctx,
		r.progress,
		"Concatenating...",
		func(runCtx context.Context, sink progress.Sink) (convert.Result, error) {
			return r.converter.Concat(runCtx, convert.ConcatJob{
				InputPath: path, Clips: clips, UploadDate: uploadDate,
				Options: options.Conversion,
			}, sink)
		},
	)
	if err != nil {
		return err
	}
	r.reportResult(options, result)
	return nil
}
