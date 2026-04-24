package storyboard

import (
	"os"
	"sync"
)

// GraphicsProtocol represents a terminal graphics protocol.
type GraphicsProtocol int

const (
	GraphicsNone GraphicsProtocol = iota
	GraphicsKitty
	GraphicsSixel
)

// DetectedProtocol returns the detected graphics protocol (cached after first call).
var DetectedProtocol = sync.OnceValue(detectGraphics)

func detectGraphics() GraphicsProtocol {
	// Zellij does not support kitty graphics protocol and its sixel
	// implementation is broken since v0.40.0. Fall back to half-block.
	if os.Getenv("ZELLIJ") != "" {
		return GraphicsNone
	}

	term := os.Getenv("TERM")
	termProg := os.Getenv("TERM_PROGRAM")

	// Kitty graphics protocol
	if term == "xterm-kitty" ||
		termProg == "WezTerm" ||
		termProg == "kitty" ||
		termProg == "ghostty" {
		return GraphicsKitty
	}

	// Sixel support: foot, xterm (with sixel build), mlterm, contour, etc.
	if termProg == "foot" ||
		termProg == "mlterm" ||
		termProg == "contour" ||
		term == "xterm-256color" && termProg == "" {
		return GraphicsSixel
	}

	return GraphicsNone
}

// IsKittySupported returns true if the terminal supports the Kitty graphics protocol.
func IsKittySupported() bool {
	return DetectedProtocol() == GraphicsKitty
}

// IsSixelSupported returns true if the terminal likely supports Sixel graphics.
func IsSixelSupported() bool {
	return DetectedProtocol() == GraphicsSixel
}
