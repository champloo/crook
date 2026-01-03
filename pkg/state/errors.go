package state

import "fmt"

// ParseError wraps JSON parsing errors with optional path context.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("failed to parse state file: %v", e.Err)
	}
	return fmt.Sprintf("failed to parse state file %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ValidationError reports a structured validation failure.
type ValidationError struct {
	Path    string
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	path := ""
	if e.Path != "" {
		path = " " + e.Path
	}
	if e.Field == "" {
		return fmt.Sprintf("invalid state file%s: %s", path, e.Message)
	}
	return fmt.Sprintf("invalid state file%s: %s: %s", path, e.Field, e.Message)
}
