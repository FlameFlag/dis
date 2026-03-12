package slider

import (
	"dis/internal/config"
	"dis/internal/sponsorblock"
	"dis/internal/subtitle"
	"dis/internal/util"
	"fmt"
	"math"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/log"
)

type sliderMode int

const (
	modeNormal sliderMode = iota
	modeInput
	modeSearch       // search active, entered from normal mode
	modeSearchSelect // search active, entered from select mode
	modeSelect
)

func (m Model) isSelectMode() bool {
	return m.mode == modeSelect || m.mode == modeSearchSelect
}

func (m Model) isSearchMode() bool {
	return m.mode == modeSearch || m.mode == modeSearchSelect
}

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

func (m Model) sliderWidth() int {
	w := m.leftPaneWidth() - 2 // 2 for inner padding
	if w < MinSliderWidth {
		w = MinSliderWidth
	}
	return w
}

func (m Model) isTwoPane() bool {
	return m.width >= MinTwoPaneWidth && (m.transcript != nil || len(m.waveform) > 0)
}

func (m Model) leftPaneWidth() int {
	if m.isTwoPane() {
		return m.width * LeftPaneRatio / 100
	}
	return m.width - 2 // single column: 1 border each side
}

func (m Model) rightPaneWidth() int {
	if !m.isTwoPane() {
		return 0
	}
	return m.width - m.leftPaneWidth() - 3 // 3 for border chars (│ left border + │ divider + │ right border)
}

func (m Model) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		if m.inputBuffer != "" {
			m.processTimeInput()
		}
		m.mode = modeNormal
		m.inputBuffer = ""
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.inputBuffer = ""
		return m, nil

	case key.Matches(msg, keys.Backspace):
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil

	default:
		ch := msg.String()
		if len(ch) == 1 && isValidTimeChar(rune(ch[0])) {
			m.inputBuffer += ch
		}
		return m, nil
	}
}

func (m Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Step-adjust keys: all follow the same pattern
	for _, s := range navigationSteps {
		if key.Matches(msg, s.binding) {
			m.adjustValue(s.step)
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
	}

	switch {
	case key.Matches(msg, keys.Escape):
		m.cancelled = true
		return m, tea.Quit

	case key.Matches(msg, keys.Enter):
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.SelectStart):
		m.adjustingStart = true
		return m, nil

	case key.Matches(msg, keys.SelectEnd):
		m.adjustingStart = false
		return m, nil

	case key.Matches(msg, keys.Tab):
		m.adjustingStart = !m.adjustingStart
		return m, nil

	case key.Matches(msg, keys.Space):
		m.mode = modeInput
		m.inputBuffer = ""
		return m, nil

	case key.Matches(msg, keys.PageUp):
		if m.transcript != nil {
			m.viewportLocked = false
			m.transcriptOffset -= TranscriptVisibleCues
			if m.transcriptOffset < 0 {
				m.transcriptOffset = 0
			}
		}
		return m, nil

	case key.Matches(msg, keys.PageDown):
		if m.transcript != nil {
			m.viewportLocked = false
			m.transcriptOffset += TranscriptVisibleCues
			maxOffset := len(m.transcript) - TranscriptVisibleCues
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.transcriptOffset > maxOffset {
				m.transcriptOffset = maxOffset
			}
		}
		return m, nil

	case key.Matches(msg, keys.Search):
		if m.transcript != nil {
			m.mode = modeSearch
			m.searchBuffer = ""
			m.searchResults = nil
			m.searchIndex = 0
		}
		return m, nil

	case key.Matches(msg, keys.NextCue):
		if m.transcript != nil {
			m.snapToNextCue()
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
		return m, nil

	case key.Matches(msg, keys.PrevCue):
		if m.transcript != nil {
			m.snapToPrevCue()
			m.viewportLocked = true
			return m, m.triggerAnim()
		}
		return m, nil

	case key.Matches(msg, keys.NextMatch):
		if m.transcript != nil && len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchResults)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if m.transcript != nil && len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchResults)) % len(m.searchResults)
			m.snapToCueSearchResult()
		}
		return m, nil

	case key.Matches(msg, keys.TranscriptSelect):
		if m.transcript != nil && len(m.words) > 0 {
			m.mode = modeSelect
			m.cursor = m.nearestWordIndex(m.activePos())
		}
		return m, nil

	case key.Matches(msg, keys.Split):
		// Save current range as a split (guard: end > start)
		if m.endPos > m.startPos {
			m.splits = append(m.splits, trimRange{start: m.startPos, end: m.endPos})
		}
		return m, nil

	case key.Matches(msg, keys.DeleteSplit):
		// Pop last saved split
		if len(m.splits) > 0 {
			m.splits = m.splits[:len(m.splits)-1]
		}
		return m, nil

	case key.Matches(msg, keys.GIFToggle):
		if !m.gifAvailable {
			m.warning = "gifski not found — install: brew install gifski"
			m.warningExpiry = time.Now().Add(2 * time.Second)
		} else {
			m.gifMode = !m.gifMode
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = modeNormal
		m.selectAnchor = -1
		m.searchResults = nil
		m.updateSliderFromSelection()
		return m, nil

	case key.Matches(msg, keys.Enter):
		m.mode = modeNormal
		m.confirmed = true
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		if m.cursor > 0 {
			m.cursor--
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Right):
		if m.cursor < len(m.words)-1 {
			m.cursor++
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.ShiftLeft):
		if m.cursor > 0 {
			if m.selectAnchor < 0 {
				m.selectAnchor = m.cursor
				m.selectAnchorSelecting = !m.selected[m.cursor]
			}
			m.selected[m.cursor] = m.selectAnchorSelecting
			m.cursor--
			m.selected[m.cursor] = m.selectAnchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.ShiftRight):
		if m.cursor < len(m.words)-1 {
			if m.selectAnchor < 0 {
				m.selectAnchor = m.cursor
				m.selectAnchorSelecting = !m.selected[m.cursor]
			}
			m.selected[m.cursor] = m.selectAnchorSelecting
			m.cursor++
			m.selected[m.cursor] = m.selectAnchorSelecting
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		// Jump to first word of previous cue
		if m.cursor > 0 {
			curCue := m.words[m.cursor].CueIndex
			i := m.cursor - 1
			// Skip remaining words in current cue
			for i > 0 && m.words[i].CueIndex == curCue {
				i--
			}
			// Find first word of that cue
			prevCue := m.words[i].CueIndex
			for i > 0 && m.words[i-1].CueIndex == prevCue {
				i--
			}
			m.cursor = i
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		// Jump to first word of next cue
		if m.cursor < len(m.words)-1 {
			curCue := m.words[m.cursor].CueIndex
			i := m.cursor + 1
			for i < len(m.words) && m.words[i].CueIndex == curCue {
				i++
			}
			if i < len(m.words) {
				m.cursor = i
			}
		}
		return m, nil

	case key.Matches(msg, keys.Space):
		m.selected[m.cursor] = !m.selected[m.cursor]
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.ParagraphSelect):
		m.toggleParagraph()
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Deselect):
		for i := range m.selected {
			m.selected[i] = false
		}
		m.selectAnchor = -1
		return m, nil

	case key.Matches(msg, keys.Search):
		m.mode = modeSearchSelect
		m.searchBuffer = ""
		m.searchResults = nil
		m.searchIndex = 0
		return m, nil

	case key.Matches(msg, keys.NextMatch):
		if len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchResults)
			m.cursor = m.searchResults[m.searchIndex]
		}
		return m, nil

	case key.Matches(msg, keys.PrevMatch):
		if len(m.searchResults) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchResults)) % len(m.searchResults)
			m.cursor = m.searchResults[m.searchIndex]
		}
		return m, nil

	case key.Matches(msg, keys.Cancel):
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		// Confirm search and snap to current match
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
			if len(m.searchResults) > 0 {
				m.cursor = m.searchResults[m.searchIndex]
			}
			return m, nil
		}
		m.mode = modeNormal
		m.snapToCueSearchResult()
		return m, m.triggerAnim()

	case key.Matches(msg, keys.Escape):
		if m.mode == modeSearchSelect {
			m.mode = modeSelect
		} else {
			m.mode = modeNormal
		}
		m.searchBuffer = ""
		m.searchResults = nil
		return m, nil

	case key.Matches(msg, keys.Backspace):
		if len(m.searchBuffer) > 0 {
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
			m.updateSearchResults()
		}
		return m, nil

	default:
		ch := msg.String()
		if len(ch) == 1 {
			m.searchBuffer += ch
			m.updateSearchResults()
		}
		return m, nil
	}
}

func (m *Model) updateSearchResults() {
	if m.searchBuffer == "" {
		m.searchResults = nil
		m.searchIndex = 0
		return
	}
	if m.mode == modeSearchSelect {
		m.searchResults = m.transcript.SearchWords(m.words, m.searchBuffer)
	} else {
		m.searchResults = m.transcript.Search(m.searchBuffer)
	}
	m.searchIndex = 0
}

func (m *Model) snapToCueSearchResult() {
	if len(m.searchResults) == 0 || m.transcript == nil {
		return
	}
	idx := m.searchResults[m.searchIndex]
	if idx >= 0 && idx < len(m.transcript) {
		cueStart := m.transcript[idx].Start
		if m.adjustingStart {
			m.startPos = math.Max(0, math.Min(m.endPos-MillisecondStep, cueStart))
		} else {
			cueEnd := m.transcript[idx].End
			m.endPos = math.Max(m.startPos+MillisecondStep, math.Min(m.duration, cueEnd))
		}
		m.roundPositions()
	}
}

func (m *Model) snapToNextCue() {
	pos := m.activePos()
	next := m.transcript.NextCueStart(pos)
	if next < 0 {
		return
	}
	// If rounding would produce the same position, skip to the next cue
	if math.Round(next*100)/100 <= math.Round(pos*100)/100 {
		next = m.transcript.NextCueStart(next + 0.001)
		if next < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = math.Min(m.endPos-MillisecondStep, next)
	} else {
		m.endPos = math.Min(m.duration, next)
	}
	m.roundPositions()
}

func (m *Model) snapToPrevCue() {
	pos := m.activePos()
	prev := m.transcript.PrevCueStart(pos)
	if prev < 0 {
		return
	}
	// If rounding would produce the same position, skip to the previous cue
	if math.Round(prev*100)/100 >= math.Round(pos*100)/100 {
		prev = m.transcript.PrevCueStart(prev - 0.001)
		if prev < 0 {
			return
		}
	}
	if m.adjustingStart {
		m.startPos = math.Max(0, prev)
	} else {
		m.endPos = math.Max(m.startPos+MillisecondStep, prev)
	}
	m.roundPositions()
}

func (m Model) nearestWordIndex(seconds float64) int {
	return util.NearestIndex(m.words, seconds, func(w subtitle.Word) float64 { return w.Start })
}

// sponsorCategoryAt returns the SponsorBlock category for a given timestamp, or empty string.
func (m Model) sponsorCategoryAt(seconds float64) sponsorblock.Category {
	for _, seg := range m.sponsorSegments {
		if seconds >= seg.Start && seconds < seg.End {
			return seg.Category
		}
	}
	return ""
}

func (m Model) isSilenceAt(seconds float64) bool {
	for _, si := range m.silenceIntervals {
		if seconds >= si.Start && seconds <= si.End {
			return true
		}
	}
	return false
}

// isSentenceEnd returns true if the word ends with sentence-ending punctuation.
func isSentenceEnd(word string) bool {
	trimmed := strings.TrimRight(word, `"')]}`)
	return strings.HasSuffix(trimmed, ".") ||
		strings.HasSuffix(trimmed, "?") ||
		strings.HasSuffix(trimmed, "!")
}

// toggleParagraph selects/deselects a sentence around the cursor, crossing cue boundaries.
func (m *Model) toggleParagraph() {
	if len(m.words) == 0 {
		return
	}

	// Search backward across all words for sentence-ending punctuation
	lo := 0
	for i := m.cursor - 1; i >= 0; i-- {
		if isSentenceEnd(m.words[i].Text) {
			lo = i + 1
			break
		}
	}

	// Search forward across all words for sentence-ending punctuation
	hi := len(m.words) - 1
	for i := m.cursor; i < len(m.words); i++ {
		if isSentenceEnd(m.words[i].Text) {
			hi = i
			break
		}
	}

	if lo > hi {
		return
	}

	// Majority toggle
	sel := 0
	for i := lo; i <= hi; i++ {
		if m.selected[i] {
			sel++
		}
	}
	newState := sel <= (hi-lo+1)/2
	for i := lo; i <= hi; i++ {
		m.selected[i] = newState
	}
}

func (m *Model) updateSliderFromSelection() {
	segs := m.selectedSegments()
	if len(segs) > 0 {
		m.startPos = segs[0].Start
		m.endPos = segs[len(segs)-1].End()
		m.roundPositions()
	}
}

func (m *Model) adjustValue(step float64) {
	if m.adjustingStart {
		newStart := m.startPos + step
		m.startPos = math.Max(0, math.Min(m.endPos-MillisecondStep, newStart))
	} else {
		newEnd := m.endPos + step
		m.endPos = math.Max(m.startPos+MillisecondStep, math.Min(m.duration, newEnd))
	}
	m.roundPositions()
}

func (m *Model) roundPositions() {
	m.startPos = math.Round(m.startPos*100) / 100
	m.endPos = math.Round(m.endPos*100) / 100
}

func (m *Model) processTimeInput() {
	seconds, err := util.ParseTimeValue(m.inputBuffer)
	if err != nil {
		return
	}

	if m.adjustingStart {
		if seconds >= 0 && seconds <= m.endPos-MillisecondStep {
			m.startPos = seconds
		}
	} else {
		if seconds >= m.startPos+MillisecondStep && seconds <= m.duration {
			m.endPos = seconds
		}
	}
	m.roundPositions()
}

func (m Model) activePos() float64 {
	if m.adjustingStart {
		return m.startPos
	}
	return m.endPos
}

// selectedSegments derives contiguous time ranges from the word selection.
func (m Model) selectedSegments() []config.TrimSettings {
	if len(m.words) == 0 {
		return nil
	}

	var segments []config.TrimSettings
	inSegment := false
	var segStart float64

	for i, sel := range m.selected {
		if sel && !inSegment {
			segStart = m.words[i].Start
			inSegment = true
		} else if !sel && inSegment {
			segments = append(segments, config.TrimSettings{
				Start:    segStart,
				Duration: m.words[i-1].End - segStart,
			})
			inSegment = false
		}
	}
	if inSegment {
		lastIdx := len(m.words) - 1
		for lastIdx >= 0 && !m.selected[lastIdx] {
			lastIdx--
		}
		if lastIdx >= 0 {
			segments = append(segments, config.TrimSettings{
				Start:    segStart,
				Duration: m.words[lastIdx].End - segStart,
			})
		}
	}

	for i, seg := range segments {
		log.Debug("Word selection segment",
			"idx", i, "start", seg.Start, "end", seg.End(),
			"duration", seg.Duration, "section", seg.DownloadSection())
	}

	return segments
}

// selectedWordCount returns the number of selected words.
func (m Model) selectedWordCount() int {
	n := 0
	for _, s := range m.selected {
		if s {
			n++
		}
	}
	return n
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

func (m Model) hasWordSelection() bool {
	return slices.Contains(m.selected, true)
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

func isValidTimeChar(c rune) bool {
	return unicode.IsDigit(c) || c == ':' || c == '.'
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
