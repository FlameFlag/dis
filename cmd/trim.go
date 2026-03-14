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
	duration   float64
	markers    []slider.ChapterMarker
	transcript subtitle.Transcript
	silenceCh  chan []subtitle.SilenceInterval
	waveformCh chan []subtitle.WaveformSample
	sbCh       <-chan *storyboard.StoryboardData
	sbSegments []sponsorblock.Segment
}

func fetchSliderData(ctx context.Context, links, localFiles []string) *sliderData {
	duration, info := probeDuration(ctx, links, localFiles)
	if duration <= 0 {
		return nil
	}

	markers, transcript := extractSliderData(ctx, info, links)

	return &sliderData{
		duration:   duration,
		markers:    markers,
		transcript: transcript,
		silenceCh:  startSilenceDetection(ctx, links),
		waveformCh: startWaveformExtraction(ctx, links),
		sbCh:       startStoryboardFetch(ctx, info),
		sbSegments: fetchSponsorSegments(ctx, links),
	}
}

func runSlider(data *sliderData, gifEnabled bool) (*slider.TrimResult, error) {
	return slider.Run(data.duration, data.transcript, data.silenceCh, data.waveformCh, data.sbCh, data.sbSegments, gifEnabled, data.markers...)
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

func extractSliderData(ctx context.Context, info *ytdlp.ExtractedInfo, links []string) ([]slider.ChapterMarker, subtitle.Transcript) {
	if info == nil {
		return nil, nil
	}

	chapters := download.ExtractChapters(info)
	markers := make([]slider.ChapterMarker, 0, len(chapters))
	for _, ch := range chapters {
		markers = append(markers, slider.ChapterMarker{
			StartTime: ch.StartTime,
			Title:     ch.Title,
		})
	}

	var transcript subtitle.Transcript
	if len(links) > 0 && sponsorblock.ExtractVideoID(links[0]) != "" {
		t, err := subtitle.FetchFromMetadata(ctx, info)
		if err != nil {
			log.Debug("Failed to fetch subtitles", "err", err)
		} else {
			transcript = t
		}
	}

	return markers, transcript
}

func startSilenceDetection(ctx context.Context, links []string) chan []subtitle.SilenceInterval {
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

func startWaveformExtraction(ctx context.Context, links []string) chan []subtitle.WaveformSample {
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

func fetchSponsorSegments(ctx context.Context, links []string) []sponsorblock.Segment {
	if len(links) == 0 {
		return nil
	}

	videoID := sponsorblock.ExtractVideoID(links[0])
	if videoID == "" {
		return nil
	}

	segs, err := sponsorblock.GetSegments(ctx, videoID)
	if err != nil {
		log.Debug("SponsorBlock fetch failed", "err", err)
		return nil
	}
	return segs
}
