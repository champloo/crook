package models

import "strings"

// contains checks if the string s contains the substring substr.
// This is a test helper to make assertions more readable.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
