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

// View wraps the given content with a styled border and title bar.
// The styling depends on whether the pane is active or inactive.
func (p *Pane) View(content string) string {
	// Select styles based on active state
	var borderStyle lipgloss.Style
	var titleStyle lipgloss.Style

	if p.active {
		borderStyle = styles.StylePaneActive
		titleStyle = styles.StylePaneTitleActive
	} else {
		borderStyle = styles.StylePaneInactive
		titleStyle = styles.StylePaneTitleInactive
	}

	// Build title bar: [1:Nodes (6)] or [2:Deployments (12/15)]
	titleText := p.buildTitleText()
	renderedTitle := titleStyle.Render(titleText)

	// Calculate content dimensions
	// Border takes 2 chars on each side (border + padding)
	// We need to account for the border characters themselves
	contentWidth := p.width - 4   // 2 for left border+padding, 2 for right border+padding
	contentHeight := p.height - 2 // 2 for top and bottom borders

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Clip/pad content to fit within pane dimensions
	clippedContent := p.clipContent(content, contentWidth, contentHeight)

	// Combine title and content
	var b strings.Builder
	b.WriteString(renderedTitle)
	b.WriteString("\n")
	b.WriteString(clippedContent)

	// Apply border style with dimensions
	styled := borderStyle.
		Width(contentWidth).
		Height(contentHeight + 1). // +1 for title line
		Render(b.String())

	return styled
}

// buildTitleText creates the title bar text with shortcut, title, and badge.
func (p *Pane) buildTitleText() string {
	var parts []string

	// Add shortcut key if present
	if p.config.ShortcutKey != "" {
		parts = append(parts, p.config.ShortcutKey+":")
	}

	// Add title
	parts = append(parts, p.config.Title)

	title := strings.Join(parts, "")

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
			// Use a simple approach: try to cut at width and add ellipsis
			line = truncateWithWidth(line, width-3) + "..."
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
