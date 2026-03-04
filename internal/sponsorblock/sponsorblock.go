package sponsorblock

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

// Segment represents a SponsorBlock segment.
type Segment struct {
	Start    float64
	End      float64
	Category Category
	Action   string // "skip", "mute", "poi", "full"
}

const (
	apiBase  = "https://sponsor.ajay.app/api/skipSegments"
	cacheTTL = 48 * time.Hour
	cleanTTL = 7 * 24 * time.Hour
)

// apiResponse is the JSON structure returned by the SponsorBlock API.
type apiResponse struct {
	Segment    [2]float64 `json:"segment"`
	Category   Category   `json:"category"`
	ActionType string     `json:"actionType"`
}

// cacheEntry is stored as JSON in the cache file.
type cacheEntry struct {
	FetchedAt time.Time `json:"fetched_at"`
	Segments  []Segment `json:"segments"` // nil means "no segments found"
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

// GetSegments returns SponsorBlock segments for a video, using a local cache.
func GetSegments(ctx context.Context, videoID string) ([]Segment, error) {
	cacheDir := cacheDirectory()

	// Lazy cleanup of old cache entries
	cleanOldEntries(cacheDir)

	// Check cache
	cachePath := filepath.Join(cacheDir, videoID+".json")
	if entry, err := readCache(cachePath); err == nil {
		if time.Since(entry.FetchedAt) < cacheTTL {
			return entry.Segments, nil
		}
	}

	// Fetch from API
	segments, err := fetchSegments(ctx, videoID)
	if err != nil {
		return nil, err
	}

	// Cache result (including nil/empty)
	writeCache(cachePath, cacheEntry{
		FetchedAt: time.Now(),
		Segments:  segments,
	})

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
			Action:   s.ActionType,
		})
	}
	return segments, nil
}

func cacheDirectory() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "dis", "sponsorblock")
}

func readCache(path string) (*cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func writeCache(path string, entry cacheEntry) {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func cleanOldEntries(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > cleanTTL {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
