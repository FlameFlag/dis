package storyboard

import (
	"fmt"
	"image"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// RenderHalfBlock renders an image as a string using half-block characters (▀)
// with true-color ANSI escapes. Each character represents 2 vertical pixels.
// targetW and targetH are in character cells (targetH chars = targetH*2 pixels).
func RenderHalfBlock(img image.Image, targetW, targetH int) string {
	if img == nil || targetW <= 0 || targetH <= 0 {
		return ""
	}

	// Resize to targetW x (targetH*2) pixels
	pixH := targetH * 2
	resized := image.NewRGBA(image.Rect(0, 0, targetW, pixH))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), xdraw.Over, nil)

	var b strings.Builder
	for y := 0; y < pixH; y += 2 {
		for x := range targetW {
			tr, tg, tb, _ := resized.At(x, y).RGBA()
			br, bg, bb, _ := resized.At(x, y+1).RGBA()
			// RGBA returns 16-bit values; shift to 8-bit
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				tr>>8, tg>>8, tb>>8,
				br>>8, bg>>8, bb>>8)
		}
		b.WriteString("\x1b[0m")
		if y+2 < pixH {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
