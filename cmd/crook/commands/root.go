// Package commands provides the CLI command implementations for crook.
package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// version information set by build flags
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	buildDate = d
}

// RootOptions holds the global options for all commands
type RootOptions struct {
	// ConfigFile is the path to the configuration file
	ConfigFile string

	// Kubeconfig is the path to the kubeconfig file
	Kubeconfig string

	// Namespace sets both rook-operator-namespace and rook-cluster-namespace
	Namespace string

	// LogLevel sets the logging level (debug, info, warn, error)
	LogLevel string

	// LogFile sets the file path for log output
	LogFile string

	// Config holds the loaded configuration
	Config config.Config

	// Context is the root context for all operations
	Context context.Context

	// CancelFunc cancels the root context
	CancelFunc context.CancelFunc
}

// GlobalOptions is the singleton instance for root options
var GlobalOptions = &RootOptions{}

// NewRootCmd creates the root cobra command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "crook",
		Short: "Kubernetes node maintenance automation for Rook-Ceph clusters",
		Long: `crook - Rook-Ceph Node Maintenance Tool

A Kubernetes node maintenance automation tool for Rook-Ceph clusters.
It safely manages the process of taking nodes down for maintenance
and bringing them back up while preserving Ceph cluster health and state.

Key features:
  - Automated safe node maintenance procedures
  - Prevents data loss by managing Ceph OSDs, monitors, and services
  - Maintains cluster state across reboots and maintenance windows
  - Interactive TUI with real-time feedback
  - Pre-flight validation and health monitoring`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initializeGlobals(cmd)
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			cleanup()
		},
	}

	// Add global flags
	addGlobalFlags(rootCmd)

	// Add subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newDownCmd())
	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newLsCmd())

	return rootCmd
}

// addGlobalFlags adds the global flags to the root command
func addGlobalFlags(cmd *cobra.Command) {
	flags := cmd.PersistentFlags()

	flags.StringVar(&GlobalOptions.ConfigFile, "config", "",
		"config file (default: ./crook.yaml, ~/.config/crook/config.yaml, /etc/crook/config.yaml)")
	flags.StringVar(&GlobalOptions.Kubeconfig, "kubeconfig", "",
		"path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flags.StringVar(&GlobalOptions.Namespace, "namespace", "",
		"rook-ceph namespace (sets both operator and cluster namespace)")
	flags.StringVar(&GlobalOptions.LogLevel, "log-level", "",
		"log level: debug, info, warn, error (default: info)")
	flags.StringVar(&GlobalOptions.LogFile, "log-file", "",
		"log file path (default: stderr)")
}

// initializeGlobals initializes global options from flags, env, and config file
func initializeGlobals(cmd *cobra.Command) error {
	// Set up context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	GlobalOptions.Context = ctx
	GlobalOptions.CancelFunc = cancel

	// Load configuration with flag bindings
	loadOpts := config.LoadOptions{
		ConfigFile: GlobalOptions.ConfigFile,
		Flags:      buildFlagSet(cmd),
	}

	result, err := config.LoadConfig(loadOpts)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	GlobalOptions.Config = result.Config

	// Initialize logger
	if logErr := initLogger(); logErr != nil {
		return fmt.Errorf("failed to initialize logger: %w", logErr)
	}

	// Log configuration source if a file was used
	if result.ConfigFileUsed != "" {
		logger.Debug("loaded configuration", "file", result.ConfigFileUsed)
	}

	return nil
}

// buildFlagSet creates a pflag.FlagSet from cobra command flags for config binding
func buildFlagSet(cmd *cobra.Command) *pflag.FlagSet {
	flags := pflag.NewFlagSet("config", pflag.ContinueOnError)

	// Helper to safely add a flag if it exists and isn't already added
	addIfExists := func(name string) {
		if flags.Lookup(name) != nil {
			return // Already added
		}
		// Check both local and inherited flags
		if localFlag := cmd.Flags().Lookup(name); localFlag != nil {
			flags.AddFlag(localFlag)
		} else if inheritedFlag := cmd.InheritedFlags().Lookup(name); inheritedFlag != nil {
			flags.AddFlag(inheritedFlag)
		}
	}

	// Add relevant flags for config binding
	addIfExists("kubeconfig")
	addIfExists("namespace")
	addIfExists("log-level")
	addIfExists("log-file")

	return flags
}

// initLogger initializes the logger based on configuration
func initLogger() error {
	cfg := GlobalOptions.Config.Logging

	// Determine log level
	level := logger.LevelInfo
	switch cfg.Level {
	case "debug":
		level = logger.LevelDebug
	case "warn":
		level = logger.LevelWarn
	case "error":
		level = logger.LevelError
	}

	// Determine output
	var output io.Writer = os.Stderr
	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", cfg.File, err)
		}
		output = f
	}

	// Determine format
	format := logger.FormatText
	if cfg.Format == "json" {
		format = logger.FormatJSON
	}

	// Create and set default logger
	log := logger.New(logger.Config{
		Level:  level,
		Format: format,
		Output: output,
	})
	logger.SetDefault(log)

	return nil
}

// cleanup performs any necessary cleanup before exit
func cleanup() {
	if GlobalOptions.CancelFunc != nil {
		GlobalOptions.CancelFunc()
	}
}

// newVersionCmd creates the version subcommand
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print version, commit, and build date information",
		Run: func(cmd *cobra.Command, _ []string) {
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "crook version %s\n", version)
			_, _ = fmt.Fprintf(out, "  commit:     %s\n", commit)
			_, _ = fmt.Fprintf(out, "  build date: %s\n", buildDate)
		},
	}
}

// Execute runs the root command
func Execute() error {
	return NewRootCmd().Execute()
}
