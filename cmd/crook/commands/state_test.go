package commands_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/cmd/crook/commands"
	"github.com/spf13/pflag"
)

func TestStateCmdExists(t *testing.T) {
	cmd := commands.NewRootCmd()

	var found bool
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "state" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'state' subcommand to exist")
	}
}

func TestStateCmdHasSubcommands(t *testing.T) {
	cmd := commands.NewRootCmd()

	var stateCmd *struct {
		Commands []string
	}

	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "state" {
			var commands []string
			for _, sub := range subCmd.Commands() {
				commands = append(commands, sub.Name())
			}
			stateCmd = &struct{ Commands []string }{Commands: commands}
			break
		}
	}

	if stateCmd == nil {
		t.Fatal("state command not found")
	}

	expectedSubcommands := []string{"list", "show", "clean"}
	for _, expected := range expectedSubcommands {
		found := false
		for _, actual := range stateCmd.Commands {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected state subcommand %q to exist", expected)
		}
	}
}

func TestStateListEmpty(t *testing.T) {
	// Create a temporary empty directory
	tmpDir := t.TempDir()

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "list", "-d", tmpDir})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No state files found") {
		t.Errorf("expected 'No state files found' message, got %q", output)
	}
}

func TestStateListWithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid state file
	stateFile := filepath.Join(tmpDir, "crook-state-test-node.json")
	stateContent := `{
  "version": "v1",
  "node": "test-node",
  "timestamp": "2024-01-15T10:00:00Z",
  "operatorReplicas": 1,
  "resources": [
    {"kind": "Deployment", "namespace": "rook-ceph", "name": "rook-ceph-osd-0", "replicas": 1}
  ]
}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("failed to create test state file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "list", "-d", tmpDir})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "test-node") {
		t.Errorf("expected output to contain node name 'test-node', got %q", output)
	}
	if !strings.Contains(output, "1 state file") {
		t.Errorf("expected output to contain '1 state file', got %q", output)
	}
}

func TestStateListJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid state file
	stateFile := filepath.Join(tmpDir, "crook-state-json-test.json")
	stateContent := `{
  "version": "v1",
  "node": "json-test",
  "timestamp": "2024-01-15T10:00:00Z",
  "operatorReplicas": 1,
  "resources": []
}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("failed to create test state file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "list", "-d", tmpDir, "-f", "json"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var result []map[string]any
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Errorf("expected valid JSON output, got error: %v", unmarshalErr)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 file in JSON output, got %d", len(result))
	}
}

func TestStateShow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid state file
	stateFile := filepath.Join(tmpDir, "crook-state-show-test.json")
	stateContent := `{
  "version": "v1",
  "node": "show-test",
  "timestamp": "2024-01-15T10:00:00Z",
  "operatorReplicas": 2,
  "resources": [
    {"kind": "Deployment", "namespace": "rook-ceph", "name": "rook-ceph-osd-0", "replicas": 1},
    {"kind": "Deployment", "namespace": "rook-ceph", "name": "rook-ceph-mon-a", "replicas": 1}
  ]
}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("failed to create test state file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "show", stateFile})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "show-test") {
		t.Errorf("expected output to contain node name, got %q", output)
	}
	if !strings.Contains(output, "rook-ceph-osd-0") {
		t.Errorf("expected output to contain resource name, got %q", output)
	}
}

func TestStateShowJSON(t *testing.T) {
	tmpDir := t.TempDir()

	stateFile := filepath.Join(tmpDir, "crook-state-json-show.json")
	stateContent := `{
  "version": "v1",
  "node": "json-show",
  "timestamp": "2024-01-15T10:00:00Z",
  "operatorReplicas": 1,
  "resources": []
}`
	if err := os.WriteFile(stateFile, []byte(stateContent), 0o600); err != nil {
		t.Fatalf("failed to create test state file: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "show", stateFile, "-f", "json"})

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
}

func TestStateShowRequiresArg(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "show"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when no file argument provided")
	}
}

func TestStateShowInvalidFile(t *testing.T) {
	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "show", "/nonexistent/file.json"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStateCleanDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an old backup file
	oldBackup := filepath.Join(tmpDir, "crook-state-node.backup.2020-01-01T00:00:00Z.json")
	backupContent := `{
  "version": "v1",
  "node": "node",
  "timestamp": "2020-01-01T00:00:00Z",
  "operatorReplicas": 1,
  "resources": []
}`
	if err := os.WriteFile(oldBackup, []byte(backupContent), 0o600); err != nil {
		t.Fatalf("failed to create test backup file: %v", err)
	}

	// Set the mod time to be old
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatalf("failed to set mod time: %v", err)
	}

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "clean", "-d", tmpDir, "--older-than", "168h", "--dry-run"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "dry run") {
		t.Errorf("expected 'dry run' in output, got %q", output)
	}

	// File should still exist
	if _, statErr := os.Stat(oldBackup); os.IsNotExist(statErr) {
		t.Error("expected backup file to still exist after dry run")
	}
}

func TestStateCleanNoBackups(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := commands.NewRootCmd()
	cmd.SetArgs([]string{"state", "clean", "-d", tmpDir, "--older-than", "168h"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No backup files older than") {
		t.Errorf("expected message about no backup files, got %q", output)
	}
}

func TestStateCleanFlags(t *testing.T) {
	cmd := commands.NewRootCmd()

	var cleanCmd *struct {
		Flags []string
	}

	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "state" {
			for _, sub := range subCmd.Commands() {
				if sub.Name() == "clean" {
					var flags []string
					sub.Flags().VisitAll(func(f *pflag.Flag) {
						flags = append(flags, f.Name)
					})
					cleanCmd = &struct{ Flags []string }{Flags: flags}
					break
				}
			}
			break
		}
	}

	if cleanCmd == nil {
		t.Fatal("clean command not found")
	}

	expectedFlags := []string{"directory", "older-than", "dry-run"}
	for _, expected := range expectedFlags {
		found := false
		for _, actual := range cleanCmd.Flags {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected clean flag %q to exist", expected)
		}
	}
}
