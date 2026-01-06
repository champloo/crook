package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/util/validation"
)

var placeholderPattern = regexp.MustCompile(`{{[^}]*}}`)

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

	if err := validateNamespace(cfg.Kubernetes.RookOperatorNamespace); err != nil {
		result.Errors = append(result.Errors, err)
	}
	if err := validateNamespace(cfg.Kubernetes.RookClusterNamespace); err != nil {
		result.Errors = append(result.Errors, err)
	}

	if err := validateKubeconfigPath(cfg.Kubernetes.Kubeconfig); err != nil {
		result.Errors = append(result.Errors, err)
	}

	result.Errors = append(result.Errors, validateStateTemplate(cfg.State.FilePathTemplate)...)

	if len(cfg.DeploymentFilters.Prefixes) == 0 {
		result.Errors = append(result.Errors, fmt.Errorf("deployment filter prefixes must be non-empty"))
	} else {
		for _, prefix := range cfg.DeploymentFilters.Prefixes {
			if strings.TrimSpace(prefix) == "" {
				result.Errors = append(result.Errors, fmt.Errorf("deployment filter prefixes must be non-empty"))
				break
			}
		}
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

func validateKubeconfigPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	resolved, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("kubeconfig file not found: %s", path)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig file not found: %s", resolved)
		}
		return fmt.Errorf("kubeconfig file not found: %s", resolved)
	}
	if info.IsDir() {
		return fmt.Errorf("kubeconfig file not found: %s", resolved)
	}
	return nil
}

func validateStateTemplate(templateValue string) []error {
	if strings.TrimSpace(templateValue) == "" {
		return []error{fmt.Errorf("invalid state file template: must be non-empty")}
	}
	if _, err := template.New("state").Parse(templateValue); err != nil {
		return []error{fmt.Errorf("invalid state file template: %s", err.Error())}
	}

	matches := placeholderPattern.FindAllString(templateValue, -1)
	var errs []error
	for _, match := range matches {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
		if inner != ".Node" {
			errs = append(errs, fmt.Errorf("invalid state file template: unknown placeholder %s, valid: {{.Node}}", match))
		}
	}
	return errs
}

func expandPath(path string) (string, error) {
	expanded := os.ExpandEnv(strings.TrimSpace(path))
	if expanded == "" {
		return "", nil
	}
	if expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(expanded, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, expanded[2:]), nil
	}
	return expanded, nil
}
