package commands_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andri/crook/cmd/crook/commands"
)

func TestConfigCmdExists(t *testing.T) {
	cmd := commands.NewRootCmd()

	var found bool
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "config" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'config' subcommand to exist")
	}
}

func TestConfigCmdHasSubcommands(t *testing.T) {
	cmd := commands.NewRootCmd()

	var configCmd *struct {
		Commands []string
	}

	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "config" {
			var commands []string
			for _, sub := range subCmd.Commands() {
				commands = append(commands, sub.Name())
			}
			configCmd = &struct{ Commands []string }{Commands: commands}
			break
		}
	}

	if configCmd == nil {
		t.Fatal("config command not found")
	}

	expectedSubcommands := []string{"show", "validate"}
	for _, expected := range expectedSubcommands {
		found := false
		for _, actual := range configCmd.Commands {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected config subcommand %q to exist", expected)
		}
	}
}

func TestConfigShow(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "show"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	// Should contain config structure
	if !strings.Contains(output, "kubernetes") {
		t.Errorf("expected output to contain 'kubernetes', got %q", output)
	}
	if !strings.Contains(output, "rook-ceph") {
		t.Errorf("expected output to contain default namespace 'rook-ceph', got %q", output)
	}
}

func TestConfigShowJSON(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "show", "-f", "json"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Errorf("expected valid JSON output, got error: %v", unmarshalErr)
	}

	// Should have config key
	if _, ok := result["config"]; !ok {
		t.Error("expected JSON output to have 'config' key")
	}
}

func TestConfigValidateDefault(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "valid") {
		t.Errorf("expected output to contain 'valid', got %q", output)
	}
}

func TestConfigValidateJSON(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate", "-f", "json"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Errorf("expected valid JSON output, got error: %v", unmarshalErr)
	}

	// Should have valid key
	if valid, ok := result["valid"]; !ok {
		t.Error("expected JSON output to have 'valid' key")
	} else if valid != true {
		t.Errorf("expected 'valid' to be true, got %v", valid)
	}
}

func TestConfigValidateInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid config file
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")
	invalidContent := `
kubernetes:
  rook-operator-namespace: ""  # Invalid: empty namespace
`
	if err := os.WriteFile(configFile, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate", configFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	// Should return error for invalid config
	if err == nil {
		t.Error("expected error for invalid config")
	}

	output := stdout.String()
	if !strings.Contains(output, "error") || !strings.Contains(strings.ToLower(output), "namespace") {
		t.Errorf("expected output to mention namespace error, got %q", output)
	}
}

func TestConfigValidateYAML(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate", "-f", "yaml"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "valid:") {
		t.Errorf("expected YAML output to contain 'valid:', got %q", output)
	}
}

func TestConfigValidateNonexistentFile(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate", "/nonexistent/config.yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestConfigShowFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "config" {
			for _, sub := range subCmd.Commands() {
				if sub.Name() == "show" {
					formatFlag := sub.Flags().Lookup("format")
					if formatFlag == nil {
						t.Error("expected show command to have 'format' flag")
					}
					if sub.Flags().ShorthandLookup("f") == nil {
						t.Error("expected -f shorthand for format flag")
					}
					return
				}
			}
		}
	}

	t.Error("show command not found")
}

func TestConfigValidateFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "config" {
			for _, sub := range subCmd.Commands() {
				if sub.Name() == "validate" {
					formatFlag := sub.Flags().Lookup("format")
					if formatFlag == nil {
						t.Error("expected validate command to have 'format' flag")
					}
					if sub.Flags().ShorthandLookup("f") == nil {
						t.Error("expected -f shorthand for format flag")
					}
					return
				}
			}
		}
	}

	t.Error("validate command not found")
}

func TestConfigValidateWithValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid config file
	configFile := filepath.Join(tmpDir, "valid-config.yaml")
	validContent := `
kubernetes:
  rook-operator-namespace: "rook-ceph"
  rook-cluster-namespace: "rook-ceph"

state:
  file-path-template: "./crook-state-{{.Node}}.json"
  backup-enabled: true

deployment-filters:
  prefixes:
    - rook-ceph-osd
    - rook-ceph-mon

timeouts:
  api-call-timeout-seconds: 30
  wait-deployment-timeout-seconds: 300
  ceph-command-timeout-seconds: 60

logging:
  level: info
  format: text
`
	if err := os.WriteFile(configFile, []byte(validContent), 0o600); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"config", "validate", configFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error for valid config: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "valid") {
		t.Errorf("expected output to confirm valid config, got %q", output)
	}
}
