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

const (
	envZellij      = "ZELLIJ"
	envTerm        = "TERM"
	envTermProgram = "TERM_PROGRAM"
	xtermSixel     = "xterm-256color"
)

var protocolByTermProgram = map[string]GraphicsProtocol{
	"WezTerm": GraphicsKitty,
	"kitty":   GraphicsKitty,
	"ghostty": GraphicsKitty,
	"foot":    GraphicsSixel,
	"mlterm":  GraphicsSixel,
	"contour": GraphicsSixel,
}

var protocolByTerm = map[string]GraphicsProtocol{
	"xterm-kitty": GraphicsKitty,
}

// DetectedProtocol returns the detected graphics protocol (cached after first call).
var DetectedProtocol = sync.OnceValue(detectGraphics)

func detectGraphics() GraphicsProtocol {
	// Zellij does not support kitty graphics protocol and its sixel
	// implementation is broken since v0.40.0. Fall back to half-block.
	if os.Getenv(envZellij) != "" {
		return GraphicsNone
	}

	term := os.Getenv(envTerm)
	termProgram := os.Getenv(envTermProgram)
	if protocol, ok := protocolByTerm[term]; ok {
		return protocol
	}
	if protocol, ok := protocolByTermProgram[termProgram]; ok {
		return protocol
	}
	if term == xtermSixel && termProgram == "" {
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
