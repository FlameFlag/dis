package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/download"
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/subtitle"
	"dis/internal/tui"
	"dis/internal/tui/slider"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	ytdlp "github.com/lrstanley/go-ytdlp"
)

func resolveTrimWithSpeedPrompt(ctx context.Context, s *config.Settings, links, localFiles []string) ([]config.TrimSettings, error) {
	if s.Trim == "" {
		return promptGIFSpeedIfNeeded(s, nil, ctx, localFiles)
	}

	if s.Trim != config.TrimInteractive {
		ts, err := parseTrimRange(s.Trim)
		if err != nil {
			return nil, fmt.Errorf("invalid trim range %q: %w", s.Trim, err)
		}
		segments := []config.TrimSettings{*ts}
		return promptGIFSpeedIfNeeded(s, segments, ctx, localFiles)
	}

	// Interactive: fetch data once, loop only re-runs the slider on go-back
	data := fetchSliderData(ctx, links, localFiles)
	if data == nil {
		log.Warn("Could not determine a valid video duration. Skipping trim.")
		return nil, nil
	}

	for {
		result, err := runSlider(data, s.GIF)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, tui.ErrUserCancelled
		}
		s.GIF = result.GIF
		if result.GIF {
			s.GIFSpeed = result.Speed
		} else {
			s.Speed = result.Speed
		}

		segments, err := promptGIFSpeedIfNeeded(s, result.Segments, ctx, localFiles)
		if err != nil {
			return nil, err
		}
		if segments != nil {
			return segments, nil
		}
		// nil segments means go-back — re-run slider
		s.GIFSpeed = 0
	}
}

// promptGIFSpeedIfNeeded shows the speed prompt for long GIFs.
// Returns nil segments (no error) as a signal to go back to the slider.
func promptGIFSpeedIfNeeded(s *config.Settings, segments []config.TrimSettings, ctx context.Context, localFiles []string) ([]config.TrimSettings, error) {
	if !s.GIF || s.GIFSpeed > 1.0 {
		return segments, nil
	}

	var gifDuration float64
	for _, seg := range segments {
		gifDuration += seg.Duration
	}
	if gifDuration <= 0 && len(localFiles) > 0 {
		if d, err := convert.ProbeDuration(ctx, localFiles[0]); err == nil {
			gifDuration = d
		}
	}
	if gifDuration < 4 {
		return segments, nil
	}

	speed, err := convert.PromptGIFSpeed(gifDuration)
	if err != nil {
		return segments, nil
	}
	if speed == convert.GIFSpeedGoBack {
		return nil, nil
	}
	if speed > 1.0 {
		s.GIFSpeed = speed
	}
	return segments, nil
}

// parseTrimRange parses a "START-END" range string into TrimSettings.
func parseTrimRange(input string) (*config.TrimSettings, error) {
	parts := strings.SplitN(input, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected format START-END (e.g. 10-20, 1:30-2:45)")
	}

	if strings.Contains(parts[0], ":") || strings.Contains(parts[1], ":") {
		var found bool
		for i := 1; i < len(input); i++ {
			if input[i] != '-' {
				continue
			}
			left, right := input[:i], input[i+1:]
			_, errL := util.ParseTimeValue(left)
			_, errR := util.ParseTimeValue(right)
			if errL == nil && errR == nil {
				parts[0], parts[1] = left, right
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("expected format START-END (e.g. 10-20, 1:30-2:45)")
		}
	}

	start, err := util.ParseTimeValue(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start time %q: %w", parts[0], err)
	}

	end, err := util.ParseTimeValue(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end time %q: %w", parts[1], err)
	}

	if end <= start {
		return nil, fmt.Errorf("end time (%.2f) must be greater than start time (%.2f)", end, start)
	}

	return &config.TrimSettings{
		Start:    start,
		Duration: end - start,
	}, nil
}

// sliderData holds pre-fetched data needed to run the trim slider.
type sliderData struct {
	duration     float64
	markers      []slider.ChapterMarker
	transcriptCh <-chan subtitle.Transcript
	silenceCh    <-chan []subtitle.SilenceInterval
	waveformCh   <-chan []subtitle.WaveformSample
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

	return &sliderData{
		duration:     duration,
		markers:      markers,
		transcriptCh: startTranscriptFetch(ctx, info, links),
		silenceCh:    startSilenceDetection(ctx, links),
		waveformCh:   startWaveformExtraction(ctx, links),
		sbCh:         startStoryboardFetch(ctx, info),
		sbSegmentsCh: startSponsorBlockFetch(ctx, links),
	}
}

func runSlider(data *sliderData, gifEnabled bool) (*slider.TrimResult, error) {
	return slider.Run(data.duration, data.transcriptCh, data.silenceCh, data.waveformCh, data.sbCh, data.sbSegmentsCh, gifEnabled, data.markers...)
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

func startTranscriptFetch(ctx context.Context, info *ytdlp.ExtractedInfo, links []string) <-chan subtitle.Transcript {
	ch := make(chan subtitle.Transcript, 1)
	go func() {
		defer close(ch)
		if info == nil || len(links) == 0 || sponsorblock.ExtractVideoID(links[0]) == "" {
			return
		}
		t, err := subtitle.FetchFromMetadata(ctx, info)
		if err != nil {
			log.Debug("Failed to fetch subtitles", "err", err)
			return
		}
		if len(t) > 0 {
			ch <- t
		}
	}()
	return ch
}

func startSponsorBlockFetch(ctx context.Context, links []string) <-chan []sponsorblock.Segment {
	ch := make(chan []sponsorblock.Segment, 1)
	go func() {
		defer close(ch)
		if len(links) == 0 {
			return
		}
		videoID := sponsorblock.ExtractVideoID(links[0])
		if videoID == "" {
			return
		}
		segs, err := sponsorblock.GetSegments(ctx, videoID)
		if err != nil {
			log.Debug("SponsorBlock fetch failed", "err", err)
			return
		}
		if len(segs) > 0 {
			ch <- segs
		}
	}()
	return ch
}

func startSilenceDetection(ctx context.Context, links []string) <-chan []subtitle.SilenceInterval {
	if len(links) == 0 {
		return nil
	}

	ch := make(chan []subtitle.SilenceInterval, 1)
	go func() {
		defer close(ch)
		sil, err := subtitle.DetectSilence(ctx, links[0])
		if err != nil {
			log.Debug("Silence detection failed", "err", err)
			return
		}
		if len(sil) > 0 {
			ch <- sil
		}
	}()
	return ch
}

func startWaveformExtraction(ctx context.Context, links []string) <-chan []subtitle.WaveformSample {
	if len(links) == 0 {
		return nil
	}

	ch := make(chan []subtitle.WaveformSample, 1)
	go func() {
		defer close(ch)
		samples, err := subtitle.ExtractWaveform(ctx, links[0], 200)
		if err != nil {
			log.Debug("Waveform extraction failed", "err", err)
			return
		}
		if len(samples) > 0 {
			ch <- samples
		}
	}()
	return ch
}

func startStoryboardFetch(ctx context.Context, info *ytdlp.ExtractedInfo) <-chan *storyboard.StoryboardData {
	if info == nil {
		return nil
	}
	return storyboard.StartFetch(ctx, info)
}

