package storyboard

import (
	"bytes"
	"image"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	xdraw "golang.org/x/image/draw"
)

const (
	kittyCellPixelWidth  = 10
	kittyCellPixelHeight = 20
	kittyImageID         = 1
	kittyQuietErrors     = 2
)

// RenderKitty renders an image using the Kitty graphics protocol.
// cols and rows specify the display size in terminal cells.
func RenderKitty(img image.Image, cols, rows int) string {
	if img == nil || cols <= 0 || rows <= 0 {
		return ""
	}

	// Pre-scale to high resolution so the terminal doesn't upscale a tiny source
	pixW := cols * kittyCellPixelWidth
	pixH := rows * kittyCellPixelHeight
	resized := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	opts := &kitty.Options{
		Action:          kitty.TransmitAndPut,
		Transmission:    kitty.Direct,
		Format:          kitty.PNG,
		Chunk:           true,
		Quiet:           kittyQuietErrors,
		ID:              kittyImageID,
		Columns:         cols,
		Rows:            rows,
		DoNotMoveCursor: true,
	}
	if err := kitty.EncodeGraphics(&buf, resized, opts); err != nil {
		return ""
	}

	// DoNotMoveCursor keeps the cursor in place. Lip Gloss reserves the cell
	// area so the surrounding layout measures the image correctly.
	return lipgloss.NewStyle().Width(cols).Height(rows).Render(buf.String())
}

// DeleteKittyImage returns an escape sequence that deletes the Kitty image with ID=1.
func DeleteKittyImage() string {
	opts := &kitty.Options{
		Action: kitty.Delete,
		Delete: kitty.DeleteID,
		ID:     kittyImageID,
		Quiet:  kittyQuietErrors,
	}
	return ansi.KittyGraphics(nil, opts.Options()...)
}
