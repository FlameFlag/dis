package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
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

// BuiltinPresets maps preset names to their definitions.
var BuiltinPresets = map[string]Preset{
	"discord": {
		TargetSize: "10MB",
	},
	"discord-nitro": {
		TargetSize: "50MB",
	},
	"twitter": {
		TargetSize:  "512MB",
		MaxDuration: 140,
	},
	"telegram": {
		TargetSize: "2GB",
	},
}

// ResolvePreset looks up a preset by name, checking user presets first, then builtins.
func ResolvePreset(name string, userPresets map[string]Preset) (*Preset, error) {
	lower := strings.ToLower(name)

	if userPresets != nil {
		if p, ok := userPresets[lower]; ok {
			return &p, nil
		}
	}

	if p, ok := BuiltinPresets[lower]; ok {
		return &p, nil
	}

	return nil, fmt.Errorf("unknown preset: %q (available: %s)", name, strings.Join(PresetNames(userPresets), ", "))
}

// ApplyPreset applies preset values to settings, but only for fields not explicitly
// set via CLI flags.
func ApplyPreset(s *Settings, p *Preset, cmd *cobra.Command) {
	if p.TargetSize != "" && !cmd.Flags().Changed("target-size") {
		s.TargetSize = p.TargetSize
	}
	if p.MaxDuration > 0 {
		s.MaxDuration = p.MaxDuration
	}
	if p.VideoCodec != "" && !cmd.Flags().Changed("video-codec") {
		s.VideoCodec = p.VideoCodec
	}
	if p.Resolution != "" && !cmd.Flags().Changed("resolution") {
		s.Resolution = p.Resolution
	}
	if p.Crf > 0 && !cmd.Flags().Changed("crf") {
		s.Crf = p.Crf
	}
	if p.AudioBitrate > 0 && !cmd.Flags().Changed("audio-bitrate") {
		s.AudioBitrate = p.AudioBitrate
	}
}

// PresetNames returns a sorted list of all available preset names (builtin + user).
func PresetNames(userPresets map[string]Preset) []string {
	seen := make(map[string]struct{})
	var names []string

	for name := range BuiltinPresets {
		names = append(names, name)
		seen[name] = struct{}{}
	}
	for name := range userPresets {
		if _, ok := seen[name]; !ok {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}
