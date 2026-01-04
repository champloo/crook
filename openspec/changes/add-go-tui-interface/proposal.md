# Change: Add Go-based TUI Interface for Rook-Ceph Node Maintenance

## Why

The current bash script (`osd-maintenance.sh`) works but has limitations:
- No visibility into operation progress (silent waiting periods)
- No pre-flight cluster health validation
- Error-prone manual workflow (users must remember exact commands)
- Difficult to extend with new features (monitoring, validation, rollback)

A Go-based TUI application will provide:
- Interactive, guided workflows with real-time feedback
- Visual cluster health monitoring before/during operations
- Safer operations with built-in validation and confirmation prompts
- Better error handling and recovery options
- Foundation for future features (scheduling, automation, multi-node support)

## What Changes

- **Create Go application** using Bubble Tea TUI framework
- **Replace bash script** with compiled Go binary (bash script will be removed)
- **Add configuration system** with YAML/TOML config files and CLI flag overrides
- **Implement live progress tracking** with progress bars for all async operations
- **Add cluster health dashboard** showing node status, OSD health, deployment states
- **Add resource listing command** (`crook ls`) for interactive cluster inspection with tabbed views
- **Kubernetes client integration** using official client-go library
- **State persistence** with JSON state file format (`resources` array, extensible)
- **Interactive workflows** for both down and up phases with confirmations
- **Pre-flight validation** checks before allowing destructive operations

Core functionality matches existing bash script:
- Down phase: cordon → noout → scale operator → scale deployments → save state
- Up phase: uncordon → restore deployments → scale operator → unset noout  
New functionality beyond bash script:
- `crook ls` command for viewing Ceph cluster resources (nodes, deployments, OSDs, pods)
- Tabbed navigation between resource types with filtering and search
- Real-time cluster health summary header
- Multiple output formats (TUI, table, JSON, YAML) for scripting integration

## Impact

- **Affected specs**: All new - this is the first implementation
  - `node-maintenance` - Core down/up operations
  - `tui-interface` - UI components and interactions
  - `kubernetes-client` - K8s API interactions
  - `cluster-monitoring` - Health dashboard
  - `configuration` - Config and CLI management
  - `state-persistence` - State file handling

- **Affected code**:
  - Remove: `osd-maintenance.sh` (replaced by Go binary)
  - Add: Complete Go codebase in `cmd/`, `pkg/`, `internal/`
  - Add: `go.mod`, `go.sum`
  - Add: Configuration files (`crook.yaml` example)
  - Update: `devenv.nix` (already has Go configured)

- **Migration impact**:
  - State files are written as JSON (`.json`) with no TSV compatibility
  - Users must compile Go binary or use pre-built releases
  - Breaking change: bash script removed (full replacement)

- **Dependencies**:
  - Bubble Tea (`github.com/charmbracelet/bubbletea`)
  - Kubernetes client-go (`k8s.io/client-go`)
  - Viper for configuration (`github.com/spf13/viper`)
  - Cobra for CLI (`github.com/spf13/cobra`)
