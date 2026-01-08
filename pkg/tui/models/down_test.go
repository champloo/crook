package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDownPhaseState_String(t *testing.T) {
	tests := []struct {
		state    DownPhaseState
		expected string
	}{
		{DownStateInit, "Initializing"},
		{DownStateConfirm, "Awaiting Confirmation"},
		{DownStatePreFlight, "Pre-flight Checks"},
		{DownStateCordoning, "Cordoning Node"},
		{DownStateSettingNoOut, "Setting NoOut Flag"},
		{DownStateScalingOperator, "Scaling Operator"},
		{DownStateDiscoveringDeployments, "Discovering Deployments"},
		{DownStateScalingDeployments, "Scaling Deployments"},
		{DownStateComplete, "Complete"},
		{DownStateError, "Error"},
		{DownPhaseState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("DownPhaseState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDownPhaseState_Description(t *testing.T) {
	tests := []struct {
		state       DownPhaseState
		shouldExist bool
	}{
		{DownStateInit, true},
		{DownStateConfirm, true},
		{DownStatePreFlight, true},
		{DownStateCordoning, true},
		{DownStateSettingNoOut, true},
		{DownStateScalingOperator, true},
		{DownStateDiscoveringDeployments, true},
		{DownStateScalingDeployments, true},
		{DownStateComplete, true},
		{DownStateError, true},
		{DownPhaseState(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			desc := tt.state.Description()
			if tt.shouldExist && desc == "" {
				t.Errorf("DownPhaseState.Description() returned empty for %v", tt.state)
			}
			if !tt.shouldExist && desc != "" {
				t.Errorf("DownPhaseState.Description() should be empty for unknown state, got %q", desc)
			}
		})
	}
}

func TestNewDownModel(t *testing.T) {
	cfg := DownModelConfig{
		NodeName: "test-node",
		Config:   config.Config{},
		Context:  context.Background(),
	}

	model := NewDownModel(cfg)

	if model == nil {
		t.Fatal("NewDownModel returned nil")
	}

	if model.config.NodeName != "test-node" {
		t.Errorf("NodeName = %q, want %q", model.config.NodeName, "test-node")
	}

	if model.state != DownStateInit {
		t.Errorf("initial state = %v, want %v", model.state, DownStateInit)
	}

	if model.confirmPrompt == nil {
		t.Error("confirmPrompt should not be nil")
	}

	if model.statusList == nil {
		t.Error("statusList should not be nil")
	}

	if model.progress == nil {
		t.Error("progress should not be nil")
	}
}

func TestDownModel_Init(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestDownModel_Update_WindowSize(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestDownModel_Update_DeploymentsDiscovered(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	replicas := int32(1)
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "osd-1"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "rook-ceph", Name: "mon-a"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
	}
	downPlan := []DownPlanItem{
		{Namespace: "rook-ceph", Name: "osd-1", CurrentReplicas: 1, Status: "pending"},
		{Namespace: "rook-ceph", Name: "mon-a", CurrentReplicas: 1, Status: "pending"},
	}
	msg := DeploymentsDiscoveredMsg{DownPlan: downPlan, Deployments: deployments}

	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	if m.state != DownStateConfirm {
		t.Errorf("state = %v, want %v", m.state, DownStateConfirm)
	}

	if m.deploymentCount != 2 {
		t.Errorf("deploymentCount = %d, want 2", m.deploymentCount)
	}

	if len(m.downPlan) != 2 {
		t.Errorf("downPlan length = %d, want 2", len(m.downPlan))
	}

	if len(m.discoveredDeployments) != 2 {
		t.Errorf("discoveredDeployments length = %d, want 2", len(m.discoveredDeployments))
	}
}

func TestDownModel_Update_DownPhaseComplete(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true
	model.state = DownStateScalingDeployments

	msg := DownPhaseCompleteMsg{}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	if m.state != DownStateComplete {
		t.Errorf("state = %v, want %v", m.state, DownStateComplete)
	}

	if m.operationInProgress {
		t.Error("operationInProgress should be false after complete")
	}
}

func TestDownModel_Update_DownPhaseError(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true

	testErr := errors.New("test error")
	msg := DownPhaseErrorMsg{Err: testErr, Stage: "cordon"}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	if m.state != DownStateError {
		t.Errorf("state = %v, want %v", m.state, DownStateError)
	}

	if m.lastError != testErr {
		t.Errorf("lastError = %v, want %v", m.lastError, testErr)
	}

	if m.operationInProgress {
		t.Error("operationInProgress should be false after error")
	}
}

func TestDownModel_Update_ConfirmYes(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DownStateConfirm

	msg := components.ConfirmResultMsg{Result: components.ConfirmYes}
	updatedModel, cmd := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	if !m.operationInProgress {
		t.Error("operationInProgress should be true after confirmation")
	}

	if m.state != DownStatePreFlight {
		t.Errorf("state = %v, want %v", m.state, DownStatePreFlight)
	}

	if cmd == nil {
		t.Error("should return a command to execute down phase")
	}
}

func TestDownModel_Update_ConfirmNo(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DownStateConfirm

	msg := components.ConfirmResultMsg{Result: components.ConfirmNo}
	_, cmd := model.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Error("should return a quit command on decline")
	}
}

func TestDownModel_Update_ConfirmNo_Embedded(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName:     "test-node",
		Context:      context.Background(),
		ExitBehavior: FlowExitMessage,
	})
	model.state = DownStateConfirm

	msg := components.ConfirmResultMsg{Result: components.ConfirmNo}
	_, cmd := model.Update(msg)
	if cmd == nil {
		t.Fatal("should return an exit message command on decline")
	}

	exitMsg, ok := cmd().(DownFlowExitMsg)
	if !ok {
		t.Fatalf("expected DownFlowExitMsg, got %T", cmd())
	}
	if exitMsg.Reason != FlowExitDeclined {
		t.Errorf("Reason = %v, want %v", exitMsg.Reason, FlowExitDeclined)
	}
}

func TestDownModel_Update_Tick(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true
	model.startTime = time.Now().Add(-5 * time.Second)

	msg := DownPhaseTickMsg{}
	updatedModel, cmd := model.Update(msg)
	m, ok := updatedModel.(*DownModel)
	if !ok {
		t.Fatal("expected *DownModel type")
	}

	// Should update elapsed time
	if m.elapsedTime < 4*time.Second {
		t.Errorf("elapsedTime should be at least 4s, got %v", m.elapsedTime)
	}

	// Should return another tick command
	if cmd == nil {
		t.Error("should return another tick command")
	}
}

func TestDownModel_handleKeyPress_ErrorState(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DownStateError

	// Test 'q' to quit
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	cmd := model.handleKeyPress(quitMsg)
	if cmd == nil {
		t.Error("'q' in error state should return quit command")
	}

	// Test 'esc' to quit
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd = model.handleKeyPress(escMsg)
	if cmd == nil {
		t.Error("'esc' in error state should return quit command")
	}
}

func TestDownModel_handleKeyPress_ErrorState_Embedded(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName:     "test-node",
		Context:      context.Background(),
		ExitBehavior: FlowExitMessage,
	})
	model.state = DownStateError
	model.lastError = errors.New("test error")

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := model.handleKeyPress(escMsg)
	if cmd == nil {
		t.Fatal("'esc' in error state should return exit message command")
	}

	exitMsg, ok := cmd().(DownFlowExitMsg)
	if !ok {
		t.Fatalf("expected DownFlowExitMsg, got %T", cmd())
	}
	if exitMsg.Reason != FlowExitError {
		t.Errorf("Reason = %v, want %v", exitMsg.Reason, FlowExitError)
	}
	if exitMsg.Err == nil {
		t.Error("Err should be set for error exit")
	}
}

func TestDownModel_handleKeyPress_CompleteState(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DownStateComplete

	// Test Enter to exit
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := model.handleKeyPress(enterMsg)
	if cmd == nil {
		t.Error("Enter in complete state should return quit command")
	}

	// Test 'q' to exit
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	cmd = model.handleKeyPress(quitMsg)
	if cmd == nil {
		t.Error("'q' in complete state should return quit command")
	}
}

func TestDownModel_handleKeyPress_CompleteState_Embedded(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName:     "test-node",
		Context:      context.Background(),
		ExitBehavior: FlowExitMessage,
	})
	model.state = DownStateComplete

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := model.handleKeyPress(enterMsg)
	if cmd == nil {
		t.Fatal("Enter in complete state should return exit message command")
	}

	exitMsg, ok := cmd().(DownFlowExitMsg)
	if !ok {
		t.Fatalf("expected DownFlowExitMsg, got %T", cmd())
	}
	if exitMsg.Reason != FlowExitCompleted {
		t.Errorf("Reason = %v, want %v", exitMsg.Reason, FlowExitCompleted)
	}
}

func TestDownModel_handleKeyPress_OperationInProgress(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DownStateCordoning
	model.operationInProgress = true

	// Regular keys should not do anything
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	cmd := model.handleKeyPress(quitMsg)
	if cmd != nil {
		t.Error("'q' during operation should not return command")
	}

	// Ctrl+C should still work
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	cmd = model.handleKeyPress(ctrlCMsg)
	if cmd == nil {
		t.Error("Ctrl+C during operation should return quit command")
	}
}

func TestDownModel_startExecution(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	model.startExecution()

	if !model.operationInProgress {
		t.Error("operationInProgress should be true")
	}

	if model.startTime.IsZero() {
		t.Error("startTime should be set")
	}

	if model.state != DownStatePreFlight {
		t.Errorf("state = %v, want %v", model.state, DownStatePreFlight)
	}

	if model.statusList.Count() != 6 {
		t.Errorf("statusList should have 6 items, got %d", model.statusList.Count())
	}
}

func TestDownModel_updateStateFromProgress(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.initStatusList()

	tests := []struct {
		stage    string
		expected DownPhaseState
	}{
		{"pre-flight", DownStatePreFlight},
		{"cordon", DownStateCordoning},
		{"noout", DownStateSettingNoOut},
		{"operator", DownStateScalingOperator},
		{"discover", DownStateDiscoveringDeployments},
		{"scale-down", DownStateScalingDeployments},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			model.updateStateFromProgress(DownPhaseProgressMsg{
				Stage:       tt.stage,
				Description: "test",
				Deployment:  "test/deployment",
			})

			if model.state != tt.expected {
				t.Errorf("state = %v, want %v", model.state, tt.expected)
			}
		})
	}
}

func TestDownModel_View_Init(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24

	view := model.View()

	if !contains(view, "Down Phase") {
		t.Errorf("View should contain 'Down Phase', got %q", view)
	}

	if !contains(view, "test-node") {
		t.Errorf("View should contain node name, got %q", view)
	}
}

func TestDownModel_View_Confirm(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = DownStateConfirm
	model.downPlan = []DownPlanItem{
		{Namespace: "ns", Name: "deploy1", CurrentReplicas: 1, Status: "pending"},
		{Namespace: "ns", Name: "deploy2", CurrentReplicas: 1, Status: "pending"},
	}
	model.deploymentCount = 2

	view := model.View()

	if !contains(view, "Target Node") {
		t.Errorf("View should contain 'Target Node', got %q", view)
	}

	if !contains(view, "deploy1") {
		t.Errorf("View should contain deployment names, got %q", view)
	}
}

func TestDownModel_View_Error(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = DownStateError
	model.lastError = errors.New("test error message")

	view := model.View()

	if !contains(view, "Error") {
		t.Errorf("View should contain 'Error', got %q", view)
	}

	if !contains(view, "test error message") {
		t.Errorf("View should contain error message, got %q", view)
	}
}

func TestDownModel_View_Complete(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = DownStateComplete
	model.deploymentCount = 3
	model.elapsedTime = 30 * time.Second

	view := model.View()

	if !contains(view, "Complete") {
		t.Errorf("View should contain 'Complete', got %q", view)
	}

	if !contains(view, "test-node") {
		t.Errorf("View should contain node name, got %q", view)
	}
}

func TestDownModel_SetSize(t *testing.T) {
	model := NewDownModel(DownModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	model.SetSize(100, 50)

	if model.width != 100 {
		t.Errorf("width = %d, want 100", model.width)
	}

	if model.height != 50 {
		t.Errorf("height = %d, want 50", model.height)
	}
}
