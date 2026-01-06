// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// PaneConfig holds configuration for a Pane component.
type PaneConfig struct {
	// Title is the pane title (e.g., "Nodes", "Deployments", "OSDs")
	Title string

	// ShortcutKey is the keyboard shortcut to activate this pane (e.g., "1", "2", "3")
	ShortcutKey string
}

// Pane is a component that wraps view content with a styled border and title bar.
// It supports active/inactive states with different visual styling.
type Pane struct {
	config PaneConfig
	active bool
	badge  string
	width  int
	height int
}

// NewPane creates a new Pane with the given configuration.
func NewPane(config PaneConfig) *Pane {
	return &Pane{
		config: config,
		active: false,
		badge:  "",
		width:  80,
		height: 10,
	}
}

// SetActive sets whether this pane is the active (focused) pane.
// Active panes receive highlighted borders and title styling.
func (p *Pane) SetActive(active bool) {
	p.active = active
}

// IsActive returns whether this pane is currently active.
func (p *Pane) IsActive() bool {
	return p.active
}

// SetBadge sets the badge text (typically a count like "6" or "12/15").
func (p *Pane) SetBadge(badge string) {
	p.badge = badge
}

// GetBadge returns the current badge text.
func (p *Pane) GetBadge() string {
	return p.badge
}

// SetSize sets the dimensions of the pane.
func (p *Pane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// GetSize returns the current width and height of the pane.
func (p *Pane) GetSize() (int, int) {
	return p.width, p.height
}

// GetTitle returns the pane title.
func (p *Pane) GetTitle() string {
	return p.config.Title
}

// SetTitle updates the pane title.
func (p *Pane) SetTitle(title string) {
	p.config.Title = title
}

// GetShortcutKey returns the keyboard shortcut for this pane.
func (p *Pane) GetShortcutKey() string {
	return p.config.ShortcutKey
}

// View wraps the given content with a styled border and title in the top border.
// The title is rendered as: ╭─[1] Nodes (3)───────────────╮
// The styling depends on whether the pane is active or inactive.
func (p *Pane) View(content string) string {
	// Select border color based on active state
	var borderColor lipgloss.AdaptiveColor
	var titleStyle lipgloss.Style

	if p.active {
		borderColor = styles.ColorPrimary
		titleStyle = styles.StylePaneTitleActive
	} else {
		borderColor = styles.ColorBorder
		titleStyle = styles.StylePaneTitleInactive
	}

	// Calculate content dimensions
	// Border takes 1 char on each side, padding adds 1 more on each side
	contentWidth := p.width - 4   // 2 for borders, 2 for padding
	contentHeight := p.height - 2 // 2 for top and bottom borders

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Build the title text: [1] Nodes (3)
	titleText := p.buildTitleText()

	// Clip/pad content to fit within pane dimensions
	clippedContent := p.clipContent(content, contentWidth, contentHeight)

	// Build the box manually with title in top border
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Ensure minimum width for border rendering
	borderWidth := p.width
	if borderWidth < 4 {
		borderWidth = 4
	}

	// Top border with title: ╭─[1] Nodes (3)─────────╮
	topLeft := borderStyle.Render("╭─")
	renderedTitle := titleStyle.Render(titleText)
	titleWidth := lipgloss.Width(titleText)
	// Calculate remaining dashes needed (subtract title width and corners)
	remainingWidth := borderWidth - 3 - titleWidth // 3 for ╭─ and ╮
	if remainingWidth < 1 {
		remainingWidth = 1
	}
	topRight := borderStyle.Render(strings.Repeat("─", remainingWidth) + "╮")
	topBorder := topLeft + renderedTitle + topRight

	// Side borders
	leftBorder := borderStyle.Render("│ ")
	rightBorder := borderStyle.Render(" │")

	// Bottom border: ╰────────────────────────╯
	bottomWidth := borderWidth - 2
	if bottomWidth < 1 {
		bottomWidth = 1
	}
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", bottomWidth) + "╯")

	// Build the complete box
	var b strings.Builder
	b.WriteString(topBorder)
	b.WriteString("\n")

	// Add content lines with side borders
	lines := strings.Split(clippedContent, "\n")
	for _, line := range lines {
		b.WriteString(leftBorder)
		b.WriteString(line)
		b.WriteString(rightBorder)
		b.WriteString("\n")
	}

	b.WriteString(bottomBorder)

	return b.String()
}

// buildTitleText creates the title text with shortcut, title, and badge.
// Format: [1] Nodes (3) or just Nodes (3) if no shortcut key.
func (p *Pane) buildTitleText() string {
	var title string

	// Add shortcut key in brackets if present
	if p.config.ShortcutKey != "" {
		title = fmt.Sprintf("[%s] %s", p.config.ShortcutKey, p.config.Title)
	} else {
		title = p.config.Title
	}

	// Add badge in parentheses if present
	if p.badge != "" {
		title = fmt.Sprintf("%s (%s)", title, p.badge)
	}

	return title
}

// clipContent clips and pads the content to fit within the specified dimensions.
func (p *Pane) clipContent(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	// Limit number of lines to height
	if len(lines) > height {
		lines = lines[:height]
	}

	// Process each line: clip or pad to width
	result := make([]string, 0, height)
	for _, line := range lines {
		// Calculate visible width (accounting for ANSI escape codes)
		visibleWidth := lipgloss.Width(line)

		if visibleWidth > width {
			// Need to truncate - this is tricky with ANSI codes
			// Use a simple approach: try to cut at width and add ellipsis if it fits
			if width <= 3 {
				line = truncateWithWidth(line, width)
			} else {
				line = truncateWithWidth(line, width-3) + "..."
			}
		} else if visibleWidth < width {
			// Pad with spaces to fill width
			line = line + strings.Repeat(" ", width-visibleWidth)
		}
		result = append(result, line)
	}

	// Pad with empty lines if needed
	for len(result) < height {
		result = append(result, strings.Repeat(" ", width))
	}

	return strings.Join(result, "\n")
}

// truncateWithWidth truncates a string to approximately the given display width.
// This is a best-effort approach that handles most common cases.
func truncateWithWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}

	// Simple approach: iterate through runes and count visible width
	// This doesn't perfectly handle ANSI codes but works for most cases
	result := []rune{}
	currentWidth := 0

	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result = append(result, r)
			continue
		}

		if inEscape {
			result = append(result, r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// Count this rune's width (simplified: assume 1 for most chars)
		runeWidth := 1
		if currentWidth+runeWidth > width {
			break
		}

		result = append(result, r)
		currentWidth += runeWidth
	}

	return string(result)
}
