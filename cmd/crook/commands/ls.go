// Package commands provides the CLI command implementations for crook.
package commands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// LsOptions holds options specific to the ls command
type LsOptions struct {
	// Watch enables continuous refresh mode
	Watch bool

	// RefreshInterval is the refresh interval in seconds (for watch mode)
	RefreshInterval int

	// Output specifies the output format: tui, table, json, yaml
	Output string

	// AllNamespaces searches across all namespaces
	AllNamespaces bool

	// Show specifies which resource types to display (comma-separated)
	Show string

	// NodeFilter is the optional node name to filter by (positional arg)
	NodeFilter string
}

// validOutputFormats are the allowed values for --output
var validOutputFormats = []string{"tui", "table", "json", "yaml"}

// validShowValues are the allowed values for --show (comma-separated)
var validShowValues = []string{"nodes", "deployments", "osds", "pods"}

// newLsCmd creates the ls subcommand
func newLsCmd() *cobra.Command {
	opts := &LsOptions{}

	cmd := &cobra.Command{
		Use:   "ls [node-name]",
		Short: "List Rook-Ceph resources",
		Long: `List Rook-Ceph resources in an interactive TUI or formatted output.

Displays nodes, deployments, OSDs, and pods related to your Rook-Ceph cluster
with support for filtering, search, and multiple output formats.

By default, opens an interactive TUI with tabbed views. Use --output to
select alternative output formats for scripting or CI/CD integration.`,
		Example: `  # Interactive TUI mode (default)
  crook ls

  # Filter by node name
  crook ls worker-1

  # Watch mode with 5 second refresh
  crook ls --watch --refresh 5

  # Table output for scripting
  crook ls --output table

  # JSON output for automation
  crook ls --output json

  # Show only specific resource types
  crook ls --show nodes,osds

  # Search across all namespaces
  crook ls --all-namespaces`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			// Store positional arg if provided
			if len(args) > 0 {
				opts.NodeFilter = args[0]
			}
			return validateLsOptions(opts)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLs(opts)
		},
	}

	// Add ls-specific flags
	flags := cmd.Flags()
	flags.BoolVarP(&opts.Watch, "watch", "w", false,
		"enable continuous refresh mode")
	flags.IntVar(&opts.RefreshInterval, "refresh", 2,
		"refresh interval in seconds (requires --watch)")
	flags.StringVarP(&opts.Output, "output", "o", "tui",
		"output format: tui, table, json, yaml")
	flags.BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", false,
		"list resources across all namespaces")
	flags.StringVar(&opts.Show, "show", "",
		"resource types to display (comma-separated): nodes,deployments,osds,pods")

	return cmd
}

// validateLsOptions validates the ls command options
func validateLsOptions(opts *LsOptions) error {
	// Validate --refresh
	if opts.RefreshInterval < 1 {
		return fmt.Errorf("--refresh must be at least 1 second, got %d", opts.RefreshInterval)
	}

	// Validate --output
	if !slices.Contains(validOutputFormats, opts.Output) {
		return fmt.Errorf("--output must be one of: %s, got %q",
			strings.Join(validOutputFormats, ", "), opts.Output)
	}

	// Validate --show values
	if opts.Show != "" {
		showValues := strings.Split(opts.Show, ",")
		for _, v := range showValues {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if !slices.Contains(validShowValues, v) {
				return fmt.Errorf("--show value %q is invalid; must be subset of: %s",
					v, strings.Join(validShowValues, ", "))
			}
		}
	}

	return nil
}

// runLs executes the ls command
func runLs(opts *LsOptions) error {
	// TODO: Implementation will be added in crook-3qm.2 (TUI model)
	// and crook-3qm.11 (alternative output formats)
	_ = opts
	return fmt.Errorf("ls command not yet implemented")
}
