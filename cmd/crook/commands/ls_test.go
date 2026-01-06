package commands_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/cmd/crook/commands"
	"github.com/spf13/pflag"
)

func TestNewLsCmd(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the ls subcommand
	var found bool
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "ls [node-name]" {
			found = true

			if subCmd.Short == "" {
				t.Error("expected Short description to be set")
			}

			if !strings.Contains(subCmd.Long, "Rook-Ceph") {
				t.Error("expected Long description to mention Rook-Ceph")
			}
			break
		}
	}

	if !found {
		t.Fatal("expected 'ls' subcommand to exist")
	}
}

func TestLsCmdHasRequiredFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the ls subcommand
	var lsFlags []string
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "ls") {
			subCmd.Flags().VisitAll(func(f *pflag.Flag) {
				lsFlags = append(lsFlags, f.Name)
			})
			break
		}
	}

	expectedFlags := []string{"output", "show"}

	for _, flagName := range expectedFlags {
		found := false
		for _, f := range lsFlags {
			if f == flagName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected ls command to have flag %q", flagName)
		}
	}
}

func TestLsCmdShorthandFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the ls subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "ls") {
			// Test -o shorthand for --output
			outputFlag := subCmd.Flags().ShorthandLookup("o")
			if outputFlag == nil {
				t.Error("expected -o shorthand for --output flag")
			} else if outputFlag.Name != "output" {
				t.Errorf("expected -o to be shorthand for 'output', got %s", outputFlag.Name)
			}

			return
		}
	}

	t.Fatal("ls subcommand not found")
}

func TestLsCmdDefaultValues(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the ls subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "ls") {
			// Test --output default (tui)
			outputFlag := subCmd.Flags().Lookup("output")
			if outputFlag == nil {
				t.Fatal("expected output flag to exist")
			}
			if outputFlag.DefValue != "tui" {
				t.Errorf("expected default output to be 'tui', got %s", outputFlag.DefValue)
			}

			return
		}
	}

	t.Fatal("ls subcommand not found")
}

func TestLsCmdAcceptsOptionalNodeArg(t *testing.T) {
	// Test that ls works without node arg (should not error on arg parsing)
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"ls"})

	// Execute the command - it should fail connecting to K8s (no valid kubeconfig)
	// but NOT fail on argument validation
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error (invalid kubeconfig), got nil")
		return
	}

	// Should NOT be an argument error (e.g., "accepts at most 1 arg")
	if strings.Contains(err.Error(), "accepts") && strings.Contains(err.Error(), "arg") {
		t.Errorf("unexpected argument validation error: %v", err)
	}
}

func TestLsCmdValidatesOutputFlag(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantError bool
		errorMsg  string
	}{
		{"valid tui", "tui", false, ""},
		{"valid table", "table", false, ""},
		{"valid json", "json", false, ""},
		{"valid yaml", "yaml", false, ""},
		{"invalid format", "csv", true, "must be one of"},
		{"invalid empty after set", "invalid", true, "must be one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			cmd.SetArgs([]string{"ls", "--output", tt.output})

			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for output=%q, got nil", tt.output)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				// Not implemented error is expected, but not validation error
				if err != nil && strings.Contains(err.Error(), "must be one of") {
					t.Errorf("unexpected validation error for valid output %q: %v", tt.output, err)
				}
			}
		})
	}
}

func TestLsCmdValidatesShowFlag(t *testing.T) {
	tests := []struct {
		name      string
		show      string
		wantError bool
		errorMsg  string
	}{
		{"valid nodes", "nodes", false, ""},
		{"valid osds", "osds", false, ""},
		{"valid multiple", "nodes,deployments,osds", false, ""},
		{"valid all", "nodes,deployments,osds,pods", false, ""},
		{"invalid type", "services", true, "is invalid"},
		{"invalid mixed", "nodes,invalid", true, "is invalid"},
		{"empty string", "", false, ""}, // Empty is allowed (show all)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := commands.NewRootCmd()
			args := []string{"ls"}
			if tt.show != "" {
				args = append(args, "--show", tt.show)
			}
			cmd.SetArgs(args)

			err := cmd.Execute()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for show=%q, got nil", tt.show)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				// Not implemented error is expected, but not validation error
				if err != nil && strings.Contains(err.Error(), "is invalid") {
					t.Errorf("unexpected validation error for valid show %q: %v", tt.show, err)
				}
			}
		})
	}
}

func TestLsCmdHelp(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"ls", "--help"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify help shows all flags
	expectedInHelp := []string{
		"--output",
		"--show",
		"-o",
	}

	for _, expected := range expectedInHelp {
		if !strings.Contains(output, expected) {
			t.Errorf("expected help to contain %q", expected)
		}
	}
}
