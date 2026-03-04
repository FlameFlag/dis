package tui

import (
	"dis/internal/config"
	"slices"

	"github.com/charmbracelet/huh"
)

// ChapterSelection holds the result of interactive chapter selection.
type ChapterSelection struct {
	Chapters []config.Chapter
	Mode     config.ChapterMode
	Reverse  bool
}

// SelectChapters runs an interactive chapter selection flow.
// Returns nil if no chapters are selected.
func SelectChapters(chapters []config.Chapter) (*ChapterSelection, error) {
	// Build options for multi-select
	opts := make([]huh.Option[int], len(chapters))
	for i, ch := range chapters {
		opts[i] = huh.NewOption(ch.Label(), i)
	}

	// Step 1: Pick chapters
	var selected []int
	err := huh.NewMultiSelect[int]().
		Title("Select chapters to download").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		return nil, nil
	}

	// Gather selected chapters
	var picked []config.Chapter
	for _, idx := range selected {
		picked = append(picked, chapters[idx])
	}

	mode := config.ChapterModeSeparate
	reverse := false

	// Step 2: Combine into single video? (only if >1 selected)
	if len(picked) > 1 {
		var combine bool
		err = huh.NewConfirm().
			Title("Combine into single video?").
			Value(&combine).
			Run()
		if err != nil {
			return nil, err
		}

		if combine {
			mode = config.ChapterModeCombined

			// Step 3: Order selection
			var order string
			err = huh.NewSelect[string]().
				Title("Chapter order").
				Options(
					huh.NewOption("Original order (recommended)", "original"),
					huh.NewOption("Reverse order", "reverse"),
				).
				Value(&order).
				Run()
			if err != nil {
				return nil, err
			}

			if order == "reverse" {
				reverse = true
				slices.Reverse(picked)
			}
		}
	}

	return &ChapterSelection{
		Chapters: picked,
		Mode:     mode,
		Reverse:  reverse,
	}, nil
}
