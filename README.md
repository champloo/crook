# crook

A Kubernetes node maintenance automation tool for Rook-Ceph clusters. Safely manage the process of taking nodes down for maintenance and bringing them back up while preserving Ceph cluster health and state.

## Features

- **Safe Node Maintenance** - Automated procedures that prevent data loss during node maintenance
- **Ceph Health Protection** - Manages OSDs, monitors, and noout flags to prevent rebalancing
- **No External State** - Uses Kubernetes nodeSelector as source of truth for deployment discovery
- **Interactive TUI** - Real-time feedback with tabbed views for nodes, deployments, OSDs, and pods
- **Pre-flight Validation** - Health checks before operations begin

## Installation

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/andri/crook/releases).

### Build from Source

**Requirements:**
- Go 1.25+
- Access to a Kubernetes cluster with Rook-Ceph

```bash
# Clone the repository
git clone https://github.com/andri/crook.git
cd crook

# Build
go build -o crook ./cmd/crook

# Or use just (if installed)
just build

# Install to GOPATH/bin
just install
```

## Quick Start

### Interactive TUI

crook provides a rich terminal user interface with tabbed views showing real-time cluster state:

<div align="center">
  <table>
    <tr>
      <td align="center">
        <strong>Nodes View</strong><br>
        <img src="docs/images/nodes.png" alt="Nodes view" width="400">
      </td>
      <td align="center">
        <strong>Deployments View</strong><br>
        <img src="docs/images/deployments.png" alt="Deployments view" width="400">
      </td>
    </tr>
    <tr>
      <td align="center">
        <strong>OSDs View</strong><br>
        <img src="docs/images/osds.png" alt="OSDs view" width="400">
      </td>
      <td align="center">
        <strong>Pods View</strong><br>
        <img src="docs/images/pods.png" alt="Pods view" width="400">
      </td>
    </tr>
  </table>
</div>

### Prerequisites

1. A Kubernetes cluster with Rook-Ceph deployed
2. Valid kubeconfig with permissions to:
   - Get/list/patch nodes
   - Get/list/patch deployments
   - Get/list pods and exec into rook-ceph-tools
   - Get/list replicasets
3. The `rook-ceph-tools` deployment running in your cluster

### Basic Usage

**Taking a node down for maintenance:**

```bash
crook down worker-1
```

**Restoring a node after maintenance:**

```bash
crook up worker-1
```

**Listing cluster resources:**

```bash
# Interactive TUI with tabbed views
crook ls

# Filter by node
crook ls worker-1

# Table output for scripting
crook ls --output table

# JSON output for automation
crook ls --output json
```

## Commands

### `crook down <node>`

Prepare a node for maintenance by safely scaling down Rook-Ceph workloads.

**What it does:**
1. Validates pre-flight conditions (node exists, Ceph healthy)
2. Cordons the node (marks it unschedulable)
3. Sets the Ceph `noout` flag to prevent data rebalancing
4. Scales down the rook-ceph-operator
5. Discovers node-pinned deployments via nodeSelector and scales them to 0

**Flags:**
| Flag | Description |
|------|-------------|
| `--timeout` | Operation timeout (default: 10m) |

### `crook up <node>`

Restore a node after maintenance by scaling up Rook-Ceph workloads.

**What it does:**
1. Discovers scaled-down deployments for the node via nodeSelector
2. Uncordons the node (marks it schedulable)
3. Restores Rook-Ceph deployments to 1 replica
4. Scales up the rook-ceph-operator
5. Unsets the Ceph `noout` flag

**Flags:**
| Flag | Description |
|------|-------------|
| `--timeout` | Operation timeout (default: 15m) |

### `crook ls [node]`

List Rook-Ceph resources in an interactive TUI or formatted output.

**Flags:**
| Flag | Description |
|------|-------------|
| `-o, --output` | Output format: tui, table, json, yaml (default: tui) |
| `--show` | Resource types to display: nodes,deployments,osds,pods |

## Configuration

Configuration is loaded from multiple sources (highest to lowest precedence):

1. CLI flags
2. Environment variables (`CROOK_*` prefix)
3. Config file
4. Built-in defaults

### Config File Locations

crook searches for configuration in:
- `./crook.yaml` (current directory)
- `~/.config/crook/config.yaml` (user config)
- `/etc/crook/config.yaml` (system config)

Or specify a custom location: `--config /path/to/config.yaml`

### Configuration Options

```yaml
# Kubernetes cluster configuration
kubernetes:
  rook-operator-namespace: rook-ceph
  rook-cluster-namespace: rook-ceph
  # context: my-cluster-context

# Terminal UI configuration
ui:
  theme: default
  progress-refresh-ms: 100
  ls-refresh-nodes-ms: 2000
  ls-refresh-deployments-ms: 2000
  ls-refresh-pods-ms: 2000
  ls-refresh-osds-ms: 5000
  ls-refresh-header-ms: 5000

# Operation timeouts
timeouts:
  api-call-timeout-seconds: 30
  wait-deployment-timeout-seconds: 300
  ceph-command-timeout-seconds: 60

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  # file: ~/.local/state/crook/crook.log
  format: text  # text, json
```

See `crook.yaml.example` for a fully documented example configuration.

### Environment Variables

All configuration options can be set via environment variables with the `CROOK_` prefix:

```bash
export CROOK_KUBERNETES_ROOK_OPERATOR_NAMESPACE=rook-ceph
export CROOK_LOGGING_LEVEL=debug
```

## Examples

### Maintenance Workflow

```bash
# 1. Check cluster status before maintenance
crook ls worker-1

# 2. Take the node down for maintenance
crook down worker-1

# 3. Perform maintenance (reboot, hardware changes, etc.)
# ... node is now safe to work on ...

# 4. Restore the node after maintenance
crook up worker-1

# 5. Verify cluster status
crook ls worker-1
```

#### Down Phase Progress

The down phase safely prepares a node for maintenance with confirmation dialogs and real-time progress tracking:

<div align="center">
  <table>
    <tr>
      <td align="center">
        <strong>Confirmation Prompt</strong><br>
        <img src="docs/images/down-confirmation.png" alt="Down confirmation prompt" width="400">
      </td>
      <td align="center">
        <strong>Operation Progress</strong><br>
        <img src="docs/images/down-progress.png" alt="Down phase progress" width="400">
      </td>
    </tr>
  </table>
</div>

### JSON Output for Automation

```bash
# Get cluster data as JSON
crook ls --output json | jq '.nodes[] | select(.schedulable == false)'
```

## Troubleshooting

### Common Issues

**"failed to create kubernetes client"**
- Verify kubeconfig path and cluster connectivity: `kubectl cluster-info`
- Ensure proper RBAC permissions

**"node not found in cluster"**
- Verify node name: `kubectl get nodes`
- Check spelling and case sensitivity

**"rook-ceph-tools pod not found"**
- Deploy rook-ceph-tools: `kubectl -n rook-ceph get deploy rook-ceph-tools`
- Check namespace configuration

**"Ceph health not OK"**
- Check Ceph status: `kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph status`
- Resolve Ceph health issues before maintenance

### Debug Logging

Enable debug logging for detailed output:

```bash
crook down worker-1 --log-level debug

# Or set in environment
export CROOK_LOGGING_LEVEL=debug
```

## Architecture

```
crook/
├── cmd/crook/           # CLI entry point and commands
│   └── commands/        # Cobra command implementations
├── pkg/
│   ├── config/          # Configuration management
│   ├── k8s/             # Kubernetes client operations
│   ├── maintenance/     # Down/up phase business logic
│   ├── monitoring/      # Resource monitoring
│   └── tui/             # Bubble Tea UI components
│       ├── components/  # Reusable UI components
│       ├── models/      # Bubble Tea models
│       ├── views/       # View renderers
│       └── styles/      # Theme and styling
└── internal/
    └── logger/          # Structured logging
```

## Development

### Prerequisites

- Go 1.25+
- [just](https://github.com/casey/just) (optional, for task automation)
- [golangci-lint](https://golangci-lint.run/) (for linting)

### Common Tasks

```bash
# Build
just build

# Run tests
just test

# Run linter
just lint

# Full verification (lint + test + build)
just verify

# Run with arguments
just run ls --output table
```

## License

MIT License - see [LICENSE](LICENSE) for details.
