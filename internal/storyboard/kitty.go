package storyboard

import (
	"bytes"
	"image"
	"strings"

	"github.com/charmbracelet/x/ansi/kitty"
	xdraw "golang.org/x/image/draw"
)

// RenderKitty renders an image using the Kitty graphics protocol.
// cols and rows specify the display size in terminal cells.
func RenderKitty(img image.Image, cols, rows int) string {
	if img == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	// Pre-scale to high resolution so the terminal doesn't upscale a tiny source
	pixW, pixH := cols*10, rows*20
	resized := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	opts := &kitty.Options{
		Action:          kitty.TransmitAndPut,
		Transmission:    kitty.Direct,
		Format:          kitty.PNG,
		Chunk:           true,
		Quite:           2,
		ID:              1,
		Columns:         cols,
		Rows:            rows,
		DoNotMoveCursor: true,
	}
	if err := kitty.EncodeGraphics(&buf, resized, opts); err != nil {
		return ""
	}

	// Build output: APC escape on first line + cols spaces per line.
	// DoNotMoveCursor keeps the cursor in place; the spaces provide correct
	// lipgloss.Width() so padRight() works correctly in the bordered layout.
	spaces := strings.Repeat(" ", cols)
	var sb strings.Builder
	sb.Write(buf.Bytes())
	sb.WriteString(spaces)
	for range rows - 1 {
		sb.WriteByte('\n')
		sb.WriteString(spaces)
	}

	return sb.String()
}

// DeleteKittyImage returns an escape sequence that deletes the Kitty image with ID=1.
func DeleteKittyImage() string {
	return "\x1b_Ga=d,d=i,i=1,q=2\x1b\\"
}
