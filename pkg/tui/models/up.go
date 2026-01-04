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
	"github.com/andri/crook/pkg/state"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// UpPhaseState represents the current state in the up phase workflow
type UpPhaseState int

const (
	// UpStateInit is the initial state before any action
	UpStateInit UpPhaseState = iota
	// UpStateLoadingState loads and validates the state file
	UpStateLoadingState
	// UpStateConfirm waits for user confirmation of restore plan
	UpStateConfirm
	// UpStateRestoringDeployments scales up deployments to saved replicas
	UpStateRestoringDeployments
	// UpStateScalingOperator scales up the rook-ceph-operator
	UpStateScalingOperator
	// UpStateUnsettingNoOut unsets the Ceph noout flag
	UpStateUnsettingNoOut
	// UpStateUncordoning uncordons the node
	UpStateUncordoning
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
	case UpStateLoadingState:
		return "Loading State"
	case UpStateConfirm:
		return "Awaiting Confirmation"
	case UpStateRestoringDeployments:
		return "Restoring Deployments"
	case UpStateScalingOperator:
		return "Scaling Operator"
	case UpStateUnsettingNoOut:
		return "Unsetting NoOut Flag"
	case UpStateUncordoning:
		return "Uncordoning Node"
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
	case UpStateLoadingState:
		return "Loading and validating state file..."
	case UpStateConfirm:
		return "Review the restore plan and confirm to proceed"
	case UpStateRestoringDeployments:
		return "Scaling deployments to original replica counts"
	case UpStateScalingOperator:
		return "Scaling up rook-ceph-operator to resume management"
	case UpStateUnsettingNoOut:
		return "Unsetting Ceph noout flag to allow rebalancing"
	case UpStateUncordoning:
		return "Uncordoning node to allow pod scheduling"
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

	// StateFilePath optionally overrides the default state file location
	StateFilePath string

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
	TargetReplicas  int
	CurrentReplicas int
	Status          string // "pending", "restoring", "success", "error", "missing"
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

	// Loaded state file
	loadedState   *state.State
	stateFilePath string
	stateLoadedAt time.Time

	// Restore plan
	restorePlan    []RestorePlanItem
	missingDeploys []string

	// Operation state
	startTime           time.Time
	elapsedTime         time.Duration
	lastError           error
	operationInProgress bool
}

// NewUpModel creates a new up phase model
func NewUpModel(cfg UpModelConfig) *UpModel {
	return &UpModel{
		config:        cfg,
		state:         UpStateInit,
		confirmPrompt: components.NewConfirmPrompt("Proceed with restoration?"),
		statusList:    components.NewStatusList(),
		progress:      components.NewIndeterminateProgress(""),
		restorePlan:   make([]RestorePlanItem, 0),
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
type UpPhaseCompleteMsg struct {
	StateFilePath string
}

// UpPhaseErrorMsg signals an error occurred
type UpPhaseErrorMsg struct {
	Err   error
	Stage string
}

// UpPhaseTickMsg is sent periodically to update elapsed time
type UpPhaseTickMsg struct{}

// StateLoadedMsg reports successful state file loading
type StateLoadedMsg struct {
	State         *state.State
	StatePath     string
	RestorePlan   []RestorePlanItem
	MissingDeploy []string
}

// Init implements tea.Model
func (m *UpModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadStateCmd(),
		m.tickCmd(),
	)
}

// loadStateCmd loads and validates the state file
func (m *UpModel) loadStateCmd() tea.Cmd {
	return func() tea.Msg {
		// Resolve state file path
		statePath := resolveUpStatePath(m.config.Config, m.config.StateFilePath, m.config.NodeName)

		// Load and parse state file
		loadedState, err := state.ParseFile(statePath)
		if err != nil {
			return UpPhaseErrorMsg{Err: fmt.Errorf("failed to load state file %s: %w", statePath, err), Stage: "load-state"}
		}

		// Validate node matches
		if loadedState.Node != m.config.NodeName {
			return UpPhaseErrorMsg{
				Err:   fmt.Errorf("state file node mismatch: expected %s, got %s", m.config.NodeName, loadedState.Node),
				Stage: "load-state",
			}
		}

		// Build restore plan and check for missing deployments
		restorePlan := make([]RestorePlanItem, 0, len(loadedState.Resources))
		missingDeploys := make([]string, 0)

		for _, resource := range loadedState.Resources {
			if resource.Kind != "Deployment" {
				continue
			}

			item := RestorePlanItem{
				Namespace:      resource.Namespace,
				Name:           resource.Name,
				TargetReplicas: resource.Replicas,
				Status:         "pending",
			}

			// Check current state in cluster
			deployment, getErr := m.config.Client.GetDeployment(m.config.Context, resource.Namespace, resource.Name)
			if getErr != nil {
				item.Status = "missing"
				item.CurrentReplicas = -1
				missingDeploys = append(missingDeploys, fmt.Sprintf("%s/%s", resource.Namespace, resource.Name))
			} else {
				if deployment.Spec.Replicas != nil {
					item.CurrentReplicas = int(*deployment.Spec.Replicas)
				}
			}

			restorePlan = append(restorePlan, item)
		}

		return StateLoadedMsg{
			State:         loadedState,
			StatePath:     statePath,
			RestorePlan:   restorePlan,
			MissingDeploy: missingDeploys,
		}
	}
}

// tickCmd returns a command that ticks every 100ms
func (m *UpModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return UpPhaseTickMsg{}
	})
}

// executeUpPhaseCmd runs the actual up phase operation
func (m *UpModel) executeUpPhaseCmd() tea.Cmd {
	return func() tea.Msg {
		opts := maintenance.UpPhaseOptions{
			ProgressCallback: func(progress maintenance.UpPhaseProgress) {
				// Progress is tracked via state transitions
			},
			StateFilePath:          m.config.StateFilePath,
			SkipMissingDeployments: len(m.missingDeploys) > 0, // Skip if user confirmed with missing
		}

		err := maintenance.ExecuteUpPhase(
			m.config.Context,
			m.config.Client,
			m.config.Config,
			m.config.NodeName,
			opts,
		)

		if err != nil {
			return UpPhaseErrorMsg{Err: err, Stage: "execute"}
		}

		return UpPhaseCompleteMsg{StateFilePath: m.stateFilePath}
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

	case StateLoadedMsg:
		m.loadedState = msg.State
		m.stateFilePath = msg.StatePath
		m.stateLoadedAt = time.Now()
		m.restorePlan = msg.RestorePlan
		m.missingDeploys = msg.MissingDeploy
		m.state = UpStateConfirm

		// Update confirm prompt based on missing deployments
		if len(m.missingDeploys) > 0 {
			m.confirmPrompt.Details = fmt.Sprintf("%d deployment(s) missing from cluster - they will be skipped", len(m.missingDeploys))
		} else {
			m.confirmPrompt.Details = fmt.Sprintf("%d deployment(s) will be restored", len(m.restorePlan))
		}

	case UpPhaseProgressMsg:
		m.updateStateFromProgress(msg)

	case UpPhaseCompleteMsg:
		m.state = UpStateComplete
		m.operationInProgress = false
		m.progress.Complete()

	case UpPhaseErrorMsg:
		m.state = UpStateError
		m.lastError = msg.Err
		m.operationInProgress = false
		m.progress.Error()

	case components.ConfirmResultMsg:
		if msg.Result == components.ConfirmYes {
			m.startExecution()
			cmds = append(cmds, m.executeUpPhaseCmd())
		} else {
			// User cancelled or declined
			return m, tea.Quit
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
	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case UpStateError:
		switch msg.String() {
		case "r":
			// Retry - restart the operation
			m.startExecution()
			return m.executeUpPhaseCmd()
		case "q", "esc":
			return tea.Quit
		}

	case UpStateComplete:
		switch msg.String() {
		case "enter", "q", "esc":
			return tea.Quit
		}

	case UpStateConfirm:
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
func (m *UpModel) startExecution() {
	m.operationInProgress = true
	m.startTime = time.Now()
	m.state = UpStateRestoringDeployments
	m.progress = components.NewIndeterminateProgress("Processing...")
	m.initStatusList()
}

// initStatusList creates the status list for tracking progress
func (m *UpModel) initStatusList() {
	m.statusList = components.NewStatusList()
	m.statusList.AddStatus("Uncordon node", components.StatusTypePending)
	m.statusList.AddStatus("Restore deployments", components.StatusTypePending)
	m.statusList.AddStatus("Scale operator", components.StatusTypePending)
	m.statusList.AddStatus("Unset noout flag", components.StatusTypePending)
}

// updateStateFromProgress updates the model state based on progress messages
func (m *UpModel) updateStateFromProgress(msg UpPhaseProgressMsg) {
	switch msg.Stage {
	case "uncordon":
		m.state = UpStateUncordoning
		m.updateStatusItem(0, components.StatusTypeRunning)
	case "scale-up":
		m.state = UpStateRestoringDeployments
		m.updateStatusItem(0, components.StatusTypeSuccess)
		m.updateStatusItem(1, components.StatusTypeRunning)
	case "operator":
		m.state = UpStateScalingOperator
		m.updateStatusItem(1, components.StatusTypeSuccess)
		m.updateStatusItem(2, components.StatusTypeRunning)
	case "unset-noout":
		m.state = UpStateUnsettingNoOut
		m.updateStatusItem(2, components.StatusTypeSuccess)
		m.updateStatusItem(3, components.StatusTypeRunning)
	case "complete":
		m.updateStatusItem(3, components.StatusTypeSuccess)
	}
}

// updateStatusItem safely updates a status item
func (m *UpModel) updateStatusItem(index int, status components.StatusType) {
	if item := m.statusList.Get(index); item != nil {
		item.SetType(status)
	}
}

// View implements tea.Model
func (m *UpModel) View() string {
	var b strings.Builder

	// Header with current state
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Main content based on state
	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case UpStateInit, UpStateLoadingState:
		b.WriteString(m.renderLoading())
	case UpStateConfirm:
		b.WriteString(m.renderConfirmation())
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
	return fmt.Sprintf("%s Loading state file for node %s...",
		styles.IconSpinner,
		m.config.NodeName)
}

// renderConfirmation renders the confirmation screen with restore plan
func (m *UpModel) renderConfirmation() string {
	var b strings.Builder

	// State file info
	b.WriteString(styles.StyleStatus.Render("State File: "))
	b.WriteString(m.stateFilePath)
	b.WriteString("\n")

	if m.loadedState != nil {
		b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("Created: %s", m.loadedState.Timestamp.Format(time.RFC3339))))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Restore plan table
	b.WriteString(styles.StyleStatus.Render("Restore Plan:\n"))

	// Create table
	table := components.NewSimpleTable("Deployment", "Current", "Target", "Status")
	for _, item := range m.restorePlan {
		deployName := fmt.Sprintf("%s/%s", item.Namespace, item.Name)

		currentStr := fmt.Sprintf("%d", item.CurrentReplicas)
		if item.CurrentReplicas < 0 {
			currentStr = "-"
		}

		targetStr := fmt.Sprintf("%d", item.TargetReplicas)

		statusStyle := styles.StyleNormal
		switch item.Status {
		case "missing":
			statusStyle = styles.StyleWarning
		case "pending":
			statusStyle = styles.StyleSubtle
		}

		table.AddStyledRow(statusStyle, deployName, currentStr, targetStr, item.Status)
	}
	table.SetMaxRows(10)
	b.WriteString(table.View())
	b.WriteString("\n")

	// Missing deployments warning
	if len(m.missingDeploys) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.StyleWarning.Render(fmt.Sprintf("%s Warning: %d deployment(s) missing from cluster:\n",
			styles.IconWarning, len(m.missingDeploys))))
		for _, d := range m.missingDeploys {
			b.WriteString(styles.StyleWarning.Render(fmt.Sprintf("  - %s\n", d)))
		}
		b.WriteString(styles.StyleSubtle.Render("These will be skipped during restoration.\n"))
	}

	// What will happen
	b.WriteString("\n")
	b.WriteString(styles.StyleStatus.Render("This will:\n"))
	b.WriteString("  1. Uncordon the node to allow pod scheduling\n")
	b.WriteString(fmt.Sprintf("  2. Scale up %d deployment(s) to original replicas\n", len(m.restorePlan)-len(m.missingDeploys)))
	if m.loadedState != nil {
		b.WriteString(fmt.Sprintf("  3. Scale up rook-ceph-operator to %d\n", m.loadedState.OperatorReplicas))
	} else {
		b.WriteString("  3. Scale up rook-ceph-operator\n")
	}
	b.WriteString("  4. Unset Ceph noout flag to allow rebalancing\n")

	b.WriteString("\n")
	b.WriteString(m.confirmPrompt.View())

	return b.String()
}

// renderProgress renders the progress view during operations
func (m *UpModel) renderProgress() string {
	var b strings.Builder

	// Elapsed time
	b.WriteString(styles.StyleSubtle.Render(fmt.Sprintf("Elapsed: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	// Status list
	b.WriteString(m.statusList.View())

	// Current operation details
	b.WriteString("\n\n")
	b.WriteString(m.progress.View())

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
	kv.Add("Deployments Restored", fmt.Sprintf("%d", len(m.restorePlan)-len(m.missingDeploys)))
	if len(m.missingDeploys) > 0 {
		kv.AddWithType("Deployments Skipped", fmt.Sprintf("%d", len(m.missingDeploys)), components.StatusTypeWarning)
	}
	kv.Add("Duration", m.elapsedTime.Round(time.Second).String())
	kv.Add("State File", m.stateFilePath)
	b.WriteString(kv.View())

	b.WriteString("\n\n")
	b.WriteString(styles.StyleSuccess.Render("The node is now fully operational."))
	b.WriteString("\n")
	b.WriteString(styles.StyleSubtle.Render("Ceph cluster should begin rebalancing if needed."))

	return b.String()
}

// renderFooter renders context-sensitive help
func (m *UpModel) renderFooter() string {
	var help string

	switch m.state { //nolint:exhaustive // default handles all operation states uniformly
	case UpStateConfirm:
		help = "y: proceed  n: cancel  ?: help"
	case UpStateError:
		help = "r: retry  q: quit  ?: help"
	case UpStateComplete:
		help = "Enter/q: exit  ?: help"
	default:
		help = "Ctrl+C: cancel  ?: help"
	}

	return styles.StyleSubtle.Render(help)
}

// SetSize implements SubModel
func (m *UpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// resolveUpStatePath resolves the state file path (helper)
func resolveUpStatePath(cfg config.Config, overridePath, nodeName string) string {
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
