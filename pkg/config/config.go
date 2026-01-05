package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRookNamespace                = "rook-ceph"
	DefaultStateFileTemplate            = "./crook-state-{{.Node}}.json"
	DefaultStateBackupDirectory         = "~/.local/state/crook/backups"
	DefaultTheme                        = "default"
	DefaultProgressRefreshMS            = 100
	DefaultDashboardRefreshNodeMS       = 2000
	DefaultDashboardRefreshCephMS       = 5000
	DefaultAPICallTimeoutSeconds        = 30
	DefaultWaitDeploymentTimeoutSeconds = 300
	DefaultCephCommandTimeoutSeconds    = 10
	DefaultLogLevel                     = "info"
	DefaultLogFormat                    = "text"
)

var DefaultDeploymentPrefixes = []string{
	"rook-ceph-osd",
	"rook-ceph-mon",
	"rook-ceph-exporter",
	"rook-ceph-crashcollector",
}

// Config holds the full configuration schema for crook.
type Config struct {
	Kubernetes        KubernetesConfig       `mapstructure:"kubernetes" yaml:"kubernetes" json:"kubernetes"`
	State             StateConfig            `mapstructure:"state" yaml:"state" json:"state"`
	DeploymentFilters DeploymentFilterConfig `mapstructure:"deployment-filters" yaml:"deployment-filters" json:"deployment-filters"`
	UI                UIConfig               `mapstructure:"ui" yaml:"ui" json:"ui"`
	Timeouts          TimeoutConfig          `mapstructure:"timeouts" yaml:"timeouts" json:"timeouts"`
	Logging           LoggingConfig          `mapstructure:"logging" yaml:"logging" json:"logging"`
}

// KubernetesConfig captures cluster-related settings.
type KubernetesConfig struct {
	RookOperatorNamespace string `mapstructure:"rook-operator-namespace" yaml:"rook-operator-namespace" json:"rook-operator-namespace" validate:"required"`
	RookClusterNamespace  string `mapstructure:"rook-cluster-namespace" yaml:"rook-cluster-namespace" json:"rook-cluster-namespace" validate:"required"`
	Kubeconfig            string `mapstructure:"kubeconfig" yaml:"kubeconfig" json:"kubeconfig"`
	Context               string `mapstructure:"context" yaml:"context" json:"context"`
}

// StateConfig controls state file behavior.
type StateConfig struct {
	FilePathTemplate string `mapstructure:"file-path-template" yaml:"file-path-template" json:"file-path-template" validate:"required"`
	BackupEnabled    bool   `mapstructure:"backup-enabled" yaml:"backup-enabled" json:"backup-enabled"`
	BackupDirectory  string `mapstructure:"backup-directory" yaml:"backup-directory" json:"backup-directory"`
}

// DeploymentFilterConfig defines which deployments are in scope for maintenance.
type DeploymentFilterConfig struct {
	Prefixes []string `mapstructure:"prefixes" yaml:"prefixes" json:"prefixes" validate:"required"`
}

// UIConfig holds terminal UI settings.
type UIConfig struct {
	Theme                  string `mapstructure:"theme" yaml:"theme" json:"theme" validate:"required"`
	ProgressRefreshMS      int    `mapstructure:"progress-refresh-ms" yaml:"progress-refresh-ms" json:"progress-refresh-ms"`
	DashboardRefreshNodeMS int    `mapstructure:"dashboard-refresh-node-ms" yaml:"dashboard-refresh-node-ms" json:"dashboard-refresh-node-ms"`
	DashboardRefreshCephMS int    `mapstructure:"dashboard-refresh-ceph-ms" yaml:"dashboard-refresh-ceph-ms" json:"dashboard-refresh-ceph-ms"`
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
	prefixes := make([]string, 0, len(DefaultDeploymentPrefixes))
	prefixes = append(prefixes, DefaultDeploymentPrefixes...)

	return Config{
		Kubernetes: KubernetesConfig{
			RookOperatorNamespace: DefaultRookNamespace,
			RookClusterNamespace:  DefaultRookNamespace,
			Kubeconfig:            "",
			Context:               "",
		},
		State: StateConfig{
			FilePathTemplate: DefaultStateFileTemplate,
			BackupEnabled:    true,
			BackupDirectory:  DefaultStateBackupDirectory,
		},
		DeploymentFilters: DeploymentFilterConfig{
			Prefixes: prefixes,
		},
		UI: UIConfig{
			Theme:                  DefaultTheme,
			ProgressRefreshMS:      DefaultProgressRefreshMS,
			DashboardRefreshNodeMS: DefaultDashboardRefreshNodeMS,
			DashboardRefreshCephMS: DefaultDashboardRefreshCephMS,
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
