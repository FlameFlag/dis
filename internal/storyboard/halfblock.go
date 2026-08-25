package storyboard

import (
	"image"
	"strings"

	"github.com/charmbracelet/x/ansi"
	xdraw "golang.org/x/image/draw"
)

const (
	halfBlockPixelHeight = 2
	colorChannelShift    = 8
)

// RenderHalfBlock renders an image as a string using half-block characters (▀)
// with true-color ANSI escapes. Each character represents 2 vertical pixels.
// targetW and targetH are in character cells (targetH chars = targetH*2 pixels).
func RenderHalfBlock(img image.Image, targetW, targetH int) string {
	if img == nil || targetW <= 0 || targetH <= 0 {
		return ""
	}

	// Resize to targetW x (targetH*2) pixels
	pixH := targetH * halfBlockPixelHeight
	resized := image.NewRGBA(image.Rect(0, 0, targetW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var b strings.Builder
	for y := 0; y < pixH; y += halfBlockPixelHeight {
		for x := range targetW {
			tr, tg, tb, _ := resized.At(x, y).RGBA()
			br, bg, bb, _ := resized.At(x, y+1).RGBA()
			style := ansi.Style{}.
				ForegroundColor(ansi.RGBColor{
					R: uint8(tr >> colorChannelShift),
					G: uint8(tg >> colorChannelShift),
					B: uint8(tb >> colorChannelShift),
				}).
				BackgroundColor(ansi.RGBColor{
					R: uint8(br >> colorChannelShift),
					G: uint8(bg >> colorChannelShift),
					B: uint8(bb >> colorChannelShift),
				})
			b.WriteString(style.String())
			b.WriteRune('▀')
		}
		b.WriteString(ansi.ResetStyle)
		if y+halfBlockPixelHeight < pixH {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
