package slider

import (
	"bytes"
	"dis/internal/sponsorblock"
	"dis/internal/tui/slider/style"
	"dis/internal/tui/slider/textbuf"
	"dis/internal/util"
	"fmt"
	"strings"

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

	buf := bytes.Repeat([]byte{' '}, width)

	for _, ch := range chapters {
		lbl := ch.title
		maxLen := width / max(len(chapters), 1)
		if len(lbl) > maxLen && maxLen > 3 {
			lbl = ansi.Truncate(lbl, maxLen, "…")
		}
		start := max(ch.pos-len(lbl)/2, 0)
		if start+len(lbl) > width {
			start = width - len(lbl)
		}
		if start < 0 {
			continue
		}
		if textbuf.HasOverlap(buf, start, len(lbl)) {
			start = ch.pos + 1
			if start+len(lbl) > width {
				continue
			}
			if textbuf.HasOverlap(buf, start, len(lbl)) {
				continue
			}
		}
		textbuf.Place(buf, start, lbl)
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
				b.WriteString(style.Warm.Render(result[textStart:i]))
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
		sc, ok := style.SponsorCategories[cats[i]]
		if !ok {
			sc.Color = style.Dim
		}
		if cats[i] == sponsorblock.CategoryHighlight {
			b.WriteString(sc.Color.Render("★"))
		} else {
			b.WriteString(sc.Color.Render("▁"))
		}
	}
	return b.String()
}

func (m Model) renderSplitsPanelLines(width, maxVisible int) []string {
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
	lines = append(lines, " "+style.Dim.Render(headerLabel+strings.Repeat("─", fillLen)))

	hidden := 0
	visible := m.splits
	if maxVisible > 0 && len(m.splits) > maxVisible {
		hidden = len(m.splits) - maxVisible
		visible = m.splits[hidden:]
	}

	if hidden > 0 {
		lines = append(lines, "   "+style.Faint.Render(fmt.Sprintf("… %d more above", hidden)))
	}

	for i, s := range visible {
		dur := s.end - s.start
		line := fmt.Sprintf("   %s  %s - %s  %s",
			style.Faint.Render(fmt.Sprintf("%d", hidden+i+1)),
			style.Value.Render(util.FormatDurationShort(s.start)),
			style.Value.Render(util.FormatDurationShort(s.end)),
			style.Faint.Render("("+util.FormatDurationShort(dur)+")"))
		lines = append(lines, line)
	}

	footerLabel := fmt.Sprintf("──────── total %s ", util.FormatDurationShort(totalDur))
	footerFill := max(panelWidth-len(footerLabel), 1)
	lines = append(lines, " "+style.Dim.Render(footerLabel+strings.Repeat("─", footerFill)))

	return lines
}
