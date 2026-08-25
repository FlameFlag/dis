// Package app owns workflow policy and sequencing.
package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"

	"github.com/4evy/dis/internal/convert"
	"github.com/4evy/dis/internal/media"
)

// TrimOptions contains resolved trim intent.
type TrimOptions struct {
	Interactive bool
	Clips       []media.Clip
}

// DownloadOptions contains resolved download intent.
type DownloadOptions struct {
	RemoveSponsors     bool
	CookiesFromBrowser string
}

// Options is resolved user intent for one application run.
type Options struct {
	Inputs       []string
	Download     DownloadOptions
	Conversion   convert.Options
	GIFOptions   convert.GIFOptions
	Trim         TrimOptions
	Chapter      bool
	NoConvert    bool
	Copy         bool
	GIF          bool
	GIFAvailable bool
}

// Validate checks global and cross-feature options.
func (o Options) Validate() ([]string, error) {
	warnings, conversionErr := o.Conversion.Validate()
	var errs []error
	if conversionErr != nil {
		errs = append(errs, conversionErr)
	}
	if len(o.Inputs) == 0 {
		errs = append(errs, errors.New("no input provided"))
	}
	if o.Conversion.OutputDir == "" {
		errs = append(errs, errors.New("output directory is empty"))
	} else if info, err := os.Stat(o.Conversion.OutputDir); err != nil {
		errs = append(errs, fmt.Errorf(
			"output directory does not exist: %s",
			o.Conversion.OutputDir,
		))
	} else if !info.IsDir() {
		errs = append(errs, fmt.Errorf(
			"output path is not a directory: %s",
			o.Conversion.OutputDir,
		))
	}
	if o.Chapter && (o.Trim.Interactive || len(o.Trim.Clips) > 0) {
		errs = append(errs, errors.New("chapter and trim modes are mutually exclusive"))
	}
	if o.GIF && o.Conversion.TargetBytes > 0 {
		errs = append(errs, errors.New("GIF and target size are mutually exclusive"))
	}
	if o.GIF {
		if !o.GIFAvailable {
			errs = append(errs, errors.New(
				"gifski not found: install it: brew install gifski (macOS) or cargo install gifski",
			))
		}
		if err := o.GIFOptions.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	if o.Chapter && !hasURL(o.Inputs) {
		errs = append(errs, errors.New("--chapter requires a URL input"))
	}
	return warnings, errors.Join(errs...)
}

func hasURL(inputs []string) bool {
	return slices.ContainsFunc(inputs, IsURL)
}

// IsURL reports whether input is an absolute URL with a host.
func IsURL(input string) bool {
	parsed, err := url.Parse(input)
	return err == nil && parsed.IsAbs() && parsed.Hostname() != ""
}
