// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/models"
	tea "github.com/charmbracelet/bubbletea"
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
	// For non-TUI output formats, defer to crook-3qm.11
	if opts.Output != "tui" {
		return fmt.Errorf("output format %q not yet implemented (see crook-3qm.11)", opts.Output)
	}

	// Initialize context
	ctx := context.Background()

	// Load configuration
	result, err := config.LoadConfig(config.LoadOptions{})
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg := result.Config

	// Initialize Kubernetes client
	client, err := k8s.GetClient(ctx, k8s.ClientConfig{})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Parse --show flag into LsTab slice
	var showTabs []models.LsTab
	if opts.Show != "" {
		showTabs = parseShowTabs(opts.Show)
	}

	// Create the ls model
	model := models.NewLsModel(models.LsModelConfig{
		NodeFilter: opts.NodeFilter,
		Config:     cfg,
		Client:     client,
		Context:    ctx,
		ShowTabs:   showTabs,
	})

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, runErr := p.Run(); runErr != nil {
		return fmt.Errorf("TUI error: %w", runErr)
	}

	return nil
}

// parseShowTabs converts the --show string to a slice of LsTab
func parseShowTabs(show string) []models.LsTab {
	var tabs []models.LsTab
	values := strings.Split(show, ",")

	for _, v := range values {
		v = strings.TrimSpace(strings.ToLower(v))
		switch v {
		case "nodes":
			tabs = append(tabs, models.LsTabNodes)
		case "deployments":
			tabs = append(tabs, models.LsTabDeployments)
		case "osds":
			tabs = append(tabs, models.LsTabOSDs)
		case "pods":
			tabs = append(tabs, models.LsTabPods)
		}
	}

	return tabs
}
