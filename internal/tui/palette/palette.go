package palette

import (
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	colorful "github.com/lucasb-eyer/go-colorful"

	"github.com/charmbracelet/lipgloss"
)

type Colors struct {
	Accent, Warm, Info, Success, Error     lipgloss.Color
	Text, Subtext0                         lipgloss.Color
	Surface0, Surface1, Surface2, Overlay0 lipgloss.Color
	Base                                   lipgloss.Color
	FadeEnd                                lipgloss.Color
	TrackDim, TrackMid, TrackWarm          lipgloss.Color
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
		Accent:    "3",
		Warm:      "11",
		Info:      "14",
		Success:   "10",
		Error:     "9",
		Text:      "15",
		Subtext0:  "7",
		Surface0:  "8",
		Surface1:  "8",
		Surface2:  "8",
		Overlay0:  "8",
		Base:      "0",
		FadeEnd:   "0",
		TrackDim:  "8",
		TrackMid:  "3",
		TrackWarm: "11",
	}
}

func mapPalette(p *base16Palette) Colors {
	bg := firstNonEmpty(p.Background, p.Color[0])
	fg := firstNonEmpty(p.Foreground, p.Color[15], p.Color[7])
	if bg == "" || fg == "" {
		return ansiDefaults()
	}

	brightBlack := firstNonEmpty(p.Color[8], blend(bg, fg, 0.30))
	accent := firstNonEmpty(p.Color[3], blend(fg, bg, 0.30))

	return Colors{
		Accent:    lc(accent),
		Warm:      lc(firstNonEmpty(p.Color[11], p.Color[3])),
		Info:      lc(firstNonEmpty(p.Color[14], p.Color[6])),
		Success:   lc(firstNonEmpty(p.Color[10], p.Color[2])),
		Error:     lc(firstNonEmpty(p.Color[9], p.Color[1])),
		Text:      lc(fg),
		Subtext0:  lc(blend(brightBlack, fg, 0.35)),
		Surface0:  lc(blend(bg, brightBlack, 0.20)),
		Surface1:  lc(blend(bg, brightBlack, 0.40)),
		Surface2:  lc(blend(bg, brightBlack, 0.60)),
		Overlay0:  lc(blend(bg, brightBlack, 0.85)),
		Base:      lc(bg),
		FadeEnd:   lc(blend(bg, brightBlack, 0.25)),
		TrackDim:  lc(blend(accent, bg, 0.60)),
		TrackMid:  lc(blend(accent, bg, 0.35)),
		TrackWarm: lc(blend(accent, bg, 0.15)),
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
	Color      [16]string
}

func (p *base16Palette) isUsable() bool {
	return (p.Foreground != "" || p.Color[7] != "") &&
		(p.Background != "" || p.Color[0] != "")
}

func lc(hex string) lipgloss.Color { return lipgloss.Color(hex) }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
	if _, err := colorful.Hex(s); err != nil {
		return ""
	}
	return strings.ToLower(s)
}

func blend(a, b string, t float64) string {
	ca, _ := colorful.Hex(a)
	cb, _ := colorful.Hex(b)
	return ca.BlendRgb(cb, t).Hex()
}

var catppuccinHex = [16]string{
	"#24273a", "#ed8796", "#a6da95", "#f5a97f",
	"#8aadf4", "#f5bde6", "#8bd5ca", "#a5adcb",
	"#6e738d", "#ed8796", "#a6da95", "#eed49f",
	"#8aadf4", "#f5bde6", "#8bd5ca", "#cad3f5",
}

// HexToNRGBA converts a color string to color.NRGBA.
// Accepts "#RRGGBB" hex or an ANSI index ("0"-"15"), mapping the latter
// through Catppuccin Macchiato as a fallback.
func HexToNRGBA(s string) color.NRGBA {
	if !strings.HasPrefix(s, "#") {
		if idx, err := strconv.Atoi(s); err == nil && idx >= 0 && idx < 16 {
			s = catppuccinHex[idx]
		}
	}
	c, err := colorful.Hex(s)
	if err != nil {
		return color.NRGBA{A: 0xff}
	}
	r, g, b := c.RGB255()
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}
