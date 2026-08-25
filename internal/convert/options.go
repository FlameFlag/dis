package convert

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

const (
	CRFMin            = 6
	CRFMax            = 63
	CRFMinRecommended = 22
	CRFMaxRecommended = 38
	CRFDefault        = 25

	AudioBitrateMinRecommended = 128
	AudioBitrateMaxRecommended = 192
	DefaultAudioBitrate        = 128
	AudioBitrateStep           = 2

	SpeedMin = 1.0
	SpeedMax = 4.0

	MinimumTargetSize = humanize.MByte

	GIFFPSMin     = 1
	GIFFPSMax     = 50
	GIFFPSDefault = 15

	GIFWidthMin     = 1
	GIFWidthMax     = 3_840
	GIFWidthDefault = 480

	GIFQualityMin     = 1
	GIFQualityMax     = 100
	GIFQualityDefault = 80

	DefaultPlaybackSpeed = 1.0
)

// Codec identifies a supported video encoder and container pairing.
type Codec int

const (
	CodecH264 Codec = iota
	CodecHEVC
	CodecVP8
	CodecVP9
	CodecAV1
)

type codecConfig struct {
	name        string
	aliases     []string
	pixelFormat string
	audioCodec  string
	extension   string
	webm        bool
	parameters  func(bool, float64) []string
}

var codecs = map[Codec]codecConfig{
	CodecH264: {
		name: "libx264", aliases: []string{"h264", "libx264"},
		pixelFormat: pixelFormatYUV420, audioCodec: "aac", extension: ".mp4",
		parameters: x26xParameters,
	},
	CodecHEVC: {
		name: "libx265", aliases: []string{"h265", "hevc", "libx265"},
		pixelFormat: pixelFormatYUV420, audioCodec: "aac", extension: ".mp4",
		parameters: x26xParameters,
	},
	CodecVP8: {
		name: "libvpx", aliases: []string{"vp8", "libvpx"},
		pixelFormat: pixelFormatYUV420, audioCodec: audioCodecOpusV2, extension: webmExtension,
		webm: true,
	},
	CodecVP9: {
		name: "libvpx-vp9", aliases: []string{"vp9", "libvpx-vp9"},
		pixelFormat: pixelFormatYUV420, audioCodec: audioCodecOpusV2, extension: webmExtension,
		webm: true, parameters: vp9Parameters,
	},
	CodecAV1: {
		name: "libaom-av1", aliases: []string{"av1", "libaom-av1"},
		pixelFormat: "yuv420p10le", audioCodec: audioCodecOpusV2, extension: webmExtension,
		webm: true, parameters: av1Parameters,
	},
}

const DefaultVideoCodec = "libx264"

const (
	pixelFormatYUV420          = "yuv420p"
	webmExtension              = ".webm"
	audioCodecOpusV2           = "libopus"
	ffmpegRowMT                = "-row-mt"
	ffmpegLagFrames            = "-lag-in-frames"
	ffmpegCPUUsed              = "-cpu-used"
	ffmpegThreads              = "-threads"
	ffmpegEnabled              = "1"
	ffmpegDisabled             = "0"
	ffmpegAutoAltRef           = "-auto-alt-ref"
	ffmpegARNRFrames           = "-arnr-maxframes"
	ffmpegARNRStrength         = "-arnr-strength"
	ffmpegAQMode               = "-aq-mode"
	ffmpegEnableTPL            = "-enable-tpl"
	ffmpegTileRows             = "-tile-rows"
	ffmpegTileColumns          = "-tile-columns"
	vp9LagFrames               = "25"
	vp9CPUPreset               = "4"
	vp9ARNRFrames              = "7"
	vp9ARNRStrength            = "4"
	av1LagFrames               = "48"
	av1LowFrameRate            = 24
	av1HighFrameRate           = 60
	av1SlowCPUPreset           = "2"
	av1DefaultPreset           = "4"
	av1FastCPUPreset           = "6"
	encoderOptionArgumentCount = 2
)

type encoderOption struct {
	flag  string
	value string
}

var vp9EncoderOptions = [...]encoderOption{
	{ffmpegRowMT, ffmpegEnabled},
	{ffmpegLagFrames, vp9LagFrames},
	{ffmpegCPUUsed, vp9CPUPreset},
	{ffmpegAutoAltRef, ffmpegEnabled},
	{ffmpegARNRFrames, vp9ARNRFrames},
	{ffmpegARNRStrength, vp9ARNRStrength},
	{ffmpegAQMode, ffmpegDisabled},
	{ffmpegEnableTPL, ffmpegEnabled},
}

var av1EncoderOptions = [...]encoderOption{
	{ffmpegLagFrames, av1LagFrames},
	{ffmpegRowMT, ffmpegEnabled},
	{ffmpegTileRows, ffmpegDisabled},
	{ffmpegTileColumns, ffmpegEnabled},
}

// ParseCodec validates a user-provided codec name.
func ParseCodec(input string) (Codec, error) {
	if input == "" {
		return CodecH264, nil
	}
	input = strings.ToLower(input)
	for codec, cfg := range codecs {
		if slices.Contains(cfg.aliases, input) {
			return codec, nil
		}
	}
	return CodecH264, fmt.Errorf(
		"invalid video codec: %s. Valid options are: %s",
		input,
		strings.Join(CodecNames(), ", "),
	)
}

// CodecNames returns canonical encoder names in sorted order.
func CodecNames() []string {
	names := make([]string, 0, len(codecs))
	for _, cfg := range codecs {
		names = append(names, cfg.name)
	}
	slices.Sort(names)
	return names
}

func (c Codec) config() codecConfig {
	return codecs[c]
}

var validResolutions = [...]int{144, 240, 360, 480, 720, 1080, 1440, 2160}

// Resolutions returns the supported output heights.
func Resolutions() []int { return slices.Clone(validResolutions[:]) }

// ResolutionStrings returns supported heights with a p suffix.
func ResolutionStrings() []string {
	values := make([]string, len(validResolutions))
	for i, resolution := range validResolutions {
		values[i] = strconv.Itoa(resolution) + "p"
	}
	return values
}

// ParseResolution validates and normalizes a resolution to its numeric height.
func ParseResolution(input string) (int, error) {
	if input == "" {
		return 0, nil
	}
	cleaned := strings.TrimSuffix(strings.ToLower(input), "p")
	value, err := strconv.Atoi(cleaned)
	if err == nil && slices.Contains(validResolutions[:], value) {
		return value, nil
	}
	return 0, fmt.Errorf(
		"invalid resolution: %s. Valid options are: %s",
		input,
		strings.Join(ResolutionStrings(), ", "),
	)
}

// ParseSize parses and validates a target file size.
func ParseSize(input string) (int64, error) {
	input = strings.TrimSpace(input)
	bytes, err := humanize.ParseBytes(input)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid size format: %q (expected e.g. 10MB, 2GB, 50MiB)",
			input,
		)
	}
	if bytes == 0 {
		return 0, fmt.Errorf("size must be positive: %s", input)
	}
	if bytes < MinimumTargetSize {
		return 0, fmt.Errorf(
			"target size must be at least %s",
			humanize.Bytes(MinimumTargetSize),
		)
	}
	if bytes > math.MaxInt64 {
		return 0, fmt.Errorf("target size is too large: %s", input)
	}
	return int64(bytes), nil
}

// Options contains resolved video conversion settings.
type Options struct {
	OutputDir    string
	CRF          int
	Resolution   int
	Codec        Codec
	AudioBitrate int
	MultiThread  bool
	RandomName   bool
	TargetBytes  int64
	MaxDuration  float64
	Speed        float64
}

// Validate checks encoding settings and returns nonfatal warnings separately.
func (o Options) Validate() ([]string, error) {
	var warnings []string
	var errs []error
	if o.CRF < CRFMin || o.CRF > CRFMax {
		errs = append(errs, fmt.Errorf(
			"CRF value must be between %d and %d (recommended: %d-%d)",
			CRFMin,
			CRFMax,
			CRFMinRecommended,
			CRFMaxRecommended,
		))
	}
	if o.CRF < CRFMinRecommended {
		warnings = append(warnings,
			"CRF value is below the recommended minimum. This may result in very large files.")
	}
	if o.CRF > CRFMaxRecommended {
		warnings = append(warnings,
			"CRF value is above the recommended maximum. This may result in poor quality.")
	}
	if o.AudioBitrate != 0 && o.AudioBitrate%AudioBitrateStep != 0 {
		errs = append(errs, fmt.Errorf(
			"audio bitrate must be a multiple of %d",
			AudioBitrateStep,
		))
	}
	if o.AudioBitrate != 0 &&
		(o.AudioBitrate < AudioBitrateMinRecommended ||
			o.AudioBitrate > AudioBitrateMaxRecommended) {
		warnings = append(warnings,
			"Audio bitrate values outside the recommended range are not generally recommended.")
	}
	if o.Speed < SpeedMin || o.Speed > SpeedMax {
		errs = append(errs, fmt.Errorf(
			"speed must be between %.1f and %.1f (got %.1f)",
			SpeedMin,
			SpeedMax,
			o.Speed,
		))
	}
	if _, ok := codecs[o.Codec]; !ok {
		errs = append(errs, fmt.Errorf("invalid video codec value: %d", o.Codec))
	}
	if o.Resolution != 0 && !slices.Contains(validResolutions[:], o.Resolution) {
		errs = append(errs, fmt.Errorf("invalid resolution: %dp", o.Resolution))
	}
	return warnings, errors.Join(errs...)
}

// GIFOptions contains resolved gifski and frame-extraction settings.
type GIFOptions struct {
	FPS           int
	Width         int
	Quality       int
	LossyQuality  int
	MotionQuality int
	Speed         float64
}

// Validate checks GIF settings.
func (o GIFOptions) Validate() error {
	constraints := [...]intConstraint{
		{"GIF fps", o.FPS, GIFFPSMin, GIFFPSMax},
		{"GIF width", o.Width, GIFWidthMin, GIFWidthMax},
		{"GIF quality", o.Quality, GIFQualityMin, GIFQualityMax},
		{"GIF lossy quality", o.LossyQuality, GIFQualityMin, GIFQualityMax},
		{"GIF motion quality", o.MotionQuality, GIFQualityMin, GIFQualityMax},
	}
	errs := make([]error, 0, len(constraints)+1)
	for _, constraint := range constraints {
		errs = append(errs, constraint.validate())
	}
	errs = append(errs, floatRange("speed", o.Speed, SpeedMin, SpeedMax))
	return errors.Join(errs...)
}

type intConstraint struct {
	name      string
	value     int
	low, high int
}

func (c intConstraint) validate() error {
	if c.value < c.low || c.value > c.high {
		return fmt.Errorf(
			"%s must be between %d and %d (got %d)",
			c.name,
			c.low,
			c.high,
			c.value,
		)
	}
	return nil
}

func floatRange(name string, value, low, high float64) error {
	if value < low || value > high {
		return fmt.Errorf(
			"%s must be between %.1f and %.1f (got %.1f)",
			name,
			low,
			high,
			value,
		)
	}
	return nil
}

const (
	rateControlBufferScale = 2
	bitsPerByte            = 8
	bitsPerKilobit         = 1_000
	containerPayloadFactor = 0.95
)

func calculateVideoBitrate(
	targetBytes int64,
	durationSeconds float64,
	audioBitrateKbps int,
) int {
	if durationSeconds <= 0 {
		return 0
	}
	total := float64(targetBytes) * bitsPerByte / durationSeconds /
		bitsPerKilobit * containerPayloadFactor
	return max(int(total-float64(audioBitrateKbps)), 0)
}

func targetSizeArguments(options Options, duration float64) []string {
	if options.TargetBytes <= 0 {
		return nil
	}
	audioBitrate := cmp.Or(options.AudioBitrate, DefaultAudioBitrate)
	bitrate := calculateVideoBitrate(options.TargetBytes, duration, audioBitrate)
	if bitrate <= 0 {
		return nil
	}
	return []string{
		"-maxrate", fmt.Sprintf("%dk", bitrate),
		"-bufsize", fmt.Sprintf("%dk", bitrate*rateControlBufferScale),
	}
}

func codecParameters(codec Codec, multiThread bool, framerate float64) []string {
	parameters := codec.config().parameters
	if parameters == nil {
		return nil
	}
	return parameters(multiThread, framerate)
}

func x26xParameters(multiThread bool, _ float64) []string {
	if !multiThread {
		return nil
	}
	return []string{ffmpegThreads, strconv.Itoa(runtime.NumCPU())}
}

func vp9Parameters(_ bool, _ float64) []string {
	return encoderArguments(vp9EncoderOptions[:])
}

func av1Parameters(_ bool, framerate float64) []string {
	cpuPreset := av1DefaultPreset
	if framerate < av1LowFrameRate {
		cpuPreset = av1SlowCPUPreset
	} else if framerate > av1HighFrameRate {
		cpuPreset = av1FastCPUPreset
	}
	return append(
		encoderArguments(av1EncoderOptions[:]),
		ffmpegCPUUsed,
		cpuPreset,
	)
}

func encoderArguments(options []encoderOption) []string {
	arguments := make(
		[]string,
		0,
		len(options)*encoderOptionArgumentCount,
	)
	for _, option := range options {
		arguments = append(arguments, option.flag, option.value)
	}
	return arguments
}
