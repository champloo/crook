# Design: Go-based TUI Interface

## Context

Current implementation is a ~130-line bash script that:
- Provides CLI-only interface with no feedback during long operations
- Has no validation of cluster state before operations
- Requires manual state file management
- Is difficult to test and extend

We're building a Go application with TUI to provide:
- Interactive, visual interface for safer operations
- Real-time progress feedback
- Cluster health monitoring
- Better error handling and recovery

### Constraints

- Must work in environments without GUI (terminal-only)
- Must be deployable as single binary (no runtime dependencies)
- Must support same namespaces/configuration as bash script

### Stakeholders

- Kubernetes cluster operators performing node maintenance
- DevOps/SRE teams managing Rook-Ceph clusters
- Automation systems that may wrap crook (future consideration)

## Goals / Non-Goals

### Goals

- Replace bash script with feature-equivalent Go application
- Add TUI with real-time progress tracking
- Add cluster health dashboard view
- Add resource listing command (`crook ls`) for cluster inspection
- Implement proper error handling and validation
- Support configuration via file + CLI flags
- Maintain a stable, versioned state file format

### Non-Goals

- Multi-node parallel operations (future enhancement)
- Operation history/logging (future enhancement)
- Scheduled maintenance automation (future enhancement)
- REST API or daemon mode (future enhancement)
- GUI application (TUI only)

## Decisions

### Architecture: Bubble Tea MVC Pattern

**Decision**: Use Bubble Tea's Model-Update-View pattern with capability-based models

**Structure**:
```
cmd/
  crook/
    main.go           # Entry point, cobra CLI setup
pkg/
  maintenance/        # Core maintenance operations
    down.go           # Down phase orchestration
    up.go             # Up phase orchestration
    validator.go      # Pre-flight checks
  k8s/               # Kubernetes client wrapper
    client.go         # K8s client initialization
    deployments.go    # Deployment operations
    nodes.go          # Node operations (cordon/uncordon)
    ceph.go           # Ceph command execution
  state/             # State persistence
    state.go          # State file read/write
    format.go         # JSON format handling
  tui/               # Bubble Tea components
    models/
      app.go          # Main app model
      dashboard.go    # Health dashboard model
      down.go         # Down phase model
      up.go           # Up phase model
      ls.go           # Resource list model (ls command)
    components/
      progress.go     # Progress bar component
      status.go       # Status display component
      confirm.go      # Confirmation prompt
      table.go        # Resource table component
      tabs.go         # Tab navigation component
    styles/
      theme.go        # Color scheme and styles
  config/            # Configuration management
    config.go         # Viper config loading
internal/            # Internal utilities
  logger/
    logger.go         # Structured logging
```

**Alternatives considered**:
- **tview**: More widgets out-of-box but less flexible for custom layouts
- **termui**: Better for dashboards but worse for interactive workflows
- **Pure CLI (no TUI)**: Simpler but loses visual feedback benefits

**Rationale**: Bubble Tea provides best balance of flexibility, maintainability, and user experience for interactive workflows with real-time updates.

### Kubernetes Client: Official client-go

**Decision**: Use `k8s.io/client-go` directly with custom wrappers

**Pattern**:
- Single shared clientset initialized from kubeconfig
- Wrapper functions for specific operations (CordonNode, ScaleDeployment)
- Context-based cancellation support for all operations
- Retry logic with exponential backoff for transient errors

**Alternatives considered**:
- **kubectl exec**: Shell out to kubectl (loses type safety, harder to test)
- **Custom REST client**: Reinventing the wheel, maintenance burden

**Rationale**: client-go is the standard, well-tested, and provides full API access with Go types.

### Configuration: Viper + Cobra Pattern

**Decision**: Layered configuration with precedence:
1. CLI flags (highest priority)
2. Environment variables (`CROOK_*`)
3. Config file (`~/.config/crook/config.yaml` or `./crook.yaml`)
4. Defaults (lowest priority)

**Config schema**:
```yaml
# crook.yaml
kubernetes:
  rook-operator-namespace: rook-ceph
  rook-cluster-namespace: rook-ceph
  kubeconfig: ~/.kube/config  # Optional, defaults to standard locations

state:
  file-path-template: "./crook-state-{{.Node}}.json"
  backup-enabled: true

deployment-filters:
  prefixes:
    - rook-ceph-osd
    - rook-ceph-mon
    - rook-ceph-exporter
    - rook-ceph-crashcollector

ui:
  theme: default  # Future: support custom themes
  progress-refresh-ms: 100
```

**Rationale**: Standard Go tooling pattern, familiar to operators, supports automation.

### State Persistence: JSON Format

**Decision**: Use JSON format for state files, with a `resources` array for future extensibility.

**Format**:
```json
{
  "version": "v1",
  "node": "worker-01",
  "timestamp": "2026-01-03T10:30:00Z",
  "operatorReplicas": 1,
  "resources": [
    {
      "kind": "Deployment",
      "namespace": "rook-ceph",
      "name": "rook-ceph-osd-0",
      "replicas": 1
    },
    {
      "kind": "Deployment",
      "namespace": "rook-ceph",
      "name": "rook-ceph-mon-a",
      "replicas": 1
    }
  ]
}
```

**Notes**:
- Use `version` for schema evolution.
- Keep the top-level shape stable; add new optional fields as needed.

**Alternatives considered**:
- **TSV**: Simpler, but requires custom parsing/validation and is harder to evolve
- **YAML**: Human-friendly, but less strict and more error-prone for automation
- **SQLite**: Too heavy for simple state persistence

**Rationale**: JSON is structured, has native Go support, and supports schema evolution cleanly.

### TUI Flow: State Machine Pattern

**Decision**: Model each phase (down/up) as a state machine

**Down Phase States**:
1. `Init` - Show cluster health dashboard
2. `Confirm` - Confirm node and show impact
3. `Cordoning` - Cordon node with progress
4. `SettingNoOut` - Set Ceph noout flag
5. `ScalingOperator` - Scale operator to 0
6. `DiscoveringDeployments` - Find target deployments
7. `ScalingDeployments` - Scale each deployment (sub-state per deployment)
8. `Complete` - Show summary, state file location

**Up Phase States**:
1. `Init` - Load state file, validate
2. `Confirm` - Show restore plan
3. `Uncordoning` - Uncordon node
4. `RestoringDeployments` - Scale deployments back up
5. `ScalingOperator` - Restore operator
6. `UnsettingNoOut` - Unset Ceph flag
7. `Complete` - Show summary

**Progress Tracking**:
- Each state with async operations shows progress bar
- Updates via Bubble Tea messages from goroutines
- Errors transition to error state with retry/abort options

**Rationale**: Clear state transitions, easy to test, visual progress mapping.

## Risks / Trade-offs

### Risk: Binary compilation requirement

**Mitigation**:
- Provide pre-built binaries for common platforms (Linux amd64/arm64)
- Document Go installation in README
- Keep devenv.nix for easy development setup

### Risk: Increased complexity vs bash script

**Trade-off**: More code to maintain but better UX, testing, and extensibility
**Mitigation**:
- Comprehensive testing (unit + integration)
- Good documentation
- Keep core logic simple and isolated

### Risk: Terminal compatibility issues

**Mitigation**:
- Bubble Tea handles most terminal quirks
- Test on common terminals (xterm, tmux, screen)
- Provide `--no-tui` fallback mode for automation (future)

### Risk: State file format changes breaking compatibility

**Mitigation**:
- Version the state file format (`version` field)
- Validate state file version before parsing

## Migration Plan

### Phase 1: Foundation (Week 1)
1. Initialize Go module
2. Set up project structure
3. Implement Kubernetes client wrapper
4. Implement state persistence (JSON read/write)
5. Write unit tests for core components

### Phase 2: Core Logic (Week 2)
1. Implement down phase operations (without TUI)
2. Implement up phase operations (without TUI)
3. Add validation and error handling
4. Write integration tests

### Phase 3: TUI (Week 3)
1. Build Bubble Tea models for each phase
2. Implement progress tracking components
3. Add cluster health dashboard
4. Implement confirmation prompts

### Phase 4: Polish (Week 4)
1. Add configuration loading (Viper)
2. Build CLI with Cobra
3. Add logging
4. Documentation and examples
5. Remove bash script

### Rollback

If Go implementation has critical issues:
- Revert bash script removal (git revert)
- Document known issues
- Fix forward in next iteration

## Open Questions

1. **Error recovery**: Should failed down phase automatically rollback changes?
   - Proposal: No auto-rollback, but provide clear manual recovery steps

2. **Logging**: Where should logs go in TUI mode?
   - Proposal: `~/.local/state/crook/crook.log` + optional `--log-file` flag

3. **Testing in production**: How to validate without impacting clusters?
   - Proposal: Add `--dry-run` mode that simulates operations without changes

4. **Concurrent deployments**: Scale deployments in parallel during down phase?
   - Proposal: Sequential for now (simpler), parallel in future enhancement
