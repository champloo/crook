package keys

import (
	"charm.land/bubbles/v2/key"
)

// LsPane represents the active pane in the ls view.
type LsPane int

const (
	LsPaneNodes LsPane = iota
	LsPaneDeployments
	LsPaneOSDs
)

// LsKeyMap contains all keybindings for the ls view.
type LsKeyMap struct {
	// Global bindings
	Quit key.Binding

	// Navigation
	Up   key.Binding
	Down key.Binding

	// Pane switching
	NextPane key.Binding
	PrevPane key.Binding
	Pane1    key.Binding
	Pane2    key.Binding
	Pane3    key.Binding

	// Actions
	Refresh    key.Binding
	NodeDown   key.Binding
	NodeUp     key.Binding
	ShowDeploy key.Binding
	ShowPods   key.Binding
}

// DefaultLsKeyMap returns the default ls view keybindings.
func DefaultLsKeyMap() LsKeyMap {
	return LsKeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/Esc", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab/1-3", "switch pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-Tab", "prev pane"),
		),
		Pane1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "Nodes"),
		),
		Pane2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "Deployments"),
		),
		Pane3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "OSDs"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		NodeDown: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "down node"),
		),
		NodeUp: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "up node"),
		),
		ShowDeploy: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "deployments"),
		),
		ShowPods: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "pods"),
		),
	}
}

// SetContext enables/disables contextual bindings based on active pane.
func (k *LsKeyMap) SetContext(pane LsPane, showingPods bool) {
	// Maintenance actions only available on Nodes pane
	isNodesPane := pane == LsPaneNodes
	k.NodeDown.SetEnabled(isNodesPane)
	k.NodeUp.SetEnabled(isNodesPane)

	// Toggle only available on Deployments pane
	isDeploymentsPane := pane == LsPaneDeployments
	k.ShowDeploy.SetEnabled(isDeploymentsPane && showingPods)
	k.ShowPods.SetEnabled(isDeploymentsPane && !showingPods)
}

// ShortHelp implements help.KeyMap for status bar display.
func (k LsKeyMap) ShortHelp() []key.Binding {
	bindings := []key.Binding{k.NextPane, k.Down, k.Up}

	// Add contextual bindings
	if k.NodeDown.Enabled() {
		bindings = append(bindings, k.NodeDown)
	}
	if k.NodeUp.Enabled() {
		bindings = append(bindings, k.NodeUp)
	}
	if k.ShowDeploy.Enabled() {
		bindings = append(bindings, k.ShowDeploy)
	}
	if k.ShowPods.Enabled() {
		bindings = append(bindings, k.ShowPods)
	}

	bindings = append(bindings, k.Refresh, k.Quit)
	return bindings
}

// FullHelp implements help.KeyMap.
func (k LsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPane, k.PrevPane, k.Pane1, k.Pane2, k.Pane3},
		{k.Up, k.Down},
		{k.NodeDown, k.NodeUp, k.Refresh, k.ShowDeploy, k.ShowPods},
		{k.Quit},
	}
}
