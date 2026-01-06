package components

import (
	"testing"

	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

type mockModalContent struct {
	width  int
	height int
}

func (m *mockModalContent) Init() tea.Cmd {
	return nil
}

func (m *mockModalContent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *mockModalContent) View() string {
	return "content"
}

func (m *mockModalContent) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func TestModal_EscCloses(t *testing.T) {
	modal := NewModal(ModalConfig{})
	_, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command on Esc")
	}

	msg := cmd()
	if _, ok := msg.(ModalCloseMsg); !ok {
		t.Fatalf("expected ModalCloseMsg, got %T", msg)
	}
}

func TestModal_SyncsContentSize(t *testing.T) {
	content := &mockModalContent{}
	modal := NewModalWithModel(ModalConfig{
		Width:  40,
		Height: 10,
	}, content)

	modal.SetSize(120, 40)

	frameW, frameH := styles.StyleBox.GetFrameSize()
	wantWidth := 40 - frameW
	wantHeight := 10 - frameH
	if wantWidth < 1 {
		wantWidth = 1
	}
	if wantHeight < 1 {
		wantHeight = 1
	}

	if content.width != wantWidth {
		t.Errorf("content width = %d, want %d", content.width, wantWidth)
	}
	if content.height != wantHeight {
		t.Errorf("content height = %d, want %d", content.height, wantHeight)
	}
}
