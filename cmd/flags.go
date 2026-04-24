package cmd

import (
	"dis/internal/config"
	"dis/internal/validate"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	f := rootCmd.Flags()
	f.StringVarP(&settings.Output, "output", "o", ".", "Output directory")
	f.IntVarP(&settings.Crf, "crf", "c", validate.CRFDefault,
		fmt.Sprintf("Constant Rate Factor (%d-%d, recommended %d-%d)",
			validate.CRFMin, validate.CRFMax, validate.CRFMinRecommended, validate.CRFMaxRecommended))
	f.StringVarP(&settings.Resolution, "resolution", "r", "", "Output resolution (e.g. 720p, 1080p)")
	f.StringVarP(&settings.Trim, "trim", "t", "", "Trim video (interactive or range e.g. 10-20, 1:30-2:45)")
	rootCmd.Flags().Lookup("trim").NoOptDefVal = config.TrimInteractive
	f.StringVar(&settings.VideoCodec, "video-codec", "libx264", "Video codec (h264, h265, vp8, vp9, av1)")
	f.IntVar(&settings.AudioBitrate, "audio-bitrate", 0, "Audio bitrate in kbit/s")
	f.BoolVar(&settings.MultiThread, "multi-thread", true, "Use all available CPU threads")
	f.BoolVar(&settings.Random, "random", false, "Randomize output filename")
	f.BoolVar(&settings.Sponsor, "sponsor", false, "Remove SponsorBlock segments (YouTube)")
	f.BoolVar(&settings.Chapter, "chapter", false, "Select chapters to download")
	f.BoolVar(&settings.NoConvert, "no-convert", false, "Skip conversion and copy the file as-is")

	f.StringVar(&settings.Preset, "preset", "", "Platform preset (discord, discord-nitro, twitter, telegram)")
	f.StringVar(&settings.TargetSize, "target-size", "", "Target file size (e.g. 10MB, 2GB)")
	f.BoolVar(&settings.Copy, "copy", false, "Copy output file path to clipboard after conversion")
	f.BoolVar(&settings.GIF, "gif", false, "Export as GIF using gifski")
	f.IntVar(&settings.GIFFps, "gif-fps", 15, "GIF frame rate (1-50)")
	f.IntVar(&settings.GIFWidth, "gif-width", 480, "GIF max width in pixels")
	f.IntVar(&settings.GIFQuality, "gif-quality", 80, "GIF quality (1-100)")
	f.IntVar(&settings.GIFLossyQuality, "gif-lossy-quality", 80, "GIF lossy compression quality (1-100, lower = smaller but grainier)")
	f.IntVar(&settings.GIFMotionQuality, "gif-motion-quality", 80, "GIF motion quality (1-100, lower = smaller but smears motion)")
	f.Float64Var(&settings.GIFSpeed, "gif-speed", 1.0, "GIF playback speed multiplier (e.g. 1.5, 2.0)")
	f.Float64Var(&settings.Speed, "speed", 1.0, "Playback speed multiplier (e.g. 1.5, 2.0)")
	rootCmd.MarkFlagsMutuallyExclusive("speed", "gif-speed")
	rootCmd.MarkFlagsMutuallyExclusive("chapter", "trim")
	rootCmd.MarkFlagsMutuallyExclusive("crf", "target-size")
	rootCmd.MarkFlagsMutuallyExclusive("gif", "video-codec")
	rootCmd.MarkFlagsMutuallyExclusive("gif", "target-size")
	_ = rootCmd.RegisterFlagCompletionFunc("video-codec", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.CodecNames(), cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("resolution", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.ResolutionStrings(), cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("preset", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.PresetNames(nil), cobra.ShellCompDirectiveNoFileComp
	})
}
