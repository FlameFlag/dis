package slider

import (
	"dis/internal/storyboard"
	"fmt"
)

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
