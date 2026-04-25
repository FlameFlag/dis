package validate

// GIFFps checks that the GIF frame rate is within valid range.
func GIFFps(fps int) error { return intRange("GIF fps", fps, 1, 50) }

// GIFWidth checks that the GIF width is within valid range.
func GIFWidth(width int) error { return intRange("GIF width", width, 1, 3840) }

// GIFQuality checks that the GIF quality is within valid range.
func GIFQuality(quality int) error { return intRange("GIF quality", quality, 1, 100) }

// GIFLossyQuality checks that the GIF lossy quality is within valid range.
func GIFLossyQuality(quality int) error { return intRange("GIF lossy quality", quality, 1, 100) }

// GIFMotionQuality checks that the GIF motion quality is within valid range.
func GIFMotionQuality(quality int) error { return intRange("GIF motion quality", quality, 1, 100) }
