package components

import (
	"strings"

	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FilterBar provides an interactive filter/search input
type FilterBar struct {
	// query is the current filter text
	query string

	// cursorPos is the cursor position in the query
	cursorPos int

	// active indicates if the filter bar is accepting input
	active bool

	// width is the available width for rendering
	width int

	// applied indicates a filter has been applied (even if not active)
	applied bool
}

// NewFilterBar creates a new filter bar
func NewFilterBar() *FilterBar {
	return &FilterBar{}
}

// FilterAppliedMsg is sent when filter is applied (Enter pressed)
type FilterAppliedMsg struct {
	Query string
}

// FilterClearedMsg is sent when filter is cleared (Esc pressed)
type FilterClearedMsg struct{}

// FilterChangedMsg is sent when the filter query changes during input
type FilterChangedMsg struct {
	Query string
}

// FilterModeExitMsg is sent when exiting filter mode
type FilterModeExitMsg struct{}

// Init implements tea.Model
func (f *FilterBar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (f *FilterBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !f.active {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return f.handleKey(msg)

	case tea.WindowSizeMsg:
		f.width = msg.Width
	}

	return f, nil
}

// handleKey processes key events when filter is active
func (f *FilterBar) handleKey(msg tea.KeyMsg) (*FilterBar, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // Only handling specific filter-mode keys
	case tea.KeyEsc:
		// Clear filter and exit
		f.query = ""
		f.cursorPos = 0
		f.active = false
		f.applied = false
		return f, tea.Batch(
			func() tea.Msg { return FilterClearedMsg{} },
			func() tea.Msg { return FilterModeExitMsg{} },
		)

	case tea.KeyEnter:
		// Apply filter and exit input mode
		f.active = false
		f.applied = f.query != ""
		return f, tea.Batch(
			func() tea.Msg { return FilterAppliedMsg{Query: f.query} },
			func() tea.Msg { return FilterModeExitMsg{} },
		)

	case tea.KeyBackspace:
		if f.cursorPos > 0 && len(f.query) > 0 {
			// Delete character before cursor
			if f.cursorPos == len(f.query) {
				f.query = f.query[:len(f.query)-1]
			} else {
				f.query = f.query[:f.cursorPos-1] + f.query[f.cursorPos:]
			}
			f.cursorPos--
			return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }
		}

	case tea.KeyDelete:
		if f.cursorPos < len(f.query) {
			// Delete character at cursor
			f.query = f.query[:f.cursorPos] + f.query[f.cursorPos+1:]
			return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }
		}

	case tea.KeyLeft:
		if f.cursorPos > 0 {
			f.cursorPos--
		}

	case tea.KeyRight:
		if f.cursorPos < len(f.query) {
			f.cursorPos++
		}

	case tea.KeyHome, tea.KeyCtrlA:
		f.cursorPos = 0

	case tea.KeyEnd, tea.KeyCtrlE:
		f.cursorPos = len(f.query)

	case tea.KeyCtrlU:
		// Clear query but stay in filter mode
		f.query = ""
		f.cursorPos = 0
		return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }

	case tea.KeyCtrlW:
		// Delete word before cursor
		if f.cursorPos > 0 {
			// Find start of previous word
			i := f.cursorPos - 1
			// Skip trailing spaces
			for i > 0 && f.query[i] == ' ' {
				i--
			}
			// Find word boundary
			for i > 0 && f.query[i-1] != ' ' {
				i--
			}
			f.query = f.query[:i] + f.query[f.cursorPos:]
			f.cursorPos = i
			return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }
		}

	case tea.KeyRunes:
		// Insert characters at cursor
		chars := string(msg.Runes)
		if f.cursorPos == len(f.query) {
			f.query += chars
		} else {
			f.query = f.query[:f.cursorPos] + chars + f.query[f.cursorPos:]
		}
		f.cursorPos += len(chars)
		return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }

	case tea.KeySpace:
		// Handle space character
		if f.cursorPos == len(f.query) {
			f.query += " "
		} else {
			f.query = f.query[:f.cursorPos] + " " + f.query[f.cursorPos:]
		}
		f.cursorPos++
		return f, func() tea.Msg { return FilterChangedMsg{Query: f.query} }
	}

	return f, nil
}

// View renders the filter bar
func (f *FilterBar) View() string {
	if !f.active {
		return ""
	}

	return f.renderInputBar()
}

// ViewStatus renders the status when filter is applied but not active
func (f *FilterBar) ViewStatus() string {
	if !f.applied || f.query == "" {
		return ""
	}

	return styles.StyleStatus.Render("Filter: \"" + f.query + "\"")
}

// renderInputBar renders the active input bar
func (f *FilterBar) renderInputBar() string {
	promptStyle := lipgloss.NewStyle().
		Foreground(styles.ColorPrimary).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(styles.ColorHighlight)

	// Build input with cursor
	var input string
	if f.cursorPos == len(f.query) {
		input = f.query + "█"
	} else {
		input = f.query[:f.cursorPos] + "█" + f.query[f.cursorPos:]
	}

	prompt := promptStyle.Render("/")
	text := inputStyle.Render(" " + input)

	return prompt + text
}

// Activate enables filter input mode
func (f *FilterBar) Activate() {
	f.active = true
	f.cursorPos = len(f.query)
}

// Deactivate disables filter input mode
func (f *FilterBar) Deactivate() {
	f.active = false
}

// IsActive returns whether the filter bar is accepting input
func (f *FilterBar) IsActive() bool {
	return f.active
}

// Query returns the current filter query
func (f *FilterBar) Query() string {
	return f.query
}

// SetQuery sets the filter query
func (f *FilterBar) SetQuery(query string) {
	f.query = query
	f.cursorPos = len(query)
	f.applied = query != ""
}

// Clear clears the filter
func (f *FilterBar) Clear() {
	f.query = ""
	f.cursorPos = 0
	f.applied = false
}

// HasFilter returns true if a filter is applied
func (f *FilterBar) HasFilter() bool {
	return f.applied && f.query != ""
}

// SetWidth sets the available width
func (f *FilterBar) SetWidth(width int) {
	f.width = width
}

// FilterStatusText returns a status string showing filter state
func (f *FilterBar) FilterStatusText(filtered, total int) string {
	if !f.applied || f.query == "" {
		return ""
	}

	return styles.StyleStatus.Render(
		"Filtered: \"" + f.query + "\" (" + intToStr(filtered) + "/" + intToStr(total) + ")",
	)
}

// intToStr converts an int to a string without importing strconv
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}

	negative := false
	if n < 0 {
		negative = true
		n = -n
	}

	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// Filterable is the interface for views that support filtering
type Filterable interface {
	// SetFilter sets the filter query
	SetFilter(query string)

	// Count returns the filtered count
	Count() int

	// TotalCount returns the total unfiltered count
	TotalCount() int
}

// MatchesFilter checks if a string matches the filter query (case-insensitive)
func MatchesFilter(s, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(s), strings.ToLower(query))
}
