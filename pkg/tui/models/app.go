// Package models provides Bubble Tea models for the TUI interface.
package models

import (
	"context"
	"fmt"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

// Route represents the current view/route in the application
type Route int

const (
	// RouteDashboard displays the cluster health dashboard
	RouteDashboard Route = iota
	// RouteDown executes the down phase workflow
	RouteDown
	// RouteUp executes the up phase workflow
	RouteUp
)

// String returns the string representation of the route
func (r Route) String() string {
	switch r {
	case RouteDashboard:
		return "dashboard"
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
}

// AppConfig holds configuration for the app model
type AppConfig struct {
	// Route determines which view to display
	Route Route

	// NodeName is the target node for operations
	NodeName string

	// StateFilePath is the optional override for state file location
	StateFilePath string

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
	dashboardModel SubModel
	downModel      SubModel
	upModel        SubModel

	// Global state
	showHelp    bool
	initError   error
	quitting    bool
	initialized bool
}

// NewAppModel creates a new app model with the given configuration
func NewAppModel(cfg AppConfig) *AppModel {
	return &AppModel{
		config: cfg,
		route:  cfg.Route,
	}
}

// Messages for internal communication

// InitCompleteMsg signals that initialization is complete
type InitCompleteMsg struct{}

// InitErrorMsg signals an initialization error
type InitErrorMsg struct {
	Err error
}

// RouteChangeMsg requests a route change
type RouteChangeMsg struct {
	Route Route
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
	return tea.Batch(
		// Get initial terminal size
		tea.EnterAltScreen,
		m.initializeSubModels,
	)
}

// initializeSubModels creates the sub-models based on the current route
func (m *AppModel) initializeSubModels() tea.Msg {
	// For now, we create placeholder models
	// These will be replaced with real implementations when their respective issues are completed
	// (crook-oro for dashboard, crook-i4e for down, crook-egi for up)

	switch m.route {
	case RouteDashboard:
		m.dashboardModel = newPlaceholderModel("Dashboard", "Cluster health dashboard coming soon...")
	case RouteDown:
		m.downModel = newPlaceholderModel("Down Phase", fmt.Sprintf("Down phase for node %s", m.config.NodeName))
	case RouteUp:
		m.upModel = newPlaceholderModel("Up Phase", fmt.Sprintf("Up phase for node %s", m.config.NodeName))
	}

	m.initialized = true
	return InitCompleteMsg{}
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
		m.propagateSizeToSubModels()

	case InitCompleteMsg:
		// Initialization complete, propagate size if we have it
		if m.width > 0 && m.height > 0 {
			m.propagateSizeToSubModels()
		}

	case InitErrorMsg:
		m.initError = msg.Err

	case RouteChangeMsg:
		m.route = msg.Route
		// Initialize the new route's model if needed
		cmds = append(cmds, m.initializeSubModels)

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	// Delegate to current sub-model if initialized
	if m.initialized && !m.showHelp {
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

	case "?":
		// Toggle help overlay
		m.showHelp = !m.showHelp
		return nil

	case "esc":
		if m.showHelp {
			// Close help if open
			m.showHelp = false
			return nil
		}
		// Otherwise let the sub-model handle it
		return nil

	case "q":
		// Quit if not in an active operation
		// Sub-models can prevent this by handling 'q' themselves
		if !m.showHelp {
			m.quitting = true
			return tea.Quit
		}
		m.showHelp = false
		return nil
	}

	return nil
}

// View implements tea.Model
func (m *AppModel) View() string {
	if m.quitting {
		return ""
	}

	// Show initialization error if any
	if m.initError != nil {
		return m.renderError()
	}

	// Show help overlay if active
	if m.showHelp {
		return m.renderHelp()
	}

	// Show loading state during initialization
	if !m.initialized {
		return m.renderLoading()
	}

	// Render current sub-model
	subModel := m.currentSubModel()
	if subModel != nil {
		return subModel.View()
	}

	return "No view available"
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

// renderHelp displays the help overlay with keyboard shortcuts
func (m *AppModel) renderHelp() string {
	helpContent := fmt.Sprintf(`%s Keyboard Shortcuts

%s Navigation
  Enter     Confirm / Proceed
  Esc       Cancel / Go back
  %s / %s     Navigate lists

%s Actions
  y / n     Answer yes/no prompts
  r         Retry failed operation
  l         Toggle log view

%s Global
  Ctrl+C    Quit immediately
  ?         Show/hide this help
  q         Quit (when safe)

Press any key to close this help.`,
		styles.StyleHeading.Render("Help"),
		styles.StyleStatus.Render(""),
		styles.IconArrow, styles.IconArrow,
		styles.StyleStatus.Render(""),
		styles.StyleStatus.Render(""))

	helpBox := styles.StyleBoxInfo.
		Width(min(60, m.width-4)).
		Render(helpContent)

	return helpBox
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
	case RouteDashboard:
		return m.dashboardModel
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
	case RouteDashboard:
		m.dashboardModel = model
	case RouteDown:
		m.downModel = model
	case RouteUp:
		m.upModel = model
	}
}

// propagateSizeToSubModels updates all sub-models with current terminal size
func (m *AppModel) propagateSizeToSubModels() {
	if m.dashboardModel != nil {
		m.dashboardModel.SetSize(m.width, m.height)
	}
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

// placeholder model for routes that aren't yet implemented

type placeholderModel struct {
	title       string
	description string
	width       int
	height      int
}

func newPlaceholderModel(title, description string) *placeholderModel {
	return &placeholderModel{
		title:       title,
		description: description,
	}
}

func (p *placeholderModel) Init() tea.Cmd {
	return nil
}

func (p *placeholderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return p, nil
}

func (p *placeholderModel) View() string {
	content := fmt.Sprintf("%s\n\n%s\n\nPress 'q' to quit or '?' for help.",
		styles.StyleHeading.Render(p.title),
		styles.StyleSubtle.Render(p.description))

	return styles.StyleBox.
		Width(min(60, p.width-4)).
		Render(content)
}

func (p *placeholderModel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
