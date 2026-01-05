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
	tea "github.com/charmbracelet/bubbletea"
)

// DashboardState represents the current state of the dashboard
type DashboardState int

const (
	// DashboardStateLoading is the initial loading state
	DashboardStateLoading DashboardState = iota
	// DashboardStateReady shows the dashboard with health data
	DashboardStateReady
	// DashboardStateError indicates an error occurred
	DashboardStateError
)

// String returns the human-readable name for the state
func (s DashboardState) String() string {
	switch s {
	case DashboardStateLoading:
		return "Loading"
	case DashboardStateReady:
		return "Ready"
	case DashboardStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// DashboardModelConfig holds configuration for the dashboard model
type DashboardModelConfig struct {
	// NodeName is the target node to display health for
	NodeName string

	// Config is the application configuration
	Config config.Config

	// Client is the Kubernetes client
	Client *k8s.Client

	// Context for cancellation
	Context context.Context

	// NextRoute determines what happens when user proceeds (Enter)
	// If empty, dashboard acts as standalone view
	NextRoute Route
}

// DashboardModel is the Bubble Tea model for the cluster health dashboard
type DashboardModel struct {
	// Configuration
	config DashboardModelConfig

	// Current state
	state DashboardState

	// Terminal dimensions
	width  int
	height int

	// Monitor manages background health updates
	monitor *monitoring.Monitor

	// Latest health data
	latestUpdate  *monitoring.MonitorUpdate
	lastUpdateErr error

	// UI state
	showDetailedView bool
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(cfg DashboardModelConfig) *DashboardModel {
	return &DashboardModel{
		config: cfg,
		state:  DashboardStateLoading,
	}
}

// Messages for dashboard state updates

// DashboardMonitorStartedMsg carries the started monitor from the init command.
// This message is returned by startMonitorCmd() and processed in Update()
// to avoid mutating model state inside tea.Cmd closures.
type DashboardMonitorStartedMsg struct {
	Monitor *monitoring.Monitor
	Initial *monitoring.MonitorUpdate
}

// DashboardMonitorUpdateMsg delivers a monitoring update
type DashboardMonitorUpdateMsg struct {
	Update *monitoring.MonitorUpdate
}

// DashboardErrorMsg indicates an error occurred
type DashboardErrorMsg struct {
	Err error
}

// DashboardProceedMsg signals user wants to proceed to next phase
type DashboardProceedMsg struct{}

// DashboardRefreshTickMsg is sent to check for monitor updates
type DashboardRefreshTickMsg struct{}

// Init implements tea.Model
func (m *DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.startMonitorCmd(),
		m.tickCmd(),
	)
}

// startMonitorCmd initializes and starts the background monitor.
// Returns a DashboardMonitorStartedMsg which is handled in Update() to
// assign the monitor - following Bubble Tea's rule that cmds return messages
// and only Update() mutates model state.
func (m *DashboardModel) startMonitorCmd() tea.Cmd {
	return func() tea.Msg {
		// Build monitor config
		monitorCfg := monitoring.DefaultMonitorConfig(
			m.config.Client,
			m.config.NodeName,
			m.config.Config.Kubernetes.RookClusterNamespace,
			m.config.Config.DeploymentFilters.Prefixes,
		)

		// Create and start monitor
		monitor := monitoring.NewMonitor(monitorCfg)
		monitor.Start()

		// Get initial state
		latest := monitor.GetLatest()
		return DashboardMonitorStartedMsg{
			Monitor: monitor,
			Initial: latest,
		}
	}
}

// tickCmd returns a command that ticks every 100ms to check for updates
func (m *DashboardModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return DashboardRefreshTickMsg{}
	})
}

// Update implements tea.Model
func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case DashboardRefreshTickMsg:
		// Check for new updates from monitor
		if m.monitor != nil {
			latest := m.monitor.GetLatest()
			if latest != nil && latest.UpdateTime.After(m.lastUpdateTime()) {
				m.latestUpdate = latest
				if m.state == DashboardStateLoading {
					m.state = DashboardStateReady
				}
			}
		}
		cmds = append(cmds, m.tickCmd())

	case DashboardMonitorStartedMsg:
		// Assign the monitor from the initialization message
		// This is where model state mutation happens, safely in Update()
		m.monitor = msg.Monitor
		m.latestUpdate = msg.Initial
		if m.state == DashboardStateLoading && msg.Initial != nil {
			m.state = DashboardStateReady
		}

	case DashboardMonitorUpdateMsg:
		m.latestUpdate = msg.Update
		if m.state == DashboardStateLoading {
			m.state = DashboardStateReady
		}

	case DashboardErrorMsg:
		m.state = DashboardStateError
		m.lastUpdateErr = msg.Err

	case DashboardProceedMsg:
		// Stop monitor and signal to proceed
		m.stopMonitor()
		if m.config.NextRoute != 0 {
			return m, func() tea.Msg { return RouteChangeMsg{Route: m.config.NextRoute} }
		}
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input
func (m *DashboardModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		// Proceed to next phase
		return func() tea.Msg { return DashboardProceedMsg{} }

	case "esc", "q":
		// Cancel and quit
		m.stopMonitor()
		return tea.Quit

	case "d":
		// Toggle detailed view
		m.showDetailedView = !m.showDetailedView

	case "r":
		// Force refresh - already handled by ticker

	case "ctrl+c":
		m.stopMonitor()
		return tea.Quit
	}

	return nil
}

// stopMonitor safely stops the background monitor
func (m *DashboardModel) stopMonitor() {
	if m.monitor != nil {
		m.monitor.Stop()
		m.monitor = nil
	}
}

// lastUpdateTime returns the time of the last update
func (m *DashboardModel) lastUpdateTime() time.Time {
	if m.latestUpdate != nil {
		return m.latestUpdate.UpdateTime
	}
	return time.Time{}
}

// View implements tea.Model
func (m *DashboardModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main content based on state
	switch m.state {
	case DashboardStateLoading:
		b.WriteString(m.renderLoading())
	case DashboardStateError:
		b.WriteString(m.renderError())
	case DashboardStateReady:
		b.WriteString(m.renderDashboard())
	}

	// Footer
	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return styles.StyleBox.Width(min(m.width-4, 80)).Render(b.String())
}

// renderHeader renders the dashboard header
func (m *DashboardModel) renderHeader() string {
	title := fmt.Sprintf("Cluster Health Dashboard: %s", m.config.NodeName)

	var statusIndicator string
	if m.latestUpdate != nil && m.latestUpdate.HealthSummary != nil {
		status := m.latestUpdate.HealthSummary.Status
		switch status {
		case monitoring.HealthStatusHealthy:
			statusIndicator = styles.StyleSuccess.Render(fmt.Sprintf("%s %s", styles.IconCheckmark, status))
		case monitoring.HealthStatusDegraded:
			statusIndicator = styles.StyleWarning.Render(fmt.Sprintf("%s %s", styles.IconWarning, status))
		case monitoring.HealthStatusCritical:
			statusIndicator = styles.StyleError.Render(fmt.Sprintf("%s %s", styles.IconCross, status))
		case monitoring.HealthStatusUnknown:
			statusIndicator = styles.StyleSubtle.Render(fmt.Sprintf("? %s", status))
		}
	}

	return fmt.Sprintf("%s  %s",
		styles.StyleHeading.Render(title),
		statusIndicator)
}

// renderLoading renders the loading state
func (m *DashboardModel) renderLoading() string {
	return fmt.Sprintf("%s Loading cluster health data...",
		styles.IconSpinner)
}

// renderError renders the error state
func (m *DashboardModel) renderError() string {
	var b strings.Builder

	b.WriteString(styles.StyleError.Render(fmt.Sprintf("%s Error loading cluster health", styles.IconCross)))
	b.WriteString("\n\n")

	if m.lastUpdateErr != nil {
		b.WriteString(styles.StyleError.Render(m.lastUpdateErr.Error()))
	}

	return b.String()
}

// renderDashboard renders the main dashboard view
func (m *DashboardModel) renderDashboard() string {
	if m.latestUpdate == nil {
		return m.renderLoading()
	}

	var b strings.Builder

	// Node status section
	b.WriteString(m.renderNodeStatus())
	b.WriteString("\n\n")

	// Ceph health section
	b.WriteString(m.renderCephHealth())
	b.WriteString("\n\n")

	// OSD status section
	b.WriteString(m.renderOSDStatus())
	b.WriteString("\n\n")

	// Deployments section
	b.WriteString(m.renderDeployments())

	// Health issues section (if any)
	if m.latestUpdate.HealthSummary != nil && len(m.latestUpdate.HealthSummary.Reasons) > 0 {
		b.WriteString("\n\n")
		b.WriteString(m.renderHealthIssues())
	}

	// Last update timestamp
	b.WriteString("\n\n")
	b.WriteString(m.renderUpdateTimestamp())

	return b.String()
}

// renderNodeStatus renders the node status section
func (m *DashboardModel) renderNodeStatus() string {
	var b strings.Builder

	b.WriteString(styles.StyleStatus.Render("Node Status"))
	b.WriteString("\n")

	if m.latestUpdate.NodeStatus == nil {
		b.WriteString(styles.StyleSubtle.Render("  No data available"))
		return b.String()
	}

	ns := m.latestUpdate.NodeStatus

	// Build status indicator
	var statusIcon, statusText string
	var statusStyle = styles.StyleNormal

	if ns.Ready {
		if ns.Unschedulable {
			statusIcon = styles.IconWarning
			statusText = "Cordoned"
			statusStyle = styles.StyleWarning
		} else {
			statusIcon = styles.IconCheckmark
			statusText = "Ready"
			statusStyle = styles.StyleSuccess
		}
	} else {
		statusIcon = styles.IconCross
		statusText = "NotReady"
		statusStyle = styles.StyleError
	}

	kv := components.NewKeyValueTable()
	kv.Add("Name", ns.Name)

	// Determine status type for coloring
	var statusType components.StatusType
	if !ns.Ready {
		statusType = components.StatusTypeError
	} else if ns.Unschedulable {
		statusType = components.StatusTypeWarning
	} else {
		statusType = components.StatusTypeSuccess
	}
	kv.AddWithType("Status", statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, statusText)), statusType)

	if ns.PodCount >= 0 {
		kv.Add("Pods", fmt.Sprintf("%d", ns.PodCount))
	}
	kv.Add("Kubelet", ns.KubeletVersion)

	b.WriteString(kv.View())
	return b.String()
}

// renderCephHealth renders the Ceph health section
func (m *DashboardModel) renderCephHealth() string {
	var b strings.Builder

	b.WriteString(styles.StyleStatus.Render("Ceph Cluster"))
	b.WriteString("\n")

	if m.latestUpdate.CephHealth == nil {
		b.WriteString(styles.StyleSubtle.Render("  No data available"))
		return b.String()
	}

	ch := m.latestUpdate.CephHealth

	// Status with icon
	var statusIcon string
	var statusStyle = styles.StyleNormal
	switch ch.OverallStatus {
	case "HEALTH_OK":
		statusIcon = styles.IconCheckmark
		statusStyle = styles.StyleSuccess
	case "HEALTH_WARN":
		statusIcon = styles.IconWarning
		statusStyle = styles.StyleWarning
	case "HEALTH_ERR":
		statusIcon = styles.IconCross
		statusStyle = styles.StyleError
	default:
		statusIcon = "?"
		statusStyle = styles.StyleSubtle
	}

	kv := components.NewKeyValueTable()
	kv.Add("Health", statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, ch.OverallStatus)))
	kv.Add("OSDs", fmt.Sprintf("%d up / %d in / %d total", ch.OSDsUp, ch.OSDsIn, ch.OSDCount))
	kv.Add("Monitors", fmt.Sprintf("%d", ch.MonCount))

	// Storage usage
	if ch.DataTotal > 0 {
		usedPct := float64(ch.DataUsed) / float64(ch.DataTotal) * 100
		kv.Add("Storage", fmt.Sprintf("%.1f%% used", usedPct))
	}

	b.WriteString(kv.View())
	return b.String()
}

// renderOSDStatus renders the OSD status section
func (m *DashboardModel) renderOSDStatus() string {
	var b strings.Builder

	b.WriteString(styles.StyleStatus.Render("OSDs on Node"))
	b.WriteString("\n")

	if m.latestUpdate.OSDStatus == nil || len(m.latestUpdate.OSDStatus.OSDs) == 0 {
		b.WriteString(styles.StyleSubtle.Render("  No OSDs found on this node"))
		return b.String()
	}

	osdStatus := m.latestUpdate.OSDStatus

	// Create a simple table for OSDs
	table := components.NewSimpleTable("OSD", "Status", "Up", "In")
	for _, osd := range osdStatus.OSDs {
		var statusStyle = styles.StyleNormal
		statusText := "OK"

		if !osd.Up || !osd.In {
			statusStyle = styles.StyleWarning
			statusText = "Degraded"
		}
		if !osd.Up && !osd.In {
			statusStyle = styles.StyleError
			statusText = "Down"
		}

		upStr := boolToYesNo(osd.Up)
		inStr := boolToYesNo(osd.In)

		table.AddStyledRow(statusStyle,
			fmt.Sprintf("osd.%d", osd.ID),
			statusText,
			upStr,
			inStr)
	}

	table.SetMaxRows(5) // Limit displayed OSDs
	b.WriteString(table.View())

	return b.String()
}

// renderDeployments renders the deployments status section
func (m *DashboardModel) renderDeployments() string {
	var b strings.Builder

	b.WriteString(styles.StyleStatus.Render("Rook-Ceph Deployments"))
	b.WriteString("\n")

	if m.latestUpdate.DeploymentsStatus == nil || len(m.latestUpdate.DeploymentsStatus.Deployments) == 0 {
		b.WriteString(styles.StyleSubtle.Render("  No deployments monitored"))
		return b.String()
	}

	ds := m.latestUpdate.DeploymentsStatus

	// Summary line
	healthyCount := 0
	totalCount := len(ds.Deployments)
	for _, d := range ds.Deployments {
		if d.Status == monitoring.DeploymentHealthy {
			healthyCount++
		}
	}

	summaryStyle := styles.StyleSuccess
	if healthyCount < totalCount {
		summaryStyle = styles.StyleWarning
	}
	if healthyCount == 0 {
		summaryStyle = styles.StyleError
	}

	b.WriteString(summaryStyle.Render(fmt.Sprintf("  %d/%d healthy", healthyCount, totalCount)))

	// Show detailed view if enabled or if there are issues
	if m.showDetailedView || healthyCount < totalCount {
		b.WriteString("\n")

		table := components.NewSimpleTable("Deployment", "Replicas", "Status")
		for _, d := range ds.Deployments {
			var statusStyle = styles.StyleNormal
			switch d.Status {
			case monitoring.DeploymentHealthy:
				statusStyle = styles.StyleSuccess
			case monitoring.DeploymentScaling, monitoring.DeploymentProgressing:
				statusStyle = styles.StyleWarning
			case monitoring.DeploymentUnavailable:
				statusStyle = styles.StyleError
			}

			replicaStr := fmt.Sprintf("%d/%d", d.ReadyReplicas, d.DesiredReplicas)
			table.AddStyledRow(statusStyle, d.Name, replicaStr, string(d.Status))
		}

		table.SetMaxRows(10)
		b.WriteString(table.View())
	}

	return b.String()
}

// renderHealthIssues renders any health issues/warnings
func (m *DashboardModel) renderHealthIssues() string {
	var b strings.Builder

	summary := m.latestUpdate.HealthSummary
	if summary == nil || len(summary.Reasons) == 0 {
		return ""
	}

	// Use warning or error style based on severity
	titleStyle := styles.StyleWarning
	if summary.Status == monitoring.HealthStatusCritical {
		titleStyle = styles.StyleError
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("%s Health Issues", styles.IconWarning)))
	b.WriteString("\n")

	for _, reason := range summary.Reasons {
		b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("  • %s\n", reason)))
	}

	return b.String()
}

// renderUpdateTimestamp renders the last update timestamp
func (m *DashboardModel) renderUpdateTimestamp() string {
	if m.latestUpdate == nil {
		return ""
	}

	updateTime := m.latestUpdate.UpdateTime.Format("15:04:05")
	return styles.StyleSubtle.Render(fmt.Sprintf("Last update: %s", updateTime))
}

// renderFooter renders context-sensitive help
func (m *DashboardModel) renderFooter() string {
	var help string

	switch m.state { //nolint:exhaustive // default handles loading state
	case DashboardStateReady:
		if m.config.NextRoute != 0 {
			help = "Enter: proceed  d: toggle details  Esc/q: cancel  ?: help"
		} else {
			help = "d: toggle details  Esc/q: exit  ?: help"
		}
	case DashboardStateError:
		help = "r: retry  Esc/q: exit  ?: help"
	default:
		help = "Ctrl+C: cancel  ?: help"
	}

	return styles.StyleSubtle.Render(help)
}

// SetSize implements SubModel
func (m *DashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// boolToYesNo converts a boolean to Yes/No string
func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
