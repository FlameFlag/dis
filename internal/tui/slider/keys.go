package slider

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	SelectStart key.Binding
	SelectEnd   key.Binding
	Tab         key.Binding
	Left        key.Binding
	Right       key.Binding
	ShiftLeft   key.Binding
	ShiftRight  key.Binding
	Up          key.Binding
	Down        key.Binding
	Space       key.Binding
	Enter       key.Binding
	Escape      key.Binding

	// Transcript / select mode bindings
	Search           key.Binding
	NextCue          key.Binding
	PrevCue          key.Binding
	NextMatch        key.Binding
	PrevMatch        key.Binding
	TranscriptSelect key.Binding
	ParagraphSelect  key.Binding
	Deselect         key.Binding

	// Viewport scrolling
	PageUp   key.Binding
	PageDown key.Binding

	// Splits
	Split       key.Binding
	DeleteSplit key.Binding

	// Format toggle
	GIFToggle key.Binding

	Backspace key.Binding
	Cancel    key.Binding
}

var keys = keyMap{
	SelectStart: key.NewBinding(key.WithKeys("1")),
	SelectEnd:   key.NewBinding(key.WithKeys("2")),
	Tab:         key.NewBinding(key.WithKeys("tab")),
	Left:        key.NewBinding(key.WithKeys("left")),
	Right:       key.NewBinding(key.WithKeys("right")),
	ShiftLeft:   key.NewBinding(key.WithKeys("shift+left")),
	ShiftRight:  key.NewBinding(key.WithKeys("shift+right")),
	Up:          key.NewBinding(key.WithKeys("up")),
	Down:        key.NewBinding(key.WithKeys("down")),
	Space:       key.NewBinding(key.WithKeys(" ")),
	Enter:       key.NewBinding(key.WithKeys("enter")),
	Escape:      key.NewBinding(key.WithKeys("esc")),

	Search:           key.NewBinding(key.WithKeys("/")),
	NextCue:          key.NewBinding(key.WithKeys("]")),
	PrevCue:          key.NewBinding(key.WithKeys("[")),
	NextMatch:        key.NewBinding(key.WithKeys("n")),
	PrevMatch:        key.NewBinding(key.WithKeys("N")),
	TranscriptSelect: key.NewBinding(key.WithKeys("t")),
	ParagraphSelect:  key.NewBinding(key.WithKeys("p")),
	Deselect:         key.NewBinding(key.WithKeys("d")),

	PageUp:   key.NewBinding(key.WithKeys("pgup")),
	PageDown: key.NewBinding(key.WithKeys("pgdown")),

	Split:       key.NewBinding(key.WithKeys("s")),
	DeleteSplit: key.NewBinding(key.WithKeys("d")),

	GIFToggle: key.NewBinding(key.WithKeys("g")),

	Backspace: key.NewBinding(key.WithKeys("backspace")),
	Cancel:    key.NewBinding(key.WithKeys("ctrl+c")),
}
