package models

import (
	"context"
	"testing"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

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

	if model.activeTab != LsTabNodes {
		t.Errorf("initial active tab = %v, want %v", model.activeTab, LsTabNodes)
	}

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

	if model.tabBar.TabCount() != 2 {
		t.Errorf("expected 2 tabs, got %d", model.tabBar.TabCount())
	}

	if model.activeTab != LsTabNodes {
		t.Errorf("initial active tab = %v, want %v", model.activeTab, LsTabNodes)
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

	if m.activeTab != LsTabOSDs {
		t.Errorf("activeTab = %v, want %v", m.activeTab, LsTabOSDs)
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

func TestLsModel_Update_FilterMsg(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.cursor = 3

	msg := LsFilterMsg{Query: "osd"}
	updatedModel, _ := model.Update(msg)
	m, ok := updatedModel.(*LsModel)
	if !ok {
		t.Fatal("expected *LsModel type")
	}

	if m.filter != "osd" {
		t.Errorf("filter = %q, want %q", m.filter, "osd")
	}

	if m.cursor != 0 {
		t.Errorf("cursor should reset to 0 on filter change, got %d", m.cursor)
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

	// Any key should close help
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if model.helpVisible {
		t.Error("help should be hidden after pressing any key")
	}
}

func TestLsModel_handleKeyPress_Filter(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	model.handleKeyPress(msg)

	if !model.filterActive {
		t.Error("filter mode should be active after pressing /")
	}
}

func TestLsModel_handleKeyPress_Navigation(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.nodeCount = 10 // Set max for G command

	// Test j/down
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if model.cursor != 1 {
		t.Errorf("cursor after j = %d, want 1", model.cursor)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyDown})
	if model.cursor != 2 {
		t.Errorf("cursor after down = %d, want 2", model.cursor)
	}

	// Test k/up
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if model.cursor != 1 {
		t.Errorf("cursor after k = %d, want 1", model.cursor)
	}

	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyUp})
	if model.cursor != 0 {
		t.Errorf("cursor after up = %d, want 0", model.cursor)
	}

	// Test k at top (should stay at 0)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if model.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", model.cursor)
	}

	// Test g (go to top)
	model.cursor = 5
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if model.cursor != 0 {
		t.Errorf("cursor after g = %d, want 0", model.cursor)
	}

	// Test G (go to bottom)
	model.handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if model.cursor != 9 {
		t.Errorf("cursor after G = %d, want 9", model.cursor)
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

func TestLsModel_handleKeyPress_TabNavigation(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})

	// Test tab key
	msg := tea.KeyMsg{Type: tea.KeyTab}
	cmd := model.handleKeyPress(msg)

	if cmd == nil {
		t.Error("tab should return a command")
	}

	// Test number keys
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}
	cmd = model.handleKeyPress(msg)

	if cmd == nil {
		t.Error("number key should return a command")
	}
}

func TestLsModel_handleFilterInput(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.filterActive = true

	// Test typing
	model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if model.filter != "osd" {
		t.Errorf("filter = %q, want %q", model.filter, "osd")
	}

	// Test backspace
	model.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})

	if model.filter != "os" {
		t.Errorf("filter after backspace = %q, want %q", model.filter, "os")
	}

	// Test escape
	model.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})

	if model.filterActive {
		t.Error("filter should be inactive after Esc")
	}

	if model.filter != "" {
		t.Errorf("filter should be cleared after Esc, got %q", model.filter)
	}
}

func TestLsModel_handleFilterInput_Enter(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.filterActive = true
	model.filter = "test-filter"

	cmd := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})

	if model.filterActive {
		t.Error("filter mode should be inactive after Enter")
	}

	if cmd == nil {
		t.Error("Enter should return a command")
	}

	// Execute the command and verify it's a FilterMsg
	result := cmd()
	if filterMsg, ok := result.(LsFilterMsg); !ok {
		t.Error("Enter should produce LsFilterMsg")
	} else if filterMsg.Query != "test-filter" {
		t.Errorf("FilterMsg.Query = %q, want %q", filterMsg.Query, "test-filter")
	}
}

func TestLsModel_getMaxCursor(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.nodeCount = 5
	model.deploymentCount = 10
	model.osdCount = 3
	model.podCount = 0

	tests := []struct {
		tab      LsTab
		expected int
	}{
		{LsTabNodes, 4},
		{LsTabDeployments, 9},
		{LsTabOSDs, 2},
		{LsTabPods, 0},
	}

	for _, tt := range tests {
		t.Run(tt.tab.String(), func(t *testing.T) {
			model.activeTab = tt.tab
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
	model.height = 24

	view := model.View()

	if view == "" {
		t.Error("View should not be empty")
	}

	if !contains(view, "crook ls") {
		t.Error("View should contain 'crook ls' header")
	}

	if !contains(view, "Nodes") {
		t.Error("View should contain 'Nodes' tab")
	}
}

func TestLsModel_View_WithNodeFilter(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		NodeFilter: "worker-1",
		Context:    context.Background(),
	})
	model.width = 80
	model.height = 24

	view := model.View()

	if !contains(view, "worker-1") {
		t.Error("View should contain node filter")
	}
}

func TestLsModel_View_Help(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 80
	model.height = 24
	model.helpVisible = true

	view := model.View()

	if !contains(view, "Help") {
		t.Error("View should show help overlay")
	}

	if !contains(view, "Navigation") {
		t.Error("Help should contain Navigation section")
	}
}

func TestLsModel_View_FilterActive(t *testing.T) {
	model := NewLsModel(LsModelConfig{
		Context: context.Background(),
	})
	model.width = 80
	model.height = 24
	model.filterActive = true
	model.filter = "test"

	view := model.View()

	if !contains(view, "test") {
		t.Error("View should show filter input")
	}

	if !contains(view, "Enter: apply") {
		t.Error("View should show filter mode hints")
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
	model.activeTab = LsTabOSDs
	model.cursor = 5
	model.filter = "test"
	model.filterActive = true
	model.helpVisible = true

	if model.GetActiveTab() != LsTabOSDs {
		t.Errorf("GetActiveTab() = %v, want %v", model.GetActiveTab(), LsTabOSDs)
	}

	if model.GetCursor() != 5 {
		t.Errorf("GetCursor() = %d, want 5", model.GetCursor())
	}

	if model.GetFilter() != "test" {
		t.Errorf("GetFilter() = %q, want %q", model.GetFilter(), "test")
	}

	if !model.IsFilterActive() {
		t.Error("IsFilterActive() should return true")
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
