package download

import (
	"context"
	"sync"
)

// CookieCandidate identifies one browser-session cookie jar.
type CookieCandidate struct {
	Source string
	Path   string
}

// CookieSelection describes the browser session selected for a web URL.
type CookieSelection struct {
	Source string
}

type cookieProbe func(
	context.Context,
	string,
	CookieCandidate,
) error

type selectedCookies struct {
	candidate CookieCandidate
	known     bool
}

type cookieSelector struct {
	candidates []CookieCandidate
	selected   map[string]selectedCookies
	mu         sync.Mutex
}

func newCookieSelector(candidates []CookieCandidate) *cookieSelector {
	filtered := make([]CookieCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Path != "" {
			filtered = append(filtered, candidate)
		}
	}
	return &cookieSelector{
		candidates: filtered,
		selected:   make(map[string]selectedCookies),
	}
}

func (c *Client) cookiePath(ctx context.Context, rawURL string) (string, error) {
	if c.cookies != "" || c.cookieSelector == nil || !IsWebURL(rawURL) {
		return c.cookies, nil
	}
	probe := c.cookieProbeFunc
	if probe == nil {
		probe = c.probeCookieCandidate
	}
	selection, selectedNow, err := c.cookieSelector.selectFor(ctx, rawURL, probe)
	if err != nil {
		return "", err
	}
	if selectedNow && selection.known && c.cookieReporter != nil {
		c.cookieReporter(CookieSelection{
			Source: selection.candidate.Source,
		})
	}
	return selection.candidate.Path, nil
}

func (s *cookieSelector) selectFor(
	ctx context.Context,
	rawURL string,
	probe cookieProbe,
) (selectedCookies, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if selection, exists := s.selected[rawURL]; exists {
		return selection, false, nil
	}

	for _, candidate := range s.candidates {
		if err := ctx.Err(); err != nil {
			return selectedCookies{}, false, err
		}
		if err := probe(ctx, rawURL, candidate); err != nil {
			continue
		}
		selection := selectedCookies{
			candidate: candidate,
			known:     true,
		}
		s.selected[rawURL] = selection
		return selection, true, nil
	}
	selection := selectedCookies{}
	s.selected[rawURL] = selection
	return selection, true, nil
}

func (c *Client) probeCookieCandidate(
	ctx context.Context,
	rawURL string,
	candidate CookieCandidate,
) error {
	command := c.command(candidate.Path).
		SkipDownload().
		Print("%(id)s")
	_, err := runCommand(ctx, command, rawURL, nil)
	return err
}
