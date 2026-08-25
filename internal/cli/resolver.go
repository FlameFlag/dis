package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/4evy/dis/internal/app"
	"github.com/4evy/dis/internal/browsercookies"
	"github.com/4evy/dis/internal/config"
	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/timecode"
)

const (
	TrimInteractive = "interactive"
)

// Values contains raw Cobra flag values before precedence and parsing.
type Values struct {
	Inputs             []string
	Output             string
	CRF                int
	Resolution         string
	Trim               string
	VideoCodec         string
	AudioBitrate       int
	MultiThread        bool
	Random             bool
	Sponsor            bool
	Chapter            bool
	NoConvert          bool
	Preset             string
	TargetSize         string
	Copy               bool
	GIF                bool
	GIFFPS             int
	GIFWidth           int
	GIFQuality         int
	GIFLossyQuality    int
	GIFMotionQuality   int
	GIFSpeed           float64
	Speed              float64
	CookiesFromBrowser string
}

// DefaultValues returns built-in CLI defaults.
func DefaultValues() Values {
	return Values{
		Output: ".", CRF: convert.CRFDefault,
		VideoCodec: convert.DefaultVideoCodec, MultiThread: true,
		GIFFPS: convert.GIFFPSDefault, GIFWidth: convert.GIFWidthDefault,
		GIFQuality:       convert.GIFQualityDefault,
		GIFLossyQuality:  convert.GIFQualityDefault,
		GIFMotionQuality: convert.GIFQualityDefault,
		GIFSpeed:         convert.DefaultPlaybackSpeed,
		Speed:            convert.DefaultPlaybackSpeed,
	}
}

// Resolve applies flag, preset, config, and built-in precedence, then parses
// user-facing values exactly once.
func Resolve(
	values Values,
	explicit map[string]bool,
	file *config.File,
	baseDirectory string,
) (app.Options, []string, error) {
	if file == nil {
		file = &config.File{}
	}
	if err := validateExplicitGroups(explicit); err != nil {
		return app.Options{}, nil, err
	}
	presetName := choose(
		values.Preset,
		explicit[flagPreset],
		"",
		false,
		file.Preset,
		"",
	)
	var preset config.Preset
	if presetName != "" {
		resolved, err := config.ResolvePreset(presetName, file.Presets)
		if err != nil {
			return app.Options{}, nil, err
		}
		preset = *resolved
	}

	output := choose(
		values.Output,
		explicit[flagOutput],
		"",
		false,
		file.Output,
		".",
	)
	if output == "." {
		output = baseDirectory
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(baseDirectory, output)
	}
	output, err := filepath.Abs(filepath.Clean(output))
	if err != nil {
		return app.Options{}, nil, fmt.Errorf("resolve output path: %w", err)
	}

	codecName := choose(
		values.VideoCodec,
		explicit[flagVideoCodec],
		preset.VideoCodec,
		preset.VideoCodec != "",
		file.VideoCodec,
		convert.DefaultVideoCodec,
	)
	codec, err := convert.ParseCodec(codecName)
	if err != nil {
		return app.Options{}, nil, err
	}
	resolutionName := choose(
		values.Resolution,
		explicit[flagResolution],
		preset.Resolution,
		preset.Resolution != "",
		file.Resolution,
		"",
	)
	resolution, err := convert.ParseResolution(resolutionName)
	if err != nil {
		return app.Options{}, nil, err
	}
	targetSize := choose(
		values.TargetSize,
		explicit[flagTargetSize],
		preset.TargetSize,
		preset.TargetSize != "",
		file.TargetSize,
		"",
	)
	var targetBytes int64
	if targetSize != "" {
		targetBytes, err = convert.ParseSize(targetSize)
		if err != nil {
			return app.Options{}, nil, fmt.Errorf("invalid target size: %w", err)
		}
	}
	cookiesFromBrowser := choose(
		values.CookiesFromBrowser,
		explicit[flagCookiesFromBrowser],
		"",
		false,
		file.CookiesFromBrowser,
		"",
	)
	if cookiesFromBrowser != "" {
		if _, parseErr := browsercookies.Parse(cookiesFromBrowser); parseErr != nil {
			return app.Options{}, nil, fmt.Errorf(
				"invalid browser cookie source: %w",
				parseErr,
			)
		}
	}

	trim, err := resolveTrim(values.Trim)
	if err != nil {
		return app.Options{}, nil, err
	}
	inputs := make([]string, len(values.Inputs))
	for index, input := range values.Inputs {
		if app.IsURL(input) || filepath.IsAbs(input) {
			inputs[index] = input
			continue
		}
		inputs[index] = filepath.Join(baseDirectory, input)
	}
	conversion := convert.Options{
		OutputDir: output,
		CRF: choose(
			values.CRF,
			explicit[flagCRF],
			preset.Crf,
			preset.Crf != 0,
			file.Crf,
			convert.CRFDefault,
		),
		Resolution: resolution,
		Codec:      codec,
		AudioBitrate: choose(
			values.AudioBitrate,
			explicit[flagAudioBitrate],
			preset.AudioBitrate,
			preset.AudioBitrate != 0,
			file.AudioBitrate,
			0,
		),
		MultiThread: choose(
			values.MultiThread,
			explicit[flagMultiThread],
			false,
			false,
			file.MultiThread,
			true,
		),
		RandomName:  values.Random,
		TargetBytes: targetBytes,
		MaxDuration: preset.MaxDuration,
		Speed:       values.Speed,
	}
	gifOptions := convert.GIFOptions{
		FPS: values.GIFFPS, Width: values.GIFWidth,
		Quality: values.GIFQuality, LossyQuality: values.GIFLossyQuality,
		MotionQuality: values.GIFMotionQuality, Speed: values.GIFSpeed,
	}
	options := app.Options{
		Inputs: inputs,
		Download: app.DownloadOptions{
			RemoveSponsors:     values.Sponsor,
			CookiesFromBrowser: cookiesFromBrowser,
		},
		Conversion:   conversion,
		GIFOptions:   gifOptions,
		Trim:         trim,
		Chapter:      values.Chapter,
		NoConvert:    values.NoConvert,
		Copy:         values.Copy,
		GIF:          values.GIF,
		GIFAvailable: true,
	}
	warnings, validationErr := options.Validate()
	return options, warnings, validationErr
}

func choose[T any](
	flagValue T,
	flagSet bool,
	presetValue T,
	presetSet bool,
	configValue *T,
	defaultValue T,
) T {
	if flagSet {
		return flagValue
	}
	if presetSet {
		return presetValue
	}
	if configValue != nil {
		return *configValue
	}
	return defaultValue
}

func resolveTrim(input string) (app.TrimOptions, error) {
	if input == "" {
		return app.TrimOptions{}, nil
	}
	if input == TrimInteractive {
		return app.TrimOptions{Interactive: true}, nil
	}
	startText, endText, ok := strings.Cut(input, "-")
	if !ok {
		return app.TrimOptions{}, fmt.Errorf(
			"invalid trim range %q: expected format START-END (e.g. 10-20, 1:30-2:45)",
			input,
		)
	}
	start, err := timecode.Parse(startText)
	if err != nil {
		return app.TrimOptions{}, fmt.Errorf(
			"invalid trim range %q: invalid start time %q: %w",
			input,
			startText,
			err,
		)
	}
	end, err := timecode.Parse(endText)
	if err != nil {
		return app.TrimOptions{}, fmt.Errorf(
			"invalid trim range %q: invalid end time %q: %w",
			input,
			endText,
			err,
		)
	}
	clip, err := media.NewClipBounds(start, end)
	if err != nil {
		return app.TrimOptions{}, fmt.Errorf("invalid trim range %q: %w", input, err)
	}
	return app.TrimOptions{Clips: []media.Clip{clip}}, nil
}

func validateExplicitGroups(explicit map[string]bool) error {
	var errs []error
	for _, group := range mutuallyExclusiveFlagGroups {
		if explicit[group[0]] && explicit[group[1]] {
			errs = append(errs, fmt.Errorf(
				"if any flags in the group [%s] are set none of the others can be; [%s] were all set",
				strings.Join(group, " "),
				strings.Join(group, " "),
			))
		}
	}
	return errors.Join(errs...)
}
