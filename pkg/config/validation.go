package config

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

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

	if cfg.UI.ProgressRefreshMS < 100 ||
		cfg.UI.LsRefreshNodesMS < 100 ||
		cfg.UI.LsRefreshDeploymentsMS < 100 ||
		cfg.UI.LsRefreshPodsMS < 100 ||
		cfg.UI.LsRefreshOSDsMS < 100 ||
		cfg.UI.LsRefreshHeaderMS < 100 {
		result.Warnings = append(result.Warnings, "Refresh rate <100ms may cause excessive API calls")
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
