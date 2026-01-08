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

// DownOptions holds options specific to the down command
type DownOptions struct {
	// NoTUI disables the interactive TUI and runs in non-interactive mode
	NoTUI bool

	// Yes automatically confirms prompts (for automation)
	Yes bool

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

  # Non-interactive mode with auto-confirm (for automation)
  crook down worker-1 --yes --no-tui

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
	flags.BoolVar(&opts.NoTUI, "no-tui", false,
		"disable interactive TUI, run in non-interactive mode")
	flags.BoolVarP(&opts.Yes, "yes", "y", false,
		"automatically confirm prompts (implies --no-tui)")
	flags.DurationVar(&opts.Timeout, "timeout", 10*time.Minute,
		"timeout for the overall operation")

	return cmd
}

// runDown executes the down phase workflow
func runDown(ctx context.Context, nodeName string, opts *DownOptions) error {
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
		return runDownNonInteractive(ctx, client, nodeName, opts)
	}

	return runDownTUI(ctx, client, nodeName, opts)
}

// runDownTUI runs the down phase with the interactive TUI
func runDownTUI(ctx context.Context, client *k8s.Client, nodeName string, _ *DownOptions) error {
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

// runDownNonInteractive runs the down phase without TUI
func runDownNonInteractive(ctx context.Context, client *k8s.Client, nodeName string, opts *DownOptions) error {
	cfg := GlobalOptions.Config
	out := os.Stdout

	// Print header
	printLine(out, "crook down - preparing node %s for maintenance", nodeName)
	printLine(out, "==============================================\n")

	// If not auto-confirming, show what will happen and ask for confirmation
	if !opts.Yes {
		printLine(out, "This will:")
		printLine(out, "  1. Cordon the node (mark unschedulable)")
		printLine(out, "  2. Set Ceph noout flag")
		printLine(out, "  3. Scale down rook-ceph-operator")
		printLine(out, "  4. Scale down Rook-Ceph deployments on the node\n")
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

	// Execute the down phase with progress callback
	downOpts := maintenance.DownPhaseOptions{
		ProgressCallback: func(progress maintenance.DownPhaseProgress) {
			printLine(out, "[%s] %s", progress.Stage, progress.Description)
			if progress.Deployment != "" {
				printLine(out, "  → %s", progress.Deployment)
			}
		},
		WaitOptions: maintenance.WaitOptions{
			Timeout:      time.Duration(cfg.Timeouts.WaitDeploymentTimeoutSeconds) * time.Second,
			APITimeout:   time.Duration(cfg.Timeouts.APICallTimeoutSeconds) * time.Second,
			PollInterval: 2 * time.Second,
		},
	}

	logger.Info("starting down phase", "node", nodeName)
	if err := maintenance.ExecuteDownPhase(ctx, client, cfg, nodeName, downOpts); err != nil {
		return fmt.Errorf("down phase failed: %w", err)
	}

	printLine(out, "\n✓ Down phase completed successfully")
	printLine(out, "Node %s is now safe for maintenance.", nodeName)
	printLine(out, "Run 'crook up %s' when maintenance is complete.", nodeName)

	return nil
}

// printLine prints a formatted line to the given writer, ignoring write errors
// (appropriate for terminal output where errors are not recoverable)
func printLine(out *os.File, format string, args ...any) {
	_, _ = fmt.Fprintf(out, format+"\n", args...)
}
