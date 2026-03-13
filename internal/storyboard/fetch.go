package storyboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	ytdlp "github.com/lrstanley/go-ytdlp"
	_ "golang.org/x/image/webp"
)

// FetchStoryboardData downloads and decodes all storyboard fragment images.
func FetchStoryboardData(ctx context.Context, info *StoryboardInfo) (*StoryboardData, error) {
	if info == nil || len(info.Fragments) == 0 {
		return nil, fmt.Errorf("no storyboard fragments")
	}

	images := make(map[int]image.Image, len(info.Fragments))

	for i, frag := range info.Fragments {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, frag.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("fragment %d: %w", i, err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fragment %d: %w", i, err)
		}

		defer func() { _ = resp.Body.Close() }()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fragment %d read: %w", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fragment %d: HTTP %d", i, resp.StatusCode)
		}

		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("fragment %d decode: %w", i, err)
		}

		// Compute cell dimensions from actual image size (first fragment)
		if i == 0 {
			bounds := img.Bounds()
			info.CellW = bounds.Dx() / info.Columns
			info.CellH = bounds.Dy() / info.Rows
		}

		images[i] = img
	}

	return &StoryboardData{
		Info:   *info,
		Images: images,
	}, nil
}

// StartFetch extracts storyboard info and fetches images asynchronously.
// Returns a channel that receives the result when ready, or nil if no storyboard is available.
func StartFetch(ctx context.Context, info *ytdlp.ExtractedInfo) <-chan *StoryboardData {
	sbInfo := ExtractStoryboard(info)
	if sbInfo == nil {
		return nil
	}

	ch := make(chan *StoryboardData, 1)
	go func() {
		defer close(ch)
		data, err := FetchStoryboardData(ctx, sbInfo)
		if err != nil {
			log.Debug("Storyboard fetch failed", "err", err)
			return
		}
		ch <- data
	}()
	return ch
}
