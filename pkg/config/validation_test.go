package config

import (
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
	cfg.Namespace = ""
	cfg.Namespace = "invalid_namespace!"
	cfg.Timeouts.APICallTimeoutSeconds = 0
	cfg.Timeouts.CephCommandTimeoutSeconds = -1
	cfg.UI.K8sRefreshMS = 50

	result := ValidateConfig(cfg)
	if len(result.Errors) < 2 {
		t.Fatalf("expected multiple errors, got %d", len(result.Errors))
	}
	if !result.HasWarnings() {
		t.Fatalf("expected warnings for aggressive refresh")
	}

	assertErrorContains(t, result.Errors, "invalid namespace")
	assertErrorContains(t, result.Errors, "timeout must be >= 1 second")
}

func assertErrorContains(t *testing.T, errors []error, substring string) {
	for _, err := range errors {
		if strings.Contains(err.Error(), substring) {
			return
		}
	}

	t.Fatalf("expected error containing %q", substring)
}
