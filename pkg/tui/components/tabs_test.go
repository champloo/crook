package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewTabBar(t *testing.T) {
	tabs := []Tab{
		{Title: "Nodes", ShortcutKey: "1"},
		{Title: "Deployments", ShortcutKey: "2"},
		{Title: "OSDs", ShortcutKey: "3"},
		{Title: "Pods", ShortcutKey: "4"},
	}

	tabBar := NewTabBar(tabs)

	if tabBar == nil {
		t.Fatal("NewTabBar returned nil")
	}

	if len(tabBar.Tabs) != 4 {
		t.Errorf("expected 4 tabs, got %d", len(tabBar.Tabs))
	}

	if tabBar.ActiveTab != 0 {
		t.Errorf("expected initial active tab to be 0, got %d", tabBar.ActiveTab)
	}
}

func TestTabBar_Init(t *testing.T) {
	tabBar := NewTabBar([]Tab{{Title: "Test"}})
	cmd := tabBar.Init()

	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestTabBar_Update_Tab(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)

	// Press Tab to go to next tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
	_, cmd := tabBar.Update(msg)

	if cmd == nil {
		t.Fatal("expected command from Tab key")
	}

	// Execute the command and get the message
	result := cmd()
	tabSwitchMsg, ok := result.(TabSwitchMsg)
	if !ok {
		t.Fatal("expected TabSwitchMsg")
	}
	if tabSwitchMsg.Index != 1 {
		t.Errorf("expected tab switch to index 1, got %d", tabSwitchMsg.Index)
	}
}

func TestTabBar_Update_ShiftTab(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)
	tabBar.ActiveTab = 1

	// Press Shift+Tab to go to previous tab
	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	_, cmd := tabBar.Update(msg)

	if cmd == nil {
		t.Fatal("expected command from Shift+Tab key")
	}

	result := cmd()
	tabSwitchMsg, ok := result.(TabSwitchMsg)
	if !ok {
		t.Fatal("expected TabSwitchMsg")
	}
	if tabSwitchMsg.Index != 0 {
		t.Errorf("expected tab switch to index 0, got %d", tabSwitchMsg.Index)
	}
}

func TestTabBar_Update_NumberKeys(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1", ShortcutKey: "1"},
		{Title: "Tab2", ShortcutKey: "2"},
		{Title: "Tab3", ShortcutKey: "3"},
	}
	tabBar := NewTabBar(tabs)

	tests := []struct {
		key      string
		expected int
	}{
		{"1", 0},
		{"2", 1},
		{"3", 2},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			_, cmd := tabBar.Update(msg)

			if cmd == nil {
				t.Fatal("expected command from number key")
			}

			result := cmd()
			tabSwitchMsg, ok := result.(TabSwitchMsg)
			if !ok {
				t.Fatal("expected TabSwitchMsg")
			}
			if tabSwitchMsg.Index != tt.expected {
				t.Errorf("expected tab switch to index %d, got %d", tt.expected, tabSwitchMsg.Index)
			}
		})
	}
}

func TestTabBar_Update_InvalidNumberKey(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
	}
	tabBar := NewTabBar(tabs)

	// Press "5" which is out of range
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")}
	_, cmd := tabBar.Update(msg)

	if cmd != nil {
		t.Error("expected no command for out-of-range number key")
	}
}

func TestTabBar_Update_TabSwitchMsg(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)

	msg := TabSwitchMsg{Index: 2}
	tabBar.Update(msg)

	if tabBar.ActiveTab != 2 {
		t.Errorf("expected active tab to be 2, got %d", tabBar.ActiveTab)
	}
}

func TestTabBar_Update_TabSwitchMsg_OutOfRange(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
	}
	tabBar := NewTabBar(tabs)

	// Invalid index should be ignored
	msg := TabSwitchMsg{Index: 10}
	tabBar.Update(msg)

	if tabBar.ActiveTab != 0 {
		t.Errorf("expected active tab to remain 0, got %d", tabBar.ActiveTab)
	}
}

func TestTabBar_View(t *testing.T) {
	tabs := []Tab{
		{Title: "Nodes", ShortcutKey: "1"},
		{Title: "Deployments", ShortcutKey: "2"},
	}
	tabBar := NewTabBar(tabs)

	view := tabBar.View()

	if view == "" {
		t.Error("View should not be empty")
	}

	// Check that tab titles are present
	if !containsString(view, "Nodes") {
		t.Error("View should contain 'Nodes'")
	}

	if !containsString(view, "Deployments") {
		t.Error("View should contain 'Deployments'")
	}
}

func TestTabBar_View_WithBadge(t *testing.T) {
	tabs := []Tab{
		{Title: "Nodes", ShortcutKey: "1", Badge: "5"},
	}
	tabBar := NewTabBar(tabs)

	view := tabBar.View()

	if !containsString(view, "5") {
		t.Error("View should contain badge '5'")
	}
}

func TestTabBar_View_Empty(t *testing.T) {
	tabBar := NewTabBar([]Tab{})

	view := tabBar.View()

	if view != "" {
		t.Errorf("View should be empty for no tabs, got %q", view)
	}
}

func TestTabBar_nextTab_Wraps(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)
	tabBar.ActiveTab = 2 // Last tab

	cmd := tabBar.nextTab()
	result := cmd()
	tabSwitchMsg, ok := result.(TabSwitchMsg)
	if !ok {
		t.Fatal("expected TabSwitchMsg")
	}

	if tabSwitchMsg.Index != 0 {
		t.Errorf("expected wrap to index 0, got %d", tabSwitchMsg.Index)
	}
}

func TestTabBar_prevTab_Wraps(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)
	tabBar.ActiveTab = 0 // First tab

	cmd := tabBar.prevTab()
	result := cmd()
	tabSwitchMsg, ok := result.(TabSwitchMsg)
	if !ok {
		t.Fatal("expected TabSwitchMsg")
	}

	if tabSwitchMsg.Index != 2 {
		t.Errorf("expected wrap to index 2, got %d", tabSwitchMsg.Index)
	}
}

func TestTabBar_SetActiveTab(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
	}
	tabBar := NewTabBar(tabs)

	tabBar.SetActiveTab(1)
	if tabBar.ActiveTab != 1 {
		t.Errorf("expected active tab to be 1, got %d", tabBar.ActiveTab)
	}

	// Out of range should be ignored
	tabBar.SetActiveTab(10)
	if tabBar.ActiveTab != 1 {
		t.Errorf("expected active tab to remain 1, got %d", tabBar.ActiveTab)
	}

	// Negative should be ignored
	tabBar.SetActiveTab(-1)
	if tabBar.ActiveTab != 1 {
		t.Errorf("expected active tab to remain 1, got %d", tabBar.ActiveTab)
	}
}

func TestTabBar_GetActiveTab(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
	}
	tabBar := NewTabBar(tabs)
	tabBar.ActiveTab = 1

	if tabBar.GetActiveTab() != 1 {
		t.Errorf("expected GetActiveTab to return 1, got %d", tabBar.GetActiveTab())
	}
}

func TestTabBar_SetBadge(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
	}
	tabBar := NewTabBar(tabs)

	tabBar.SetBadge(0, "10")
	if tabBar.Tabs[0].Badge != "10" {
		t.Errorf("expected badge to be '10', got %q", tabBar.Tabs[0].Badge)
	}

	// Out of range should be ignored
	tabBar.SetBadge(10, "invalid")
	// Should not panic
}

func TestTabBar_SetWidth(t *testing.T) {
	tabBar := NewTabBar([]Tab{{Title: "Test"}})

	tabBar.SetWidth(100)
	if tabBar.Width != 100 {
		t.Errorf("expected width to be 100, got %d", tabBar.Width)
	}
}

func TestTabBar_TabCount(t *testing.T) {
	tabs := []Tab{
		{Title: "Tab1"},
		{Title: "Tab2"},
		{Title: "Tab3"},
	}
	tabBar := NewTabBar(tabs)

	if tabBar.TabCount() != 3 {
		t.Errorf("expected TabCount to return 3, got %d", tabBar.TabCount())
	}
}

// containsString is a helper to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || containsString(s[1:], substr)))
}
