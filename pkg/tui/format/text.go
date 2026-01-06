// Package format provides formatting helpers for TUI output.
package format

import (
	"strings"
	"unicode"

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

// TruncateWithEllipsis truncates a string to a maximum display width and adds "..." if truncated.
// ANSI escape codes are preserved and not broken.
func TruncateWithEllipsis(s string, width int) string {
	if width <= 3 {
		return Truncate(s, width)
	}
	if DisplayWidth(s) <= width {
		return s
	}
	return Truncate(s, width-3) + "..."
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

// SanitizeForDisplay removes control characters and ANSI escape sequences from a string.
// This prevents malicious or malformed error messages from corrupting the terminal display.
// Printable characters, spaces, and newlines are preserved.
func SanitizeForDisplay(s string) string {
	// First strip ANSI escape sequences
	s = ansi.Strip(s)

	// Then remove control characters (except space, tab, newline)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || (unicode.IsPrint(r) && !unicode.IsControl(r)) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
