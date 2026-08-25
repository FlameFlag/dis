package cli

import (
	"fmt"
	"strings"

	"github.com/4evy/dis/internal/browsercookies"
	"github.com/4evy/dis/internal/convert"

	"github.com/spf13/cobra"
)

const (
	flagOutput             = "output"
	flagCRF                = "crf"
	flagResolution         = "resolution"
	flagTrim               = "trim"
	flagVideoCodec         = "video-codec"
	flagAudioBitrate       = "audio-bitrate"
	flagMultiThread        = "multi-thread"
	flagRandom             = "random"
	flagSponsor            = "sponsor"
	flagChapter            = "chapter"
	flagNoConvert          = "no-convert"
	flagPreset             = "preset"
	flagTargetSize         = "target-size"
	flagCopy               = "copy"
	flagGIF                = "gif"
	flagGIFFPS             = "gif-fps"
	flagGIFWidth           = "gif-width"
	flagGIFQuality         = "gif-quality"
	flagGIFLossyQuality    = "gif-lossy-quality"
	flagGIFMotionQuality   = "gif-motion-quality"
	flagGIFSpeed           = "gif-speed"
	flagSpeed              = "speed"
	flagCookiesFromBrowser = "cookies-from-browser"
)

var mutuallyExclusiveFlagGroups = [][]string{
	{flagSpeed, flagGIFSpeed},
	{flagChapter, flagTrim},
	{flagCRF, flagTargetSize},
	{flagGIF, flagVideoCodec},
	{flagGIF, flagTargetSize},
}

func addFlags(command *cobra.Command, values *Values) {
	flags := command.Flags()
	flags.StringVarP(&values.Output, flagOutput, "o", values.Output, "Output directory")
	flags.IntVarP(
		&values.CRF,
		flagCRF,
		"c",
		values.CRF,
		fmt.Sprintf(
			"Constant Rate Factor (%d-%d, recommended %d-%d)",
			convert.CRFMin,
			convert.CRFMax,
			convert.CRFMinRecommended,
			convert.CRFMaxRecommended,
		),
	)
	flags.StringVarP(
		&values.Resolution,
		flagResolution,
		"r",
		values.Resolution,
		"Output resolution (e.g. 720p, 1080p)",
	)
	flags.StringVarP(
		&values.Trim,
		flagTrim,
		"t",
		values.Trim,
		"Trim video (interactive or range e.g. 10-20, 1:30-2:45)",
	)
	flags.Lookup(flagTrim).NoOptDefVal = TrimInteractive
	flags.StringVar(
		&values.VideoCodec,
		flagVideoCodec,
		values.VideoCodec,
		"Video codec (h264, h265, vp8, vp9, av1)",
	)
	flags.IntVar(
		&values.AudioBitrate,
		flagAudioBitrate,
		values.AudioBitrate,
		"Audio bitrate in kbit/s",
	)
	flags.BoolVar(
		&values.MultiThread,
		flagMultiThread,
		values.MultiThread,
		"Use all available CPU threads",
	)
	flags.BoolVar(&values.Random, flagRandom, values.Random, "Randomize output filename")
	flags.BoolVar(
		&values.Sponsor,
		flagSponsor,
		values.Sponsor,
		"Remove SponsorBlock segments (YouTube)",
	)
	flags.BoolVar(
		&values.Chapter,
		flagChapter,
		values.Chapter,
		"Select chapters to download",
	)
	flags.BoolVar(
		&values.NoConvert,
		flagNoConvert,
		values.NoConvert,
		"Skip conversion and copy the file as-is",
	)
	flags.StringVar(
		&values.Preset,
		flagPreset,
		values.Preset,
		fmt.Sprintf("Platform preset (%s)", strings.Join(configPresetNames(), ", ")),
	)
	flags.StringVar(
		&values.TargetSize,
		flagTargetSize,
		values.TargetSize,
		"Target file size (e.g. 10MB, 2GB)",
	)
	flags.BoolVar(
		&values.Copy,
		flagCopy,
		values.Copy,
		"Copy output file path to clipboard after conversion",
	)
	flags.BoolVar(&values.GIF, flagGIF, values.GIF, "Export as GIF using gifski")
	flags.IntVar(
		&values.GIFFPS,
		flagGIFFPS,
		values.GIFFPS,
		fmt.Sprintf("GIF frame rate (%d-%d)", convert.GIFFPSMin, convert.GIFFPSMax),
	)
	flags.IntVar(
		&values.GIFWidth,
		flagGIFWidth,
		values.GIFWidth,
		fmt.Sprintf(
			"GIF max width in pixels (%d-%d)",
			convert.GIFWidthMin,
			convert.GIFWidthMax,
		),
	)
	flags.IntVar(
		&values.GIFQuality,
		flagGIFQuality,
		values.GIFQuality,
		fmt.Sprintf(
			"GIF quality (%d-%d)",
			convert.GIFQualityMin,
			convert.GIFQualityMax,
		),
	)
	flags.IntVar(
		&values.GIFLossyQuality,
		flagGIFLossyQuality,
		values.GIFLossyQuality,
		fmt.Sprintf(
			"GIF lossy compression quality (%d-%d, lower = smaller but grainier)",
			convert.GIFQualityMin,
			convert.GIFQualityMax,
		),
	)
	flags.IntVar(
		&values.GIFMotionQuality,
		flagGIFMotionQuality,
		values.GIFMotionQuality,
		fmt.Sprintf(
			"GIF motion quality (%d-%d, lower = smaller but smears motion)",
			convert.GIFQualityMin,
			convert.GIFQualityMax,
		),
	)
	flags.Float64Var(
		&values.GIFSpeed,
		flagGIFSpeed,
		values.GIFSpeed,
		"GIF playback speed multiplier (e.g. 1.5, 2.0)",
	)
	flags.Float64Var(
		&values.Speed,
		flagSpeed,
		values.Speed,
		"Playback speed multiplier (e.g. 1.5, 2.0)",
	)
	flags.StringVar(
		&values.CookiesFromBrowser,
		flagCookiesFromBrowser,
		values.CookiesFromBrowser,
		fmt.Sprintf(
			"Override automatic cookies with BROWSER[:PROFILE] (%s)",
			strings.Join(browsercookies.BrowserNames(), ", "),
		),
	)

	for _, group := range mutuallyExclusiveFlagGroups {
		command.MarkFlagsMutuallyExclusive(group...)
	}
	completions := map[string]func() []string{
		flagVideoCodec: convert.CodecNames,
		flagResolution: convert.ResolutionStrings,
		flagPreset:     configPresetNames,
	}
	for name, values := range completions {
		_ = command.RegisterFlagCompletionFunc(
			name,
			func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
				return values(), cobra.ShellCompDirectiveNoFileComp
			},
		)
	}
}
