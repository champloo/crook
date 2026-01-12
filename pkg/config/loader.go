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
	applyNamespaceDefault(v, &cfg)

	validation := ValidateConfig(cfg)

	if validation.HasErrors() {
		return LoadResult{
			Config:         cfg,
			Validation:     validation,
			ConfigFileUsed: v.ConfigFileUsed(),
		}, &ValidationError{Result: validation}
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
		"namespace": "namespace",
		"log-level": "logging.level",
		"log-file":  "logging.file",
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

	v.SetDefault("ui.k8s-refresh-ms", defaults.UI.K8sRefreshMS)
	v.SetDefault("ui.ceph-refresh-ms", defaults.UI.CephRefreshMS)

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

// applyNamespaceDefault applies default namespace if not set via config/env/flag.
func applyNamespaceDefault(v *viper.Viper, cfg *Config) {
	if cfg == nil {
		return
	}
	// Check if namespace was set via any source (config file, env, or flag)
	namespace := strings.TrimSpace(v.GetString("namespace"))
	if namespace != "" {
		cfg.Namespace = namespace
	} else if cfg.Namespace == "" {
		cfg.Namespace = DefaultRookNamespace
	}
}
