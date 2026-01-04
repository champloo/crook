package commands_test

import (
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/cmd/crook/commands"
	"github.com/spf13/pflag"
)

func TestNewDownCmd(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the down subcommand
	var downCmd *struct {
		Use   string
		Short string
		Long  string
	}
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "down <node>" {
			downCmd = &struct {
				Use   string
				Short string
				Long  string
			}{
				Use:   subCmd.Use,
				Short: subCmd.Short,
				Long:  subCmd.Long,
			}
			break
		}
	}

	if downCmd == nil {
		t.Fatal("expected 'down' subcommand to exist")
	}

	if downCmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if !strings.Contains(downCmd.Long, "maintenance") {
		t.Error("expected Long description to mention maintenance")
	}
}

func TestDownCmdHasRequiredFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the down subcommand
	var downFlags []string
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "down") {
			subCmd.Flags().VisitAll(func(f *pflag.Flag) {
				downFlags = append(downFlags, f.Name)
			})
			break
		}
	}

	expectedFlags := []string{"state-file", "no-tui", "yes", "timeout"}

	for _, flagName := range expectedFlags {
		found := false
		for _, f := range downFlags {
			if f == flagName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected down command to have flag %q", flagName)
		}
	}
}

func TestDownCmdRequiresNodeArg(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"down"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no node argument provided")
	}

	if !strings.Contains(err.Error(), "requires") && !strings.Contains(err.Error(), "argument") {
		// Check for cobra's error message patterns
		if !strings.Contains(err.Error(), "accepts 1 arg") {
			t.Errorf("expected error about missing argument, got: %v", err)
		}
	}
}

func TestDownCmdDefaultTimeout(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the down subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "down") {
			timeoutFlag := subCmd.Flags().Lookup("timeout")
			if timeoutFlag == nil {
				t.Fatal("expected timeout flag to exist")
			}

			// Default should be 10 minutes
			expectedDefault := (10 * time.Minute).String()
			if timeoutFlag.DefValue != expectedDefault {
				t.Errorf("expected default timeout to be %s, got %s", expectedDefault, timeoutFlag.DefValue)
			}
			return
		}
	}

	t.Fatal("down subcommand not found")
}

func TestDownCmdYesShortFlag(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the down subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "down") {
			yesFlag := subCmd.Flags().ShorthandLookup("y")
			if yesFlag == nil {
				t.Fatal("expected -y shorthand for --yes flag")
			}

			if yesFlag.Name != "yes" {
				t.Errorf("expected -y to be shorthand for 'yes', got %s", yesFlag.Name)
			}
			return
		}
	}

	t.Fatal("down subcommand not found")
}
