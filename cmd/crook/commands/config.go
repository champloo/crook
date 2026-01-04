// Package commands provides the CLI command implementations for crook.
package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ConfigShowOptions holds options for the config show command
type ConfigShowOptions struct {
	Format string
}

// ConfigValidateOptions holds options for the config validate command
type ConfigValidateOptions struct {
	ConfigFile string
	Format     string
}

// newConfigCmd creates the config subcommand with its subcommands
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage crook configuration.

Configuration is loaded from multiple sources in order of precedence:
  1. CLI flags (highest priority)
  2. Environment variables (CROOK_* prefix)
  3. Config file (./crook.yaml, ~/.config/crook/config.yaml, /etc/crook/config.yaml)
  4. Default values (lowest priority)`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigValidateCmd())

	return cmd
}

// newConfigShowCmd creates the config show subcommand
func newConfigShowCmd() *cobra.Command {
	opts := &ConfigShowOptions{}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show effective configuration",
		Long: `Show the effective configuration after merging all sources.

Displays the final configuration values that will be used by crook,
including the source file if one was loaded.`,
		Example: `  # Show configuration in YAML format (default)
  crook config show

  # Show configuration in JSON format
  crook config show --format json

  # Show configuration with a specific config file
  crook config show --config /path/to/config.yaml`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShow(cmd, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Format, "format", "f", "yaml",
		"output format: yaml, json")

	return cmd
}

// newConfigValidateCmd creates the config validate subcommand
func newConfigValidateCmd() *cobra.Command {
	opts := &ConfigValidateOptions{}

	cmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration",
		Long: `Validate configuration file and report any errors or warnings.

Returns exit code 0 if configuration is valid, 1 if there are errors.
Warnings are reported but don't affect the exit code.`,
		Example: `  # Validate default configuration
  crook config validate

  # Validate a specific config file
  crook config validate /path/to/config.yaml

  # Output validation results as JSON
  crook config validate --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.ConfigFile = args[0]
			}
			return runConfigValidate(cmd, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.Format, "format", "f", "text",
		"output format: text, json, yaml")

	return cmd
}

// ConfigOutput represents the configuration output structure
type ConfigOutput struct {
	ConfigFile string        `json:"configFile,omitempty" yaml:"configFile,omitempty"`
	Config     config.Config `json:"config" yaml:"config"`
}

// ValidationOutput represents validation results for output
type ValidationOutput struct {
	Valid    bool     `json:"valid" yaml:"valid"`
	Errors   []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

func runConfigShow(cmd *cobra.Command, opts *ConfigShowOptions) error {
	out := cmd.OutOrStdout()
	cfg := GlobalOptions.Config

	output := ConfigOutput{
		Config: cfg,
	}

	// Get the config file used (if any)
	loadOpts := config.LoadOptions{
		ConfigFile: GlobalOptions.ConfigFile,
	}
	result, err := config.LoadConfig(loadOpts)
	if err == nil && result.ConfigFileUsed != "" {
		output.ConfigFile = result.ConfigFileUsed
	}

	switch strings.ToLower(opts.Format) {
	case "json":
		data, marshalErr := json.MarshalIndent(output, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal config: %w", marshalErr)
		}
		_, _ = fmt.Fprintln(out, string(data))

	default: // yaml
		data, marshalErr := yaml.Marshal(output)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal config: %w", marshalErr)
		}
		_, _ = fmt.Fprint(out, string(data))
	}

	return nil
}

func runConfigValidate(cmd *cobra.Command, opts *ConfigValidateOptions) error {
	out := cmd.OutOrStdout()

	// Determine config file to validate
	configFile := opts.ConfigFile
	if configFile == "" {
		configFile = GlobalOptions.ConfigFile
	}

	// Load configuration for validation
	loadOpts := config.LoadOptions{
		ConfigFile: configFile,
	}

	result, loadErr := config.LoadConfig(loadOpts)

	// Build validation output
	validationOutput := ValidationOutput{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check for load errors
	if loadErr != nil {
		validationOutput.Valid = false
		validationOutput.Errors = append(validationOutput.Errors, loadErr.Error())
	}

	// Add validation result errors
	for _, err := range result.Validation.Errors {
		validationOutput.Valid = false
		validationOutput.Errors = append(validationOutput.Errors, err.Error())
	}

	// Add warnings
	validationOutput.Warnings = append(validationOutput.Warnings, result.Validation.Warnings...)

	// Output results
	switch strings.ToLower(opts.Format) {
	case "json":
		data, marshalErr := json.MarshalIndent(validationOutput, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal validation result: %w", marshalErr)
		}
		_, _ = fmt.Fprintln(out, string(data))

	case "yaml":
		data, marshalErr := yaml.Marshal(validationOutput)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal validation result: %w", marshalErr)
		}
		_, _ = fmt.Fprint(out, string(data))

	default: // text
		if result.ConfigFileUsed != "" {
			_, _ = fmt.Fprintf(out, "Config file: %s\n\n", result.ConfigFileUsed)
		} else {
			_, _ = fmt.Fprint(out, "Config file: (none - using defaults)\n\n")
		}

		if validationOutput.Valid {
			_, _ = fmt.Fprintln(out, "Configuration is valid.")
		} else {
			_, _ = fmt.Fprintln(out, "Configuration has errors:")
			for _, err := range validationOutput.Errors {
				_, _ = fmt.Fprintf(out, "  - %s\n", err)
			}
		}

		if len(validationOutput.Warnings) > 0 {
			_, _ = fmt.Fprintln(out, "\nWarnings:")
			for _, warn := range validationOutput.Warnings {
				_, _ = fmt.Fprintf(out, "  - %s\n", warn)
			}
		}
	}

	// Return error to set non-zero exit code for invalid config
	if !validationOutput.Valid {
		// Return a generic error - the actual errors have been printed
		return fmt.Errorf("configuration validation failed")
	}

	return nil
}
