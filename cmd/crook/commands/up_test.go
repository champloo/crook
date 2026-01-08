package commands_test

import (
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/cmd/crook/commands"
	"github.com/spf13/pflag"
)

func TestNewUpCmd(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the up subcommand
	var upCmd *struct {
		Use   string
		Short string
		Long  string
	}
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "up <node>" {
			upCmd = &struct {
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

	if upCmd == nil {
		t.Fatal("expected 'up' subcommand to exist")
	}

	if upCmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if !strings.Contains(upCmd.Long, "maintenance") {
		t.Error("expected Long description to mention maintenance")
	}
}

func TestUpCmdHasRequiredFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the up subcommand
	var upFlags []string
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "up") {
			subCmd.Flags().VisitAll(func(f *pflag.Flag) {
				upFlags = append(upFlags, f.Name)
			})
			break
		}
	}

	expectedFlags := []string{"no-tui", "yes", "timeout"}

	for _, flagName := range expectedFlags {
		found := false
		for _, f := range upFlags {
			if f == flagName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected up command to have flag %q", flagName)
		}
	}
}

func TestUpCmdRequiresNodeArg(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"up"})

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

func TestUpCmdDefaultTimeout(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the up subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "up") {
			timeoutFlag := subCmd.Flags().Lookup("timeout")
			if timeoutFlag == nil {
				t.Fatal("expected timeout flag to exist")
			}

			// Default should be 15 minutes (longer than down phase)
			expectedDefault := (15 * time.Minute).String()
			if timeoutFlag.DefValue != expectedDefault {
				t.Errorf("expected default timeout to be %s, got %s", expectedDefault, timeoutFlag.DefValue)
			}
			return
		}
	}

	t.Fatal("up subcommand not found")
}

func TestUpCmdYesShortFlag(t *testing.T) {
	cmd := commands.NewRootCmd()

	// Find the up subcommand
	for _, subCmd := range cmd.Commands() {
		if strings.HasPrefix(subCmd.Use, "up") {
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

	t.Fatal("up subcommand not found")
}
