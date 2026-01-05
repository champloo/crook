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
		expandedDir, expandErr := expandHome(opts.Directory)
		if expandErr != nil {
			return "", fmt.Errorf("expand backup directory path: %w", expandErr)
		}
		if mkdirErr := os.MkdirAll(expandedDir, 0o750); mkdirErr != nil {
			return "", fmt.Errorf("create backup directory %s: %w", expandedDir, mkdirErr)
		}
		backupName := fmt.Sprintf("crook-state-%s.%s.json", opts.Node, timestamp)
		backupPath = filepath.Join(expandedDir, backupName)
	} else {
		backupPath = fmt.Sprintf("%s.backup.%s.json", path, timestamp)
	}

	if copyErr := copyFile(backupPath, path, info.Mode().Perm()); copyErr != nil {
		return "", copyErr
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
	if writeErr := WriteFile(path, state); writeErr != nil {
		return backupPath, writeErr
	}
	return backupPath, nil
}

func copyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src) // #nosec G304 -- src is validated by BackupFile caller
	if err != nil {
		return fmt.Errorf("open state file %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, openErr := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) // #nosec G304 -- dst is constructed from validated path
	if openErr != nil {
		return fmt.Errorf("create backup file %s: %w", dst, openErr)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, copyErr := io.Copy(out, in); copyErr != nil {
		return fmt.Errorf("copy state file to backup %s: %w", dst, copyErr)
	}
	if syncErr := out.Sync(); syncErr != nil {
		return fmt.Errorf("sync backup file %s: %w", dst, syncErr)
	}
	if closeErr := out.Close(); closeErr != nil {
		return fmt.Errorf("close backup file %s: %w", dst, closeErr)
	}

	return nil
}
