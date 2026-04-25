package slider

import (
	"dis/internal/config"
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/subtitle"
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
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

// TrimResult holds one or more trim segments from the slider.
type TrimResult struct {
	Segments []config.TrimSettings
	GIF      bool
	Speed    float64
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

// animState holds the spring-driven slider animation.
type animState struct {
	spring   harmonica.Spring
	startPos float64
	startVel float64
	endPos   float64
	endVel   float64
	active   bool
}

// Model is the BubbleTea model for the trim slider.
type Model struct {
	duration       float64
	startPos       float64
	endPos         float64
	adjustingStart bool
	mode           sliderMode
	timeInput      textinput.Model
	confirmed      bool
	cancelled      bool
	width          int
	chapters       []ChapterMarker

	// Loading spinner
	loadingSpinner spinner.Model

	// Transcript support (async)
	transcript   subtitle.Transcript // nil until received
	transcriptCh <-chan subtitle.Transcript
	words        []subtitle.Word // flattened word list

	sel    selectState
	search searchState

	// Storyboard (async)
	storyboard   *storyboard.StoryboardData
	storyboardCh <-chan *storyboard.StoryboardData

	// SponsorBlock segments (async)
	sponsorSegments []sponsorblock.Segment
	sponsorSegsCh   <-chan []sponsorblock.Segment

	// Transcript viewport
	viewportLocked   bool // auto-follow mode (default true)
	transcriptOffset int  // scroll offset in cues (used when unlocked)

	// Saved splits
	splits []trimRange

	// GIF export
	gifMode         bool
	gifAvailable    bool
	speedMultiplier float64
	warning         string
	warningExpiry   time.Time

	// Terminal height (for conditional thumbnail rendering)
	height int

	anim animState
}

// trimRange represents a single trim range with start and end times.
type trimRange struct {
	start float64
	end   float64
}

var brailleSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

func (m Model) isLoading() bool {
	return m.storyboardCh != nil ||
		m.transcriptCh != nil || m.sponsorSegsCh != nil
}

// New creates a new trim slider model.
func New(duration float64, transcriptCh <-chan subtitle.Transcript, storyboardCh <-chan *storyboard.StoryboardData, sponsorSegsCh <-chan []sponsorblock.Segment, gifEnabled bool, chapters ...ChapterMarker) Model {
	_, gifErr := exec.LookPath("gifski")
	si := textinput.New()
	si.Prompt = ""
	ti := textinput.New()
	ti.Prompt = ""
	ti.Validate = validateTimeInput
	return Model{
		loadingSpinner:  spinner.New(spinner.WithSpinner(brailleSpinner)),
		search:          searchState{input: si},
		timeInput:       ti,
		duration:        duration,
		startPos:        0,
		endPos:          duration,
		adjustingStart:  true,
		chapters:        chapters,
		transcriptCh:    transcriptCh,
		storyboardCh:    storyboardCh,
		sponsorSegsCh:   sponsorSegsCh,
		viewportLocked:  true,
		sel:             selectState{anchor: -1},
		gifMode:         gifEnabled,
		gifAvailable:    gifErr == nil,
		speedMultiplier: 1.0,
		anim: animState{
			spring: harmonica.NewSpring(harmonica.FPS(AnimFPS), SpringFreq, SpringDamping),
			endPos: duration,
		},
	}
}

// Result returns the TrimResult if confirmed, nil if cancelled.
func (m Model) Result() *TrimResult {
	if m.cancelled {
		return nil
	}

	result := &TrimResult{GIF: m.gifMode, Speed: m.speedMultiplier}

	// Saved splits take priority
	if len(m.splits) > 0 {
		for _, r := range m.splits {
			result.Segments = append(result.Segments, config.TrimSettings{
				Start:    r.start,
				Duration: r.end - r.start,
			})
		}
		return result
	}

	// If word selection was used and has selections, use those segments
	if m.mode == modeSelect || m.hasWordSelection() {
		segs := m.selectedSegments()
		if len(segs) > 0 {
			result.Segments = segs
			return result
		}
	}

	// Default: single segment from slider handles
	result.Segments = []config.TrimSettings{
		{
			Start:    m.startPos,
			Duration: m.endPos - m.startPos,
		},
	}
	return result
}

// Run launches the trim slider as a full-screen BubbleTea program.
func Run(duration float64, transcriptCh <-chan subtitle.Transcript, storyboardCh <-chan *storyboard.StoryboardData, sponsorSegsCh <-chan []sponsorblock.Segment, gifEnabled bool, chapters ...ChapterMarker) (*TrimResult, error) {
	m := New(duration, transcriptCh, storyboardCh, sponsorSegsCh, gifEnabled, chapters...)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("trim slider error: %w", err)
	}

	m, ok := finalModel.(Model)
	if !ok {
		return nil, fmt.Errorf("trim slider: unexpected model type %T", finalModel)
	}
	result := m.Result()
	return result, nil
}

// stepBinding maps a key binding to a time-step value for slider adjustment.
type stepBinding struct {
	binding key.Binding
	step    float64
}

// navigationSteps defines all keys that adjust the slider position by a fixed step.
var navigationSteps = []stepBinding{
	{keys.Left, -SecondStep},
	{keys.Right, SecondStep},
	{keys.ShiftLeft, -MillisecondStep},
	{keys.ShiftRight, MillisecondStep},
	{keys.Up, MinuteStep},
	{keys.Down, -MinuteStep},
}
