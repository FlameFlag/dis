package validate

import "fmt"

func intRange(name string, val, lo, hi int) error {
	if val < lo || val > hi {
		return fmt.Errorf("%s must be between %d and %d (got %d)", name, lo, hi, val)
	}
	return nil
}

func floatRange(name string, val, lo, hi float64) error {
	if val < lo || val > hi {
		return fmt.Errorf("%s must be between %.1f and %.1f (got %.1f)", name, lo, hi, val)
	}
	return nil
}
