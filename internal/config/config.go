package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// FileConfig represents the TOML configuration file structure.
// Pointer fields allow detecting "not set" vs zero-value.
type FileConfig struct {
	Crf          *int    `toml:"crf"`
	Resolution   *string `toml:"resolution"`
	VideoCodec   *string `toml:"video_codec"`
	AudioBitrate *int    `toml:"audio_bitrate"`
	MultiThread  *bool   `toml:"multi_thread"`
	Output       *string `toml:"output"`
	Preset       *string `toml:"preset"`
	TargetSize   *string `toml:"target_size"`

	Presets map[string]Preset `toml:"presets"`
}

// ConfigPath returns the path to the config file.
// Checks XDG_CONFIG_HOME first, then falls back to ~/.config.
func ConfigPath() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "dis", "config.toml")
}

// LoadConfig reads and parses the TOML config file.
// Returns a zero-value FileConfig if the file does not exist.
func LoadConfig() (*FileConfig, error) {
	path := ConfigPath()
	if path == "" {
		return &FileConfig{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &FileConfig{}, nil
		}
		return nil, err
	}

	var cfg FileConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefault sets dst from src if src is non-nil and the flag was not explicitly set.
func applyDefault[T any](dst *T, src *T, cmd *cobra.Command, flag string) {
	if src != nil && !cmd.Flags().Changed(flag) {
		*dst = *src
	}
}

// ApplyDefaults sets settings fields from the config file, but only for fields
// not explicitly set via CLI flags.
func ApplyDefaults(s *Settings, cfg *FileConfig, cmd *cobra.Command) {
	applyDefault(&s.Crf, cfg.Crf, cmd, "crf")
	applyDefault(&s.Resolution, cfg.Resolution, cmd, "resolution")
	applyDefault(&s.VideoCodec, cfg.VideoCodec, cmd, "video-codec")
	applyDefault(&s.AudioBitrate, cfg.AudioBitrate, cmd, "audio-bitrate")
	applyDefault(&s.MultiThread, cfg.MultiThread, cmd, "multi-thread")
	applyDefault(&s.Output, cfg.Output, cmd, "output")
	applyDefault(&s.Preset, cfg.Preset, cmd, "preset")
	applyDefault(&s.TargetSize, cfg.TargetSize, cmd, "target-size")
}
