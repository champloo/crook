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
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
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

	data, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(data) != "data" {
		t.Fatalf("unexpected backup contents: %s", string(data))
	}
}

func TestBackupFileDisabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
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
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
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
