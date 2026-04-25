package convert

import (
	"cmp"
	"dis/internal/config"
	"dis/internal/validate"
	"fmt"
)

// targetSizeArgs returns FFmpeg arguments to constrain the video bitrate.
func targetSizeArgs(videoBitrateKbps int) []string {
	return []string{
		"-maxrate", fmt.Sprintf("%dk", videoBitrateKbps),
		"-bufsize", fmt.Sprintf("%dk", videoBitrateKbps*2),
	}
}

// targetBitrateArgs resolves the target-size setting into -maxrate/-bufsize
// args, or returns nil when unset, unparseable, or the duration is zero.
func targetBitrateArgs(s *config.Settings, duration float64) []string {
	if s.TargetSize == "" {
		return nil
	}
	targetBytes, _ := config.ParseSize(s.TargetSize)
	if targetBytes <= 0 {
		return nil
	}
	audioBitrate := cmp.Or(s.AudioBitrate, validate.DefaultAudioBitrate)
	kbps := config.CalculateVideoBitrate(targetBytes, duration, audioBitrate)
	if kbps <= 0 {
		return nil
	}
	return targetSizeArgs(kbps)
}
