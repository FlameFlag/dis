package slider

import (
	"dis/internal/sponsorblock"
	"dis/internal/tui"

	"github.com/charmbracelet/lipgloss"
)

var (
	accentStyle   = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	accentBold    = lipgloss.NewStyle().Foreground(tui.ColorPeach).Bold(true)
	warmStyle     = lipgloss.NewStyle().Foreground(tui.ColorYellow)
	valueStyle    = lipgloss.NewStyle().Foreground(tui.ColorTeal)
	dimStyle      = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	faintStyle    = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
	boldStyle     = lipgloss.NewStyle().Bold(true)
	reverseStyle  = lipgloss.NewStyle().Reverse(true)
	helpKeyStyle  = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	helpPillStyle = lipgloss.NewStyle().Background(tui.ColorSurface1).Foreground(tui.ColorText)
	borderStyle   = lipgloss.NewStyle().Foreground(tui.ColorSurface2)
	warnStyle     = lipgloss.NewStyle().Foreground(tui.ColorRed)
)

// Fade gradient for transcript cues below the active one (progressively dimmer).
var fadeGradient = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(tui.ColorSubtext0),         // 0: slight fade
	lipgloss.NewStyle().Foreground(tui.ColorOverlay0),         // 1: moderate
	lipgloss.NewStyle().Foreground(tui.ColorSurface2),         // 2: dim
	lipgloss.NewStyle().Foreground(tui.ColorSurface1),         // 3: very dim
	lipgloss.NewStyle().Foreground(lipgloss.Color("#3b3f52")), // 4: near-invisible
}

// Track gradient for the selected region edges (#2: fade-in/fade-out at selection boundaries).
var trackGradient = []lipgloss.Style{
	lipgloss.NewStyle().Foreground(lipgloss.Color("#8b6a50")), // 0: dim peach
	lipgloss.NewStyle().Foreground(lipgloss.Color("#c08a68")), // 1: mid peach
	lipgloss.NewStyle().Foreground(lipgloss.Color("#daa078")), // 2: warm peach
}

var (
	selectedTrack       = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	unselectedTrack     = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	handleActiveStyle   = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	handleInactiveStyle = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
)

type sponsorCategoryStyle struct {
	Color    lipgloss.Style
	HexColor lipgloss.Color // raw color for track overlay
	Label    string
}

var sponsorCategories = map[sponsorblock.Category]sponsorCategoryStyle{
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
