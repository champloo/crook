// Package format provides formatting helpers for TUI output.
package format

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// DisplayWidth returns the visible width of a string in terminal cells.
// ANSI escape codes are ignored and wide characters (e.g. CJK, emojis) are accounted for.
func DisplayWidth(s string) int {
	return ansi.StringWidth(s)
}

// Truncate trims a string to a maximum display width in terminal cells.
// ANSI escape codes are preserved and not broken.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

// PadRight pads a string on the right to the target display width.
func PadRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	displayWidth := DisplayWidth(s)
	if displayWidth >= width {
		return Truncate(s, width)
	}
	return s + strings.Repeat(" ", width-displayWidth)
}
