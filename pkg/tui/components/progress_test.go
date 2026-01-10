package components

import (
	"strings"
	"testing"
)

func TestNewProgressBar(t *testing.T) {
	p := NewProgressBar("Test")

	if p.Label != "Test" {
		t.Errorf("Label = %q, want %q", p.Label, "Test")
	}

	if p.Progress != 0 {
		t.Errorf("Progress = %f, want 0", p.Progress)
	}

	if p.State != ProgressStateInProgress {
		t.Errorf("State = %v, want ProgressStateInProgress", p.State)
	}

	if !p.ShowPercentage {
		t.Error("ShowPercentage should be true by default")
	}
}

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
	p := NewProgressBar("Test")

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
	p := NewProgressBar("Test")
	p.Complete()

	if p.Progress != 1.0 {
		t.Errorf("Progress = %f, want 1.0", p.Progress)
	}

	if p.State != ProgressStateComplete {
		t.Error("State should be Complete")
	}
}

func TestProgressBar_Error(t *testing.T) {
	p := NewProgressBar("Test")
	p.Error()

	if p.State != ProgressStateError {
		t.Error("State should be Error")
	}
}

func TestProgressBar_Reset(t *testing.T) {
	p := NewProgressBar("Test")
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
	p := NewProgressBar("Downloading")
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
	p := NewProgressBar("Test")
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
	p := NewProgressBar("Test")
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

func TestMultiProgress(t *testing.T) {
	mp := NewMultiProgress()

	bar1 := NewProgressBar("Task 1")
	bar2 := NewProgressBar("Task 2")

	mp.AddBar(bar1)
	mp.AddBar(bar2)

	if mp.Count() != 2 {
		t.Errorf("Count() = %d, want 2", mp.Count())
	}

	if mp.GetBar(0) != bar1 {
		t.Error("GetBar(0) should return bar1")
	}

	if mp.GetBar(5) != nil {
		t.Error("GetBar(5) should return nil for out of bounds")
	}

	// Test view contains both bars
	view := mp.Render()
	if !strings.Contains(view, "Task 1") {
		t.Error("View should contain Task 1")
	}
	if !strings.Contains(view, "Task 2") {
		t.Error("View should contain Task 2")
	}
}

func TestMultiProgress_SetWidth(t *testing.T) {
	mp := NewMultiProgress()
	bar := NewProgressBar("Test")
	mp.AddBar(bar)

	mp.SetWidth(60)

	if bar.Width != 60 {
		t.Errorf("Bar width = %d, want 60", bar.Width)
	}
}

func TestProgressBar_View_SmallWidth(t *testing.T) {
	// Test that small widths with ShowPercentage=true don't panic
	smallWidths := []int{0, 1, 2, 3, 4, 5}

	for _, width := range smallWidths {
		p := NewProgressBar("Test")
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
	p := NewProgressBar("Test")
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
	p := NewProgressBar("Test")
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
