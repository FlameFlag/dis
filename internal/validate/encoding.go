package validate

import (
	"dis/internal/config"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
)

const (
	CRFMin            = 6
	CRFMax            = 63
	CRFMinRecommended = 22
	CRFMaxRecommended = 38
	CRFDefault        = 25

	AudioBitrateMinRecommended = 128
	AudioBitrateMaxRecommended = 192

	DefaultAudioBitrate = 128
)

// Crf checks the CRF value is within valid range (6-63), warns if outside recommended (22-38).
func Crf(crf int) error {
	if crf < CRFMin || crf > CRFMax {
		return fmt.Errorf("CRF value must be between %d and %d (recommended: %d-%d)", CRFMin, CRFMax, CRFMinRecommended, CRFMaxRecommended)
	}
	if crf < CRFMinRecommended {
		log.Warn("CRF value is below the recommended minimum. This may result in very large files.", "crf", crf, "min_recommended", CRFMinRecommended)
	}
	if crf > CRFMaxRecommended {
		log.Warn("CRF value is above the recommended maximum. This may result in poor quality.", "crf", crf, "max_recommended", CRFMaxRecommended)
	}
	return nil
}

// AudioBitrate checks that the audio bitrate is even and warns if outside the recommended range.
func AudioBitrate(bitrate int) error {
	if bitrate == 0 {
		return nil
	}
	if bitrate%2 != 0 {
		return fmt.Errorf("audio bitrate must be a multiple of 2")
	}
	if bitrate < AudioBitrateMinRecommended || bitrate > AudioBitrateMaxRecommended {
		log.Warn("Audio bitrate values outside the recommended range are not generally recommended.",
			"bitrate", bitrate, "min_recommended", AudioBitrateMinRecommended, "max_recommended", AudioBitrateMaxRecommended)
	}
	return nil
}

// Speed checks that the speed multiplier is within valid range.
func Speed(speed float64) error { return floatRange("speed", speed, 1.0, 4.0) }

// VideoCodec checks whether the given codec string is known.
func VideoCodec(codec string) error {
	if codec == "" {
		return nil
	}
	if !config.IsValidCodec(codec) {
		return fmt.Errorf("invalid video codec: %s. Valid options are: %s", codec, strings.Join(config.CodecNames(), ", "))
	}
	return nil
}

// TargetSize checks whether the target size string is parseable.
func TargetSize(size string) error {
	if size == "" {
		return nil
	}
	bytes, err := config.ParseSize(size)
	if err != nil {
		return fmt.Errorf("invalid target size: %w", err)
	}
	if bytes < 1_000_000 {
		return fmt.Errorf("target size must be at least 1MB")
	}
	return nil
}

// Preset checks whether the preset name is valid.
func Preset(name string, userPresets map[string]config.Preset) error {
	if name == "" {
		return nil
	}
	_, err := config.ResolvePreset(name, userPresets)
	return err
}
