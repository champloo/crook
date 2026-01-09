// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// UpOptions holds options specific to the up command
type UpOptions struct {
	// Timeout for the overall operation
	Timeout time.Duration
}

// newUpCmd creates the up subcommand
func newUpCmd() *cobra.Command {
	opts := &UpOptions{}

	cmd := &cobra.Command{
		Use:   "up <node>",
		Short: "Restore a node after maintenance",
		Long: `Restore a Kubernetes node after maintenance by scaling up Rook-Ceph workloads.

This command performs the following steps:
  1. Validates pre-flight conditions (node exists, etc.)
  2. Discovers scaled-down node-pinned deployments
  3. Uncordons the node (marks it schedulable again)
  4. Restores Rook-Ceph deployments to 1 replica
  5. Scales up the rook-ceph-operator
  6. Unsets the Ceph 'noout' flag

This command should be run after 'crook down <node>' and after node maintenance
is complete.`,
		Example: `  # Restore node 'worker-1' after maintenance
  crook up worker-1

  # Set a timeout for the operation
  crook up worker-1 --timeout 15m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeName := args[0]
			return runUp(cmd.Context(), nodeName, opts)
		},
	}

	// Add up-specific flags
	flags := cmd.Flags()
	flags.DurationVar(&opts.Timeout, "timeout", 15*time.Minute,
		"timeout for the overall operation")

	return cmd
}

// runUp executes the up phase workflow
func runUp(ctx context.Context, nodeName string, opts *UpOptions) error {
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

	return runUpTUI(ctx, client, nodeName)
}

// runUpTUI runs the up phase with the interactive TUI
func runUpTUI(ctx context.Context, client *k8s.Client, nodeName string) error {
	cfg := GlobalOptions.Config

	// Create the TUI app model configured for up phase
	appCfg := models.AppConfig{
		Route:    models.RouteUp,
		NodeName: nodeName,
		Config:   cfg,
		Client:   client,
		Context:  ctx,
	}

	app := models.NewAppModel(appCfg)

	// Run the Bubble Tea program
	p := tea.NewProgram(app, tea.WithAltScreen())
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
