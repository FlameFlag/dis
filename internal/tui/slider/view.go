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
	borderStyle  = lipgloss.NewStyle().Foreground(tui.ColorSurface2)
	warnStyle    = lipgloss.NewStyle().Foreground(tui.ColorRed)
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

// ── View (main entry point) ──────────────────────────────────────────

func (m Model) View() string {
	if m.width < 20 {
		return ""
	}

	helpBar := m.renderHelpBar()

	if m.isTwoPane() {
		leftW := m.leftPaneWidth()
		rightW := m.rightPaneWidth()
		left := m.renderLeftPane(leftW)
		right := m.renderRightPane(rightW)
		body := m.renderBorderedLayout(left, leftW, right, rightW)
		return body + "\n" + helpBar + "\n"
	}

	// Single-column fallback
	left := m.renderLeftPane(m.width - 2)
	body := m.renderSingleColumnLayout(left, m.width-2)
	return body + "\n" + helpBar + "\n"
}

// ── Left Pane (Timeline) ─────────────────────────────────────────────

func (m Model) renderLeftPane(width int) string {
	var lines []string
	w := width - 2 // inner padding
	if w < MinSliderWidth {
		w = MinSliderWidth
	}

	// Header: "✂ Trim" ... right-aligned M:SS
	header := boldStyle.Render("✂ Trim")
	durStr := faintStyle.Render(util.FormatDurationShort(m.duration))
	pad := width - lipgloss.Width(header) - lipgloss.Width(durStr)
	if pad < 1 {
		pad = 1
	}
	lines = append(lines, " "+header+strings.Repeat(" ", pad)+durStr)

	// Blank line
	lines = append(lines, "")

	// Time ruler
	labels, ticks := m.renderTimeRuler(w)
	lines = append(lines, " "+labels)
	lines = append(lines, " "+ticks)

	// Slider track (with integrated waveform)
	if m.isSelectMode() && m.hasWordSelection() {
		lines = append(lines, " "+m.renderSliderWithSegments(w))
	} else {
		lines = append(lines, " "+m.renderIntegratedSlider(w))
	}

	// SponsorBlock segments row (no legend)
	if len(m.sponsorSegments) > 0 {
		lines = append(lines, " "+m.renderSponsorSegments(w))
	}

	// Chapter labels (no connector row)
	if len(m.chapters) > 0 {
		if lbl := m.renderChapterLabels(w); lbl != "" {
			lines = append(lines, " "+lbl)
		}
	}

	// Blank line
	lines = append(lines, "")

	// Info row / inline input / select info
	if m.isSelectMode() {
		lines = append(lines, m.renderSelectInfo())
	} else if m.mode == modeInput {
		lines = append(lines, m.renderInlineInput())
	} else {
		lines = append(lines, m.renderInfoRow())
	}

	// Splits panel
	if len(m.splits) > 0 && !m.isSelectMode() {
		lines = append(lines, "")
		lines = append(lines, m.renderSplitsPanelLines(w)...)
	}

	// Warning (if any)
	if m.warning != "" {
		lines = append(lines, "")
		lines = append(lines, " "+warnStyle.Render(m.warning))
	}

	// Format badge at bottom-right
	formatBadge := m.renderFormatBadge()
	if formatBadge != "" {
		// Pad the last line or add a new one with right-aligned badge
		lines = append(lines, strings.Repeat(" ", max(width-lipgloss.Width(formatBadge)-1, 0))+formatBadge)
	}

	return strings.Join(lines, "\n")
}

// ── Right Pane ───────────────────────────────────────────────────────

func (m Model) renderRightPane(width int) string {
	if m.isSelectMode() && len(m.words) > 0 {
		return m.renderWordSelectPanel(width)
	}
	if m.transcript != nil {
		return m.renderTranscriptPanel(width)
	}
	if len(m.waveform) > 0 {
		return m.renderVerticalWaveform(width)
	}
	return ""
}

// ── Bordered Layout ──────────────────────────────────────────────────

func (m Model) renderBorderedLayout(left string, leftW int, right string, rightW int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Equalize heights
	maxH := max(len(leftLines), len(rightLines))
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	// Right pane border color: peach in select mode, default otherwise
	divColor := borderStyle
	if m.isSelectMode() {
		divColor = accentStyle
	}

	// Build right pane title
	rightTitle := m.rightPaneTitle()

	var b strings.Builder

	// Top border: ╭─ Timeline ─────────────┬─ Transcript ──────────╮
	leftTitle := " Timeline "
	topLeft := "─" + borderStyle.Render(leftTitle) + borderStyle.Render(strings.Repeat("─", max(leftW-lipgloss.Width(leftTitle)-1, 0)))
	topRight := "─" + divColor.Render(rightTitle) + divColor.Render(strings.Repeat("─", max(rightW-lipgloss.Width(rightTitle)-1, 0)))
	b.WriteString(borderStyle.Render("╭") + borderStyle.Render(topLeft) + divColor.Render("┬") + divColor.Render(topRight) + divColor.Render("╮") + "\n")

	// Body rows
	for i := 0; i < maxH; i++ {
		ll := padRight(leftLines[i], leftW)
		rl := padRight(rightLines[i], rightW)
		b.WriteString(borderStyle.Render("│") + ll + divColor.Render("│") + rl + divColor.Render("│") + "\n")
	}

	// Bottom border: ╰─────────────────────────┴──────────────────────╯
	// Search input sits above the bottom border if active
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := m.width - 2 - lipgloss.Width(searchLine)
		if searchPad < 0 {
			searchPad = 0
		}
		b.WriteString(borderStyle.Render("│") + searchLine + strings.Repeat(" ", searchPad) + borderStyle.Render("│") + "\n")
	}

	b.WriteString(borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", leftW)) + borderStyle.Render("┴") + divColor.Render(strings.Repeat("─", rightW)) + divColor.Render("╯"))

	return b.String()
}

func (m Model) renderSingleColumnLayout(content string, innerW int) string {
	lines := strings.Split(content, "\n")

	var b strings.Builder

	// Top border
	b.WriteString(borderStyle.Render("╭") + borderStyle.Render("─ Timeline "+strings.Repeat("─", max(innerW-11, 0))) + borderStyle.Render("╮") + "\n")

	for _, line := range lines {
		b.WriteString(borderStyle.Render("│") + padRight(line, innerW) + borderStyle.Render("│") + "\n")
	}

	// Search input
	if m.isSearchMode() {
		searchLine := m.renderSearchInput()
		searchPad := innerW - lipgloss.Width(searchLine)
		if searchPad < 0 {
			searchPad = 0
		}
		b.WriteString(borderStyle.Render("│") + searchLine + strings.Repeat(" ", searchPad) + borderStyle.Render("│") + "\n")
	}

	b.WriteString(borderStyle.Render("╰") + borderStyle.Render(strings.Repeat("─", innerW)) + borderStyle.Render("╯"))

	return b.String()
}

func (m Model) rightPaneTitle() string {
	if m.isSelectMode() && len(m.words) > 0 {
		selCount := m.selectedWordCount()
		return fmt.Sprintf(" Select Words %d/%d ", selCount, len(m.words))
	}
	if m.transcript != nil {
		return " Transcript "
	}
	return " Audio "
}

// ── Integrated Slider (waveform merged into track) ───────────────────

var (
	selectedTrack       = lipgloss.NewStyle().Foreground(tui.ColorPeach)
	unselectedTrack     = lipgloss.NewStyle().Foreground(tui.ColorSurface1)
	silenceInStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8c7060"))
	silenceOutStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#5b4f49"))
	handleActiveStyle   = lipgloss.NewStyle().Foreground(tui.ColorText).Bold(true)
	handleInactiveStyle = lipgloss.NewStyle().Foreground(tui.ColorOverlay0)
)

var sparks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// renderIntegratedSlider merges the waveform into the slider track characters.
func (m Model) renderIntegratedSlider(width int) string {
	var b strings.Builder

	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	hasWaveform := len(m.waveform) > 0

	for i := 0; i < width; i++ {
		seconds := float64(i) / float64(width) * m.duration
		silence := m.isSilenceAt(seconds)

		// Handle positions
		switch {
		case i == startIdx:
			if m.adjustingStart {
				b.WriteString(handleActiveStyle.Render("┃"))
			} else {
				b.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		case i == endIdx:
			if !m.adjustingStart {
				b.WriteString(handleActiveStyle.Render("┃"))
			} else {
				b.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}

		inRange := i > startIdx && i < endIdx

		// Get waveform character if available
		if hasWaveform && !silence {
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
			if inRange {
				b.WriteString(selectedTrack.Render(ch))
			} else {
				b.WriteString(unselectedTrack.Render(ch))
			}
			continue
		}

		// Fallback to simple track chars
		if inRange {
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

// renderSlider is the original slider without waveform (used as fallback).
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
		return m.renderIntegratedSlider(width)
	}

	cols := make([]byte, width)
	for i := range cols {
		cols[i] = 'u'
	}

	for _, seg := range segments {
		si := int(seg.Start / m.duration * float64(width))
		ei := int(seg.End() / m.duration * float64(width))
		if si < 0 {
			si = 0
		}
		if ei >= width {
			ei = width - 1
		}
		for i := si; i <= ei && i < width; i++ {
			cols[i] = 's'
		}
	}

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

// ── Vertical Waveform (right pane fallback when no transcript) ───────

func (m Model) renderVerticalWaveform(width int) string {
	if len(m.waveform) == 0 {
		return ""
	}

	barChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}
	maxBarWidth := width - 8 // leave space for position indicator
	if maxBarWidth < 4 {
		maxBarWidth = 4
	}

	// Determine visible height (we'll use the available space)
	visibleRows := 16
	if visibleRows > len(m.waveform) {
		visibleRows = len(m.waveform)
	}

	// Find which row the active handle corresponds to
	activePos := m.activePos()
	activeRow := int(activePos / m.duration * float64(visibleRows))
	if activeRow >= visibleRows {
		activeRow = visibleRows - 1
	}

	var lines []string

	for row := 0; row < visibleRows; row++ {
		// Map row to waveform sample
		sampleIdx := row * len(m.waveform) / visibleRows
		if sampleIdx >= len(m.waveform) {
			sampleIdx = len(m.waveform) - 1
		}
		amp := m.waveform[sampleIdx].Amplitude

		// Calculate bar width
		fullBlocks := int(amp * float64(maxBarWidth))
		fractional := amp*float64(maxBarWidth) - float64(fullBlocks)

		var bar strings.Builder
		for i := 0; i < fullBlocks && i < maxBarWidth; i++ {
			bar.WriteString("█")
		}
		if fullBlocks < maxBarWidth && fractional > 0.1 {
			level := int(fractional * float64(len(barChars)-1))
			if level >= len(barChars) {
				level = len(barChars) - 1
			}
			bar.WriteString(barChars[level])
		}

		barStr := bar.String()
		barWidth := lipgloss.Width(barStr)

		// Determine if this row is in the selected range
		rowTime := float64(row) / float64(visibleRows) * m.duration
		inRange := rowTime >= m.animStartPos && rowTime <= m.animEndPos

		var styledBar string
		if inRange {
			styledBar = accentStyle.Render(barStr)
		} else {
			styledBar = dimStyle.Render(barStr)
		}

		// Position indicator
		indicator := ""
		if row == activeRow {
			timeStr := util.FormatDurationShort(activePos)
			indicator = " " + accentBold.Render("◀ "+timeStr)
		}

		padding := max(maxBarWidth-barWidth, 0)
		line := "  " + styledBar + strings.Repeat(" ", padding) + indicator
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// ── Time Ruler ───────────────────────────────────────────────────────

func (m Model) renderTimeRuler(width int) (labels string, ticks string) {
	if m.duration <= 0 {
		return strings.Repeat(" ", width), dimStyle.Render(strings.Repeat("┈", width))
	}

	pixelsPerSecond := float64(width) / m.duration

	intervals := []float64{10, 15, 30, 60, 120, 300, 600}
	interval := intervals[len(intervals)-1]
	for _, iv := range intervals {
		if iv*pixelsPerSecond >= 10 {
			interval = iv
			break
		}
	}

	labelBuf := make([]byte, width)
	for i := range labelBuf {
		labelBuf[i] = ' '
	}

	for t := 0.0; t <= m.duration; t += interval {
		pos := int(t / m.duration * float64(width-1))
		if pos >= width {
			pos = width - 1
		}
		lbl := util.FormatDurationShort(t)
		lblLen := len(lbl)
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

// ── Info Rows ────────────────────────────────────────────────────────

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

	info := fmt.Sprintf(" %s %s  %s %s  %s %s",
		faintStyle.Render("start"), styledStart,
		faintStyle.Render("end"), styledEnd,
		faintStyle.Render("length"), faintStyle.Render(lengthStr))
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
		return faintStyle.Render(" No words selected")
	}

	segText := "segment"
	if len(segs) != 1 {
		segText = "segments"
	}

	return fmt.Sprintf(" %s %s · %s · %s",
		valueStyle.Render(fmt.Sprintf("%d %s", len(segs), segText)),
		faintStyle.Render(util.FormatDurationShort(totalDur)),
		faintStyle.Render("total"),
		faintStyle.Render(fmt.Sprintf("%d/%d", selCount, totalWords)))
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
		return fmt.Sprintf(" %s %s  %s %s  %s %s",
			faintStyle.Render("start"), accentBold.Render(inputVal),
			faintStyle.Render("end"), valueStyle.Render(endStr),
			faintStyle.Render("length"), faintStyle.Render("--:--"))
	}
	return fmt.Sprintf(" %s %s  %s %s  %s %s",
		faintStyle.Render("start"), valueStyle.Render(startStr),
		faintStyle.Render("end"), accentBold.Render(inputVal),
		faintStyle.Render("length"), faintStyle.Render("--:--"))
}

func (m Model) renderFormatBadge() string {
	if m.gifMode {
		return accentBold.Render("GIF")
	}
	return faintStyle.Render("MP4")
}

// ── Splits Panel ─────────────────────────────────────────────────────

func (m Model) renderSplitsPanelLines(width int) []string {
	if len(m.splits) == 0 {
		return nil
	}

	panelWidth := width
	if panelWidth > 56 {
		panelWidth = 56
	}

	var lines []string
	var totalDur float64
	for _, s := range m.splits {
		totalDur += s.end - s.start
	}

	headerLabel := fmt.Sprintf("── splits (%d) ", len(m.splits))
	fillLen := panelWidth - len(headerLabel)
	if fillLen < 1 {
		fillLen = 1
	}
	lines = append(lines, " "+dimStyle.Render(headerLabel+strings.Repeat("─", fillLen)))

	for i, s := range m.splits {
		dur := s.end - s.start
		line := fmt.Sprintf("   %s  %s - %s  %s",
			faintStyle.Render(fmt.Sprintf("%d", i+1)),
			valueStyle.Render(util.FormatDurationShort(s.start)),
			valueStyle.Render(util.FormatDurationShort(s.end)),
			faintStyle.Render("("+util.FormatDurationShort(dur)+")"))
		lines = append(lines, line)
	}

	footerLabel := fmt.Sprintf("──────── total %s ", util.FormatDurationShort(totalDur))
	footerFill := panelWidth - len(footerLabel)
	if footerFill < 1 {
		footerFill = 1
	}
	lines = append(lines, " "+dimStyle.Render(footerLabel+strings.Repeat("─", footerFill)))

	return lines
}

// renderSplitsPanel kept for compatibility, returns joined string.
func (m Model) renderSplitsPanel(width int) string {
	lines := m.renderSplitsPanelLines(width)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// ── Chapter Labels ───────────────────────────────────────────────────

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

	buf := make([]byte, width)
	for i := range buf {
		buf[i] = ' '
	}

	for _, ch := range chapters {
		lbl := ch.title
		maxLen := width / max(len(chapters), 1)
		if len(lbl) > maxLen && maxLen > 3 {
			lbl = lbl[:maxLen-1] + "…"
		}
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

	result := string(buf)
	var b strings.Builder
	inText := false
	var textStart int
	for i := 0; i <= len(result); i++ {
		if i < len(result) && result[i] != ' ' {
			if !inText {
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

// ── SponsorBlock ─────────────────────────────────────────────────────

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
			pos := int(seg.Start / m.duration * float64(width))
			if pos >= 0 && pos < width {
				buf[pos] = '*'
				cats[pos] = seg.Category
			}
			continue
		}
		si := int(seg.Start / m.duration * float64(width))
		ei := int(seg.End / m.duration * float64(width))
		if si < 0 {
			si = 0
		}
		if ei >= width {
			ei = width - 1
		}
		for i := si; i <= ei; i++ {
			buf[i] = '_'
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

// ── Transcript Panel (right pane) ────────────────────────────────────

func (m Model) renderTranscriptPanel(width int) string {
	if len(m.transcript) == 0 {
		return ""
	}

	var lines []string

	pos := m.activePos()
	activeCue := m.transcript.NearestCue(pos)

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

	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	// Scroll indicator above
	if startCue > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▲ %d more", startCue)))
	}

	activeBg := lipgloss.NewStyle().Background(tui.ColorSurface1)
	textWidth := width - 10 // timestamp + padding
	if textWidth < 10 {
		textWidth = 10
	}

	for i := startCue; i < endCue; i++ {
		cue := m.transcript[i]
		timeStr := util.FormatDurationShort(cue.Start)
		isActive := i == activeCue

		text := cue.Text
		if lipgloss.Width(text) > textWidth && textWidth > 3 {
			text = truncateVisual(text, textWidth-1) + "…"
		}

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

		// Active indicator on right side
		indicator := "  "
		if isActive {
			indicator = accentStyle.Render(" ◀")
		}

		line := fmt.Sprintf("  %s  %s%s", faintStyle.Render(timeStr), styledText, indicator)
		lines = append(lines, line)
	}

	// Scroll indicator below
	remaining := len(m.transcript) - endCue
	if remaining > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▼ %d more", remaining)))
	}

	return strings.Join(lines, "\n")
}

// ── Word Select Panel (right pane) ───────────────────────────────────

func (m Model) renderWordSelectPanel(width int) string {
	if len(m.words) == 0 {
		return ""
	}

	markerCol := 3
	timestampCol := 6
	textWidth := width - markerCol - timestampCol
	if textWidth < 20 {
		textWidth = 20
	}

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

	cursorGroup := 0
	for i, g := range groups {
		if m.cursor >= g.startIdx && m.cursor <= g.endIdx {
			cursorGroup = i
			break
		}
	}

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

	searchSet := make(map[int]bool, len(m.searchResults))
	for _, idx := range m.searchResults {
		searchSet[idx] = true
	}

	var lines []string

	// Scroll indicator above
	if startGroup > 0 {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▲ %d more", startGroup)))
	}

	cursorTime := m.words[m.cursor].Start

	for gi := startGroup; gi < endGroup; gi++ {
		g := groups[gi]
		timestamp := util.FormatDurationShort(m.words[g.startIdx].Start)
		tsPrefix := dimStyle.Render(fmt.Sprintf("%-5s ", timestamp))

		marker := "   "
		if gi == cursorGroup {
			marker = accentStyle.Render(" › ")
		}

		groupPassed := m.words[g.endIdx].End <= cursorTime

		var line strings.Builder
		lineLen := 0
		firstLine := true

		for i := g.startIdx; i <= g.endIdx; i++ {
			wordText := m.words[i].Text
			wordText = strings.TrimPrefix(wordText, ">>")
			wordText = strings.TrimSpace(wordText)
			if wordText == "" {
				continue
			}
			displayLen := len(wordText)

			if lineLen > 0 && lineLen+1+displayLen > textWidth {
				if firstLine {
					lines = append(lines, marker+tsPrefix+line.String())
					firstLine = false
				} else {
					lines = append(lines, strings.Repeat(" ", markerCol+timestampCol)+line.String())
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

		if lineLen > 0 {
			if firstLine {
				lines = append(lines, marker+tsPrefix+line.String())
			} else {
				lines = append(lines, strings.Repeat(" ", markerCol+timestampCol)+line.String())
			}
		}
	}

	// Scroll indicator below
	if endGroup < len(groups) {
		lines = append(lines, faintStyle.Render(fmt.Sprintf("  ▼ %d more", len(groups)-endGroup)))
	}

	return strings.Join(lines, "\n")
}

// ── Search Input ─────────────────────────────────────────────────────

func (m Model) renderSearchInput() string {
	matchInfo := ""
	if m.searchBuffer != "" {
		matchInfo = fmt.Sprintf("  (%d matches)", len(m.searchResults))
	}
	return fmt.Sprintf(" %s %s%s%s",
		accentStyle.Render("/"),
		m.searchBuffer,
		faintStyle.Render("█"),
		faintStyle.Render(matchInfo))
}

// ── Help Bar (two-line) ──────────────────────────────────────────────

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
		line1 := "  " + strings.Join([]string{
			helpEntry("←→", "word"),
			helpEntry("↑↓", "cue"),
			helpEntry("␣", "toggle"),
			helpEntry("⇧←⇧→", "range"),
			helpEntry("p", "sentence"),
		}, sep)
		line2 := "  " + strings.Join([]string{
			helpEntry("d", "clear"),
			helpEntry("/", "search"),
			helpEntry("esc", "back"),
			helpEntry("⏎", "done"),
		}, sep)
		return line1 + "\n" + line2
	}

	if m.transcript != nil {
		line1 := "  " + strings.Join([]string{
			helpEntry("tab", "switch"),
			helpEntry("←→", "1s"),
			helpEntry("↑↓", "1m"),
			helpEntry("[]", "snap"),
			helpEntry("/", "search"),
		}, sep)
		line2 := "  " + strings.Join([]string{
			helpEntry("s", "split"),
			helpEntry("d", "undo"),
			helpEntry("g", "gif"),
			helpEntry("t", "words"),
			helpEntry("⏎", "done"),
		}, sep)
		return line1 + "\n" + line2
	}

	line1 := "  " + strings.Join([]string{
		helpEntry("tab", "switch"),
		helpEntry("←→", "1s"),
		helpEntry("↑↓", "1m"),
		helpEntry("shift", "10ms"),
		helpEntry("space", "type"),
	}, sep)
	line2 := "  " + strings.Join([]string{
		helpEntry("s", "split"),
		helpEntry("d", "undo"),
		helpEntry("g", "gif"),
		helpEntry("⏎", "done"),
	}, sep)
	return line1 + "\n" + line2
}

// ── Utilities ────────────────────────────────────────────────────────

// padRight pads a string with spaces to reach the target visual width.
func padRight(s string, targetWidth int) string {
	w := lipgloss.Width(s)
	if w >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-w)
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
