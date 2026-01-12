package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidationError wraps a ValidationResult as an error.
// It provides actionable error messages that include all validation issues.
type ValidationError struct {
	Result ValidationResult
}

// Error implements the error interface, returning all validation errors as a single message.
func (e *ValidationError) Error() string {
	if len(e.Result.Errors) == 0 {
		return "configuration validation failed"
	}
	if len(e.Result.Errors) == 1 {
		return fmt.Sprintf("configuration validation failed: %s", e.Result.Errors[0])
	}
	var b strings.Builder
	b.WriteString("configuration validation failed:")
	for _, err := range e.Result.Errors {
		b.WriteString("\n  - ")
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap returns the underlying errors joined together.
func (e *ValidationError) Unwrap() error {
	return errors.Join(e.Result.Errors...)
}

// ValidationResult captures validation errors and warnings.
type ValidationResult struct {
	Errors   []error
	Warnings []string
}

// HasErrors reports whether validation errors exist.
func (r ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings reports whether validation warnings exist.
func (r ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Allowed values for logging configuration.
var (
	allowedLogLevels  = []string{"debug", "info", "warn", "error"}
	allowedLogFormats = []string{"text", "json"}
)

// ValidateConfig validates configuration values and returns all issues.
func ValidateConfig(cfg Config) ValidationResult {
	var result ValidationResult

	if err := validateNamespace(cfg.Namespace); err != nil {
		result.Errors = append(result.Errors, err)
	}

	for _, timeout := range []int{
		cfg.Timeouts.APICallTimeoutSeconds,
		cfg.Timeouts.WaitDeploymentTimeoutSeconds,
		cfg.Timeouts.CephCommandTimeoutSeconds,
	} {
		if timeout < 1 {
			result.Errors = append(result.Errors, fmt.Errorf("timeout must be >= 1 second, got: %d", timeout))
		}
	}

	// Validate logging.level
	if cfg.Logging.Level != "" && !slices.Contains(allowedLogLevels, cfg.Logging.Level) {
		result.Errors = append(result.Errors, fmt.Errorf(
			"invalid logging.level %q: allowed values are %v",
			cfg.Logging.Level, allowedLogLevels))
	}

	// Validate logging.format
	if cfg.Logging.Format != "" && !slices.Contains(allowedLogFormats, cfg.Logging.Format) {
		result.Errors = append(result.Errors, fmt.Errorf(
			"invalid logging.format %q: allowed values are %v",
			cfg.Logging.Format, allowedLogFormats))
	}

	// Validate refresh intervals: must be > 0
	if cfg.UI.K8sRefreshMS <= 0 {
		result.Errors = append(result.Errors, fmt.Errorf(
			"ui.k8s-refresh-ms must be > 0, got: %d", cfg.UI.K8sRefreshMS))
	}
	if cfg.UI.CephRefreshMS <= 0 {
		result.Errors = append(result.Errors, fmt.Errorf(
			"ui.ceph-refresh-ms must be > 0, got: %d", cfg.UI.CephRefreshMS))
	}

	// Warn for very small refresh intervals
	if cfg.UI.K8sRefreshMS > 0 && cfg.UI.K8sRefreshMS < 100 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("ui.k8s-refresh-ms=%d is below 100ms - may cause excessive API calls", cfg.UI.K8sRefreshMS))
	}
	if cfg.UI.CephRefreshMS > 0 && cfg.UI.CephRefreshMS < 100 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("ui.ceph-refresh-ms=%d is below 100ms - may cause excessive API calls", cfg.UI.CephRefreshMS))
	}

	return result
}

func validateNamespace(namespace string) error {
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("invalid namespace '%s': must be non-empty and match Kubernetes naming rules", namespace)
	}
	if errs := validation.IsDNS1123Label(namespace); len(errs) > 0 {
		return fmt.Errorf("invalid namespace '%s': must be non-empty and match Kubernetes naming rules", namespace)
	}
	return nil
}
