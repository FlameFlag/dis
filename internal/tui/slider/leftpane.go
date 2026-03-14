package slider

import (
	"dis/internal/sponsorblock"
	"dis/internal/storyboard"
	"dis/internal/util"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

func (m Model) renderLeftPane(width int) string {
	var lines []string
	w := max(width-2, MinSliderWidth) // inner padding

	// Header: "✂ Trim" ... right-aligned M:SS
	header := boldStyle.Render("✂ Trim")
	durStr := faintStyle.Render(util.FormatDurationShort(m.duration))
	pad := max(width-1-lipgloss.Width(header)-lipgloss.Width(durStr), 1)
	lines = append(lines, " "+header+strings.Repeat(" ", pad)+durStr)

	// Blank line
	lines = append(lines, "")

	// Time ruler
	labels, ticks := m.renderTimeRuler(w)
	lines = append(lines, " "+labels)
	lines = append(lines, " "+ticks)

	// Slider track (with integrated waveform) — two rows: peaks top, density bottom
	if m.isSelectMode() && m.hasWordSelection() {
		top, bot := m.renderSliderWithSegments(w)
		lines = append(lines, " "+top)
		lines = append(lines, " "+bot)
	} else {
		top, bot := m.renderIntegratedSlider(w)
		lines = append(lines, " "+top)
		lines = append(lines, " "+bot)
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

	// Loading status row
	if loading := m.renderLoadingStatus(); loading != "" {
		lines = append(lines, " "+loading)
	}

	// Splits panel
	if len(m.splits) > 0 && !m.isSelectMode() {
		lines = append(lines, "")
		lines = append(lines, m.renderSplitsPanelLines(w)...)
	}

	// Thumbnail preview
	if thumb := m.renderThumbnail(w); thumb != "" {
		lines = append(lines, "")
		for _, tl := range strings.Split(thumb, "\n") {
			lines = append(lines, " "+tl)
		}
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

// renderIntegratedSlider merges the waveform into the slider track characters.
// Returns two rows: top = amplitude peaks, bottom = activity density + silence brackets.
func (m Model) renderIntegratedSlider(width int) (topRow, bottomRow string) {
	startIdx := int(m.animStartPos / m.duration * float64(width))
	endIdx := int(m.animEndPos / m.duration * float64(width))
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx >= width {
		endIdx = width - 1
	}

	hasWaveform := len(m.waveform) > 0

	// Pre-pass: find silence run boundaries for bracket markers.
	silenceStart := make(map[int]bool, width)
	silenceEnd := make(map[int]bool, width)
	prevSilence := false
	for i := range width {
		s := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		if s && !prevSilence {
			silenceStart[i] = true
		}
		if !s && prevSilence {
			silenceEnd[i-1] = true
		}
		prevSilence = s
	}
	if prevSilence {
		silenceEnd[width-1] = true
	}

	var top, bot strings.Builder

	for i := range width {
		silence := m.isSilenceAt(float64(i) / float64(width) * m.duration)

		// Handle positions — same vertical bar in both rows.
		if i == startIdx {
			if m.adjustingStart {
				top.WriteString(handleActiveStyle.Render("┃"))
				bot.WriteString(handleActiveStyle.Render("┃"))
			} else {
				top.WriteString(handleInactiveStyle.Render("│"))
				bot.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}
		if i == endIdx {
			if !m.adjustingStart {
				top.WriteString(handleActiveStyle.Render("┃"))
				bot.WriteString(handleActiveStyle.Render("┃"))
			} else {
				top.WriteString(handleInactiveStyle.Render("│"))
				bot.WriteString(handleInactiveStyle.Render("│"))
			}
			continue
		}

		inRange := i > startIdx && i < endIdx

		if silence {
			// Top row: blank space in silence style.
			if inRange {
				top.WriteString(silenceInStyle.Render(" "))
			} else {
				top.WriteString(silenceOutStyle.Render(" "))
			}
			// Bottom row: bracket markers.
			var bracket string
			switch {
			case silenceStart[i] && silenceEnd[i]:
				bracket = "⌊"
			case silenceStart[i]:
				bracket = "⌊"
			case silenceEnd[i]:
				bracket = "⌋"
			default:
				bracket = "╌"
			}
			if inRange {
				bot.WriteString(silenceInStyle.Render(bracket))
			} else {
				bot.WriteString(silenceOutStyle.Render(bracket))
			}
			continue
		}

		// Audio column.
		if hasWaveform {
			sampleIdx := i * len(m.waveform) / width
			if sampleIdx >= len(m.waveform) {
				sampleIdx = len(m.waveform) - 1
			}
			amp := m.waveform[sampleIdx].Amplitude

			// Top row: amplitude peak bar.
			level := max(int(amp*float64(len(sparks)-1)), 0)
			if level >= len(sparks) {
				level = len(sparks) - 1
			}
			topCh := string(sparks[level])
			if inRange {
				top.WriteString(selectedTrack.Render(topCh))
			} else {
				top.WriteString(unselectedTrack.Render(topCh))
			}

			// Bottom row: activity density.
			var density string
			switch {
			case amp < 0.10:
				density = " "
			case amp < 0.30:
				density = "░"
			case amp < 0.60:
				density = "▒"
			case amp < 0.80:
				density = "▓"
			default:
				density = "█"
			}
			if inRange {
				bot.WriteString(selectedTrack.Render(density))
			} else {
				bot.WriteString(unselectedTrack.Render(density))
			}
		} else {
			// No waveform: simple track chars in both rows.
			if inRange {
				top.WriteString(selectedTrack.Render("━"))
				bot.WriteString(selectedTrack.Render("━"))
			} else {
				top.WriteString(unselectedTrack.Render("─"))
				bot.WriteString(unselectedTrack.Render("─"))
			}
		}
	}

	return top.String(), bot.String()
}

// renderSliderWithSegments renders the slider showing multiple selected segments.
// Returns two rows: top = track chars, bottom = silence brackets.
func (m Model) renderSliderWithSegments(width int) (topRow, bottomRow string) {
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

	// Pre-pass: silence run boundaries.
	silenceStart := make(map[int]bool, width)
	silenceEnd := make(map[int]bool, width)
	prevSilence := false
	for i := range width {
		s := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		if s && !prevSilence {
			silenceStart[i] = true
		}
		if !s && prevSilence {
			silenceEnd[i-1] = true
		}
		prevSilence = s
	}
	if prevSilence {
		silenceEnd[width-1] = true
	}

	var top, bot strings.Builder
	for i := range width {
		if i == cursorCol {
			top.WriteString(handleActiveStyle.Render("┃"))
			bot.WriteString(handleActiveStyle.Render("┃"))
			continue
		}
		silence := m.isSilenceAt(float64(i) / float64(width) * m.duration)
		selected := cols[i] == 's'

		if silence {
			// Top: blank.
			if selected {
				top.WriteString(silenceInStyle.Render(" "))
			} else {
				top.WriteString(silenceOutStyle.Render(" "))
			}
			// Bottom: bracket markers.
			var bracket string
			switch {
			case silenceStart[i]:
				bracket = "⌊"
			case silenceEnd[i]:
				bracket = "⌋"
			default:
				bracket = "╌"
			}
			if selected {
				bot.WriteString(silenceInStyle.Render(bracket))
			} else {
				bot.WriteString(silenceOutStyle.Render(bracket))
			}
		} else {
			if selected {
				top.WriteString(selectedTrack.Render("━"))
				bot.WriteString(selectedTrack.Render("━"))
			} else {
				top.WriteString(unselectedTrack.Render("─"))
				bot.WriteString(unselectedTrack.Render("─"))
			}
		}
	}
	return top.String(), bot.String()
}

func (m Model) renderLoadingStatus() string {
	if !m.isLoading() {
		return ""
	}
	spinner := string(loadingSpinner[m.loadingFrame%len(loadingSpinner)])
	var items []string
	if m.waveformCh != nil {
		items = append(items, "waveform")
	}
	if m.silenceCh != nil {
		items = append(items, "silence")
	}
	if m.storyboardCh != nil {
		items = append(items, "storyboard")
	}
	return faintStyle.Render(spinner + " " + strings.Join(items, " · "))
}

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
		start := max(pos-lblLen/2, 0)
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
		badge := accentBold.Render("GIF")
		if m.speedMultiplier > 1.0 {
			badge += " " + accentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
		}
		return badge
	}
	badge := faintStyle.Render("MP4")
	if m.speedMultiplier > 1.0 {
		badge += " " + accentBold.Render(fmt.Sprintf("%.1fx", m.speedMultiplier))
	}
	return badge
}

func (m Model) renderSplitsPanelLines(width int) []string {
	if len(m.splits) == 0 {
		return nil
	}

	panelWidth := min(width, 56)

	var lines []string
	var totalDur float64
	for _, s := range m.splits {
		totalDur += s.end - s.start
	}

	headerLabel := fmt.Sprintf("── splits (%d) ", len(m.splits))
	fillLen := max(panelWidth-len(headerLabel), 1)
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
	footerFill := max(panelWidth-len(footerLabel), 1)
	lines = append(lines, " "+dimStyle.Render(footerLabel+strings.Repeat("─", footerFill)))

	return lines
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
		start := max(ch.pos-len(lbl)/2, 0)
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
	for i := range width {
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

// thumbnailCache stores the last rendered thumbnail to avoid re-rendering on every frame.
type thumbnailCache struct {
	cellKey string
	width   int
	output  string
}

var thumbCache thumbnailCache

func (m Model) renderThumbnail(width int) string {
	if m.storyboard == nil || m.height < 25 {
		if storyboard.IsKittySupported() {
			return storyboard.DeleteKittyImage()
		}
		return ""
	}

	thumbW := min(width-2, 56)
	thumbH := 14 // character rows = 28 pixels tall in half-block mode

	// Quantize position to cell boundary to avoid re-rendering every frame
	pos := m.activePos()
	info := &m.storyboard.Info
	cellsPerFrag := info.Rows * info.Columns
	cellDuration := 0.0
	if len(info.Fragments) > 0 && cellsPerFrag > 0 {
		cellDuration = info.Fragments[0].Duration / float64(cellsPerFrag)
	}
	if cellDuration <= 0 {
		return ""
	}
	quantized := int(pos / cellDuration)
	cacheKey := fmt.Sprintf("%d", quantized)

	if thumbCache.cellKey == cacheKey && thumbCache.width == thumbW {
		return thumbCache.output
	}

	cell := storyboard.CellAt(m.storyboard, pos)
	if cell == nil {
		return ""
	}

	var rendered string
	switch {
	case storyboard.IsKittySupported():
		rendered = storyboard.RenderKitty(cell, thumbW, thumbH)
	case storyboard.IsSixelSupported():
		rendered = storyboard.RenderSixel(cell, thumbW, thumbH)
	default:
		rendered = storyboard.RenderHalfBlock(cell, thumbW, thumbH)
	}
	thumbCache = thumbnailCache{cellKey: cacheKey, width: thumbW, output: rendered}
	return rendered
}
