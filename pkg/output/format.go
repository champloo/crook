// Package output provides CLI output formatters for the ls command.
package output

import "fmt"

// Format represents the output format type
type Format string

const (
	// FormatTable is a plain text table format
	FormatTable Format = "table"
	// FormatJSON is JSON format
	FormatJSON Format = "json"
)

// ParseFormat parses a string into a Format
func ParseFormat(s string) (Format, error) {
	switch s {
	case "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unknown output format: %s (valid formats: table, json)", s)
	}
}

// String returns the string representation of the format
func (f Format) String() string {
	return string(f)
}
