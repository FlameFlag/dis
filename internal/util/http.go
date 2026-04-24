package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HTTPError is returned by HTTPGet for non-2xx responses.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d from %s", e.StatusCode, e.URL)
}

// HTTPGet performs a GET request and returns the response body as bytes.
// Optional headers are applied to the request. Non-2xx responses become *HTTPError.
func HTTPGet(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: url}
	}
	return io.ReadAll(resp.Body)
}
