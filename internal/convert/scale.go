package convert

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// scaleFilter returns a "scale=W:H" filter string for the given resolution,
// preserving aspect ratio and ensuring even dimensions. Returns "" on invalid input.
func scaleFilter(resolution string, origWidth, origHeight int) string {
	cleaned := strings.TrimSuffix(strings.ToLower(resolution), "p")
	resInt, err := strconv.Atoi(cleaned)
	if err != nil {
		return ""
	}

	aspectRatio := float64(origWidth) / float64(origHeight)
	outWidth := int(math.Round(float64(resInt) * aspectRatio))
	outHeight := resInt

	// Ensure even dimensions
	outWidth -= outWidth % 2
	outHeight -= outHeight % 2

	return fmt.Sprintf("scale=%d:%d", outWidth, outHeight)
}

func resolutionArgs(resolution string, origWidth, origHeight int) []string {
	if sf := scaleFilter(resolution, origWidth, origHeight); sf != "" {
		return []string{"-vf", sf}
	}
	return nil
}
