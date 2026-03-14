package sponsorblock

import (
	"context"
	"dis/internal/cache"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/charmbracelet/log"
)

// Category is a typed SponsorBlock segment category.
type Category string

const (
	CategorySponsor       Category = "sponsor"
	CategoryIntro         Category = "intro"
	CategoryOutro         Category = "outro"
	CategorySelfPromo     Category = "selfpromo"
	CategoryInteraction   Category = "interaction"
	CategoryMusicOfftopic Category = "music_offtopic"
	CategoryPreview       Category = "preview"
	CategoryHighlight     Category = "poi_highlight"
	CategoryFiller        Category = "filler"
)

// AllCategories returns all known SponsorBlock categories.
func AllCategories() []Category {
	return []Category{
		CategorySponsor, CategoryIntro, CategoryOutro, CategorySelfPromo,
		CategoryInteraction, CategoryMusicOfftopic, CategoryPreview,
		CategoryHighlight, CategoryFiller,
	}
}

// Action is a typed SponsorBlock action type.
type Action string

const (
	ActionSkip Action = "skip"
	ActionMute Action = "mute"
	ActionPOI  Action = "poi"
	ActionFull Action = "full"
)

// Segment represents a SponsorBlock segment.
type Segment struct {
	Start    float64
	End      float64
	Category Category
	Action   Action
}

const (
	apiBase = "https://sponsor.ajay.app/api/skipSegments"
)

// apiResponse is the JSON structure returned by the SponsorBlock API.
type apiResponse struct {
	Segment    [2]float64 `json:"segment"`
	Category   Category   `json:"category"`
	ActionType string     `json:"actionType"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

var videoIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:youtube\.com/watch\?.*v=|youtu\.be/|youtube\.com/embed/|youtube\.com/shorts/)([a-zA-Z0-9_-]{11})`),
}

// ExtractVideoID extracts the YouTube video ID from a URL.
// Returns empty string for non-YouTube URLs.
func ExtractVideoID(rawURL string) string {
	for _, re := range videoIDPatterns {
		if matches := re.FindStringSubmatch(rawURL); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

func openCache() (*cache.Store, bool) {
	s, err := cache.Open()
	if err != nil {
		log.Debug("cache unavailable", "err", err)
		return nil, false
	}
	return s, true
}

// GetSegments returns SponsorBlock segments for a video, using a local cache.
func GetSegments(ctx context.Context, videoID string) ([]Segment, error) {
	if store, ok := openCache(); ok {
		defer func() { _ = store.Close() }()
		store.DeleteExpired()
		if data, ok := store.GetSponsorBlock(videoID); ok {
			var segments []Segment
			if json.Unmarshal(data, &segments) == nil {
				return segments, nil
			}
		}
	}

	segments, err := fetchSegments(ctx, videoID)
	if err != nil {
		return nil, err
	}

	if store, ok := openCache(); ok {
		defer func() { _ = store.Close() }()
		if blob, err := json.Marshal(segments); err == nil {
			store.SetSponsorBlock(videoID, blob)
		}
	}
	return segments, nil
}

func fetchSegments(ctx context.Context, videoID string) ([]Segment, error) {
	cats, _ := json.Marshal(AllCategories())
	u := fmt.Sprintf("%s?videoID=%s&categories=%s", apiBase, url.QueryEscape(videoID), url.QueryEscape(string(cats)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no segments
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sponsorblock API returned %d", resp.StatusCode)
	}

	var apiSegs []apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiSegs); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	segments := make([]Segment, 0, len(apiSegs))
	for _, s := range apiSegs {
		segments = append(segments, Segment{
			Start:    s.Segment[0],
			End:      s.Segment[1],
			Category: s.Category,
			Action:   Action(s.ActionType),
		})
	}
	return segments, nil
}
