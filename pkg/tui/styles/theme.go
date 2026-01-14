// Package styles provides theming and styling utilities for the TUI.
package styles

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Color palette for the TUI interface
// These colors are defined to work with both 256-color and 16-color terminals
var (
	// Primary colors
	ColorPrimary   = compat.AdaptiveColor{Light: lipgloss.Color("#5A56E0"), Dark: lipgloss.Color("#7C7AE6")}
	ColorPrimaryFg = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#FFFFFF")}

	// Status colors (semantic)
	ColorSuccess   = compat.AdaptiveColor{Light: lipgloss.Color("#00AF87"), Dark: lipgloss.Color("#00D787")} // Green
	ColorSuccessFg = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#000000")}

	ColorWarning   = compat.AdaptiveColor{Light: lipgloss.Color("#D7AF00"), Dark: lipgloss.Color("#FFD700")} // Yellow
	ColorWarningFg = compat.AdaptiveColor{Light: lipgloss.Color("#000000"), Dark: lipgloss.Color("#000000")}

	ColorError   = compat.AdaptiveColor{Light: lipgloss.Color("#D70000"), Dark: lipgloss.Color("#FF5F5F")} // Red
	ColorErrorFg = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#000000")}

	ColorInfo   = compat.AdaptiveColor{Light: lipgloss.Color("#0087D7"), Dark: lipgloss.Color("#5FAFFF")} // Blue
	ColorInfoFg = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#000000")}

	// Progress indicator colors
	ColorInProgress = ColorInfo    // Blue for in-progress
	ColorComplete   = ColorSuccess // Green for complete
	ColorFailed     = ColorError   // Red for error

	// UI element colors
	ColorBorder        = compat.AdaptiveColor{Light: lipgloss.Color("#B2B2B2"), Dark: lipgloss.Color("#585858")}
	ColorSubtle        = compat.AdaptiveColor{Light: lipgloss.Color("#6C6C6C"), Dark: lipgloss.Color("#8A8A8A")}
	ColorHighlight     = compat.AdaptiveColor{Light: lipgloss.Color("#000000"), Dark: lipgloss.Color("#FFFFFF")}
	ColorBackground    = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#000000")}
	ColorSubtleBg      = compat.AdaptiveColor{Light: lipgloss.Color("#E8E8E8"), Dark: lipgloss.Color("#303030")} // Subtle background for group headers
	ColorSubtleBgLight = compat.AdaptiveColor{Light: lipgloss.Color("#F0F0F0"), Dark: lipgloss.Color("#252525")} // Even lighter variant
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

// Pane border styles for multi-pane layouts
var (
	// StylePaneActive is the highlighted border style for focused panes
	StylePaneActive = lipgloss.NewStyle().
			Border(BorderRounded).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// StylePaneInactive is the muted border style for unfocused panes
	StylePaneInactive = lipgloss.NewStyle().
				Border(BorderRounded).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	// StylePaneTitleActive is the bold title style for active panes
	StylePaneTitleActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	// StylePaneTitleInactive is the muted title style for inactive panes
	StylePaneTitleInactive = lipgloss.NewStyle().
				Foreground(ColorSubtle)
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

// Maintenance mode styles for status bar
var (
	// ColorMaintenance is the color for maintenance mode elements
	ColorMaintenance   = compat.AdaptiveColor{Light: lipgloss.Color("#D75F00"), Dark: lipgloss.Color("#FF8700")} // Orange
	ColorMaintenanceFg = compat.AdaptiveColor{Light: lipgloss.Color("#FFFFFF"), Dark: lipgloss.Color("#000000")}

	// StyleMaintenanceBadge is the badge shown when maintenance flow is active
	StyleMaintenanceBadge = lipgloss.NewStyle().
				Background(ColorMaintenance).
				Foreground(ColorMaintenanceFg).
				Bold(true).
				Padding(0, 1)
)

// Maintenance mode icon
const IconMaintenance = "⚙"
