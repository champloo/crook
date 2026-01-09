package commands_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/cmd/crook/commands"
)

func TestNewRootCmd(t *testing.T) {
	cmd := commands.NewRootCmd()

	if cmd.Use != "crook" {
		t.Errorf("expected Use to be 'crook', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestRootCmdHasGlobalFlags(t *testing.T) {
	cmd := commands.NewRootCmd()
	flags := cmd.PersistentFlags()

	expectedFlags := []string{"config", "namespace", "log-level", "log-file"}

	for _, flagName := range expectedFlags {
		if flags.Lookup(flagName) == nil {
			t.Errorf("expected global flag %q to exist", flagName)
		}
	}
}

func TestRootCmdHasVersionSubcommand(t *testing.T) {
	cmd := commands.NewRootCmd()

	var found bool
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "version" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'version' subcommand to exist")
	}
}

func TestVersionCommand(t *testing.T) {
	commands.SetVersionInfo("1.2.3", "abc123", "2024-01-01")

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"version"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "1.2.3") {
		t.Errorf("expected output to contain version '1.2.3', got %q", output)
	}
	if !strings.Contains(output, "abc123") {
		t.Errorf("expected output to contain commit 'abc123', got %q", output)
	}
}

func TestHelpCommand(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"--help"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	// Help should not return an error
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "crook") {
		t.Errorf("expected help output to contain 'crook', got %q", output)
	}
	if !strings.Contains(output, "Rook-Ceph") {
		t.Errorf("expected help output to contain 'Rook-Ceph', got %q", output)
	}
}
