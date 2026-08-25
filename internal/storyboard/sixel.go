package storyboard

import (
	"bytes"
	"image"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
	xdraw "golang.org/x/image/draw"
)

const (
	sixelCellPixelWidth  = 8
	sixelCellPixelHeight = 16
	sixelAspectRatio     = 0
	sixelBackgroundMode  = 1
	sixelGridSize        = 0
)

// RenderSixel renders an image using the Sixel graphics protocol.
// cols and rows specify the display size in terminal cells.
func RenderSixel(img image.Image, cols, rows int) string {
	if img == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	pixW := cols * sixelCellPixelWidth
	pixH := rows * sixelCellPixelHeight
	resized := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var payload bytes.Buffer
	enc := &sixel.Encoder{}
	if err := enc.Encode(&payload, resized); err != nil {
		return ""
	}

	// Wrap payload in DCS sequence: DCS 0;1;0 q <payload> ST
	// p2=1 avoids the black-bar transparency issue
	sixelSeq := ansi.SixelGraphics(
		sixelAspectRatio,
		sixelBackgroundMode,
		sixelGridSize,
		payload.Bytes(),
	)

	return lipgloss.NewStyle().Width(cols).Height(rows).Render(sixelSeq)
}
