package config_test

import (
	"os"
	"path/filepath"
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
	if cfg.Namespace != config.DefaultRookNamespace {
		t.Fatalf("expected rook operator namespace default %q, got %q", config.DefaultRookNamespace, cfg.Namespace)
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
	flags.String("namespace", "", "")
	if err := flags.Set("namespace", "flag-ns"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{Flags: flags})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Namespace != "flag-ns" {
		t.Fatalf("expected flag override for operator namespace, got %q", cfg.Namespace)
	}
	if cfg.Namespace != "flag-ns" {
		t.Fatalf("expected flag override for cluster namespace, got %q", cfg.Namespace)
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
	if cfg.Namespace != "shared" {
		t.Fatalf("expected namespace override for operator, got %q", cfg.Namespace)
	}
	if cfg.Namespace != "shared" {
		t.Fatalf("expected namespace override for cluster, got %q", cfg.Namespace)
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
	// Kubernetes values use defaults (yaml:"-" tag excludes from YAML parsing)
	if cfg.Namespace != config.DefaultRookNamespace {
		t.Fatalf("expected default operator namespace, got %q", cfg.Namespace)
	}
	if cfg.Namespace != config.DefaultRookNamespace {
		t.Fatalf("expected default cluster namespace, got %q", cfg.Namespace)
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
	// Kubernetes values use defaults (yaml:"-" tag excludes from YAML parsing)
	if cfg.Namespace != config.DefaultRookNamespace {
		t.Fatalf("expected default operator namespace, got %q", cfg.Namespace)
	}
	if cfg.Namespace != config.DefaultRookNamespace {
		t.Fatalf("expected default cluster namespace, got %q", cfg.Namespace)
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
	t.Setenv("CROOK_NAMESPACE", "env-ns")
	t.Setenv("CROOK_UI_PROGRESS_REFRESH_MS", "220")

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: testdataPath(t, "full.yaml")})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Namespace != "env-ns" {
		t.Fatalf("expected env override for operator namespace, got %q", cfg.Namespace)
	}
	if cfg.Namespace != "env-ns" {
		t.Fatalf("expected env override for cluster namespace, got %q", cfg.Namespace)
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

func TestLoadConfigUnknownKeysIgnored(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// Unknown keys should be silently ignored (idiomatic Viper behavior)
	configContents := []byte(`logging:
  level: info
unknown-section:
  foo: bar
ui:
  unknown-key: value
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	// Should succeed - unknown keys are ignored
	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("load config should succeed with unknown keys: %v", err)
	}

	// Known keys should still be parsed correctly
	if result.Config.Logging.Level != "info" {
		t.Errorf("expected logging.level=info, got %q", result.Config.Logging.Level)
	}
}

func TestLoadConfigKubernetesInConfigFileIgnored(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// kubernetes section in config file is ignored (mapstructure:"-" on Kubernetes field)
	// Use "namespace:" key instead
	configContents := []byte(`kubernetes:
  rook-operator-namespace: from-file
  rook-cluster-namespace: from-file
logging:
  level: debug
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("load config should succeed: %v", err)
	}

	// kubernetes section is ignored - uses defaults
	if result.Config.Namespace != config.DefaultRookNamespace {
		t.Errorf("expected default namespace, got %q", result.Config.Namespace)
	}
	// Other values should still be parsed
	if result.Config.Logging.Level != "debug" {
		t.Errorf("expected logging.level=debug, got %q", result.Config.Logging.Level)
	}
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}
