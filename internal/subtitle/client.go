package subtitle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/4evy/dis/internal/cache"
	"github.com/4evy/dis/internal/media"
)

// Client fetches subtitle sources with injected HTTP and cache access.
type Client struct {
	http  *http.Client
	cache *cache.Store
}

// New creates a subtitle client.
func New(httpClient *http.Client, store *cache.Store) *Client {
	return &Client{http: httpClient, cache: store}
}

// Fetch selects, downloads, and parses subtitles from neutral metadata.
func (c *Client) Fetch(
	ctx context.Context,
	metadata media.Metadata,
) (Transcript, error) {
	if metadata.ID == "" {
		return c.fetchUncached(ctx, metadata)
	}
	return cache.FetchFrom(
		c.cache,
		cache.Transcript,
		metadata.ID,
		func() (Transcript, error) { return c.fetchUncached(ctx, metadata) },
	)
}

func (c *Client) fetchUncached(
	ctx context.Context,
	metadata media.Metadata,
) (Transcript, error) {
	var failures []error
	sources := []map[string][]media.SubtitleSource{
		metadata.Subtitles,
		metadata.AutomaticCaptions,
	}
	for _, source := range sources {
		if len(source) == 0 {
			continue
		}
		for _, language := range []string{"en", "en-US", "en-GB", "en-orig"} {
			entries := source[language]
			if len(entries) == 0 {
				continue
			}
			transcript, err := c.fetchAndParse(ctx, entries)
			if err == nil {
				return transcript, nil
			}
			failures = append(failures, err)
		}
		for _, entries := range source {
			if len(entries) == 0 {
				continue
			}
			transcript, err := c.fetchAndParse(ctx, entries)
			if err == nil {
				return transcript, nil
			}
			failures = append(failures, err)
		}
	}
	return nil, errors.Join(failures...)
}

type sourceFormat int

const (
	sourceJSON3 sourceFormat = iota
	sourceVTT
	sourceSRT
	sourceUnknown
)

func detectSourceFormat(rawURL string) sourceFormat {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return sourceUnknown
	}
	switch strings.ToLower(parsed.Query().Get("fmt")) {
	case "json3":
		return sourceJSON3
	case "vtt":
		return sourceVTT
	case "srt":
		return sourceSRT
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".vtt":
		return sourceVTT
	case ".srt":
		return sourceSRT
	default:
		return sourceUnknown
	}
}

func (c *Client) fetchAndParse(
	ctx context.Context,
	entries []media.SubtitleSource,
) (Transcript, error) {
	formats := make(map[sourceFormat]media.SubtitleSource)
	for _, entry := range entries {
		if entry.URL == "" {
			continue
		}
		format := detectSourceFormat(entry.URL)
		if _, exists := formats[format]; !exists {
			formats[format] = entry
		}
	}
	var selected media.SubtitleSource
	var format sourceFormat
	for _, candidate := range []sourceFormat{
		sourceJSON3, sourceVTT, sourceSRT, sourceUnknown,
	} {
		if entry, ok := formats[candidate]; ok {
			selected = entry
			format = candidate
			break
		}
	}
	if selected.URL == "" {
		return nil, errors.New("no subtitle entries with URLs")
	}
	body, err := c.get(ctx, selected)
	if err != nil {
		return nil, fmt.Errorf("fetching subtitle: %w", err)
	}
	data := string(body)
	switch format {
	case sourceJSON3:
		return ParseJSON3(data)
	case sourceVTT:
		return ParseVTT(data)
	case sourceSRT:
		return ParseSRT(data)
	default:
		if strings.HasPrefix(strings.TrimSpace(data), "WEBVTT") {
			return ParseVTT(data)
		}
		return ParseSRT(data)
	}
}

func (c *Client) get(
	ctx context.Context,
	source media.SubtitleSource,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		source.URL,
		nil,
	)
	if err != nil {
		return nil, err
	}
	for key, value := range source.Headers {
		request.Header.Set(key, value)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"HTTP %d from %s",
			response.StatusCode,
			source.URL,
		)
	}
	return io.ReadAll(response.Body)
}
