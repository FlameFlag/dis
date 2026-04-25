// Package textbuf provides byte-buffer helpers for placing labels onto a
// fixed-width row without overlapping previously-written text.
package textbuf

// HasOverlap reports whether any byte in buf[start:start+length] is non-space.
func HasOverlap(buf []byte, start, length int) bool {
	for i := start; i < start+length && i < len(buf); i++ {
		if buf[i] != ' ' {
			return true
		}
	}
	return false
}

// Place writes s into buf starting at start, stopping at the buffer end.
func Place(buf []byte, start int, s string) {
	for i := 0; i < len(s) && start+i < len(buf); i++ {
		buf[start+i] = s[i]
	}
}
