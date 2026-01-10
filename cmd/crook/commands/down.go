// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/models"
	"github.com/spf13/cobra"
)

// DownOptions holds options specific to the down command
type DownOptions struct {
	// Timeout for the overall operation
	Timeout time.Duration
}

// newDownCmd creates the down subcommand
func newDownCmd() *cobra.Command {
	opts := &DownOptions{}

	cmd := &cobra.Command{
		Use:   "down <node>",
		Short: "Prepare a node for maintenance",
		Long: `Prepare a Kubernetes node for maintenance by safely scaling down Rook-Ceph workloads.

This command performs the following steps:
  1. Validates pre-flight conditions (node exists, Ceph healthy, etc.)
  2. Cordons the node (marks it unschedulable)
  3. Sets the Ceph 'noout' flag to prevent data rebalancing
  4. Scales down the rook-ceph-operator
  5. Discovers and scales down node-pinned Rook-Ceph deployments

After running this command, the node is safe for maintenance operations
like reboots, hardware changes, or OS upgrades.

Use 'crook up <node>' to restore the node after maintenance is complete.`,
		Example: `  # Prepare node 'worker-1' for maintenance
  crook down worker-1

  # Set a timeout for the operation
  crook down worker-1 --timeout 10m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeName := args[0]
			return runDown(cmd.Context(), nodeName, opts)
		},
	}

	// Add down-specific flags
	flags := cmd.Flags()
	flags.DurationVar(&opts.Timeout, "timeout", 10*time.Minute,
		"timeout for the overall operation")

	return cmd
}

// runDown executes the down phase workflow
func runDown(ctx context.Context, nodeName string, opts *DownOptions) error {
	cfg := GlobalOptions.Config

	// Apply timeout to context
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Initialize Kubernetes client
	logger.Info("connecting to kubernetes cluster")
	client, err := k8s.NewClient(ctx, k8s.ClientConfig{
		CephCommandTimeout: time.Duration(cfg.Timeouts.CephCommandTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return runDownTUI(ctx, client, nodeName)
}

// runDownTUI runs the down phase with the interactive TUI
func runDownTUI(ctx context.Context, client *k8s.Client, nodeName string) error {
	cfg := GlobalOptions.Config

	// Create the TUI app model configured for down phase
	appCfg := models.AppConfig{
		Route:    models.RouteDown,
		NodeName: nodeName,
		Config:   cfg,
		Client:   client,
		Context:  ctx,
	}

	app := models.NewAppModel(appCfg)

	// Run the Bubble Tea program
	p := tea.NewProgram(app)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if the operation completed successfully
	if appModel, ok := finalModel.(*models.AppModel); ok {
		if !appModel.IsInitialized() {
			return fmt.Errorf("operation was cancelled or failed to initialize")
		}
	}

	return nil
}
