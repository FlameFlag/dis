package cmd

import (
	"context"
	"dis/internal/config"
	"dis/internal/tui"
	"dis/internal/tui/palette"
	"image/color"
	"os"
	"syscall"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var Version = "11.3.0"

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
