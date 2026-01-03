# Project Context

## Purpose
**crook** is a Kubernetes node maintenance automation tool for Rook-Ceph clusters. It safely manages the process of taking nodes down for maintenance and bringing them back up while preserving Ceph cluster health and state.

**Key Goals:**
- Automate safe node maintenance procedures for Rook-Ceph deployments
- Prevent data loss during node maintenance by managing Ceph OSDs, monitors, and related services
- Maintain cluster state across node reboots and maintenance windows
- Provide an interactive TUI with real-time feedback for operators
- Enable safe, guided workflows with pre-flight validation and health monitoring

## Tech Stack
- **Go 1.24+** - Primary implementation language
- **Bubble Tea** - Terminal user interface framework (`github.com/charmbracelet/bubbletea`)
- **Kubernetes client-go** - Official Kubernetes Go client library (`k8s.io/client-go`)
- **Cobra** - CLI framework (`github.com/spf13/cobra`)
- **Viper** - Configuration management (`github.com/spf13/viper`)
- **Nix/devenv** - Reproducible development environment management
- **git** - Version control
- **just** - Command runner (task automation)

## Project Conventions

### Code Style
- **Go Standards:**
  - Follow official Go style guide and `gofmt` formatting
  - Use `golangci-lint` for comprehensive linting
  - Effective Go principles: clear, idiomatic code over cleverness
  - Package names: lowercase, single-word, descriptive (e.g., `maintenance`, `monitoring`)
  - Exported identifiers: PascalCase; unexported: camelCase
  - Error messages: lowercase, no punctuation at end
  - Prefer early returns over deeply nested conditionals

- **Error Handling:**
  - Always check errors; use `if err != nil` pattern
  - Wrap errors with context using `fmt.Errorf("operation failed: %w", err)`
  - Return errors up the stack; log at top level only
  - Use custom error types for domain-specific errors
  - Provide actionable error messages for operators

- **Function Design:**
  - Small, focused functions with single responsibility
  - Pure functions where possible (no side effects)
  - Clear function signatures with meaningful parameter names
  - Use context.Context for cancellation and timeouts
  - Return `(result, error)` pattern; avoid panics except for programmer errors

- **Naming Conventions:**
  - Interfaces: end with `-er` (e.g., `ClientFactory`, `NodeManager`)
  - Constructors: `New<Type>()` or `New<Type>WithOptions()`
  - Boolean functions: start with `Is`, `Has`, `Can`, `Should`
  - Getters: no `Get` prefix (e.g., `Name()` not `GetName()`)
  - Avoid stuttering: `k8s.Client` not `k8s.K8sClient`

### Architecture Patterns
- **MVC Pattern (Bubble Tea):** Model-Update-View separation for TUI components
- **Package Organization:**
  - `cmd/crook/` - Main entry point and CLI commands
  - `pkg/` - Public packages (reusable by external tools)
  - `internal/` - Private packages (project-specific utilities)
- **State Management:**
  - TSV-based state files for deployment replica tracking (backward compatible)
  - In-memory state machines for TUI workflow progression
- **Dependency Injection:** Pass dependencies explicitly (no global state)
- **Interface Segregation:** Small, focused interfaces for testability
- **Separation of Concerns:**
  - Kubernetes operations in `pkg/k8s/`
  - Business logic in `pkg/maintenance/`
  - UI in `pkg/tui/`
  - Configuration in `pkg/config/`
- **Idempotent Operations:** Operations safe to retry without side effects
- **Context-Based Cancellation:** Propagate context.Context for graceful shutdown
- **External State Validation:** Poll Kubernetes API until desired state reached

### Testing Strategy

**Unit Tests:**
- Use Go's built-in `testing` package
- Table-driven tests for multiple scenarios
- Mock external dependencies (Kubernetes API, file I/O)
- Target 80%+ code coverage for business logic
- Test files: `*_test.go` in same package as code

**Integration Tests:**
- Test against real Kubernetes API (or kind cluster)
- Validate end-to-end workflows (down/up phases)
- Test state file backward compatibility
- Use build tags: `// +build integration`

**TUI Tests:**
- Test Bubble Tea models in isolation
- Verify state transitions
- Test message handling and updates
- Mock background operations

**Test Organization:**
- Unit tests: co-located with code (`pkg/*/`)
- Integration tests: `test/integration/`
- Test utilities: `test/util/`
- Fixtures: `test/fixtures/`

**Test Commands:**
```bash
go test ./...                    # Unit tests
go test -tags=integration ./...  # Integration tests
go test -race ./...              # Race condition detection
go test -bench=. ./...           # Benchmarks
```

### Git Workflow
- **Issue Tracking:** Beads (AI-native, git-based issue tracker in `.beads/`)
- **Change Management:** OpenSpec for architectural changes and proposals
- **Session Completion:** Mandatory workflow (see AGENTS.md):
  1. File issues for remaining work
  2. Run quality gates:
     - `go test ./...` - All tests pass
     - `golangci-lint run` - No linter errors
     - `go build ./cmd/crook` - Clean build
  3. Update issue status
  4. **MANDATORY:** `git pull --rebase && bd sync && git push`
  5. Clean up stashes and branches
  6. Verify all changes pushed
  7. Provide handoff context
- **Branching:** Standard feature branch workflow
- **Commits:**
  - Conventional commits format: `type(scope): description`
  - Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`
  - Must push to remote; work is not complete until `git push` succeeds
- **Code Review:**
  - All PRs require passing CI (tests, lints, build)
  - No direct commits to main branch

## Domain Context

### Kubernetes & Rook-Ceph Architecture
- **Rook-Ceph:** Cloud-native storage orchestrator for Ceph on Kubernetes
- **OSDs (Object Storage Daemons):** Core Ceph storage components, typically node-pinned
- **Monitors (MON):** Maintain cluster state and consensus
- **Operators:** Kubernetes operators that manage Ceph lifecycle
- **Node Cordoning:** Marks nodes as unschedulable to prevent new pod scheduling
- **Deployment Scaling:** Used to gracefully stop and start Ceph components

### Maintenance Workflow
1. **Down Phase:** Cordon node → Set noout flag → Scale operator to 0 → Scale node deployments to 0 → Save state
2. **Maintenance:** Node can be safely rebooted/maintained
3. **Up Phase:** Restore deployment replicas → Scale operator to 1 → Unset noout flag → Uncordon node

### Critical Ceph Concepts
- **noout flag:** Prevents Ceph from rebalancing data when OSDs go offline temporarily
- **ReplicaSet/Deployment ownership:** Pods → ReplicaSets → Deployments (tracked for state preservation)
- **rook-ceph-tools:** Deployment providing `ceph` CLI access within the cluster

## Important Constraints

### Technical Constraints
- **Ceph Cluster Health:** Must not trigger data rebalancing during maintenance
- **Operator Reconciliation:** Rook operator must be scaled down to prevent it from restarting scaled-down components
- **State Preservation:** Must accurately track and restore deployment replica counts
- **Kubernetes API Availability:** Requires functioning kubectl access throughout operation
- **Node-Pinned Workloads:** Only manages deployments with pods actually running on the target node

### Operational Constraints
- **Single Node Maintenance:** Designed for one node at a time (parallel maintenance not supported initially)
- **State File Dependency:** "up" operation requires state file from corresponding "down" operation
- **Kubernetes API Dependency:** Requires valid kubeconfig and cluster access
- **Terminal Requirements:** TUI requires terminal with minimum 80 columns width
- **rook-ceph-tools Required:** Needs rook-ceph-tools deployment for Ceph CLI access

### Safety Constraints
- **No Data Loss:** Must prevent Ceph data rebalancing/recovery during maintenance
- **Graceful Shutdown:** Must wait for deployments to fully scale down before proceeding
- **Atomic Operations:** State changes must be tracked for complete restoration

## External Dependencies

### Required Services
- **Kubernetes Cluster:** Target infrastructure (tested with production Kubernetes environments)
- **Rook-Ceph Operator:** Must be deployed in cluster (typically in `rook-ceph` namespace)
- **Rook-Ceph Cluster:** Active Ceph cluster managed by Rook
- **rook-ceph-tools Deployment:** Provides `ceph` CLI access for flag management

### Runtime Dependencies
- **Go binary:** Single compiled binary (no external runtime dependencies)
- **Kubeconfig:** Valid Kubernetes configuration for cluster access
- **Terminal:** For TUI mode (optional for automation/scripting)

### Configuration Dependencies
- **Configuration Sources (in precedence order):**
  1. CLI flags (highest priority)
  2. Environment variables (`CROOK_*` prefix)
  3. Config file (`crook.yaml`, `~/.config/crook/config.yaml`, or `/etc/crook/config.yaml`)
  4. Defaults (lowest priority)

- **Key Configuration Options:**
  - `kubernetes.rook-operator-namespace` (default: `rook-ceph`)
  - `kubernetes.rook-cluster-namespace` (default: `rook-ceph`)
  - `kubernetes.kubeconfig` (default: standard kubeconfig locations)
  - `state.file-path-template` (default: `./crook-state-{{.Node}}.tsv`)
  - `deployment-filters.prefixes` (default: `[rook-ceph-osd, rook-ceph-mon, ...]`)

- **RBAC Permissions Required:**
  - Nodes: get, list, patch (cordon/uncordon)
  - Deployments: get, list, patch (scale)
  - Pods: get, list, exec (Ceph commands via rook-ceph-tools)
  - ReplicaSets: get, list (ownership chain traversal)
