package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// LoadOptions controls configuration loading behavior.
type LoadOptions struct {
	ConfigFile  string
	ConfigFiles []string
	Flags       *pflag.FlagSet
}

// LoadResult contains the merged configuration and validation output.
type LoadResult struct {
	Config         Config
	Validation     ValidationResult
	ConfigFileUsed string
}

// LoadConfig loads configuration from defaults, file, env, and flags.
func LoadConfig(opts LoadOptions) (LoadResult, error) {
	v := viper.New()
	setDefaults(v)
	configureEnv(v)

	if opts.Flags != nil {
		if err := BindFlags(v, opts.Flags); err != nil {
			return LoadResult{}, fmt.Errorf("bind flags: %w", err)
		}
	}

	configPath, err := resolveConfigFile(opts)
	if err != nil {
		return LoadResult{}, err
	}
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return LoadResult{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return LoadResult{}, fmt.Errorf("unmarshal config: %w", err)
	}
	applyNamespaceOverride(v, &cfg)

	validation := ValidateConfig(cfg)
	if validation.HasErrors() {
		return LoadResult{
			Config:         cfg,
			Validation:     validation,
			ConfigFileUsed: v.ConfigFileUsed(),
		}, fmt.Errorf("configuration validation failed")
	}

	return LoadResult{
		Config:         cfg,
		Validation:     validation,
		ConfigFileUsed: v.ConfigFileUsed(),
	}, nil
}

// BindFlags binds supported CLI flags to viper keys.
func BindFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	bindings := map[string]string{
		"kubeconfig":              "kubernetes.kubeconfig",
		"namespace":               "namespace",
		"rook-operator-namespace": "kubernetes.rook-operator-namespace",
		"rook-cluster-namespace":  "kubernetes.rook-cluster-namespace",
		"state-file":              "state.file-path-template",
		"log-level":               "logging.level",
		"log-file":                "logging.file",
	}

	for flag, key := range bindings {
		if flags.Lookup(flag) == nil {
			continue
		}
		if err := v.BindPFlag(key, flags.Lookup(flag)); err != nil {
			return fmt.Errorf("bind flag %q: %w", flag, err)
		}
	}

	return nil
}

func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	v.SetDefault("kubernetes.rook-operator-namespace", defaults.Kubernetes.RookOperatorNamespace)
	v.SetDefault("kubernetes.rook-cluster-namespace", defaults.Kubernetes.RookClusterNamespace)
	v.SetDefault("kubernetes.kubeconfig", defaults.Kubernetes.Kubeconfig)
	v.SetDefault("kubernetes.context", defaults.Kubernetes.Context)

	v.SetDefault("state.file-path-template", defaults.State.FilePathTemplate)
	v.SetDefault("state.backup-enabled", defaults.State.BackupEnabled)
	v.SetDefault("state.backup-directory", defaults.State.BackupDirectory)

	v.SetDefault("deployment-filters.prefixes", defaults.DeploymentFilters.Prefixes)

	v.SetDefault("ui.theme", defaults.UI.Theme)
	v.SetDefault("ui.progress-refresh-ms", defaults.UI.ProgressRefreshMS)
	v.SetDefault("ui.dashboard-refresh-node-ms", defaults.UI.DashboardRefreshNodeMS)
	v.SetDefault("ui.dashboard-refresh-ceph-ms", defaults.UI.DashboardRefreshCephMS)

	v.SetDefault("timeouts.api-call-timeout-seconds", defaults.Timeouts.APICallTimeoutSeconds)
	v.SetDefault("timeouts.wait-deployment-timeout-seconds", defaults.Timeouts.WaitDeploymentTimeoutSeconds)
	v.SetDefault("timeouts.ceph-command-timeout-seconds", defaults.Timeouts.CephCommandTimeoutSeconds)

	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.file", defaults.Logging.File)
	v.SetDefault("logging.format", defaults.Logging.Format)
}

func configureEnv(v *viper.Viper) {
	replacer := strings.NewReplacer(".", "_", "-", "_")
	v.SetEnvKeyReplacer(replacer)
	v.SetEnvPrefix("CROOK")
	v.AutomaticEnv()
}

func resolveConfigFile(opts LoadOptions) (string, error) {
	if opts.ConfigFile != "" {
		if _, err := os.Stat(opts.ConfigFile); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("config file not found: %s", opts.ConfigFile)
			}
			return "", fmt.Errorf("config file error: %w", err)
		}
		return opts.ConfigFile, nil
	}

	candidates := opts.ConfigFiles
	if len(candidates) == 0 {
		candidates = defaultConfigFiles()
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("config file error: %w", err)
		}
		if info.IsDir() {
			continue
		}
		return candidate, nil
	}

	return "", nil
}

func defaultConfigFiles() []string {
	files := []string{"./crook.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".config", "crook", "config.yaml"))
	}
	files = append(files, "/etc/crook/config.yaml")
	return files
}

func applyNamespaceOverride(v *viper.Viper, cfg *Config) {
	if cfg == nil {
		return
	}
	namespace := strings.TrimSpace(v.GetString("namespace"))
	if namespace == "" {
		return
	}
	cfg.Kubernetes.RookOperatorNamespace = namespace
	cfg.Kubernetes.RookClusterNamespace = namespace
}
