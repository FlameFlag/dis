package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
)

// errGoBack signals that the GIF speed prompt was dismissed with "go back",
// asking the caller to re-run the slider.
var errGoBack = errors.New("go back to slider")

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
		if errors.Is(err, errGoBack) {
			s.GIFSpeed = 0
			continue
		}
		if err != nil {
			return nil, err
		}
		return segments, nil
	}
}

// promptGIFSpeedIfNeeded shows the speed prompt for long GIFs.
// Returns errGoBack when the user chose "go back" at the prompt.
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
		return nil, errGoBack
	}
	if speed > 1.0 {
		s.GIFSpeed = speed
	}
	return segments, nil
}
