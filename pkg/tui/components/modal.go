// Package components provides reusable TUI components.
package components

import (
	"fmt"

	"github.com/andri/crook/pkg/tui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ModalContent is the interface for content hosted inside a modal.
type ModalContent interface {
	tea.Model
	SetSize(width, height int)
}

// ModalConfig holds configuration for a modal component.
type ModalConfig struct {
	Title           string
	Width           int
	Height          int
	MinWidth        int
	MinHeight       int
	MaxWidth        int
	MaxHeight       int
	DisableEscClose bool
	DisableFrame    bool
}

// ModalCloseMsg is emitted when the user requests closing the modal.
type ModalCloseMsg struct{}

// Modal renders a centered box with optional embedded content.
type Modal struct {
	config  ModalConfig
	model   ModalContent
	content string
	width   int
	height  int
}

// NewModal creates a new modal with configuration.
func NewModal(config ModalConfig) *Modal {
	return &Modal{config: config}
}

// NewModalWithModel creates a new modal hosting a model.
func NewModalWithModel(config ModalConfig, model ModalContent) *Modal {
	return &Modal{
		config: config,
		model:  model,
	}
}

// Init implements tea.Model.
func (m *Modal) Init() tea.Cmd {
	if m.model == nil {
		return nil
	}
	return m.model.Init()
}

// SetModel assigns an embedded model to the modal.
func (m *Modal) SetModel(model ModalContent) {
	m.model = model
	m.syncContentSize()
}

// SetContent assigns raw string content to the modal.
func (m *Modal) SetContent(content string) {
	m.content = content
}

// SetSize sets the terminal dimensions.
func (m *Modal) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.syncContentSize()
}

// Update handles messages and forwards them to the embedded model.
func (m *Modal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" && !m.config.DisableEscClose {
			return m, func() tea.Msg { return ModalCloseMsg{} }
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	if m.model == nil {
		return m, nil
	}

	updatedModel, cmd := m.model.Update(msg)
	if updated, ok := updatedModel.(ModalContent); ok {
		m.model = updated
		m.syncContentSize()
	}
	return m, cmd
}

// View renders the modal.
func (m *Modal) View() tea.View {
	return tea.NewView(m.Render())
}

// Render returns the string representation for composition.
func (m *Modal) Render() string {
	content := m.content
	if m.model != nil {
		// Use Render() if available, otherwise get the view content
		if renderer, ok := m.model.(interface{ Render() string }); ok {
			content = renderer.Render()
		}
	}

	if m.config.Title != "" {
		content = fmt.Sprintf("%s\n\n%s", styles.StyleHeading.Render(m.config.Title), content)
	}

	modalWidth, modalHeight := m.modalSize()
	frameW, frameH := m.frameSize()
	if modalWidth < frameW+1 {
		modalWidth = frameW + 1
	}
	if modalHeight < frameH+1 {
		modalHeight = frameH + 1
	}

	box := m.renderFrame(modalWidth, modalHeight, content)

	if m.width == 0 || m.height == 0 {
		return box
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *Modal) modalSize() (int, int) {
	width := m.config.Width
	height := m.config.Height

	if width == 0 {
		width = min(m.width-4, 80)
	}
	if height == 0 {
		height = min(m.height-4, 24)
	}

	if m.config.MaxWidth > 0 {
		width = min(width, m.config.MaxWidth)
	}
	if m.config.MaxHeight > 0 {
		height = min(height, m.config.MaxHeight)
	}
	if m.config.MinWidth > 0 {
		width = max(width, m.config.MinWidth)
	}
	if m.config.MinHeight > 0 {
		height = max(height, m.config.MinHeight)
	}

	if width < 20 {
		width = 20
	}
	if height < 8 {
		height = 8
	}
	return width, height
}

func (m *Modal) syncContentSize() {
	if m.model == nil || m.width == 0 || m.height == 0 {
		return
	}

	modalWidth, modalHeight := m.modalSize()
	frameW, frameH := m.frameSize()
	contentWidth := modalWidth - frameW
	contentHeight := modalHeight - frameH
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	m.model.SetSize(contentWidth, contentHeight)
}

func (m *Modal) frameSize() (int, int) {
	if m.config.DisableFrame {
		return 0, 0
	}
	return styles.StyleBox.GetFrameSize()
}

func (m *Modal) renderFrame(width, height int, content string) string {
	if m.config.DisableFrame {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Render(content)
	}

	return styles.StyleBox.
		Width(width).
		Height(height).
		Render(content)
}
