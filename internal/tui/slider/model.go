package slider

import (
	"context"
	"fmt"

	"github.com/4evy/dis/internal/media"
	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/storyboard"
	"github.com/4evy/dis/internal/subtitle"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type sliderMode int

const (
	modeNormal sliderMode = iota
	modeInput
	modeSearch       // search active, entered from normal mode
	modeSearchSelect // search active, entered from select mode
	modeSelect
)

// ChapterMarker represents a chapter boundary on the slider.
type ChapterMarker struct {
	StartTime float64
	Title     string
}

// Options contains all synchronous and asynchronous slider inputs.
type Options struct {
	Duration        float64
	Chapters        []media.Chapter
	Transcript      <-chan subtitle.Transcript
	Storyboard      <-chan *storyboard.StoryboardData
	SponsorSegments <-chan []sponsorblock.Segment
	GIFAvailable    bool
	GIF             bool
}

// State contains non-clip choices made in the slider.
type State struct {
	GIF   bool
	Speed float64
}

// selectState holds word-level selection mode state.
type selectState struct {
	cursor          int    // word cursor index
	selected        []bool // per-word selection state
	anchor          int    // shift-select origin; -1 = no anchor
	anchorSelecting bool   // true = shift-extend selects; false = deselects
}

// searchState holds search mode UI state.
type searchState struct {
	input   textinput.Model
	results []int // matching indices (cue or word depending on mode)
	index   int   // current match position
}

// Model is the BubbleTea model for the trim slider.
type Model struct {
	timelineModel
	transcriptModel
	searchModel
	helpModel

	mode      sliderMode
	confirmed bool
	cancelled bool
	width     int
	height    int

	loadingSpinner spinner.Model

	storyboard   *storyboard.StoryboardData
	storyboardCh <-chan *storyboard.StoryboardData

	sponsorSegments []sponsorblock.Segment
	sponsorSegsCh   <-chan []sponsorblock.Segment

	gifMode         bool
	gifAvailable    bool
	speedMultiplier float64
	warning         string
}

type timelineModel struct {
	duration       float64
	startPos       float64
	endPos         float64
	adjustingStart bool
	timeInput      textinput.Model
	chapters       []ChapterMarker
	splits         []trimRange
}

type helpModel struct {
	keyHelp     help.Model
	helpVisible bool
	helpScroll  int
}

type transcriptModel struct {
	transcript   subtitle.Transcript // nil until received
	transcriptCh <-chan subtitle.Transcript
	words        []subtitle.Word // flattened word list

	sel              selectState
	viewportLocked   bool // auto-follow mode (default true)
	transcriptOffset int  // scroll offset in cues (used when unlocked)
}

type searchModel struct {
	search searchState
}

// trimRange represents a single trim range with start and end times.
type trimRange struct {
	start float64
	end   float64
}

func (m Model) isLoading() bool {
	return m.storyboardCh != nil ||
		m.transcriptCh != nil || m.sponsorSegsCh != nil
}

// New creates a new trim slider model.
func New(options Options) Model {
	chapters := make([]ChapterMarker, 0, len(options.Chapters))
	for _, chapter := range options.Chapters {
		chapters = append(chapters, ChapterMarker{
			StartTime: chapter.Clip.Start,
			Title:     chapter.Title,
		})
	}
	si := textinput.New()
	si.Prompt = ""
	si.Placeholder = "title or phrase"
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "mm:ss.mmm"
	ti.CharLimit = timeInputCharacterLimit
	ti.Validate = validateTimeInput
	kh := help.New()
	kh.ShortSeparator = "  ·  "
	kh.Styles.ShortKey = HelpKey
	kh.Styles.ShortDesc = HelpDesc
	kh.Styles.ShortSeparator = HelpSep
	kh.Styles.FullKey = HelpKey
	kh.Styles.FullDesc = HelpDesc
	kh.Styles.Ellipsis = HelpSep
	return Model{
		timeInput: ti, duration: options.Duration,
		endPos: options.Duration, adjustingStart: true, chapters: chapters,
		transcriptCh:    options.Transcript,
		viewportLocked:  true,
		sel:             selectState{anchor: -1},
		search:          searchState{input: si},
		keyHelp:         kh,
		loadingSpinner:  spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		storyboardCh:    options.Storyboard,
		sponsorSegsCh:   options.SponsorSegments,
		gifMode:         options.GIF,
		gifAvailable:    options.GIFAvailable,
		speedMultiplier: playbackSpeeds[0],
	}
}

// Result returns neutral clips, or nil when the slider was cancelled.
func (m Model) Result() []media.Clip {
	if m.cancelled {
		return nil
	}

	// Saved splits take priority
	if len(m.splits) > 0 {
		clips := make([]media.Clip, 0, len(m.splits))
		for _, r := range m.splits {
			clips = append(clips, media.Clip{
				Start:    r.start,
				Duration: r.end - r.start,
			})
		}
		return clips
	}

	// If word selection was used and has selections, use those segments
	if m.mode == modeSelect || m.hasWordSelection() {
		segs := m.selectedSegments()
		if len(segs) > 0 {
			return segs
		}
	}

	// Default: single segment from slider handles
	return []media.Clip{
		{
			Start:    m.startPos,
			Duration: m.endPos - m.startPos,
		},
	}
}

// State returns the current format choices.
func (m Model) State() State {
	return State{GIF: m.gifMode, Speed: m.speedMultiplier}
}

// Run launches the trim slider as a full-screen BubbleTea program.
func Run(ctx context.Context, options Options) ([]media.Clip, State, error) {
	m := New(options)
	p := tea.NewProgram(m, tea.WithContext(ctx))

	finalModel, err := p.Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, State{}, ctx.Err()
		}
		return nil, State{}, fmt.Errorf("trim slider error: %w", err)
	}

	m, ok := finalModel.(Model)
	if !ok {
		return nil, State{}, fmt.Errorf(
			"trim slider: unexpected model type %T",
			finalModel,
		)
	}
	return m.Result(), m.State(), nil
}

// stepBinding maps a key binding to a time-step value for slider adjustment.
type stepBinding struct {
	binding key.Binding
	step    float64
}

// navigationSteps defines all keys that adjust the slider position by a fixed step.
var navigationSteps = []stepBinding{
	{Left, -SecondStep},
	{Right, SecondStep},
	{ShiftLeft, -MillisecondStep},
	{ShiftRight, MillisecondStep},
	{Up, MinuteStep},
	{Down, -MinuteStep},
}
