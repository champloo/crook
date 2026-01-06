// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/maintenance"
	"github.com/andri/crook/pkg/tui/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// UpOptions holds options specific to the up command
type UpOptions struct {
	// StateFile overrides the default state file path
	StateFile string

	// NoTUI disables the interactive TUI and runs in non-interactive mode
	NoTUI bool

	// Yes automatically confirms prompts (for automation)
	Yes bool

	// Timeout for the overall operation
	Timeout time.Duration

	// SkipMissing continues even if some deployments from state file are missing
	SkipMissing bool
}

// newUpCmd creates the up subcommand
func newUpCmd() *cobra.Command {
	opts := &UpOptions{}

	cmd := &cobra.Command{
		Use:   "up <node>",
		Short: "Restore a node after maintenance",
		Long: `Restore a Kubernetes node after maintenance by scaling up Rook-Ceph workloads.

This command performs the following steps:
  1. Validates pre-flight conditions (node exists, state file valid, etc.)
  2. Loads the state file created during the down phase
  3. Uncordons the node (marks it schedulable again)
  4. Restores Rook-Ceph deployments to their original replica counts
  5. Scales up the rook-ceph-operator
  6. Unsets the Ceph 'noout' flag

This command should be run after 'crook down <node>' and after node maintenance
is complete. It requires the state file that was created during the down phase.`,
		Example: `  # Restore node 'worker-1' after maintenance
  crook up worker-1

  # Use a custom state file location
  crook up worker-1 --state-file /tmp/worker-1-state.json

  # Non-interactive mode with auto-confirm (for automation)
  crook up worker-1 --yes --no-tui

  # Skip deployments that no longer exist in the cluster
  crook up worker-1 --skip-missing

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
	flags.StringVar(&opts.StateFile, "state-file", "",
		"path to load state file (default: from config template)")
	flags.BoolVar(&opts.NoTUI, "no-tui", false,
		"disable interactive TUI, run in non-interactive mode")
	flags.BoolVarP(&opts.Yes, "yes", "y", false,
		"automatically confirm prompts (implies --no-tui)")
	flags.DurationVar(&opts.Timeout, "timeout", 15*time.Minute,
		"timeout for the overall operation")
	flags.BoolVar(&opts.SkipMissing, "skip-missing", false,
		"continue if deployments in state file don't exist in cluster")

	return cmd
}

// runUp executes the up phase workflow
func runUp(ctx context.Context, nodeName string, opts *UpOptions) error {
	cfg := GlobalOptions.Config

	// --yes implies --no-tui
	if opts.Yes {
		opts.NoTUI = true
	}

	// Apply timeout to context
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Initialize Kubernetes client
	logger.Info("connecting to kubernetes cluster")
	client, err := k8s.NewClient(ctx, k8s.ClientConfig{
		Kubeconfig: cfg.Kubernetes.Kubeconfig,
		Context:    cfg.Kubernetes.Context,
	})
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	if opts.NoTUI {
		return runUpNonInteractive(ctx, client, nodeName, opts)
	}

	return runUpTUI(ctx, client, nodeName, opts)
}

// runUpTUI runs the up phase with the interactive TUI
func runUpTUI(ctx context.Context, client *k8s.Client, nodeName string, opts *UpOptions) error {
	cfg := GlobalOptions.Config

	// Create the TUI app model configured for up phase
	appCfg := models.AppConfig{
		Route:         models.RouteUp,
		NodeName:      nodeName,
		StateFilePath: opts.StateFile,
		Config:        cfg,
		Client:        client,
		Context:       ctx,
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

// runUpNonInteractive runs the up phase without TUI
func runUpNonInteractive(ctx context.Context, client *k8s.Client, nodeName string, opts *UpOptions) error {
	cfg := GlobalOptions.Config
	out := os.Stdout

	// Print header
	printLine(out, "crook up - restoring node %s after maintenance", nodeName)
	printLine(out, "=============================================\n")

	// If not auto-confirming, show what will happen and ask for confirmation
	if !opts.Yes {
		printLine(out, "This will:")
		printLine(out, "  1. Load state from the down phase state file")
		printLine(out, "  2. Uncordon the node (mark schedulable)")
		printLine(out, "  3. Restore Rook-Ceph deployments to original replicas")
		printLine(out, "  4. Scale up rook-ceph-operator")
		printLine(out, "  5. Unset Ceph noout flag")
		printLine(out, "Continue? [y/N] ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if response != "y" && response != "Y" {
			printLine(out, "Aborted.")
			return nil
		}
	}

	// Execute the up phase with progress callback
	upOpts := maintenance.UpPhaseOptions{
		StateFilePath: opts.StateFile,
		ProgressCallback: func(progress maintenance.UpPhaseProgress) {
			printLine(out, "[%s] %s", progress.Stage, progress.Description)
			if progress.Deployment != "" {
				printLine(out, "  → %s", progress.Deployment)
			}
		},
		WaitOptions: maintenance.WaitOptions{
			Timeout:      time.Duration(cfg.Timeouts.WaitDeploymentTimeoutSeconds) * time.Second,
			PollInterval: 2 * time.Second,
		},
		SkipMissingDeployments: opts.SkipMissing,
	}

	logger.Info("starting up phase", "node", nodeName)
	if err := maintenance.ExecuteUpPhase(ctx, client, cfg, nodeName, upOpts); err != nil {
		return fmt.Errorf("up phase failed: %w", err)
	}

	printLine(out, "\n✓ Up phase completed successfully")
	printLine(out, "Node %s is now operational.", nodeName)

	return nil
}
