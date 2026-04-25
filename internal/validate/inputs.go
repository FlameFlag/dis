package validate

import (
	"dis/internal/util"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// Inputs checks that each input is either a valid URL or an existing media file.
func Inputs(inputs []string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no input provided")
	}

	for _, input := range inputs {
		isFile := util.FileExists(input)
		isURL := util.IsValidURL(input)

		if !isFile && !isURL {
			return fmt.Errorf("invalid input file or link: %s", input)
		}

		if !isFile {
			continue
		}
		ext := filepath.Ext(input)
		mtype := mime.TypeByExtension(ext)
		if mtype == "" {
			log.Warn("Could not determine content type for file", "input", input)
			continue
		}
		if !strings.HasPrefix(mtype, "video/") && !strings.HasPrefix(mtype, "audio/") {
			return fmt.Errorf("input file is not a recognized video/audio type: %s (type: %s)", input, mtype)
		}
	}
	return nil
}

// Output checks that the output directory exists.
func Output(output string) error {
	if output == "" {
		return nil
	}
	info, err := os.Stat(output)
	if err != nil {
		return fmt.Errorf("output directory does not exist: %s", output)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", output)
	}
	return nil
}
