package keys

import (
	"charm.land/bubbles/v2/key"
)

// FlowBindings contains state-aware bindings for down/up maintenance flows.
type FlowBindings struct {
	Proceed   key.Binding
	Cancel    key.Binding
	Retry     key.Binding
	Exit      key.Binding
	Interrupt key.Binding
	Quit      key.Binding
}

// DefaultFlowBindings returns the default flow keybindings.
// All bindings start disabled except Interrupt; use SetStateXxx to enable.
func DefaultFlowBindings() FlowBindings {
	return FlowBindings{
		Proceed: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "proceed"),
			key.WithDisabled(),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "N", "esc"),
			key.WithHelp("n/Esc", "cancel"),
			key.WithDisabled(),
		),
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
			key.WithDisabled(),
		),
		Exit: key.NewBinding(
			key.WithKeys("enter", "q", "esc"),
			key.WithHelp("Enter/q", "exit"),
			key.WithDisabled(),
		),
		Interrupt: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Ctrl+C", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q", "quit"),
			key.WithDisabled(),
		),
	}
}

// SetStateConfirm enables bindings for confirmation state.
func (f *FlowBindings) SetStateConfirm() {
	f.disableAll()
	f.Proceed.SetEnabled(true)
	f.Cancel.SetEnabled(true)
}

// SetStateError enables bindings for error state.
func (f *FlowBindings) SetStateError() {
	f.disableAll()
	f.Retry.SetEnabled(true)
	f.Quit.SetEnabled(true)
}

// SetStateComplete enables bindings for complete state.
func (f *FlowBindings) SetStateComplete() {
	f.disableAll()
	f.Exit.SetEnabled(true)
}

// SetStateRunning enables bindings for running state.
func (f *FlowBindings) SetStateRunning() {
	f.disableAll()
	// Only Interrupt is active (always enabled)
}

// disableAll disables all state-specific bindings.
func (f *FlowBindings) disableAll() {
	f.Proceed.SetEnabled(false)
	f.Cancel.SetEnabled(false)
	f.Retry.SetEnabled(false)
	f.Exit.SetEnabled(false)
	f.Quit.SetEnabled(false)
}

// ShortHelp implements help.KeyMap.
func (f FlowBindings) ShortHelp() []key.Binding {
	var bindings []key.Binding
	for _, b := range []key.Binding{f.Proceed, f.Cancel, f.Retry, f.Exit, f.Quit, f.Interrupt} {
		if b.Enabled() {
			bindings = append(bindings, b)
		}
	}
	return bindings
}

// FullHelp implements help.KeyMap.
func (f FlowBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{f.ShortHelp()}
}
