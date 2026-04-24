package palette

import (
	"dis/internal/util"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type alacrittyConfig struct {
	General struct {
		Import []string `toml:"import"`
	} `toml:"general"`
	Colors struct {
		Primary struct {
			Foreground string `toml:"foreground"`
			Background string `toml:"background"`
		} `toml:"primary"`
		Normal struct {
			Black   string `toml:"black"`
			Red     string `toml:"red"`
			Green   string `toml:"green"`
			Yellow  string `toml:"yellow"`
			Blue    string `toml:"blue"`
			Magenta string `toml:"magenta"`
			Cyan    string `toml:"cyan"`
			White   string `toml:"white"`
		} `toml:"normal"`
		Bright struct {
			Black   string `toml:"black"`
			Red     string `toml:"red"`
			Green   string `toml:"green"`
			Yellow  string `toml:"yellow"`
			Blue    string `toml:"blue"`
			Magenta string `toml:"magenta"`
			Cyan    string `toml:"cyan"`
			White   string `toml:"white"`
		} `toml:"bright"`
	} `toml:"colors"`
}

func alacrittyConfigPath() string {
	var candidates []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates,
			filepath.Join(xdg, "alacritty", "alacritty.toml"),
			filepath.Join(xdg, "alacritty.toml"),
		)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".config", "alacritty", "alacritty.toml"),
			filepath.Join(home, ".alacritty.toml"),
		)
	}
	return util.FirstExistingFile(candidates...)
}

func parseAlacrittyPalette() *base16Palette {
	path := alacrittyConfigPath()
	if path == "" {
		return nil
	}
	p := &base16Palette{}
	parseAlacrittyFile(path, p, 0)

	if !p.isUsable() {
		return nil
	}
	return p
}

func parseAlacrittyFile(path string, p *base16Palette, depth int) {
	if depth > 5 {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg alacrittyConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return
	}

	dir := filepath.Dir(path)
	for _, imp := range cfg.General.Import {
		parseAlacrittyFile(expandPath(imp, dir), p, depth+1)
	}

	c := &cfg.Colors
	applyHex(&p.Foreground, c.Primary.Foreground)
	applyHex(&p.Background, c.Primary.Background)

	normal := [8]string{
		c.Normal.Black, c.Normal.Red, c.Normal.Green, c.Normal.Yellow,
		c.Normal.Blue, c.Normal.Magenta, c.Normal.Cyan, c.Normal.White,
	}
	bright := [8]string{
		c.Bright.Black, c.Bright.Red, c.Bright.Green, c.Bright.Yellow,
		c.Bright.Blue, c.Bright.Magenta, c.Bright.Cyan, c.Bright.White,
	}
	for i, v := range normal {
		applyHex(&p.Color[i], v)
	}
	for i, v := range bright {
		applyHex(&p.Color[8+i], v)
	}
}
