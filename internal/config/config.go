package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// File represents the raw TOML configuration file structure.
// Pointer fields allow detecting "not set" vs zero-value.
type File struct {
	Crf                *int    `toml:"crf"`
	Resolution         *string `toml:"resolution"`
	VideoCodec         *string `toml:"video_codec"`
	AudioBitrate       *int    `toml:"audio_bitrate"`
	MultiThread        *bool   `toml:"multi_thread"`
	Output             *string `toml:"output"`
	Preset             *string `toml:"preset"`
	TargetSize         *string `toml:"target_size"`
	CookiesFromBrowser *string `toml:"cookies_from_browser"`

	Presets map[string]Preset `toml:"presets"`
}

// DefaultPath returns the default configuration path.
func DefaultPath() string {
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

// Load reads a raw TOML configuration file. A missing file is not an error.
func Load(path string) (*File, error) {
	if path == "" {
		return &File{}, nil
	}

	var cfg File
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &File{}, nil
		}
		return nil, err
	}

	return &cfg, nil
}
