package palette

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func kittyConfigDirs() []string {
	if dir := os.Getenv("KITTY_CONFIG_DIRECTORY"); dir != "" {
		return []string{dir}
	}
	// configPaths with empty filename produces the bare directory candidates.
	return configPaths("kitty", false, "")
}

func parseKittyPalette() *base16Palette {
	dirs := kittyConfigDirs()
	if len(dirs) == 0 {
		return nil
	}

	p := &base16Palette{}
	for _, dir := range dirs {
		dir = filepath.Clean(dir)
		parseKittyFile(filepath.Join(dir, "kitty.conf"), p, 0)
		parseKittyFile(filepath.Join(dir, "current-theme.conf"), p, 0)
	}

	if !p.isUsable() {
		return nil
	}
	return p
}

func parseKittyFile(path string, p *base16Palette, depth int) {
	if depth > maxConfigIncludeDepth {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	dir := filepath.Dir(path)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < kittyConfigFields {
			continue
		}
		key, val := fields[0], fields[1]

		if key == "include" {
			parseKittyFile(expandPath(val, dir), p, depth+1)
			continue
		}

		hex := normalizeHex(val)
		if hex == "" {
			continue
		}

		switch {
		case key == "foreground":
			p.Foreground = hex
		case key == "background":
			p.Background = hex
		case strings.HasPrefix(key, kittyColorPrefix):
			if idx, err := strconv.Atoi(key[len(kittyColorPrefix):]); err == nil && idx >= 0 && idx < len(p.Color) {
				p.Color[idx] = hex
			}
		}
	}
}

const (
	kittyConfigFields = 2
	kittyColorPrefix  = "color"
)
