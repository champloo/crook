package components

import (
	"fmt"
	"time"

	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusType represents the type of status being displayed
type StatusType int

const (
	// StatusTypeInfo is for informational messages
	StatusTypeInfo StatusType = iota
	// StatusTypeSuccess is for success messages
	StatusTypeSuccess
	// StatusTypeWarning is for warning messages
	StatusTypeWarning
	// StatusTypeError is for error messages
	StatusTypeError
	// StatusTypePending is for pending/waiting states
	StatusTypePending
	// StatusTypeRunning is for in-progress operations
	StatusTypeRunning
)

// StatusIndicator displays a status with icon, label, and optional details
type StatusIndicator struct {
	// Label is the main status text
	Label string

	// Details provides additional context (optional)
	Details string

	// Type determines the icon and color
	Type StatusType

	// ShowIcon determines if the icon is displayed
	ShowIcon bool

	// Inline renders icon and label on same line (default: true)
	Inline bool

	// spinnerFrame for running state animation
	spinnerFrame int
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator(label string, statusType StatusType) *StatusIndicator {
	return &StatusIndicator{
		Label:    label,
		Type:     statusType,
		ShowIcon: true,
		Inline:   true,
	}
}

// NewInfoStatus creates an info status indicator
func NewInfoStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypeInfo)
}

// NewSuccessStatus creates a success status indicator
func NewSuccessStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypeSuccess)
}

// NewWarningStatus creates a warning status indicator
func NewWarningStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypeWarning)
}

// NewErrorStatus creates an error status indicator
func NewErrorStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypeError)
}

// NewPendingStatus creates a pending status indicator
func NewPendingStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypePending)
}

// NewRunningStatus creates a running/in-progress status indicator
func NewRunningStatus(label string) *StatusIndicator {
	return NewStatusIndicator(label, StatusTypeRunning)
}

// Init implements tea.Model
func (s *StatusIndicator) Init() tea.Cmd {
	if s.Type == StatusTypeRunning {
		return s.tick()
	}
	return nil
}

// StatusTickMsg is sent to advance the spinner animation
type StatusTickMsg struct{}

// tick returns a command that sends a StatusTickMsg after a delay
func (s *StatusIndicator) tick() tea.Cmd {
	return tea.Tick(100*1000000, func(_ time.Time) tea.Msg { // 100ms
		return StatusTickMsg{}
	})
}

// Update implements tea.Model
func (s *StatusIndicator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case StatusTickMsg:
		if s.Type == StatusTypeRunning {
			s.spinnerFrame = (s.spinnerFrame + 1) % len(spinnerFrames)
			return s, s.tick()
		}
	}
	return s, nil
}

// View implements tea.Model
func (s *StatusIndicator) View() string {
	icon := s.getIcon()
	style := s.getStyle()

	var result string

	if s.ShowIcon {
		if s.Inline {
			result = fmt.Sprintf("%s %s", style.Render(icon), s.Label)
		} else {
			result = fmt.Sprintf("%s\n%s", style.Render(icon), s.Label)
		}
	} else {
		result = style.Render(s.Label)
	}

	if s.Details != "" {
		detailStyle := styles.StyleSubtle
		if s.Inline {
			result = fmt.Sprintf("%s %s", result, detailStyle.Render(s.Details))
		} else {
			result = fmt.Sprintf("%s\n%s", result, detailStyle.Render(s.Details))
		}
	}

	return result
}

// getIcon returns the appropriate icon for the status type
func (s *StatusIndicator) getIcon() string {
	switch s.Type {
	case StatusTypeSuccess:
		return styles.IconCheckmark
	case StatusTypeWarning:
		return styles.IconWarning
	case StatusTypeError:
		return styles.IconCross
	case StatusTypeInfo:
		return styles.IconInfo
	case StatusTypePending:
		return "○"
	case StatusTypeRunning:
		return spinnerFrames[s.spinnerFrame]
	default:
		return ""
	}
}

// getStyle returns the appropriate style for the status type
func (s *StatusIndicator) getStyle() lipgloss.Style {
	switch s.Type {
	case StatusTypeSuccess:
		return styles.StyleSuccess
	case StatusTypeWarning:
		return styles.StyleWarning
	case StatusTypeError:
		return styles.StyleError
	case StatusTypeInfo:
		return styles.StyleStatus
	case StatusTypePending:
		return styles.StyleSubtle
	case StatusTypeRunning:
		return styles.StyleStatus
	default:
		return styles.StyleNormal
	}
}

// SetType updates the status type
func (s *StatusIndicator) SetType(statusType StatusType) {
	s.Type = statusType
}

// SetLabel updates the label text
func (s *StatusIndicator) SetLabel(label string) {
	s.Label = label
}

// SetDetails updates the details text
func (s *StatusIndicator) SetDetails(details string) {
	s.Details = details
}

// WithDetails returns the status indicator with details set (for chaining)
func (s *StatusIndicator) WithDetails(details string) *StatusIndicator {
	s.Details = details
	return s
}

// WithoutIcon returns the status indicator with icon hidden (for chaining)
func (s *StatusIndicator) WithoutIcon() *StatusIndicator {
	s.ShowIcon = false
	return s
}

// StatusList displays a list of status indicators
type StatusList struct {
	items []*StatusIndicator
}

// NewStatusList creates a new status list
func NewStatusList() *StatusList {
	return &StatusList{
		items: make([]*StatusIndicator, 0),
	}
}

// Add adds a status indicator to the list
func (l *StatusList) Add(item *StatusIndicator) {
	l.items = append(l.items, item)
}

// AddStatus adds a new status indicator with the given label and type
func (l *StatusList) AddStatus(label string, statusType StatusType) *StatusIndicator {
	item := NewStatusIndicator(label, statusType)
	l.items = append(l.items, item)
	return item
}

// Init implements tea.Model
func (l *StatusList) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, item := range l.items {
		if cmd := item.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model
func (l *StatusList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	for i, item := range l.items {
		newItem, cmd := item.Update(msg)
		if si, ok := newItem.(*StatusIndicator); ok {
			l.items[i] = si
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return l, tea.Batch(cmds...)
}

// View implements tea.Model
func (l *StatusList) View() string {
	var lines []string
	for _, item := range l.items {
		lines = append(lines, item.View())
	}
	return joinLines(lines)
}

// Get returns the status indicator at the given index
func (l *StatusList) Get(index int) *StatusIndicator {
	if index < 0 || index >= len(l.items) {
		return nil
	}
	return l.items[index]
}

// Count returns the number of status indicators
func (l *StatusList) Count() int {
	return len(l.items)
}

// joinLines joins strings with newlines, handling empty strings
func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
