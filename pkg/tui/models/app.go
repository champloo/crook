// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/styles"
	"github.com/andri/crook/pkg/tui/terminal"
)

// Route represents the current view/route in the application
type Route int

const (
	// RouteDown executes the down phase workflow
	RouteDown Route = iota
	// RouteUp executes the up phase workflow
	RouteUp
)

// String returns the string representation of the route
func (r Route) String() string {
	switch r {
	case RouteDown:
		return "down"
	case RouteUp:
		return "up"
	default:
		return "unknown"
	}
}

// SubModel interface that all routed models must implement
type SubModel interface {
	tea.Model
	// SetSize updates the model's terminal dimensions
	SetSize(width, height int)
	// Render returns the string representation for composition
	Render() string
}

// AppConfig holds configuration for the app model
type AppConfig struct {
	// Route determines which view to display
	Route Route

	// NodeName is the target node for operations
	NodeName string

	// Config is the application configuration
	Config config.Config

	// Client is the Kubernetes client
	Client *k8s.Client

	// Context for cancellation
	Context context.Context
}

// AppModel is the main Bubble Tea model that coordinates routing and global state
type AppModel struct {
	// Configuration
	config AppConfig

	// Current route
	route Route

	// Terminal dimensions
	width  int
	height int

	// Sub-models for each route
	downModel SubModel
	upModel   SubModel

	// Global state
	sizeWarning    string
	initError      error
	quitting       bool
	initialized    bool
	termCapability terminal.Capability
}

// NewAppModel creates a new app model with the given configuration
func NewAppModel(cfg AppConfig) *AppModel {
	cap := terminal.DetectCapabilities()
	terminal.ConfigureLipgloss(cap)

	return &AppModel{
		config:         cfg,
		route:          cfg.Route,
		termCapability: cap,
	}
}

// Messages for internal communication

// SubModelsInitializedMsg carries initialized sub-models from the init command.
// This message is returned by initializeSubModels() and processed in Update()
// to avoid mutating model state inside tea.Cmd closures.
type SubModelsInitializedMsg struct {
	DownModel SubModel
	UpModel   SubModel
	Route     Route
}

// InitErrorMsg signals an initialization error
type InitErrorMsg struct {
	Err error
}

// TerminalSizeMsg updates terminal dimensions
type TerminalSizeMsg struct {
	Width  int
	Height int
}

// QuitMsg signals the application should quit
type QuitMsg struct{}

// Init implements tea.Model
func (m *AppModel) Init() tea.Cmd {
	return m.initializeSubModels
}

// initializeSubModels creates the sub-models based on the current route.
// This function returns a message carrying the created models, which are then
// assigned to the AppModel fields in Update() - following Bubble Tea's rule
// that cmds return messages and only Update() mutates model state.
func (m *AppModel) initializeSubModels() tea.Msg {
	msg := SubModelsInitializedMsg{Route: m.route}

	switch m.route {
	case RouteDown:
		msg.DownModel = NewDownModel(DownModelConfig{
			NodeName: m.config.NodeName,
			Config:   m.config.Config,
			Client:   m.config.Client,
			Context:  m.config.Context,
		})
	case RouteUp:
		msg.UpModel = NewUpModel(UpModelConfig{
			NodeName: m.config.NodeName,
			Config:   m.config.Config,
			Client:   m.config.Client,
			Context:  m.config.Context,
		})
	}

	return msg
}

// Update implements tea.Model
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd := m.handleGlobalKeys(msg)
		if cmd != nil {
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sizeWarning = terminal.SizeWarning(msg.Width, msg.Height)
		m.propagateSizeToSubModels()

	case SubModelsInitializedMsg:
		// Assign sub-models from the initialization message
		// This is where model state mutation happens, safely in Update()
		switch msg.Route {
		case RouteDown:
			m.downModel = msg.DownModel
		case RouteUp:
			m.upModel = msg.UpModel
		}
		m.initialized = true

		// Propagate size if we have it
		if m.width > 0 && m.height > 0 {
			m.propagateSizeToSubModels()
		}
		// Call the sub-model's Init() to start its operations
		subModel := m.currentSubModel()
		if subModel != nil {
			cmd := subModel.Init()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case InitErrorMsg:
		m.initError = msg.Err

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	// Delegate to current sub-model if initialized
	if m.initialized {
		subModel := m.currentSubModel()
		if subModel != nil {
			newModel, cmd := subModel.Update(msg)
			if sm, ok := newModel.(SubModel); ok {
				m.setCurrentSubModel(sm)
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// handleGlobalKeys processes global keyboard shortcuts
func (m *AppModel) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		// Graceful quit
		m.quitting = true
		return tea.Quit
	}

	return nil
}

// View implements tea.Model
func (m *AppModel) View() tea.View {
	v := tea.NewView(m.Render())
	v.AltScreen = true
	return v
}

// Render returns the string representation for composition
func (m *AppModel) Render() string {
	if m.quitting {
		return ""
	}

	// Show initialization error if any
	if m.initError != nil {
		return m.renderError()
	}

	// Show loading state during initialization
	if !m.initialized {
		return m.renderLoading()
	}

	// Build main view
	var view string

	// Add size warning if terminal is too small
	if m.sizeWarning != "" {
		view = styles.StyleWarning.Render(fmt.Sprintf("%s %s", styles.IconWarning, m.sizeWarning)) + "\n\n"
	}

	// Render current sub-model
	subModel := m.currentSubModel()
	if subModel != nil {
		view += subModel.Render()
	} else {
		view += "No view available"
	}

	return view
}

// renderError displays initialization or fatal errors
func (m *AppModel) renderError() string {
	errorBox := styles.StyleBoxError.
		Width(m.width - 4).
		Render(fmt.Sprintf("%s Error\n\n%s\n\nPress 'q' to quit.",
			styles.IconCross,
			m.initError.Error()))

	return errorBox
}

// renderLoading displays a loading indicator during initialization
func (m *AppModel) renderLoading() string {
	return styles.StyleBox.Render(
		fmt.Sprintf("%s Initializing...", styles.IconSpinner),
	)
}

// currentSubModel returns the sub-model for the current route
func (m *AppModel) currentSubModel() SubModel {
	switch m.route {
	case RouteDown:
		return m.downModel
	case RouteUp:
		return m.upModel
	default:
		return nil
	}
}

// setCurrentSubModel updates the sub-model for the current route
func (m *AppModel) setCurrentSubModel(model SubModel) {
	switch m.route {
	case RouteDown:
		m.downModel = model
	case RouteUp:
		m.upModel = model
	}
}

// propagateSizeToSubModels updates all sub-models with current terminal size
func (m *AppModel) propagateSizeToSubModels() {
	if m.downModel != nil {
		m.downModel.SetSize(m.width, m.height)
	}
	if m.upModel != nil {
		m.upModel.SetSize(m.width, m.height)
	}
}

// GetRoute returns the current route
func (m *AppModel) GetRoute() Route {
	return m.route
}

// GetTerminalSize returns the current terminal dimensions
func (m *AppModel) GetTerminalSize() (width, height int) {
	return m.width, m.height
}

// IsInitialized returns whether the app has completed initialization
func (m *AppModel) IsInitialized() bool {
	return m.initialized
}
