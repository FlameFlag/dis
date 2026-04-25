package sponsorblock

import (
	"context"
	"dis/internal/cache"
	"dis/internal/util"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
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

// GetSegments returns SponsorBlock segments for a video, using a local cache.
func GetSegments(ctx context.Context, videoID string) ([]Segment, error) {
	return cache.FetchCached(
		videoID,
		(*cache.Store).GetSponsorBlock,
		(*cache.Store).SetSponsorBlock,
		func() ([]Segment, error) {
			return fetchSegments(ctx, videoID)
		},
	)
}

func fetchSegments(ctx context.Context, videoID string) ([]Segment, error) {
	cats, _ := json.Marshal(AllCategories())
	u := fmt.Sprintf("%s?videoID=%s&categories=%s", apiBase, url.QueryEscape(videoID), url.QueryEscape(string(cats)))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	body, err := util.HTTPGet(ctx, u, nil)
	if err != nil {
		// A 404 means "no segments for this video", not a real error.
		var httpErr *util.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	var apiSegs []apiResponse
	if err := json.Unmarshal(body, &apiSegs); err != nil {
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
