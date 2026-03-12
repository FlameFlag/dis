package subtitle

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/log"
)

// WaveformSample represents a single amplitude sample, normalized to [0, 1].
type WaveformSample struct {
	Amplitude float64
}

// ExtractWaveform downloads audio and extracts amplitude data.
// numSamples is the desired number of output samples (typically terminal width).
func ExtractWaveform(ctx context.Context, rawURL string, numSamples int) ([]WaveformSample, error) {
	start := time.Now()

	tmpDir, err := os.MkdirTemp("", "dis-waveform-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	audioPath, err := downloadAudio(ctx, rawURL, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("downloading audio: %w", err)
	}

	samples, err := extractPCM(ctx, audioPath, numSamples)
	if err != nil {
		return nil, fmt.Errorf("PCM extraction: %w", err)
	}

	log.Debug("Waveform extraction complete",
		"samples", len(samples), "elapsed", time.Since(start).Round(time.Millisecond))
	return samples, nil
}

func extractPCM(ctx context.Context, audioPath string, numSamples int) ([]WaveformSample, error) {
	const sampleRate = 8000

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", audioPath,
		"-f", "f32le",
		"-acodec", "pcm_f32le",
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	var rawSamples []float32
	buf := make([]byte, 4)
	for {
		_, err := io.ReadFull(stdout, buf)
		if err != nil {
			break
		}
		val := math.Float32frombits(binary.LittleEndian.Uint32(buf))
		rawSamples = append(rawSamples, val)
	}

	if err := cmd.Wait(); err != nil {
		if len(rawSamples) == 0 {
			return nil, fmt.Errorf("ffmpeg PCM extraction failed: %w", err)
		}
	}

	if len(rawSamples) == 0 {
		return nil, fmt.Errorf("no audio samples extracted")
	}

	bucketSize := max(len(rawSamples)/numSamples, 1)

	samples := make([]WaveformSample, 0, numSamples)
	maxAmplitude := float64(0)

	for i := 0; i < len(rawSamples) && len(samples) < numSamples; i += bucketSize {
		end := min(i+bucketSize, len(rawSamples))
		peak := float64(0)
		for j := i; j < end; j++ {
			abs := math.Abs(float64(rawSamples[j]))
			if abs > peak {
				peak = abs
			}
		}
		samples = append(samples, WaveformSample{Amplitude: peak})
		if peak > maxAmplitude {
			maxAmplitude = peak
		}
	}

	if maxAmplitude > 0 {
		for i := range samples {
			samples[i].Amplitude /= maxAmplitude
		}
	}

	return samples, nil
}
