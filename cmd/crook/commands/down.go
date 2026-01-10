// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/cli"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
	"github.com/spf13/cobra"
)

// DownOptions holds options specific to the down command
type DownOptions struct {
	// Timeout for the overall operation
	Timeout time.Duration

	// Yes skips the confirmation prompt
	Yes bool
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

  # Skip confirmation prompt
  crook down -y worker-1

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
	flags.BoolVarP(&opts.Yes, "yes", "y", false,
		"skip confirmation prompt")

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
	client, err := newK8sClient(ctx, k8s.ClientConfig{
		CephCommandTimeout: time.Duration(cfg.Timeouts.CephCommandTimeoutSeconds) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Validate node exists
	exists, err := client.NodeExists(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("failed to check if node %q exists: %w", nodeName, err)
	}
	if !exists {
		return fmt.Errorf("node %q not found in cluster", nodeName)
	}

	// Discover deployments to show summary
	deployments, err := client.ListNodePinnedDeployments(ctx, cfg.Namespace, nodeName)
	if err != nil {
		return fmt.Errorf("failed to discover deployments: %w", err)
	}

	pw := cli.NewProgressWriter(os.Stdout)

	if maintenance.IsInDownState(ctx, client, cfg, nodeName, deployments) {
		pw.PrintSuccess(fmt.Sprintf("Node %s is already prepared for maintenance (cordoned, noout set, operator down)", nodeName))
		return nil
	}

	// Build deployment names for display
	var deploymentNames []string
	for _, d := range deployments {
		deploymentNames = append(deploymentNames, fmt.Sprintf("%s/%s", d.Namespace, d.Name))
	}

	// Show summary
	pw.PrintSummary(nodeName, len(deployments), deploymentNames)

	// Confirm unless -y
	if !opts.Yes {
		confirmed, confirmErr := cli.Confirm(cli.ConfirmOptions{
			Question: "Proceed with down phase?",
		})
		if confirmErr != nil {
			return fmt.Errorf("confirmation failed: %w", confirmErr)
		}
		if !confirmed {
			return fmt.Errorf("operation cancelled by user")
		}
	}

	// Execute the down phase with progress callback
	executeErr := executeDownPhase(ctx, client, cfg, nodeName, maintenance.DownPhaseOptions{
		ProgressCallback: pw.OnDownProgress,
	})
	if executeErr != nil {
		pw.PrintError(fmt.Sprintf("Down phase failed: %s", executeErr.Error()))
		return executeErr
	}

	pw.PrintSuccess(fmt.Sprintf("Node %s is now ready for maintenance", nodeName))
	return nil
}
