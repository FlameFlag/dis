package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Preset defines encoding constraints for a target platform.
type Preset struct {
	TargetSize   string  `toml:"target_size"`
	MaxDuration  float64 `toml:"max_duration"`
	VideoCodec   string  `toml:"video_codec"`
	Resolution   string  `toml:"resolution"`
	Crf          int     `toml:"crf"`
	AudioBitrate int     `toml:"audio_bitrate"`
}

const xMaxDurationSeconds = 140

var xPreset = Preset{
	TargetSize:  "512MB",
	MaxDuration: xMaxDurationSeconds,
}

var builtinPresets = map[string]Preset{
	"discord": {
		TargetSize: "10MB",
	},
	"discord-nitro": {
		TargetSize: "50MB",
	},
	"twitter": xPreset,
	"x":       xPreset,
	"telegram": {
		TargetSize: "2GB",
	},
}

// ResolvePreset looks up a preset by name, checking user presets first, then builtins.
func ResolvePreset(name string, userPresets map[string]Preset) (*Preset, error) {
	lower := strings.ToLower(name)

	if userPresets != nil {
		if p, ok := userPresets[lower]; ok {
			return new(p), nil
		}
	}

	if p, ok := builtinPresets[lower]; ok {
		return new(p), nil
	}

	return nil, fmt.Errorf("unknown preset: %q (available: %s)", name, strings.Join(PresetNames(userPresets), ", "))
}

// PresetNames returns a sorted list of all available preset names (builtin + user).
func PresetNames(userPresets map[string]Preset) []string {
	combined := maps.Clone(builtinPresets)
	maps.Copy(combined, userPresets)
	return slices.Sorted(maps.Keys(combined))
}
