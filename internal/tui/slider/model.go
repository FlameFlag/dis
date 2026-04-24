package slider

import (
	"dis/internal/config"
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/subtitle"
	"fmt"
	"math"
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

	// Select mode (word-level selection)
	cursor                int    // word cursor index
	selected              []bool // per-word selection state
	selectAnchor          int    // shift-select origin; -1 = no anchor
	selectAnchorSelecting bool   // true = shift-extend selects; false = deselects

	// Search mode
	searchInput   textinput.Model
	searchResults []int // matching indices (cue or word depending on mode)
	searchIndex   int   // current match position

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

	// Animation
	animSpring   harmonica.Spring
	animStartPos float64
	animStartVel float64
	animEndPos   float64
	animEndVel   float64
	animating    bool
}

// trimRange represents a single trim range with start and end times.
type trimRange struct {
	start float64
	end   float64
}

// StoryboardReadyMsg is sent when background storyboard fetch completes.
type StoryboardReadyMsg struct {
	Data *storyboard.StoryboardData
}

// TranscriptReadyMsg is sent when background transcript fetch completes.
type TranscriptReadyMsg struct {
	Transcript subtitle.Transcript
}

// SponsorSegsReadyMsg is sent when background SponsorBlock fetch completes.
type SponsorSegsReadyMsg struct {
	Segments []sponsorblock.Segment
}

type animTickMsg struct{}

func animTick() tea.Cmd {
	return tea.Tick(time.Second/AnimFPS, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
}

var brailleSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

func (m Model) isLoading() bool {
	return m.storyboardCh != nil ||
		m.transcriptCh != nil || m.sponsorSegsCh != nil
}

func (m *Model) triggerAnim() tea.Cmd {
	if !m.animating {
		m.animating = true
		return animTick()
	}
	return nil
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
		searchInput:     si,
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
		selectAnchor:    -1,
		gifMode:         gifEnabled,
		gifAvailable:    gifErr == nil,
		speedMultiplier: 1.0,
		animSpring:      harmonica.NewSpring(harmonica.FPS(AnimFPS), SpringFreq, SpringDamping),
		animStartPos:    0,
		animEndPos:      duration,
	}
}

// waitForChan returns a tea.Cmd that blocks on ch and wraps the received
// value with wrap. A closed channel produces wrap's zero-value message.
func waitForChan[T any, M tea.Msg](ch <-chan T, wrap func(T) M) tea.Cmd {
	return func() tea.Msg {
		v, ok := <-ch
		if !ok {
			var zero T
			return wrap(zero)
		}
		return wrap(v)
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.storyboardCh != nil {
		cmds = append(cmds, waitForChan(m.storyboardCh, func(d *storyboard.StoryboardData) StoryboardReadyMsg {
			return StoryboardReadyMsg{Data: d}
		}))
	}
	if m.transcriptCh != nil {
		cmds = append(cmds, waitForChan(m.transcriptCh, func(t subtitle.Transcript) TranscriptReadyMsg {
			return TranscriptReadyMsg{Transcript: t}
		}))
	}
	if m.sponsorSegsCh != nil {
		cmds = append(cmds, waitForChan(m.sponsorSegsCh, func(s []sponsorblock.Segment) SponsorSegsReadyMsg {
			return SponsorSegsReadyMsg{Segments: s}
		}))
	}

	if m.isLoading() {
		cmds = append(cmds, m.loadingSpinner.Tick)
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case StoryboardReadyMsg:
		m.storyboard = msg.Data
		m.storyboardCh = nil
		return m, nil
	case TranscriptReadyMsg:
		m.transcript = msg.Transcript
		m.transcriptCh = nil
		if len(m.transcript) > 0 {
			m.words = m.transcript.Words()
			m.selected = make([]bool, len(m.words))
		}
		return m, nil
	case SponsorSegsReadyMsg:
		m.sponsorSegments = msg.Segments
		m.sponsorSegsCh = nil
		return m, nil
	case spinner.TickMsg:
		if m.isLoading() {
			var cmd tea.Cmd
			m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case animTickMsg:
		if m.warning != "" && time.Now().After(m.warningExpiry) {
			m.warning = ""
		}
		if !m.animating {
			return m, nil
		}
		m.animStartPos, m.animStartVel = m.animSpring.Update(m.animStartPos, m.animStartVel, m.startPos)
		m.animEndPos, m.animEndVel = m.animSpring.Update(m.animEndPos, m.animEndVel, m.endPos)
		// Settle threshold: half a column width in seconds
		threshold := m.duration / float64(m.sliderWidth()) / 2
		startSettled := math.Abs(m.animStartPos-m.startPos) < threshold && math.Abs(m.animStartVel) < threshold
		endSettled := math.Abs(m.animEndPos-m.endPos) < threshold && math.Abs(m.animEndVel) < threshold
		if startSettled && endSettled {
			m.animStartPos = m.startPos
			m.animEndPos = m.endPos
			m.animStartVel = 0
			m.animEndVel = 0
			m.animating = false
			return m, nil
		}
		return m, animTick()
	case tea.KeyMsg:
		switch m.mode {
		case modeSearch, modeSearchSelect:
			return m.handleSearchMode(msg)
		case modeInput:
			return m.handleInputMode(msg)
		case modeSelect:
			return m.handleSelectMode(msg)
		default:
			return m.handleNavigation(msg)
		}
	}
	// Route cursor blink messages to active textinput
	switch {
	case m.isSearchMode():
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	case m.mode == modeInput:
		var cmd tea.Cmd
		m.timeInput, cmd = m.timeInput.Update(msg)
		return m, cmd
	}
	return m, nil
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
