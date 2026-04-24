package cmd

import (
	"dis/internal/config"
	"dis/internal/util"
	"fmt"
	"strings"
)

// parseTrimRange parses a "START-END" range string into TrimSettings.
func parseTrimRange(input string) (*config.TrimSettings, error) {
	parts := strings.SplitN(input, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected format START-END (e.g. 10-20, 1:30-2:45)")
	}

	if strings.Contains(parts[0], ":") || strings.Contains(parts[1], ":") {
		var found bool
		for i := 1; i < len(input); i++ {
			if input[i] != '-' {
				continue
			}
			left, right := input[:i], input[i+1:]
			_, errL := util.ParseTimeValue(left)
			_, errR := util.ParseTimeValue(right)
			if errL == nil && errR == nil {
				parts[0], parts[1] = left, right
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("expected format START-END (e.g. 10-20, 1:30-2:45)")
		}
	}

	start, err := util.ParseTimeValue(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid start time %q: %w", parts[0], err)
	}

	end, err := util.ParseTimeValue(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid end time %q: %w", parts[1], err)
	}

	if end <= start {
		return nil, fmt.Errorf("end time (%.2f) must be greater than start time (%.2f)", end, start)
	}

	return &config.TrimSettings{
		Start:    start,
		Duration: end - start,
	}, nil
}
