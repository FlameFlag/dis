// Package keys defines the key bindings used by the trim slider TUI.
package keys

import "github.com/charmbracelet/bubbles/key"

var (
	SelectStart = key.NewBinding(key.WithKeys("1"))
	SelectEnd   = key.NewBinding(key.WithKeys("2"))
	Tab         = key.NewBinding(key.WithKeys("tab"))
	Left        = key.NewBinding(key.WithKeys("left"))
	Right       = key.NewBinding(key.WithKeys("right"))
	ShiftLeft   = key.NewBinding(key.WithKeys("shift+left"))
	ShiftRight  = key.NewBinding(key.WithKeys("shift+right"))
	Up          = key.NewBinding(key.WithKeys("up"))
	Down        = key.NewBinding(key.WithKeys("down"))
	Space       = key.NewBinding(key.WithKeys(" "))
	Enter       = key.NewBinding(key.WithKeys("enter"))
	Escape      = key.NewBinding(key.WithKeys("esc"))

	// Transcript / select mode bindings.
	Search           = key.NewBinding(key.WithKeys("/"))
	NextCue          = key.NewBinding(key.WithKeys("]"))
	PrevCue          = key.NewBinding(key.WithKeys("["))
	NextMatch        = key.NewBinding(key.WithKeys("n"))
	PrevMatch        = key.NewBinding(key.WithKeys("N"))
	TranscriptSelect = key.NewBinding(key.WithKeys("t"))
	ParagraphSelect  = key.NewBinding(key.WithKeys("p"))
	SelectTrimRange  = key.NewBinding(key.WithKeys("a"))
	Deselect         = key.NewBinding(key.WithKeys("d"))

	// Viewport scrolling.
	PageUp   = key.NewBinding(key.WithKeys("pgup"))
	PageDown = key.NewBinding(key.WithKeys("pgdown"))

	// Splits.
	Split       = key.NewBinding(key.WithKeys("s"))
	DeleteSplit = key.NewBinding(key.WithKeys("d"))

	// Format toggles.
	GIFToggle   = key.NewBinding(key.WithKeys("g"))
	SpeedToggle = key.NewBinding(key.WithKeys("v"))

	Backspace = key.NewBinding(key.WithKeys("backspace"))
	Cancel    = key.NewBinding(key.WithKeys("ctrl+c"))
)
