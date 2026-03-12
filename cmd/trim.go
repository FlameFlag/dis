package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/download"
	"dis/internal/sponsorblock"
	"dis/internal/subtitle"
	"dis/internal/tui"
	"dis/internal/tui/slider"
	"dis/internal/util"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
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

func getTrimSettings(ctx context.Context, links, localFiles []string) (*slider.TrimResult, error) {
	duration, info := probeDuration(ctx, links, localFiles)
	if duration <= 0 {
		log.Warn("Could not determine a valid video duration. Skipping trim.")
		return nil, nil
	}

	markers, transcript := extractSliderData(ctx, info, links)
	silenceCh := startSilenceDetection(ctx, links)
	waveformCh := startWaveformExtraction(ctx, links)
	sbSegments := fetchSponsorSegments(ctx, links)

	return slider.Run(duration, transcript, silenceCh, waveformCh, sbSegments, settings.GIF, markers...)
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

// runMultiSegmentDownload handles downloads when the user selected non-contiguous segments.
func runMultiSegmentDownload(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	choice, err := promptSegmentChoice(len(segments))
	if err != nil {
		return err
	}

	switch choice {
	case "split":
		return runSplitSegments(ctx, s, links, localFiles, segments)
	case "combine":
		return runCombineSegments(ctx, s, links, localFiles, segments)
	case "span":
		return runSpanSegments(ctx, s, links, localFiles, segments)
	}
	return nil
}

func promptSegmentChoice(count int) (string, error) {
	var choice string
	err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Your selection has %d separate segments. How should they be handled?", count)).
		Options(
			huh.NewOption(fmt.Sprintf("Split into %d separate videos", count), "split"),
			huh.NewOption("Combine into one video (skip gaps)", "combine"),
			huh.NewOption("One video including gaps", "span"),
		).
		Value(&choice).
		Run()
	return choice, err
}

func runSplitSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return err
		}

		trimSettings := seg
		log.Info("Processing segment", "index", i+1, "start", util.FormatDurationShort(seg.Start), "end", util.FormatDurationShort(seg.End()))

		for _, link := range links {
			result, err := downloadWithProgress(ctx, fmt.Sprintf("Downloading segment %d...", i+1), link, s, &trimSettings)
			if errors.Is(err, tui.ErrUserCancelled) {
				return nil
			}
			if err != nil {
				log.Error("Failed to download video", "url", link, "err", err)
				continue
			}
			tempDirs = append(tempDirs, result.TempDir)
			if err := convertDownloaded(ctx, s, result); err != nil {
				log.Error("Failed to convert video", "path", result.OutputPath, "err", err)
			}
		}

		for _, path := range localFiles {
			if err := convert.ConvertVideo(ctx, path, s, &trimSettings, ""); err != nil {
				log.Error("Failed to convert video", "path", path, "err", err)
			}
		}
	}
	return nil
}

func runCombineSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	spanTrim := spanFromSegments(segments)

	relativeSegments := make([]config.TrimSettings, len(segments))
	for i, seg := range segments {
		relativeSegments[i] = config.TrimSettings{
			Start:    seg.Start - spanTrim.Start,
			Duration: seg.Duration,
		}
	}

	for _, link := range links {
		result, err := downloadWithProgress(ctx, "Downloading...", link, s, spanTrim)
		if errors.Is(err, tui.ErrUserCancelled) {
			return nil
		}
		if err != nil {
			log.Error("Failed to download video", "url", link, "err", err)
			continue
		}
		tempDirs = append(tempDirs, result.TempDir)
		if err := convert.ConcatSegments(ctx, result.OutputPath, s, relativeSegments, result.UploadDate); err != nil {
			log.Error("Failed to concatenate segments", "err", err)
		}
	}

	for _, path := range localFiles {
		if err := convert.ConcatSegments(ctx, path, s, segments, ""); err != nil {
			log.Error("Failed to concatenate segments", "path", path, "err", err)
		}
	}
	return nil
}

func runSpanSegments(ctx context.Context, s *config.Settings, links, localFiles []string, segments []config.TrimSettings) error {
	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	spanTrim := spanFromSegments(segments)

	for _, link := range links {
		result, err := downloadWithProgress(ctx, "Downloading...", link, s, spanTrim)
		if errors.Is(err, tui.ErrUserCancelled) {
			return nil
		}
		if err != nil {
			log.Error("Failed to download video", "url", link, "err", err)
			continue
		}
		tempDirs = append(tempDirs, result.TempDir)
		if err := convertDownloaded(ctx, s, result); err != nil {
			log.Error("Failed to convert video", "path", result.OutputPath, "err", err)
		}
	}

	for _, path := range localFiles {
		if err := convert.ConvertVideo(ctx, path, s, spanTrim, ""); err != nil {
			log.Error("Failed to convert video", "path", path, "err", err)
		}
	}
	return nil
}

func spanFromSegments(segments []config.TrimSettings) *config.TrimSettings {
	return &config.TrimSettings{
		Start:    segments[0].Start,
		Duration: segments[len(segments)-1].End() - segments[0].Start,
	}
}
