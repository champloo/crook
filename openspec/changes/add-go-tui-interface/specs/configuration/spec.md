# Capability: Configuration

Configuration management with layered precedence: CLI flags, environment variables, config file, defaults.

## ADDED Requirements

### Requirement: Configuration Loading

The system SHALL load configuration from multiple sources with defined precedence order.

#### Scenario: Configuration precedence

- **WHEN** system initializes configuration
- **THEN** system loads defaults first
- **THEN** system loads config file if it exists (lowest precedence)
- **THEN** system overrides with environment variables (CROOK_* prefix)
- **THEN** system overrides with CLI flags (highest precedence)
- **THEN** system validates final merged configuration

#### Scenario: Config file discovery

- **WHEN** user does not specify `--config` flag
- **THEN** system searches for config file in order:
  1. `./crook.yaml` (current directory)
  2. `~/.config/crook/config.yaml` (user config)
  3. `/etc/crook/config.yaml` (system config)
- **THEN** system uses first file found
- **THEN** system continues with defaults if no config file found

#### Scenario: Explicit config file

- **WHEN** user specifies `--config /path/to/config.yaml`
- **THEN** system loads config from specified path
- **THEN** system returns error if file does not exist
- **THEN** system returns error if file is not valid YAML

### Requirement: Configuration Schema

The system SHALL support configuration options for all operational parameters.

#### Scenario: Kubernetes configuration

- **WHEN** config file contains kubernetes section
- **THEN** system reads:
  - `rook-operator-namespace` (default: "rook-ceph")
  - `rook-cluster-namespace` (default: "rook-ceph")
  - `kubeconfig` (default: standard kubeconfig locations)
  - `context` (optional, default: current context)

#### Scenario: State file configuration

- **WHEN** config file contains state section
- **THEN** system reads:
  - `file-path-template` (default: "./crook-state-{{.Node}}.json")
  - `backup-enabled` (default: true)
  - `backup-directory` (default: "~/.local/state/crook/backups")

#### Scenario: Deployment filter configuration

- **WHEN** config file contains deployment-filters section
- **THEN** system reads:
  - `prefixes` (array of strings, default: ["rook-ceph-osd", "rook-ceph-mon", "rook-ceph-exporter", "rook-ceph-crashcollector"])

#### Scenario: UI configuration

- **WHEN** config file contains ui section
- **THEN** system reads:
  - `theme` (default: "default")
  - `progress-refresh-ms` (default: 100)
  - `dashboard-refresh-node-ms` (default: 2000)
  - `dashboard-refresh-ceph-ms` (default: 5000)

#### Scenario: Timeout configuration

- **WHEN** config file contains timeouts section
- **THEN** system reads:
  - `api-call-timeout-seconds` (default: 30)
  - `wait-deployment-timeout-seconds` (default: 300)
  - `ceph-command-timeout-seconds` (default: 60)

### Requirement: Environment Variable Overrides

The system SHALL support environment variables with CROOK_ prefix.

#### Scenario: Environment variable mapping

- **WHEN** environment variable `CROOK_KUBERNETES_ROOK_OPERATOR_NAMESPACE=custom-rook` is set
- **THEN** system overrides `kubernetes.rook-operator-namespace` with "custom-rook"
- **WHEN** environment variable `CROOK_STATE_FILE_PATH_TEMPLATE=/tmp/state-{{.Node}}.json` is set
- **THEN** system overrides `state.file-path-template` with specified value

#### Scenario: Nested configuration via env vars

- **WHEN** environment variables use underscores to denote nesting
- **THEN** system maps `CROOK_A_B_C` to config path `a.b.c`
- **THEN** system converts to lowercase (CROOK_FOO_BAR → foo.bar)
- **THEN** system replaces hyphens with underscores in env var names

### Requirement: CLI Flag Overrides

The system SHALL provide CLI flags for common configuration options.

#### Scenario: Global flags

- **WHEN** user specifies CLI flags
- **THEN** system accepts:
  - `--config <path>` - Config file path
  - `--kubeconfig <path>` - Kubernetes config file
  - `--namespace <name>` - Override both operator and cluster namespaces
  - `--rook-operator-namespace <name>` - Operator namespace
  - `--rook-cluster-namespace <name>` - Cluster namespace
  - `--state-file <path>` - State file path (overrides template)
  - `--log-level <level>` - Log level (debug, info, warn, error)
  - `--log-file <path>` - Log file path

#### Scenario: Command-specific flags

- **WHEN** user runs `crook down` command
- **THEN** system accepts:
  - `--force` - Skip confirmation prompts
  - `--no-backup` - Disable state file backup
- **WHEN** user runs `crook up` command
- **THEN** system accepts:
  - `--force` - Skip confirmation prompts
  - `--state-file <path>` - Explicit state file path
- **WHEN** user runs `crook ls` command
- **THEN** system accepts:
  - `--output <format>` or `-o` - Output format: tui (default), table, json, yaml
  - `--all-namespaces` or `-A` - Show resources across all namespaces (not just configured ones)
  - `--show <types>` - Comma-separated list of resource types to show: nodes,deployments,osds,pods (default: all)

### Requirement: Configuration Validation

The system SHALL validate configuration values and provide clear error messages.

#### Scenario: Validate namespace names

- **WHEN** namespace configuration is invalid (empty string, invalid characters)
- **THEN** system returns error: "Invalid namespace '<value>': must be non-empty and match Kubernetes naming rules"
- **THEN** system exits with error code 1

#### Scenario: Validate file paths

- **WHEN** state file template contains invalid placeholder
- **THEN** system returns error: "Invalid state file template: unknown placeholder '{{.Foo}}', valid: {{.Node}}"
- **WHEN** kubeconfig path does not exist
- **THEN** system returns error: "Kubeconfig file not found: <path>"

#### Scenario: Validate numeric ranges

- **WHEN** timeout configuration is < 1 second
- **THEN** system returns error: "Timeout must be >= 1 second, got: <value>"
- **WHEN** refresh rate is < 100ms
- **THEN** system returns warning: "Refresh rate <100ms may cause excessive API calls"
- **THEN** system continues with warning (non-fatal)

### Requirement: Configuration Display

The system SHALL provide command to display effective configuration.

#### Scenario: Show merged configuration

- **WHEN** user runs `crook config show`
- **THEN** system displays complete merged configuration from all sources
- **THEN** system indicates source for each value (default, file, env, flag)
- **THEN** system masks sensitive values (kubeconfig tokens, if any)

#### Scenario: Validate configuration

- **WHEN** user runs `crook config validate`
- **THEN** system loads configuration from all sources
- **THEN** system validates all values
- **THEN** system displays validation results: OK or list of errors
- **THEN** system exits with code 0 if valid, 1 if invalid

### Requirement: Configuration File Format

The system SHALL support YAML configuration file format.

#### Scenario: Example valid configuration

- **WHEN** config file contains:
```yaml
kubernetes:
  rook-operator-namespace: rook-ceph
  rook-cluster-namespace: rook-ceph
  kubeconfig: ~/.kube/config

state:
  file-path-template: "./crook-state-{{.Node}}.json"
  backup-enabled: true

deployment-filters:
  prefixes:
    - rook-ceph-osd
    - rook-ceph-mon
    - rook-ceph-exporter

ui:
  theme: default
  progress-refresh-ms: 100

timeouts:
  api-call-timeout-seconds: 30
  wait-deployment-timeout-seconds: 300
```
- **THEN** system parses and applies all values

#### Scenario: Partial configuration

- **WHEN** config file contains only subset of options
- **THEN** system uses defaults for missing values
- **THEN** system merges partial config with defaults
- **THEN** system does not error on missing optional values
