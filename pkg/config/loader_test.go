package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andri/crook/pkg/config"
	"github.com/spf13/pflag"
)

func TestLoadConfigDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("# empty\n"), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected rook operator namespace default %q, got %q", config.DefaultRookNamespace, cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Timeouts.APICallTimeoutSeconds != config.DefaultAPICallTimeoutSeconds {
		t.Fatalf("expected api timeout default %d, got %d", config.DefaultAPICallTimeoutSeconds, cfg.Timeouts.APICallTimeoutSeconds)
	}
	if result.Validation.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Validation.Errors)
	}
}

func TestLoadConfigPrecedence(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContents := []byte(`kubernetes:
  rook-operator-namespace: file-op
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("rook-operator-namespace", "", "")
	if err := flags.Set("rook-operator-namespace", "flag-op"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	t.Setenv("CROOK_KUBERNETES_ROOK_OPERATOR_NAMESPACE", "env-op")

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath, Flags: flags})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "flag-op" {
		t.Fatalf("expected flag override, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
}

func TestLoadConfigNamespaceOverride(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContents := []byte(`kubernetes:
  rook-operator-namespace: file-op
  rook-cluster-namespace: file-cluster
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("namespace", "", "")
	if err := flags.Set("namespace", "shared"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath, Flags: flags})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "shared" {
		t.Fatalf("expected namespace override for operator, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Kubernetes.RookClusterNamespace != "shared" {
		t.Fatalf("expected namespace override for cluster, got %q", cfg.Kubernetes.RookClusterNamespace)
	}
}

func TestLoadConfigMissingExplicitFile(t *testing.T) {
	_, err := config.LoadConfig(config.LoadOptions{ConfigFile: filepath.Join(t.TempDir(), "missing.yaml")})
	if err == nil {
		t.Fatalf("expected error for missing config file")
	}
}

func TestLoadConfigFromFileFixture(t *testing.T) {
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: testdataPath(t, "full.yaml")})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "custom-operator" {
		t.Fatalf("expected operator namespace from file, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Kubernetes.RookClusterNamespace != "custom-cluster" {
		t.Fatalf("expected cluster namespace from file, got %q", cfg.Kubernetes.RookClusterNamespace)
	}
	if cfg.UI.Theme != "neon" {
		t.Fatalf("expected ui theme from file, got %q", cfg.UI.Theme)
	}
	if cfg.Timeouts.APICallTimeoutSeconds != 10 {
		t.Fatalf("expected api timeout from file, got %d", cfg.Timeouts.APICallTimeoutSeconds)
	}
	if cfg.Logging.Format != "json" {
		t.Fatalf("expected log format from file, got %q", cfg.Logging.Format)
	}
	if result.Validation.HasErrors() {
		t.Fatalf("unexpected validation errors: %v", result.Validation.Errors)
	}
}

func TestLoadConfigPartialUsesDefaults(t *testing.T) {
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: testdataPath(t, "partial.yaml")})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "partial-operator" {
		t.Fatalf("expected operator namespace from file, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Kubernetes.RookClusterNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected default cluster namespace, got %q", cfg.Kubernetes.RookClusterNamespace)
	}
	if cfg.UI.ProgressRefreshMS != config.DefaultProgressRefreshMS {
		t.Fatalf("expected default progress refresh, got %d", cfg.UI.ProgressRefreshMS)
	}
	if !strings.EqualFold(cfg.UI.Theme, "minimal") {
		t.Fatalf("expected ui theme from file, got %q", cfg.UI.Theme)
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	t.Setenv("CROOK_KUBERNETES_ROOK_OPERATOR_NAMESPACE", "env-operator")
	t.Setenv("CROOK_UI_PROGRESS_REFRESH_MS", "220")

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: testdataPath(t, "full.yaml")})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "env-operator" {
		t.Fatalf("expected env override for operator namespace, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.UI.ProgressRefreshMS != 220 {
		t.Fatalf("expected env override for progress refresh, got %d", cfg.UI.ProgressRefreshMS)
	}
}

func TestLoadConfigConfigFileDiscovery(t *testing.T) {
	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "missing.yaml")
	first := filepath.Join(tempDir, "first.yaml")
	second := filepath.Join(tempDir, "second.yaml")

	firstContents := []byte("kubernetes:\n  rook-operator-namespace: first\n")
	if err := os.WriteFile(first, firstContents, 0o600); err != nil {
		t.Fatalf("write first config: %v", err)
	}
	secondContents := []byte("kubernetes:\n  rook-operator-namespace: second\n")
	if err := os.WriteFile(second, secondContents, 0o600); err != nil {
		t.Fatalf("write second config: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{ConfigFiles: []string{missing, first, second}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if result.Config.Kubernetes.RookOperatorNamespace != "first" {
		t.Fatalf("expected first config file to win, got %q", result.Config.Kubernetes.RookOperatorNamespace)
	}
	if result.ConfigFileUsed != first {
		t.Fatalf("expected ConfigFileUsed %q, got %q", first, result.ConfigFileUsed)
	}
}

func TestLoadConfigNoConfigFileFound(t *testing.T) {
	tempDir := t.TempDir()
	result, err := config.LoadConfig(config.LoadOptions{ConfigFiles: []string{filepath.Join(tempDir, "missing.yaml")}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if result.ConfigFileUsed != "" {
		t.Fatalf("expected no config file used, got %q", result.ConfigFileUsed)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte("kubernetes: ["), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected error for invalid YAML")
	}
}

func TestLoadConfigUnknownTopLevelKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// unknown-section is not a valid config key
	configContents := []byte(`kubernetes:
  rook-operator-namespace: test
unknown-section:
  foo: bar
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Unknown keys should be warnings, not errors (backwards compatibility)
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("unknown config keys should not cause error (backwards compatibility): %v", err)
	}

	// Check that a warning was issued for the unknown key
	found := false
	for _, warning := range result.Validation.Warnings {
		if strings.Contains(warning, "unknown-section") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning for unknown-section, got warnings: %v", result.Validation.Warnings)
	}
}

func TestLoadConfigUnknownNestedKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// kubernetes.invalid-key is not a valid config key
	configContents := []byte(`kubernetes:
  rook-operator-namespace: test
  invalid-key: some-value
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Unknown keys should be warnings, not errors (backwards compatibility)
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("unknown config keys should not cause error (backwards compatibility): %v", err)
	}

	// Check that a warning was issued for the unknown nested key
	found := false
	for _, warning := range result.Validation.Warnings {
		if strings.Contains(warning, "kubernetes.invalid-key") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning for kubernetes.invalid-key, got warnings: %v", result.Validation.Warnings)
	}
}

func TestLoadConfigTypoInKnownKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// rook-operator-namesapce is a typo of rook-operator-namespace
	configContents := []byte(`kubernetes:
  rook-operator-namesapce: typo-value
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Typos in keys should be warnings, not errors (backwards compatibility)
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("typo in config key should not cause error (backwards compatibility): %v", err)
	}

	// Check that a warning was issued for the typo
	found := false
	for _, warning := range result.Validation.Warnings {
		if strings.Contains(warning, "rook-operator-namesapce") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning for typo key, got warnings: %v", result.Validation.Warnings)
	}
}

func TestLoadConfigValidKeysDoNotWarn(t *testing.T) {
	// Test that a config file with all valid keys doesn't report unknown keys
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: testdataPath(t, "full.yaml")})
	if err != nil {
		t.Fatalf("load config with valid keys should succeed: %v", err)
	}

	// Check no unknown key warnings
	for _, warning := range result.Validation.Warnings {
		if strings.Contains(warning, "unknown config key") {
			t.Errorf("unexpected unknown key warning for valid config: %v", warning)
		}
	}
}

func TestLoadConfigDeprecatedSectionsBackwardsCompatible(t *testing.T) {
	// Test that old configs with deprecated state and deployment-filters sections
	// still work (with warnings). This ensures backwards compatibility for users
	// upgrading from older versions that used state files.
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContents := []byte(`kubernetes:
  rook-operator-namespace: test
  rook-cluster-namespace: test

# Deprecated sections that should be ignored with warnings
state:
  file-path-template: "./crook-state-{{.Node}}.json"
  backup-enabled: true

deployment-filters:
  prefixes:
    - rook-ceph-osd
    - rook-ceph-mon
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Config should load successfully (deprecated sections are warnings, not errors)
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("config with deprecated sections should load successfully: %v", err)
	}

	// Verify the valid config was applied
	if result.Config.Kubernetes.RookOperatorNamespace != "test" {
		t.Errorf("expected rook-operator-namespace to be 'test', got %s",
			result.Config.Kubernetes.RookOperatorNamespace)
	}

	// Verify warnings were issued for deprecated sections
	foundState := false
	foundFilters := false
	for _, warning := range result.Validation.Warnings {
		if strings.Contains(warning, "state") {
			foundState = true
		}
		if strings.Contains(warning, "deployment-filters") {
			foundFilters = true
		}
	}
	if !foundState {
		t.Errorf("expected warning for deprecated 'state' section, got: %v", result.Validation.Warnings)
	}
	if !foundFilters {
		t.Errorf("expected warning for deprecated 'deployment-filters' section, got: %v", result.Validation.Warnings)
	}
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}
