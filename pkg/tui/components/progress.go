// Package components provides reusable Bubble Tea UI components.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/tui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ProgressState represents the state of a progress bar
type ProgressState int

const (
	// ProgressStateInProgress indicates the operation is in progress
	ProgressStateInProgress ProgressState = iota
	// ProgressStateComplete indicates the operation completed successfully
	ProgressStateComplete
	// ProgressStateError indicates the operation failed
	ProgressStateError
)

// ProgressBar is a reusable progress bar component
type ProgressBar struct {
	// Label displayed above or beside the progress bar
	Label string

	// Current progress value (0.0 to 1.0)
	Progress float64

	// State determines the color scheme
	State ProgressState

	// Width of the progress bar in characters (0 = auto-fit to terminal)
	Width int

	// ShowPercentage displays percentage on the right
	ShowPercentage bool

	// Indeterminate shows a spinner instead of progress bar
	Indeterminate bool

	// spinnerFrame for indeterminate progress
	spinnerFrame int
}

// Progress bar characters
const (
	progressFull  = "█"
	progressEmpty = "░"
)

// Spinner frames for indeterminate progress
var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

// NewProgressBar creates a new progress bar with default settings
func NewProgressBar(label string) *ProgressBar {
	return &ProgressBar{
		Label:          label,
		Progress:       0,
		State:          ProgressStateInProgress,
		Width:          40,
		ShowPercentage: true,
		Indeterminate:  false,
	}
}

// NewIndeterminateProgress creates a spinner-style progress indicator
func NewIndeterminateProgress(label string) *ProgressBar {
	return &ProgressBar{
		Label:         label,
		State:         ProgressStateInProgress,
		Indeterminate: true,
	}
}

// Init implements tea.Model
func (p *ProgressBar) Init() tea.Cmd {
	if p.Indeterminate {
		return p.tick()
	}
	return nil
}

// SpinnerTickMsg is sent to advance the spinner animation
type SpinnerTickMsg struct{}

// tick returns a command that sends a SpinnerTickMsg after a delay
func (p *ProgressBar) tick() tea.Cmd {
	return tea.Tick(100*1000000, func(_ time.Time) tea.Msg { // 100ms
		return SpinnerTickMsg{}
	})
}

// Update implements tea.Model
func (p *ProgressBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case SpinnerTickMsg:
		if p.Indeterminate && p.State == ProgressStateInProgress {
			p.spinnerFrame = (p.spinnerFrame + 1) % len(spinnerFrames)
			return p, p.tick()
		}
	}
	return p, nil
}

// View implements tea.Model
func (p *ProgressBar) View() tea.View {
	return tea.NewView(p.Render())
}

// Render returns the string representation for composition
func (p *ProgressBar) Render() string {
	if p.Indeterminate {
		return p.renderIndeterminate()
	}
	return p.renderDeterminate()
}

// renderDeterminate renders a progress bar with percentage
func (p *ProgressBar) renderDeterminate() string {
	// Clamp progress to valid range
	progress := p.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	// Calculate bar width
	barWidth := p.Width
	if barWidth <= 0 {
		barWidth = 40
	}

	// Reserve space for percentage if shown
	if p.ShowPercentage {
		barWidth -= 5 // " 100%"
	}

	// Clamp bar width to minimum of 1 to prevent negative strings.Repeat
	if barWidth < 1 {
		barWidth = 1
	}

	// Calculate filled portion
	filled := int(float64(barWidth) * progress)
	empty := barWidth - filled

	// Build the bar
	bar := strings.Repeat(progressFull, filled) + strings.Repeat(progressEmpty, empty)

	// Apply color based on state
	style := p.getBarStyle()
	coloredBar := style.Render(bar)

	// Add percentage
	var result string
	if p.ShowPercentage {
		percent := int(progress * 100)
		result = fmt.Sprintf("%s %3d%%", coloredBar, percent)
	} else {
		result = coloredBar
	}

	// Add label if present
	if p.Label != "" {
		var labelStyle lipgloss.Style
		switch p.State {
		case ProgressStateComplete:
			labelStyle = styles.StyleSuccess
		case ProgressStateError:
			labelStyle = styles.StyleError
		case ProgressStateInProgress:
			labelStyle = styles.StyleNormal
		}
		result = labelStyle.Render(p.Label) + "\n" + result
	}

	return result
}

// renderIndeterminate renders a spinner with label
func (p *ProgressBar) renderIndeterminate() string {
	var spinner string
	var style lipgloss.Style

	switch p.State {
	case ProgressStateComplete:
		spinner = styles.IconCheckmark
		style = styles.StyleSuccess
	case ProgressStateError:
		spinner = styles.IconCross
		style = styles.StyleError
	case ProgressStateInProgress:
		spinner = spinnerFrames[p.spinnerFrame]
		style = styles.StyleStatus
	}

	if p.Label != "" {
		return fmt.Sprintf("%s %s", style.Render(spinner), p.Label)
	}
	return style.Render(spinner)
}

// getBarStyle returns the appropriate style for the current state
func (p *ProgressBar) getBarStyle() lipgloss.Style {
	switch p.State {
	case ProgressStateComplete:
		return styles.StyleProgressBarComplete
	case ProgressStateError:
		return styles.StyleProgressBarError
	case ProgressStateInProgress:
		return styles.StyleProgressBar
	}
	return styles.StyleProgressBar
}

// SetProgress updates the progress value (0.0 to 1.0)
func (p *ProgressBar) SetProgress(progress float64) {
	p.Progress = progress
	if progress >= 1.0 {
		p.State = ProgressStateComplete
	}
}

// SetState updates the progress bar state
func (p *ProgressBar) SetState(state ProgressState) {
	p.State = state
}

// SetWidth updates the progress bar width
func (p *ProgressBar) SetWidth(width int) {
	p.Width = width
}

// Complete marks the progress bar as complete
func (p *ProgressBar) Complete() {
	p.Progress = 1.0
	p.State = ProgressStateComplete
}

// Error marks the progress bar as failed
func (p *ProgressBar) Error() {
	p.State = ProgressStateError
}

// Reset resets the progress bar to initial state
func (p *ProgressBar) Reset() {
	p.Progress = 0
	p.State = ProgressStateInProgress
	p.spinnerFrame = 0
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	bars  []*ProgressBar
	width int
}

// NewMultiProgress creates a container for multiple progress bars
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		bars:  make([]*ProgressBar, 0),
		width: 40,
	}
}

// AddBar adds a progress bar to the multi-progress container
func (m *MultiProgress) AddBar(bar *ProgressBar) {
	bar.Width = m.width
	m.bars = append(m.bars, bar)
}

// SetWidth sets the width for all progress bars
func (m *MultiProgress) SetWidth(width int) {
	m.width = width
	for _, bar := range m.bars {
		bar.Width = width
	}
}

// Init implements tea.Model
func (m *MultiProgress) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, bar := range m.bars {
		if cmd := bar.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (m *MultiProgress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for i, bar := range m.bars {
		newBar, cmd := bar.Update(msg)
		if pb, ok := newBar.(*ProgressBar); ok {
			m.bars[i] = pb
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *MultiProgress) View() tea.View {
	return tea.NewView(m.Render())
}

// Render returns the string representation for composition
func (m *MultiProgress) Render() string {
	var views []string
	for _, bar := range m.bars {
		views = append(views, bar.Render())
	}
	return strings.Join(views, "\n")
}

// GetBar returns a progress bar by index
func (m *MultiProgress) GetBar(index int) *ProgressBar {
	if index < 0 || index >= len(m.bars) {
		return nil
	}
	return m.bars[index]
}

// Count returns the number of progress bars
func (m *MultiProgress) Count() int {
	return len(m.bars)
}
