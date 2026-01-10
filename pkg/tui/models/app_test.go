package models

import (
	"context"
	"testing"

	"github.com/andri/crook/pkg/config"
	tea "charm.land/bubbletea/v2"
)

type testSubModel struct {
	width  int
	height int
}

func (m *testSubModel) Init() tea.Cmd { return nil }

func (m *testSubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *testSubModel) View() string { return "test submodel" }

func (m *testSubModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func TestRouteString(t *testing.T) {
	tests := []struct {
		route    Route
		expected string
	}{
		{RouteDown, "down"},
		{RouteUp, "up"},
		{Route(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.route.String(); got != tt.expected {
				t.Errorf("Route.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewAppModel(t *testing.T) {
	cfg := AppConfig{
		Route:    RouteDown,
		NodeName: "test-node",
		Config:   config.Config{},
		Context:  context.Background(),
	}

	model := NewAppModel(cfg)

	if model == nil {
		t.Fatal("NewAppModel returned nil")
	}

	if model.route != RouteDown {
		t.Errorf("route = %v, want %v", model.route, RouteDown)
	}

	if model.config.NodeName != "test-node" {
		t.Errorf("NodeName = %q, want %q", model.config.NodeName, "test-node")
	}

	if model.initialized {
		t.Error("model should not be initialized immediately")
	}
}

func TestAppModel_Init(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestAppModel_Update_WindowSize(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	// Simulate window size message
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m, _ := updatedModel.(*AppModel)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestAppModel_Update_GlobalKeys_Quit(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	// Test ctrl+c
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := model.Update(msg)
	m, _ := updatedModel.(*AppModel)

	if !m.quitting {
		t.Error("ctrl+c should set quitting to true")
	}

	if cmd == nil {
		t.Error("ctrl+c should return a quit command")
	}
}

func TestAppModel_View_SizeWarning(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	model.initialized = true
	model.width = 60 // Below 80
	model.height = 24
	model.sizeWarning = "Terminal too narrow"

	// Add a placeholder to make View() work
	model.downModel = &testSubModel{}

	view := model.View()

	if !contains(view, "narrow") {
		t.Errorf("View should show size warning, got %q", view)
	}
}

func TestAppModel_Update_InitError(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	// Simulate init error
	testErr := tea.Msg(InitErrorMsg{Err: context.DeadlineExceeded})
	updatedModel, _ := model.Update(testErr)
	m, ok := updatedModel.(*AppModel)
	if !ok {
		t.Fatal("expected *AppModel type")
	}

	if m.initError == nil {
		t.Error("initError should be set")
	}
}

func TestAppModel_View_Quitting(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	model.quitting = true

	view := model.View()

	if view != "" {
		t.Errorf("View() when quitting should be empty, got %q", view)
	}
}

func TestAppModel_View_Loading(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	// Not initialized yet

	view := model.View()

	if view == "" {
		t.Error("View() should show loading state")
	}

	if !contains(view, "Initializing") {
		t.Errorf("View() should contain 'Initializing', got %q", view)
	}
}

func TestAppModel_View_Error(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	model.initError = context.DeadlineExceeded
	model.width = 80
	model.height = 24

	view := model.View()

	if !contains(view, "Error") {
		t.Errorf("Error view should contain 'Error', got %q", view)
	}

	if !contains(view, "deadline exceeded") {
		t.Errorf("Error view should contain error message, got %q", view)
	}
}

func TestAppModel_GetRoute(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteUp,
		Context: context.Background(),
	})

	if model.GetRoute() != RouteUp {
		t.Errorf("GetRoute() = %v, want %v", model.GetRoute(), RouteUp)
	}
}

func TestAppModel_GetTerminalSize(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	model.width = 100
	model.height = 50

	w, h := model.GetTerminalSize()

	if w != 100 || h != 50 {
		t.Errorf("GetTerminalSize() = (%d, %d), want (100, 50)", w, h)
	}
}

func TestAppModel_IsInitialized(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	if model.IsInitialized() {
		t.Error("IsInitialized() should be false initially")
	}

	model.initialized = true

	if !model.IsInitialized() {
		t.Error("IsInitialized() should be true after initialization")
	}
}

func TestAppModel_Update_SubModelsInitializedMsg(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})
	model.width = 80
	model.height = 24

	// Create a down model placeholder
	downModel := &testSubModel{}

	// Simulate SubModelsInitializedMsg
	msg := SubModelsInitializedMsg{
		DownModel: downModel,
		Route:     RouteDown,
	}
	updatedModel, cmd := model.Update(msg)
	m, ok := updatedModel.(*AppModel)
	if !ok {
		t.Fatal("expected *AppModel type")
	}

	if !m.initialized {
		t.Error("initialized should be true after SubModelsInitializedMsg")
	}

	if m.downModel != downModel {
		t.Error("downModel should be set from message")
	}

	// Should return a command to call Init on the submodel
	// But since placeholder.Init() returns nil, cmd might be nil
	// or it could be a batch
	_ = cmd // Test doesn't require cmd check here
}

func TestAppModel_Update_SubModelsInitializedMsg_Up(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteUp,
		Context: context.Background(),
	})

	upModel := &testSubModel{}

	msg := SubModelsInitializedMsg{
		UpModel: upModel,
		Route:   RouteUp,
	}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*AppModel)
	if !ok {
		t.Fatal("expected *AppModel type")
	}

	if !m.initialized {
		t.Error("initialized should be true")
	}

	if m.upModel != upModel {
		t.Error("upModel should be set from message")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{5, 5, 5},
		{0, 10, 0},
		{-5, 5, -5},
	}

	for _, tt := range tests {
		if got := min(tt.a, tt.b); got != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestAppModel_PropagateSizeToSubModels(t *testing.T) {
	model := NewAppModel(AppConfig{
		Route:   RouteDown,
		Context: context.Background(),
	})

	// Create placeholder models
	model.downModel = &testSubModel{}
	model.upModel = &testSubModel{}

	model.width = 120
	model.height = 40
	model.propagateSizeToSubModels()

	// Check that all models received the size
	if pm, ok := model.downModel.(*testSubModel); ok {
		if pm.width != 120 || pm.height != 40 {
			t.Error("downModel didn't receive size update")
		}
	}

	if pm, ok := model.upModel.(*testSubModel); ok {
		if pm.width != 120 || pm.height != 40 {
			t.Error("upModel didn't receive size update")
		}
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
