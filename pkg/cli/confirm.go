// Package cli provides CLI utilities for non-TUI command execution.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// ConfirmOptions holds options for the confirmation prompt.
type ConfirmOptions struct {
	// Question is the prompt to display to the user.
	Question string

	// SkipPrompt skips the confirmation and returns true immediately.
	// Use this with -y/--yes flags.
	SkipPrompt bool

	// Input is the reader for user input (defaults to os.Stdin).
	Input io.Reader

	// Output is the writer for the prompt (defaults to os.Stdout).
	Output io.Writer
}

// Confirm prompts the user for confirmation with a y/n question.
// Returns true if the user confirms (y/Y/yes), false otherwise.
func Confirm(opts ConfirmOptions) (bool, error) {
	if opts.SkipPrompt {
		return true, nil
	}

	input := opts.Input
	if input == nil {
		input = os.Stdin
	}

	output := opts.Output
	if output == nil {
		output = os.Stdout
	}

	// Print the question with (y/N) suffix
	_, _ = fmt.Fprintf(output, "%s (y/N): ", opts.Question)

	// Read the response
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("failed to read input: %w", err)
		}
		// EOF without input
		return false, nil
	}

	response := strings.TrimSpace(strings.ToLower(scanner.Text()))

	switch response {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
