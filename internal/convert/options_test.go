package convert

import (
	"slices"
	"testing"
)

func TestVP9ParametersBelongOnlyToVP9(t *testing.T) {
	t.Parallel()

	want := encoderArguments(vp9EncoderOptions[:])
	if got := codecParameters(CodecVP9, false, 0); !slices.Equal(got, want) {
		t.Fatalf("codecParameters(VP9) = %q, want %q", got, want)
	}
	if got := codecParameters(CodecVP8, false, 0); got != nil {
		t.Fatalf("codecParameters(VP8) = %q, want nil", got)
	}
}
