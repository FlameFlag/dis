package convert

import (
	"dis/internal/config"
	"dis/internal/util"
	"os"
	"path/filepath"
	"strings"
)

// OutputExtension returns the file extension for a given codec.
func OutputExtension(codec config.Codec) string {
	if codec.IsWebM() {
		return ".webm"
	}
	return ".mp4"
}

// ConstructOutputPath builds the full output path for a converted file.
func ConstructOutputPath(inputPath string, s *config.Settings) string {
	codec := config.ParseCodec(s.VideoCodec)
	ext := OutputExtension(codec)
	return ConstructOutputPathWithExt(inputPath, s, ext)
}

// ConstructOutputPathWithExt builds the full output path using the given extension.
func ConstructOutputPathWithExt(inputPath string, s *config.Settings, ext string) string {
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

	if s.Random {
		baseName = util.ShortGUID()
	}

	outName := baseName + ext
	outPath := filepath.Join(s.Output, outName)

	// Handle collision
	if _, err := os.Stat(outPath); err == nil {
		id := util.ShortGUID()
		outName = baseName + "-" + id + ext
		outPath = filepath.Join(s.Output, outName)
	}

	return outPath
}
