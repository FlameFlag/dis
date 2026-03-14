package validate

import (
	"dis/internal/config"
	"dis/internal/util"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
)

// Inputs checks that each input is either a valid URL or an existing media file.
func Inputs(inputs []string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no input provided")
	}

	for _, input := range inputs {
		isFile := util.FileExists(input)
		isURL := util.IsValidURL(input)

		if !isFile && !isURL {
			return fmt.Errorf("invalid input file or link: %s", input)
		}

		if !isFile {
			continue
		}
		ext := filepath.Ext(input)
		mtype := mime.TypeByExtension(ext)
		if mtype == "" {
			log.Warn("Could not determine content type for file", "input", input)
			continue
		}
		if !strings.HasPrefix(mtype, "video/") && !strings.HasPrefix(mtype, "audio/") {
			return fmt.Errorf("input file is not a recognized video/audio type: %s (type: %s)", input, mtype)
		}
	}
	return nil
}

// Output checks that the output directory exists.
func Output(output string) error {
	if output == "" {
		return nil
	}
	info, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("output directory does not exist: %s", output)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", output)
	}
	return nil
}

const (
	CRFMin            = 6
	CRFMax            = 63
	CRFMinRecommended = 22
	CRFMaxRecommended = 38
	CRFDefault        = 25

	AudioBitrateMinRecommended = 128
	AudioBitrateMaxRecommended = 192
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

// Resolution checks whether the given resolution string is valid.
func Resolution(res string) error {
	if res == "" {
		return nil
	}

	cleaned := strings.TrimSuffix(strings.ToLower(res), "p")
	val, err := strconv.Atoi(cleaned)
	if err != nil {
		return fmt.Errorf("invalid resolution: %s", res)
	}

	if slices.Contains(config.ValidResolutions, val) {
		return nil
	}

	validStrs := make([]string, len(config.ValidResolutions))
	for i, r := range config.ValidResolutions {
		validStrs[i] = fmt.Sprintf("%dp", r)
	}
	return fmt.Errorf("invalid resolution: %s. Valid options are: %s", res, strings.Join(validStrs, ", "))
}

// GIFFps checks that the GIF frame rate is within valid range.
func GIFFps(fps int) error {
	if fps < 1 || fps > 50 {
		return fmt.Errorf("GIF fps must be between 1 and 50 (got %d)", fps)
	}
	return nil
}

// GIFWidth checks that the GIF width is within valid range.
func GIFWidth(width int) error {
	if width < 1 || width > 3840 {
		return fmt.Errorf("GIF width must be between 1 and 3840 (got %d)", width)
	}
	return nil
}

// GIFQuality checks that the GIF quality is within valid range.
func GIFQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("GIF quality must be between 1 and 100 (got %d)", quality)
	}
	return nil
}

// GIFLossyQuality checks that the GIF lossy quality is within valid range.
func GIFLossyQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("GIF lossy quality must be between 1 and 100 (got %d)", quality)
	}
	return nil
}

// GIFMotionQuality checks that the GIF motion quality is within valid range.
func GIFMotionQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("GIF motion quality must be between 1 and 100 (got %d)", quality)
	}
	return nil
}

// GIFSpeed checks that the GIF speed multiplier is within valid range.
func GIFSpeed(speed float64) error {
	if speed < 1.0 || speed > 4.0 {
		return fmt.Errorf("GIF speed must be between 1.0 and 4.0 (got %.1f)", speed)
	}
	return nil
}

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
