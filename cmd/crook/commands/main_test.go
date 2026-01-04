package commands_test

import (
	"os"
	"testing"
)

// TestMain runs before all tests in this package - isolate from environment
func TestMain(m *testing.M) {
	// Set invalid KUBECONFIG to prevent tests from connecting to real cluster
	// This applies to ALL test files in the commands_test package
	if err := os.Setenv("KUBECONFIG", "/nonexistent/test-kubeconfig"); err != nil {
		panic("failed to set test KUBECONFIG: " + err.Error())
	}
	os.Exit(m.Run())
}
