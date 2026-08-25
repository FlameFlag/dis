package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/4evy/dis/internal/app"
	"github.com/4evy/dis/internal/browsercookies"
	"github.com/4evy/dis/internal/cache"
	"github.com/4evy/dis/internal/config"
	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/download"
	"github.com/4evy/dis/internal/ffmpeg"
	"github.com/4evy/dis/internal/procgroup"
	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/storyboard"
	"github.com/4evy/dis/internal/subtitle"
	"github.com/4evy/dis/internal/tui"
	"github.com/4evy/dis/internal/tuiadapter"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	httpTimeout        = 10 * time.Second
	signalChannelDepth = 2
)

// Command is one isolated CLI instance.
type Command struct {
	root    *cobra.Command
	version string
}

// New constructs fresh commands, flags, and option state.
func New(version string) *Command {
	tui.ConfigureLogger()
	values := DefaultValues()
	command := &Command{version: version}
	command.root = &cobra.Command{
		Use:   "dis [flags] <input>...",
		Short: "Video downloader and compressor",
		Long:  "Download and compress videos from URLs or local files using yt-dlp and FFmpeg.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cobraCommand *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cobraCommand.Help()
			}
			values.Inputs = args
			return command.run(cobraCommand.Context(), values, cobraCommand)
		},
	}
	addFlags(command.root, &values)
	addCompletionCommands(command.root)
	return command
}

// Execute runs the command and manages process-group signal escalation.
func (c *Command) Execute(ctx context.Context) error {
	signals := make(chan os.Signal, signalChannelDepth)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(signals)
		close(done)
	}()
	go func() {
		select {
		case <-signals:
		case <-done:
			return
		}
		select {
		case <-signals:
			procgroup.KillAll()
		case <-done:
		}
	}()
	return fang.Execute(
		ctx,
		c.root,
		fang.WithVersion(c.version),
		fang.WithColorSchemeFunc(termColorScheme),
		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
	)
}

func (c *Command) run(
	ctx context.Context,
	values Values,
	command *cobra.Command,
) error {
	terminal := tuiadapter.NewAdapter()
	file, configErr := config.Load(config.DefaultPath())
	if configErr != nil {
		terminal.Warning(fmt.Sprintf("Failed to load config file: %v", configErr))
		file = &config.File{}
	}
	explicit := make(map[string]bool)
	command.Flags().Visit(func(flag *pflag.Flag) { explicit[flag.Name] = true })
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine working directory: %w", err)
	}
	options, warnings, err := Resolve(values, explicit, file, workingDirectory)
	if err != nil {
		return err
	}
	for _, warning := range warnings {
		terminal.Warning(warning)
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf(
			"ffmpeg not found, please install it and ensure it is in your PATH: %w",
			err,
		)
	}
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return fmt.Errorf(
			"ffprobe not found, please install FFmpeg and ensure it is in your PATH: %w",
			err,
		)
	}
	ytdlpPath, err := exec.LookPath("yt-dlp")
	if err != nil {
		return errors.New(
			"yt-dlp not found, please install it and ensure it is in your PATH",
		)
	}
	gifskiPath, _ := exec.LookPath("gifski")
	options.GIFAvailable = gifskiPath != ""
	if options.GIF && !options.GIFAvailable {
		return errors.New(
			"gifski not found - install it: brew install gifski (macOS) or cargo install gifski",
		)
	}

	var cookieFile *browsercookies.File
	var cookieCollection *browsercookies.Collection
	if options.Download.CookiesFromBrowser != "" {
		cookieFile, err = browsercookies.Export(
			ctx,
			options.Download.CookiesFromBrowser,
		)
		if err != nil {
			return fmt.Errorf("load browser cookies: %w", err)
		}
		defer func() { _ = cookieFile.Close() }()
	} else if hasWebInput(options.Inputs) {
		var cookieWarnings []error
		cookieCollection, cookieWarnings, err = browsercookies.Discover(ctx)
		if err != nil {
			return fmt.Errorf("discover browser cookies: %w", err)
		}
		defer func() { _ = cookieCollection.Close() }()
		for _, warning := range cookieWarnings {
			terminal.Warning(fmt.Sprintf("Browser cookie profile unavailable: %v", warning))
		}
	}

	store, cacheOK := cache.TryOpen()
	if cacheOK {
		defer func() { _ = store.Close() }()
		store.DeleteExpired()
	}
	httpClient := &http.Client{Timeout: httpTimeout}
	ffmpegRunner := ffmpeg.New(ffmpegPath, ffprobePath)
	downloadClient := download.New(ytdlpPath, ffmpegPath, store)
	if cookieFile != nil {
		downloadClient.WithCookies(cookieFile.Path())
	} else if cookieCollection != nil {
		browserCandidates := cookieCollection.Candidates()
		candidates := make([]download.CookieCandidate, 0, len(browserCandidates))
		for _, candidate := range browserCandidates {
			candidates = append(candidates, download.CookieCandidate{
				Source: candidate.Source,
				Path:   candidate.Path,
			})
		}
		downloadClient.
			WithCookieCandidates(candidates).
			OnCookieSelection(func(selection download.CookieSelection) {
				terminal.Info(fmt.Sprintf(
					"Using browser cookies from %s.",
					selection.Source,
				))
			})
	}
	application := app.New(
		downloadClient,
		convert.New(ffmpegRunner, gifskiPath),
		terminal,
		terminal,
		terminal,
		subtitle.New(httpClient, store),
		storyboard.New(httpClient, store),
		sponsorblock.New(httpClient, store),
	)
	return application.Run(ctx, options)
}

func hasWebInput(inputs []string) bool {
	return slices.ContainsFunc(inputs, download.IsWebURL)
}

func termColorScheme(lightDark lipgloss.LightDarkFunc) fang.ColorScheme {
	scheme := fang.AnsiColorScheme(lightDark)
	scheme.Base = tui.ColorText
	scheme.Title = tui.ColorPeach
	scheme.Description = tui.ColorSubtext0
	scheme.Codeblock = tui.ColorSurface0
	scheme.Program = tui.ColorTeal
	scheme.DimmedArgument = tui.ColorOverlay0
	scheme.Comment = tui.ColorOverlay0
	scheme.Flag = tui.ColorGreen
	scheme.FlagDefault = tui.ColorSurface2
	scheme.Command = tui.ColorYellow
	scheme.QuotedString = tui.ColorPeach
	scheme.Argument = tui.ColorText
	scheme.Help = tui.ColorSubtext0
	scheme.Dash = tui.ColorOverlay0
	scheme.ErrorHeader[0] = tui.ColorBase
	scheme.ErrorHeader[1] = tui.ColorRed
	scheme.ErrorDetails = tui.ColorRed
	return scheme
}
