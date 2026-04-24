package cmd

import (
	"context"
	"dis/internal/convert"
	"dis/internal/download"
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/subtitle"
	"dis/internal/tui"
	"dis/internal/tui/slider"

	"github.com/charmbracelet/log"
	ytdlp "github.com/lrstanley/go-ytdlp"
	"golang.org/x/sync/errgroup"
)

// sliderData holds pre-fetched data needed to run the trim slider.
type sliderData struct {
	duration     float64
	markers      []slider.ChapterMarker
	transcriptCh <-chan subtitle.Transcript
	sbCh         <-chan *storyboard.StoryboardData
	sbSegmentsCh <-chan []sponsorblock.Segment
}

func fetchSliderData(ctx context.Context, links, localFiles []string) *sliderData {
	duration, info := probeDuration(ctx, links, localFiles)
	if duration <= 0 {
		return nil
	}

	// Chapters are fast (parsed from already-loaded metadata)
	var markers []slider.ChapterMarker
	if info != nil {
		chapters := download.ExtractChapters(info)
		markers = make([]slider.ChapterMarker, 0, len(chapters))
		for _, ch := range chapters {
			markers = append(markers, slider.ChapterMarker{StartTime: ch.StartTime, Title: ch.Title})
		}
	}

	transcriptCh := make(chan subtitle.Transcript, 1)
	sbSegmentsCh := make(chan []sponsorblock.Segment, 1)

	d := &sliderData{
		duration:     duration,
		markers:      markers,
		transcriptCh: transcriptCh,
		sbSegmentsCh: sbSegmentsCh,
	}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(transcriptCh)
		if info == nil || len(links) == 0 || sponsorblock.ExtractVideoID(links[0]) == "" {
			return nil
		}
		t, err := subtitle.FetchFromMetadata(gctx, info)
		if err != nil {
			log.Debug("Failed to fetch subtitles", "err", err)
			return nil
		}
		if len(t) > 0 {
			transcriptCh <- t
		}
		return nil
	})

	g.Go(func() error {
		defer close(sbSegmentsCh)
		if len(links) == 0 {
			return nil
		}
		videoID := sponsorblock.ExtractVideoID(links[0])
		if videoID == "" {
			return nil
		}
		segs, err := sponsorblock.GetSegments(gctx, videoID)
		if err != nil {
			log.Debug("SponsorBlock fetch failed", "err", err)
			return nil
		}
		if len(segs) > 0 {
			sbSegmentsCh <- segs
		}
		return nil
	})

	if info != nil {
		d.sbCh = storyboard.StartFetch(ctx, info)
	}

	go func() { _ = g.Wait() }()

	return d
}

func runSlider(data *sliderData, gifEnabled bool) (*slider.TrimResult, error) {
	return slider.Run(data.duration, data.transcriptCh, data.sbCh, data.sbSegmentsCh, gifEnabled, data.markers...)
}

func probeDuration(ctx context.Context, links, localFiles []string) (float64, *ytdlp.ExtractedInfo) {
	if len(localFiles) > 0 {
		d, err := convert.ProbeDuration(ctx, localFiles[0])
		if err != nil {
			log.Error("Failed to get duration from file", "file", localFiles[0], "err", err)
		} else if d > 0 {
			return d, nil
		}
	}

	if len(links) > 0 {
		info, err := tui.RunWithSpinnerResult(ctx, "Fetching metadata...", func() (*ytdlp.ExtractedInfo, error) {
			return download.FetchMetadata(ctx, links[0])
		})
		if err != nil {
			log.Error("Failed to fetch metadata from URL", "url", links[0], "err", err)
			return 0, nil
		}
		if info.Duration != nil {
			return *info.Duration, info
		}
	}

	return 0, nil
}
