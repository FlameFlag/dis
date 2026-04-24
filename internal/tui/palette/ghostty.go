package palette

import (
	"bufio"
	"dis/internal/util"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ghosttyConfigPath() string {
	var candidates []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates,
			filepath.Join(xdg, "ghostty", "config.ghostty"),
			filepath.Join(xdg, "ghostty", "config"),
		)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Library", "Application Support", "ghostty", "config.ghostty"),
			filepath.Join(home, "Library", "Application Support", "ghostty", "config"),
			filepath.Join(home, ".config", "ghostty", "config.ghostty"),
			filepath.Join(home, ".config", "ghostty", "config"),
		)
	}
	return util.FirstExistingFile(candidates...)
}

func parseGhosttyPalette() *base16Palette {
	cfgPath := ghosttyConfigPath()
	if cfgPath == "" {
		return nil
	}

	p := &base16Palette{}
	if name := extractGhosttyThemeName(cfgPath); name != "" {
		resolveGhosttyTheme(name, p, cfgPath)
	}
	parseGhosttyColors(cfgPath, p)

	if !p.isUsable() {
		return nil
	}
	return p
}

func extractGhosttyThemeName(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if key, val, ok := strings.Cut(line, "="); ok {
			if strings.TrimSpace(key) == "theme" {
				return strings.TrimSpace(val)
			}
		}
	}
	return ""
}

func parseGhosttyColors(path string, p *base16Palette) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "foreground":
			applyHex(&p.Foreground, val)
		case "background":
			applyHex(&p.Background, val)
		case "palette":
			idxStr, colorVal, ok := strings.Cut(val, "=")
			if !ok {
				continue
			}
			idx, err := strconv.Atoi(strings.TrimSpace(idxStr))
			if err == nil && idx >= 0 && idx < 16 {
				applyHex(&p.Color[idx], strings.TrimSpace(colorVal))
			}
		}
	}
}

func resolveGhosttyTheme(name string, p *base16Palette, cfgPath string) {
	name = ghosttyThemeFromSpec(name)
	if name == "" {
		return
	}

	configDir := filepath.Dir(cfgPath)
	searchDirs := []string{filepath.Join(configDir, "themes")}
	if resDir := os.Getenv("GHOSTTY_RESOURCES_DIR"); resDir != "" {
		searchDirs = append(searchDirs, filepath.Join(resDir, "themes"))
	}

	for _, dir := range searchDirs {
		for _, candidate := range []string{name, name + ".ghostty"} {
			themePath := filepath.Join(dir, candidate)
			if _, err := os.Stat(themePath); err == nil {
				parseGhosttyColors(themePath, p)
				return
			}
		}
	}
}

func ghosttyThemeFromSpec(s string) string {
	if !strings.Contains(s, ",") {
		return strings.TrimSpace(s)
	}
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "dark:"); ok {
			return strings.TrimSpace(after)
		}
	}
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "light:"); ok {
			return strings.TrimSpace(after)
		}
	}
	return strings.TrimSpace(s)
}
