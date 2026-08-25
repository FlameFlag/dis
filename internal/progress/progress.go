// Package progress defines presentation-neutral progress updates.
package progress

import "time"

// Update describes task progress without prescribing how it is displayed.
type Update struct {
	Percent     float64
	Transferred int64
	Total       int64
	Speed       float64
	ETA         time.Duration
}

// Sink receives progress updates.
type Sink interface {
	Report(Update)
}

// SinkFunc adapts a function to Sink.
type SinkFunc func(Update)

// Report implements Sink.
func (f SinkFunc) Report(update Update) {
	if f != nil {
		f(update)
	}
}

type discard struct{}

func (discard) Report(Update) {}

// Discard ignores progress updates.
var Discard Sink = discard{}
