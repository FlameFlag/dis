package convert

import "dis/internal/config"

// clipDuration returns the trim duration if a trim is set, else the full
// media duration — the common "what duration are we operating on?" choice.
func clipDuration(info *MediaInfo, trim *config.TrimSettings) float64 {
	if trim != nil {
		return trim.Duration
	}
	return info.Duration
}

// playbackDuration scales an input duration by the configured speed so it
// reflects the wall-clock duration of the encoded output.
func playbackDuration(d, speed float64) float64 {
	if speed > 1.0 {
		return d / speed
	}
	return d
}
