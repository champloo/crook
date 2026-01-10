package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRookNamespace                = "rook-ceph"
	DefaultK8sRefreshMS                 = 2000 // Kubernetes API resources (nodes, deployments, pods)
	DefaultCephRefreshMS                = 5000 // Ceph CLI operations (OSDs, header)
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
	// K8sRefreshMS is the refresh interval for Kubernetes API resources (nodes, deployments, pods)
	K8sRefreshMS int `mapstructure:"k8s-refresh-ms" yaml:"k8s-refresh-ms" json:"k8s-refresh-ms"`

	// CephRefreshMS is the refresh interval for Ceph CLI operations (OSDs, header)
	CephRefreshMS int `mapstructure:"ceph-refresh-ms" yaml:"ceph-refresh-ms" json:"ceph-refresh-ms"`
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
			K8sRefreshMS:  DefaultK8sRefreshMS,
			CephRefreshMS: DefaultCephRefreshMS,
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
