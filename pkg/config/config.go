package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRookNamespace                = "rook-ceph"
	DefaultProgressRefreshMS            = 100
	DefaultLsRefreshNodesMS             = 2000
	DefaultLsRefreshDeploymentsMS       = 2000
	DefaultLsRefreshPodsMS              = 2000
	DefaultLsRefreshOSDsMS              = 5000
	DefaultLsRefreshHeaderMS            = 5000
	DefaultAPICallTimeoutSeconds        = 30
	DefaultWaitDeploymentTimeoutSeconds = 300
	DefaultCephCommandTimeoutSeconds    = 20
	DefaultLogLevel                     = "info"
	DefaultLogFormat                    = "text"
)

// Config holds the full configuration schema for crook.
type Config struct {
	// Namespace is the rook-ceph namespace (used for both operator and cluster).
	// Set via --namespace flag, CROOK_NAMESPACE env var, or "namespace:" in config file.
	Namespace string        `mapstructure:"namespace" yaml:"namespace" json:"namespace"`
	UI        UIConfig      `mapstructure:"ui" yaml:"ui" json:"ui"`
	Timeouts  TimeoutConfig `mapstructure:"timeouts" yaml:"timeouts" json:"timeouts"`
	Logging   LoggingConfig `mapstructure:"logging" yaml:"logging" json:"logging"`
}

// UIConfig holds terminal UI settings.
type UIConfig struct {
	ProgressRefreshMS int `mapstructure:"progress-refresh-ms" yaml:"progress-refresh-ms" json:"progress-refresh-ms"`

	// Ls refresh intervals (independent per resource type)
	LsRefreshNodesMS       int `mapstructure:"ls-refresh-nodes-ms" yaml:"ls-refresh-nodes-ms" json:"ls-refresh-nodes-ms"`
	LsRefreshDeploymentsMS int `mapstructure:"ls-refresh-deployments-ms" yaml:"ls-refresh-deployments-ms" json:"ls-refresh-deployments-ms"`
	LsRefreshPodsMS        int `mapstructure:"ls-refresh-pods-ms" yaml:"ls-refresh-pods-ms" json:"ls-refresh-pods-ms"`
	LsRefreshOSDsMS        int `mapstructure:"ls-refresh-osds-ms" yaml:"ls-refresh-osds-ms" json:"ls-refresh-osds-ms"`
	LsRefreshHeaderMS      int `mapstructure:"ls-refresh-header-ms" yaml:"ls-refresh-header-ms" json:"ls-refresh-header-ms"`
}

// TimeoutConfig captures configurable timeouts.
type TimeoutConfig struct {
	APICallTimeoutSeconds        int `mapstructure:"api-call-timeout-seconds" yaml:"api-call-timeout-seconds" json:"api-call-timeout-seconds"`
	WaitDeploymentTimeoutSeconds int `mapstructure:"wait-deployment-timeout-seconds" yaml:"wait-deployment-timeout-seconds" json:"wait-deployment-timeout-seconds"`
	CephCommandTimeoutSeconds    int `mapstructure:"ceph-command-timeout-seconds" yaml:"ceph-command-timeout-seconds" json:"ceph-command-timeout-seconds"`
}

// LoggingConfig controls log output settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level" json:"level"`
	File   string `mapstructure:"file" yaml:"file" json:"file"`
	Format string `mapstructure:"format" yaml:"format" json:"format"`
}

// DefaultConfig returns a config with all default values applied.
func DefaultConfig() Config {
	return Config{
		Namespace: DefaultRookNamespace,
		UI: UIConfig{
			ProgressRefreshMS:      DefaultProgressRefreshMS,
			LsRefreshNodesMS:       DefaultLsRefreshNodesMS,
			LsRefreshDeploymentsMS: DefaultLsRefreshDeploymentsMS,
			LsRefreshPodsMS:        DefaultLsRefreshPodsMS,
			LsRefreshOSDsMS:        DefaultLsRefreshOSDsMS,
			LsRefreshHeaderMS:      DefaultLsRefreshHeaderMS,
		},
		Timeouts: TimeoutConfig{
			APICallTimeoutSeconds:        DefaultAPICallTimeoutSeconds,
			WaitDeploymentTimeoutSeconds: DefaultWaitDeploymentTimeoutSeconds,
			CephCommandTimeoutSeconds:    DefaultCephCommandTimeoutSeconds,
		},
		Logging: LoggingConfig{
			Level:  DefaultLogLevel,
			File:   "",
			Format: DefaultLogFormat,
		},
	}
}

// String renders the configuration as YAML.
func (c Config) String() string {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("Config{error: %v}", err)
	}

	return strings.TrimSpace(string(data))
}
