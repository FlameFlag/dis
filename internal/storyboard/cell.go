package storyboard

import (
	"image"
	"image/draw"
)

// CellAt extracts the thumbnail cell at the given timestamp.
func CellAt(data *StoryboardData, timestamp float64) image.Image {
	if data == nil || len(data.Images) == 0 {
		return nil
	}

	info := &data.Info
	cellsPerFragment := info.Rows * info.Columns
	accumulated := 0.0
	cellDuration := 0.0

	// Find which fragment and cell index this timestamp falls in
	for fragIdx, frag := range info.Fragments {
		if cellsPerFragment == 0 {
			continue
		}
		cellDuration = frag.Duration / float64(cellsPerFragment)
		if cellDuration <= 0 {
			continue
		}

		fragEnd := accumulated + frag.Duration
		if timestamp < fragEnd || fragIdx == len(info.Fragments)-1 {
			img, ok := data.Images[fragIdx]
			if !ok {
				return nil
			}

			// Cell index within this fragment
			localTime := timestamp - accumulated
			if localTime < 0 {
				localTime = 0
			}
			cellIdx := int(localTime / cellDuration)
			if cellIdx >= cellsPerFragment {
				cellIdx = cellsPerFragment - 1
			}

			row := cellIdx / info.Columns
			col := cellIdx % info.Columns

			x0 := col * info.CellW
			y0 := row * info.CellH
			rect := image.Rect(x0, y0, x0+info.CellW, y0+info.CellH)

			// Crop sub-image
			type subImager interface {
				SubImage(r image.Rectangle) image.Image
			}
			if si, ok := img.(subImager); ok {
				return si.SubImage(rect)
			}

			// Fallback: manual crop
			cropped := image.NewRGBA(image.Rect(0, 0, info.CellW, info.CellH))
			draw.Draw(cropped, cropped.Bounds(), img, image.Pt(x0, y0), draw.Src)
			return cropped
		}
		accumulated = fragEnd
	}

	return nil
}
