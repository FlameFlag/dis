package util

import "github.com/atotto/clipboard"

// CopyToClipboard copies the given text to the system clipboard.
func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}
