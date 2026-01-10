// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/keys"
	"github.com/andri/crook/pkg/tui/styles"
	appsv1 "k8s.io/api/apps/v1"
)

// UpPhaseState represents the current state in the up phase workflow
type UpPhaseState int

const (
	// UpStateInit is the initial state before any action
	UpStateInit UpPhaseState = iota
	// UpStateDiscovering discovers scaled-down deployments
	UpStateDiscovering
	// UpStateConfirm waits for user confirmation of restore plan
	UpStateConfirm
	// UpStateNothingToDo indicates all deployments are already at target replicas
	UpStateNothingToDo
	// UpStatePreFlight runs pre-flight validation checks
	UpStatePreFlight
	// UpStateUncordoning uncordons the node
	UpStateUncordoning
	// UpStateRestoringDeployments scales up deployments to 1 replica
	UpStateRestoringDeployments
	// UpStateScalingOperator scales up the rook-ceph-operator
	UpStateScalingOperator
	// UpStateUnsettingNoOut unsets the Ceph noout flag
	UpStateUnsettingNoOut
	// UpStateComplete indicates successful completion
	UpStateComplete
	// UpStateError indicates an error occurred
	UpStateError
)

// String returns the human-readable name for the state
func (s UpPhaseState) String() string {
	switch s {
	case UpStateInit:
		return "Initializing"
	case UpStateDiscovering:
		return "Discovering Deployments"
	case UpStateConfirm:
		return "Awaiting Confirmation"
	case UpStateNothingToDo:
		return "Nothing To Do"
	case UpStatePreFlight:
		return "Pre-flight Checks"
	case UpStateUncordoning:
		return "Uncordoning Node"
	case UpStateRestoringDeployments:
		return "Restoring Deployments"
	case UpStateScalingOperator:
		return "Scaling Operator"
	case UpStateUnsettingNoOut:
		return "Unsetting NoOut Flag"
	case UpStateComplete:
		return "Complete"
	case UpStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Description returns a detailed description of what's happening in this state
func (s UpPhaseState) Description() string {
	switch s {
	case UpStateInit:
		return "Preparing up phase workflow..."
	case UpStateDiscovering:
		return "Discovering scaled-down deployments..."
	case UpStateConfirm:
		return "Review the restore plan and confirm to proceed"
	case UpStateNothingToDo:
		return "All deployments are already scaled up"
	case UpStatePreFlight:
		return "Validating cluster prerequisites and permissions"
	case UpStateUncordoning:
		return "Uncordoning node to allow pod scheduling"
	case UpStateRestoringDeployments:
		return "Scaling deployments to 1 replica"
	case UpStateScalingOperator:
		return "Scaling up rook-ceph-operator to resume management"
	case UpStateUnsettingNoOut:
		return "Unsetting Ceph noout flag to allow rebalancing"
	case UpStateComplete:
		return "All operations completed successfully"
	case UpStateError:
		return "An error occurred during the operation"
	default:
		return ""
	}
}

// UpModelConfig holds configuration for the up phase model
type UpModelConfig struct {
	// NodeName is the target node for the up phase
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

// RestorePlanItem represents a deployment to be restored
type RestorePlanItem struct {
	Namespace       string
	Name            string
	CurrentReplicas int
	Status          string // "pending", "restoring", "success", "error"
}

// UpModel is the Bubble Tea model for the up phase workflow
type UpModel struct {
	// Configuration
	config UpModelConfig

	// Current state machine state
	state UpPhaseState

	// Terminal dimensions
	width  int
	height int

	// UI components
	confirmPrompt *components.ConfirmPrompt
	statusList    *components.StatusList
	progress      *components.ProgressBar

	// Restore plan (discovered scaled-down deployments)
	restorePlan []RestorePlanItem

	// discoveredDeployments holds the actual Deployment objects discovered during
	// the confirmation phase. These are passed to ExecuteUpPhase to avoid plan drift
	// (where the confirmed plan differs from what actually gets executed).
	discoveredDeployments []appsv1.Deployment

	// Operation state
	startTime           time.Time
	elapsedTime         time.Duration
	lastError           error
	operationInProgress bool

	// Deployment scaling progress (for display)
	currentDeployment   string
	deploymentsRestored int

	// Cancellation and progress
	cancelFunc     context.CancelFunc // Cancel function for ongoing operation
	progressChan   chan maintenance.UpPhaseProgress
	progressClosed bool // Track if progress channel is closed

	// Keybindings and help
	keyBindings keys.FlowBindings
	helpModel   help.Model
}

// NewUpModel creates a new up phase model
func NewUpModel(cfg UpModelConfig) *UpModel {
	h := help.New()
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(styles.ColorInfo)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(styles.ColorSubtle)

	return &UpModel{
		config:        cfg,
		state:         UpStateInit,
		confirmPrompt: components.NewConfirmPrompt("Proceed with restoration?"),
		statusList:    components.NewStatusList(),
		progress:      components.NewIndeterminateProgress(""),
		restorePlan:   make([]RestorePlanItem, 0),
		keyBindings:   keys.DefaultFlowBindings(),
		helpModel:     h,
	}
}

// Messages for up phase state transitions

// UpPhaseStartMsg signals to start the up phase
type UpPhaseStartMsg struct{}

// UpPhaseProgressMsg reports progress during up phase
type UpPhaseProgressMsg struct {
	Stage       string
	Description string
	Deployment  string
}

// UpPhaseCompleteMsg signals successful completion
type UpPhaseCompleteMsg struct{}

// UpPhaseErrorMsg signals an error occurred
type UpPhaseErrorMsg struct {
	Err   error
	Stage string
}

// UpPhaseTickMsg is sent periodically to update elapsed time
type UpPhaseTickMsg struct{}

// DeploymentsDiscoveredForUpMsg reports discovered scaled-down deployments
type DeploymentsDiscoveredForUpMsg struct {
	RestorePlan []RestorePlanItem
	// Deployments contains the actual Deployment objects for execution.
	// This avoids plan drift between confirmation and execution.
	Deployments []appsv1.Deployment
	// AlreadyInDesiredState indicates the node is fully in up state
	// (uncordoned, noout unset, operator running, no scaled-down deployments).
	AlreadyInDesiredState bool
}

// UpProgressChannelClosedMsg signals that the progress channel was closed
// (operation completed or errored)
type UpProgressChannelClosedMsg struct{}

// Init implements tea.Model
func (m *UpModel) Init() tea.Cmd {
	return tea.Batch(
		m.discoverDeploymentsCmd(),
		m.tickCmd(),
	)
}

// discoverDeploymentsCmd discovers scaled-down deployments for the confirmation screen
func (m *UpModel) discoverDeploymentsCmd() tea.Cmd {
	return func() tea.Msg {
		// Use nodeSelector-based discovery to find scaled-down deployments
		deployments, err := m.config.Client.ListScaledDownDeploymentsForNode(
			m.config.Context,
			m.config.Config.Namespace,
			m.config.NodeName,
		)
		if err != nil {
			return UpPhaseErrorMsg{Err: fmt.Errorf("failed to discover deployments: %w", err), Stage: "discover"}
		}

		// Order deployments for up phase (same order as actual scaling: MONs first, then others)
		orderedDeployments := maintenance.OrderDeploymentsForUp(deployments)

		// Build restore plan for display
		restorePlan := make([]RestorePlanItem, 0, len(orderedDeployments))
		for _, dep := range orderedDeployments {
			item := RestorePlanItem{
				Namespace:       dep.Namespace,
				Name:            dep.Name,
				CurrentReplicas: 0, // All discovered deployments are at 0
				Status:          "pending",
			}
			restorePlan = append(restorePlan, item)
		}

		// Check if already in desired up state (complete check including node/operator/noout)
		alreadyInState := maintenance.IsInUpState(
			m.config.Context,
			m.config.Client,
			m.config.Config,
			m.config.NodeName,
			orderedDeployments,
		)

		return DeploymentsDiscoveredForUpMsg{
			RestorePlan:           restorePlan,
			Deployments:           orderedDeployments, // Include ordered deployments for execution
			AlreadyInDesiredState: alreadyInState,
		}
	}
}

// tickCmd returns a command that ticks every 100ms
func (m *UpModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return UpPhaseTickMsg{}
	})
}

// executeUpPhaseCmd runs the actual up phase operation.
// It sets up a progress channel and returns a batch of commands:
// one to execute the operation and one to listen for progress.
func (m *UpModel) executeUpPhaseCmd() tea.Cmd {
	// Create cancellable context
	ctx, cancel := context.WithCancel(m.config.Context)
	m.cancelFunc = cancel

	// Create progress channel (buffered to avoid blocking)
	m.progressChan = make(chan maintenance.UpPhaseProgress, 10)
	m.progressClosed = false

	// Return batch of operation + progress listener
	return tea.Batch(
		m.runUpPhase(ctx),
		m.listenForProgress(),
	)
}

// runUpPhase executes the maintenance operation in a goroutine
func (m *UpModel) runUpPhase(ctx context.Context) tea.Cmd {
	progressChan := m.progressChan
	client := m.config.Client
	cfg := m.config.Config
	nodeName := m.config.NodeName
	deployments := m.discoveredDeployments // Capture discovered deployments

	return func() tea.Msg {
		opts := maintenance.UpPhaseOptions{
			ProgressCallback: func(progress maintenance.UpPhaseProgress) {
				// Non-blocking send to channel
				select {
				case progressChan <- progress:
				default:
					// Channel full, skip this update
				}
			},
			// Pass pre-discovered deployments to avoid plan drift between
			// confirmation and execution (what user confirmed is what executes)
			Deployments: deployments,
		}

		err := maintenance.ExecuteUpPhase(
			ctx,
			client,
			cfg,
			nodeName,
			opts,
		)

		// Close progress channel when done
		close(progressChan)

		if err != nil {
			return UpPhaseErrorMsg{Err: err, Stage: "execute"}
		}

		return UpPhaseCompleteMsg{}
	}
}

// listenForProgress creates a command that listens for progress updates
// and returns them as messages. It reschedules itself until the channel closes.
func (m *UpModel) listenForProgress() tea.Cmd {
	progressChan := m.progressChan

	return func() tea.Msg {
		progress, ok := <-progressChan
		if !ok {
			// Channel closed
			return UpProgressChannelClosedMsg{}
		}
		return UpPhaseProgressMsg{
			Stage:       progress.Stage,
			Description: progress.Description,
			Deployment:  progress.Deployment,
		}
	}
}

// Update implements tea.Model
func (m *UpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case UpPhaseTickMsg:
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

	case DeploymentsDiscoveredForUpMsg:
		m.restorePlan = msg.RestorePlan
		m.discoveredDeployments = msg.Deployments // Store for execution

		// Check if already in desired up state (node uncordoned, noout unset, operator running, no scaled-down deployments)
		if msg.AlreadyInDesiredState {
			m.state = UpStateNothingToDo
		} else {
			m.state = UpStateConfirm
			m.confirmPrompt.Details = fmt.Sprintf("%d deployment(s) will be restored to 1 replica", len(m.restorePlan))
		}

	case UpPhaseProgressMsg:
		m.updateStateFromProgress(msg)
		// Re-schedule the progress listener if channel is still open
		if !m.progressClosed {
			cmds = append(cmds, m.listenForProgress())
		}

	case UpProgressChannelClosedMsg:
		m.progressClosed = true
		// No need to reschedule listener

	case UpPhaseCompleteMsg:
		m.state = UpStateComplete
		m.operationInProgress = false
		m.cancelFunc = nil // Clear cancel func
		m.progress.Complete()

	case UpPhaseErrorMsg:
		m.state = UpStateError
		m.lastError = msg.Err
		m.operationInProgress = false
		m.cancelFunc = nil // Clear cancel func
		m.progress.Error()

	case components.ConfirmResultMsg:
		if msg.Result == components.ConfirmYes {
			m.startExecution()
			cmds = append(cmds, m.executeUpPhaseCmd())
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
	if m.state == UpStateConfirm {
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
func (m *UpModel) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// Update keybinding state based on current flow state
	m.updateKeyBindings()

	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case UpStateError:
		switch {
		case key.Matches(msg, m.keyBindings.Retry):
			m.startExecution()
			return m.executeUpPhaseCmd()
		case key.Matches(msg, m.keyBindings.Quit):
			return m.exitCmd(FlowExitError, m.lastError)
		}

	case UpStateComplete:
		if key.Matches(msg, m.keyBindings.Exit) {
			return m.exitCmd(FlowExitCompleted, nil)
		}

	case UpStateNothingToDo:
		if key.Matches(msg, m.keyBindings.Exit) {
			return m.exitCmd(FlowExitNothingToDo, nil)
		}

	case UpStateConfirm:
		// Let the confirm prompt handle it
		return nil

	default:
		// During operations, allow cancel
		if key.Matches(msg, m.keyBindings.Interrupt) {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			return m.exitCmd(FlowExitCancelled, nil)
		}
	}

	return nil
}

// updateKeyBindings updates the keybinding state based on current flow state
func (m *UpModel) updateKeyBindings() {
	switch m.state { //nolint:exhaustive // default handles all operation states
	case UpStateConfirm:
		m.keyBindings.SetStateConfirm()
	case UpStateError:
		m.keyBindings.SetStateError()
	case UpStateComplete, UpStateNothingToDo:
		m.keyBindings.SetStateComplete()
	default:
		m.keyBindings.SetStateRunning()
	}
}

func (m *UpModel) exitCmd(reason FlowExitReason, err error) tea.Cmd {
	return flowExitCmd(m.config.ExitBehavior, UpFlowExitMsg{Reason: reason, Err: err})
}

// startExecution initializes state for operation execution
func (m *UpModel) startExecution() {
	m.operationInProgress = true
	m.startTime = time.Now()
	m.state = UpStatePreFlight // First stage is pre-flight checks
	m.progress = components.NewIndeterminateProgress("Processing...")
	m.initStatusList()
}

// initStatusList creates the status list for tracking progress
func (m *UpModel) initStatusList() {
	m.statusList = components.NewStatusList()
	m.statusList.AddStatus("Pre-flight checks", components.StatusTypePending)
	m.statusList.AddStatus("Discover deployments", components.StatusTypePending)
	m.statusList.AddStatus("Uncordon node", components.StatusTypePending)
	m.statusList.AddStatus("Restore deployments", components.StatusTypePending)
	m.statusList.AddStatus("Scale operator", components.StatusTypePending)
	m.statusList.AddStatus("Unset noout flag", components.StatusTypePending)
}

// updateStateFromProgress updates the model state based on progress messages
func (m *UpModel) updateStateFromProgress(msg UpPhaseProgressMsg) {
	switch msg.Stage {
	case "pre-flight":
		m.state = UpStatePreFlight
		m.updateStatusItem(0, components.StatusTypeRunning)
	case "discover":
		m.state = UpStateDiscovering
		m.updateStatusItem(0, components.StatusTypeSuccess)
		m.updateStatusItem(1, components.StatusTypeRunning)
	case "uncordon":
		m.state = UpStateUncordoning
		m.updateStatusItem(1, components.StatusTypeSuccess)
		m.updateStatusItem(2, components.StatusTypeRunning)
	case "scale-up", "quorum":
		m.state = UpStateRestoringDeployments
		m.updateStatusItem(2, components.StatusTypeSuccess)
		m.updateStatusItem(3, components.StatusTypeRunning)
		// Track deployment progress when a deployment name is provided
		if msg.Deployment != "" {
			// If there was a previous deployment being restored, mark it as complete
			if m.currentDeployment != "" {
				m.updateDeploymentStatus(m.currentDeployment, "success")
				m.deploymentsRestored++
			}
			// Mark the new deployment as in-progress
			m.currentDeployment = msg.Deployment
			m.updateDeploymentStatus(msg.Deployment, "restoring")
			// Update status item to show progress counter and deployment list
			if item := m.statusList.Get(3); item != nil {
				item.SetLabel(fmt.Sprintf("Restore deployments (%d/%d)", m.deploymentsRestored, len(m.restorePlan)))
				item.SetDetails(m.buildDeploymentListDetails())
				item.DetailsOnNewLine = true
			}
		}
	case "operator":
		m.state = UpStateScalingOperator
		// Mark the last deployment as complete before moving to operator
		if m.currentDeployment != "" {
			m.updateDeploymentStatus(m.currentDeployment, "success")
			m.deploymentsRestored++
			m.currentDeployment = ""
		}
		m.updateStatusItem(3, components.StatusTypeSuccess)
		m.updateStatusItem(4, components.StatusTypeRunning)
		// Keep deployment list visible with final count
		if item := m.statusList.Get(3); item != nil {
			item.SetLabel(fmt.Sprintf("Restore deployments (%d/%d)", m.deploymentsRestored, len(m.restorePlan)))
			item.SetDetails(m.buildDeploymentListDetails())
		}
	case "unset-noout":
		m.state = UpStateUnsettingNoOut
		m.updateStatusItem(4, components.StatusTypeSuccess)
		m.updateStatusItem(5, components.StatusTypeRunning)
	case "complete":
		m.updateStatusItem(5, components.StatusTypeSuccess)
	}
}

// updateStatusItem safely updates a status item
func (m *UpModel) updateStatusItem(index int, status components.StatusType) {
	if item := m.statusList.Get(index); item != nil {
		item.SetType(status)
	}
}

// updateDeploymentStatus updates the status of a deployment in the restore plan
// deploymentName should be in "namespace/name" format
func (m *UpModel) updateDeploymentStatus(deploymentName, status string) {
	for i := range m.restorePlan {
		fullName := fmt.Sprintf("%s/%s", m.restorePlan[i].Namespace, m.restorePlan[i].Name)
		if fullName == deploymentName {
			m.restorePlan[i].Status = status
			return
		}
	}
}

// buildDeploymentListDetails builds a multi-line string showing all deployments with status icons
func (m *UpModel) buildDeploymentListDetails() string {
	var lines []string
	for _, item := range m.restorePlan {
		var styledIcon string
		switch item.Status {
		case "success":
			styledIcon = styles.StyleSuccess.Render(styles.IconCheckmark)
		case "restoring":
			styledIcon = styles.StyleStatus.Render(styles.IconSpinner)
		default: // pending
			styledIcon = styles.StyleSubtle.Render("○")
		}
		lines = append(lines, fmt.Sprintf("%s %s", styledIcon, item.Name))
	}
	return strings.Join(lines, "\n    ")
}

// View implements tea.Model
func (m *UpModel) View() tea.View {
	return tea.NewView(m.Render())
}

// Render returns the string representation for composition
func (m *UpModel) Render() string {
	var b strings.Builder

	// Header with current state
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main content based on state
	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case UpStateInit, UpStateDiscovering:
		b.WriteString(m.renderLoading())
	case UpStateConfirm:
		b.WriteString(m.renderConfirmation())
	case UpStateNothingToDo:
		b.WriteString(m.renderNothingToDo())
	case UpStateError:
		b.WriteString(m.renderError())
	case UpStateComplete:
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
func (m *UpModel) renderHeader() string {
	title := fmt.Sprintf("Up Phase: %s", m.config.NodeName)
	stateInfo := fmt.Sprintf("%s - %s", m.state.String(), m.state.Description())

	return fmt.Sprintf("%s\n%s",
		styles.StyleHeading.Render(title),
		styles.StyleSubtle.Render(stateInfo))
}

// renderLoading renders the loading state
func (m *UpModel) renderLoading() string {
	return fmt.Sprintf("%s Discovering scaled-down deployments on node %s...",
		styles.IconSpinner,
		m.config.NodeName)
}

// renderConfirmation renders the confirmation screen with restore plan
func (m *UpModel) renderConfirmation() string {
	var b strings.Builder

	// Target node info
	b.WriteString(styles.StyleStatus.Render("Target Node: "))
	b.WriteString(m.config.NodeName)
	b.WriteString("\n\n")

	// Restore plan table
	if len(m.restorePlan) > 0 {
		b.WriteString(styles.StyleStatus.Render("Deployments to restore:"))
		b.WriteString("\n")

		// Create table
		table := components.NewSimpleTable("Deployment", "Current", "Target", "Status")
		for _, item := range m.restorePlan {
			deployName := fmt.Sprintf("%s/%s", item.Namespace, item.Name)
			currentStr := fmt.Sprintf("%d", item.CurrentReplicas)
			targetStr := "1" // All deployments will be scaled to 1

			statusStyle := styles.StyleSubtle
			table.AddStyledRow(statusStyle, deployName, currentStr, targetStr, item.Status)
		}
		table.SetMaxRows(10)
		b.WriteString(table.Render())
	} else {
		b.WriteString(styles.StyleWarning.Render("No scaled-down deployments found on this node."))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// What will happen
	b.WriteString("\n")
	b.WriteString(styles.StyleStatus.Render("This will:"))
	b.WriteString("\n")
	b.WriteString("  1. Uncordon the node to allow pod scheduling\n")
	b.WriteString(fmt.Sprintf("  2. Scale up %d deployment(s) to 1 replica\n", len(m.restorePlan)))
	b.WriteString("  3. Scale up rook-ceph-operator to 1\n")
	b.WriteString("  4. Unset Ceph noout flag to allow rebalancing\n")

	b.WriteString("\n")
	b.WriteString(m.confirmPrompt.Render())

	return b.String()
}

// renderNothingToDo renders the view when all deployments are already scaled up
func (m *UpModel) renderNothingToDo() string {
	var b strings.Builder

	b.WriteString(styles.StyleSuccess.Render(fmt.Sprintf("%s All deployments are already scaled up", styles.IconCheckmark)))
	b.WriteString("\n\n")

	// Target node info
	b.WriteString(styles.StyleStatus.Render("Target Node: "))
	b.WriteString(m.config.NodeName)
	b.WriteString("\n\n")

	b.WriteString(styles.StyleSubtle.Render("No scaling action needed - the node is already operational."))

	return b.String()
}

// renderProgress renders the progress view during operations
func (m *UpModel) renderProgress() string {
	var b strings.Builder

	// Elapsed time
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("Elapsed: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	// Status list (includes deployment progress inline)
	b.WriteString(m.statusList.Render())

	return b.String()
}

// renderError renders the error state
func (m *UpModel) renderError() string {
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
func (m *UpModel) renderComplete() string {
	var b strings.Builder

	b.WriteString(styles.StyleSuccess.Render(fmt.Sprintf("%s Up Phase Complete", styles.IconCheckmark)))
	b.WriteString("\n\n")

	// Summary table
	kv := components.NewKeyValueTable()
	kv.Add("Node", m.config.NodeName)
	kv.Add("Deployments Restored", fmt.Sprintf("%d", len(m.restorePlan)))
	kv.Add("Duration", m.elapsedTime.Round(time.Second).String())
	b.WriteString(kv.Render())

	b.WriteString("\n\n")
	b.WriteString(styles.StyleSuccess.Render("The node is now fully operational."))
	b.WriteString("\n")
	b.WriteString(styles.StyleSubtle.Render("Ceph cluster should begin rebalancing if needed."))

	return b.String()
}

// renderFooter renders context-sensitive help
func (m *UpModel) renderFooter() string {
	m.updateKeyBindings()
	m.helpModel.SetWidth(m.width)
	return m.helpModel.View(&m.keyBindings)
}

// SetSize implements SubModel
func (m *UpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
