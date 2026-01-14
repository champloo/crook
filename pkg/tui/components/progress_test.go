package components

import (
	"strings"
	"testing"
)

func TestNewIndeterminateProgress(t *testing.T) {
	p := NewIndeterminateProgress("Loading")

	if !p.Indeterminate {
		t.Error("Indeterminate should be true")
	}

	if p.Label != "Loading" {
		t.Errorf("Label = %q, want %q", p.Label, "Loading")
	}
}

func TestProgressBar_SetProgress(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Indeterminate = false
	p.ShowPercentage = true
	p.Width = 40

	p.SetProgress(0.5)
	if p.Progress != 0.5 {
		t.Errorf("Progress = %f, want 0.5", p.Progress)
	}

	// Should auto-complete at 1.0
	p.SetProgress(1.0)
	if p.State != ProgressStateComplete {
		t.Error("State should be Complete at progress 1.0")
	}
}

func TestProgressBar_Complete(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Complete()

	if p.Progress != 1.0 {
		t.Errorf("Progress = %f, want 1.0", p.Progress)
	}

	if p.State != ProgressStateComplete {
		t.Error("State should be Complete")
	}
}

func TestProgressBar_Error(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Error()

	if p.State != ProgressStateError {
		t.Error("State should be Error")
	}
}

func TestProgressBar_Reset(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.SetProgress(0.5)
	p.Error()
	p.Reset()

	if p.Progress != 0 {
		t.Errorf("Progress = %f, want 0", p.Progress)
	}

	if p.State != ProgressStateInProgress {
		t.Error("State should be InProgress after reset")
	}
}

func TestProgressBar_View_Determinate(t *testing.T) {
	p := NewIndeterminateProgress("Downloading")
	p.Indeterminate = false
	p.ShowPercentage = true
	p.Width = 20
	p.SetProgress(0.5)

	view := p.Render()

	if !strings.Contains(view, "Downloading") {
		t.Error("View should contain label")
	}

	if !strings.Contains(view, "50%") {
		t.Error("View should contain percentage")
	}

	if !strings.Contains(view, progressFull) {
		t.Error("View should contain progress bar characters")
	}
}

func TestProgressBar_View_Indeterminate(t *testing.T) {
	p := NewIndeterminateProgress("Loading")

	view := p.Render()

	if !strings.Contains(view, "Loading") {
		t.Error("View should contain label")
	}

	// Should contain one of the spinner frames
	hasSpinner := false
	for _, frame := range spinnerFrames {
		if strings.Contains(view, frame) {
			hasSpinner = true
			break
		}
	}
	if !hasSpinner {
		t.Error("View should contain spinner character")
	}
}

func TestProgressBar_View_Clamping(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Indeterminate = false
	p.Width = 20
	p.ShowPercentage = false

	// Test negative progress
	p.Progress = -0.5
	view := p.Render()
	if strings.Contains(view, "-") {
		t.Error("Negative progress should be clamped to 0")
	}

	// Test progress > 1
	p.Progress = 1.5
	view = p.Render()
	fullCount := strings.Count(view, progressFull)
	emptyCount := strings.Count(view, progressEmpty)
	if emptyCount > 0 {
		t.Error("Progress > 1 should show full bar")
	}
	if fullCount != 20 {
		t.Errorf("Full bar should have 20 filled chars, got %d", fullCount)
	}
}

func TestProgressBar_Init(t *testing.T) {
	// Determinate should return nil
	p := NewIndeterminateProgress("Test")
	p.Indeterminate = false
	cmd := p.Init()
	if cmd != nil {
		t.Error("Determinate progress Init should return nil")
	}

	// Indeterminate should return tick command
	p2 := NewIndeterminateProgress("Loading")
	cmd = p2.Init()
	if cmd == nil {
		t.Error("Indeterminate progress Init should return tick command")
	}
}

func TestProgressBar_Update_SpinnerTick(t *testing.T) {
	p := NewIndeterminateProgress("Loading")
	initialFrame := p.spinnerFrame

	// Send tick message
	newModel, cmd := p.Update(SpinnerTickMsg{})
	updated, ok := newModel.(*ProgressBar)
	if !ok {
		t.Fatal("expected *ProgressBar type")
	}

	if updated.spinnerFrame == initialFrame {
		t.Error("Spinner frame should advance on tick")
	}

	if cmd == nil {
		t.Error("Should return another tick command")
	}
}

func TestProgressBar_View_SmallWidth(t *testing.T) {
	// Test that small widths with ShowPercentage=true don't panic
	smallWidths := []int{0, 1, 2, 3, 4, 5}

	for _, width := range smallWidths {
		p := NewIndeterminateProgress("Test")
		p.Indeterminate = false
		p.Width = width
		p.ShowPercentage = true
		p.Progress = 0.5

		// This should not panic
		view := p.Render()

		if view == "" {
			t.Errorf("View should not be empty for width=%d", width)
		}
	}
}

func TestProgressBar_View_ZeroWidth(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Indeterminate = false
	p.Width = 0
	p.ShowPercentage = true
	p.Progress = 0.75

	// Width 0 should default to 40, then subtract 5 for percentage = 35
	view := p.Render()

	if !strings.Contains(view, "75%") {
		t.Error("View should show 75%")
	}
}

func TestProgressBar_View_NegativeWidth(t *testing.T) {
	p := NewIndeterminateProgress("Test")
	p.Indeterminate = false
	p.Width = -10
	p.ShowPercentage = true
	p.Progress = 0.5

	// Negative width should be treated like 0 (default to 40)
	// This should not panic
	view := p.Render()

	if view == "" {
		t.Error("View should not be empty for negative width")
	}
}
