package config_test

import (
	"errors"
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
	if cfg.UI.K8sRefreshMS != 3000 {
		t.Fatalf("expected k8s refresh from file, got %d", cfg.UI.K8sRefreshMS)
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
	// partial.yaml sets k8s-refresh-ms to 3000
	if cfg.UI.K8sRefreshMS != 3000 {
		t.Fatalf("expected k8s refresh from file, got %d", cfg.UI.K8sRefreshMS)
	}
	// Other UI settings should use defaults
	if cfg.UI.CephRefreshMS != config.DefaultCephRefreshMS {
		t.Fatalf("expected default ceph refresh, got %d", cfg.UI.CephRefreshMS)
	}
}

func TestLoadConfigEnvOverridesDefault(t *testing.T) {
	t.Setenv("CROOK_NAMESPACE", "env-ns")
	t.Setenv("CROOK_UI_K8S_REFRESH_MS", "2500")

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
	if cfg.UI.K8sRefreshMS != 2500 {
		t.Fatalf("expected env override for k8s refresh, got %d", cfg.UI.K8sRefreshMS)
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

func TestLoadConfigValidationErrorActionable(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.yaml")
	// Invalid config: bad namespace, invalid log level
	configContents := []byte(`
namespace: "INVALID_NS!"
logging:
  level: "verbose"
  format: "xml"
ui:
  k8s-refresh-ms: 0
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath})
	if err == nil {
		t.Fatalf("expected validation error")
	}

	// Verify it's a ValidationError
	var validationErr *config.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *config.ValidationError, got %T", err)
	}

	// Verify error message contains actionable details
	msg := err.Error()
	if !strings.Contains(msg, "invalid namespace") {
		t.Errorf("expected namespace error in message, got: %s", msg)
	}
	if !strings.Contains(msg, "invalid logging.level") {
		t.Errorf("expected log level error in message, got: %s", msg)
	}
	if !strings.Contains(msg, "invalid logging.format") {
		t.Errorf("expected log format error in message, got: %s", msg)
	}
	if !strings.Contains(msg, "k8s-refresh-ms must be > 0") {
		t.Errorf("expected refresh interval error in message, got: %s", msg)
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
