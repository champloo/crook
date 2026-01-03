package state

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andri/crook/internal/logger"
)

// BackupOptions controls state file backup behavior.
type BackupOptions struct {
	Enabled   bool
	Directory string
	Node      string
	Now       func() time.Time
}

// BackupFile creates a backup of an existing state file if present.
func BackupFile(path string, opts BackupOptions) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("state file path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat state file %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("state file path is a directory: %s", path)
	}

	if !opts.Enabled {
		logger.Warn("Overwriting state file without backup (backup disabled in config)", "path", path)
		return "", nil
	}

	clock := opts.Now
	if clock == nil {
		clock = time.Now
	}
	timestamp := clock().UTC().Format(time.RFC3339)

	backupPath := ""
	if strings.TrimSpace(opts.Directory) != "" {
		if strings.TrimSpace(opts.Node) == "" {
			return "", errors.New("state node is required for backup directory")
		}
		if err := os.MkdirAll(opts.Directory, 0o755); err != nil {
			return "", fmt.Errorf("create backup directory %s: %w", opts.Directory, err)
		}
		backupName := fmt.Sprintf("crook-state-%s.%s.json", opts.Node, timestamp)
		backupPath = filepath.Join(opts.Directory, backupName)
	} else {
		backupPath = fmt.Sprintf("%s.backup.%s.json", path, timestamp)
	}

	if err := copyFile(backupPath, path, info.Mode().Perm()); err != nil {
		return "", err
	}

	logger.Info("Backed up existing state file to "+backupPath, "path", path, "backup_path", backupPath)
	return backupPath, nil
}

// WriteFileWithBackup backs up existing state file (if configured) before writing.
func WriteFileWithBackup(path string, state State, backup BackupOptions) (string, error) {
	backupPath, err := BackupFile(path, backup)
	if err != nil {
		return "", err
	}
	if err := WriteFile(path, state); err != nil {
		return backupPath, err
	}
	return backupPath, nil
}

func copyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open state file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return fmt.Errorf("create backup file %s: %w", dst, err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy state file to backup %s: %w", dst, err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync backup file %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close backup file %s: %w", dst, err)
	}

	return nil
}
