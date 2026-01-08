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
	appsv1 "k8s.io/api/apps/v1"
)

// DownPhaseState represents the current state in the down phase workflow
type DownPhaseState int

const (
	// DownStateInit is the initial state before any action
	DownStateInit DownPhaseState = iota
	// DownStateConfirm waits for user confirmation
	DownStateConfirm
	// DownStateNothingToDo indicates all deployments are already at 0 replicas
	DownStateNothingToDo
	// DownStatePreFlight runs pre-flight validation checks
	DownStatePreFlight
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
	case DownStateNothingToDo:
		return "Nothing To Do"
	case DownStatePreFlight:
		return "Pre-flight Checks"
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
	case DownStateNothingToDo:
		return "All deployments are already scaled down"
	case DownStatePreFlight:
		return "Validating cluster prerequisites and permissions"
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

	// ExitBehavior controls how the flow exits (quit vs message).
	ExitBehavior FlowExitBehavior

	// Embedded renders the model without an outer frame so it can be hosted inside
	// another container (for example, the `crook ls` Maintenance pane).
	Embedded bool

	// Config is the application configuration
	Config config.Config

	// Client is the Kubernetes client
	Client *k8s.Client

	// Context for cancellation
	Context context.Context
}

// DownPlanItem represents a deployment to be scaled down
type DownPlanItem struct {
	Namespace       string
	Name            string
	CurrentReplicas int
	Status          string // "pending", "scaling", "success", "error"
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
	startTime           time.Time
	elapsedTime         time.Duration
	lastError           error
	operationInProgress bool

	// Down plan (discovered deployments to scale down)
	downPlan []DownPlanItem

	// discoveredDeployments holds the actual Deployment objects discovered during
	// the confirmation phase. These are passed to ExecuteDownPhase to avoid plan drift
	// (where the confirmed plan differs from what actually gets executed).
	discoveredDeployments []appsv1.Deployment

	// Cancellation and progress
	cancelFunc     context.CancelFunc // Cancel function for ongoing operation
	progressChan   chan maintenance.DownPhaseProgress
	progressClosed bool // Track if progress channel is closed
}

// NewDownModel creates a new down phase model
func NewDownModel(cfg DownModelConfig) *DownModel {
	return &DownModel{
		config:        cfg,
		state:         DownStateInit,
		confirmPrompt: components.NewConfirmPrompt("Proceed with down phase?"),
		statusList:    components.NewStatusList(),
		progress:      components.NewIndeterminateProgress(""),
		downPlan:      make([]DownPlanItem, 0),
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
type DownPhaseCompleteMsg struct{}

// DownPhaseErrorMsg signals an error occurred
type DownPhaseErrorMsg struct {
	Err   error
	Stage string
}

// DownPhaseTickMsg is sent periodically to update elapsed time
type DownPhaseTickMsg struct{}

// DeploymentsDiscoveredMsg reports discovered deployments for confirmation
type DeploymentsDiscoveredMsg struct {
	DownPlan []DownPlanItem
	// Deployments contains the actual Deployment objects for execution.
	// This avoids plan drift between confirmation and execution.
	Deployments []appsv1.Deployment
}

// DownProgressChannelClosedMsg signals that the progress channel was closed
// (operation completed or errored)
type DownProgressChannelClosedMsg struct{}

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
		// Use nodeSelector-based discovery instead of pod-based
		deployments, err := m.config.Client.ListNodePinnedDeployments(
			m.config.Context,
			m.config.Config.Kubernetes.RookClusterNamespace,
			m.config.NodeName,
		)
		if err != nil {
			return DownPhaseErrorMsg{Err: err, Stage: "discover"}
		}

		// Build down plan for display
		downPlan := make([]DownPlanItem, 0, len(deployments))
		for _, dep := range deployments {
			currentReplicas := int32(0)
			if dep.Spec.Replicas != nil {
				currentReplicas = *dep.Spec.Replicas
			}
			item := DownPlanItem{
				Namespace:       dep.Namespace,
				Name:            dep.Name,
				CurrentReplicas: int(currentReplicas),
				Status:          "pending",
			}
			downPlan = append(downPlan, item)
		}

		return DeploymentsDiscoveredMsg{
			DownPlan:    downPlan,
			Deployments: deployments, // Include actual deployments for execution
		}
	}
}

// tickCmd returns a command that ticks every 100ms
func (m *DownModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return DownPhaseTickMsg{}
	})
}

// executeDownPhaseCmd runs the actual down phase operation.
// It sets up a progress channel and returns a batch of commands:
// one to execute the operation and one to listen for progress.
func (m *DownModel) executeDownPhaseCmd() tea.Cmd {
	// Create cancellable context
	ctx, cancel := context.WithCancel(m.config.Context)
	m.cancelFunc = cancel

	// Create progress channel (buffered to avoid blocking)
	m.progressChan = make(chan maintenance.DownPhaseProgress, 10)
	m.progressClosed = false

	// Return batch of operation + progress listener
	return tea.Batch(
		m.runDownPhase(ctx),
		m.listenForProgress(),
	)
}

// runDownPhase executes the maintenance operation in a goroutine
func (m *DownModel) runDownPhase(ctx context.Context) tea.Cmd {
	progressChan := m.progressChan
	client := m.config.Client
	cfg := m.config.Config
	nodeName := m.config.NodeName

	return func() tea.Msg {
		opts := maintenance.DownPhaseOptions{
			ProgressCallback: func(progress maintenance.DownPhaseProgress) {
				// Non-blocking send to channel
				select {
				case progressChan <- progress:
				default:
					// Channel full, skip this update
				}
			},
		}

		err := maintenance.ExecuteDownPhase(
			ctx,
			client,
			cfg,
			nodeName,
			opts,
		)

		// Close progress channel when done
		close(progressChan)

		if err != nil {
			return DownPhaseErrorMsg{Err: err, Stage: "execute"}
		}

		return DownPhaseCompleteMsg{}
	}
}

// listenForProgress creates a command that listens for progress updates
// and returns them as messages. It reschedules itself until the channel closes.
func (m *DownModel) listenForProgress() tea.Cmd {
	progressChan := m.progressChan

	return func() tea.Msg {
		progress, ok := <-progressChan
		if !ok {
			// Channel closed
			return DownProgressChannelClosedMsg{}
		}
		return DownPhaseProgressMsg{
			Stage:       progress.Stage,
			Description: progress.Description,
			Deployment:  progress.Deployment,
		}
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
		m.downPlan = msg.DownPlan
		m.discoveredDeployments = msg.Deployments // Store for execution
		m.deploymentCount = len(msg.DownPlan)

		// Check if all deployments are already at 0 replicas
		if m.allDeploymentsAlreadyScaledDown() {
			m.state = DownStateNothingToDo
		} else {
			m.state = DownStateConfirm
			m.confirmPrompt.Details = fmt.Sprintf("%d deployment(s) will be scaled to 0", m.deploymentCount)
		}

	case DownPhaseProgressMsg:
		m.updateStateFromProgress(msg)
		// Re-schedule the progress listener if channel is still open
		if !m.progressClosed {
			cmds = append(cmds, m.listenForProgress())
		}

	case DownProgressChannelClosedMsg:
		m.progressClosed = true
		// No need to reschedule listener

	case DownPhaseCompleteMsg:
		m.state = DownStateComplete
		m.operationInProgress = false
		m.cancelFunc = nil // Clear cancel func
		m.progress.Complete()

	case DownPhaseErrorMsg:
		m.state = DownStateError
		m.lastError = msg.Err
		m.operationInProgress = false
		m.cancelFunc = nil // Clear cancel func
		m.progress.Error()

	case components.ConfirmResultMsg:
		if msg.Result == components.ConfirmYes {
			m.startExecution()
			cmds = append(cmds, m.executeDownPhaseCmd())
		} else {
			// User cancelled or declined
			reason := FlowExitDeclined
			if msg.Result == components.ConfirmCancelled {
				reason = FlowExitCancelled
			}
			return m, m.exitCmd(reason, nil)
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
			return m.exitCmd(FlowExitError, m.lastError)
		}

	case DownStateComplete:
		switch msg.String() {
		case "enter", "q", "esc":
			return m.exitCmd(FlowExitCompleted, nil)
		}

	case DownStateNothingToDo:
		switch msg.String() {
		case "enter", "q", "esc":
			return m.exitCmd(FlowExitNothingToDo, nil)
		}

	case DownStateConfirm:
		// Let the confirm prompt handle it
		return nil

	default:
		// During operations, allow cancel
		if msg.String() == "ctrl+c" {
			// Cancel ongoing operation before quitting
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m.exitCmd(FlowExitCancelled, nil)
		}
	}

	return nil
}

func (m *DownModel) exitCmd(reason FlowExitReason, err error) tea.Cmd {
	return flowExitCmd(m.config.ExitBehavior, DownFlowExitMsg{Reason: reason, Err: err})
}

// startExecution initializes state for operation execution
func (m *DownModel) startExecution() {
	m.operationInProgress = true
	m.startTime = time.Now()
	m.state = DownStatePreFlight // First stage is pre-flight checks
	m.progress = components.NewIndeterminateProgress("Processing...")
	m.initStatusList()
}

// initStatusList creates the status list for tracking progress
func (m *DownModel) initStatusList() {
	m.statusList = components.NewStatusList()
	m.statusList.AddStatus("Pre-flight checks", components.StatusTypePending)
	m.statusList.AddStatus("Cordon node", components.StatusTypePending)
	m.statusList.AddStatus("Set noout flag", components.StatusTypePending)
	m.statusList.AddStatus("Scale operator", components.StatusTypePending)
	m.statusList.AddStatus("Discover deployments", components.StatusTypePending)
	m.statusList.AddStatus("Scale deployments", components.StatusTypePending)
}

// allDeploymentsAlreadyScaledDown returns true if all deployments in the down plan
// are already at 0 replicas (nothing to do).
func (m *DownModel) allDeploymentsAlreadyScaledDown() bool {
	// If there are no deployments to scale down, there's nothing to do
	if len(m.downPlan) == 0 {
		return true
	}

	// Check if all deployments are already at 0 replicas
	for _, item := range m.downPlan {
		if item.CurrentReplicas > 0 {
			return false
		}
	}
	return true
}

// updateStateFromProgress updates the model state based on progress messages
func (m *DownModel) updateStateFromProgress(msg DownPhaseProgressMsg) {
	switch msg.Stage {
	case "pre-flight":
		m.state = DownStatePreFlight
		m.updateStatusItem(0, components.StatusTypeRunning)
	case "cordon":
		m.state = DownStateCordoning
		m.updateStatusItem(0, components.StatusTypeSuccess)
		m.updateStatusItem(1, components.StatusTypeRunning)
	case "noout":
		m.state = DownStateSettingNoOut
		m.updateStatusItem(1, components.StatusTypeSuccess)
		m.updateStatusItem(2, components.StatusTypeRunning)
	case "operator":
		m.state = DownStateScalingOperator
		m.updateStatusItem(2, components.StatusTypeSuccess)
		m.updateStatusItem(3, components.StatusTypeRunning)
	case "discover":
		m.state = DownStateDiscoveringDeployments
		m.updateStatusItem(3, components.StatusTypeSuccess)
		m.updateStatusItem(4, components.StatusTypeRunning)
	case "scale-down":
		m.state = DownStateScalingDeployments
		m.updateStatusItem(4, components.StatusTypeSuccess)
		m.updateStatusItem(5, components.StatusTypeRunning)
		m.currentDeployment = msg.Deployment
		m.deploymentsScaled++
		// Update status item to show progress counter and current deployment
		if item := m.statusList.Get(5); item != nil {
			item.SetLabel(fmt.Sprintf("Scale deployments (%d/%d)", m.deploymentsScaled, m.deploymentCount))
			item.SetDetails(msg.Deployment)
			item.DetailsOnNewLine = true
		}
	case "complete":
		m.updateStatusItem(5, components.StatusTypeSuccess)
		// Clear the details and show final count
		if item := m.statusList.Get(5); item != nil {
			item.SetLabel(fmt.Sprintf("Scale deployments (%d/%d)", m.deploymentsScaled, m.deploymentCount))
			item.SetDetails("")
		}
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
	case DownStateNothingToDo:
		b.WriteString(m.renderNothingToDo())
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

	if m.config.Embedded {
		return b.String()
	}

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

// renderConfirmation renders the confirmation screen with down plan
func (m *DownModel) renderConfirmation() string {
	var b strings.Builder

	// Target node info
	b.WriteString(styles.StyleStatus.Render("Target Node: "))
	b.WriteString(m.config.NodeName)
	b.WriteString("\n\n")

	// Down plan table
	if len(m.downPlan) > 0 {
		b.WriteString(styles.StyleStatus.Render("Deployments to scale down:"))
		b.WriteString("\n")

		// Create table
		table := components.NewSimpleTable("Deployment", "Current", "Target", "Status")
		for _, item := range m.downPlan {
			deployName := fmt.Sprintf("%s/%s", item.Namespace, item.Name)
			currentStr := fmt.Sprintf("%d", item.CurrentReplicas)
			targetStr := "0" // All deployments will be scaled to 0

			statusStyle := styles.StyleSubtle
			table.AddStyledRow(statusStyle, deployName, currentStr, targetStr, item.Status)
		}
		table.SetMaxRows(10)
		b.WriteString(table.View())
	} else {
		b.WriteString(styles.StyleWarning.Render("No deployments found on this node."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// What will happen
	b.WriteString("\n")
	b.WriteString(styles.StyleWarning.Render("This will:"))
	b.WriteString("\n")
	b.WriteString("  1. Cordon the node (mark unschedulable)\n")
	b.WriteString("  2. Set Ceph noout flag\n")
	b.WriteString("  3. Scale down rook-ceph-operator\n")
	b.WriteString(fmt.Sprintf("  4. Scale down %d deployment(s) to 0 replicas\n", m.deploymentCount))

	b.WriteString("\n")
	b.WriteString(m.confirmPrompt.View())

	return b.String()
}

// renderNothingToDo renders the view when all deployments are already scaled down
func (m *DownModel) renderNothingToDo() string {
	var b strings.Builder

	b.WriteString(styles.StyleSuccess.Render(fmt.Sprintf("%s All deployments are already scaled down", styles.IconCheckmark)))
	b.WriteString("\n\n")

	// Target node info
	b.WriteString(styles.StyleStatus.Render("Target Node: "))
	b.WriteString(m.config.NodeName)
	b.WriteString("\n\n")

	b.WriteString(styles.StyleSubtle.Render("No scaling action needed - the node is already prepared for maintenance."))

	return b.String()
}

// renderProgress renders the progress view during operations
func (m *DownModel) renderProgress() string {
	var b strings.Builder

	// Elapsed time
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("Elapsed: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	// Status list (includes deployment progress inline)
	b.WriteString(m.statusList.View())

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
	case DownStateNothingToDo:
		help = "Enter/q: exit"
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
