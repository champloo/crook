// Package output provides non-TUI output formatters for the ls command.
package output

import "fmt"

// Format represents the output format type
type Format string

const (
	// FormatTUI is the interactive terminal UI format
	FormatTUI Format = "tui"
	// FormatTable is a plain text table format
	FormatTable Format = "table"
	// FormatJSON is JSON format
	FormatJSON Format = "json"
	// FormatYAML is YAML format
	FormatYAML Format = "yaml"
)

// ParseFormat parses a string into a Format
func ParseFormat(s string) (Format, error) {
	switch s {
	case "tui":
		return FormatTUI, nil
	case "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown output format: %s", s)
	}
}

// IsNonTUI returns true if the format requires non-TUI output
func (f Format) IsNonTUI() bool {
	return f != FormatTUI
}

// String returns the string representation of the format
func (f Format) String() string {
	return string(f)
}
