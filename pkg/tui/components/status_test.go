package components

import (
	"strings"
	"testing"

	"github.com/andri/crook/pkg/tui/styles"
)

func TestNewStatusIndicator(t *testing.T) {
	s := NewStatusIndicator("Test", StatusTypeInfo)

	if s.Label != "Test" {
		t.Errorf("Label = %q, want %q", s.Label, "Test")
	}

	if s.Type != StatusTypeInfo {
		t.Errorf("Type = %v, want StatusTypeInfo", s.Type)
	}

	if !s.ShowIcon {
		t.Error("ShowIcon should be true by default")
	}

	if !s.Inline {
		t.Error("Inline should be true by default")
	}
}

func TestStatusIndicator_Constructors(t *testing.T) {
	tests := []struct {
		name         string
		constructor  func(string) *StatusIndicator
		expectedType StatusType
	}{
		{"Info", NewInfoStatus, StatusTypeInfo},
		{"Success", NewSuccessStatus, StatusTypeSuccess},
		{"Warning", NewWarningStatus, StatusTypeWarning},
		{"Error", NewErrorStatus, StatusTypeError},
		{"Pending", NewPendingStatus, StatusTypePending},
		{"Running", NewRunningStatus, StatusTypeRunning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.constructor("Test")
			if s.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", s.Type, tt.expectedType)
			}
		})
	}
}

func TestStatusIndicator_View_WithIcon(t *testing.T) {
	s := NewSuccessStatus("Operation complete")

	view := s.Render()

	if !strings.Contains(view, "Operation complete") {
		t.Error("View should contain label")
	}

	if !strings.Contains(view, styles.IconCheckmark) {
		t.Error("View should contain success icon")
	}
}

func TestStatusIndicator_View_WithDetails(t *testing.T) {
	s := NewInfoStatus("Status").WithDetails("Additional info")

	view := s.Render()

	if !strings.Contains(view, "Status") {
		t.Error("View should contain label")
	}

	if !strings.Contains(view, "Additional info") {
		t.Error("View should contain details")
	}
}

func TestStatusIndicator_View_WithoutIcon(t *testing.T) {
	s := NewInfoStatus("Status").WithoutIcon()

	view := s.Render()

	if !strings.Contains(view, "Status") {
		t.Error("View should contain label")
	}

	if strings.Contains(view, styles.IconInfo) {
		t.Error("View should not contain icon")
	}
}

func TestStatusIndicator_GetIcon(t *testing.T) {
	tests := []struct {
		statusType   StatusType
		expectedIcon string
	}{
		{StatusTypeSuccess, styles.IconCheckmark},
		{StatusTypeWarning, styles.IconWarning},
		{StatusTypeError, styles.IconCross},
		{StatusTypeInfo, styles.IconInfo},
		{StatusTypePending, "○"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedIcon, func(t *testing.T) {
			s := NewStatusIndicator("Test", tt.statusType)
			icon := s.getIcon()
			if icon != tt.expectedIcon {
				t.Errorf("getIcon() = %q, want %q", icon, tt.expectedIcon)
			}
		})
	}
}

func TestStatusIndicator_SetMethods(t *testing.T) {
	s := NewInfoStatus("Original")

	s.SetType(StatusTypeError)
	if s.Type != StatusTypeError {
		t.Error("SetType should update type")
	}

	s.SetLabel("Updated")
	if s.Label != "Updated" {
		t.Error("SetLabel should update label")
	}

	s.SetDetails("Details")
	if s.Details != "Details" {
		t.Error("SetDetails should update details")
	}
}

func TestStatusIndicator_Running_Spinner(t *testing.T) {
	s := NewRunningStatus("Processing")

	// Should return tick command
	cmd := s.Init()
	if cmd == nil {
		t.Error("Running status should return tick command")
	}

	// View should show spinner frame
	view := s.Render()
	hasSpinner := false
	for _, frame := range spinnerFrames {
		if strings.Contains(view, frame) {
			hasSpinner = true
			break
		}
	}
	if !hasSpinner {
		t.Error("View should contain spinner frame")
	}
}

func TestStatusIndicator_Update_Tick(t *testing.T) {
	s := NewRunningStatus("Processing")
	initialFrame := s.spinnerFrame

	// Send tick
	newModel, cmd := s.Update(StatusTickMsg{})
	updated, ok := newModel.(*StatusIndicator)
	if !ok {
		t.Fatal("expected *StatusIndicator type")
	}

	if updated.spinnerFrame == initialFrame {
		t.Error("Spinner frame should advance")
	}

	if cmd == nil {
		t.Error("Should return another tick command")
	}
}

func TestStatusList(t *testing.T) {
	list := NewStatusList()

	list.Add(NewSuccessStatus("Step 1"))
	list.Add(NewRunningStatus("Step 2"))
	list.AddStatus("Step 3", StatusTypePending)

	if list.Count() != 3 {
		t.Errorf("Count() = %d, want 3", list.Count())
	}

	// Test Get
	item := list.Get(1)
	if item.Type != StatusTypeRunning {
		t.Error("Get(1) should return running status")
	}

	if list.Get(10) != nil {
		t.Error("Get(10) should return nil for out of bounds")
	}

	// Test View
	view := list.Render()
	if !strings.Contains(view, "Step 1") {
		t.Error("View should contain Step 1")
	}
	if !strings.Contains(view, "Step 2") {
		t.Error("View should contain Step 2")
	}
	if !strings.Contains(view, "Step 3") {
		t.Error("View should contain Step 3")
	}
}

func TestJoinLines(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b"}, "a\nb"},
		{[]string{"a", "b", "c"}, "a\nb\nc"},
	}

	for _, tt := range tests {
		result := joinLines(tt.input)
		if result != tt.expected {
			t.Errorf("joinLines(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
