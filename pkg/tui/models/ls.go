// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/monitoring"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/styles"
	"github.com/andri/crook/pkg/tui/views"
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

	// Views
	header          *components.ClusterHeader
	nodesView       *views.NodesView
	deploymentsView *views.DeploymentsView
	osdsView        *views.OSDsView
	podsView        *views.PodsView

	// Data counts (for tab badges)
	nodeCount       int
	deploymentCount int
	osdCount        int
	podCount        int

	// Total counts (unfiltered, for displaying filtered/total format)
	nodeTotalCount       int
	deploymentTotalCount int
	osdTotalCount        int
	podTotalCount        int

	// Cluster state (for OSD view noout flag)
	nooutSet bool

	// Error state
	lastError error

	// Monitor for background updates
	monitor        *monitoring.LsMonitor
	lastUpdateTime time.Time
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

// LsMonitorStartedMsg is sent when the monitor is ready
type LsMonitorStartedMsg struct{}

// LsRefreshTickMsg triggers checking for monitor updates
type LsRefreshTickMsg struct{}

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

	// Create views
	nodesView := views.NewNodesView()
	deploymentsView := views.NewDeploymentsView()
	osdsView := views.NewOSDsView()
	podsView := views.NewPodsView()

	// Apply node filter to pods view if specified
	if cfg.NodeFilter != "" {
		podsView.SetNodeFilter(cfg.NodeFilter)
	}

	return &LsModel{
		config:          cfg,
		activeTab:       showTabs[0],
		tabBar:          components.NewTabBar(tabs),
		cursor:          0,
		header:          components.NewClusterHeader(),
		nodesView:       nodesView,
		deploymentsView: deploymentsView,
		osdsView:        osdsView,
		podsView:        podsView,
	}
}

// Init implements tea.Model
func (m *LsModel) Init() tea.Cmd {
	return tea.Batch(
		m.startMonitorCmd(),
		m.tickCmd(),
	)
}

// tickCmd returns a command that ticks every 100ms to check for monitor updates
func (m *LsModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return LsRefreshTickMsg{}
	})
}

// startMonitorCmd starts the LsMonitor in a goroutine and returns when ready
func (m *LsModel) startMonitorCmd() tea.Cmd {
	return func() tea.Msg {
		cfg := &monitoring.LsMonitorConfig{
			Client:                     m.config.Client,
			Namespace:                  m.config.Config.Kubernetes.RookClusterNamespace,
			Prefixes:                   m.config.Config.DeploymentFilters.Prefixes,
			NodeFilter:                 m.config.NodeFilter,
			NodesRefreshInterval:       time.Duration(m.config.Config.UI.LsRefreshNodesMS) * time.Millisecond,
			DeploymentsRefreshInterval: time.Duration(m.config.Config.UI.LsRefreshDeploymentsMS) * time.Millisecond,
			PodsRefreshInterval:        time.Duration(m.config.Config.UI.LsRefreshPodsMS) * time.Millisecond,
			OSDsRefreshInterval:        time.Duration(m.config.Config.UI.LsRefreshOSDsMS) * time.Millisecond,
			HeaderRefreshInterval:      time.Duration(m.config.Config.UI.LsRefreshHeaderMS) * time.Millisecond,
		}
		m.monitor = monitoring.NewLsMonitor(cfg)
		m.monitor.Start()
		return LsMonitorStartedMsg{}
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
		m.header.SetWidth(msg.Width)
		// Update view sizes (subtract space for header, tabs, status bar)
		viewHeight := msg.Height - 10 // Approximate header + tabs + status
		m.nodesView.SetSize(msg.Width, viewHeight)
		m.deploymentsView.SetSize(msg.Width, viewHeight)
		m.osdsView.SetSize(msg.Width, viewHeight)
		m.podsView.SetSize(msg.Width, viewHeight)

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
		// Apply filter to all views
		m.nodesView.SetFilter(msg.Query)
		m.deploymentsView.SetFilter(msg.Query)
		m.osdsView.SetFilter(msg.Query)
		m.podsView.SetFilter(msg.Query)
		// Update counts after filtering
		m.updateAllCounts()

	case LsRefreshMsg:
		// Manual refresh - force update from monitor's latest data
		if m.monitor != nil {
			latest := m.monitor.GetLatest()
			if latest != nil {
				m.updateFromMonitor(latest)
			}
		}

	case LsMonitorStartedMsg:
		// Monitor is ready, nothing special to do

	case LsRefreshTickMsg:
		// Check for new data from monitor
		if m.monitor != nil {
			latest := m.monitor.GetLatest()
			if latest != nil && latest.UpdateTime.After(m.lastUpdateTime) {
				m.updateFromMonitor(latest)
				m.lastUpdateTime = latest.UpdateTime
			}
		}
		cmds = append(cmds, m.tickCmd())
	}

	return m, tea.Batch(cmds...)
}

// updateFromMonitor updates the model from a monitor update
func (m *LsModel) updateFromMonitor(update *monitoring.LsMonitorUpdate) {
	// Update header
	if update.Header != nil {
		m.header.SetData(update.Header)
		m.nooutSet = update.Header.NooutSet
		m.osdsView.SetNooutFlag(update.Header.NooutSet)
	}

	// Update views with data
	if update.Nodes != nil {
		m.nodesView.SetNodes(update.Nodes)
	}
	if update.Deployments != nil {
		m.deploymentsView.SetDeployments(update.Deployments)
	}
	if update.OSDs != nil {
		m.osdsView.SetOSDs(update.OSDs)
	}
	if update.Pods != nil {
		m.podsView.SetPods(update.Pods)
	}

	// Update counts and badges
	m.updateAllCounts()

	// Store any error
	m.lastError = update.Error
}

// updateAllCounts updates all tab counts and badges
func (m *LsModel) updateAllCounts() {
	// Nodes
	m.nodeCount = m.nodesView.Count()
	m.nodeTotalCount = m.nodesView.TotalCount()
	m.updateBadge(0, m.nodeCount, m.nodeTotalCount)

	// Deployments
	m.deploymentCount = m.deploymentsView.Count()
	m.deploymentTotalCount = m.deploymentsView.TotalCount()
	m.updateBadge(1, m.deploymentCount, m.deploymentTotalCount)

	// OSDs
	m.osdCount = m.osdsView.Count()
	m.osdTotalCount = m.osdsView.TotalCount()
	m.updateBadge(2, m.osdCount, m.osdTotalCount)

	// Pods
	m.podCount = m.podsView.Count()
	m.podTotalCount = m.podsView.TotalCount()
	m.updateBadge(3, m.podCount, m.podTotalCount)
}

// updateBadge updates a tab badge with count information
func (m *LsModel) updateBadge(tabIndex, count, totalCount int) {
	badge := fmt.Sprintf("%d", count)
	if totalCount > 0 && count != totalCount {
		badge = fmt.Sprintf("%d/%d", count, totalCount)
	}
	m.tabBar.SetBadge(tabIndex, badge)
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
		if m.monitor != nil {
			m.monitor.Stop()
		}
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
		m.updateActiveViewCursor(1)
		return nil

	case "k", "up":
		m.updateActiveViewCursor(-1)
		return nil

	case "g":
		m.setActiveViewCursor(0)
		return nil

	case "G":
		// Go to end - actual max will depend on current view
		m.setActiveViewCursor(m.getMaxCursor())
		return nil

	case "r":
		return func() tea.Msg {
			return LsRefreshMsg{Tab: m.activeTab}
		}

	case "enter":
		// Placeholder: will trigger detail view in future
		return nil

	case "ctrl+c":
		if m.monitor != nil {
			m.monitor.Stop()
		}
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

// updateActiveViewCursor moves the cursor in the active view by delta
func (m *LsModel) updateActiveViewCursor(delta int) {
	switch m.activeTab {
	case LsTabNodes:
		newCursor := m.nodesView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.nodesView.Count() {
			m.nodesView.SetCursor(newCursor)
		}
	case LsTabDeployments:
		newCursor := m.deploymentsView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.deploymentsView.Count() {
			m.deploymentsView.SetCursor(newCursor)
		}
	case LsTabOSDs:
		newCursor := m.osdsView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.osdsView.Count() {
			m.osdsView.SetCursor(newCursor)
		}
	case LsTabPods:
		newCursor := m.podsView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.podsView.Count() {
			m.podsView.SetCursor(newCursor)
		}
	}
}

// setActiveViewCursor sets the cursor position in the active view
func (m *LsModel) setActiveViewCursor(pos int) {
	switch m.activeTab {
	case LsTabNodes:
		m.nodesView.SetCursor(pos)
	case LsTabDeployments:
		m.deploymentsView.SetCursor(pos)
	case LsTabOSDs:
		m.osdsView.SetCursor(pos)
	case LsTabPods:
		m.podsView.SetCursor(pos)
	}
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
	header.WriteString("\n")

	// Use the header component for cluster health
	header.WriteString(m.header.View())

	return header.String()
}

// renderActiveView renders the currently active tab's content
func (m *LsModel) renderActiveView() string {
	switch m.activeTab {
	case LsTabNodes:
		return m.nodesView.View()
	case LsTabDeployments:
		return m.deploymentsView.View()
	case LsTabOSDs:
		return m.osdsView.View()
	case LsTabPods:
		return m.podsView.View()
	}
	return ""
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
