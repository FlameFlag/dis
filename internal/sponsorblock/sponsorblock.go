package sponsorblock

import (
	"slices"
	"strings"
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

const supportedExtractor = "youtube"

// SupportsExtractor reports whether SponsorBlock accepts IDs from a yt-dlp
// extractor.
func SupportsExtractor(extractor string) bool {
	return strings.EqualFold(extractor, supportedExtractor)
}

// AllCategories returns all known SponsorBlock categories.
var allCategories = [...]Category{
	CategorySponsor, CategoryIntro, CategoryOutro, CategorySelfPromo,
	CategoryInteraction, CategoryMusicOfftopic, CategoryPreview,
	CategoryHighlight, CategoryFiller,
}

func AllCategories() []Category { return slices.Clone(allCategories[:]) }

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
	apiBase           = "https://sponsor.ajay.app/api/skipSegments"
	segmentStartIndex = 0
	segmentEndIndex   = 1
	segmentBounds     = 2
)

// apiResponse is the JSON structure returned by the SponsorBlock API.
type apiResponse struct {
	Segment    [segmentBounds]float64 `json:"segment"`
	Category   Category               `json:"category"`
	ActionType string                 `json:"actionType"`
}
