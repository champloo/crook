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
		if bindErr := BindFlags(v, opts.Flags); bindErr != nil {
			return LoadResult{}, fmt.Errorf("bind flags: %w", bindErr)
		}
	}

	configPath, err := resolveConfigFile(opts)
	if err != nil {
		return LoadResult{}, err
	}
	if configPath != "" {
		v.SetConfigFile(configPath)
		if readErr := v.ReadInConfig(); readErr != nil {
			return LoadResult{}, fmt.Errorf("read config: %w", readErr)
		}
	}

	var cfg Config
	if unmarshalErr := v.Unmarshal(&cfg); unmarshalErr != nil {
		return LoadResult{}, fmt.Errorf("unmarshal config: %w", unmarshalErr)
	}
	applyKubernetesDefaults(v, &cfg)
	applyNamespaceOverride(v, &cfg)

	validation := ValidateConfig(cfg)

	// Check for unknown keys (only if config file was loaded)
	// Unknown keys are errors to catch typos that would silently use defaults
	if configPath != "" {
		unknownKeys := detectUnknownKeys(v)
		for _, key := range unknownKeys {
			validation.Errors = append(validation.Errors, fmt.Errorf("unknown config key: %s", key))
		}
	}

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
		"namespace":               "namespace",
		"rook-operator-namespace": "kubernetes.rook-operator-namespace",
		"rook-cluster-namespace":  "kubernetes.rook-cluster-namespace",
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

	// Note: kubernetes.* settings are NOT set here as viper defaults.
	// They're handled via CLI flags and applied directly to the Config struct.
	// This prevents detectUnknownKeys from seeing them as config file entries.

	v.SetDefault("ui.progress-refresh-ms", defaults.UI.ProgressRefreshMS)
	v.SetDefault("ui.ls-refresh-nodes-ms", defaults.UI.LsRefreshNodesMS)
	v.SetDefault("ui.ls-refresh-deployments-ms", defaults.UI.LsRefreshDeploymentsMS)
	v.SetDefault("ui.ls-refresh-pods-ms", defaults.UI.LsRefreshPodsMS)
	v.SetDefault("ui.ls-refresh-osds-ms", defaults.UI.LsRefreshOSDsMS)
	v.SetDefault("ui.ls-refresh-header-ms", defaults.UI.LsRefreshHeaderMS)

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

// applyKubernetesDefaults applies default values for kubernetes settings.
// These are handled via CLI flags only, not config files, so we apply defaults
// directly to the Config struct after unmarshaling.
func applyKubernetesDefaults(v *viper.Viper, cfg *Config) {
	if cfg == nil {
		return
	}
	defaults := DefaultConfig()

	// Check for flag or env overrides first, otherwise use defaults
	if operatorNS := strings.TrimSpace(v.GetString("kubernetes.rook-operator-namespace")); operatorNS != "" {
		cfg.Kubernetes.RookOperatorNamespace = operatorNS
	} else if cfg.Kubernetes.RookOperatorNamespace == "" {
		cfg.Kubernetes.RookOperatorNamespace = defaults.Kubernetes.RookOperatorNamespace
	}

	if clusterNS := strings.TrimSpace(v.GetString("kubernetes.rook-cluster-namespace")); clusterNS != "" {
		cfg.Kubernetes.RookClusterNamespace = clusterNS
	} else if cfg.Kubernetes.RookClusterNamespace == "" {
		cfg.Kubernetes.RookClusterNamespace = defaults.Kubernetes.RookClusterNamespace
	}
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

// knownConfigKeys returns the set of all valid configuration keys for config FILES.
// Note: kubernetes.* keys are handled via CLI flags only, not config files.
func knownConfigKeys() map[string]bool {
	return map[string]bool{
		// Top-level sections (kubernetes not allowed in config files)
		"ui":       true,
		"timeouts": true,
		"logging":  true,

		// ui section
		"ui.progress-refresh-ms":       true,
		"ui.ls-refresh-nodes-ms":       true,
		"ui.ls-refresh-deployments-ms": true,
		"ui.ls-refresh-pods-ms":        true,
		"ui.ls-refresh-osds-ms":        true,
		"ui.ls-refresh-header-ms":      true,

		// timeouts section
		"timeouts.api-call-timeout-seconds":        true,
		"timeouts.wait-deployment-timeout-seconds": true,
		"timeouts.ceph-command-timeout-seconds":    true,

		// logging section
		"logging.level":  true,
		"logging.file":   true,
		"logging.format": true,

		// Special keys (CLI-only via flag binding)
		"namespace": true,
	}
}

// detectUnknownKeys returns a list of configuration keys that are not in the known schema
func detectUnknownKeys(v *viper.Viper) []string {
	known := knownConfigKeys()
	allSettings := v.AllSettings()

	var unknown []string
	collectUnknownKeys(allSettings, "", known, &unknown)
	return unknown
}

// collectUnknownKeys recursively collects unknown keys from nested maps
func collectUnknownKeys(settings map[string]interface{}, prefix string, known map[string]bool, unknown *[]string) {
	for key, value := range settings {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		// Check if this key is known
		if !known[fullKey] {
			// Skip nested keys under unknown parent (parent already reported)
			if prefix != "" && !known[prefix] {
				continue
			}
			// Report unknown key (top-level or under known parent)
			*unknown = append(*unknown, fullKey)
		}

		// Recurse into nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			collectUnknownKeys(nestedMap, fullKey, known, unknown)
		}
	}
}
