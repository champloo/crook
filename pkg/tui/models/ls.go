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

// LsPane represents the available panes in the multi-pane ls view
type LsPane int

const (
	// LsPaneNodes shows cluster nodes
	LsPaneNodes LsPane = iota
	// LsPaneDeployments shows Rook-Ceph deployments (toggleable to pods)
	LsPaneDeployments
	// LsPaneOSDs shows Ceph OSDs
	LsPaneOSDs
)

// String returns the human-readable name for the pane
func (p LsPane) String() string {
	switch p {
	case LsPaneNodes:
		return "Nodes"
	case LsPaneDeployments:
		return "Deployments"
	case LsPaneOSDs:
		return "OSDs"
	default:
		return "Unknown"
	}
}

// LsTab is kept for backwards compatibility with LsModelConfig.ShowTabs
// Deprecated: use LsPane instead
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
	// Deprecated: the new multi-pane layout always shows all 3 panes
	ShowTabs []LsTab
}

// LsModel is the Bubble Tea model for the ls command TUI
type LsModel struct {
	// Configuration
	config LsModelConfig

	// UI state - multi-pane layout
	activePane LsPane
	panes      [3]*components.Pane

	// Legacy cursor field (kept for backwards compatibility with tests)
	cursor int

	// Filter state
	filter       string
	filterActive bool
	helpVisible  bool

	// Terminal dimensions
	width  int
	height int

	// Views
	header              *components.ClusterHeader
	nodesView           *views.NodesView
	deploymentsPodsView *views.DeploymentsPodsView
	osdsView            *views.OSDsView

	// Data counts (for pane badges)
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

	// Legacy fields for backwards compatibility
	tabBar          *components.TabBar
	activeTab       LsTab
	deploymentsView *views.DeploymentsView
	podsView        *views.PodsView
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
	// Create panes
	panes := [3]*components.Pane{
		components.NewPane(components.PaneConfig{Title: "Nodes", ShortcutKey: "1"}),
		components.NewPane(components.PaneConfig{Title: "Deployments", ShortcutKey: "2"}),
		components.NewPane(components.PaneConfig{Title: "OSDs", ShortcutKey: "3"}),
	}

	// Set first pane as active
	panes[0].SetActive(true)

	// Create views
	nodesView := views.NewNodesView()
	deploymentsPodsView := views.NewDeploymentsPodsView()
	osdsView := views.NewOSDsView()

	// Apply node filter to pods view if specified
	if cfg.NodeFilter != "" {
		deploymentsPodsView.SetNodeFilter(cfg.NodeFilter)
	}

	// Create legacy tab bar for backwards compatibility with tests
	showTabs := cfg.ShowTabs
	if len(showTabs) == 0 {
		showTabs = []LsTab{LsTabNodes, LsTabDeployments, LsTabOSDs, LsTabPods}
	}
	tabs := make([]components.Tab, len(showTabs))
	for i, t := range showTabs {
		tabs[i] = components.Tab{
			Title:       t.String(),
			ShortcutKey: fmt.Sprintf("%d", i+1),
		}
	}

	return &LsModel{
		config:              cfg,
		activePane:          LsPaneNodes,
		panes:               panes,
		cursor:              0,
		header:              components.NewClusterHeader(),
		nodesView:           nodesView,
		deploymentsPodsView: deploymentsPodsView,
		osdsView:            osdsView,
		// Legacy fields
		tabBar:          components.NewTabBar(tabs),
		activeTab:       showTabs[0],
		deploymentsView: deploymentsPodsView.GetDeploymentsView(),
		podsView:        deploymentsPodsView.GetPodsView(),
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
		// Helper to ensure non-zero duration with default fallback
		getInterval := func(ms int, defaultMS int) time.Duration {
			if ms <= 0 {
				ms = defaultMS
			}
			return time.Duration(ms) * time.Millisecond
		}

		cfg := &monitoring.LsMonitorConfig{
			Client:                     m.config.Client,
			Namespace:                  m.config.Config.Kubernetes.RookClusterNamespace,
			Prefixes:                   m.config.Config.DeploymentFilters.Prefixes,
			NodeFilter:                 m.config.NodeFilter,
			NodesRefreshInterval:       getInterval(m.config.Config.UI.LsRefreshNodesMS, config.DefaultLsRefreshNodesMS),
			DeploymentsRefreshInterval: getInterval(m.config.Config.UI.LsRefreshDeploymentsMS, config.DefaultLsRefreshDeploymentsMS),
			PodsRefreshInterval:        getInterval(m.config.Config.UI.LsRefreshPodsMS, config.DefaultLsRefreshPodsMS),
			OSDsRefreshInterval:        getInterval(m.config.Config.UI.LsRefreshOSDsMS, config.DefaultLsRefreshOSDsMS),
			HeaderRefreshInterval:      getInterval(m.config.Config.UI.LsRefreshHeaderMS, config.DefaultLsRefreshHeaderMS),
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
		m.header.SetWidth(msg.Width)
		m.tabBar.SetWidth(msg.Width)
		m.updateViewSizes()

	case components.TabSwitchMsg:
		// Legacy tab switch support - map to pane
		newTabBar, _ := m.tabBar.Update(msg)
		if tb, ok := newTabBar.(*components.TabBar); ok {
			m.tabBar = tb
		}
		m.updateActiveTab(msg.Index)
		// Also update pane based on tab index
		if msg.Index < 3 {
			m.setActivePane(LsPane(msg.Index))
		}
		m.cursor = 0

	case LsDataUpdateMsg:
		m.updateDataCount(msg)

	case LsFilterMsg:
		m.filter = msg.Query
		m.cursor = 0
		// Apply filter to all views
		m.nodesView.SetFilter(msg.Query)
		m.deploymentsPodsView.SetFilter(msg.Query)
		m.osdsView.SetFilter(msg.Query)
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

// updateViewSizes updates view dimensions based on the multi-pane layout
func (m *LsModel) updateViewSizes() {
	// Calculate available height for panes
	// Header: ~4 lines, Nav bar: 2 lines, Status bar: 2 lines
	headerHeight := 4
	navBarHeight := 2
	statusBarHeight := 2
	availableHeight := m.height - headerHeight - navBarHeight - statusBarHeight

	// Calculate height distribution: active pane gets 50%, inactive get 25% each
	activeHeight := availableHeight / 2
	inactiveHeight := availableHeight / 4

	// Ensure minimum heights
	if activeHeight < 8 {
		activeHeight = 8
	}
	if inactiveHeight < 4 {
		inactiveHeight = 4
	}

	// Set sizes for each pane and view
	for i, pane := range m.panes {
		height := inactiveHeight
		if LsPane(i) == m.activePane {
			height = activeHeight
		}
		pane.SetSize(m.width, height)
	}

	// Set view sizes (use active pane height for all, they'll be clipped by pane)
	m.nodesView.SetSize(m.width-4, activeHeight-3)
	m.deploymentsPodsView.SetSize(m.width-4, activeHeight-3)
	m.osdsView.SetSize(m.width-4, activeHeight-3)
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
		m.deploymentsPodsView.SetDeployments(update.Deployments)
	}
	if update.OSDs != nil {
		m.osdsView.SetOSDs(update.OSDs)
	}
	if update.Pods != nil {
		m.deploymentsPodsView.SetPods(update.Pods)
	}

	// Update counts and badges
	m.updateAllCounts()

	// Store any error
	m.lastError = update.Error
}

// updateAllCounts updates all pane counts and badges
func (m *LsModel) updateAllCounts() {
	// Nodes
	m.nodeCount = m.nodesView.Count()
	m.nodeTotalCount = m.nodesView.TotalCount()
	m.updatePaneBadge(LsPaneNodes, m.nodeCount, m.nodeTotalCount)
	m.updateBadge(0, m.nodeCount, m.nodeTotalCount)

	// Deployments/Pods - show count from currently active sub-view
	m.deploymentCount = m.deploymentsPodsView.DeploymentsCount()
	m.deploymentTotalCount = m.deploymentsPodsView.DeploymentsTotalCount()
	m.podCount = m.deploymentsPodsView.PodsCount()
	m.podTotalCount = m.deploymentsPodsView.PodsTotalCount()

	// Update deployments pane badge based on which view is showing
	if m.deploymentsPodsView.IsShowingPods() {
		m.updatePaneBadge(LsPaneDeployments, m.podCount, m.podTotalCount)
		m.panes[LsPaneDeployments].SetTitle("Pods")
	} else {
		m.updatePaneBadge(LsPaneDeployments, m.deploymentCount, m.deploymentTotalCount)
		m.panes[LsPaneDeployments].SetTitle("Deployments")
	}
	m.updateBadge(1, m.deploymentCount, m.deploymentTotalCount)
	m.updateBadge(3, m.podCount, m.podTotalCount)

	// OSDs
	m.osdCount = m.osdsView.Count()
	m.osdTotalCount = m.osdsView.TotalCount()
	m.updatePaneBadge(LsPaneOSDs, m.osdCount, m.osdTotalCount)
	m.updateBadge(2, m.osdCount, m.osdTotalCount)
}

// updatePaneBadge updates a pane badge with count information
func (m *LsModel) updatePaneBadge(pane LsPane, count, totalCount int) {
	badge := fmt.Sprintf("%d", count)
	if totalCount > 0 && count != totalCount {
		badge = fmt.Sprintf("%d/%d", count, totalCount)
	}
	m.panes[pane].SetBadge(badge)
}

// updateBadge updates a tab badge with count information (legacy)
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

	case "tab":
		m.nextPane()
		return nil

	case "shift+tab":
		m.prevPane()
		return nil

	case "1":
		m.setActivePane(LsPaneNodes)
		// Also update legacy tab bar
		_, cmd := m.tabBar.Update(msg)
		return cmd

	case "2":
		m.setActivePane(LsPaneDeployments)
		_, cmd := m.tabBar.Update(msg)
		return cmd

	case "3":
		m.setActivePane(LsPaneOSDs)
		_, cmd := m.tabBar.Update(msg)
		return cmd

	case "4", "5", "6", "7", "8", "9":
		// Delegate to tab bar for backward compatibility
		_, cmd := m.tabBar.Update(msg)
		return cmd

	case "[":
		// Toggle to deployments view in middle pane
		if m.activePane == LsPaneDeployments {
			m.deploymentsPodsView.ShowDeployments()
			m.panes[LsPaneDeployments].SetTitle("Deployments")
			m.updateAllCounts()
		}
		return nil

	case "]":
		// Toggle to pods view in middle pane
		if m.activePane == LsPaneDeployments {
			m.deploymentsPodsView.ShowPods()
			m.panes[LsPaneDeployments].SetTitle("Pods")
			m.updateAllCounts()
		}
		return nil

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

// nextPane cycles to the next pane
func (m *LsModel) nextPane() {
	newPane := (m.activePane + 1) % 3
	m.setActivePane(newPane)
}

// prevPane cycles to the previous pane
func (m *LsModel) prevPane() {
	newPane := m.activePane - 1
	if newPane < 0 {
		newPane = 2
	}
	m.setActivePane(newPane)
}

// setActivePane changes the active pane
func (m *LsModel) setActivePane(pane LsPane) {
	// Deactivate all panes
	for _, p := range m.panes {
		p.SetActive(false)
	}

	// Activate the selected pane
	m.activePane = pane
	m.panes[pane].SetActive(true)

	// Update legacy activeTab
	switch pane {
	case LsPaneNodes:
		m.activeTab = LsTabNodes
	case LsPaneDeployments:
		if m.deploymentsPodsView.IsShowingPods() {
			m.activeTab = LsTabPods
		} else {
			m.activeTab = LsTabDeployments
		}
	case LsPaneOSDs:
		m.activeTab = LsTabOSDs
	}

	// Update view sizes for new active pane
	m.updateViewSizes()
}

// updateActiveTab updates the active tab based on the index (legacy)
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
		m.panes[LsPaneNodes].SetBadge(badge)
	case LsTabDeployments:
		m.deploymentCount = msg.Count
		m.deploymentTotalCount = msg.TotalCount
		m.tabBar.SetBadge(1, badge)
		if !m.deploymentsPodsView.IsShowingPods() {
			m.panes[LsPaneDeployments].SetBadge(badge)
		}
	case LsTabOSDs:
		m.osdCount = msg.Count
		m.osdTotalCount = msg.TotalCount
		m.tabBar.SetBadge(2, badge)
		m.panes[LsPaneOSDs].SetBadge(badge)
	case LsTabPods:
		m.podCount = msg.Count
		m.podTotalCount = msg.TotalCount
		m.tabBar.SetBadge(3, badge)
		if m.deploymentsPodsView.IsShowingPods() {
			m.panes[LsPaneDeployments].SetBadge(badge)
		}
	}
}

// getMaxCursor returns the maximum cursor position for the current pane
func (m *LsModel) getMaxCursor() int {
	switch m.activePane {
	case LsPaneNodes:
		return max(0, m.nodeCount-1)
	case LsPaneDeployments:
		return max(0, m.deploymentsPodsView.Count()-1)
	case LsPaneOSDs:
		return max(0, m.osdCount-1)
	}
	return 0
}

// updateActiveViewCursor moves the cursor in the active view by delta
func (m *LsModel) updateActiveViewCursor(delta int) {
	switch m.activePane {
	case LsPaneNodes:
		newCursor := m.nodesView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.nodesView.Count() {
			m.nodesView.SetCursor(newCursor)
		}
	case LsPaneDeployments:
		newCursor := m.deploymentsPodsView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.deploymentsPodsView.Count() {
			m.deploymentsPodsView.SetCursor(newCursor)
		}
	case LsPaneOSDs:
		newCursor := m.osdsView.GetCursor() + delta
		if newCursor >= 0 && newCursor < m.osdsView.Count() {
			m.osdsView.SetCursor(newCursor)
		}
	}
}

// setActiveViewCursor sets the cursor position in the active view
func (m *LsModel) setActiveViewCursor(pos int) {
	switch m.activePane {
	case LsPaneNodes:
		m.nodesView.SetCursor(pos)
	case LsPaneDeployments:
		m.deploymentsPodsView.SetCursor(pos)
	case LsPaneOSDs:
		m.osdsView.SetCursor(pos)
	}
}

// View implements tea.Model
func (m *LsModel) View() string {
	var b strings.Builder

	// Help overlay takes precedence
	if m.helpVisible {
		return m.renderHelp()
	}

	// Header with cluster summary
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Render all three panes (title is now in the border)
	b.WriteString(m.renderAllPanes())
	b.WriteString("\n")

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

	// Use the header component for cluster health
	header.WriteString(m.header.View())

	return header.String()
}

// renderAllPanes renders all three panes stacked vertically
func (m *LsModel) renderAllPanes() string {
	var b strings.Builder

	// Calculate heights
	headerHeight := 4
	statusBarHeight := 2
	filterBarHeight := 0
	if m.filterActive {
		filterBarHeight = 2
	}
	availableHeight := m.height - headerHeight - statusBarHeight - filterBarHeight

	// Height distribution: active pane gets 50%, inactive get 25% each
	activeHeight := availableHeight / 2
	inactiveHeight := availableHeight / 4

	// Ensure minimum heights
	if activeHeight < 8 {
		activeHeight = 8
	}
	if inactiveHeight < 4 {
		inactiveHeight = 4
	}

	// Render each pane
	for i, pane := range m.panes {
		height := inactiveHeight
		if LsPane(i) == m.activePane {
			height = activeHeight
		}

		// Get content for this pane
		content := m.getPaneContent(LsPane(i))

		// Update pane size and render
		pane.SetSize(m.width, height)
		b.WriteString(pane.View(content))

		if i < len(m.panes)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// getPaneContent returns the view content for a specific pane
func (m *LsModel) getPaneContent(pane LsPane) string {
	switch pane {
	case LsPaneNodes:
		return m.nodesView.View()
	case LsPaneDeployments:
		return m.deploymentsPodsView.View()
	case LsPaneOSDs:
		return m.osdsView.View()
	}
	return ""
}

// renderFilterBar renders the filter input bar
func (m *LsModel) renderFilterBar() string {
	prompt := styles.StyleStatus.Render("/")
	input := styles.StyleNormal.Render(m.filter + "\u2588")
	return prompt + input
}

// renderStatusBar renders the bottom status bar with help hints
func (m *LsModel) renderStatusBar() string {
	var hints []string

	if m.filterActive {
		hints = append(hints, "Enter: apply", "Esc: cancel")
	} else {
		hints = append(hints, "Tab/1-3: pane", "j/k: navigate")
		if m.activePane == LsPaneDeployments {
			hints = append(hints, "[/]: deps/pods")
		}
		hints = append(hints, "/: filter", "r: refresh", "?: help", "q: quit")
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
│    Tab, 1-3    Switch panes             │
│    [, ]        Toggle deploymentss/pods │
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
	m.updateViewSizes()
}

// GetActiveTab returns the currently active tab (legacy)
func (m *LsModel) GetActiveTab() LsTab {
	return m.activeTab
}

// GetActivePane returns the currently active pane
func (m *LsModel) GetActivePane() LsPane {
	return m.activePane
}

// GetCursor returns the current cursor position (legacy)
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

// IsShowingPods returns true if the deployments pane is showing pods
func (m *LsModel) IsShowingPods() bool {
	return m.deploymentsPodsView.IsShowingPods()
}
