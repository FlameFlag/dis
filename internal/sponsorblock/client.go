package sponsorblock

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/4evy/dis/internal/cache"
)

// HTTPError reports a non-success SponsorBlock response.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d from %s", e.StatusCode, e.URL)
}

// Client fetches SponsorBlock segments with injected HTTP and cache access.
type Client struct {
	http    *http.Client
	cache   *cache.Store
	baseURL string
}

// New creates a SponsorBlock client.
func New(httpClient *http.Client, store *cache.Store) *Client {
	return NewWithBaseURL(httpClient, store, apiBase)
}

// NewWithBaseURL creates a client with an alternate API endpoint for tests.
func NewWithBaseURL(
	httpClient *http.Client,
	store *cache.Store,
	baseURL string,
) *Client {
	return &Client{http: httpClient, cache: store, baseURL: baseURL}
}

// Segments returns SponsorBlock segments for a video ID.
func (c *Client) Segments(ctx context.Context, videoID string) ([]Segment, error) {
	return cache.FetchFrom(
		c.cache,
		cache.SponsorBlock,
		videoID,
		func() ([]Segment, error) { return c.fetch(ctx, videoID) },
	)
}

func (c *Client) fetch(ctx context.Context, videoID string) ([]Segment, error) {
	categories, _ := json.Marshal(AllCategories())
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse API URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("videoID", videoID)
	query.Set("categories", string(categories))
	endpoint.RawQuery = query.Encode()
	requestURL := endpoint.String()
	body, err := c.get(ctx, requestURL)
	if err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	var response []apiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	segments := make([]Segment, 0, len(response))
	for _, segment := range response {
		if segment.Segment[segmentEndIndex] <= segment.Segment[segmentStartIndex] {
			continue
		}
		segments = append(segments, Segment{
			Start:    segment.Segment[segmentStartIndex],
			End:      segment.Segment[segmentEndIndex],
			Category: segment.Category,
			Action:   Action(segment.ActionType),
		})
	}
	return segments, nil
}

func (c *Client) get(ctx context.Context, requestURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
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
		return nil, &HTTPError{StatusCode: response.StatusCode, URL: requestURL}
	}
	return io.ReadAll(response.Body)
}
