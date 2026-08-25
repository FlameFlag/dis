package slider

import (
	"strconv"

	"github.com/4evy/dis/internal/storyboard"
)

// thumbnailCache stores the last rendered thumbnail to avoid re-rendering on every frame.
type thumbnailCache struct {
	cellKey string
	width   int
	height  int
	output  string
}

var thumbCache thumbnailCache

func (m Model) renderThumbnail(width, maxHeight int) string {
	if m.storyboard == nil || m.height < minimumThumbnailScreenHeight || m.isStacked() ||
		(maxHeight >= 0 && maxHeight < minimumThumbnailHeight) {
		if storyboard.IsKittySupported() {
			return storyboard.DeleteKittyImage()
		}
		return ""
	}

	thumbW := min(width-thumbnailHorizontalInset, thumbnailMaximumWidth)
	thumbH := thumbnailDefaultHeight
	if maxHeight > 0 {
		thumbH = min(thumbH, maxHeight)
	}

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
	cacheKey := strconv.Itoa(quantized)

	if thumbCache.cellKey == cacheKey && thumbCache.width == thumbW &&
		thumbCache.height == thumbH {
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
	thumbCache = thumbnailCache{
		cellKey: cacheKey,
		width:   thumbW,
		height:  thumbH,
		output:  rendered,
	}
	return rendered
}
