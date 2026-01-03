package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfigValidDefaults(t *testing.T) {
	cfg := DefaultConfig()
	result := ValidateConfig(cfg)
	if result.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func TestValidateConfigMultipleErrors(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Kubernetes.RookOperatorNamespace = ""
	cfg.Kubernetes.RookClusterNamespace = "invalid_namespace!"
	cfg.Kubernetes.Kubeconfig = filepath.Join(t.TempDir(), "missing-kubeconfig")
	cfg.State.FilePathTemplate = "{{.Foo}}"
	cfg.Timeouts.APICallTimeoutSeconds = 0
	cfg.Timeouts.CephCommandTimeoutSeconds = -1
	cfg.DeploymentFilters.Prefixes = []string{}
	cfg.UI.ProgressRefreshMS = 50

	result := ValidateConfig(cfg)
	if len(result.Errors) < 5 {
		t.Fatalf("expected multiple errors, got %d", len(result.Errors))
	}
	if !result.HasWarnings() {
		t.Fatalf("expected warnings for aggressive refresh")
	}

	assertErrorContains(t, result.Errors, "Invalid namespace")
	assertErrorContains(t, result.Errors, "Kubeconfig file not found")
	assertErrorContains(t, result.Errors, "Invalid state file template: unknown placeholder")
	assertErrorContains(t, result.Errors, "Timeout must be >= 1 second")
	assertErrorContains(t, result.Errors, "deployment filter prefixes must be non-empty")
}

func TestValidateConfigKubeconfigExists(t *testing.T) {
	cfg := DefaultConfig()
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	cfg.Kubernetes.Kubeconfig = kubeconfigPath

	result := ValidateConfig(cfg)
	if result.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func TestValidateConfigKubeconfigTildePath(t *testing.T) {
	cfg := DefaultConfig()
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	t.Setenv("HOME", tempDir)
	cfg.Kubernetes.Kubeconfig = "~/kubeconfig"

	result := ValidateConfig(cfg)
	if result.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func TestValidateConfigTemplateAllowsLiteralPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.State.FilePathTemplate = "/tmp/state.json"

	result := ValidateConfig(cfg)
	if result.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func assertErrorContains(t *testing.T, errors []error, substring string) {
	for _, err := range errors {
		if strings.Contains(err.Error(), substring) {
			return
		}
	}

	t.Fatalf("expected error containing %q", substring)
}
