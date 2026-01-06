// Package styles provides theming and styling utilities for the TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette for the TUI interface
// These colors are defined to work with both 256-color and 16-color terminals
var (
	// Primary colors
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7C7AE6"}
	ColorPrimaryFg = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}

	// Status colors (semantic)
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "#00AF87", Dark: "#00D787"} // Green
	ColorSuccessFg = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}

	ColorWarning   = lipgloss.AdaptiveColor{Light: "#D7AF00", Dark: "#FFD700"} // Yellow
	ColorWarningFg = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}

	ColorError   = lipgloss.AdaptiveColor{Light: "#D70000", Dark: "#FF5F5F"} // Red
	ColorErrorFg = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}

	ColorInfo   = lipgloss.AdaptiveColor{Light: "#0087D7", Dark: "#5FAFFF"} // Blue
	ColorInfoFg = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}

	// Progress indicator colors
	ColorInProgress = ColorInfo    // Blue for in-progress
	ColorComplete   = ColorSuccess // Green for complete
	ColorFailed     = ColorError   // Red for error

	// UI element colors
	ColorBorder        = lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#585858"}
	ColorSubtle        = lipgloss.AdaptiveColor{Light: "#6C6C6C", Dark: "#8A8A8A"}
	ColorHighlight     = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	ColorBackground    = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}
	ColorSubtleBg      = lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#303030"} // Subtle background for group headers
	ColorSubtleBgLight = lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#252525"} // Even lighter variant
)

// Text styles for various UI elements
var (
	// StyleHeading is used for section headings and titles
	StyleHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// StyleNormal is the default text style
	StyleNormal = lipgloss.NewStyle().
			Foreground(ColorHighlight)

	// StyleStatus is used for status messages and labels
	StyleStatus = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	// StyleError is used for error messages
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// StyleWarning is used for warning messages
	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// StyleSuccess is used for success messages
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// StyleSubtle is used for secondary information
	StyleSubtle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	// StyleHighlight is used for emphasized text
	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// StyleGroupHeader is used for group headers in tables/lists
	StyleGroupHeader = lipgloss.NewStyle().
				Foreground(ColorSubtle).
				Background(ColorSubtleBg).
				Bold(true)
)

// Border styles for boxes and containers
var (
	// BorderNormal is the default border style
	BorderNormal = lipgloss.NormalBorder()

	// BorderRounded is a rounded border style
	BorderRounded = lipgloss.RoundedBorder()

	// BorderDouble is a double-line border style
	BorderDouble = lipgloss.DoubleBorder()

	// BorderThick is a thick border style
	BorderThick = lipgloss.ThickBorder()

	// StyleBox is a standard bordered box
	StyleBox = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// StyleBoxSuccess is a success-themed box
	StyleBoxSuccess = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorSuccess).
			Padding(1, 2)

	// StyleBoxError is an error-themed box
	StyleBoxError = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorError).
			Padding(1, 2)

	// StyleBoxWarning is a warning-themed box
	StyleBoxWarning = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorWarning).
			Padding(1, 2)

	// StyleBoxInfo is an info-themed box
	StyleBoxInfo = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorInfo).
			Padding(1, 2)
)

// Progress bar styles
var (
	// StyleProgressBar is the style for progress bars
	StyleProgressBar = lipgloss.NewStyle().
				Foreground(ColorInProgress)

	// StyleProgressBarComplete is the style for completed progress bars
	StyleProgressBarComplete = lipgloss.NewStyle().
					Foreground(ColorComplete)

	// StyleProgressBarError is the style for failed progress bars
	StyleProgressBarError = lipgloss.NewStyle().
				Foreground(ColorFailed)
)

// Icons and symbols
const (
	IconCheckmark = "✓"
	IconCross     = "✗"
	IconWarning   = "⚠"
	IconInfo      = "ℹ"
	IconSpinner   = "◐"
	IconArrow     = "→"
)

// Fallback icons for terminals without Unicode support
const (
	IconCheckmarkASCII = "[OK]"
	IconCrossASCII     = "[X]"
	IconWarningASCII   = "[!]"
	IconInfoASCII      = "[i]"
	IconSpinnerASCII   = "[*]"
	IconArrowASCII     = "->"
)
