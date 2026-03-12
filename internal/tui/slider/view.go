package slider

import (
	"dis/internal/sponsorblock"
	"dis/internal/tui"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	accentStyle  = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	accentBold   = lipgloss.NewStyle().Foreground(tui.ColorPeach).Bold(true)
	warmStyle    = lipgloss.NewStyle().Foreground(tui.ColorYellow)
	valueStyle   = lipgloss.NewStyle().Foreground(tui.ColorTeal)
	dimStyle     = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	faintStyle   = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
	boldStyle    = lipgloss.NewStyle().Bold(true)
	reverseStyle = lipgloss.NewStyle().Reverse(true)
	helpKeyStyle = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
)

func bufHasOverlap(buf []byte, start, length int) bool {
	for i := start; i < start+length && i < len(buf); i++ {
		if buf[i] != ' ' {
			return true
		}
	}
	return false
}

func bufPlace(buf []byte, start int, s string) {
	for i := 0; i < len(s) && start+i < len(buf); i++ {
		buf[start+i] = s[i]
	}
}

func (m Model) View() string {
	var b strings.Builder

	w := m.sliderWidth()

	// 1. Leading newline
	b.WriteString("\n")

	// 2. Header: "  ✂ Trim" (bold) ... right-aligned M:SS (dim)
	header := boldStyle.Render("  ✂ Trim")
	durStr := faintStyle.Render(util.FormatDurationShort(m.duration))
	padding := w + 4 - lipgloss.Width(header) - lipgloss.Width(durStr)
	if padding < 1 {
		padding = 1
	}
	b.WriteString(header + strings.Repeat(" ", padding) + durStr + "\n")

	// 3. Blank line
	b.WriteString("\n")

	// 4-5. Time ruler (labels + ticks)
	labels, ticks := m.renderTimeRuler(w)
	b.WriteString("  " + labels + "\n")
	b.WriteString("  " + ticks + "\n")

	// 6. Slider track
	if m.isSelectMode() && m.hasWordSelection() {
		b.WriteString("  " + m.renderSliderWithSegments(w) + "\n")
	} else {
		b.WriteString("  " + m.renderSlider(w) + "\n")
	}

	// 6a. Waveform row (below slider track)
	if wf := m.renderWaveform(w); wf != "" {
		b.WriteString("  " + wf + "\n")
	}

	// 6b. SponsorBlock segments row
	if len(m.sponsorSegments) > 0 {
		b.WriteString("  " + m.renderSponsorSegments(w) + "\n")
		b.WriteString("  " + m.renderSponsorLegend() + "\n")
	}

	// 7-8. Chapter rows (only if chapters exist)
	if len(m.chapters) > 0 {
		connectors := m.renderChapterConnectors(w)
		chapterLabels := m.renderChapterLabels(w)
		if connectors != "" {
			b.WriteString("  " + connectors + "\n")
			b.WriteString("  " + chapterLabels + "\n")
		}
	}

	// 9. Blank line
	b.WriteString("\n")

	// 10. Info row or inline input
	if m.isSelectMode() {
		b.WriteString(m.renderSelectInfo())
	} else if m.mode == modeInput {
		b.WriteString(m.renderInlineInput())
	} else {
		b.WriteString(m.renderInfoRow())
	}
	b.WriteString("\n")

	// 10b. Splits panel (if splits saved)
	if len(m.splits) > 0 && !m.isSelectMode() {
		b.WriteString("\n")
		b.WriteString(m.renderSplitsPanel(w))
	}

	// 11. Blank line
	b.WriteString("\n")

	// 12. Transcript / word select / search panel
	if m.isSelectMode() && len(m.words) > 0 {
		b.WriteString(m.renderWordSelectPanel(w))
	} else if m.transcript != nil && !m.isSelectMode() {
		b.WriteString(m.renderTranscriptPanel(w))
	}

	// 13. Search input (if active)
	if m.isSearchMode() {
		b.WriteString(m.renderSearchInput())
		b.WriteString("\n")
	}

	// 14. Help bar
	b.WriteString(m.renderHelpBar())

	// 15. Trailing newline
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderTimeRuler(width int) (labels string, ticks string) {
	if m.duration <= 0 {
		return strings.Repeat(" ", width), dimStyle.Render(strings.Repeat("┈", width))
	}

	pixelsPerSecond := float64(width) / m.duration

	// Pick adaptive interval
	intervals := []float64{10, 15, 30, 60, 120, 300, 600}
	interval := intervals[len(intervals)-1]
	for _, iv := range intervals {
		if iv*pixelsPerSecond >= 10 {
			interval = iv
			break
		}
	}

	// Build labels line as a byte buffer
	labelBuf := make([]byte, width)
	for i := range labelBuf {
		labelBuf[i] = ' '
	}

	// Place labels at each interval tick
	for t := 0.0; t <= m.duration; t += interval {
		pos := int(t / m.duration * float64(width-1))
		if pos >= width {
			pos = width - 1
		}

		lbl := util.FormatDurationShort(t)
		lblLen := len(lbl)

		// Center label on position
		start := pos - lblLen/2
		if start < 0 {
			start = 0
		}
		if start+lblLen > width {
			start = width - lblLen
		}
		if start < 0 {
			continue
		}

		if bufHasOverlap(labelBuf, start, lblLen) {
			continue
		}
		bufPlace(labelBuf, start, lbl)
	}

	labels = faintStyle.Render(string(labelBuf))
	ticks = dimStyle.Render(strings.Repeat("┈", width))
	return
}

var (
	selectedTrack       = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	unselectedTrack     = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	silenceInStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8c7060"))
	silenceOutStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b4f49"))
	handleActiveStyle   = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	handleInactiveStyle = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
)

func (m Model) renderSlider(width int) string {
	var b strings.Builder

	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))

	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	for i := 0; i < width; i++ {
		seconds := float64(i) / float64(width) * m.duration
		silence := m.isSilenceAt(seconds)
		switch {
		case i == startIdx:
			if m.adjustingStart {
				b.WriteString(handleActiveStyle.Render("┃"))
			} else {
				b.WriteString(handleInactiveStyle.Render("│"))
			}
		case i == endIdx:
			if !m.adjustingStart {
				b.WriteString(handleActiveStyle.Render("┃"))
			} else {
				b.WriteString(handleInactiveStyle.Render("│"))
			}
		case i > startIdx && i < endIdx:
			if silence {
				b.WriteString(silenceInStyle.Render("┄"))
			} else {
				b.WriteString(selectedTrack.Render("━"))
			}
		default:
			if silence {
				b.WriteString(silenceOutStyle.Render("┈"))
			} else {
				b.WriteString(unselectedTrack.Render("─"))
			}
		}
	}

	return b.String()
}

// renderSliderWithSegments renders the slider showing multiple selected segments.
func (m Model) renderSliderWithSegments(width int) string {
	segments := m.selectedSegments()
	if len(segments) == 0 {
		return m.renderSlider(width)
	}

	// Build a per-column state: unselected, selected, or gap
	cols := make([]byte, width) // 'u' = unselected, 's' = selected
	for i := range cols {
		cols[i] = 'u'
	}

	for _, seg := range segments {
		startIdx := int(seg.Start / m.duration * float64(width))
		endIdx := int(seg.End() / m.duration * float64(width))
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx >= width {
			endIdx = width - 1
		}
		for i := startIdx; i <= endIdx && i < width; i++ {
			cols[i] = 's'
		}
	}

	// Compute cursor beam position
	cursorCol := -1
	if m.isSelectMode() && m.cursor >= 0 && m.cursor < len(m.words) {
		cursorCol = int(m.words[m.cursor].Start / m.duration * float64(width))
		if cursorCol >= width {
			cursorCol = width - 1
		}
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		if i == cursorCol {
			b.WriteString(handleActiveStyle.Render("┃"))
			continue
		}
		seconds := float64(i) / float64(width) * m.duration
		silence := m.isSilenceAt(seconds)
		if cols[i] == 's' {
			if silence {
				b.WriteString(silenceInStyle.Render("┄"))
			} else {
				b.WriteString(selectedTrack.Render("━"))
			}
		} else {
			if silence {
				b.WriteString(silenceOutStyle.Render("┈"))
			} else {
				b.WriteString(unselectedTrack.Render("─"))
			}
		}
	}
	return b.String()
}

// renderWaveform renders an amplitude sparkline row aligned with the slider.
func (m Model) renderWaveform(width int) string {
	if len(m.waveform) == 0 {
		return ""
	}

	sparks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var b strings.Builder
	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	for i := 0; i < width; i++ {
		sampleIdx := i * len(m.waveform) / width
		if sampleIdx >= len(m.waveform) {
			sampleIdx = len(m.waveform) - 1
		}
		amp := m.waveform[sampleIdx].Amplitude

		level := int(amp * float64(len(sparks)-1))
		if level < 0 {
			level = 0
		}
		if level >= len(sparks) {
			level = len(sparks) - 1
		}

		ch := string(sparks[level])

		if i >= startIdx && i <= endIdx {
			b.WriteString(accentStyle.Render(ch))
		} else {
			b.WriteString(dimStyle.Render(ch))
		}
	}

	return b.String()
}

// renderSplitsPanel renders a numbered list of saved splits with total duration.
func (m Model) renderSplitsPanel(width int) string {
	if len(m.splits) == 0 {
		return ""
	}

	panelWidth := width
	if panelWidth > 60 {
		panelWidth = 60
	}

	var b strings.Builder
	var totalDur float64
	for _, s := range m.splits {
		totalDur += s.end - s.start
	}

	// Header
	headerLabel := fmt.Sprintf("── splits (%d) ", len(m.splits))
	fillLen := panelWidth - len(headerLabel)
	if fillLen < 1 {
		fillLen = 1
	}
	b.WriteString("  " + dimStyle.Render(headerLabel+strings.Repeat("─", fillLen)) + "\n")

	// List each split
	for i, s := range m.splits {
		dur := s.end - s.start
		line := fmt.Sprintf("    %s  %s - %s  %s",
			faintStyle.Render(fmt.Sprintf("%d", i+1)),
			valueStyle.Render(util.FormatDurationShort(s.start)),
			valueStyle.Render(util.FormatDurationShort(s.end)),
			faintStyle.Render("("+util.FormatDurationShort(dur)+")"))
		b.WriteString(line + "\n")
	}

	// Footer with total
	footerLabel := fmt.Sprintf("──────── total %s ", util.FormatDurationShort(totalDur))
	footerFill := panelWidth - len(footerLabel)
	if footerFill < 1 {
		footerFill = 1
	}
	b.WriteString("  " + dimStyle.Render(footerLabel+strings.Repeat("─", footerFill)) + "\n")

	return b.String()
}

func (m Model) renderChapterConnectors(width int) string {
	buf := make([]byte, width)
	for i := range buf {
		buf[i] = ' '
	}

	hasAny := false
	for _, ch := range m.chapters {
		if ch.StartTime > 0 && ch.StartTime < m.duration {
			pos := int(ch.StartTime / m.duration * float64(width))
			if pos >= 0 && pos < width {
				buf[pos] = '|' // placeholder, we'll render with color
				hasAny = true
			}
		}
	}

	if !hasAny {
		return ""
	}

	// Build styled string
	var b strings.Builder
	for _, c := range buf {
		if c == '|' {
			b.WriteString(warmStyle.Render("╵"))
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func (m Model) renderChapterLabels(width int) string {
	type chapterInfo struct {
		pos   int
		title string
	}

	var chapters []chapterInfo
	for _, ch := range m.chapters {
		if ch.StartTime >= 0 && ch.StartTime < m.duration && ch.Title != "" {
			pos := int(ch.StartTime / m.duration * float64(width))
			if pos >= 0 && pos < width {
				chapters = append(chapters, chapterInfo{pos: pos, title: ch.Title})
			}
		}
	}

	if len(chapters) == 0 {
		return ""
	}

	// Build a character buffer
	buf := make([]byte, width)
	for i := range buf {
		buf[i] = ' '
	}

	// Place each label, avoiding overlaps
	for _, ch := range chapters {
		lbl := ch.title
		// Truncate if too long
		maxLen := width / max(len(chapters), 1)
		if len(lbl) > maxLen && maxLen > 3 {
			lbl = lbl[:maxLen-1] + "…"
		}

		// Center on position
		start := ch.pos - len(lbl)/2
		if start < 0 {
			start = 0
		}
		if start+len(lbl) > width {
			start = width - len(lbl)
		}
		if start < 0 {
			continue
		}

		if bufHasOverlap(buf, start, len(lbl)) {
			// Try placing right of position
			start = ch.pos + 1
			if start+len(lbl) > width {
				continue
			}
			if bufHasOverlap(buf, start, len(lbl)) {
				continue
			}
		}
		bufPlace(buf, start, lbl)
	}

	// Render in yellow
	result := string(buf)
	// Only color the non-space parts
	var b strings.Builder
	inText := false
	var textStart int
	for i := 0; i <= len(result); i++ {
		if i < len(result) && result[i] != ' ' {
			if !inText {
				// Flush spaces
				b.WriteString(result[textStart:i])
				textStart = i
				inText = true
			}
		} else {
			if inText {
				b.WriteString(warmStyle.Render(result[textStart:i]))
				textStart = i
				inText = false
			}
		}
	}
	if textStart < len(result) {
		b.WriteString(result[textStart:])
	}

	return b.String()
}

func (m Model) renderInfoRow() string {
	startStr := util.FormatDurationShort(m.startPos)
	endStr := util.FormatDurationShort(m.endPos)
	length := m.endPos - m.startPos
	lengthStr := util.FormatDurationShort(length)

	var styledStart, styledEnd string
	if m.adjustingStart {
		styledStart = accentBold.Render(startStr)
		styledEnd = valueStyle.Render(endStr)
	} else {
		styledStart = valueStyle.Render(startStr)
		styledEnd = accentBold.Render(endStr)
	}

	info := fmt.Sprintf("  %s %s   %s %s   %s %s",
		faintStyle.Render("start"), styledStart,
		faintStyle.Render("end"), styledEnd,
		faintStyle.Render("length"), faintStyle.Render(lengthStr))
	if n := len(m.splits); n > 0 {
		info += fmt.Sprintf("   %s", accentStyle.Render(fmt.Sprintf("%d splits", n)))
	}
	return info
}

func (m Model) renderSelectInfo() string {
	segs := m.selectedSegments()
	selCount := m.selectedWordCount()
	totalWords := len(m.words)

	var totalDur float64
	for _, seg := range segs {
		totalDur += seg.Duration
	}

	if len(segs) == 0 {
		return faintStyle.Render("  No words selected")
	}

	segText := "segment"
	if len(segs) != 1 {
		segText = "segments"
	}

	return fmt.Sprintf("  %s %s · %s total · %s",
		valueStyle.Render(fmt.Sprintf("%d %s", len(segs), segText)),
		faintStyle.Render(util.FormatDurationShort(totalDur)),
		faintStyle.Render(fmt.Sprintf("%d/%d words", selCount, totalWords)),
		"")
}

func (m Model) renderInlineInput() string {
	cursor := "█"
	inputVal := cursor
	if m.inputBuffer != "" {
		inputVal = m.inputBuffer + cursor
	}

	startStr := util.FormatDurationShort(m.startPos)
	endStr := util.FormatDurationShort(m.endPos)

	if m.adjustingStart {
		return fmt.Sprintf("  %s %s   %s %s   %s %s",
			faintStyle.Render("start"), accentBold.Render(inputVal),
			faintStyle.Render("end"), valueStyle.Render(endStr),
			faintStyle.Render("length"), faintStyle.Render("--:--"))
	}
	return fmt.Sprintf("  %s %s   %s %s   %s %s",
		faintStyle.Render("start"), valueStyle.Render(startStr),
		faintStyle.Render("end"), accentBold.Render(inputVal),
		faintStyle.Render("length"), faintStyle.Render("--:--"))
}

// renderTranscriptPanel renders a scrollable transcript view in slider mode.
func (m Model) renderTranscriptPanel(width int) string {
	if len(m.transcript) == 0 {
		return ""
	}

	var b strings.Builder
	panelWidth := width
	if panelWidth > 60 {
		panelWidth = 60
	}

	// Find active cue based on current handle position
	pos := m.activePos()
	activeCue := m.transcript.NearestCue(pos)

	// Determine scroll offset
	var startCue int
	if m.viewportLocked {
		startCue = activeCue - TranscriptPinOffset
		if startCue < 0 {
			startCue = 0
		}
		maxOffset := len(m.transcript) - TranscriptVisibleCues
		if maxOffset < 0 {
			maxOffset = 0
		}
		if startCue > maxOffset {
			startCue = maxOffset
		}
	} else {
		startCue = m.transcriptOffset
	}

	endCue := startCue + TranscriptVisibleCues
	if endCue > len(m.transcript) {
		endCue = len(m.transcript)
	}

	// Build search match set
	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	// Header
	lockIndicator := ""
	if !m.viewportLocked {
		lockIndicator = " (scrolled)"
	}
	headerWord := accentStyle.Render("transcript")
	headerRest := dimStyle.Render("── ") + headerWord + dimStyle.Render(lockIndicator+" "+strings.Repeat("─", max(panelWidth-15-len(lockIndicator), 1)))
	b.WriteString("  " + headerRest + "\n")

	// Scroll indicator if content above
	if startCue > 0 {
		b.WriteString("  " + faintStyle.Render(fmt.Sprintf("  ▲ %d more", startCue)) + "\n")
	}

	activeBg := lipgloss.NewStyle().Background(tui.ColorSurface1)

	for i := startCue; i < endCue; i++ {
		cue := m.transcript[i]
		timeStr := util.FormatDurationShort(cue.Start)
		isActive := i == activeCue

		// Build marker first so we can measure prefix width
		marker := "   "
		if isActive {
			marker = accentStyle.Render(" › ")
		}

		// Truncate text to fit within terminal width
		prefixWidth := lipgloss.Width(marker) + len(timeStr) + 2 // marker + timestamp + "  "
		maxTextLen := m.width - prefixWidth - 1                   // -1 safety margin
		if cap := panelWidth - prefixWidth; cap < maxTextLen {
			maxTextLen = cap // cap for readability
		}
		text := cue.Text
		if lipgloss.Width(text) > maxTextLen && maxTextLen > 3 {
			text = truncateVisual(text, maxTextLen-1) + "…"
		}

		// Apply SponsorBlock category color if cue overlaps a segment
		sponsorCat := m.sponsorCategoryAt(cue.Start)
		styledText := text
		if isActive {
			styledText = activeBg.Render(accentStyle.Render(text))
		} else if searchSet[i] {
			styledText = warmStyle.Render(text)
		} else if sponsorCat != "" {
			if sc, ok := sponsorCategories[sponsorCat]; ok {
				styledText = sc.Color.Render(text)
			}
		} else if cue.End <= pos {
			styledText = faintStyle.Render(text)
		}
		line := fmt.Sprintf("%s%s  %s", marker, faintStyle.Render(timeStr), styledText)
		b.WriteString(line + "\n")
	}

	// Scroll indicator if content below
	remaining := len(m.transcript) - endCue
	if remaining > 0 {
		b.WriteString("  " + faintStyle.Render(fmt.Sprintf("  ▼ %d more", remaining)) + "\n")
	}

	// Footer
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", panelWidth)) + "\n")
	b.WriteString("\n")

	return b.String()
}

// renderWordSelectPanel renders the word selection view grouped by cue with timestamps.
func (m Model) renderWordSelectPanel(width int) string {
	if len(m.words) == 0 {
		return ""
	}

	panelWidth := width
	if panelWidth > 60 {
		panelWidth = 60
	}

	selCount := m.selectedWordCount()
	totalWords := len(m.words)

	markerCol := 3    // " › " or "   " prefix
	timestampCol := 6 // "M:SS  " prefix
	textWidth := panelWidth - markerCol - timestampCol

	if textWidth < 20 {
		textWidth = 20
	}

	// Build cue groups: [{startIdx, endIdx}]
	type cueGroup struct {
		cueIndex int
		startIdx int
		endIdx   int
	}
	var groups []cueGroup
	if len(m.words) > 0 {
		cur := cueGroup{cueIndex: m.words[0].CueIndex, startIdx: 0}
		for i := 1; i < len(m.words); i++ {
			if m.words[i].CueIndex != cur.cueIndex {
				cur.endIdx = i - 1
				groups = append(groups, cur)
				cur = cueGroup{cueIndex: m.words[i].CueIndex, startIdx: i}
			}
		}
		cur.endIdx = len(m.words) - 1
		groups = append(groups, cur)
	}

	// Find which cue group the cursor is in
	cursorGroup := 0
	for i, g := range groups {
		if m.cursor >= g.startIdx && m.cursor <= g.endIdx {
			cursorGroup = i
			break
		}
	}

	// Window by cues centered on cursor's cue
	startGroup := cursorGroup - WordSelectPinOffset
	if startGroup < 0 {
		startGroup = 0
	}
	endGroup := startGroup + WordSelectVisibleCues
	if endGroup > len(groups) {
		endGroup = len(groups)
		startGroup = endGroup - WordSelectVisibleCues
		if startGroup < 0 {
			startGroup = 0
		}
	}

	selectedStyle := lipgloss.NewStyle().Foreground(tui.ColorPeach)
	cursorSelectedStyle := lipgloss.NewStyle().Reverse(true).Bold(true).Foreground(tui.ColorPeach)

	// Build search match set for fast lookup
	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	var b strings.Builder

	// Header: ── Select Words ──── N/M ──
	countStr := fmt.Sprintf(" %d/%d ", selCount, totalWords)
	headerLabel := "── Select Words "
	headerRight := countStr + "──"
	fillLen := panelWidth - len(headerLabel) - len(headerRight)
	if fillLen < 1 {
		fillLen = 1
	}
	b.WriteString("  " + dimStyle.Render(headerLabel+strings.Repeat("─", fillLen)+headerRight) + "\n")

	// Scroll indicator if content above
	if startGroup > 0 {
		b.WriteString("  " + faintStyle.Render(fmt.Sprintf("  ▲ %d more", startGroup)) + "\n")
	}

	// Cursor time for dimming passed groups
	cursorTime := m.words[m.cursor].Start

	for gi := startGroup; gi < endGroup; gi++ {
		g := groups[gi]
		timestamp := util.FormatDurationShort(m.words[g.startIdx].Start)
		tsPrefix := dimStyle.Render(fmt.Sprintf("%-5s ", timestamp))

		// Marker: " › " for active cue group, "   " otherwise
		marker := "   "
		if gi == cursorGroup {
			marker = accentStyle.Render(" › ")
		}

		// Check if this group has passed (all words ended before cursor)
		groupPassed := m.words[g.endIdx].End <= cursorTime

		// Render words for this cue, wrapping at textWidth
		var line strings.Builder
		lineLen := 0
		firstLine := true

		for i := g.startIdx; i <= g.endIdx; i++ {
			wordText := m.words[i].Text
			// Strip >> speaker-change markers
			wordText = strings.TrimPrefix(wordText, ">>")
			wordText = strings.TrimSpace(wordText)
			if wordText == "" {
				continue
			}
			displayLen := len(wordText)

			if lineLen > 0 && lineLen+1+displayLen > textWidth {
				// Flush current line
				if firstLine {
					b.WriteString("  " + marker + tsPrefix + line.String() + "\n")
					firstLine = false
				} else {
					b.WriteString("  " + strings.Repeat(" ", markerCol+timestampCol) + line.String() + "\n")
				}
				line.Reset()
				lineLen = 0
			}

			if lineLen > 0 {
				line.WriteByte(' ')
				lineLen++
			}

			isCursor := i == m.cursor
			isSelected := m.selected[i]
			isSearchMatch := searchSet[i]

			switch {
			case isCursor && isSelected:
				line.WriteString(cursorSelectedStyle.Render(wordText))
			case isCursor:
				line.WriteString(reverseStyle.Render(wordText))
			case isSelected:
				line.WriteString(selectedStyle.Render(wordText))
			case isSearchMatch:
				line.WriteString(warmStyle.Render(wordText))
			case groupPassed:
				line.WriteString(faintStyle.Render(wordText))
			default:
				// Apply SponsorBlock category color if word is in a segment
				if cat := m.sponsorCategoryAt(m.words[i].Start); cat != "" {
					if sc, ok := sponsorCategories[cat]; ok {
						line.WriteString(sc.Color.Render(wordText))
					} else {
						line.WriteString(wordText)
					}
				} else {
					line.WriteString(wordText)
				}
			}
			lineLen += displayLen
		}

		// Flush remaining
		if lineLen > 0 {
			if firstLine {
				b.WriteString("  " + marker + tsPrefix + line.String() + "\n")
			} else {
				b.WriteString("  " + strings.Repeat(" ", markerCol+timestampCol) + line.String() + "\n")
			}
		}
	}

	// Scroll indicator if content below
	if endGroup < len(groups) {
		b.WriteString("  " + faintStyle.Render(fmt.Sprintf("  ▼ %d more", len(groups)-endGroup)) + "\n")
	}

	// Footer
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", panelWidth)) + "\n")
	b.WriteString("\n")

	return b.String()
}

// renderSearchInput renders the inline search bar.
func (m Model) renderSearchInput() string {
	matchInfo := ""
	if m.searchBuffer != "" {
		matchInfo = fmt.Sprintf("  (%d matches)", len(m.searchResults))
	}
	return fmt.Sprintf("  %s %s%s%s\n",
		accentStyle.Render("/"),
		m.searchBuffer,
		faintStyle.Render("█"),
		faintStyle.Render(matchInfo))
}

func helpEntry(key, desc string) string {
	return helpKeyStyle.Render(key) + " " + faintStyle.Render(desc)
}

func (m Model) renderHelpBar() string {
	sep := dimStyle.Render(" │ ")
	if m.isSearchMode() {
		return "  " + strings.Join([]string{
			faintStyle.Render("type to search"),
			helpEntry("⏎", "snap"),
			helpEntry("esc", "cancel"),
		}, sep)
	}
	if m.isSelectMode() {
		return "  " + strings.Join([]string{
			helpEntry("←→", "word"),
			helpEntry("↑↓", "cue"),
			helpEntry("␣", "toggle"),
			helpEntry("⇧←⇧→", "range"),
			helpEntry("p", "sentence"),
			helpEntry("d", "clear"),
			helpEntry("/", "search"),
			helpEntry("esc", "back"),
			helpEntry("⏎", "done"),
		}, sep)
	}
	if m.transcript != nil {
		return "  " + strings.Join([]string{
			helpEntry("tab", "switch"),
			helpEntry("←→", "1s"),
			helpEntry("↑↓", "1m"),
			helpEntry("[]", "snap"),
			helpEntry("pgup/dn", "scroll"),
			helpEntry("/", "search"),
			helpEntry("s", "split"),
			helpEntry("d", "undo"),
			helpEntry("t", "select"),
			helpEntry("⏎", "done"),
		}, sep)
	}
	return "  " + strings.Join([]string{
		helpEntry("tab", "switch"),
		helpEntry("←→", "1s"),
		helpEntry("↑↓", "1m"),
		helpEntry("shift", "10ms"),
		helpEntry("space", "type"),
		helpEntry("s", "split"),
		helpEntry("d", "undo"),
		helpEntry("⏎", "done"),
	}, sep)
}

type sponsorCategoryStyle struct {
	Color lipgloss.Style
	Label string
}

var sponsorCategories = map[sponsorblock.Category]sponsorCategoryStyle{
	sponsorblock.CategorySponsor:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00d400")), Label: "spon"},
	sponsorblock.CategoryIntro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")), Label: "intr"},
	sponsorblock.CategoryOutro:         {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#0202ed")), Label: "outr"},
	sponsorblock.CategorySelfPromo:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")), Label: "self"},
	sponsorblock.CategoryInteraction:   {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#cc00ff")), Label: "intr"},
	sponsorblock.CategoryMusicOfftopic: {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9900")), Label: "musc"},
	sponsorblock.CategoryPreview:       {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#008fd6")), Label: "prev"},
	sponsorblock.CategoryHighlight:     {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#82")), Label: "high"},
	sponsorblock.CategoryFiller:        {Color: lipgloss.NewStyle().Foreground(lipgloss.Color("#7300FF")), Label: "fill"},
}

// renderSponsorSegments renders a row of colored markers for SponsorBlock segments.
func (m Model) renderSponsorSegments(width int) string {
	if m.duration <= 0 {
		return ""
	}

	buf := make([]byte, width)
	cats := make([]sponsorblock.Category, width)

	for i := range buf {
		buf[i] = ' '
	}

	for _, seg := range m.sponsorSegments {
		if seg.Category == sponsorblock.CategoryHighlight {
			// Point marker
			pos := int(seg.Start / m.duration * float64(width))
			if pos >= 0 && pos < width {
				buf[pos] = '*' // placeholder
				cats[pos] = seg.Category
			}
			continue
		}
		// Range segment
		startIdx := int(seg.Start / m.duration * float64(width))
		endIdx := int(seg.End / m.duration * float64(width))
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx >= width {
			endIdx = width - 1
		}
		for i := startIdx; i <= endIdx; i++ {
			buf[i] = '_' // placeholder for ▁
			cats[i] = seg.Category
		}
	}

	var b strings.Builder
	for i := 0; i < width; i++ {
		if buf[i] == ' ' {
			b.WriteByte(' ')
			continue
		}
		sc, ok := sponsorCategories[cats[i]]
		if !ok {
			sc.Color = dimStyle
		}
		if cats[i] == sponsorblock.CategoryHighlight {
			b.WriteString(sc.Color.Render("★"))
		} else {
			b.WriteString(sc.Color.Render("▁"))
		}
	}
	return b.String()
}

// renderSponsorLegend renders a compact legend for the SponsorBlock categories present.
func (m Model) renderSponsorLegend() string {
	// Collect unique categories present
	seen := make(map[sponsorblock.Category]bool)
	var order []sponsorblock.Category
	for _, seg := range m.sponsorSegments {
		if !seen[seg.Category] {
			seen[seg.Category] = true
			order = append(order, seg.Category)
		}
	}

	var parts []string
	for _, cat := range order {
		sc, ok := sponsorCategories[cat]
		if !ok {
			sc = sponsorCategoryStyle{Color: dimStyle, Label: string(cat)}
		}
		parts = append(parts, sc.Color.Render("■")+" "+dimStyle.Render(sc.Label))
	}
	return strings.Join(parts, "  ")
}

// truncateVisual truncates a string to at most maxWidth visual columns.
func truncateVisual(s string, maxWidth int) string {
	w := 0
	for i, r := range s {
		rw := lipgloss.Width(string(r))
		if w+rw > maxWidth {
			return s[:i]
		}
		w += rw
	}
	return s
}
