package storyboard

import (
	"bytes"
	"image"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
	xdraw "golang.org/x/image/draw"
)

// RenderSixel renders an image using the Sixel graphics protocol.
// cols and rows specify the display size in terminal cells.
func RenderSixel(img image.Image, cols, rows int) string {
	if img == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	// Sixel operates in pixels; estimate terminal cell size as ~8x16 pixels
	pixW, pixH := cols*8, rows*16
	resized := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var payload bytes.Buffer
	enc := &sixel.Encoder{}
	if err := enc.Encode(&payload, resized); err != nil {
		return ""
	}

	// Wrap payload in DCS sequence: DCS 0;1;0 q <payload> ST
	// p2=1 avoids the black-bar transparency issue
	sixelSeq := ansi.SixelGraphics(0, 1, 0, payload.Bytes())

	// Add spacing so the TUI layout allocates correct vertical space
	// (same approach as RenderKitty).
	spaces := strings.Repeat(" ", cols)
	var sb strings.Builder
	sb.WriteString(sixelSeq)
	sb.WriteString(spaces)
	for range rows - 1 {
		sb.WriteByte('\n')
		sb.WriteString(spaces)
	}
	return sb.String()
}
