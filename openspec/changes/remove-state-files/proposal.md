# Change: Remove State Files - Use nodeSelector for Node Discovery

## Why

The current implementation uses external JSON state files to track which deployments were scaled down during node maintenance. This approach has several problems:
- State files can be lost or corrupted
- Multi-node maintenance is error-prone (which state file for which node?)
- External state adds complexity and failure modes

Kubernetes already contains all information needed via deployment `nodeSelector` fields, which persist even when deployments are scaled to 0 replicas.

## What Changes

- **BREAKING**: Remove `--state-file` CLI flag from `up` and `down` commands
- **BREAKING**: Remove `state` configuration section from config files
- **BREAKING**: Remove `deployment-filters` configuration section from config files
- Remove entire `pkg/state/` package (~500 lines)
- Add nodeSelector-based deployment discovery (`GetDeploymentTargetNode`, `ListNodePinnedDeployments`)
- Simplify down phase: discover node-pinned deployments via nodeSelector, scale to 0
- Simplify up phase: discover scaled-down deployments for node via nodeSelector, scale to 1
- Add warning when deployments have >1 replica (unexpected for Rook-Ceph)

## Impact

- Affected specs: node-maintenance (new capability spec)
- Affected code:
  - `pkg/state/*` - DELETE entirely
  - `pkg/k8s/deployments.go` - Add node extraction functions
  - `pkg/maintenance/down.go` - Use nodeSelector discovery
  - `pkg/maintenance/up.go` - Use nodeSelector discovery
  - `pkg/maintenance/discovery.go` - Simplify or remove
  - `pkg/config/config.go` - Remove StateConfig, DeploymentFilterConfig
  - `cmd/crook/commands/up.go` - Remove state file flags
  - `cmd/crook/commands/down.go` - Remove state file flags
