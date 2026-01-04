package components_test

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestFilterBar_NewFilterBar(t *testing.T) {
	fb := components.NewFilterBar()
	if fb == nil {
		t.Fatal("NewFilterBar() returned nil")
	}
	if fb.IsActive() {
		t.Error("NewFilterBar() should not be active by default")
	}
	if fb.Query() != "" {
		t.Error("NewFilterBar() should have empty query")
	}
}

func TestFilterBar_Activate(t *testing.T) {
	fb := components.NewFilterBar()

	fb.Activate()

	if !fb.IsActive() {
		t.Error("Activate() should make filter bar active")
	}
}

func TestFilterBar_Deactivate(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.Deactivate()

	if fb.IsActive() {
		t.Error("Deactivate() should make filter bar inactive")
	}
}

func TestFilterBar_TypeInput(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()

	// Type some characters
	fb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}})

	if fb.Query() != "test" {
		t.Errorf("Query() = %q, want %q", fb.Query(), "test")
	}
}

func TestFilterBar_Backspace(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	fb.Update(tea.KeyMsg{Type: tea.KeyBackspace})

	if fb.Query() != "tes" {
		t.Errorf("Query() after backspace = %q, want %q", fb.Query(), "tes")
	}
}

func TestFilterBar_Enter_AppliesFilter(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	_, cmd := fb.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if fb.IsActive() {
		t.Error("Enter should deactivate filter bar")
	}
	if !fb.HasFilter() {
		t.Error("Enter should apply filter")
	}

	// Check command returns FilterAppliedMsg
	if cmd == nil {
		t.Fatal("Enter should return a command")
	}

	// Execute batch to check messages
	msgs := collectBatchMsgs(cmd)
	hasApplied := false
	hasExit := false
	for _, m := range msgs {
		if _, ok := m.(components.FilterAppliedMsg); ok {
			hasApplied = true
		}
		if _, ok := m.(components.FilterModeExitMsg); ok {
			hasExit = true
		}
	}
	if !hasApplied {
		t.Error("Enter should send FilterAppliedMsg")
	}
	if !hasExit {
		t.Error("Enter should send FilterModeExitMsg")
	}
}

func TestFilterBar_Esc_ClearsFilter(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	_, cmd := fb.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if fb.IsActive() {
		t.Error("Esc should deactivate filter bar")
	}
	if fb.Query() != "" {
		t.Error("Esc should clear query")
	}
	if fb.HasFilter() {
		t.Error("Esc should clear filter")
	}

	// Check command returns FilterClearedMsg
	if cmd == nil {
		t.Fatal("Esc should return a command")
	}

	msgs := collectBatchMsgs(cmd)
	hasCleared := false
	for _, m := range msgs {
		if _, ok := m.(components.FilterClearedMsg); ok {
			hasCleared = true
		}
	}
	if !hasCleared {
		t.Error("Esc should send FilterClearedMsg")
	}
}

func TestFilterBar_CtrlU_ClearsQueryKeepsMode(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	_, cmd := fb.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	if !fb.IsActive() {
		t.Error("Ctrl+U should keep filter bar active")
	}
	if fb.Query() != "" {
		t.Error("Ctrl+U should clear query")
	}

	// Check command returns FilterChangedMsg
	if cmd == nil {
		t.Fatal("Ctrl+U should return a command")
	}

	msg := cmd()
	if _, ok := msg.(components.FilterChangedMsg); !ok {
		t.Error("Ctrl+U should send FilterChangedMsg")
	}
}

func TestFilterBar_View_WhenInactive(t *testing.T) {
	fb := components.NewFilterBar()

	view := fb.View()

	if view != "" {
		t.Error("View() should return empty string when inactive")
	}
}

func TestFilterBar_View_WhenActive(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	view := fb.View()

	if !strings.Contains(view, "/") {
		t.Error("View() should contain prompt")
	}
	if !strings.Contains(view, "test") {
		t.Error("View() should contain query")
	}
}

func TestFilterBar_ViewStatus_NoFilter(t *testing.T) {
	fb := components.NewFilterBar()

	status := fb.ViewStatus()

	if status != "" {
		t.Error("ViewStatus() should be empty when no filter applied")
	}
}

func TestFilterBar_ViewStatus_WithFilter(t *testing.T) {
	fb := components.NewFilterBar()
	fb.SetQuery("test")
	fb.Activate()
	fb.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Apply filter

	status := fb.ViewStatus()

	if !strings.Contains(status, "test") {
		t.Error("ViewStatus() should contain query")
	}
}

func TestFilterBar_FilterStatusText(t *testing.T) {
	fb := components.NewFilterBar()
	fb.SetQuery("test")
	fb.Activate()
	fb.Update(tea.KeyMsg{Type: tea.KeyEnter})

	text := fb.FilterStatusText(5, 20)

	if !strings.Contains(text, "test") {
		t.Error("FilterStatusText() should contain query")
	}
	if !strings.Contains(text, "5/20") {
		t.Error("FilterStatusText() should contain counts")
	}
}

func TestFilterBar_Clear(t *testing.T) {
	fb := components.NewFilterBar()
	fb.SetQuery("test")
	fb.Activate()
	fb.Update(tea.KeyMsg{Type: tea.KeyEnter})

	fb.Clear()

	if fb.Query() != "" {
		t.Error("Clear() should clear query")
	}
	if fb.HasFilter() {
		t.Error("Clear() should clear filter")
	}
}

func TestFilterBar_SetQuery(t *testing.T) {
	fb := components.NewFilterBar()

	fb.SetQuery("new query")

	if fb.Query() != "new query" {
		t.Errorf("Query() = %q, want %q", fb.Query(), "new query")
	}
}

func TestFilterBar_CursorNavigation(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	// Move cursor left
	fb.Update(tea.KeyMsg{Type: tea.KeyLeft})
	fb.Update(tea.KeyMsg{Type: tea.KeyLeft})

	// Insert character in middle
	fb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	if fb.Query() != "teXst" {
		t.Errorf("Query() = %q, want %q", fb.Query(), "teXst")
	}
}

func TestFilterBar_Home_End(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	// Go to home and insert
	fb.Update(tea.KeyMsg{Type: tea.KeyHome})
	fb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	if fb.Query() != "Xtest" {
		t.Errorf("Query() after Home = %q, want %q", fb.Query(), "Xtest")
	}

	// Go to end and insert
	fb.Update(tea.KeyMsg{Type: tea.KeyEnd})
	fb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	if fb.Query() != "XtestY" {
		t.Errorf("Query() after End = %q, want %q", fb.Query(), "XtestY")
	}
}

func TestFilterBar_Delete(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("test")

	// Go to beginning and delete
	fb.Update(tea.KeyMsg{Type: tea.KeyHome})
	fb.Update(tea.KeyMsg{Type: tea.KeyDelete})

	if fb.Query() != "est" {
		t.Errorf("Query() after Delete = %q, want %q", fb.Query(), "est")
	}
}

func TestFilterBar_Space(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("hello")

	fb.Update(tea.KeyMsg{Type: tea.KeySpace})

	if fb.Query() != "hello " {
		t.Errorf("Query() = %q, want %q", fb.Query(), "hello ")
	}
}

func TestFilterBar_CtrlW_DeleteWord(t *testing.T) {
	fb := components.NewFilterBar()
	fb.Activate()
	fb.SetQuery("hello world")

	fb.Update(tea.KeyMsg{Type: tea.KeyCtrlW})

	if fb.Query() != "hello " {
		t.Errorf("Query() after Ctrl+W = %q, want %q", fb.Query(), "hello ")
	}
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		s     string
		query string
		want  bool
	}{
		{"test", "", true},     // Empty query matches all
		{"test", "es", true},   // Partial match
		{"TEST", "es", true},   // Case insensitive
		{"test", "ES", true},   // Case insensitive query
		{"test", "xyz", false}, // No match
		{"rook-ceph-osd-0", "osd", true},
		{"rook-ceph-mon-a", "osd", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.query, func(t *testing.T) {
			got := components.MatchesFilter(tt.s, tt.query)
			if got != tt.want {
				t.Errorf("MatchesFilter(%q, %q) = %v, want %v", tt.s, tt.query, got, tt.want)
			}
		})
	}
}

func TestFilterBar_UpdateWhenInactive(t *testing.T) {
	fb := components.NewFilterBar()

	// Should not process keys when inactive
	fb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if fb.Query() != "" {
		t.Error("Should not process keys when inactive")
	}
}

func TestFilterBar_HasFilter(t *testing.T) {
	fb := components.NewFilterBar()

	if fb.HasFilter() {
		t.Error("HasFilter() should be false initially")
	}

	fb.SetQuery("test")
	fb.Activate()
	fb.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !fb.HasFilter() {
		t.Error("HasFilter() should be true after applying filter")
	}

	fb.Clear()

	if fb.HasFilter() {
		t.Error("HasFilter() should be false after Clear()")
	}
}

// collectBatchMsgs collects messages from a batch command
func collectBatchMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()

	// Check if it's a batch
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batchMsg {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
		return msgs
	}

	return []tea.Msg{msg}
}
