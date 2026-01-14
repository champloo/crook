package keys

import (
	"charm.land/bubbles/v2/key"
)

// ConfirmBindings for confirmation prompts.
type ConfirmBindings struct {
	Yes    key.Binding
	No     key.Binding
	Accept key.Binding // Enter key - uses default
	Cancel key.Binding
}

// DefaultConfirmBindings returns the default confirmation keybindings.
func DefaultConfirmBindings() ConfirmBindings {
	return ConfirmBindings{
		Yes: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("n", "no"),
		),
		Accept: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "accept default"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("Esc", "cancel"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (c ConfirmBindings) ShortHelp() []key.Binding {
	return []key.Binding{c.Yes, c.No, c.Cancel}
}

// FullHelp implements help.KeyMap.
func (c ConfirmBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{{c.Yes, c.No, c.Accept, c.Cancel}}
}

// TabBindings for tab bar navigation.
type TabBindings struct {
	Next   key.Binding
	Prev   key.Binding
	Select key.Binding // 1-9 keys
}

// DefaultTabBindings returns the default tab navigation keybindings.
func DefaultTabBindings() TabBindings {
	return TabBindings{
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next tab"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-Tab", "prev tab"),
		),
		Select: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("1-9", "select tab"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (t TabBindings) ShortHelp() []key.Binding {
	return []key.Binding{t.Next, t.Select}
}

// FullHelp implements help.KeyMap.
func (t TabBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{{t.Next, t.Prev, t.Select}}
}
