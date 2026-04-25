// Package style centralises the lipgloss styles, color gradients, and
// SponsorBlock category palette used by the slider TUI.
package style

import (
	"dis/internal/sponsorblock"
	"dis/internal/tui"

	"github.com/charmbracelet/lipgloss"
)

var (
	Accent       = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	AccentBold   = lipgloss.NewStyle().Foreground(tui.ColorPeach).Bold(true)
	Warm         = lipgloss.NewStyle().Foreground(tui.ColorYellow)
	Value        = lipgloss.NewStyle().Foreground(tui.ColorTeal)
	Dim          = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	Faint        = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
	Bold         = lipgloss.NewStyle().Bold(true)
	Reverse      = lipgloss.NewStyle().Reverse(true)
	HelpPill     = lipgloss.NewStyle().Background(tui.ColorSurface1).Foreground(tui.ColorText)
	Border       = lipgloss.NewStyle().Foreground(tui.ColorSurface2)
	Warn         = lipgloss.NewStyle().Foreground(tui.ColorRed)

	HandleActive   = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	HandleInactive = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)

	SelectedTrack   = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	UnselectedTrack = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
)

// Fade is the gradient for transcript cues below the active one (progressively dimmer).
var Fade = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(tui.ColorSubtext0), // 0: slight fade
	lipgloss.NewStyle().Foreground(tui.ColorOverlay0), // 1: moderate
	lipgloss.NewStyle().Foreground(tui.ColorSurface2), // 2: dim
	lipgloss.NewStyle().Foreground(tui.ColorSurface1), // 3: very dim
	lipgloss.NewStyle().Foreground(tui.ColorFadeEnd),  // 4: near-invisible
}

// Track is the gradient for the selected region edges (fade-in/fade-out at boundaries).
var Track = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(tui.ColorTrackDim),  // 0: dim accent
	lipgloss.NewStyle().Foreground(tui.ColorTrackMid),  // 1: mid accent
	lipgloss.NewStyle().Foreground(tui.ColorTrackWarm), // 2: warm accent
}

// SponsorCategory bundles the rendering attributes for a SponsorBlock category.
type SponsorCategory struct {
	Color    lipgloss.Style
	HexColor lipgloss.Color // raw color for track overlay
	Label    string
}

var SponsorCategories = map[sponsorblock.Category]SponsorCategory{
	sponsorblock.CategorySponsor:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00d400")), HexColor: "#00d400", Label: "spon"},
	sponsorblock.CategoryIntro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")), HexColor: "#00ffff", Label: "intr"},
	sponsorblock.CategoryOutro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#0202ed")), HexColor: "#0202ed", Label: "outr"},
	sponsorblock.CategorySelfPromo:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")), HexColor: "#ffff00", Label: "self"},
	sponsorblock.CategoryInteraction:   {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#cc00ff")), HexColor: "#cc00ff", Label: "intr"},
	sponsorblock.CategoryMusicOfftopic: {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9900")), HexColor: "#ff9900", Label: "musc"},
	sponsorblock.CategoryPreview:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#008fd6")), HexColor: "#008fd6", Label: "prev"},
	sponsorblock.CategoryHighlight:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#82")), HexColor: "#82", Label: "high"},
	sponsorblock.CategoryFiller:        {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#7300FF")), HexColor: "#7300FF", Label: "fill"},
}
