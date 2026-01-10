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

// UpOptions holds options specific to the up command
type UpOptions struct {
	// Timeout for the overall operation
	Timeout time.Duration

	// Yes skips the confirmation prompt
	Yes bool
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

  # Skip confirmation prompt
  crook up -y worker-1

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
	flags.BoolVarP(&opts.Yes, "yes", "y", false,
		"skip confirmation prompt")

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

	// Discover scaled-down deployments to show summary
	deployments, err := client.ListScaledDownDeploymentsForNode(ctx, cfg.Namespace, nodeName)
	if err != nil {
		return fmt.Errorf("failed to discover deployments: %w", err)
	}

	// Build deployment names for display
	var deploymentNames []string
	for _, d := range deployments {
		deploymentNames = append(deploymentNames, fmt.Sprintf("%s/%s", d.Namespace, d.Name))
	}

	// Show summary
	pw := cli.NewProgressWriter(os.Stdout)
	pw.PrintSummary(nodeName, len(deployments), deploymentNames)

	// Confirm unless -y
	if !opts.Yes {
		confirmed, confirmErr := cli.Confirm(cli.ConfirmOptions{
			Question: "Proceed with up phase?",
		})
		if confirmErr != nil {
			return fmt.Errorf("confirmation failed: %w", confirmErr)
		}
		if !confirmed {
			return fmt.Errorf("operation cancelled by user")
		}
	}

	// Execute the up phase with progress callback
	// Pass discovered deployments to ensure consistency between confirmation and execution
	executeErr := maintenance.ExecuteUpPhase(ctx, client, cfg, nodeName, maintenance.UpPhaseOptions{
		ProgressCallback: pw.OnUpProgress,
		Deployments:      deployments,
	})
	if executeErr != nil {
		pw.PrintError(fmt.Sprintf("Up phase failed: %s", executeErr.Error()))
		return executeErr
	}

	pw.PrintSuccess(fmt.Sprintf("Node %s has been restored and is operational", nodeName))
	return nil
}
