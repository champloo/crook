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

#### Scenario: Namespace configuration

- **WHEN** config file contains namespace field
- **THEN** system reads `namespace` (default: "rook-ceph")

#### Scenario: UI configuration

- **WHEN** config file contains ui section
- **THEN** system reads:
  - `k8s-refresh-ms` (default: 2000) - all Kubernetes API resources (nodes, deployments, pods)
  - `ceph-refresh-ms` (default: 5000) - all Ceph CLI operations (OSDs, header)

#### Scenario: Timeout configuration

- **WHEN** config file contains timeouts section
- **THEN** system reads:
  - `api-call-timeout-seconds` (default: 30)
  - `wait-deployment-timeout-seconds` (default: 300)
  - `ceph-command-timeout-seconds` (default: 20)

#### Scenario: Logging configuration

- **WHEN** config file contains logging section
- **THEN** system reads:
  - `level` (default: "info") - one of debug, info, warn, error
  - `file` (default: "") - optional log file path
  - `format` (default: "text") - one of text, json

#### Scenario: Example valid configuration

- **WHEN** config file contains:
```yaml
namespace: rook-ceph

ui:
  k8s-refresh-ms: 2000
  ceph-refresh-ms: 5000

timeouts:
  api-call-timeout-seconds: 30
  wait-deployment-timeout-seconds: 300
  ceph-command-timeout-seconds: 20

logging:
  level: info
  format: text
```
- **THEN** system parses and applies all values

### Requirement: Environment Variable Overrides

The system SHALL support environment variables with CROOK_ prefix.

#### Scenario: Environment variable mapping

- **WHEN** environment variable `CROOK_NAMESPACE=custom-rook` is set
- **THEN** system overrides `namespace` with "custom-rook"
- **WHEN** environment variable `CROOK_UI_K8S_REFRESH_MS=3000` is set
- **THEN** system overrides `ui.k8s-refresh-ms` with 3000

#### Scenario: Nested configuration via env vars

- **WHEN** environment variables use underscores to denote nesting
- **THEN** system maps `CROOK_A_B_C` to config path `a.b.c`
- **THEN** system converts to lowercase (CROOK_FOO_BAR → foo.bar)

### Requirement: Configuration Validation

The system SHALL validate configuration values and provide clear error messages.

#### Scenario: Validate namespace names

- **WHEN** namespace configuration is invalid (empty string, invalid characters)
- **THEN** system returns error: "Invalid namespace '<value>': must be non-empty and match Kubernetes naming rules"
- **THEN** system exits with error code 1

#### Scenario: Validate numeric ranges

- **WHEN** timeout configuration is < 1 second
- **THEN** system returns error: "Timeout must be >= 1 second, got: <value>"
- **WHEN** refresh rate is < 100ms
- **THEN** system returns warning: "Refresh rate <100ms may cause excessive API calls"
- **THEN** system continues with warning (non-fatal)

### Requirement: Configuration File Format

The system SHALL support YAML configuration file format.

#### Scenario: Partial configuration

- **WHEN** config file contains only subset of options
- **THEN** system uses defaults for missing values
- **THEN** system merges partial config with defaults
- **THEN** system does not error on missing optional values
