package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/convert"
	"dis/internal/tui"
	"dis/internal/tui/palette"
	"dis/internal/validate"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var Version = "11.1.0"

var settings config.Settings

var rootCmd = &cobra.Command{
	Use:   "dis [flags] <input>...",
	Short: "Video downloader and compressor",
	Long:  "Download and compress videos from URLs or local files using yt-dlp and FFmpeg.",
	Args:  cobra.ArbitraryArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
		settings.Input = args

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Warn("Failed to load config file", "err", err)
			cfg = &config.FileConfig{}
		}

		config.ApplyDefaults(&settings, cfg, cmd)

		if settings.Preset != "" {
			preset, err := config.ResolvePreset(settings.Preset, cfg.Presets)
			if err != nil {
				return err
			}
			config.ApplyPreset(&settings, preset, cmd)
		}

		return validateAll(&settings, cfg)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(cmd.Context(), &settings)
	},
}

func init() {
	tui.ConfigureLogger()
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

func Execute(ctx context.Context) error {
	return fang.Execute(ctx, rootCmd,
		fang.WithVersion(Version),
		fang.WithColorSchemeFunc(termColorScheme),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	)
}

func termColorScheme(_ lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           palette.HexToNRGBA(string(tui.ColorText)),
		Title:          palette.HexToNRGBA(string(tui.ColorPeach)),
		Description:    palette.HexToNRGBA(string(tui.ColorSubtext0)),
		Codeblock:      palette.HexToNRGBA(string(tui.ColorSurface0)),
		Program:        palette.HexToNRGBA(string(tui.ColorTeal)),
		DimmedArgument: palette.HexToNRGBA(string(tui.ColorOverlay0)),
		Comment:        palette.HexToNRGBA(string(tui.ColorOverlay0)),
		Flag:           palette.HexToNRGBA(string(tui.ColorGreen)),
		FlagDefault:    palette.HexToNRGBA(string(tui.ColorSurface2)),
		Command:        palette.HexToNRGBA(string(tui.ColorYellow)),
		QuotedString:   palette.HexToNRGBA(string(tui.ColorPeach)),
		Argument:       palette.HexToNRGBA(string(tui.ColorText)),
		ErrorHeader: [2]color.Color{
			palette.HexToNRGBA(string(tui.ColorBase)),
			palette.HexToNRGBA(string(tui.ColorRed)),
		},
		ErrorDetails: palette.HexToNRGBA(string(tui.ColorRed)),
	}
}

func validateAll(s *config.Settings, cfg *config.FileConfig) error {
	errs := errors.Join(
		validate.Inputs(s.Input),
		validate.Output(s.Output),
		validate.Crf(s.Crf),
		validate.AudioBitrate(s.AudioBitrate),
		validate.Resolution(s.Resolution),
		validate.VideoCodec(s.VideoCodec),
		validate.TargetSize(s.TargetSize),
		validate.Preset(s.Preset, cfg.Presets),
		validate.Speed(s.Speed),
	)
	if s.GIF {
		errs = errors.Join(errs,
			validate.GIFFps(s.GIFFps),
			validate.GIFWidth(s.GIFWidth),
			validate.GIFQuality(s.GIFQuality),
			validate.GIFLossyQuality(s.GIFLossyQuality),
			validate.GIFMotionQuality(s.GIFMotionQuality),
			validate.Speed(s.GIFSpeed),
		)
	}
	return errs
}

func run(ctx context.Context, s *config.Settings) error {
	for _, dep := range []string{"ffmpeg", "yt-dlp"} {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("%s not found, please install it and ensure it is in your PATH", dep)
		}
	}

	if s.GIF {
		if _, err := exec.LookPath("gifski"); err != nil {
			return fmt.Errorf("gifski not found - install it: brew install gifski (macOS) or cargo install gifski")
		}
	}

	if err := resolveOutput(s); err != nil {
		return err
	}

	links, localFiles := categorizeInputs(s.Input)
	if len(links) == 0 && len(localFiles) == 0 {
		log.Warn("No valid input links or local files were provided.")
		return nil
	}

	if s.Chapter {
		if len(links) == 0 {
			return fmt.Errorf("--chapter requires a URL input")
		}
		return runChapterMode(ctx, s, links)
	}

	trimSegments, err := resolveTrimWithSpeedPrompt(ctx, s, links, localFiles)
	if errors.Is(err, tui.ErrUserCancelled) {
		return nil
	}
	if err != nil {
		return err
	}

	if len(trimSegments) > 1 {
		return runMultiSegmentDownload(ctx, s, links, localFiles, trimSegments)
	}

	var trimSettings *config.TrimSettings
	if len(trimSegments) == 1 {
		trimSettings = &trimSegments[0]
	}

	var tempDirs []string
	defer cleanupDirs(&tempDirs)

	downloaded := downloadLinks(ctx, s, links, trimSettings, &tempDirs)

	for _, r := range downloaded {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := convertDownloaded(ctx, s, r); err != nil {
			log.Error("Failed to convert video", "path", r.OutputPath, "err", err)
		}
	}

	for _, path := range localFiles {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.GIF {
			if err := convert.ExportGIF(ctx, path, s, trimSettings, ""); err != nil {
				log.Error("Failed to export GIF", "path", path, "err", err)
			}
		} else {
			if err := convert.ConvertVideo(ctx, path, s, trimSettings, ""); err != nil {
				log.Error("Failed to convert video", "path", path, "err", err)
			}
		}
	}

	return nil
}

func resolveOutput(s *config.Settings) error {
	if s.Output == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine working directory: %w", err)
		}
		s.Output = cwd
	}
	if abs, err := filepath.Abs(s.Output); err == nil {
		s.Output = abs
	}
	return nil
}
