// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// DownPhaseState represents the current state in the down phase workflow
type DownPhaseState int

const (
	// DownStateInit is the initial state before any action
	DownStateInit DownPhaseState = iota
	// DownStateConfirm waits for user confirmation
	DownStateConfirm
	// DownStateCordoning marks the node as unschedulable
	DownStateCordoning
	// DownStateSettingNoOut sets the Ceph noout flag
	DownStateSettingNoOut
	// DownStateScalingOperator scales down the rook-ceph-operator
	DownStateScalingOperator
	// DownStateDiscoveringDeployments discovers deployments on the node
	DownStateDiscoveringDeployments
	// DownStateScalingDeployments scales down node deployments
	DownStateScalingDeployments
	// DownStateComplete indicates successful completion
	DownStateComplete
	// DownStateError indicates an error occurred
	DownStateError
)

// String returns the human-readable name for the state
func (s DownPhaseState) String() string {
	switch s {
	case DownStateInit:
		return "Initializing"
	case DownStateConfirm:
		return "Awaiting Confirmation"
	case DownStateCordoning:
		return "Cordoning Node"
	case DownStateSettingNoOut:
		return "Setting NoOut Flag"
	case DownStateScalingOperator:
		return "Scaling Operator"
	case DownStateDiscoveringDeployments:
		return "Discovering Deployments"
	case DownStateScalingDeployments:
		return "Scaling Deployments"
	case DownStateComplete:
		return "Complete"
	case DownStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Description returns a detailed description of what's happening in this state
func (s DownPhaseState) Description() string {
	switch s {
	case DownStateInit:
		return "Preparing down phase workflow..."
	case DownStateConfirm:
		return "Review the impact and confirm to proceed"
	case DownStateCordoning:
		return "Marking node as unschedulable to prevent new pods"
	case DownStateSettingNoOut:
		return "Setting Ceph noout flag to prevent rebalancing"
	case DownStateScalingOperator:
		return "Scaling down rook-ceph-operator to prevent reconciliation"
	case DownStateDiscoveringDeployments:
		return "Finding Rook-Ceph deployments running on this node"
	case DownStateScalingDeployments:
		return "Scaling down deployments to 0 replicas"
	case DownStateComplete:
		return "All operations completed successfully"
	case DownStateError:
		return "An error occurred during the operation"
	default:
		return ""
	}
}

// DownModelConfig holds configuration for the down phase model
type DownModelConfig struct {
	// NodeName is the target node for the down phase
	NodeName string

	// StateFilePath optionally overrides the default state file location
	StateFilePath string

	// Config is the application configuration
	Config config.Config

	// Client is the Kubernetes client
	Client *k8s.Client

	// Context for cancellation
	Context context.Context
}

// DownModel is the Bubble Tea model for the down phase workflow
type DownModel struct {
	// Configuration
	config DownModelConfig

	// Current state machine state
	state DownPhaseState

	// Terminal dimensions
	width  int
	height int

	// UI components
	confirmPrompt *components.ConfirmPrompt
	statusList    *components.StatusList
	progress      *components.ProgressBar

	// Operation state
	deploymentCount     int
	currentDeployment   string
	deploymentsScaled   int
	stateFilePath       string
	startTime           time.Time
	elapsedTime         time.Duration
	lastError           error
	operationInProgress bool

	// Results for display
	discoveredDeployments []string
}

// NewDownModel creates a new down phase model
func NewDownModel(cfg DownModelConfig) *DownModel {
	return &DownModel{
		config:        cfg,
		state:         DownStateInit,
		confirmPrompt: components.NewConfirmPrompt("Proceed with down phase?"),
		statusList:    components.NewStatusList(),
		progress:      components.NewIndeterminateProgress(""),
	}
}

// Messages for down phase state transitions

// DownPhaseStartMsg signals to start the down phase
type DownPhaseStartMsg struct{}

// DownPhaseProgressMsg reports progress during down phase
type DownPhaseProgressMsg struct {
	Stage       string
	Description string
	Deployment  string
}

// DownPhaseCompleteMsg signals successful completion
type DownPhaseCompleteMsg struct {
	StateFilePath string
}

// DownPhaseErrorMsg signals an error occurred
type DownPhaseErrorMsg struct {
	Err   error
	Stage string
}

// DownPhaseTickMsg is sent periodically to update elapsed time
type DownPhaseTickMsg struct{}

// DeploymentsDiscoveredMsg reports discovered deployments for confirmation
type DeploymentsDiscoveredMsg struct {
	Deployments []string
}

// Init implements tea.Model
func (m *DownModel) Init() tea.Cmd {
	return tea.Batch(
		m.discoverDeploymentsCmd(),
		m.tickCmd(),
	)
}

// discoverDeploymentsCmd discovers deployments for the confirmation screen
func (m *DownModel) discoverDeploymentsCmd() tea.Cmd {
	return func() tea.Msg {
		deployments, err := maintenance.DiscoverDeployments(
			m.config.Context,
			m.config.Client,
			m.config.NodeName,
			m.config.Config.Kubernetes.RookClusterNamespace,
			m.config.Config.DeploymentFilters.Prefixes,
		)
		if err != nil {
			return DownPhaseErrorMsg{Err: err, Stage: "discover"}
		}

		names := make([]string, len(deployments))
		for i, d := range deployments {
			names[i] = fmt.Sprintf("%s/%s", d.Namespace, d.Name)
		}

		return DeploymentsDiscoveredMsg{Deployments: names}
	}
}

// tickCmd returns a command that ticks every 100ms
func (m *DownModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return DownPhaseTickMsg{}
	})
}

// executeDownPhaseCmd runs the actual down phase operation
func (m *DownModel) executeDownPhaseCmd() tea.Cmd {
	return func() tea.Msg {
		opts := maintenance.DownPhaseOptions{
			ProgressCallback: func(progress maintenance.DownPhaseProgress) {
				// Progress is tracked via state transitions
			},
			StateFilePath: m.config.StateFilePath,
		}

		err := maintenance.ExecuteDownPhase(
			m.config.Context,
			m.config.Client,
			m.config.Config,
			m.config.NodeName,
			opts,
		)

		if err != nil {
			return DownPhaseErrorMsg{Err: err, Stage: "execute"}
		}

		// Resolve the state file path for display
		statePath := resolveDownStatePath(m.config.Config, m.config.StateFilePath, m.config.NodeName)
		return DownPhaseCompleteMsg{StateFilePath: statePath}
	}
}

// Update implements tea.Model
func (m *DownModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case DownPhaseTickMsg:
		if m.operationInProgress {
			m.elapsedTime = time.Since(m.startTime)
		}
		// Update spinner
		newProgress, cmd := m.progress.Update(msg)
		if p, ok := newProgress.(*components.ProgressBar); ok {
			m.progress = p
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, m.tickCmd())

	case DeploymentsDiscoveredMsg:
		m.discoveredDeployments = msg.Deployments
		m.deploymentCount = len(msg.Deployments)
		m.state = DownStateConfirm
		m.confirmPrompt.Details = fmt.Sprintf("%d deployment(s) will be scaled to 0", m.deploymentCount)

	case DownPhaseProgressMsg:
		m.updateStateFromProgress(msg)

	case DownPhaseCompleteMsg:
		m.state = DownStateComplete
		m.stateFilePath = msg.StateFilePath
		m.operationInProgress = false
		m.progress.Complete()

	case DownPhaseErrorMsg:
		m.state = DownStateError
		m.lastError = msg.Err
		m.operationInProgress = false
		m.progress.Error()

	case components.ConfirmResultMsg:
		if msg.Result == components.ConfirmYes {
			m.startExecution()
			cmds = append(cmds, m.executeDownPhaseCmd())
		} else {
			// User cancelled or declined
			return m, tea.Quit
		}
	}

	// Update confirm prompt if in confirm state
	if m.state == DownStateConfirm {
		newPrompt, cmd := m.confirmPrompt.Update(msg)
		if p, ok := newPrompt.(*components.ConfirmPrompt); ok {
			m.confirmPrompt = p
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input based on current state
func (m *DownModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case DownStateError:
		switch msg.String() {
		case "r":
			// Retry - restart the operation
			m.startExecution()
			return m.executeDownPhaseCmd()
		case "q", "esc":
			return tea.Quit
		}

	case DownStateComplete:
		switch msg.String() {
		case "enter", "q", "esc":
			return tea.Quit
		}

	case DownStateConfirm:
		// Let the confirm prompt handle it
		return nil

	default:
		// During operations, only allow cancel
		if msg.String() == "ctrl+c" {
			return tea.Quit
		}
	}

	return nil
}

// startExecution initializes state for operation execution
func (m *DownModel) startExecution() {
	m.operationInProgress = true
	m.startTime = time.Now()
	m.state = DownStateCordoning
	m.progress = components.NewIndeterminateProgress("Processing...")
	m.initStatusList()
}

// initStatusList creates the status list for tracking progress
func (m *DownModel) initStatusList() {
	m.statusList = components.NewStatusList()
	m.statusList.AddStatus("Cordon node", components.StatusTypePending)
	m.statusList.AddStatus("Set noout flag", components.StatusTypePending)
	m.statusList.AddStatus("Scale operator", components.StatusTypePending)
	m.statusList.AddStatus("Discover deployments", components.StatusTypePending)
	m.statusList.AddStatus("Scale deployments", components.StatusTypePending)
	m.statusList.AddStatus("Save state", components.StatusTypePending)
}

// updateStateFromProgress updates the model state based on progress messages
func (m *DownModel) updateStateFromProgress(msg DownPhaseProgressMsg) {
	switch msg.Stage {
	case "cordon":
		m.state = DownStateCordoning
		m.updateStatusItem(0, components.StatusTypeRunning)
	case "noout":
		m.state = DownStateSettingNoOut
		m.updateStatusItem(0, components.StatusTypeSuccess)
		m.updateStatusItem(1, components.StatusTypeRunning)
	case "operator":
		m.state = DownStateScalingOperator
		m.updateStatusItem(1, components.StatusTypeSuccess)
		m.updateStatusItem(2, components.StatusTypeRunning)
	case "discover":
		m.state = DownStateDiscoveringDeployments
		m.updateStatusItem(2, components.StatusTypeSuccess)
		m.updateStatusItem(3, components.StatusTypeRunning)
	case "scale-down":
		m.state = DownStateScalingDeployments
		m.updateStatusItem(3, components.StatusTypeSuccess)
		m.updateStatusItem(4, components.StatusTypeRunning)
		m.currentDeployment = msg.Deployment
		m.deploymentsScaled++
	case "save-state":
		m.updateStatusItem(4, components.StatusTypeSuccess)
		m.updateStatusItem(5, components.StatusTypeRunning)
	case "complete":
		m.updateStatusItem(5, components.StatusTypeSuccess)
	}
}

// updateStatusItem safely updates a status item
func (m *DownModel) updateStatusItem(index int, status components.StatusType) {
	if item := m.statusList.Get(index); item != nil {
		item.SetType(status)
	}
}

// View implements tea.Model
func (m *DownModel) View() string {
	var b strings.Builder

	// Header with current state
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main content based on state
	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case DownStateInit:
		b.WriteString(m.renderLoading())
	case DownStateConfirm:
		b.WriteString(m.renderConfirmation())
	case DownStateError:
		b.WriteString(m.renderError())
	case DownStateComplete:
		b.WriteString(m.renderComplete())
	default:
		b.WriteString(m.renderProgress())
	}

	// Footer with help
	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return styles.StyleBox.Width(min(m.width-4, 80)).Render(b.String())
}

// renderHeader renders the state header
func (m *DownModel) renderHeader() string {
	title := fmt.Sprintf("Down Phase: %s", m.config.NodeName)
	stateInfo := fmt.Sprintf("%s - %s", m.state.String(), m.state.Description())

	return fmt.Sprintf("%s\n%s",
		styles.StyleHeading.Render(title),
		styles.StyleSubtle.Render(stateInfo))
}

// renderLoading renders the loading state
func (m *DownModel) renderLoading() string {
	return fmt.Sprintf("%s Discovering deployments on node %s...",
		styles.IconSpinner,
		m.config.NodeName)
}

// renderConfirmation renders the confirmation screen
func (m *DownModel) renderConfirmation() string {
	var b strings.Builder

	b.WriteString(styles.StyleStatus.Render("Target Node: "))
	b.WriteString(m.config.NodeName)
	b.WriteString("\n\n")

	if len(m.discoveredDeployments) > 0 {
		b.WriteString(styles.StyleStatus.Render("Deployments to scale down:\n"))
		for _, d := range m.discoveredDeployments {
			b.WriteString(fmt.Sprintf("  %s %s\n", styles.IconArrow, d))
		}
	} else {
		b.WriteString(styles.StyleWarning.Render("No deployments found on this node.\n"))
	}

	b.WriteString("\n")
	b.WriteString(styles.StyleWarning.Render("This will:\n"))
	b.WriteString("  1. Cordon the node (mark unschedulable)\n")
	b.WriteString("  2. Set Ceph noout flag\n")
	b.WriteString("  3. Scale down rook-ceph-operator\n")
	b.WriteString(fmt.Sprintf("  4. Scale down %d deployment(s)\n", m.deploymentCount))
	b.WriteString("  5. Save state for restoration\n")

	b.WriteString("\n")
	b.WriteString(m.confirmPrompt.View())

	return b.String()
}

// renderProgress renders the progress view during operations
func (m *DownModel) renderProgress() string {
	var b strings.Builder

	// Elapsed time
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("Elapsed: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	// Status list
	b.WriteString(m.statusList.View())

	// Current operation details
	if m.currentDeployment != "" {
		b.WriteString("\n\n")
		b.WriteString(m.progress.View())
		b.WriteString("\n")
		b.WriteString(styles.StyleSubtle.Render(
			fmt.Sprintf("  %s (%d/%d)",
				m.currentDeployment,
				m.deploymentsScaled,
				m.deploymentCount)))
	}

	return b.String()
}

// renderError renders the error state
func (m *DownModel) renderError() string {
	var b strings.Builder

	b.WriteString(styles.StyleError.Render(fmt.Sprintf("%s Error", styles.IconCross)))
	b.WriteString("\n\n")

	if m.lastError != nil {
		b.WriteString(styles.StyleError.Render(m.lastError.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(styles.StyleSubtle.Render("The cluster may be in a partial state."))
	b.WriteString("\n")
	b.WriteString(styles.StyleSubtle.Render("Review the error and decide how to proceed."))

	return b.String()
}

// renderComplete renders the completion view
func (m *DownModel) renderComplete() string {
	var b strings.Builder

	b.WriteString(styles.StyleSuccess.Render(fmt.Sprintf("%s Down Phase Complete", styles.IconCheckmark)))
	b.WriteString("\n\n")

	// Summary table
	kv := components.NewKeyValueTable()
	kv.Add("Node", m.config.NodeName)
	kv.Add("Deployments Scaled", fmt.Sprintf("%d", m.deploymentCount))
	kv.Add("Duration", m.elapsedTime.Round(time.Second).String())
	kv.Add("State File", m.stateFilePath)
	b.WriteString(kv.View())

	b.WriteString("\n\n")
	b.WriteString(styles.StyleSubtle.Render("The node is now safe for maintenance."))
	b.WriteString("\n")
	b.WriteString(styles.StyleSubtle.Render("Run 'crook up' when maintenance is complete."))

	return b.String()
}

// renderFooter renders context-sensitive help
func (m *DownModel) renderFooter() string {
	var help string

	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case DownStateConfirm:
		help = "y: proceed  n: cancel  ?: help"
	case DownStateError:
		help = "r: retry  q: quit  ?: help"
	case DownStateComplete:
		help = "Enter/q: exit  ?: help"
	default:
		help = "Ctrl+C: cancel  ?: help"
	}

	return styles.StyleSubtle.Render(help)
}

// SetSize implements SubModel
func (m *DownModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// resolveDownStatePath resolves the state file path (helper)
func resolveDownStatePath(cfg config.Config, overridePath, nodeName string) string {
	if overridePath != "" {
		return overridePath
	}
	// Use template from config
	tmpl := cfg.State.FilePathTemplate
	if tmpl == "" {
		tmpl = "./crook-state-{{.Node}}.json"
	}
	return strings.ReplaceAll(tmpl, "{{.Node}}", nodeName)
}
