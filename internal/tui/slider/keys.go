package slider

import "charm.land/bubbles/v2/key"

var (
	SelectStart = key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "edit start"))
	SelectEnd   = key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "edit end"))
	Tab         = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch edge"))
	Left        = key.NewBinding(key.WithKeys("left"), key.WithHelp("←/→", "adjust 1 second"))
	Right       = key.NewBinding(key.WithKeys("right"))
	ShiftLeft   = key.NewBinding(key.WithKeys("shift+left"), key.WithHelp("shift+←/→", "adjust 10 ms"))
	ShiftRight  = key.NewBinding(key.WithKeys("shift+right"))
	Up          = key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "adjust 1 minute"))
	Down        = key.NewBinding(key.WithKeys("down"))
	Space       = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "enter exact time"))
	Enter       = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply trim"))
	Escape      = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	Help        = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "all keys"))
	Quit        = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))

	Search           = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search transcript"))
	NextCue          = key.NewBinding(key.WithKeys("]"), key.WithHelp("[/]", "snap to cue"))
	PrevCue          = key.NewBinding(key.WithKeys("["))
	NextMatch        = key.NewBinding(key.WithKeys("n"), key.WithHelp("n/N", "next/previous match"))
	PrevMatch        = key.NewBinding(key.WithKeys("N"))
	TranscriptSelect = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "select words"))
	ParagraphSelect  = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "toggle sentence"))
	SelectTrimRange  = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select trim range"))
	Deselect         = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "clear selection"))

	PageUp   = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup/pgdn", "browse transcript"))
	PageDown = key.NewBinding(key.WithKeys("pgdown"))

	Split       = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "add range"))
	DeleteSplit = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "undo range"))

	GIFToggle   = key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "toggle GIF"))
	SpeedToggle = key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "cycle speed"))

	Backspace = key.NewBinding(key.WithKeys("backspace"))
	Cancel    = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit anywhere"))
)
