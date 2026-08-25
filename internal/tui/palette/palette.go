package palette

import (
	"cmp"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	colorful "github.com/lucasb-eyer/go-colorful"

	"charm.land/lipgloss/v2"
)

const (
	baseBlack = iota
	baseRed
	baseGreen
	baseYellow
	baseBlue
	baseMagenta
	baseCyan
	baseWhite
	baseBrightBlack
	baseBrightRed
	baseBrightGreen
	baseBrightYellow
	baseBrightBlue
	baseBrightMagenta
	baseBrightCyan
	baseBrightWhite
	baseColorCount
)

const (
	defaultBlend          = 0.30
	subtextBlend          = 0.35
	surfaceLowBlend       = 0.20
	surfaceMediumBlend    = 0.40
	surfaceHighBlend      = 0.60
	overlayBlend          = 0.85
	fadeEndBlend          = 0.25
	trackDimBlend         = 0.60
	trackMediumBlend      = 0.35
	trackWarmBlend        = 0.15
	maxConfigIncludeDepth = 5
)

type Colors struct {
	Accent, Warm, Info, Success, Error     color.Color
	Text, Subtext0                         color.Color
	Surface0, Surface1, Surface2, Overlay0 color.Color
	Base                                   color.Color
	FadeEnd                                color.Color
	TrackDim, TrackMid, TrackWarm          color.Color
}

var Resolved = resolve()

func resolve() Colors {
	if p := detect(); p != nil {
		return mapPalette(p)
	}
	return ansiDefaults()
}

func ansiDefaults() Colors {
	return Colors{
		Accent:    ansiColor(baseYellow),
		Warm:      ansiColor(baseBrightYellow),
		Info:      ansiColor(baseBrightCyan),
		Success:   ansiColor(baseBrightGreen),
		Error:     ansiColor(baseBrightRed),
		Text:      ansiColor(baseBrightWhite),
		Subtext0:  ansiColor(baseWhite),
		Surface0:  ansiColor(baseBrightBlack),
		Surface1:  ansiColor(baseBrightBlack),
		Surface2:  ansiColor(baseBrightBlack),
		Overlay0:  ansiColor(baseBrightBlack),
		Base:      ansiColor(baseBlack),
		FadeEnd:   ansiColor(baseBlack),
		TrackDim:  ansiColor(baseBrightBlack),
		TrackMid:  ansiColor(baseYellow),
		TrackWarm: ansiColor(baseBrightYellow),
	}
}

func mapPalette(p *base16Palette) Colors {
	bg := cmp.Or(p.Background, p.Color[baseBlack])
	fg := cmp.Or(p.Foreground, p.Color[baseBrightWhite], p.Color[baseWhite])
	if bg == "" || fg == "" {
		return ansiDefaults()
	}

	brightBlack := cmp.Or(p.Color[baseBrightBlack], blend(bg, fg, defaultBlend))
	accent := cmp.Or(p.Color[baseYellow], blend(fg, bg, defaultBlend))

	return Colors{
		Accent:    lc(accent),
		Warm:      lc(cmp.Or(p.Color[baseBrightYellow], p.Color[baseYellow])),
		Info:      lc(cmp.Or(p.Color[baseBrightCyan], p.Color[baseCyan])),
		Success:   lc(cmp.Or(p.Color[baseBrightGreen], p.Color[baseGreen])),
		Error:     lc(cmp.Or(p.Color[baseBrightRed], p.Color[baseRed])),
		Text:      lc(fg),
		Subtext0:  lc(blend(brightBlack, fg, subtextBlend)),
		Surface0:  lc(blend(bg, brightBlack, surfaceLowBlend)),
		Surface1:  lc(blend(bg, brightBlack, surfaceMediumBlend)),
		Surface2:  lc(blend(bg, brightBlack, surfaceHighBlend)),
		Overlay0:  lc(blend(bg, brightBlack, overlayBlend)),
		Base:      lc(bg),
		FadeEnd:   lc(blend(bg, brightBlack, fadeEndBlend)),
		TrackDim:  lc(blend(accent, bg, trackDimBlend)),
		TrackMid:  lc(blend(accent, bg, trackMediumBlend)),
		TrackWarm: lc(blend(accent, bg, trackWarmBlend)),
	}
}

func detect() *base16Palette {
	termProg := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	switch {
	case termProg == "kitty" || term == "xterm-kitty":
		return parseKittyPalette()
	case termProg == "alacritty" || term == "alacritty":
		return parseAlacrittyPalette()
	case termProg == "ghostty":
		return parseGhosttyPalette()
	}
	return nil
}

type base16Palette struct {
	Foreground string
	Background string
	Color      [baseColorCount]string
}

func (p *base16Palette) isUsable() bool {
	return (p.Foreground != "" || p.Color[baseWhite] != "") &&
		(p.Background != "" || p.Color[baseBlack] != "")
}

func lc(hex string) color.Color { return lipgloss.Color(hex) }

func ansiColor(index int) color.Color { return lc(strconv.Itoa(index)) }

// configPaths returns candidate config paths for appName under XDG_CONFIG_HOME
// (if set), $HOME/.config, and macOS's ~/Library/Application Support. Each
// filename in files is joined under every base. macOS paths are skipped when
// includeMacOS is false.
func configPaths(appName string, includeMacOS bool, files ...string) []string {
	var roots []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		roots = append(roots, filepath.Join(xdg, appName))
	}
	if home, err := os.UserHomeDir(); err == nil {
		if includeMacOS {
			roots = append(roots, filepath.Join(home, "Library", "Application Support", appName))
		}
		roots = append(roots, filepath.Join(home, ".config", appName))
	}
	out := make([]string, 0, len(roots)*len(files))
	for _, root := range roots {
		for _, f := range files {
			out = append(out, filepath.Join(root, f))
		}
	}
	return out
}

func expandPath(p, baseDir string) string {
	if filepath.IsAbs(p) {
		return p
	}
	if after, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, after)
		}
	}
	return filepath.Join(baseDir, p)
}

func applyHex(dst *string, src string) {
	if hex := normalizeHex(src); hex != "" {
		*dst = hex
	}
}

func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		s = "#" + s
	}
	parsed, err := colorful.Hex(s)
	if err != nil {
		return ""
	}
	return parsed.Hex()
}

func blend(a, b string, t float64) string {
	ca, _ := colorful.Hex(a)
	cb, _ := colorful.Hex(b)
	return ca.BlendLab(cb, t).Hex()
}
