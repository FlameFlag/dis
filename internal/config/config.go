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

// ApplyDefaults sets settings fields from the config file, but only for fields
// not explicitly set via CLI flags.
func ApplyDefaults(s *Settings, cfg *FileConfig, cmd *cobra.Command) {
	if cfg.Crf != nil && !cmd.Flags().Changed("crf") {
		s.Crf = *cfg.Crf
	}
	if cfg.Resolution != nil && !cmd.Flags().Changed("resolution") {
		s.Resolution = *cfg.Resolution
	}
	if cfg.VideoCodec != nil && !cmd.Flags().Changed("video-codec") {
		s.VideoCodec = *cfg.VideoCodec
	}
	if cfg.AudioBitrate != nil && !cmd.Flags().Changed("audio-bitrate") {
		s.AudioBitrate = *cfg.AudioBitrate
	}
	if cfg.MultiThread != nil && !cmd.Flags().Changed("multi-thread") {
		s.MultiThread = *cfg.MultiThread
	}
	if cfg.Output != nil && !cmd.Flags().Changed("output") {
		s.Output = *cfg.Output
	}
	if cfg.Preset != nil && !cmd.Flags().Changed("preset") {
		s.Preset = *cfg.Preset
	}
	if cfg.TargetSize != nil && !cmd.Flags().Changed("target-size") {
		s.TargetSize = *cfg.TargetSize
	}
}
