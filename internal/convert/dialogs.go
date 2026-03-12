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

// checkSkipConversion asks the user whether to skip conversion.
// Returns true if conversion should be skipped.
func checkSkipConversion(s *config.Settings, trimSettings *config.TrimSettings) bool {
	if s.NoConvert {
		if trimSettings != nil {
			log.Warn("--no-convert: trimming will not be applied (requires FFmpeg)")
		}
		return true
	}

	var skip bool
	err := huh.NewConfirm().
		Title("Skip conversion?").
		Description("The file will be copied as-is without re-encoding.").
		Value(&skip).
		Run()
	if err != nil {
		return false
	}

	if skip && trimSettings != nil {
		log.Warn("Skipping conversion will also skip trimming for local files")
	}

	return skip
}
