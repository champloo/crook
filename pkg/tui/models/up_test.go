package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/state"
	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpPhaseState_String(t *testing.T) {
	tests := []struct {
		state    UpPhaseState
		expected string
	}{
		{UpStateInit, "Initializing"},
		{UpStateLoadingState, "Loading State"},
		{UpStateConfirm, "Awaiting Confirmation"},
		{UpStateUncordoning, "Uncordoning Node"},
		{UpStateRestoringDeployments, "Restoring Deployments"},
		{UpStateScalingOperator, "Scaling Operator"},
		{UpStateUnsettingNoOut, "Unsetting NoOut Flag"},
		{UpStateComplete, "Complete"},
		{UpStateError, "Error"},
		{UpPhaseState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("UpPhaseState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestUpPhaseState_Description(t *testing.T) {
	tests := []struct {
		state       UpPhaseState
		shouldExist bool
	}{
		{UpStateInit, true},
		{UpStateLoadingState, true},
		{UpStateConfirm, true},
		{UpStateRestoringDeployments, true},
		{UpStateUncordoning, true},
		{UpStateScalingOperator, true},
		{UpStateUnsettingNoOut, true},
		{UpStateComplete, true},
		{UpStateError, true},
		{UpPhaseState(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			desc := tt.state.Description()
			if tt.shouldExist && desc == "" {
				t.Errorf("UpPhaseState.Description() returned empty for %v", tt.state)
			}
			if !tt.shouldExist && desc != "" {
				t.Errorf("UpPhaseState.Description() should be empty for unknown state, got %q", desc)
			}
		})
	}
}

func TestNewUpModel(t *testing.T) {
	cfg := UpModelConfig{
		NodeName:      "test-node",
		StateFilePath: "/tmp/test-state.json",
		Config:        config.Config{},
		Context:       context.Background(),
	}

	model := NewUpModel(cfg)

	if model == nil {
		t.Fatal("NewUpModel returned nil")
	}

	if model.config.NodeName != "test-node" {
		t.Errorf("NodeName = %q, want %q", model.config.NodeName, "test-node")
	}

	if model.state != UpStateInit {
		t.Errorf("initial state = %v, want %v", model.state, UpStateInit)
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

	if model.restorePlan == nil {
		t.Error("restorePlan should not be nil")
	}
}

func TestUpModel_Init(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestUpModel_Update_WindowSize(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestUpModel_Update_StateLoaded(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	testState := &state.State{
		Version:          "v1",
		Node:             "test-node",
		Timestamp:        time.Now(),
		OperatorReplicas: 1,
		Resources: []state.Resource{
			{Kind: "Deployment", Namespace: "ns1", Name: "deploy1", Replicas: 2},
		},
	}

	restorePlan := []RestorePlanItem{
		{Namespace: "ns1", Name: "deploy1", TargetReplicas: 2, CurrentReplicas: 0, Status: "pending"},
	}

	msg := StateLoadedMsg{
		State:         testState,
		StatePath:     "/tmp/state.json",
		RestorePlan:   restorePlan,
		MissingDeploy: nil,
	}

	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if m.state != UpStateConfirm {
		t.Errorf("state = %v, want %v", m.state, UpStateConfirm)
	}

	if m.loadedState != testState {
		t.Error("loadedState should be set")
	}

	if m.stateFilePath != "/tmp/state.json" {
		t.Errorf("stateFilePath = %q, want %q", m.stateFilePath, "/tmp/state.json")
	}

	if len(m.restorePlan) != 1 {
		t.Errorf("restorePlan length = %d, want 1", len(m.restorePlan))
	}
}

func TestUpModel_Update_StateLoaded_WithMissing(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	testState := &state.State{
		Version:          "v1",
		Node:             "test-node",
		Timestamp:        time.Now(),
		OperatorReplicas: 1,
	}

	restorePlan := []RestorePlanItem{
		{Namespace: "ns1", Name: "deploy1", TargetReplicas: 2, CurrentReplicas: -1, Status: "missing"},
	}

	msg := StateLoadedMsg{
		State:         testState,
		StatePath:     "/tmp/state.json",
		RestorePlan:   restorePlan,
		MissingDeploy: []string{"ns1/deploy1"},
	}

	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if len(m.missingDeploys) != 1 {
		t.Errorf("missingDeploys length = %d, want 1", len(m.missingDeploys))
	}

	if !contains(m.confirmPrompt.Details, "missing") {
		t.Errorf("confirmPrompt.Details should mention missing, got %q", m.confirmPrompt.Details)
	}
}

func TestUpModel_Update_UpPhaseComplete(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true
	model.state = UpStateUncordoning
	model.stateFilePath = "/tmp/state.json"

	msg := UpPhaseCompleteMsg{StateFilePath: "/tmp/state.json"}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if m.state != UpStateComplete {
		t.Errorf("state = %v, want %v", m.state, UpStateComplete)
	}

	if m.operationInProgress {
		t.Error("operationInProgress should be false after complete")
	}
}

func TestUpModel_Update_UpPhaseError(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true

	testErr := errors.New("test error")
	msg := UpPhaseErrorMsg{Err: testErr, Stage: "scale-up"}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if m.state != UpStateError {
		t.Errorf("state = %v, want %v", m.state, UpStateError)
	}

	if m.lastError != testErr {
		t.Errorf("lastError = %v, want %v", m.lastError, testErr)
	}

	if m.operationInProgress {
		t.Error("operationInProgress should be false after error")
	}
}

func TestUpModel_Update_ConfirmYes(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = UpStateConfirm

	msg := components.ConfirmResultMsg{Result: components.ConfirmYes}
	updatedModel, cmd := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
	}

	if !m.operationInProgress {
		t.Error("operationInProgress should be true after confirmation")
	}

	if m.state != UpStateRestoringDeployments {
		t.Errorf("state = %v, want %v", m.state, UpStateRestoringDeployments)
	}

	if cmd == nil {
		t.Error("should return a command to execute up phase")
	}
}

func TestUpModel_Update_ConfirmNo(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = UpStateConfirm

	msg := components.ConfirmResultMsg{Result: components.ConfirmNo}
	_, cmd := model.Update(msg)

	// Should return quit command
	if cmd == nil {
		t.Error("should return a quit command on decline")
	}
}

func TestUpModel_Update_Tick(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.operationInProgress = true
	model.startTime = time.Now().Add(-5 * time.Second)

	msg := UpPhaseTickMsg{}
	updatedModel, cmd := model.Update(msg)
	m, ok := updatedModel.(*UpModel)
	if !ok {
		t.Fatal("expected *UpModel type")
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

func TestUpModel_handleKeyPress_ErrorState(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = UpStateError

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

func TestUpModel_handleKeyPress_CompleteState(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = UpStateComplete

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

func TestUpModel_handleKeyPress_OperationInProgress(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = UpStateRestoringDeployments
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

func TestUpModel_startExecution(t *testing.T) {
	model := NewUpModel(UpModelConfig{
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

	if model.state != UpStateRestoringDeployments {
		t.Errorf("state = %v, want %v", model.state, UpStateRestoringDeployments)
	}

	if model.statusList.Count() != 4 {
		t.Errorf("statusList should have 4 items, got %d", model.statusList.Count())
	}
}

func TestUpModel_updateStateFromProgress(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.initStatusList()

	tests := []struct {
		stage    string
		expected UpPhaseState
	}{
		{"uncordon", UpStateUncordoning},
		{"scale-up", UpStateRestoringDeployments},
		{"operator", UpStateScalingOperator},
		{"unset-noout", UpStateUnsettingNoOut},
	}

	for _, tt := range tests {
		t.Run(tt.stage, func(t *testing.T) {
			model.updateStateFromProgress(UpPhaseProgressMsg{
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

func TestUpModel_View_Init(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24

	view := model.View()

	if !contains(view, "Up Phase") {
		t.Errorf("View should contain 'Up Phase', got %q", view)
	}

	if !contains(view, "test-node") {
		t.Errorf("View should contain node name, got %q", view)
	}
}

func TestUpModel_View_Confirm(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 40
	model.state = UpStateConfirm
	model.stateFilePath = "/tmp/state.json"
	model.loadedState = &state.State{
		Version:          "v1",
		Node:             "test-node",
		Timestamp:        time.Now(),
		OperatorReplicas: 1,
	}
	model.restorePlan = []RestorePlanItem{
		{Namespace: "ns1", Name: "deploy1", TargetReplicas: 2, CurrentReplicas: 0, Status: "pending"},
	}

	view := model.View()

	if !contains(view, "State File") {
		t.Errorf("View should contain 'State File', got %q", view)
	}

	if !contains(view, "Restore Plan") {
		t.Errorf("View should contain 'Restore Plan', got %q", view)
	}
}

func TestUpModel_View_Error(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = UpStateError
	model.lastError = errors.New("test error message")

	view := model.View()

	if !contains(view, "Error") {
		t.Errorf("View should contain 'Error', got %q", view)
	}

	if !contains(view, "test error message") {
		t.Errorf("View should contain error message, got %q", view)
	}
}

func TestUpModel_View_Complete(t *testing.T) {
	model := NewUpModel(UpModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = UpStateComplete
	model.stateFilePath = "/tmp/state.json"
	model.restorePlan = []RestorePlanItem{
		{Namespace: "ns1", Name: "deploy1", TargetReplicas: 2},
	}
	model.elapsedTime = 30 * time.Second

	view := model.View()

	if !contains(view, "Complete") {
		t.Errorf("View should contain 'Complete', got %q", view)
	}

	if !contains(view, "/tmp/state.json") {
		t.Errorf("View should contain state file path, got %q", view)
	}

	if !contains(view, "operational") {
		t.Errorf("View should contain 'operational', got %q", view)
	}
}

func TestUpModel_SetSize(t *testing.T) {
	model := NewUpModel(UpModelConfig{
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

func TestResolveUpStatePath(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.Config
		overridePath string
		nodeName     string
		expected     string
	}{
		{
			name:         "with override",
			cfg:          config.Config{},
			overridePath: "/custom/path.json",
			nodeName:     "node1",
			expected:     "/custom/path.json",
		},
		{
			name: "with template",
			cfg: config.Config{
				State: config.StateConfig{
					FilePathTemplate: "./state-{{.Node}}.json",
				},
			},
			overridePath: "",
			nodeName:     "node1",
			expected:     "./state-node1.json",
		},
		{
			name:         "default template",
			cfg:          config.Config{},
			overridePath: "",
			nodeName:     "mynode",
			expected:     "./crook-state-mynode.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveUpStatePath(tt.cfg, tt.overridePath, tt.nodeName)
			if got != tt.expected {
				t.Errorf("resolveUpStatePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRestorePlanItem(t *testing.T) {
	item := RestorePlanItem{
		Namespace:       "rook-ceph",
		Name:            "osd-1",
		TargetReplicas:  2,
		CurrentReplicas: 0,
		Status:          "pending",
	}

	if item.Namespace != "rook-ceph" {
		t.Errorf("Namespace = %q, want 'rook-ceph'", item.Namespace)
	}

	if item.Name != "osd-1" {
		t.Errorf("Name = %q, want 'osd-1'", item.Name)
	}

	if item.TargetReplicas != 2 {
		t.Errorf("TargetReplicas = %d, want 2", item.TargetReplicas)
	}

	if item.CurrentReplicas != 0 {
		t.Errorf("CurrentReplicas = %d, want 0", item.CurrentReplicas)
	}

	if item.Status != "pending" {
		t.Errorf("Status = %q, want 'pending'", item.Status)
	}
}
