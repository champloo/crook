package config

import (
	"errors"
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

func TestValidateConfigLoggingLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{"debug valid", "debug", false},
		{"info valid", "info", false},
		{"warn valid", "warn", false},
		{"error valid", "error", false},
		{"empty valid", "", false},
		{"invalid level", "verbose", true},
		{"invalid case", "DEBUG", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Logging.Level = tt.level
			result := ValidateConfig(cfg)
			hasErr := hasErrorContaining(result.Errors, "invalid logging.level")
			if hasErr != tt.wantErr {
				t.Errorf("level=%q: wantErr=%v, gotErr=%v, errors=%v",
					tt.level, tt.wantErr, hasErr, result.Errors)
			}
		})
	}
}

func TestValidateConfigLoggingFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"text valid", "text", false},
		{"json valid", "json", false},
		{"empty valid", "", false},
		{"invalid format", "xml", true},
		{"invalid case", "JSON", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Logging.Format = tt.format
			result := ValidateConfig(cfg)
			hasErr := hasErrorContaining(result.Errors, "invalid logging.format")
			if hasErr != tt.wantErr {
				t.Errorf("format=%q: wantErr=%v, gotErr=%v, errors=%v",
					tt.format, tt.wantErr, hasErr, result.Errors)
			}
		})
	}
}

func TestValidateConfigRefreshIntervals(t *testing.T) {
	tests := []struct {
		name        string
		k8sMS       int
		cephMS      int
		wantErrors  int
		wantWarning bool
	}{
		{"defaults ok", DefaultK8sRefreshMS, DefaultCephRefreshMS, 0, false},
		{"k8s zero", 0, DefaultCephRefreshMS, 1, false},
		{"ceph zero", DefaultK8sRefreshMS, 0, 1, false},
		{"both zero", 0, 0, 2, false},
		{"k8s negative", -1, DefaultCephRefreshMS, 1, false},
		{"k8s low warning", 50, DefaultCephRefreshMS, 0, true},
		{"ceph low warning", DefaultK8sRefreshMS, 50, 0, true},
		{"both low warning", 50, 50, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.UI.K8sRefreshMS = tt.k8sMS
			cfg.UI.CephRefreshMS = tt.cephMS
			result := ValidateConfig(cfg)

			refreshErrors := countErrorsContaining(result.Errors, "refresh-ms must be > 0")
			if refreshErrors != tt.wantErrors {
				t.Errorf("expected %d refresh errors, got %d: %v", tt.wantErrors, refreshErrors, result.Errors)
			}

			if tt.wantWarning && !result.HasWarnings() {
				t.Errorf("expected warnings for low refresh rate")
			}
		})
	}
}

func TestValidationErrorMessage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Namespace = "invalid!"
	cfg.Timeouts.APICallTimeoutSeconds = 0
	result := ValidateConfig(cfg)

	err := &ValidationError{Result: result}
	msg := err.Error()

	if !strings.Contains(msg, "configuration validation failed") {
		t.Errorf("expected 'configuration validation failed', got: %s", msg)
	}
	if !strings.Contains(msg, "invalid namespace") {
		t.Errorf("expected 'invalid namespace' in error, got: %s", msg)
	}
	if !strings.Contains(msg, "timeout must be >= 1 second") {
		t.Errorf("expected timeout error in message, got: %s", msg)
	}
}

func TestValidationErrorUnwrap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Namespace = "invalid!"
	result := ValidateConfig(cfg)

	err := &ValidationError{Result: result}

	// Verify Unwrap returns joined errors
	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Fatalf("expected non-nil unwrapped error")
	}

	// Verify we can find individual errors via errors.Is/As
	var found bool
	for _, e := range result.Errors {
		if errors.Is(unwrapped, e) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unwrapped error should contain original errors")
	}
}

func TestValidationErrorSingleError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Namespace = "invalid!"
	result := ValidateConfig(cfg)

	// Filter to just namespace error
	result.Errors = result.Errors[:1]

	err := &ValidationError{Result: result}
	msg := err.Error()

	// Single error should be inline, not bulleted
	if strings.Contains(msg, "\n") {
		t.Errorf("single error should be inline, got: %s", msg)
	}
	if !strings.Contains(msg, "configuration validation failed:") {
		t.Errorf("expected 'configuration validation failed:', got: %s", msg)
	}
}

func assertErrorContains(t *testing.T, errs []error, substring string) {
	t.Helper()
	if !hasErrorContaining(errs, substring) {
		t.Fatalf("expected error containing %q", substring)
	}
}

func hasErrorContaining(errs []error, substring string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substring) {
			return true
		}
	}
	return false
}

func countErrorsContaining(errs []error, substring string) int {
	count := 0
	for _, err := range errs {
		if strings.Contains(err.Error(), substring) {
			count++
		}
	}
	return count
}
