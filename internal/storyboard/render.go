package storyboard

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/charmbracelet/x/ansi/sixel"
	xdraw "golang.org/x/image/draw"
)

const (
	graphicsNone  = 0
	graphicsKitty = 1
	graphicsSixel = 2
)

var (
	graphicsOnce     sync.Once
	graphicsProtocol int
)

func detectGraphics() int {
	// Zellij does not support kitty graphics protocol and its sixel
	// implementation is broken since v0.40.0. Fall back to half-block.
	if os.Getenv("ZELLIJ") != "" {
		return graphicsNone
	}

	term := os.Getenv("TERM")
	termProg := os.Getenv("TERM_PROGRAM")

	// Kitty graphics protocol
	if term == "xterm-kitty" ||
		termProg == "WezTerm" ||
		termProg == "kitty" ||
		termProg == "ghostty" {
		return graphicsKitty
	}

	// Sixel support: foot, xterm (with sixel build), mlterm, contour, etc.
	if termProg == "foot" ||
		termProg == "mlterm" ||
		termProg == "contour" ||
		term == "xterm-256color" && termProg == "" {
		return graphicsSixel
	}

	return graphicsNone
}

// GraphicsProtocol returns the detected graphics protocol.
func GraphicsProtocol() int {
	graphicsOnce.Do(func() {
		graphicsProtocol = detectGraphics()
	})
	return graphicsProtocol
}

// IsKittySupported returns true if the terminal supports the Kitty graphics protocol.
func IsKittySupported() bool {
	return GraphicsProtocol() == graphicsKitty
}

// IsSixelSupported returns true if the terminal likely supports Sixel graphics.
func IsSixelSupported() bool {
	return GraphicsProtocol() == graphicsSixel
}

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
