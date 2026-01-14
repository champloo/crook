package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewConfirmPrompt(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	if p.Question != "Continue?" {
		t.Errorf("Question = %q, want %q", p.Question, "Continue?")
	}

	if p.DefaultYes {
		t.Error("DefaultYes should be false by default")
	}

	if p.Result != ConfirmPending {
		t.Errorf("Result = %v, want ConfirmPending", p.Result)
	}

	if !p.ShowHint {
		t.Error("ShowHint should be true by default")
	}
}

func TestConfirmPrompt_Update_Yes(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	// Test 'y'
	newModel, cmd := p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmYes {
		t.Errorf("Result = %v, want ConfirmYes", updated.Result)
	}

	if cmd == nil {
		t.Error("Should return ConfirmResultMsg command")
	}
}

func TestConfirmPrompt_Update_No(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	newModel, cmd := p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmNo {
		t.Errorf("Result = %v, want ConfirmNo", updated.Result)
	}

	if cmd == nil {
		t.Error("Should return ConfirmResultMsg command")
	}
}

func TestConfirmPrompt_Update_Enter_DefaultNo(t *testing.T) {
	p := NewConfirmPrompt("Continue?")
	p.DefaultYes = false

	newModel, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmNo {
		t.Errorf("Result = %v, want ConfirmNo (default)", updated.Result)
	}
}

func TestConfirmPrompt_Update_Enter_DefaultYes(t *testing.T) {
	p := NewConfirmPrompt("Continue?")
	p.DefaultYes = true

	newModel, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmYes {
		t.Errorf("Result = %v, want ConfirmYes (default)", updated.Result)
	}
}

func TestConfirmPrompt_Update_Escape(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	newModel, _ := p.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmCancelled {
		t.Errorf("Result = %v, want ConfirmCancelled", updated.Result)
	}
}

func TestConfirmPrompt_Update_CtrlC(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	newModel, _ := p.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	updated, _ := newModel.(*ConfirmPrompt)

	if updated.Result != ConfirmCancelled {
		t.Errorf("Result = %v, want ConfirmCancelled", updated.Result)
	}
}

func TestConfirmPrompt_Update_AlreadyAnswered(t *testing.T) {
	p := NewConfirmPrompt("Continue?")
	p.Result = ConfirmYes

	// Should ignore further input
	newModel, cmd := p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	updated, ok := newModel.(*ConfirmPrompt)
	if !ok {
		t.Fatal("expected *ConfirmPrompt type")
	}

	if updated.Result != ConfirmYes {
		t.Error("Result should not change after already answered")
	}

	if cmd != nil {
		t.Error("Should not return command when already answered")
	}
}

func TestConfirmPrompt_View_Pending(t *testing.T) {
	p := NewConfirmPrompt("Continue?")

	view := p.Render()

	if !strings.Contains(view, "Continue?") {
		t.Error("View should contain question")
	}

	if !strings.Contains(view, "(y/N)") {
		t.Error("View should contain hint when pending")
	}
}

func TestConfirmPrompt_View_DefaultYes_Hint(t *testing.T) {
	p := NewConfirmPrompt("Continue?")
	p.DefaultYes = true

	view := p.Render()

	if !strings.Contains(view, "(Y/n)") {
		t.Error("View should show (Y/n) when default is yes")
	}
}

func TestConfirmPrompt_View_Answered(t *testing.T) {
	p := NewConfirmPrompt("Continue?")
	p.Result = ConfirmYes

	view := p.Render()

	// Should not show hint after answered
	if strings.Contains(view, "(y/N)") {
		t.Error("View should not show hint after answered")
	}
}

func TestConfirmPrompt_View_WithDetails(t *testing.T) {
	p := NewConfirmPrompt("Delete file?").WithDetails("This action cannot be undone")

	view := p.Render()

	if !strings.Contains(view, "This action cannot be undone") {
		t.Error("View should contain details")
	}
}

func TestConfirmPrompt_IsAnswered(t *testing.T) {
	p := NewConfirmPrompt("Test?")

	if p.IsAnswered() {
		t.Error("Should not be answered initially")
	}

	p.Result = ConfirmYes
	if !p.IsAnswered() {
		t.Error("Should be answered after setting result")
	}
}

func TestConfirmPrompt_IsConfirmed(t *testing.T) {
	p := NewConfirmPrompt("Test?")

	if p.IsConfirmed() {
		t.Error("Should not be confirmed initially")
	}

	p.Result = ConfirmYes
	if !p.IsConfirmed() {
		t.Error("Should be confirmed when result is Yes")
	}

	p.Result = ConfirmNo
	if p.IsConfirmed() {
		t.Error("Should not be confirmed when result is No")
	}
}

func TestConfirmPrompt_IsCancelled(t *testing.T) {
	p := NewConfirmPrompt("Test?")

	if p.IsCancelled() {
		t.Error("Should not be cancelled initially")
	}

	p.Result = ConfirmCancelled
	if !p.IsCancelled() {
		t.Error("Should be cancelled when result is Cancelled")
	}
}

func TestConfirmPrompt_Reset(t *testing.T) {
	p := NewConfirmPrompt("Test?")
	p.Result = ConfirmYes

	p.Reset()

	if p.Result != ConfirmPending {
		t.Errorf("Result = %v, want ConfirmPending after reset", p.Result)
	}
}

func TestConfirmPrompt_Chaining(t *testing.T) {
	p := NewConfirmPrompt("Test?").
		WithDetails("Details").
		WithDefaultYes().
		WithoutHint()

	if p.Details != "Details" {
		t.Error("WithDetails should set details")
	}

	if !p.DefaultYes {
		t.Error("WithDefaultYes should set default to yes")
	}

	if p.ShowHint {
		t.Error("WithoutHint should hide hint")
	}
}
