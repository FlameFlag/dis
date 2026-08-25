package tuiadapter

import (
	"slices"
	"testing"
)

func TestLowerResolutionsUsesInputHeight(t *testing.T) {
	t.Parallel()

	want := []int{144, 240, 360, 480, 720}
	got := lowerResolutions(1080)
	if !slices.Equal(got, want) {
		t.Fatalf("lowerResolutions(1080) = %v, want %v", got, want)
	}
}
