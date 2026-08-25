package slider

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/4evy/dis/internal/sponsorblock"
	"github.com/4evy/dis/internal/timecode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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

	occupied := make([]bool, width)
	var labels []*lipgloss.Layer

	for _, ch := range chapters {
		lbl := ch.title
		maxLen := width / max(len(chapters), 1)
		if ansi.StringWidth(lbl) > maxLen && maxLen > 0 {
			lbl = ansi.Truncate(lbl, maxLen, "…")
		}
		labelWidth := ansi.StringWidth(lbl)
		start := max(ch.pos-labelWidth/labelCenterDivisor, 0)
		if start+labelWidth > width {
			start = width - labelWidth
		}
		if start < 0 {
			continue
		}
		if slices.Contains(occupied[start:start+labelWidth], true) {
			start = ch.pos + 1
			if start+labelWidth > width {
				continue
			}
			if slices.Contains(occupied[start:start+labelWidth], true) {
				continue
			}
		}
		for i := start; i < start+labelWidth; i++ {
			occupied[i] = true
		}
		labels = append(labels, lipgloss.NewLayer(Warm.Render(lbl)).X(start))
	}

	canvas := lipgloss.NewCanvas(width, 1)
	canvas.Compose(lipgloss.NewCompositor(labels...))
	return canvas.Render()
}

func (m Model) renderSponsorSegments(width int) string {
	if m.duration <= 0 {
		return ""
	}

	buf := bytes.Repeat([]byte{' '}, width)
	cats := make([]sponsorblock.Category, width)

	for _, seg := range m.sponsorSegments {
		if seg.Category == sponsorblock.CategoryHighlight {
			pos := int(seg.Start / m.duration * float64(width))
			if pos >= 0 && pos < width {
				buf[pos] = '*'
				cats[pos] = seg.Category
			}
			continue
		}
		si := max(int(seg.Start/m.duration*float64(width)), 0)
		ei := min(int(seg.End/m.duration*float64(width)), width-1)
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
		sc, ok := SponsorCategories[cats[i]]
		if !ok {
			sc.Color = Dim
		}
		glyph := sc.Glyph
		if glyph == "" {
			glyph = "•"
		}
		b.WriteString(sc.Color.Render(glyph))
	}
	return b.String()
}

func (m Model) renderSponsorLegend(width int) string {
	seen := make(map[sponsorblock.Category]bool)
	items := make([]string, 0, len(m.sponsorSegments))
	for _, segment := range m.sponsorSegments {
		if seen[segment.Category] {
			continue
		}
		category, ok := SponsorCategories[segment.Category]
		if !ok {
			continue
		}
		seen[segment.Category] = true
		items = append(items,
			category.Color.Render(category.Glyph)+" "+Faint.Render(category.Label))
	}
	if len(items) == 0 {
		return ""
	}
	return ansi.Truncate(strings.Join(items, "  "), width, "…")
}

func (m Model) renderSplitsPanelLines(width, maxVisible int) []string {
	if len(m.splits) == 0 {
		return nil
	}

	panelWidth := min(width, splitsPanelMaxWidth)

	var lines []string
	var totalDur float64
	for _, s := range m.splits {
		totalDur += s.end - s.start
	}

	headerLabel := fmt.Sprintf("── splits (%d) ", len(m.splits))
	fillLen := max(panelWidth-lipgloss.Width(headerLabel), 1)
	lines = append(lines, " "+Dim.Render(headerLabel+strings.Repeat("─", fillLen)))

	hidden := 0
	visible := m.splits
	if maxVisible > 0 && len(m.splits) > maxVisible {
		hidden = len(m.splits) - maxVisible
		visible = m.splits[hidden:]
	}

	if hidden > 0 {
		lines = append(lines, "   "+Faint.Render(fmt.Sprintf("… %d more above", hidden)))
	}

	for i, s := range visible {
		dur := s.end - s.start
		line := fmt.Sprintf("   %s  %s - %s  %s",
			Faint.Render(strconv.Itoa(hidden+i+1)),
			Value.Render(timecode.FormatShort(s.start)),
			Value.Render(timecode.FormatShort(s.end)),
			Faint.Render("("+timecode.FormatShort(dur)+")"))
		lines = append(lines, line)
	}

	footerLabel := fmt.Sprintf("──────── total %s ", timecode.FormatShort(totalDur))
	footerFill := max(panelWidth-lipgloss.Width(footerLabel), 1)
	lines = append(lines, " "+Dim.Render(footerLabel+strings.Repeat("─", footerFill)))

	return lines
}
