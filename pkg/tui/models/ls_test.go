package models

import (
	"context"
	"fmt"
	"testing"

	"github.com/andri/crook/pkg/k8s"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/monitoring"
	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

type stubSizedModel struct {
	updated bool
}

func (m *stubSizedModel) Init() tea.Cmd { return nil }

func (m *stubSizedModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	m.updated = true
	return m, nil
}

func (m *stubSizedModel) View() string { return "" }

func (m *stubSizedModel) SetSize(width, height int) {}

func TestLsTab_String(t *testing.T) {
	tests := []struct {
		tab      LsTab
		expected string
	}{
		{LsTabNodes, "Nodes"},
		{LsTabDeployments, "Deployments"},
		{LsTabOSDs, "OSDs"},
		{LsTabPods, "Pods"},
		{LsTab(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.tab.String(); got != tt.expected {
				t.Errorf("LsTab.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLsPane_String(t *testing.T) {
	tests := []struct {
		pane     LsPane
		expected string
	}{
		{LsPaneNodes, "Nodes"},
		{LsPaneDeployments, "Deployments"},
		{LsPaneOSDs, "OSDs"},
		{LsPane(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.pane.String(); got != tt.expected {
				t.Errorf("LsPane.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewLsModel(t *testing.T) {
	cfg := LsModelConfig{
		NodeFilter: "",
		Config:     config.Config{},
		Context:    context.Background(),
	}

	model := NewLsModel(cfg)

	if model == nil {
		t.Fatal("NewLsModel returned nil")
	}

	// Check new pane-based fields
	if model.activePane != LsPaneNodes {
		t.Errorf("initial active pane = %v, want %v", model.activePane, LsPaneNodes)
	}

	if model.panes[0] == nil || model.panes[1] == nil || model.panes[2] == nil {
		t.Error("panes should not be nil")
	}

	if !model.panes[0].IsActive() {
		t.Error("first pane should be active initially")
	}

	// Legacy tab bar still exists for backwards compatibility
	if model.tabBar == nil {
		t.Error("tabBar should not be nil")
	}

	if model.tabBar.TabCount() != 4 {
		t.Errorf("expected 4 tabs, got %d", model.tabBar.TabCount())
	}

	if model.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", model.cursor)
	}
}

func TestNewLsModel_WithShowTabs(t *testing.T) {
	cfg := LsModelConfig{
		Context:  context.Background(),
		ShowTabs: []LsTab{LsTabNodes, LsTabOSDs},
	}

	model := NewLsModel(cfg)

	// Legacy tab bar respects ShowTabs
	if model.tabBar.TabCount() != 2 {
		t.Errorf("expected 2 tabs, got %d", model.tabBar.TabCount())
	}

	// But panes are always all 3
	if model.panes[0] == nil || model.panes[1] == nil || model.panes[2] == nil {
		t.Error("all 3 panes should exist regardless of ShowTabs")
	}
}

func TestNewLsModel_WithNodeFilter(t *testing.T) {
	cfg := LsModelConfig{
		NodeFilter: "worker-1",
		Context:    context.Background(),
	}

	model := NewLsModel(cfg)

	if model.config.NodeFilter != "worker-1" {
		t.Errorf("NodeFilter = %q, want %q", model.config.NodeFilter, "worker-1")
	}
}

func TestLsModel_Init(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	cmd := model.Init()

	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestLsModel_Update_WindowSize(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*LsModel)
	if !ok {
		t.Fatal("expected *LsModel type")
	}

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	if m.height != 40 {
		t.Errorf("height = %d, want 40", m.height)
	}
}

func TestLsModel_View_DoesNotPanicOnTinySize(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked on tiny size: %v", r)
		}
	}()

	_, _ = model.Update(tea.WindowSizeMsg{Width: 10, Height: 6})
	_ = model.View()
}

func TestLsModel_topRowWidths(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	const gap = 1
	tests := []struct {
		width                int
		wantNodes, wantMaint int
		wantExact            bool
	}{
		{width: 120, wantNodes: 79, wantMaint: 40, wantExact: true}, // maintenance uses total/3
		{width: 80, wantNodes: 44, wantMaint: 35, wantExact: true},  // maintenance uses min
		{width: 76, wantNodes: 40, wantMaint: 35, wantExact: true},  // exact mins
		{width: 70, wantNodes: 40, wantMaint: 29, wantExact: true},  // maintenance shrinks to preserve nodes
		{width: 42, wantNodes: 40, wantMaint: 1, wantExact: true},   // minimum maintenance
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("width=%d", tt.width), func(t *testing.T) {
			model.width = tt.width
			nodes, maint := model.topRowWidths()

			if tt.wantExact {
				if nodes != tt.wantNodes || maint != tt.wantMaint {
					t.Fatalf("topRowWidths() = (%d,%d), want (%d,%d)", nodes, maint, tt.wantNodes, tt.wantMaint)
				}
			}

			if tt.width >= gap+2 {
				if nodes < 1 || maint < 1 {
					t.Fatalf("expected positive widths, got (%d,%d)", nodes, maint)
				}
				if got := nodes + maint; got != tt.width-gap {
					t.Fatalf("expected nodes+maintenance == width-gap, got %d want %d", got, tt.width-gap)
				}
			}
		})
	}
}

func TestLsModel_computeLayout_SizesAreAlwaysPositive(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	model.width = 1
	model.height = 1
	layout := model.computeLayout()

	if layout.nodesInnerWidth < 1 || layout.nodesInnerHeight < 1 {
		t.Fatalf("expected nodes inner size to be positive, got %dx%d", layout.nodesInnerWidth, layout.nodesInnerHeight)
	}
}

func TestLsModel_Update_TabSwitch(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.cursor = 5 // Set cursor to non-zero

	msg := components.TabSwitchMsg{Index: 2}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*LsModel)
	if !ok {
		t.Fatal("expected *LsModel type")
	}

	// Legacy activeTab is updated
	if m.activeTab != LsTabOSDs {
		t.Errorf("activeTab = %v, want %v", m.activeTab, LsTabOSDs)
	}

	// Pane is also updated
	if m.activePane != LsPaneOSDs {
		t.Errorf("activePane = %v, want %v", m.activePane, LsPaneOSDs)
	}

	if m.cursor != 0 {
		t.Errorf("cursor should reset to 0 on tab switch, got %d", m.cursor)
	}
}

func TestLsModel_Update_DataUpdate(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	tests := []struct {
		tab   LsTab
		count int
	}{
		{LsTabNodes, 5},
		{LsTabDeployments, 10},
		{LsTabOSDs, 3},
		{LsTabPods, 15},
	}

	for _, tt := range tests {
		msg := LsDataUpdateMsg{Tab: tt.tab, Count: tt.count}
		model.Update(msg)
	}

	if model.nodeCount != 5 {
		t.Errorf("nodeCount = %d, want 5", model.nodeCount)
	}

	if model.deploymentCount != 10 {
		t.Errorf("deploymentCount = %d, want 10", model.deploymentCount)
	}

	if model.osdCount != 3 {
		t.Errorf("osdCount = %d, want 3", model.osdCount)
	}

	if model.podCount != 15 {
		t.Errorf("podCount = %d, want 15", model.podCount)
	}
}

func TestLsModel_handleKeyPress_Quit(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	tests := []struct {
		key string
	}{
		{"q"},
		{"esc"},
		{"ctrl+c"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var msg tea.KeyMsg
			switch tt.key {
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			cmd := model.handleKeyPress(msg)
			if cmd == nil {
				t.Errorf("%s should return quit command", tt.key)
			}
		})
	}
}

func TestLsModel_handleKeyPress_Help(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	model.handleKeyPress(msg)

	if !model.helpVisible {
		t.Error("help should be visible after pressing ?")
	}

	// Non-escape keys should not close help
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if !model.helpVisible {
		t.Error("help should remain visible after pressing non-close key")
	}

	// ? should toggle help off
	model.handleKeyPress(msg)
	if model.helpVisible {
		t.Error("help should be hidden after pressing ? again")
	}

	// Esc should close help
	model.helpVisible = true
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyEsc})
	if model.helpVisible {
		t.Error("help should be hidden after pressing esc")
	}
}

func TestLsModel_Update_Help_WhileMaintenanceFlowActive(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	flow := &stubSizedModel{}
	model.maintenanceFlow = flow

	// Help key should be handled by the container model, not the embedded flow.
	msgHelp := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := model.Update(msgHelp)
	m, _ := updatedModel.(*LsModel)
	if !m.helpVisible {
		t.Error("help should be visible after pressing ? even while flow is active")
	}
	if flow.updated {
		t.Error("embedded flow should not receive key input while help is opening")
	}

	// While help is visible, keys should not be routed to the embedded flow.
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m, _ = updatedModel.(*LsModel)
	if !m.helpVisible {
		t.Error("help should remain visible after pressing non-close key while flow is active")
	}
	if flow.updated {
		t.Error("embedded flow should not receive key input while help is visible")
	}

	// Esc should close help without routing to the embedded flow.
	updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = updatedModel.(*LsModel)
	if m.helpVisible {
		t.Error("help should be hidden after pressing esc while flow is active")
	}
	if flow.updated {
		t.Error("embedded flow should not receive key input when closing help")
	}

	// Non-help keys should continue to be routed to the embedded flow.
	flow.updated = false
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if !flow.updated {
		t.Error("embedded flow should receive non-help keys")
	}
}

func TestLsModel_handleKeyPress_Navigation(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Set up nodes view with test data so cursor navigation works
	testNodes := make([]k8s.NodeInfo, 10)
	for i := 0; i < 10; i++ {
		testNodes[i] = k8s.NodeInfo{Name: fmt.Sprintf("node-%d", i)}
	}
	model.nodesView.SetNodes(testNodes)
	model.nodeCount = 10

	// Test j/down (cursor at 0 initially)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if model.nodesView.GetCursor() != 1 {
		t.Errorf("cursor after j = %d, want 1", model.nodesView.GetCursor())
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyDown})
	if model.nodesView.GetCursor() != 2 {
		t.Errorf("cursor after down = %d, want 2", model.nodesView.GetCursor())
	}

	// Test k/up
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if model.nodesView.GetCursor() != 1 {
		t.Errorf("cursor after k = %d, want 1", model.nodesView.GetCursor())
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyUp})
	if model.nodesView.GetCursor() != 0 {
		t.Errorf("cursor after up = %d, want 0", model.nodesView.GetCursor())
	}

	// Test k at top (should stay at 0)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if model.nodesView.GetCursor() != 0 {
		t.Errorf("cursor should stay at 0, got %d", model.nodesView.GetCursor())
	}

	// Test g (go to top)
	model.nodesView.SetCursor(5)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if model.nodesView.GetCursor() != 0 {
		t.Errorf("cursor after g = %d, want 0", model.nodesView.GetCursor())
	}

	// Test G (go to bottom)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if model.nodesView.GetCursor() != 9 {
		t.Errorf("cursor after G = %d, want 9", model.nodesView.GetCursor())
	}
}

func TestLsModel_handleKeyPress_Refresh(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	cmd := model.handleKeyPress(msg)

	if cmd == nil {
		t.Error("r should return a refresh command")
	}

	// Execute the command and verify it's a RefreshMsg
	result := cmd()
	if _, ok := result.(LsRefreshMsg); !ok {
		t.Error("r should produce LsRefreshMsg")
	}
}

func TestLsModel_handleKeyPress_PaneNavigation(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Test tab key cycles through panes
	if model.activePane != LsPaneNodes {
		t.Errorf("initial pane should be Nodes, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyTab})
	if model.activePane != LsPaneDeployments {
		t.Errorf("after tab, pane should be Deployments, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyTab})
	if model.activePane != LsPaneOSDs {
		t.Errorf("after second tab, pane should be OSDs, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyTab})
	if model.activePane != LsPaneNodes {
		t.Errorf("after third tab, pane should wrap to Nodes, got %v", model.activePane)
	}

	// Test number keys for direct pane selection
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if model.activePane != LsPaneDeployments {
		t.Errorf("pressing 2 should select Deployments pane, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if model.activePane != LsPaneOSDs {
		t.Errorf("pressing 3 should select OSDs pane, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if model.activePane != LsPaneNodes {
		t.Errorf("pressing 1 should select Nodes pane, got %v", model.activePane)
	}
}

func TestLsModel_handleKeyPress_ShiftTab(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Start at Nodes, shift+tab should go to OSDs (wrap backwards)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.activePane != LsPaneOSDs {
		t.Errorf("shift+tab from Nodes should go to OSDs, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.activePane != LsPaneDeployments {
		t.Errorf("shift+tab from OSDs should go to Deployments, got %v", model.activePane)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyShiftTab})
	if model.activePane != LsPaneNodes {
		t.Errorf("shift+tab from Deployments should go to Nodes, got %v", model.activePane)
	}
}

func TestLsModel_handleKeyPress_DeploymentsPodsToggle(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Switch to deployments pane
	model.setActivePane(LsPaneDeployments)

	// Initially showing deployments
	if model.deploymentsPodsView.IsShowingPods() {
		t.Error("should initially show deployments")
	}

	// Press ] to show pods
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if !model.deploymentsPodsView.IsShowingPods() {
		t.Error("after pressing ], should show pods")
	}

	// Press [ to show deployments
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if model.deploymentsPodsView.IsShowingPods() {
		t.Error("after pressing [, should show deployments")
	}
}

func TestLsModel_handleKeyPress_ToggleOnlyWorksOnDeploymentsPane(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Start on Nodes pane
	if model.activePane != LsPaneNodes {
		t.Fatal("expected to start on Nodes pane")
	}

	// Press ] - should not toggle since not on Deployments pane
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if model.deploymentsPodsView.IsShowingPods() {
		t.Error("] should not toggle when not on Deployments pane")
	}

	// Press [ - should not toggle since not on Deployments pane
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if model.deploymentsPodsView.IsShowingPods() {
		t.Error("[ should not toggle when not on Deployments pane")
	}
}

func TestLsModel_handleKeyPress_MaintenanceFlowOpens(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.setActivePane(LsPaneNodes)

	nodes := []k8s.NodeInfo{{Name: "node-a"}, {Name: "node-b"}}
	model.nodesView.SetNodes(nodes)
	model.nodeCount = len(nodes)
	model.nodesView.SetCursor(1)

	cmd := model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if model.maintenanceFlow == nil {
		t.Fatal("expected maintenance flow to open on 'd'")
	}
	if cmd == nil {
		t.Error("expected init command when opening flow")
	}
}

func TestLsModel_handleKeyPress_MaintenanceFlowIgnoredOutsideNodesPane(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.setActivePane(LsPaneDeployments)

	nodes := []k8s.NodeInfo{{Name: "node-a"}}
	model.nodesView.SetNodes(nodes)
	model.nodeCount = len(nodes)

	cmd := model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if cmd != nil {
		t.Error("did not expect command when not in Nodes pane")
	}
	if model.maintenanceFlow != nil {
		t.Error("did not expect flow to open outside Nodes pane")
	}
}

type testSizedModel struct{}

func (m *testSizedModel) Init() tea.Cmd { return nil }
func (m *testSizedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
func (m *testSizedModel) View() string              { return "" }
func (m *testSizedModel) SetSize(width, height int) {}

func TestLsModel_Update_MaintenanceExitClosesFlow(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.maintenanceFlow = &testSizedModel{}

	updatedModel, cmd := model.Update(DownFlowExitMsg{Reason: FlowExitDeclined})
	m, ok := updatedModel.(*LsModel)
	if !ok {
		t.Fatal("expected *LsModel type")
	}
	if m.maintenanceFlow != nil {
		t.Error("expected maintenance flow to close")
	}
	if cmd == nil {
		t.Fatal("expected refresh command after closing flow")
	}
	got := cmd()
	refreshMsg, okRefresh := got.(LsRefreshMsg)
	if !okRefresh || refreshMsg.Tab != LsTabNodes {
		t.Fatalf("expected LsRefreshMsg for nodes, got %T", got)
	}
}

func TestLsModel_reselectNodeAfterUpdate(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.setActivePane(LsPaneNodes)

	nodes := []k8s.NodeInfo{{Name: "node-a"}, {Name: "node-b"}, {Name: "node-c"}}
	model.nodesView.SetNodes(nodes)
	model.nodeCount = len(nodes)
	model.nodesView.SetCursor(2)

	model.pendingReselectNode = "node-c"
	update := &monitoring.LsMonitorUpdate{
		Nodes: []k8s.NodeInfo{
			{Name: "node-c"},
			{Name: "node-a"},
			{Name: "node-b"},
		},
	}
	model.updateFromMonitor(update)

	selected := model.nodesView.GetSelectedNode()
	if selected == nil || selected.Name != "node-c" {
		t.Errorf("expected selected node to remain node-c, got %#v", selected)
	}
	if model.pendingReselectNode != "" {
		t.Error("expected pendingReselectNode to clear after reselect")
	}
}

func TestLsModel_getMaxCursor(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Set up test data in views
	testNodes := make([]k8s.NodeInfo, 5)
	for i := 0; i < 5; i++ {
		testNodes[i] = k8s.NodeInfo{Name: fmt.Sprintf("node-%d", i)}
	}
	model.nodesView.SetNodes(testNodes)
	model.nodeCount = 5

	testDeployments := make([]k8s.DeploymentInfo, 10)
	for i := 0; i < 10; i++ {
		testDeployments[i] = k8s.DeploymentInfo{Name: fmt.Sprintf("dep-%d", i)}
	}
	model.deploymentsPodsView.SetDeployments(testDeployments)
	model.deploymentCount = 10

	testOSDs := make([]k8s.OSDInfo, 3)
	for i := 0; i < 3; i++ {
		testOSDs[i] = k8s.OSDInfo{ID: i}
	}
	model.osdsView.SetOSDs(testOSDs)
	model.osdCount = 3

	tests := []struct {
		pane     LsPane
		expected int
	}{
		{LsPaneNodes, 4},
		{LsPaneDeployments, 9},
		{LsPaneOSDs, 2},
	}

	for _, tt := range tests {
		t.Run(tt.pane.String(), func(t *testing.T) {
			model.activePane = tt.pane
			if got := model.getMaxCursor(); got != tt.expected {
				t.Errorf("getMaxCursor() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestLsModel_View(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 80
	model.height = 40

	view := model.View()

	if view == "" {
		t.Error("View should not be empty")
	}

	// View should contain pane titles
	if !contains(view, "Nodes") {
		t.Error("View should contain 'Nodes' pane")
	}

	if !contains(view, "Deployments") {
		t.Error("View should contain 'Deployments' pane")
	}

	if !contains(view, "OSDs") {
		t.Error("View should contain 'OSDs' pane")
	}
}

func TestLsModel_View_AllPanesVisible(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 100
	model.height = 50

	// Set up some test data
	model.nodesView.SetNodes([]k8s.NodeInfo{{Name: "test-node"}})
	model.deploymentsPodsView.SetDeployments([]k8s.DeploymentInfo{{Name: "test-deploy"}})
	model.osdsView.SetOSDs([]k8s.OSDInfo{{ID: 0}})
	model.updateAllCounts()

	view := model.View()

	// All three pane titles should be visible in their borders (new format: [1] Nodes)
	if !contains(view, "[1] Nodes") {
		t.Error("View should contain '[1] Nodes' in pane border")
	}
	if !contains(view, "[2] Deployments") && !contains(view, "[2] Pods") {
		t.Error("View should contain '[2] Deployments' or '[2] Pods' in pane border")
	}
	if !contains(view, "[3] OSDs") {
		t.Error("View should contain '[3] OSDs' in pane border")
	}
}

func TestLsModel_View_Help(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 80
	model.height = 40
	model.helpVisible = true

	view := model.View()

	if !contains(view, "Help") {
		t.Error("View should show help overlay")
	}

	if !contains(view, "Navigation") {
		t.Error("Help should contain Navigation section")
	}

	if !contains(view, "Switch panes") {
		t.Error("Help should mention pane switching")
	}

	if !contains(view, "deployments/pods") {
		t.Error("Help should mention deployments/pods toggle")
	}
}

func TestLsModel_View_StatusBarShowsToggleHint(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 100
	model.height = 40

	// On Nodes pane - should not show toggle hint
	model.setActivePane(LsPaneNodes)
	view := model.View()
	// The status bar should show pane hints but not the toggle hint
	if !contains(view, "Tab/1-3: pane") {
		t.Error("View should contain pane navigation hint")
	}
	if !contains(view, "u/d: up/down") {
		t.Error("View should contain maintenance hint on Nodes pane")
	}
	if contains(view, "[/]: deployments/pods") {
		t.Error("View should not contain toggle hint when on Nodes pane")
	}

	// On Deployments pane - should show toggle hint
	model.setActivePane(LsPaneDeployments)
	view = model.View()
	if !contains(view, "[/]: deployments/pods") {
		t.Error("View should contain toggle hint when on Deployments pane")
	}
}

func TestLsModel_SetSize(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	model.SetSize(100, 50)

	if model.width != 100 {
		t.Errorf("width = %d, want 100", model.width)
	}

	if model.height != 50 {
		t.Errorf("height = %d, want 50", model.height)
	}
}

func TestLsModel_Getters(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.activePane = LsPaneOSDs
	model.activeTab = LsTabOSDs
	model.cursor = 5
	model.helpVisible = true

	if model.GetActiveTab() != LsTabOSDs {
		t.Errorf("GetActiveTab() = %v, want %v", model.GetActiveTab(), LsTabOSDs)
	}

	if model.GetActivePane() != LsPaneOSDs {
		t.Errorf("GetActivePane() = %v, want %v", model.GetActivePane(), LsPaneOSDs)
	}

	if model.GetCursor() != 5 {
		t.Errorf("GetCursor() = %d, want 5", model.GetCursor())
	}

	if !model.IsHelpVisible() {
		t.Error("IsHelpVisible() should return true")
	}
}

func TestLsModel_updateActiveTab_WithCustomShowTabs(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context:  context.Background(),
		ShowTabs: []LsTab{LsTabNodes, LsTabOSDs, LsTabPods},
	})

	// Index 1 should map to OSDs (not Deployments)
	model.updateActiveTab(1)
	if model.activeTab != LsTabOSDs {
		t.Errorf("activeTab = %v, want %v", model.activeTab, LsTabOSDs)
	}

	// Index 2 should map to Pods
	model.updateActiveTab(2)
	if model.activeTab != LsTabPods {
		t.Errorf("activeTab = %v, want %v", model.activeTab, LsTabPods)
	}
}

func TestLsModel_IsShowingPods(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Initially showing deployments
	if model.IsShowingPods() {
		t.Error("should initially show deployments, not pods")
	}

	// Switch to pods
	model.deploymentsPodsView.ShowPods()
	if !model.IsShowingPods() {
		t.Error("should show pods after ShowPods()")
	}

	// Switch back to deployments
	model.deploymentsPodsView.ShowDeployments()
	if model.IsShowingPods() {
		t.Error("should show deployments after ShowDeployments()")
	}
}

func TestLsModel_CursorNavigationOnlyAffectsActivePane(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Set up test data in all views
	testNodes := make([]k8s.NodeInfo, 5)
	for i := 0; i < 5; i++ {
		testNodes[i] = k8s.NodeInfo{Name: fmt.Sprintf("node-%d", i)}
	}
	model.nodesView.SetNodes(testNodes)
	model.nodeCount = 5

	testDeployments := make([]k8s.DeploymentInfo, 5)
	for i := 0; i < 5; i++ {
		testDeployments[i] = k8s.DeploymentInfo{Name: fmt.Sprintf("dep-%d", i)}
	}
	model.deploymentsPodsView.SetDeployments(testDeployments)
	model.deploymentCount = 5

	testOSDs := make([]k8s.OSDInfo, 5)
	for i := 0; i < 5; i++ {
		testOSDs[i] = k8s.OSDInfo{ID: i}
	}
	model.osdsView.SetOSDs(testOSDs)
	model.osdCount = 5

	// Start on Nodes pane
	model.setActivePane(LsPaneNodes)

	// Move cursor down twice
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	if model.nodesView.GetCursor() != 2 {
		t.Errorf("nodes cursor should be 2, got %d", model.nodesView.GetCursor())
	}

	// Deployments cursor should still be at 0
	if model.deploymentsPodsView.GetCursor() != 0 {
		t.Errorf("deployments cursor should still be 0, got %d", model.deploymentsPodsView.GetCursor())
	}

	// Switch to deployments pane
	model.setActivePane(LsPaneDeployments)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// Deployments cursor should now be 1
	if model.deploymentsPodsView.GetCursor() != 1 {
		t.Errorf("deployments cursor should be 1, got %d", model.deploymentsPodsView.GetCursor())
	}

	// Nodes cursor should still be 2
	if model.nodesView.GetCursor() != 2 {
		t.Errorf("nodes cursor should still be 2, got %d", model.nodesView.GetCursor())
	}
}

// NOTE: contains() helper is defined in app_test.go
