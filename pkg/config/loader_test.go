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

func TestLoadConfigFlagOverridesDefault(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("rook-operator-namespace", "", "")
	if err := flags.Set("rook-operator-namespace", "flag-op"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{Flags: flags})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "flag-op" {
		t.Fatalf("expected flag override, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
}

func TestLoadConfigNamespaceOverride(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("namespace", "", "")
	if err := flags.Set("namespace", "shared"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{Flags: flags})
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
	// Kubernetes values should use defaults (not allowed in config files)
	if cfg.Kubernetes.RookOperatorNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected default operator namespace, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Kubernetes.RookClusterNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected default cluster namespace, got %q", cfg.Kubernetes.RookClusterNamespace)
	}
	if cfg.UI.ProgressRefreshMS != 150 {
		t.Fatalf("expected progress refresh from file, got %d", cfg.UI.ProgressRefreshMS)
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
	// Kubernetes values should use defaults (not allowed in config files)
	if cfg.Kubernetes.RookOperatorNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected default operator namespace, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.Kubernetes.RookClusterNamespace != config.DefaultRookNamespace {
		t.Fatalf("expected default cluster namespace, got %q", cfg.Kubernetes.RookClusterNamespace)
	}
	// partial.yaml sets progress-refresh-ms to 200
	if cfg.UI.ProgressRefreshMS != 200 {
		t.Fatalf("expected progress refresh from file, got %d", cfg.UI.ProgressRefreshMS)
	}
	// Other UI settings should use defaults
	if cfg.UI.LsRefreshNodesMS != config.DefaultLsRefreshNodesMS {
		t.Fatalf("expected default ls refresh nodes, got %d", cfg.UI.LsRefreshNodesMS)
	}
}

func TestLoadConfigEnvOverridesDefault(t *testing.T) {
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

	firstContents := []byte("logging:\n  level: debug\n")
	if err := os.WriteFile(first, firstContents, 0o600); err != nil {
		t.Fatalf("write first config: %v", err)
	}
	secondContents := []byte("logging:\n  level: warn\n")
	if err := os.WriteFile(second, secondContents, 0o600); err != nil {
		t.Fatalf("write second config: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{ConfigFiles: []string{missing, first, second}})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if result.Config.Logging.Level != "debug" {
		t.Fatalf("expected first config file to win, got %q", result.Config.Logging.Level)
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
	if err := os.WriteFile(configPath, []byte("ui: ["), 0o600); err != nil {
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
	configContents := []byte(`logging:
  level: info
unknown-section:
  foo: bar
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Unknown keys should cause validation errors
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected error for unknown config key")
	}

	// Check that an error was reported for the unknown key
	found := false
	for _, errMsg := range result.Validation.Errors {
		if strings.Contains(errMsg.Error(), "unknown-section") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for unknown-section, got errors: %v", result.Validation.Errors)
	}
}

func TestLoadConfigKubernetesInConfigFileIsError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// kubernetes section is no longer allowed in config files
	configContents := []byte(`kubernetes:
  rook-operator-namespace: test
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// kubernetes section should cause validation error
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected error for kubernetes section in config file")
	}

	// Check that an error was reported for the kubernetes section
	found := false
	for _, errMsg := range result.Validation.Errors {
		if strings.Contains(errMsg.Error(), "kubernetes") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for kubernetes section, got errors: %v", result.Validation.Errors)
	}
}

func TestLoadConfigUnknownNestedKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// ui.invalid-key is not a valid config key
	configContents := []byte(`ui:
  progress-refresh-ms: 100
  invalid-key: some-value
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Unknown keys should cause validation errors
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected error for unknown config key")
	}

	// Check that an error was reported for the unknown nested key
	found := false
	for _, errMsg := range result.Validation.Errors {
		if strings.Contains(errMsg.Error(), "ui.invalid-key") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for ui.invalid-key, got errors: %v", result.Validation.Errors)
	}
}

func TestLoadConfigTypoInKnownKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// progres-refresh-ms is a typo of progress-refresh-ms
	configContents := []byte(`ui:
  progres-refresh-ms: 200
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Typos in keys should cause validation errors
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected error for typo in config key")
	}

	// Check that an error was reported for the typo
	found := false
	for _, errMsg := range result.Validation.Errors {
		if strings.Contains(errMsg.Error(), "progres-refresh-ms") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for typo key, got errors: %v", result.Validation.Errors)
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

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}
