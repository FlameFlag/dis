package app

import (
	"context"
	"strings"
	"testing"

	"github.com/4evy/dis/internal/convert"
)

func TestRunFailsWhenEveryInputIsInvalid(t *testing.T) {
	t.Parallel()

	temporaryDirectory := t.TempDir()
	runner := New(nil, nil, nil, nil, nil, nil, nil, nil)
	err := runner.Run(context.Background(), Options{
		Inputs: []string{temporaryDirectory + "/missing.mp4", "://invalid"},
		Conversion: convert.Options{
			OutputDir: temporaryDirectory,
			CRF:       convert.CRFDefault,
			Codec:     convert.CodecH264,
			Speed:     convert.DefaultPlaybackSpeed,
		},
	})

	if err == nil {
		t.Fatal("Run returned nil for an entirely invalid input set")
	}
	if !strings.Contains(err.Error(), "no valid input") {
		t.Fatalf("Run returned %q, want a no-valid-input error", err)
	}
}
