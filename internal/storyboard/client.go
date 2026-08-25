package storyboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"slices"

	_ "golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"

	"github.com/4evy/dis/internal/cache"
	"github.com/4evy/dis/internal/media"
)

const maxConcurrentFetches = 4

// Client fetches storyboard images with an injected HTTP client.
type Client struct {
	http  *http.Client
	cache *cache.Store
}

// New creates a storyboard client.
func New(httpClient *http.Client, store *cache.Store) *Client {
	return &Client{http: httpClient, cache: store}
}

// Fetch chooses the highest-resolution storyboard and downloads its fragments.
func (c *Client) Fetch(
	ctx context.Context,
	metadata media.Metadata,
) (*StoryboardData, error) {
	if len(metadata.Storyboards) == 0 {
		return nil, errors.New("no storyboard fragments")
	}
	format := slices.MaxFunc(
		metadata.Storyboards,
		func(left, right media.StoryboardFormat) int {
			leftArea := left.Width * left.Height
			rightArea := right.Width * right.Height
			switch {
			case leftArea < rightArea:
				return -1
			case leftArea > rightArea:
				return 1
			default:
				return 0
			}
		},
	)
	info := StoryboardInfo{Rows: format.Rows, Columns: format.Columns}
	for _, fragment := range format.Fragments {
		info.Fragments = append(info.Fragments, FragmentInfo{
			URL: fragment.URL, Duration: fragment.Duration,
		})
	}
	if info.Rows <= 0 || info.Columns <= 0 || len(info.Fragments) == 0 {
		return nil, errors.New("no storyboard fragments")
	}

	decoded := make([]image.Image, len(info.Fragments))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxConcurrentFetches)
	for index, fragment := range info.Fragments {
		group.Go(func() error {
			data, err := c.get(groupCtx, fragment.URL)
			if err != nil {
				return fmt.Errorf("fragment %d: %w", index, err)
			}
			decodedImage, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				return fmt.Errorf("fragment %d decode: %w", index, err)
			}
			decoded[index] = decodedImage
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	bounds := decoded[0].Bounds()
	info.CellW = bounds.Dx() / info.Columns
	info.CellH = bounds.Dy() / info.Rows
	return &StoryboardData{Info: info, Images: decoded}, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	return cache.FetchFrom(
		c.cache,
		cache.Storyboard,
		rawURL,
		func() ([]byte, error) { return c.fetch(ctx, rawURL) },
	)
}

func (c *Client) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %d from %s", response.StatusCode, rawURL)
	}
	return io.ReadAll(response.Body)
}
