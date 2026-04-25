package convert

import (
	"dis/internal/config"
	"runtime"
	"strconv"
)

// codecParams returns codec-specific FFmpeg tuning flags.
func codecParams(codec config.Codec, multiThread bool, framerate float64) []string {
	switch codec {
	case config.CodecH264, config.CodecHEVC:
		if multiThread {
			return []string{"-threads", strconv.Itoa(runtime.NumCPU())}
		}
		return nil

	case config.CodecVP9:
		return []string{
			"-row-mt", "1",
			"-lag-in-frames", "25",
			"-cpu-used", "4",
			"-auto-alt-ref", "1",
			"-arnr-maxframes", "7",
			"-arnr-strength", "4",
			"-aq-mode", "0",
			"-enable-tpl", "1",
		}

	case config.CodecAV1:
		cpuUsed := "4"
		if framerate < 24 {
			cpuUsed = "2"
		} else if framerate > 60 {
			cpuUsed = "6"
		}
		return []string{
			"-lag-in-frames", "48",
			"-row-mt", "1",
			"-tile-rows", "0",
			"-tile-columns", "1",
			"-cpu-used", cpuUsed,
		}

	default:
		return nil
	}
}
