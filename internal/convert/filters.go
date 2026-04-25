package convert

import "fmt"

// videoSpeedFilter returns the setpts filter for the given speed, or "" if
// the speed is at or below normal playback (1.0).
func videoSpeedFilter(speed float64) string {
	if speed <= 1.0 {
		return ""
	}
	return fmt.Sprintf("setpts=PTS/%.4g", speed)
}

// audioSpeedFilter returns the atempo filter for the given speed, or "" if
// the speed is at or below normal playback (1.0).
func audioSpeedFilter(speed float64) string {
	if speed <= 1.0 {
		return ""
	}
	return fmt.Sprintf("atempo=%.4g", speed)
}
