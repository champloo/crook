// Package components provides reusable TUI components.
package components

import (
	"strings"

	"github.com/andri/crook/pkg/tui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Tab represents a single tab in the tab bar
type Tab struct {
	// Title is the display text for the tab
	Title string

	// ShortcutKey is the key that activates this tab (e.g., "1", "2")
	ShortcutKey string

	// Badge shows a count or indicator (optional)
	Badge string
}

// TabBar is a horizontal tab navigation component
type TabBar struct {
	// Tabs is the list of tabs
	Tabs []Tab

	// ActiveTab is the index of the currently active tab
	ActiveTab int

	// Width is the total width of the tab bar
	Width int

	// Styles
	activeStyle   lipgloss.Style
	inactiveStyle lipgloss.Style
	badgeStyle    lipgloss.Style
}

// TabSwitchMsg is sent when a tab is switched
type TabSwitchMsg struct {
	Index int
}

// NewTabBar creates a new tab bar with the given tabs
func NewTabBar(tabs []Tab) *TabBar {
	return &TabBar{
		Tabs:      tabs,
		ActiveTab: 0,
		activeStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimaryFg).
			Background(styles.ColorPrimary).
			Padding(0, 2),
		inactiveStyle: lipgloss.NewStyle().
			Foreground(styles.ColorSubtle).
			Padding(0, 2),
		badgeStyle: lipgloss.NewStyle().
			Foreground(styles.ColorWarning).
			Bold(true),
	}
}

// Init implements tea.Model
func (t *TabBar) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (t *TabBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			return t, t.nextTab()
		case "shift+tab":
			return t, t.prevTab()
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			index := int(msg.String()[0] - '1')
			if index < len(t.Tabs) {
				return t, t.switchToTab(index)
			}
		}
	case TabSwitchMsg:
		if msg.Index >= 0 && msg.Index < len(t.Tabs) {
			t.ActiveTab = msg.Index
		}
	}
	return t, nil
}

// View implements tea.Model
func (t *TabBar) View() tea.View {
	return tea.NewView(t.Render())
}

// Render returns the string representation for composition
func (t *TabBar) Render() string {
	if len(t.Tabs) == 0 {
		return ""
	}

	var tabViews []string

	for i, tab := range t.Tabs {
		var tabText string

		// Build tab text with shortcut hint
		if tab.ShortcutKey != "" {
			tabText = tab.ShortcutKey + ":" + tab.Title
		} else {
			tabText = tab.Title
		}

		// Add badge if present
		if tab.Badge != "" {
			tabText += " " + t.badgeStyle.Render(tab.Badge)
		}

		// Apply appropriate style
		if i == t.ActiveTab {
			tabViews = append(tabViews, t.activeStyle.Render(tabText))
		} else {
			tabViews = append(tabViews, t.inactiveStyle.Render(tabText))
		}
	}

	return strings.Join(tabViews, " ")
}

// nextTab advances to the next tab
func (t *TabBar) nextTab() tea.Cmd {
	next := (t.ActiveTab + 1) % len(t.Tabs)
	return func() tea.Msg {
		return TabSwitchMsg{Index: next}
	}
}

// prevTab goes to the previous tab
func (t *TabBar) prevTab() tea.Cmd {
	prev := t.ActiveTab - 1
	if prev < 0 {
		prev = len(t.Tabs) - 1
	}
	return func() tea.Msg {
		return TabSwitchMsg{Index: prev}
	}
}

// switchToTab switches to a specific tab index
func (t *TabBar) switchToTab(index int) tea.Cmd {
	return func() tea.Msg {
		return TabSwitchMsg{Index: index}
	}
}

// SetActiveTab sets the active tab index
func (t *TabBar) SetActiveTab(index int) {
	if index >= 0 && index < len(t.Tabs) {
		t.ActiveTab = index
	}
}

// GetActiveTab returns the current active tab index
func (t *TabBar) GetActiveTab() int {
	return t.ActiveTab
}

// SetBadge sets the badge for a specific tab
func (t *TabBar) SetBadge(index int, badge string) {
	if index >= 0 && index < len(t.Tabs) {
		t.Tabs[index].Badge = badge
	}
}

// SetWidth sets the tab bar width
func (t *TabBar) SetWidth(width int) {
	t.Width = width
}

// TabCount returns the number of tabs
func (t *TabBar) TabCount() int {
	return len(t.Tabs)
}
