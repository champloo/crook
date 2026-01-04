// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// LsTab represents the available tabs in the ls view
type LsTab int

const (
	// LsTabNodes shows cluster nodes
	LsTabNodes LsTab = iota
	// LsTabDeployments shows Rook-Ceph deployments
	LsTabDeployments
	// LsTabOSDs shows Ceph OSDs
	LsTabOSDs
	// LsTabPods shows Rook-Ceph pods
	LsTabPods
)

// String returns the human-readable name for the tab
func (t LsTab) String() string {
	switch t {
	case LsTabNodes:
		return "Nodes"
	case LsTabDeployments:
		return "Deployments"
	case LsTabOSDs:
		return "OSDs"
	case LsTabPods:
		return "Pods"
	default:
		return "Unknown"
	}
}

// LsModelConfig holds configuration for the ls model
type LsModelConfig struct {
	// NodeFilter optionally filters resources to a specific node
	NodeFilter string

	// Config is the application configuration
	Config config.Config

	// Client is the Kubernetes client
	Client *k8s.Client

	// Context for cancellation
	Context context.Context

	// ShowTabs specifies which tabs to display (nil = all)
	ShowTabs []LsTab
}

// LsModel is the Bubble Tea model for the ls command TUI
type LsModel struct {
	// Configuration
	config LsModelConfig

	// UI state
	activeTab    LsTab
	tabBar       *components.TabBar
	cursor       int
	filter       string
	filterActive bool
	helpVisible  bool

	// Terminal dimensions
	width  int
	height int

	// Data (placeholders for child views to populate)
	nodeCount       int
	deploymentCount int
	osdCount        int
	podCount        int

	// Total counts (unfiltered, for displaying filtered/total format)
	nodeTotalCount       int
	deploymentTotalCount int
	osdTotalCount        int
	podTotalCount        int

	// Error state
	lastError error //nolint:unused // Reserved for future error handling in child views
}

// LsDataUpdateMsg is sent when data is updated
type LsDataUpdateMsg struct {
	Tab        LsTab
	Count      int
	TotalCount int // TotalCount is the unfiltered count (for displaying X/Y format)
}

// LsFilterMsg is sent when the filter changes
type LsFilterMsg struct {
	Query string
}

// LsRefreshMsg triggers a data refresh
type LsRefreshMsg struct {
	Tab LsTab
}

// NewLsModel creates a new ls model
func NewLsModel(cfg LsModelConfig) *LsModel {
	// Determine which tabs to show
	showTabs := cfg.ShowTabs
	if len(showTabs) == 0 {
		showTabs = []LsTab{LsTabNodes, LsTabDeployments, LsTabOSDs, LsTabPods}
	}

	// Create tab definitions
	tabs := make([]components.Tab, len(showTabs))
	for i, t := range showTabs {
		tabs[i] = components.Tab{
			Title:       t.String(),
			ShortcutKey: fmt.Sprintf("%d", i+1),
		}
	}

	return &LsModel{
		config:    cfg,
		activeTab: showTabs[0],
		tabBar:    components.NewTabBar(tabs),
		cursor:    0,
	}
}

// Init implements tea.Model
func (m *LsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchInitialDataCmd(),
	)
}

// fetchInitialDataCmd fetches initial data for all tabs
func (m *LsModel) fetchInitialDataCmd() tea.Cmd {
	return func() tea.Msg {
		// Placeholder: actual data fetching will be implemented in child views
		// For now, return a refresh message to trigger the active tab
		return LsRefreshMsg{Tab: m.activeTab}
	}
}

// Update implements tea.Model
func (m *LsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tabBar.SetWidth(msg.Width)

	case components.TabSwitchMsg:
		// Update tab bar
		newTabBar, _ := m.tabBar.Update(msg)
		if tb, ok := newTabBar.(*components.TabBar); ok {
			m.tabBar = tb
		}
		// Update active tab based on the tab bar's show tabs
		m.updateActiveTab(msg.Index)
		m.cursor = 0 // Reset cursor on tab switch

	case LsDataUpdateMsg:
		m.updateDataCount(msg)

	case LsFilterMsg:
		m.filter = msg.Query
		m.cursor = 0 // Reset cursor on filter change

	case LsRefreshMsg:
		// Placeholder: refresh will trigger data fetching in child views
		// For now, just acknowledge
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input
func (m *LsModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// If filter mode is active, handle filter input
	if m.filterActive {
		return m.handleFilterInput(msg)
	}

	// If help is visible, any key closes it
	if m.helpVisible {
		m.helpVisible = false
		return nil
	}

	key := msg.String()

	switch key {
	case "q", "esc":
		return tea.Quit

	case "?":
		m.helpVisible = true
		return nil

	case "/":
		m.filterActive = true
		return nil

	case "tab", "shift+tab", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Delegate to tab bar
		_, cmd := m.tabBar.Update(msg)
		return cmd

	case "j", "down":
		m.cursor++
		return nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return nil

	case "g":
		m.cursor = 0
		return nil

	case "G":
		// Go to end - actual max will depend on current view
		m.cursor = m.getMaxCursor()
		return nil

	case "r":
		return func() tea.Msg {
			return LsRefreshMsg{Tab: m.activeTab}
		}

	case "enter":
		// Placeholder: will trigger detail view in future
		return nil

	case "ctrl+c":
		return tea.Quit
	}

	return nil
}

// handleFilterInput handles input during filter mode
func (m *LsModel) handleFilterInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type { //nolint:exhaustive // Only handling specific filter-mode keys
	case tea.KeyEsc:
		m.filterActive = false
		m.filter = ""
		return nil

	case tea.KeyEnter:
		m.filterActive = false
		return func() tea.Msg {
			return LsFilterMsg{Query: m.filter}
		}

	case tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
		return nil

	case tea.KeyRunes:
		m.filter += string(msg.Runes)
		return nil
	}

	return nil
}

// updateActiveTab updates the active tab based on the index
func (m *LsModel) updateActiveTab(index int) {
	showTabs := m.config.ShowTabs
	if len(showTabs) == 0 {
		showTabs = []LsTab{LsTabNodes, LsTabDeployments, LsTabOSDs, LsTabPods}
	}

	if index >= 0 && index < len(showTabs) {
		m.activeTab = showTabs[index]
	}
}

// updateDataCount updates the count for a specific tab
func (m *LsModel) updateDataCount(msg LsDataUpdateMsg) {
	badge := fmt.Sprintf("%d", msg.Count)

	// Show filtered/total format if filtering and counts differ
	if msg.TotalCount > 0 && msg.Count != msg.TotalCount {
		badge = fmt.Sprintf("%d/%d", msg.Count, msg.TotalCount)
	}

	switch msg.Tab {
	case LsTabNodes:
		m.nodeCount = msg.Count
		m.nodeTotalCount = msg.TotalCount
		m.tabBar.SetBadge(0, badge)
	case LsTabDeployments:
		m.deploymentCount = msg.Count
		m.deploymentTotalCount = msg.TotalCount
		m.tabBar.SetBadge(1, badge)
	case LsTabOSDs:
		m.osdCount = msg.Count
		m.osdTotalCount = msg.TotalCount
		m.tabBar.SetBadge(2, badge)
	case LsTabPods:
		m.podCount = msg.Count
		m.podTotalCount = msg.TotalCount
		m.tabBar.SetBadge(3, badge)
	}
}

// getMaxCursor returns the maximum cursor position for the current tab
func (m *LsModel) getMaxCursor() int {
	switch m.activeTab {
	case LsTabNodes:
		return max(0, m.nodeCount-1)
	case LsTabDeployments:
		return max(0, m.deploymentCount-1)
	case LsTabOSDs:
		return max(0, m.osdCount-1)
	case LsTabPods:
		return max(0, m.podCount-1)
	}
	return 0
}

// View implements tea.Model
func (m *LsModel) View() string {
	var b strings.Builder

	// Help overlay takes precedence
	if m.helpVisible {
		return m.renderHelp()
	}

	// Header with cluster summary placeholder
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Tab bar
	b.WriteString(m.tabBar.View())
	b.WriteString("\n\n")

	// Active view content (placeholder)
	b.WriteString(m.renderActiveView())
	b.WriteString("\n\n")

	// Filter bar (if active)
	if m.filterActive {
		b.WriteString(m.renderFilterBar())
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderHeader renders the cluster summary header
func (m *LsModel) renderHeader() string {
	var header strings.Builder

	title := "crook ls"
	if m.config.NodeFilter != "" {
		title += fmt.Sprintf(" (node: %s)", m.config.NodeFilter)
	}

	header.WriteString(styles.StyleHeading.Render(title))

	// Placeholder for cluster health - will be populated by header component
	header.WriteString("\n")
	header.WriteString(styles.StyleSubtle.Render("Ceph: loading..."))

	return header.String()
}

// renderActiveView renders the currently active tab's content
func (m *LsModel) renderActiveView() string {
	// Placeholder - actual views will be implemented in crook-3qm.4-7
	var content string

	switch m.activeTab {
	case LsTabNodes:
		content = fmt.Sprintf("Nodes view (placeholder) - %d nodes", m.nodeCount)
	case LsTabDeployments:
		content = fmt.Sprintf("Deployments view (placeholder) - %d deployments", m.deploymentCount)
	case LsTabOSDs:
		content = fmt.Sprintf("OSDs view (placeholder) - %d OSDs", m.osdCount)
	case LsTabPods:
		content = fmt.Sprintf("Pods view (placeholder) - %d pods", m.podCount)
	}

	if m.filter != "" {
		content += fmt.Sprintf("\nFilter: %q", m.filter)
	}

	return styles.StyleNormal.Render(content)
}

// renderFilterBar renders the filter input bar
func (m *LsModel) renderFilterBar() string {
	prompt := styles.StyleStatus.Render("/")
	input := styles.StyleNormal.Render(m.filter + "█")
	return prompt + input
}

// renderStatusBar renders the bottom status bar with help hints
func (m *LsModel) renderStatusBar() string {
	var hints []string

	if m.filterActive {
		hints = append(hints, "Enter: apply", "Esc: cancel")
	} else {
		hints = append(hints, "Tab: switch", "j/k: navigate", "/: filter", "r: refresh", "?: help", "q: quit")
	}

	return styles.StyleSubtle.Render(strings.Join(hints, "  "))
}

// renderHelp renders the help overlay
func (m *LsModel) renderHelp() string {
	help := `
╭─────────────────────────────────────────╮
│             crook ls Help               │
├─────────────────────────────────────────┤
│  Navigation                             │
│    Tab, 1-4    Switch tabs              │
│    j/k, ↑/↓    Move cursor              │
│    g/G         Go to top/bottom         │
│    Enter       View details             │
│                                         │
│  Actions                                │
│    r           Refresh data             │
│    /           Enter filter mode        │
│                                         │
│  Filter Mode                            │
│    Enter       Apply filter             │
│    Esc         Cancel filter            │
│                                         │
│  General                                │
│    ?           Toggle this help         │
│    q, Esc      Quit                     │
╰─────────────────────────────────────────╯

Press any key to close
`
	return styles.StyleBox.Render(help)
}

// SetSize implements SubModel
func (m *LsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.tabBar.SetWidth(width)
}

// GetActiveTab returns the currently active tab
func (m *LsModel) GetActiveTab() LsTab {
	return m.activeTab
}

// GetCursor returns the current cursor position
func (m *LsModel) GetCursor() int {
	return m.cursor
}

// GetFilter returns the current filter string
func (m *LsModel) GetFilter() string {
	return m.filter
}

// IsFilterActive returns whether filter mode is active
func (m *LsModel) IsFilterActive() bool {
	return m.filterActive
}

// IsHelpVisible returns whether help overlay is visible
func (m *LsModel) IsHelpVisible() bool {
	return m.helpVisible
}

// GetNodeFilter returns the current node filter
func (m *LsModel) GetNodeFilter() string {
	return m.config.NodeFilter
}

// HasNodeFilter returns true if a node filter is active
func (m *LsModel) HasNodeFilter() bool {
	return m.config.NodeFilter != ""
}
