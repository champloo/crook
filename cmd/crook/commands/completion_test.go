package commands_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andri/crook/cmd/crook/commands"
)

func TestCompletionCmdExists(t *testing.T) {
	cmd := commands.NewRootCmd()

	var found bool
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "completion") {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'completion' subcommand to exist")
	}
}

func TestCompletionBash(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion", "bash"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "bash") || !strings.Contains(output, "completion") {
		t.Errorf("expected bash completion script, got %q", output[:min(100, len(output))])
	}
}

func TestCompletionZsh(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion", "zsh"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "zsh") || !strings.Contains(output, "compdef") {
		t.Errorf("expected zsh completion script, got %q", output[:min(100, len(output))])
	}
}

func TestCompletionFish(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion", "fish"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "fish") || !strings.Contains(output, "complete") {
		t.Errorf("expected fish completion script, got %q", output[:min(100, len(output))])
	}
}

func TestCompletionPowershell(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion", "powershell"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Errorf("expected powershell completion script, got %q", output[:min(100, len(output))])
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion", "invalid"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid shell type")
	}
}

func TestCompletionRequiresArg(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"completion"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no shell type provided")
	}
}
