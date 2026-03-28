package convert

import (
	"dis/internal/config"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
)

func shouldRetry(s *config.Settings, outputPath string, info *MediaInfo) (bool, error) {
	log.Warn("The resulting file is larger than the original.")

	deleteAndRetry, err := confirmPrompt("Delete and try again with better settings?")
	if err != nil || !deleteAndRetry {
		return false, err
	}

	_ = os.Remove(outputPath)
	log.Info("Deleted the oversized converted video.")

	changed := false

	if wantRes, err := confirmPrompt("Would you like to change the resolution?"); err != nil {
		return false, err
	} else if wantRes {
		changed = changeResolution(s, info) || changed
	}

	if wantCrf, err := confirmPrompt("Would you like to enter a new CRF value?"); err != nil {
		return false, err
	} else if wantCrf {
		changed = changeCrfValue(s) || changed
	}

	return changed, nil
}

// confirmPrompt shows a yes/no confirmation and returns the result.
func confirmPrompt(title string) (bool, error) {
	var result bool
	err := huh.NewConfirm().Title(title).Value(&result).Run()
	return result, err
}

func changeResolution(s *config.Settings, info *MediaInfo) bool {
	maxDim := max(info.Width, info.Height)

	// Find current resolution bucket
	currentRes := closestResolution(maxDim)
	var lowerRes []string
	for _, r := range config.ValidResolutions {
		if r < currentRes {
			lowerRes = append(lowerRes, fmt.Sprintf("%dp", r))
		}
	}

	if len(lowerRes) == 0 {
		log.Warn("No lower resolutions available.")
		return false
	}

	var chosen string
	opts := make([]huh.Option[string], len(lowerRes))
	for i, r := range lowerRes {
		opts[i] = huh.NewOption(r, r)
	}

	err := huh.NewSelect[string]().
		Title("Select a lower resolution:").
		Options(opts...).
		Value(&chosen).
		Run()
	if err != nil {
		return false
	}

	s.Resolution = chosen
	return true
}

func closestResolution(dimension int) int {
	resolutions := config.ValidResolutions
	closest := resolutions[0]
	for _, r := range resolutions {
		if r <= dimension {
			closest = r
		}
	}
	return closest
}

func changeCrfValue(s *config.Settings) bool {
	for {
		var input string
		err := huh.NewInput().
			Title("Enter new CRF value (higher = smaller file):").
			Value(&input).
			Run()
		if err != nil {
			return false
		}

		val, err := strconv.Atoi(input)
		if err != nil {
			log.Warn("Invalid CRF value, please enter a number.")
			continue
		}

		if val <= s.Crf {
			log.Warn("Please enter a value higher than the current CRF.", "current", s.Crf)
			continue
		}

		s.Crf = val
		return true
	}
}

// GIFSpeedGoBack is a sentinel value indicating the user wants to go back to the trim slider.
const GIFSpeedGoBack = -1.0

// PromptGIFSpeed asks the user to pick a playback speed for the GIF.
func PromptGIFSpeed(duration float64) (float64, error) {
	desc := fmt.Sprintf("GIF is %.0fs - speeding up reduces file size", duration)
	if duration >= 6 {
		desc = fmt.Sprintf("GIF is %.0fs - long GIFs produce large files, speeding up helps", duration)
	}

	opts := []huh.Option[float64]{
		huh.NewOption("1x (no change)", 1.0),
		huh.NewOption("1.5x", 1.5),
		huh.NewOption("2x", 2.0),
	}
	if duration >= 12 {
		opts = append(opts, huh.NewOption("Go back and trim shorter", GIFSpeedGoBack))
	}

	var choice float64
	err := huh.NewSelect[float64]().
		Title("Speed up GIF playback?").
		Description(desc).
		Options(opts...).
		Value(&choice).
		Run()
	return choice, err
}

// checkSkipConversion asks the user whether to skip conversion.
// Returns true if conversion should be skipped.
func checkSkipConversion(s *config.Settings, trimSettings *config.TrimSettings) bool {
	if s.NoConvert {
		if trimSettings != nil {
			log.Warn("--no-convert: trimming will not be applied (requires FFmpeg)")
		}
		return true
	}

	var convert bool
	err := huh.NewConfirm().
		Title("Convert this file?").
		Description("Re-encode with your current settings. Choose No to copy the file as-is.").
		Value(&convert).
		Run()
	if err != nil {
		return false
	}

	if !convert && trimSettings != nil {
		log.Warn("Skipping conversion will also skip trimming for local files")
	}

	return !convert
}
