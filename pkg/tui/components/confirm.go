package components

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/andri/crook/pkg/tui/keys"
	"github.com/andri/crook/pkg/tui/styles"
)

// ConfirmResult represents the result of a confirmation prompt
type ConfirmResult int

const (
	// ConfirmPending indicates no answer has been given yet
	ConfirmPending ConfirmResult = iota
	// ConfirmYes indicates the user confirmed
	ConfirmYes
	// ConfirmNo indicates the user declined
	ConfirmNo
	// ConfirmCancelled indicates the user cancelled (Esc)
	ConfirmCancelled
)

// ConfirmPrompt is a yes/no confirmation prompt component
type ConfirmPrompt struct {
	// Question is the prompt text displayed to the user
	Question string

	// Details provides additional context (optional)
	Details string

	// DefaultYes makes 'y' the default when Enter is pressed
	DefaultYes bool

	// Result holds the user's answer
	Result ConfirmResult

	// ShowHint displays the "(y/N)" or "(Y/n)" hint
	ShowHint bool

	// Width for text wrapping (0 = no wrapping)
	Width int

	// keyBindings holds the keybindings for this component
	keyBindings keys.ConfirmBindings
}

// NewConfirmPrompt creates a new confirmation prompt
func NewConfirmPrompt(question string) *ConfirmPrompt {
	return &ConfirmPrompt{
		Question:    question,
		DefaultYes:  false,
		Result:      ConfirmPending,
		ShowHint:    true,
		keyBindings: keys.DefaultConfirmBindings(),
	}
}

// NewConfirmPromptWithDefault creates a confirmation prompt with a default answer
func NewConfirmPromptWithDefault(question string, defaultYes bool) *ConfirmPrompt {
	return &ConfirmPrompt{
		Question:    question,
		DefaultYes:  defaultYes,
		Result:      ConfirmPending,
		ShowHint:    true,
		keyBindings: keys.DefaultConfirmBindings(),
	}
}

// ConfirmResultMsg is sent when the user answers the prompt
type ConfirmResultMsg struct {
	Result ConfirmResult
}

// Init implements tea.Model
func (c *ConfirmPrompt) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (c *ConfirmPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if c.Result != ConfirmPending {
		// Already answered, ignore further input
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, c.keyBindings.Yes):
			c.Result = ConfirmYes
			return c, func() tea.Msg { return ConfirmResultMsg{Result: ConfirmYes} }

		case key.Matches(msg, c.keyBindings.No):
			c.Result = ConfirmNo
			return c, func() tea.Msg { return ConfirmResultMsg{Result: ConfirmNo} }

		case key.Matches(msg, c.keyBindings.Accept):
			if c.DefaultYes {
				c.Result = ConfirmYes
				return c, func() tea.Msg { return ConfirmResultMsg{Result: ConfirmYes} }
			}
			c.Result = ConfirmNo
			return c, func() tea.Msg { return ConfirmResultMsg{Result: ConfirmNo} }

		case key.Matches(msg, c.keyBindings.Cancel):
			c.Result = ConfirmCancelled
			return c, func() tea.Msg { return ConfirmResultMsg{Result: ConfirmCancelled} }
		}
	}

	return c, nil
}

// View implements tea.Model
func (c *ConfirmPrompt) View() tea.View {
	return tea.NewView(c.Render())
}

// Render returns the string representation for composition
func (c *ConfirmPrompt) Render() string {
	var result string

	// Build the question with optional hint
	question := c.Question
	if c.ShowHint && c.Result == ConfirmPending {
		hint := c.getHint()
		question = fmt.Sprintf("%s %s", question, styles.StyleSubtle.Render(hint))
	}

	// Style based on result
	switch c.Result {
	case ConfirmYes:
		result = fmt.Sprintf("%s %s",
			styles.StyleSuccess.Render(styles.IconCheckmark),
			question)
	case ConfirmNo:
		result = fmt.Sprintf("%s %s",
			styles.StyleWarning.Render(styles.IconCross),
			question)
	case ConfirmCancelled:
		result = fmt.Sprintf("%s %s",
			styles.StyleSubtle.Render("○"),
			styles.StyleSubtle.Render(question))
	case ConfirmPending:
		result = fmt.Sprintf("%s %s",
			styles.StyleStatus.Render("?"),
			question)
	}

	// Add details if present
	if c.Details != "" {
		detailText := styles.StyleSubtle.Render(c.Details)
		result = fmt.Sprintf("%s\n  %s", result, detailText)
	}

	return result
}

// getHint returns the appropriate hint based on default setting
func (c *ConfirmPrompt) getHint() string {
	if c.DefaultYes {
		return "(Y/n)"
	}
	return "(y/N)"
}

// IsAnswered returns true if the user has provided an answer
func (c *ConfirmPrompt) IsAnswered() bool {
	return c.Result != ConfirmPending
}

// IsConfirmed returns true if the user confirmed (answered yes)
func (c *ConfirmPrompt) IsConfirmed() bool {
	return c.Result == ConfirmYes
}

// IsCancelled returns true if the user cancelled the prompt
func (c *ConfirmPrompt) IsCancelled() bool {
	return c.Result == ConfirmCancelled
}

// Reset resets the prompt to pending state
func (c *ConfirmPrompt) Reset() {
	c.Result = ConfirmPending
}

// SetWidth sets the width for text wrapping
func (c *ConfirmPrompt) SetWidth(width int) {
	c.Width = width
}

// WithDetails adds details to the prompt (for chaining)
func (c *ConfirmPrompt) WithDetails(details string) *ConfirmPrompt {
	c.Details = details
	return c
}

// WithDefaultYes sets the default to yes (for chaining)
func (c *ConfirmPrompt) WithDefaultYes() *ConfirmPrompt {
	c.DefaultYes = true
	return c
}

// WithoutHint hides the (y/N) hint (for chaining)
func (c *ConfirmPrompt) WithoutHint() *ConfirmPrompt {
	c.ShowHint = false
	return c
}

// ConfirmDialog wraps a confirmation prompt in a styled box
type ConfirmDialog struct {
	prompt *ConfirmPrompt
	title  string
	width  int
}

// NewConfirmDialog creates a boxed confirmation dialog
func NewConfirmDialog(title, question string) *ConfirmDialog {
	return &ConfirmDialog{
		prompt: NewConfirmPrompt(question),
		title:  title,
		width:  50,
	}
}

// Init implements tea.Model
func (d *ConfirmDialog) Init() tea.Cmd {
	return d.prompt.Init()
}

// Update implements tea.Model
func (d *ConfirmDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := d.prompt.Update(msg)
	d.prompt, _ = newPrompt.(*ConfirmPrompt)
	return d, cmd
}

// View implements tea.Model
func (d *ConfirmDialog) View() tea.View {
	return tea.NewView(d.Render())
}

// Render returns the string representation for composition
func (d *ConfirmDialog) Render() string {
	content := ""

	if d.title != "" {
		content = styles.StyleHeading.Render(d.title) + "\n\n"
	}

	content += d.prompt.Render()

	box := styles.StyleBox.Width(d.width)

	return box.Render(content)
}

// SetWidth sets the dialog width
func (d *ConfirmDialog) SetWidth(width int) {
	d.width = width
}

// GetPrompt returns the underlying prompt
func (d *ConfirmDialog) GetPrompt() *ConfirmPrompt {
	return d.prompt
}

// IsAnswered returns true if the prompt has been answered
func (d *ConfirmDialog) IsAnswered() bool {
	return d.prompt.IsAnswered()
}

// IsConfirmed returns true if the user confirmed
func (d *ConfirmDialog) IsConfirmed() bool {
	return d.prompt.IsConfirmed()
}
