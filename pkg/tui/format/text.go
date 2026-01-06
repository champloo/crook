// Package format provides formatting helpers for TUI output.
package format

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// DisplayWidth returns the visible width of an unstyled string (no ANSI escape codes).
func DisplayWidth(s string) int {
	return runewidth.StringWidth(s)
}

// Truncate trims an unstyled string (no ANSI escape codes) to a maximum display width.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	return runewidth.Truncate(s, width, "")
}

// PadRight pads a string on the right to the target display width.
func PadRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	displayWidth := runewidth.StringWidth(s)
	if displayWidth >= width {
		return Truncate(s, width)
	}
	return s + strings.Repeat(" ", width-displayWidth)
}
