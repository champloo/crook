// Package commands provides the CLI command implementations for crook.
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andri/crook/pkg/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// StateListOptions holds options for the state list command
type StateListOptions struct {
	Directory string
	Format    string
}

// StateShowOptions holds options for the state show command
type StateShowOptions struct {
	Format string
}

// StateCleanOptions holds options for the state clean command
type StateCleanOptions struct {
	Directory string
	OlderThan time.Duration
	DryRun    bool
}

// newStateCmd creates the state subcommand with its subcommands
func newStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Manage state files",
		Long: `Manage crook state files created during maintenance operations.

State files are JSON files that capture the state of Rook-Ceph deployments
before taking a node down for maintenance. These files are used to restore
the cluster to its original state after maintenance is complete.`,
	}

	cmd.AddCommand(newStateListCmd())
	cmd.AddCommand(newStateShowCmd())
	cmd.AddCommand(newStateCleanCmd())

	return cmd
}

// newStateListCmd creates the state list subcommand
func newStateListCmd() *cobra.Command {
	opts := &StateListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List state files",
		Long: `List state files in the current directory or specified directory.

Displays state files with their metadata including node name, timestamp,
and number of tracked resources.`,
		Example: `  # List state files in current directory
  crook state list

  # List state files in a specific directory
  crook state list --directory /var/lib/crook/states

  # Output as JSON
  crook state list --format json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStateList(cmd, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Directory, "directory", "d", ".",
		"directory to search for state files")
	flags.StringVarP(&opts.Format, "format", "f", "table",
		"output format: table, json, yaml")

	return cmd
}

// newStateShowCmd creates the state show subcommand
func newStateShowCmd() *cobra.Command {
	opts := &StateShowOptions{}

	cmd := &cobra.Command{
		Use:   "show <file>",
		Short: "Show details of a state file",
		Long: `Show detailed information about a specific state file.

Displays the complete contents of a state file including all tracked
resources and their original replica counts.`,
		Example: `  # Show state file contents
  crook state show crook-state-worker-1.json

  # Output as JSON
  crook state show crook-state-worker-1.json --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStateShow(cmd, args[0], opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Format, "format", "f", "yaml",
		"output format: yaml, json")

	return cmd
}

// newStateCleanCmd creates the state clean subcommand
func newStateCleanCmd() *cobra.Command {
	opts := &StateCleanOptions{}

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean old backup state files",
		Long: `Clean old backup state files from the backup directory.

Removes backup state files older than the specified duration. By default,
searches in the configured backup directory.`,
		Example: `  # Remove backups older than 7 days (dry run)
  crook state clean --older-than 168h --dry-run

  # Remove backups older than 30 days
  crook state clean --older-than 720h

  # Clean backups from a specific directory
  crook state clean --directory /var/lib/crook/backups --older-than 168h`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStateClean(cmd, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Directory, "directory", "d", "",
		"directory to clean (default: configured backup directory)")
	flags.DurationVar(&opts.OlderThan, "older-than", 7*24*time.Hour,
		"remove backups older than this duration (e.g., 168h, 720h)")
	flags.BoolVar(&opts.DryRun, "dry-run", false,
		"show what would be deleted without actually deleting")

	return cmd
}

// StateFileInfo holds metadata about a state file
type StateFileInfo struct {
	Path             string    `json:"path" yaml:"path"`
	Node             string    `json:"node" yaml:"node"`
	Timestamp        time.Time `json:"timestamp" yaml:"timestamp"`
	ResourceCount    int       `json:"resourceCount" yaml:"resourceCount"`
	Size             int64     `json:"size" yaml:"size"`
	IsBackup         bool      `json:"isBackup" yaml:"isBackup"`
	OperatorReplicas int       `json:"operatorReplicas" yaml:"operatorReplicas"`
}

func runStateList(cmd *cobra.Command, opts *StateListOptions) error {
	out := cmd.OutOrStdout()

	// Find all state files
	files, err := findStateFiles(opts.Directory)
	if err != nil {
		return fmt.Errorf("failed to find state files: %w", err)
	}

	if len(files) == 0 {
		_, _ = fmt.Fprintln(out, "No state files found.")
		return nil
	}

	switch strings.ToLower(opts.Format) {
	case "json":
		data, marshalErr := json.MarshalIndent(files, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal state files: %w", marshalErr)
		}
		_, _ = fmt.Fprintln(out, string(data))

	case "yaml":
		data, marshalErr := yaml.Marshal(files)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal state files: %w", marshalErr)
		}
		_, _ = fmt.Fprint(out, string(data))

	default: // table
		_, _ = fmt.Fprintf(out, "%-40s %-20s %-25s %-10s %-8s\n",
			"FILE", "NODE", "TIMESTAMP", "RESOURCES", "BACKUP")
		_, _ = fmt.Fprintf(out, "%s\n", strings.Repeat("-", 105))

		for _, f := range files {
			backup := ""
			if f.IsBackup {
				backup = "yes"
			}
			_, _ = fmt.Fprintf(out, "%-40s %-20s %-25s %-10d %-8s\n",
				truncate(f.Path, 40),
				truncate(f.Node, 20),
				f.Timestamp.Format("2006-01-02 15:04:05"),
				f.ResourceCount,
				backup,
			)
		}
		_, _ = fmt.Fprintf(out, "\nFound %d state file(s)\n", len(files))
	}

	return nil
}

func runStateShow(cmd *cobra.Command, filePath string, opts *StateShowOptions) error {
	out := cmd.OutOrStdout()

	// Parse the state file
	stateData, err := state.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	switch strings.ToLower(opts.Format) {
	case "json":
		data, marshalErr := json.MarshalIndent(stateData, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal state: %w", marshalErr)
		}
		_, _ = fmt.Fprintln(out, string(data))

	default: // yaml
		data, marshalErr := yaml.Marshal(stateData)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal state: %w", marshalErr)
		}
		_, _ = fmt.Fprint(out, string(data))
	}

	return nil
}

func runStateClean(cmd *cobra.Command, opts *StateCleanOptions) error {
	out := cmd.OutOrStdout()
	cfg := GlobalOptions.Config

	// Determine directory to clean
	dir := opts.Directory
	if dir == "" {
		dir = cfg.State.BackupDirectory
	}

	// Expand home directory
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(home, dir[1:])
	}

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(out, "Backup directory does not exist:", dir)
			return nil
		}
		return fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}

	// Find backup files to clean
	cutoff := time.Now().Add(-opts.OlderThan)
	var toDelete []string
	var totalSize int64

	walkErr := filepath.Walk(dir, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fileInfo.IsDir() {
			return nil
		}

		// Check if it's a backup state file
		if !isBackupStateFile(path) {
			return nil
		}

		// Check if it's older than cutoff
		if fileInfo.ModTime().Before(cutoff) {
			toDelete = append(toDelete, path)
			totalSize += fileInfo.Size()
		}

		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("failed to scan directory: %w", walkErr)
	}

	if len(toDelete) == 0 {
		_, _ = fmt.Fprintln(out, "No backup files older than", opts.OlderThan, "found.")
		return nil
	}

	// Show what will be deleted
	_, _ = fmt.Fprintf(out, "Found %d backup file(s) older than %s (%.2f KB total)\n\n",
		len(toDelete), opts.OlderThan, float64(totalSize)/1024)

	if opts.DryRun {
		_, _ = fmt.Fprintln(out, "Files that would be deleted (dry run):")
		for _, path := range toDelete {
			_, _ = fmt.Fprintln(out, " ", path)
		}
		return nil
	}

	// Delete files
	_, _ = fmt.Fprintln(out, "Deleting files:")
	var deleted int
	for _, path := range toDelete {
		if removeErr := os.Remove(path); removeErr != nil {
			_, _ = fmt.Fprintf(out, "  ERROR: %s: %v\n", path, removeErr)
		} else {
			_, _ = fmt.Fprintln(out, " ", path)
			deleted++
		}
	}

	_, _ = fmt.Fprintf(out, "\nDeleted %d file(s)\n", deleted)
	return nil
}

// findStateFiles finds all state files in the given directory
func findStateFiles(dir string) ([]StateFileInfo, error) {
	var files []StateFileInfo

	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Check if it looks like a state file
		if !isStateFile(path) {
			return nil
		}

		// Try to parse the state file (skip invalid files silently)
		stateData, parseErr := state.ParseFile(path)
		if parseErr != nil {
			return nil //nolint:nilerr // intentionally skip unparseable files
		}

		files = append(files, StateFileInfo{
			Path:             path,
			Node:             stateData.Node,
			Timestamp:        stateData.Timestamp,
			ResourceCount:    len(stateData.Resources),
			Size:             info.Size(),
			IsBackup:         isBackupStateFile(path),
			OperatorReplicas: stateData.OperatorReplicas,
		})

		return nil
	})

	return files, walkErr
}

// isStateFile checks if a file looks like a crook state file
func isStateFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".json") &&
		(strings.HasPrefix(base, "crook-state-") || strings.Contains(base, "-state-"))
}

// isBackupStateFile checks if a file is a backup state file
func isBackupStateFile(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, ".backup.") ||
		(strings.HasPrefix(base, "crook-state-") && strings.Count(base, ".") > 1)
}

// truncate truncates a string to the given length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
