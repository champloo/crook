// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/models"
	"github.com/andri/crook/pkg/tui/output"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// LsOptions holds options specific to the ls command
type LsOptions struct {
	// Output specifies the output format: tui, table, json, yaml
	Output string

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

  # Table output for scripting
  crook ls --output table

  # JSON output for automation
  crook ls --output json

  # Show only specific resource types
  crook ls --show nodes,osds`,
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
	flags.StringVarP(&opts.Output, "output", "o", "tui",
		"output format: tui, table, json, yaml")
	flags.StringVar(&opts.Show, "show", "",
		"resource types to display (comma-separated): nodes,deployments,osds,pods")

	return cmd
}

// validateLsOptions validates the ls command options
func validateLsOptions(opts *LsOptions) error {
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
	// Initialize context
	ctx := context.Background()

	// Load configuration
	result, err := config.LoadConfig(config.LoadOptions{})
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg := result.Config

	// Initialize Kubernetes client with config-derived settings
	client, err := k8s.NewClient(ctx, k8s.ClientConfig{
		Kubeconfig:         cfg.Kubernetes.Kubeconfig,
		Context:            cfg.Kubernetes.Context,
		CephCommandTimeout: time.Duration(cfg.Timeouts.CephCommandTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Validate node filter if provided
	if opts.NodeFilter != "" {
		exists, checkErr := client.NodeExists(ctx, opts.NodeFilter)
		if checkErr != nil {
			return fmt.Errorf("failed to verify node: %w", checkErr)
		}
		if !exists {
			return fmt.Errorf("node %q not found in cluster", opts.NodeFilter)
		}
	}

	// Handle non-TUI output formats
	if opts.Output != "tui" {
		return runLsNonTUI(ctx, opts, cfg, client)
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

// runLsNonTUI handles non-TUI output formats (table, json, yaml)
func runLsNonTUI(ctx context.Context, opts *LsOptions, cfg config.Config, client *k8s.Client) error {
	// Parse output format
	format, err := output.ParseFormat(opts.Output)
	if err != nil {
		return err
	}

	// Parse resource types to display
	resourceTypes := output.ParseResourceTypes(opts.Show)

	// Fetch and render
	data, fetchErr := output.FetchData(ctx, output.FetchOptions{
		Client:        client,
		Config:        cfg,
		ResourceTypes: resourceTypes,
		NodeFilter:    opts.NodeFilter,
	})
	if fetchErr != nil {
		return fmt.Errorf("failed to fetch data: %w", fetchErr)
	}

	if renderErr := output.Render(os.Stdout, data, format); renderErr != nil {
		return fmt.Errorf("failed to render output: %w", renderErr)
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
