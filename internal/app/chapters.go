package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/4evy/dis/internal/download"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/progress"
)

func (r *Runner) runChapterMode(
	ctx context.Context,
	options Options,
	inputs []input,
) error {
	for _, item := range inputs {
		if item.kind != inputURL {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
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
			r.reporter.Failure(item.value, err)
			continue
		}
		if len(metadata.Chapters) == 0 {
			r.reporter.Failure(item.value, errors.New("no chapters found in video"))
			continue
		}
		selection, err := r.interaction.SelectChapters(ctx, metadata.Chapters)
		if err != nil {
			return err
		}
		if selection == nil || len(selection.Chapters) == 0 {
			r.reporter.Info("No chapters selected, skipping " + item.value)
			continue
		}
		switch selection.Mode {
		case ChapterCombined:
			err = r.downloadChapters(
				ctx,
				options,
				item.value,
				selection.Chapters,
				"Downloading chapters...",
			)
			if inputErr := r.reportInputError(ctx, item.value, err); inputErr != nil {
				return inputErr
			}
		case ChapterSeparate:
			for _, chapter := range selection.Chapters {
				err = r.downloadChapter(ctx, options, item.value, chapter)
				if inputErr := r.reportInputError(
					ctx,
					chapter.Title,
					err,
				); inputErr != nil {
					return inputErr
				}
			}
		default:
			return fmt.Errorf("unknown chapter mode: %d", selection.Mode)
		}
	}
	return nil
}

func (r *Runner) downloadChapters(
	ctx context.Context,
	options Options,
	rawURL string,
	chapters []media.Chapter,
	label string,
) error {
	artifact, err := progressValue(
		ctx,
		r.progress,
		label,
		func(runCtx context.Context, sink progress.Sink) (*download.Artifact, error) {
			return r.downloader.Download(runCtx, download.Request{
				URL: rawURL, Chapters: chapters,
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

func (r *Runner) downloadChapter(
	ctx context.Context,
	options Options,
	rawURL string,
	chapter media.Chapter,
) error {
	artifact, err := progressValue(
		ctx,
		r.progress,
		fmt.Sprintf("Downloading %q...", chapter.Title),
		func(runCtx context.Context, sink progress.Sink) (*download.Artifact, error) {
			return r.downloader.Download(runCtx, download.Request{
				URL: rawURL, Clip: &chapter.Clip,
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
