package slider

import (
	"image/color"

	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/tui"

	"charm.land/lipgloss/v2"
)

const (
	modalVerticalPadding   = 1
	modalHorizontalPadding = 2
)

var (
	Accent     = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	AccentBold = lipgloss.NewStyle().Foreground(tui.ColorPeach).Bold(true)
	Warm       = lipgloss.NewStyle().Foreground(tui.ColorYellow)
	Value      = lipgloss.NewStyle().Foreground(tui.ColorTeal)
	Dim        = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	Faint      = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
	Bold       = lipgloss.NewStyle().Bold(true)
	Reverse    = lipgloss.NewStyle().Reverse(true)
	Border     = lipgloss.NewStyle().Foreground(tui.ColorSurface2)
	Warn       = lipgloss.NewStyle().Foreground(tui.ColorRed)
	HelpKey    = lipgloss.NewStyle().Foreground(tui.ColorPeach).Bold(true)
	HelpDesc   = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
	HelpSep    = lipgloss.NewStyle().Foreground(tui.ColorSurface2)
	Modal      = lipgloss.NewStyle().
			Background(tui.ColorBase).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tui.ColorPeach).
			Padding(modalVerticalPadding, modalHorizontalPadding)

	HandleActive   = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	HandleInactive = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)

	SelectedTrack   = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	UnselectedTrack = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
)

var Fade = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(tui.ColorSubtext0),
	lipgloss.NewStyle().Foreground(tui.ColorOverlay0),
	lipgloss.NewStyle().Foreground(tui.ColorSurface2),
	lipgloss.NewStyle().Foreground(tui.ColorSurface1),
	lipgloss.NewStyle().Foreground(tui.ColorFadeEnd),
}

var Track = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(tui.ColorTrackDim),
	lipgloss.NewStyle().Foreground(tui.ColorTrackMid),
	lipgloss.NewStyle().Foreground(tui.ColorTrackWarm),
}

type SponsorCategory struct {
	Color    lipgloss.Style
	HexColor color.Color
	Glyph    string
	Label    string
}

var SponsorCategories = map[sponsorblock.Category]SponsorCategory{
	sponsorblock.CategorySponsor:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00d400")), HexColor: lipgloss.Color("#00d400"), Glyph: "s", Label: "sponsor"},
	sponsorblock.CategoryIntro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")), HexColor: lipgloss.Color("#00ffff"), Glyph: "i", Label: "intro"},
	sponsorblock.CategoryOutro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#0202ed")), HexColor: lipgloss.Color("#0202ed"), Glyph: "o", Label: "outro"},
	sponsorblock.CategorySelfPromo:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")), HexColor: lipgloss.Color("#ffff00"), Glyph: "p", Label: "self-promo"},
	sponsorblock.CategoryInteraction:   {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#cc00ff")), HexColor: lipgloss.Color("#cc00ff"), Glyph: "a", Label: "interaction"},
	sponsorblock.CategoryMusicOfftopic: {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9900")), HexColor: lipgloss.Color("#ff9900"), Glyph: "m", Label: "music"},
	sponsorblock.CategoryPreview:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#008fd6")), HexColor: lipgloss.Color("#008fd6"), Glyph: "v", Label: "preview"},
	sponsorblock.CategoryHighlight:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#82")), HexColor: lipgloss.Color("#82"), Glyph: "★", Label: "highlight"},
	sponsorblock.CategoryFiller:        {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#7300FF")), HexColor: lipgloss.Color("#7300FF"), Glyph: "f", Label: "filler"},
}
