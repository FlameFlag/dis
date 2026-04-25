package validate

import (
	"dis/internal/config"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Resolution checks whether the given resolution string is valid.
func Resolution(res string) error {
	if res == "" {
		return nil
	}

	cleaned := strings.TrimSuffix(strings.ToLower(res), "p")
	val, err := strconv.Atoi(cleaned)
	if err != nil {
		return fmt.Errorf("invalid resolution: %s", res)
	}

	if slices.Contains(config.ValidResolutions, val) {
		return nil
	}

	return fmt.Errorf("invalid resolution: %s. Valid options are: %s", res, strings.Join(config.ResolutionStrings(), ", "))
}
