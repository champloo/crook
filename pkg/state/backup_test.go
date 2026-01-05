package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupFileCreatesBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write state: %v", err)
	}

	backup, err := BackupFile(path, BackupOptions{
		Enabled: true,
		Node:    "worker-01",
		Now: func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("backup state: %v", err)
	}
	if backup == "" {
		t.Fatalf("expected backup path")
	}

	data, readErr := os.ReadFile(backup) //nolint:gosec // reading test backup file
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(data) != "data" {
		t.Fatalf("unexpected backup contents: %s", string(data))
	}
}

func TestBackupFileDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write state: %v", err)
	}

	backup, err := BackupFile(path, BackupOptions{Enabled: false})
	if err != nil {
		t.Fatalf("backup disabled: %v", err)
	}
	if backup != "" {
		t.Fatalf("expected no backup path")
	}
}

func TestBackupFileDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write state: %v", err)
	}

	backupDir := filepath.Join(dir, "backups")
	backup, err := BackupFile(path, BackupOptions{
		Enabled:   true,
		Directory: backupDir,
		Node:      "worker-01",
		Now: func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("backup state: %v", err)
	}
	if backup == "" {
		t.Fatalf("expected backup path")
	}
	if !strings.HasPrefix(backup, backupDir) {
		t.Fatalf("expected backup in %s, got %s", backupDir, backup)
	}
}

func TestBackupFileDirectoryExpandsTilde(t *testing.T) {
	t.Parallel()

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Skipf("cannot get home directory: %v", homeErr)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("write state: %v", err)
	}

	// Create a unique subdirectory under home for this test
	testSubDir := filepath.Join(".crook-test-backup", t.Name())
	tildeDir := "~/" + testSubDir
	expectedDir := filepath.Join(home, testSubDir)

	// Clean up after test
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(home, ".crook-test-backup"))
	})

	backup, err := BackupFile(path, BackupOptions{
		Enabled:   true,
		Directory: tildeDir,
		Node:      "worker-01",
		Now: func() time.Time {
			return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("backup state: %v", err)
	}
	if backup == "" {
		t.Fatalf("expected backup path")
	}

	// Verify it expands to home directory, not literal ~/
	if !strings.HasPrefix(backup, expectedDir) {
		t.Fatalf("expected backup in %s, got %s", expectedDir, backup)
	}
	if strings.Contains(backup, "~/") {
		t.Fatalf("tilde was not expanded in backup path: %s", backup)
	}

	// Verify backup file exists and has correct content
	data, readErr := os.ReadFile(backup) //nolint:gosec // reading test backup file
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(data) != "data" {
		t.Fatalf("unexpected backup contents: %s", string(data))
	}
}
