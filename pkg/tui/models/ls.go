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
	"github.com/andri/crook/pkg/tui/format"
	"github.com/andri/crook/pkg/tui/styles"
	"github.com/andri/crook/pkg/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// Help overlay state
	helpVisible bool

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

	// Cluster state (for OSD view noout flag)
	nooutSet bool

	// Error state
	lastError error

	// Maintenance pane state
	maintenanceFlow     sizedModel
	pendingReselectNode string
	maintenancePane     *components.Pane

	// Monitor for background updates
	monitor        *monitoring.LsMonitor
	lastUpdateTime time.Time

	// Legacy fields for backwards compatibility
	tabBar          *components.TabBar
	activeTab       LsTab
	deploymentsView *views.DeploymentsView
	podsView        *views.PodsView
}

type sizedModel interface {
	tea.Model
	SetSize(width, height int)
}

func (m *LsModel) handleHelpKeyMsg(msg tea.KeyMsg) (bool, tea.Cmd) {
	key := msg.String()

	// When help is visible, consume key presses so underlying models don't receive input.
	if m.helpVisible {
		switch key {
		case "esc", "?":
			m.helpVisible = false
			return true, nil
		case "ctrl+c":
			if m.monitor != nil {
				m.monitor.Stop()
			}
			return true, tea.Quit
		default:
			return true, nil
		}
	}

	if key == "?" {
		m.helpVisible = true
		return true, nil
	}

	return false, nil
}

const (
	paneContentHorizontalPadding = 4 // 2 borders + 2 padding (see components.Pane)
	paneContentVerticalPadding   = 3 // pane borders plus typical view chrome
)

func innerViewSize(paneWidth, paneHeight int) (int, int) {
	width := max(paneWidth-paneContentHorizontalPadding, 1)
	height := max(paneHeight-paneContentVerticalPadding, 1)
	return width, height
}

type lsLayout struct {
	nodesWidth        int
	maintenanceWidth  int
	nodesHeight       int
	deploymentsHeight int
	osdsHeight        int

	nodesInnerWidth        int
	nodesInnerHeight       int
	deploymentsInnerWidth  int
	deploymentsInnerHeight int
	osdsInnerWidth         int
	osdsInnerHeight        int
	maintenanceInnerWidth  int
	maintenanceInnerHeight int

	maintenanceActive bool
}

// LsDataUpdateMsg is sent when data is updated
type LsDataUpdateMsg struct {
	Tab   LsTab
	Count int
}

// LsRefreshMsg triggers a data refresh
type LsRefreshMsg struct {
	Tab LsTab
}

// LsMonitorStartedMsg is sent when the monitor is ready
type LsMonitorStartedMsg struct {
	Monitor *monitoring.LsMonitor
}

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
	maintenancePane := components.NewPane(components.PaneConfig{Title: "Node Maintenance", ShortcutKey: ""})

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
		maintenancePane:     maintenancePane,
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
			Context:                    m.config.Context,
			Client:                     m.config.Client,
			Namespace:                  m.config.Config.Namespace,
			NodeFilter:                 m.config.NodeFilter,
			NodesRefreshInterval:       getInterval(m.config.Config.UI.LsRefreshNodesMS, config.DefaultLsRefreshNodesMS),
			DeploymentsRefreshInterval: getInterval(m.config.Config.UI.LsRefreshDeploymentsMS, config.DefaultLsRefreshDeploymentsMS),
			PodsRefreshInterval:        getInterval(m.config.Config.UI.LsRefreshPodsMS, config.DefaultLsRefreshPodsMS),
			OSDsRefreshInterval:        getInterval(m.config.Config.UI.LsRefreshOSDsMS, config.DefaultLsRefreshOSDsMS),
			HeaderRefreshInterval:      getInterval(m.config.Config.UI.LsRefreshHeaderMS, config.DefaultLsRefreshHeaderMS),
		}
		monitor := monitoring.NewLsMonitor(cfg)
		monitor.Start()
		return LsMonitorStartedMsg{Monitor: monitor}
	}
}

// Update implements tea.Model
func (m *LsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case DownFlowExitMsg:
		if msg.Err != nil {
			m.lastError = msg.Err
		}
		cmds = append(cmds, m.closeMaintenanceFlow())
		return m, tea.Batch(cmds...)
	case UpFlowExitMsg:
		if msg.Err != nil {
			m.lastError = msg.Err
		}
		cmds = append(cmds, m.closeMaintenanceFlow())
		return m, tea.Batch(cmds...)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if handled, cmd := m.handleHelpKeyMsg(keyMsg); handled {
			return m, cmd
		}
	}

	if m.maintenanceFlow != nil {
		updatedFlow, cmd := m.maintenanceFlow.Update(msg)
		if flow, ok := updatedFlow.(sizedModel); ok {
			m.maintenanceFlow = flow
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Batch(cmds...)
		}
	}

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

	case LsRefreshMsg:
		// Manual refresh - force update from monitor's latest data
		if m.monitor != nil {
			latest := m.monitor.GetLatest()
			if latest != nil {
				m.updateFromMonitor(latest)
			}
		}

	case LsMonitorStartedMsg:
		m.monitor = msg.Monitor

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

func (m *LsModel) computeLayout() lsLayout {
	activeHeight, inactiveHeight := m.paneHeights()

	nodesHeight := inactiveHeight
	if m.activePane == LsPaneNodes {
		nodesHeight = activeHeight
	}
	deploymentsHeight := inactiveHeight
	if m.activePane == LsPaneDeployments {
		deploymentsHeight = activeHeight
	}
	osdsHeight := inactiveHeight
	if m.activePane == LsPaneOSDs {
		osdsHeight = activeHeight
	}

	nodesWidth, maintenanceWidth := m.topRowWidths()

	nodesInnerWidth, nodesInnerHeight := innerViewSize(nodesWidth, nodesHeight)
	deploymentsInnerWidth, deploymentsInnerHeight := innerViewSize(m.width, deploymentsHeight)
	osdsInnerWidth, osdsInnerHeight := innerViewSize(m.width, osdsHeight)
	maintenanceInnerWidth, maintenanceInnerHeight := innerViewSize(maintenanceWidth, nodesHeight)

	return lsLayout{
		nodesWidth:        nodesWidth,
		maintenanceWidth:  maintenanceWidth,
		nodesHeight:       nodesHeight,
		deploymentsHeight: deploymentsHeight,
		osdsHeight:        osdsHeight,

		nodesInnerWidth:        nodesInnerWidth,
		nodesInnerHeight:       nodesInnerHeight,
		deploymentsInnerWidth:  deploymentsInnerWidth,
		deploymentsInnerHeight: deploymentsInnerHeight,
		osdsInnerWidth:         osdsInnerWidth,
		osdsInnerHeight:        osdsInnerHeight,
		maintenanceInnerWidth:  maintenanceInnerWidth,
		maintenanceInnerHeight: maintenanceInnerHeight,

		maintenanceActive: m.maintenanceFlow != nil,
	}
}

func (m *LsModel) applyLayout(layout lsLayout) {
	m.panes[LsPaneNodes].SetSize(layout.nodesWidth, layout.nodesHeight)
	m.maintenancePane.SetSize(layout.maintenanceWidth, layout.nodesHeight)
	m.maintenancePane.SetActive(layout.maintenanceActive)

	m.panes[LsPaneDeployments].SetSize(m.width, layout.deploymentsHeight)
	m.panes[LsPaneOSDs].SetSize(m.width, layout.osdsHeight)

	m.nodesView.SetSize(layout.nodesInnerWidth, layout.nodesInnerHeight)
	m.deploymentsPodsView.SetSize(layout.deploymentsInnerWidth, layout.deploymentsInnerHeight)
	m.osdsView.SetSize(layout.osdsInnerWidth, layout.osdsInnerHeight)

	if m.maintenanceFlow != nil {
		m.maintenanceFlow.SetSize(layout.maintenanceInnerWidth, layout.maintenanceInnerHeight)
	}
}

// updateViewSizes updates view dimensions based on the multi-pane layout.
func (m *LsModel) updateViewSizes() {
	layout := m.computeLayout()
	m.applyLayout(layout)
}

// paneHeights calculates the active/inactive pane heights based on layout chrome.
func (m *LsModel) paneHeights() (int, int) {
	headerHeight := 4
	statusBarHeight := 2
	availableHeight := m.height - headerHeight - statusBarHeight

	// Height distribution: active pane gets 50%, inactive get 25% each.
	activeHeight := availableHeight / 2
	inactiveHeight := availableHeight / 4

	// Ensure minimum heights.
	if activeHeight < 8 {
		activeHeight = 8
	}
	if inactiveHeight < 4 {
		inactiveHeight = 4
	}

	return activeHeight, inactiveHeight
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

	m.reselectNodeIfNeeded()
}

// updateAllCounts updates all pane counts and badges
func (m *LsModel) updateAllCounts() {
	// Nodes
	m.nodeCount = m.nodesView.Count()
	m.updatePaneBadge(LsPaneNodes, m.nodeCount)
	m.updateBadge(0, m.nodeCount)

	// Deployments/Pods - show count from currently active sub-view
	m.deploymentCount = m.deploymentsPodsView.DeploymentsCount()
	m.podCount = m.deploymentsPodsView.PodsCount()

	// Update deployments pane badge based on which view is showing
	if m.deploymentsPodsView.IsShowingPods() {
		m.updatePaneBadge(LsPaneDeployments, m.podCount)
		m.panes[LsPaneDeployments].SetTitle("Pods")
	} else {
		m.updatePaneBadge(LsPaneDeployments, m.deploymentCount)
		m.panes[LsPaneDeployments].SetTitle("Deployments")
	}
	m.updateBadge(1, m.deploymentCount)
	m.updateBadge(3, m.podCount)

	// OSDs
	m.osdCount = m.osdsView.Count()
	m.updatePaneBadge(LsPaneOSDs, m.osdCount)
	m.updateBadge(2, m.osdCount)
}

// updatePaneBadge updates a pane badge with count information
func (m *LsModel) updatePaneBadge(pane LsPane, count int) {
	m.panes[pane].SetBadge(fmt.Sprintf("%d", count))
}

// updateBadge updates a tab badge with count information (legacy)
func (m *LsModel) updateBadge(tabIndex, count int) {
	m.tabBar.SetBadge(tabIndex, fmt.Sprintf("%d", count))
}

// handleKeyPress processes keyboard input
func (m *LsModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// If help is visible, only Esc or ? closes it (Ctrl+C always quits)
	if m.helpVisible {
		switch key {
		case "esc", "?":
			m.helpVisible = false
		case "ctrl+c":
			if m.monitor != nil {
				m.monitor.Stop()
			}
			return tea.Quit
		}
		return nil
	}

	if cmd, ok := m.handleQuitKey(key); ok {
		return cmd
	}
	if m.handleHelpAndPaneNavKey(key) {
		return nil
	}
	if cmd, ok := m.handlePaneSwitchKey(key, msg); ok {
		return cmd
	}
	if m.handleDeploymentsToggleKey(key) {
		return nil
	}
	if m.handleCursorKey(key) {
		return nil
	}
	if cmd, ok := m.handleActionKey(key); ok {
		return cmd
	}

	return nil
}

func (m *LsModel) handleQuitKey(key string) (tea.Cmd, bool) {
	switch key {
	case "q", "esc", "ctrl+c":
		if m.monitor != nil {
			m.monitor.Stop()
		}
		return tea.Quit, true
	default:
		return nil, false
	}
}

func (m *LsModel) handleHelpAndPaneNavKey(key string) bool {
	switch key {
	case "?":
		m.helpVisible = !m.helpVisible
		return true
	case "tab":
		m.nextPane()
		return true
	case "shift+tab":
		m.prevPane()
		return true
	default:
		return false
	}
}

func (m *LsModel) handlePaneSwitchKey(key string, msg tea.KeyMsg) (tea.Cmd, bool) {
	switch key {
	case "1":
		m.setActivePane(LsPaneNodes)
		_, cmd := m.tabBar.Update(msg)
		return cmd, true
	case "2":
		m.setActivePane(LsPaneDeployments)
		_, cmd := m.tabBar.Update(msg)
		return cmd, true
	case "3":
		m.setActivePane(LsPaneOSDs)
		_, cmd := m.tabBar.Update(msg)
		return cmd, true
	default:
		if len(key) == 1 && key[0] >= '4' && key[0] <= '9' {
			_, cmd := m.tabBar.Update(msg)
			return cmd, true
		}
		return nil, false
	}
}

func (m *LsModel) handleDeploymentsToggleKey(key string) bool {
	switch key {
	case "[":
		if m.activePane == LsPaneDeployments {
			m.deploymentsPodsView.ShowDeployments()
			m.panes[LsPaneDeployments].SetTitle("Deployments")
			m.updateAllCounts()
		}
		return true
	case "]":
		if m.activePane == LsPaneDeployments {
			m.deploymentsPodsView.ShowPods()
			m.panes[LsPaneDeployments].SetTitle("Pods")
			m.updateAllCounts()
		}
		return true
	default:
		return false
	}
}

func (m *LsModel) handleCursorKey(key string) bool {
	switch key {
	case "j", "down":
		m.updateActiveViewCursor(1)
		return true
	case "k", "up":
		m.updateActiveViewCursor(-1)
		return true
	case "g":
		m.setActiveViewCursor(0)
		return true
	case "G":
		m.setActiveViewCursor(m.getMaxCursor())
		return true
	default:
		return false
	}
}

func (m *LsModel) handleActionKey(key string) (tea.Cmd, bool) {
	switch key {
	case "r":
		tab := m.activeTab
		return func() tea.Msg { return LsRefreshMsg{Tab: tab} }, true
	case "enter":
		return nil, true
	case "d":
		if m.activePane != LsPaneNodes {
			return nil, true
		}
		node := m.nodesView.GetSelectedNode()
		if node == nil {
			return nil, true
		}
		return m.openMaintenanceFlow(node.Name, false), true
	case "u":
		if m.activePane != LsPaneNodes {
			return nil, true
		}
		node := m.nodesView.GetSelectedNode()
		if node == nil {
			return nil, true
		}
		return m.openMaintenanceFlow(node.Name, true), true
	default:
		return nil, false
	}
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

	switch msg.Tab {
	case LsTabNodes:
		m.nodeCount = msg.Count
		m.tabBar.SetBadge(0, badge)
		m.panes[LsPaneNodes].SetBadge(badge)
	case LsTabDeployments:
		m.deploymentCount = msg.Count
		m.tabBar.SetBadge(1, badge)
		if !m.deploymentsPodsView.IsShowingPods() {
			m.panes[LsPaneDeployments].SetBadge(badge)
		}
	case LsTabOSDs:
		m.osdCount = msg.Count
		m.tabBar.SetBadge(2, badge)
		m.panes[LsPaneOSDs].SetBadge(badge)
	case LsTabPods:
		m.podCount = msg.Count
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

	layout := m.computeLayout()
	m.applyLayout(layout)

	nodes := m.panes[LsPaneNodes].View(m.nodesView.View())
	maintenance := m.maintenancePane.View(m.maintenanceContent())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, nodes, " ", maintenance))
	b.WriteString("\n")

	b.WriteString(m.panes[LsPaneDeployments].View(m.deploymentsPodsView.View()))
	b.WriteString("\n")

	b.WriteString(m.panes[LsPaneOSDs].View(m.osdsView.View()))

	return b.String()
}

func (m *LsModel) openMaintenanceFlow(nodeName string, isUp bool) tea.Cmd {
	if m.maintenanceFlow != nil {
		return nil
	}

	m.pendingReselectNode = nodeName

	var flow sizedModel
	if isUp {
		flow = NewUpModel(UpModelConfig{
			NodeName:     nodeName,
			Config:       m.config.Config,
			Client:       m.config.Client,
			Context:      m.config.Context,
			ExitBehavior: FlowExitMessage,
			Embedded:     true,
		})
	} else {
		flow = NewDownModel(DownModelConfig{
			NodeName:     nodeName,
			Config:       m.config.Config,
			Client:       m.config.Client,
			Context:      m.config.Context,
			ExitBehavior: FlowExitMessage,
			Embedded:     true,
		})
	}

	m.maintenanceFlow = flow
	m.updateViewSizes()

	return flow.Init()
}

func (m *LsModel) closeMaintenanceFlow() tea.Cmd {
	m.maintenanceFlow = nil
	m.updateViewSizes()
	return func() tea.Msg {
		return LsRefreshMsg{Tab: LsTabNodes}
	}
}

func (m *LsModel) reselectNodeIfNeeded() {
	if m.pendingReselectNode == "" {
		return
	}
	if m.nodesView.SetCursorByName(m.pendingReselectNode) {
		m.pendingReselectNode = ""
	}
}

// getPaneContent returns the view content for a specific pane
func (m *LsModel) maintenanceContent() string {
	if m.maintenanceFlow != nil {
		return m.maintenanceFlow.View()
	}

	var b strings.Builder
	b.WriteString(styles.StyleSubtle.Render("Select a node and start maintenance:"))
	b.WriteString("\n\n")
	b.WriteString(styles.StyleStatus.Render("d"))
	b.WriteString(styles.StyleSubtle.Render(" → down"))
	b.WriteString("\n")
	b.WriteString(styles.StyleStatus.Render("u"))
	b.WriteString(styles.StyleSubtle.Render(" → up"))

	if node := m.nodesView.GetSelectedNode(); node != nil {
		b.WriteString("\n\n")
		b.WriteString(styles.StyleStatus.Render("Selected: "))
		b.WriteString(node.Name)
	}

	return b.String()
}

func (m *LsModel) topRowWidths() (int, int) {
	// Leave a single character gap between the two panes.
	const gap = 1
	const minNodes = 40
	const minMaintenance = 35

	total := m.width
	if total <= 0 {
		return 0, 0
	}
	if total <= gap+1 {
		return total, 1
	}

	available := total - gap
	maintenance := max(minMaintenance, total/3)
	maintenance = min(maintenance, available-1)
	nodes := available - maintenance

	if nodes < minNodes {
		maintenance = available - minNodes
		if maintenance < 1 {
			maintenance = 1
		}
		nodes = available - maintenance
	}

	nodes = max(nodes, 1)
	maintenance = max(available-nodes, 1)
	return nodes, maintenance
}

// renderStatusBar renders the bottom status bar with help hints
func (m *LsModel) renderStatusBar() string {
	var hints []string

	hints = append(hints, "Tab/1-3: pane", "j/k: navigate")
	if m.activePane == LsPaneNodes {
		hints = append(hints, "u/d: up/down")
	}
	if m.activePane == LsPaneDeployments {
		hints = append(hints, "[/]: deployments/pods")
	}
	hints = append(hints, "r: refresh", "?: help", "q: quit")

	status := styles.StyleSubtle.Render(strings.Join(hints, "  "))
	if m.lastError == nil {
		return status
	}

	errText := styles.StyleError.Render("error: " + format.SanitizeForDisplay(m.lastError.Error()))
	return errText + "  " + status
}

// renderHelp renders the help overlay
func (m *LsModel) renderHelp() string {
	help := `
╭─────────────────────────────────────────╮
│             crook ls Help               │
├─────────────────────────────────────────┤
│  Navigation                             │
│    Tab, 1-3    Switch panes             │
│    [, ]        Toggle deployments/pods  │
│    j/k, ↑/↓    Move cursor              │
│    g/G         Go to top/bottom         │
│    Enter       View details             │
│                                         │
│  Actions                                │
│    u/d         Up/down selected node    │
│    r           Refresh data             │
│                                         │
│  General                                │
│    ?           Toggle this help         │
│    q, Esc      Quit                     │
╰─────────────────────────────────────────╯

Press Esc or ? to close
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
