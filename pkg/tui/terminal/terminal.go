// Package terminal provides terminal detection and compatibility utilities.
package terminal

import (
	"os"
	"strings"
)

// Capability represents terminal capabilities
type Capability struct {
	// Has256Colors indicates 256-color support
	Has256Colors bool

	// Has16Colors indicates basic 16-color support
	Has16Colors bool

	// HasNoColors indicates no color support (TERM=dumb)
	HasNoColors bool

	// HasUnicode indicates Unicode symbol support
	HasUnicode bool

	// IsTmux indicates running inside tmux
	IsTmux bool

	// IsScreen indicates running inside GNU screen
	IsScreen bool

	// Term is the TERM environment variable
	Term string
}

// MinRecommendedWidth is the minimum recommended terminal width
const MinRecommendedWidth = 80

// MinRecommendedHeight is the minimum recommended terminal height
const MinRecommendedHeight = 24

// DetectCapabilities detects terminal capabilities from environment
func DetectCapabilities() Capability {
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")

	cap := Capability{
		Term:         term,
		Has256Colors: false,
		Has16Colors:  false,
		HasNoColors:  false,
		HasUnicode:   true, // Assume Unicode by default
		IsTmux:       os.Getenv("TMUX") != "",
		IsScreen:     strings.Contains(term, "screen"),
	}

	// Detect color support
	switch {
	case term == "dumb" || term == "":
		cap.HasNoColors = true
		cap.HasUnicode = false
	case colorTerm == "truecolor" || colorTerm == "24bit":
		cap.Has256Colors = true
	case strings.Contains(term, "256color"):
		cap.Has256Colors = true
	case strings.HasPrefix(term, "xterm") ||
		strings.HasPrefix(term, "screen") ||
		strings.HasPrefix(term, "tmux") ||
		strings.HasPrefix(term, "rxvt") ||
		strings.HasPrefix(term, "linux"):
		cap.Has16Colors = true
	default:
		// Conservative fallback
		cap.Has16Colors = true
	}

	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		cap.HasNoColors = true
		cap.Has256Colors = false
		cap.Has16Colors = false
	}

	return cap
}

// Icons provides terminal-appropriate icons
type Icons struct {
	Checkmark string
	Cross     string
	Warning   string
	Info      string
	Spinner   string
	Arrow     string
	Pending   string
}

// GetIcons returns appropriate icons for the terminal
func GetIcons(cap Capability) Icons {
	if cap.HasNoColors || !cap.HasUnicode {
		return Icons{
			Checkmark: "[OK]",
			Cross:     "[X]",
			Warning:   "[!]",
			Info:      "[i]",
			Spinner:   "[*]",
			Arrow:     "->",
			Pending:   "[ ]",
		}
	}

	return Icons{
		Checkmark: "✓",
		Cross:     "✗",
		Warning:   "⚠",
		Info:      "ℹ",
		Spinner:   "◐",
		Arrow:     "→",
		Pending:   "○",
	}
}

// SpinnerFrames returns appropriate spinner frames for the terminal
func SpinnerFrames(cap Capability) []string {
	if cap.HasNoColors || !cap.HasUnicode {
		return []string{"-", "\\", "|", "/"}
	}
	return []string{"◐", "◓", "◑", "◒"}
}

// ProgressChars returns appropriate progress bar characters
type ProgressChars struct {
	Full  string
	Empty string
}

// GetProgressChars returns appropriate progress bar characters
func GetProgressChars(cap Capability) ProgressChars {
	if cap.HasNoColors || !cap.HasUnicode {
		return ProgressChars{
			Full:  "=",
			Empty: "-",
		}
	}
	return ProgressChars{
		Full:  "█",
		Empty: "░",
	}
}

// IsTooNarrow checks if the terminal width is below minimum
func IsTooNarrow(width int) bool {
	return width > 0 && width < MinRecommendedWidth
}

// IsTooShort checks if the terminal height is below minimum
func IsTooShort(height int) bool {
	return height > 0 && height < MinRecommendedHeight
}

// SizeWarning returns a warning message if terminal is too small
func SizeWarning(width, height int) string {
	var warnings []string

	if IsTooNarrow(width) {
		warnings = append(warnings, "Terminal too narrow, recommend 80+ columns")
	}
	if IsTooShort(height) {
		warnings = append(warnings, "Terminal too short, recommend 24+ rows")
	}

	if len(warnings) == 0 {
		return ""
	}

	return strings.Join(warnings, "; ")
}

// ConfigureLipgloss configures lipgloss based on terminal capabilities.
// In lipgloss v2 with bubbletea v2, color profile is managed automatically
// by the tea.Program, so this function is now a no-op.
func ConfigureLipgloss(_ Capability) {
	// Color profile is now managed automatically by bubbletea v2
}
