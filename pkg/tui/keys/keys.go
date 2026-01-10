// Package keys provides centralized keybinding definitions for the TUI.
package keys

import (
	"charm.land/bubbles/v2/key"
)

// GlobalBindings are active everywhere in the application.
type GlobalBindings struct {
	Quit key.Binding
	Help key.Binding
}

// DefaultGlobalBindings returns the default global keybindings.
func DefaultGlobalBindings() GlobalBindings {
	return GlobalBindings{
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/Esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// NavigationBindings for cursor movement.
type NavigationBindings struct {
	Up     key.Binding
	Down   key.Binding
	Top    key.Binding
	Bottom key.Binding
}

// DefaultNavigationBindings returns the default navigation keybindings.
func DefaultNavigationBindings() NavigationBindings {
	return NavigationBindings{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "move down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "go to top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
	}
}

// PaneBindings for pane/tab navigation.
type PaneBindings struct {
	NextPane key.Binding
	PrevPane key.Binding
	Pane1    key.Binding
	Pane2    key.Binding
	Pane3    key.Binding
}

// DefaultPaneBindings returns the default pane navigation keybindings.
func DefaultPaneBindings() PaneBindings {
	return PaneBindings{
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next pane"),
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
	}
}
