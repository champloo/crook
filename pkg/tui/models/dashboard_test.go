package models

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/monitoring"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardState_String(t *testing.T) {
	tests := []struct {
		state    DashboardState
		expected string
	}{
		{DashboardStateLoading, "Loading"},
		{DashboardStateReady, "Ready"},
		{DashboardStateError, "Error"},
		{DashboardState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("DashboardState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewDashboardModel(t *testing.T) {
	cfg := DashboardModelConfig{
		NodeName:  "test-node",
		Config:    config.Config{},
		Context:   context.Background(),
		NextRoute: RouteDown,
	}

	model := NewDashboardModel(cfg)

	if model == nil {
		t.Fatal("NewDashboardModel returned nil")
	}

	if model.config.NodeName != "test-node" {
		t.Errorf("NodeName = %q, want %q", model.config.NodeName, "test-node")
	}

	if model.state != DashboardStateLoading {
		t.Errorf("initial state = %v, want %v", model.state, DashboardStateLoading)
	}

	if model.config.NextRoute != RouteDown {
		t.Errorf("NextRoute = %v, want %v", model.config.NextRoute, RouteDown)
	}
}

func TestDashboardModel_Init(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestDashboardModel_Update_WindowSize(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DashboardModel)
	if !ok {
		t.Fatal("expected *DashboardModel type")
	}

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestDashboardModel_Update_MonitorUpdate(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	update := &monitoring.MonitorUpdate{
		NodeStatus: &monitoring.NodeStatus{
			Name:  "test-node",
			Ready: true,
		},
		HealthSummary: &monitoring.HealthSummary{
			Status: monitoring.HealthStatusHealthy,
		},
		UpdateTime: time.Now(),
	}

	msg := DashboardMonitorUpdateMsg{Update: update}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DashboardModel)
	if !ok {
		t.Fatal("expected *DashboardModel type")
	}

	if m.state != DashboardStateReady {
		t.Errorf("state = %v, want %v", m.state, DashboardStateReady)
	}

	if m.latestUpdate != update {
		t.Error("latestUpdate should be set")
	}
}

func TestDashboardModel_Update_ErrorMsg(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	testErr := errors.New("test error")
	msg := DashboardErrorMsg{Err: testErr}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DashboardModel)
	if !ok {
		t.Fatal("expected *DashboardModel type")
	}

	if m.state != DashboardStateError {
		t.Errorf("state = %v, want %v", m.state, DashboardStateError)
	}

	if m.lastUpdateErr != testErr {
		t.Errorf("lastUpdateErr = %v, want %v", m.lastUpdateErr, testErr)
	}
}

func TestDashboardModel_Update_ProceedMsg(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName:  "test-node",
		Context:   context.Background(),
		NextRoute: RouteDown,
	})
	model.state = DashboardStateReady

	msg := DashboardProceedMsg{}
	_, cmd := model.Update(msg)

	// Should return a command for route change
	if cmd == nil {
		t.Error("should return a command")
	}
}

func TestDashboardModel_Update_ProceedMsg_NoNextRoute(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
		// No NextRoute set
	})
	model.state = DashboardStateReady

	msg := DashboardProceedMsg{}
	_, cmd := model.Update(msg)

	// Should return quit command when no next route
	if cmd == nil {
		t.Error("should return a quit command")
	}
}

func TestDashboardModel_Update_RefreshTick(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	msg := DashboardRefreshTickMsg{}
	_, cmd := model.Update(msg)

	// Should return another tick command
	if cmd == nil {
		t.Error("should return another tick command")
	}
}

func TestDashboardModel_handleKeyPress_Enter(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName:  "test-node",
		Context:   context.Background(),
		NextRoute: RouteDown,
	})
	model.state = DashboardStateReady

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	cmd := model.handleKeyPress(enterMsg)

	if cmd == nil {
		t.Error("Enter should return a proceed command")
	}
}

func TestDashboardModel_handleKeyPress_Escape(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DashboardStateReady

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := model.handleKeyPress(escMsg)

	if cmd == nil {
		t.Error("Esc should return a quit command")
	}
}

func TestDashboardModel_handleKeyPress_Q(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DashboardStateReady

	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	cmd := model.handleKeyPress(qMsg)

	if cmd == nil {
		t.Error("'q' should return a quit command")
	}
}

func TestDashboardModel_handleKeyPress_D(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.state = DashboardStateReady
	model.showDetailedView = false

	dMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model.handleKeyPress(dMsg)

	if !model.showDetailedView {
		t.Error("'d' should toggle showDetailedView to true")
	}

	model.handleKeyPress(dMsg)

	if model.showDetailedView {
		t.Error("'d' again should toggle showDetailedView to false")
	}
}

func TestDashboardModel_lastUpdateTime(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	// No update yet
	if !model.lastUpdateTime().IsZero() {
		t.Error("lastUpdateTime should be zero when no updates")
	}

	// With update
	updateTime := time.Now()
	model.latestUpdate = &monitoring.MonitorUpdate{
		UpdateTime: updateTime,
	}

	if model.lastUpdateTime() != updateTime {
		t.Errorf("lastUpdateTime() = %v, want %v", model.lastUpdateTime(), updateTime)
	}
}

func TestDashboardModel_View_Loading(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = DashboardStateLoading

	view := model.View()

	if !contains(view, "Loading") {
		t.Errorf("View should contain 'Loading', got %q", view)
	}
}

func TestDashboardModel_View_Error(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 24
	model.state = DashboardStateError
	model.lastUpdateErr = errors.New("test error message")

	view := model.View()

	if !contains(view, "Error") {
		t.Errorf("View should contain 'Error', got %q", view)
	}

	if !contains(view, "test error message") {
		t.Errorf("View should contain error message, got %q", view)
	}
}

func TestDashboardModel_View_Ready(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 40
	model.state = DashboardStateReady
	model.latestUpdate = &monitoring.MonitorUpdate{
		NodeStatus: &monitoring.NodeStatus{
			Name:           "test-node",
			Ready:          true,
			Unschedulable:  false,
			PodCount:       10,
			KubeletVersion: "v1.28.0",
		},
		CephHealth: &monitoring.CephHealth{
			OverallStatus: "HEALTH_OK",
			OSDCount:      6,
			OSDsUp:        6,
			OSDsIn:        6,
			MonCount:      3,
		},
		DeploymentsStatus: &monitoring.DeploymentsStatus{
			Deployments: []monitoring.DeploymentStatus{
				{Name: "osd-1", Status: monitoring.DeploymentHealthy, ReadyReplicas: 1, DesiredReplicas: 1},
			},
			OverallStatus: monitoring.DeploymentHealthy,
		},
		HealthSummary: &monitoring.HealthSummary{
			Status:  monitoring.HealthStatusHealthy,
			Reasons: []string{},
		},
		UpdateTime: time.Now(),
	}

	view := model.View()

	if !contains(view, "Node Status") {
		t.Errorf("View should contain 'Node Status', got %q", view)
	}

	if !contains(view, "Ceph Cluster") {
		t.Errorf("View should contain 'Ceph Cluster', got %q", view)
	}

	if !contains(view, "Deployments") {
		t.Errorf("View should contain 'Deployments', got %q", view)
	}
}

func TestDashboardModel_View_WithHealthIssues(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})
	model.width = 80
	model.height = 40
	model.state = DashboardStateReady
	model.latestUpdate = &monitoring.MonitorUpdate{
		NodeStatus: &monitoring.NodeStatus{
			Name:          "test-node",
			Ready:         true,
			Unschedulable: true, // Cordoned
		},
		CephHealth: &monitoring.CephHealth{
			OverallStatus: "HEALTH_WARN",
		},
		HealthSummary: &monitoring.HealthSummary{
			Status:  monitoring.HealthStatusDegraded,
			Reasons: []string{"Node is cordoned", "Ceph warnings"},
		},
		UpdateTime: time.Now(),
	}

	view := model.View()

	if !contains(view, "Health Issues") {
		t.Errorf("View should contain 'Health Issues' when there are reasons, got %q", view)
	}
}

func TestDashboardModel_SetSize(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
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

func TestBoolToYesNo(t *testing.T) {
	tests := []struct {
		input    bool
		expected string
	}{
		{true, "Yes"},
		{false, "No"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := boolToYesNo(tt.input); got != tt.expected {
				t.Errorf("boolToYesNo(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDashboardModel_renderHeader_HealthStatus(t *testing.T) {
	tests := []struct {
		name   string
		status monitoring.OverallHealthStatus
	}{
		{"healthy", monitoring.HealthStatusHealthy},
		{"degraded", monitoring.HealthStatusDegraded},
		{"critical", monitoring.HealthStatusCritical},
		{"unknown", monitoring.HealthStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewDashboardModel(DashboardModelConfig{
				NodeName: "test-node",
				Context:  context.Background(),
			})
			model.width = 80
			model.latestUpdate = &monitoring.MonitorUpdate{
				HealthSummary: &monitoring.HealthSummary{
					Status: tt.status,
				},
			}

			header := model.renderHeader()

			if !contains(header, string(tt.status)) {
				t.Errorf("header should contain status %q, got %q", tt.status, header)
			}
		})
	}
}

func TestDashboardModel_renderNodeStatus_Variations(t *testing.T) {
	tests := []struct {
		name          string
		ready         bool
		unschedulable bool
		expectedText  string
	}{
		{"ready", true, false, "Ready"},
		{"cordoned", true, true, "Cordoned"},
		{"not ready", false, false, "NotReady"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewDashboardModel(DashboardModelConfig{
				NodeName: "test-node",
				Context:  context.Background(),
			})
			model.latestUpdate = &monitoring.MonitorUpdate{
				NodeStatus: &monitoring.NodeStatus{
					Name:          "test-node",
					Ready:         tt.ready,
					Unschedulable: tt.unschedulable,
				},
			}

			rendered := model.renderNodeStatus()

			if !contains(rendered, tt.expectedText) {
				t.Errorf("renderNodeStatus should contain %q, got %q", tt.expectedText, rendered)
			}
		})
	}
}

func TestDashboardModel_renderCephHealth_Variations(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"healthy", "HEALTH_OK"},
		{"warning", "HEALTH_WARN"},
		{"error", "HEALTH_ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewDashboardModel(DashboardModelConfig{
				NodeName: "test-node",
				Context:  context.Background(),
			})
			model.latestUpdate = &monitoring.MonitorUpdate{
				CephHealth: &monitoring.CephHealth{
					OverallStatus: tt.status,
					OSDCount:      3,
					OSDsUp:        3,
					OSDsIn:        3,
					MonCount:      3,
				},
			}

			rendered := model.renderCephHealth()

			if !contains(rendered, tt.status) {
				t.Errorf("renderCephHealth should contain %q, got %q", tt.status, rendered)
			}
		})
	}
}

func TestDashboardModel_Update_MonitorStartedMsg(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	initial := &monitoring.MonitorUpdate{
		NodeStatus: &monitoring.NodeStatus{
			Name:  "test-node",
			Ready: true,
		},
		HealthSummary: &monitoring.HealthSummary{
			Status: monitoring.HealthStatusHealthy,
		},
		UpdateTime: time.Now(),
	}

	// We can't easily create a real monitor here, so test with nil monitor
	// but with initial update
	msg := DashboardMonitorStartedMsg{
		Monitor: nil, // Would be a real monitor in production
		Initial: initial,
	}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DashboardModel)
	if !ok {
		t.Fatal("expected *DashboardModel type")
	}

	if m.latestUpdate != initial {
		t.Error("latestUpdate should be set from DashboardMonitorStartedMsg")
	}

	if m.state != DashboardStateReady {
		t.Errorf("state = %v, want %v", m.state, DashboardStateReady)
	}
}

func TestDashboardModel_Update_MonitorStartedMsg_NilInitial(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	// Monitor started but no initial data yet
	msg := DashboardMonitorStartedMsg{
		Monitor: nil,
		Initial: nil,
	}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*DashboardModel)
	if !ok {
		t.Fatal("expected *DashboardModel type")
	}

	// Should still be loading since no initial data
	if m.state != DashboardStateLoading {
		t.Errorf("state = %v, want %v (should stay loading with nil initial)", m.state, DashboardStateLoading)
	}
}

func TestDashboardModel_stopMonitor(t *testing.T) {
	model := NewDashboardModel(DashboardModelConfig{
		NodeName: "test-node",
		Context:  context.Background(),
	})

	// Should not panic when monitor is nil
	model.stopMonitor()

	if model.monitor != nil {
		t.Error("monitor should be nil")
	}
}
