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
	if cfg.State.FilePathTemplate != config.DefaultStateFileTemplate {
		t.Fatalf("expected state file template default %q, got %q", config.DefaultStateFileTemplate, cfg.State.FilePathTemplate)
	}
	if !cfg.State.BackupEnabled {
		t.Fatalf("expected backup enabled by default")
	}
	if len(cfg.DeploymentFilters.Prefixes) == 0 {
		t.Fatalf("expected default deployment prefixes")
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
state:
  file-path-template: "./file-{{.Node}}.json"
`)
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("rook-operator-namespace", "", "")
	flags.String("state-file", "", "")
	if err := flags.Set("rook-operator-namespace", "flag-op"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	if err := flags.Set("state-file", "/tmp/flag-state.json"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	t.Setenv("CROOK_KUBERNETES_ROOK_OPERATOR_NAMESPACE", "env-op")
	t.Setenv("CROOK_STATE_FILE_PATH_TEMPLATE", "./env-{{.Node}}.json")

	result, err := config.LoadConfig(config.LoadOptions{ConfigFile: configPath, Flags: flags})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	cfg := result.Config
	if cfg.Kubernetes.RookOperatorNamespace != "flag-op" {
		t.Fatalf("expected flag override, got %q", cfg.Kubernetes.RookOperatorNamespace)
	}
	if cfg.State.FilePathTemplate != "/tmp/flag-state.json" {
		t.Fatalf("expected flag override for state file, got %q", cfg.State.FilePathTemplate)
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
	if cfg.State.BackupEnabled {
		t.Fatalf("expected backup disabled from file")
	}
	if cfg.State.BackupDirectory != "/tmp/backups" {
		t.Fatalf("expected backup directory from file, got %q", cfg.State.BackupDirectory)
	}
	if len(cfg.DeploymentFilters.Prefixes) != 2 || cfg.DeploymentFilters.Prefixes[0] != "custom-a" {
		t.Fatalf("expected deployment prefixes from file, got %v", cfg.DeploymentFilters.Prefixes)
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
	if cfg.State.FilePathTemplate != config.DefaultStateFileTemplate {
		t.Fatalf("expected default state file template, got %q", cfg.State.FilePathTemplate)
	}
	if cfg.State.BackupEnabled {
		t.Fatalf("expected backup disabled from file")
	}
	if cfg.UI.ProgressRefreshMS != config.DefaultProgressRefreshMS {
		t.Fatalf("expected default progress refresh, got %d", cfg.UI.ProgressRefreshMS)
	}
	if !strings.EqualFold(cfg.UI.Theme, "minimal") {
		t.Fatalf("expected ui theme from file, got %q", cfg.UI.Theme)
	}
	if len(cfg.DeploymentFilters.Prefixes) != len(config.DefaultDeploymentPrefixes) {
		t.Fatalf("expected default deployment prefixes, got %v", cfg.DeploymentFilters.Prefixes)
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

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}
