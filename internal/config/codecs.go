package config

import (
	"slices"
	"strings"
)

// Codec represents a video codec.
type Codec int

const (
	CodecH264 Codec = iota
	CodecHEVC
	CodecVP8
	CodecVP9
	CodecAV1
)

// codecAliases maps user-facing names to internal codec values.
var codecAliases = map[string]Codec{
	"h264":       CodecH264,
	"libx264":    CodecH264,
	"h265":       CodecHEVC,
	"libx265":    CodecHEVC,
	"hevc":       CodecHEVC,
	"vp8":        CodecVP8,
	"libvpx":     CodecVP8,
	"vp9":        CodecVP9,
	"libvpx-vp9": CodecVP9,
	"av1":        CodecAV1,
	"libaom-av1": CodecAV1,
}

// ValidResolutions lists all accepted resolution values.
var ValidResolutions = []int{144, 240, 360, 480, 720, 1080, 1440, 2160}

// ParseCodec converts a user-provided codec string to a Codec value.
// Returns CodecH264 as default if not found.
func ParseCodec(input string) Codec {
	if input == "" {
		return CodecH264
	}
	lower := strings.ToLower(input)
	if c, ok := codecAliases[lower]; ok {
		return c
	}
	return CodecH264
}

// IsValidCodec checks whether the input string is a known codec alias.
func IsValidCodec(input string) bool {
	_, ok := codecAliases[strings.ToLower(input)]
	return ok
}

type codecConfig struct {
	FFmpegName  string
	PixelFormat string // empty = defaultPixelFormat
	IsWebM      bool
}

var codecConfigs = map[Codec]codecConfig{
	CodecH264: {FFmpegName: "libx264"},
	CodecHEVC: {FFmpegName: "libx265"},
	CodecVP8:  {FFmpegName: "libvpx", IsWebM: true},
	CodecVP9:  {FFmpegName: "libvpx-vp9", IsWebM: true},
	CodecAV1:  {FFmpegName: "libaom-av1", IsWebM: true, PixelFormat: "yuv420p10le"},
}

const defaultPixelFormat = "yuv420p"

// CodecNames returns the canonical FFmpeg codec names, sorted.
func CodecNames() []string {
	names := make([]string, 0, len(codecConfigs))
	for _, cfg := range codecConfigs {
		names = append(names, cfg.FFmpegName)
	}
	slices.Sort(names)
	return names
}

// String returns the primary FFmpeg codec name (implements fmt.Stringer).
func (c Codec) String() string {
	return c.FFmpegCodecName()
}

// FFmpegCodecName returns the FFmpeg-compatible codec name.
func (c Codec) FFmpegCodecName() string {
	if cfg, ok := codecConfigs[c]; ok {
		return cfg.FFmpegName
	}
	return "libx264"
}

func (c Codec) IsWebM() bool {
	if cfg, ok := codecConfigs[c]; ok {
		return cfg.IsWebM
	}
	return false
}

// PixelFormat returns the appropriate pixel format for this codec.
func (c Codec) PixelFormat() string {
	if cfg, ok := codecConfigs[c]; ok && cfg.PixelFormat != "" {
		return cfg.PixelFormat
	}
	return defaultPixelFormat
}

var audioCodecNames = map[bool]string{
	true:  "libopus", // WebM codecs
	false: "aac",     // MP4 codecs
}

// AudioCodecName returns the audio codec to pair with this video codec.
func (c Codec) AudioCodecName() string {
	return audioCodecNames[c.IsWebM()]
}
