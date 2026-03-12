package slider

import (
	"dis/internal/config"
	"dis/internal/sponsorblock"
	"dis/internal/subtitle"
	"fmt"
	"math"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/key"
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
}

// Model is the BubbleTea model for the trim slider.
type Model struct {
	duration       float64
	startPos       float64
	endPos         float64
	adjustingStart bool
	mode           sliderMode
	inputBuffer    string
	confirmed      bool
	cancelled      bool
	width          int
	chapters       []ChapterMarker

	// Transcript support
	transcript subtitle.Transcript // nil if unavailable
	words      []subtitle.Word     // flattened word list

	// Select mode (word-level selection)
	cursor                int    // word cursor index
	selected              []bool // per-word selection state
	selectAnchor          int    // shift-select origin; -1 = no anchor
	selectAnchorSelecting bool   // true = shift-extend selects; false = deselects

	// Search mode
	searchBuffer  string
	searchResults []int // matching indices (cue or word depending on mode)
	searchIndex   int   // current match position

	// Silence detection (async)
	silenceIntervals []subtitle.SilenceInterval
	silenceCh        <-chan []subtitle.SilenceInterval

	// Waveform data (async)
	waveform   []subtitle.WaveformSample
	waveformCh <-chan []subtitle.WaveformSample

	// SponsorBlock segments
	sponsorSegments []sponsorblock.Segment

	// Transcript viewport
	viewportLocked   bool // auto-follow mode (default true)
	transcriptOffset int  // scroll offset in cues (used when unlocked)

	// Saved splits
	splits []trimRange

	// GIF export
	gifMode       bool
	gifAvailable  bool
	warning       string
	warningExpiry time.Time

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

// SilenceDetectedMsg is sent when background silence detection completes.
type SilenceDetectedMsg struct {
	Intervals []subtitle.SilenceInterval
}

// WaveformReadyMsg is sent when background waveform extraction completes.
type WaveformReadyMsg struct {
	Samples []subtitle.WaveformSample
}

type animTickMsg struct{}

func animTick() tea.Cmd {
	return tea.Tick(time.Second/AnimFPS, func(time.Time) tea.Msg {
		return animTickMsg{}
	})
}

func (m *Model) triggerAnim() tea.Cmd {
	if !m.animating {
		m.animating = true
		return animTick()
	}
	return nil
}

// New creates a new trim slider model.
func New(duration float64, transcript subtitle.Transcript, silenceCh <-chan []subtitle.SilenceInterval, waveformCh <-chan []subtitle.WaveformSample, sponsorSegs []sponsorblock.Segment, gifEnabled bool, chapters ...ChapterMarker) Model {
	_, gifErr := exec.LookPath("gifski")
	m := Model{
		duration:        duration,
		startPos:        0,
		endPos:          duration,
		adjustingStart:  true,
		chapters:        chapters,
		transcript:      transcript,
		silenceCh:       silenceCh,
		waveformCh:      waveformCh,
		sponsorSegments: sponsorSegs,
		viewportLocked:  true,
		selectAnchor:    -1,
		gifMode:         gifEnabled,
		gifAvailable:    gifErr == nil,
		animSpring:      harmonica.NewSpring(harmonica.FPS(AnimFPS), SpringFreq, SpringDamping),
		animStartPos:    0,
		animEndPos:      duration,
	}
	if len(transcript) > 0 {
		m.words = transcript.Words()
		m.selected = make([]bool, len(m.words))
	}
	return m
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.silenceCh != nil {
		ch := m.silenceCh
		cmds = append(cmds, func() tea.Msg {
			intervals, ok := <-ch
			if !ok || len(intervals) == 0 {
				return SilenceDetectedMsg{}
			}
			return SilenceDetectedMsg{Intervals: intervals}
		})
	}

	if m.waveformCh != nil {
		ch := m.waveformCh
		cmds = append(cmds, func() tea.Msg {
			samples, ok := <-ch
			if !ok || len(samples) == 0 {
				return WaveformReadyMsg{}
			}
			return WaveformReadyMsg{Samples: samples}
		})
	}

	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case SilenceDetectedMsg:
		m.silenceIntervals = msg.Intervals
		return m, nil
	case WaveformReadyMsg:
		m.waveform = msg.Samples
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
	return m, nil
}

// Result returns the TrimResult if confirmed, nil if cancelled.
func (m Model) Result() *TrimResult {
	if m.cancelled {
		return nil
	}

	result := &TrimResult{GIF: m.gifMode}

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
func Run(duration float64, transcript subtitle.Transcript, silenceCh <-chan []subtitle.SilenceInterval, waveformCh <-chan []subtitle.WaveformSample, sponsorSegs []sponsorblock.Segment, gifEnabled bool, chapters ...ChapterMarker) (*TrimResult, error) {
	m := New(duration, transcript, silenceCh, waveformCh, sponsorSegs, gifEnabled, chapters...)
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
