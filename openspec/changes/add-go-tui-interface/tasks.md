# Implementation Tasks: Go TUI Interface

## 1. Project Setup

- [ ] 1.1 Initialize Go module with `go mod init github.com/<user>/crook`
- [ ] 1.2 Create directory structure (cmd/, pkg/, internal/)
- [ ] 1.3 Add initial dependencies to go.mod:
  - [ ] Bubble Tea (github.com/charmbracelet/bubbletea)
  - [ ] Kubernetes client-go (k8s.io/client-go)
  - [ ] Cobra (github.com/spf13/cobra)
  - [ ] Viper (github.com/spf13/viper)
- [ ] 1.4 Create .gitignore for Go (binaries, vendor, IDE files)
- [ ] 1.5 Set up basic logger package in internal/logger/
- [ ] 1.6 Create example config file (crook.yaml.example)

## 2. Kubernetes Client Wrapper

- [ ] 2.1 Implement client initialization in pkg/k8s/client.go:
  - [ ] Load kubeconfig from standard locations
  - [ ] Support in-cluster configuration
  - [ ] Validate connectivity to cluster
- [ ] 2.2 Implement node operations in pkg/k8s/nodes.go:
  - [ ] CordonNode()
  - [ ] UncordonNode()
  - [ ] GetNodeStatus()
- [ ] 2.3 Implement deployment operations in pkg/k8s/deployments.go:
  - [ ] ScaleDeployment()
  - [ ] GetDeploymentStatus()
  - [ ] ListDeploymentsInNamespace()
  - [ ] WaitForReadyReplicas()
  - [ ] WaitForReplicas()
- [ ] 2.4 Implement pod operations in pkg/k8s/pods.go:
  - [ ] ListPodsOnNode()
  - [ ] GetOwnerChain()
  - [ ] ExecInPod()
- [ ] 2.5 Implement Ceph operations in pkg/k8s/ceph.go:
  - [ ] ExecuteCephCommand()
  - [ ] SetNoOut()
  - [ ] UnsetNoOut()
  - [ ] GetCephStatus()
  - [ ] GetOSDTree()
- [ ] 2.6 Add retry logic with exponential backoff for transient errors
- [ ] 2.7 Add context-based cancellation support
- [ ] 2.8 Write unit tests for all k8s client functions

## 3. State Persistence

- [ ] 3.1 Define state data structure in pkg/state/state.go
- [ ] 3.2 Implement JSON format writer in pkg/state/format.go:
  - [ ] Write root object fields: `version`, `node`, `timestamp`, `operatorReplicas`, `resources`
  - [ ] Pretty-print with 2-space indentation (deterministic output)
  - [ ] Sort resources deterministically (namespace, then name)
  - [ ] Atomic file write (temp file + rename)
- [ ] 3.3 Implement JSON format parser:
  - [ ] Parse file as JSON
  - [ ] Validate required fields and data types (`version`, `resources`, resource fields)
  - [ ] Return structured errors for malformed JSON / missing fields
- [ ] 3.4 Implement state file path resolution with template substitution
- [ ] 3.5 Implement state file backup functionality
- [ ] 3.6 Add state file validation functions
- [ ] 3.7 Write unit tests for state persistence

## 4. Configuration Management

- [ ] 4.1 Define configuration schema in pkg/config/config.go
- [ ] 4.2 Implement Viper-based config loading:
  - [ ] Load from default locations
  - [ ] Support explicit --config flag
  - [ ] Environment variable overrides (CROOK_*)
  - [ ] CLI flag overrides
- [ ] 4.3 Implement configuration validation
- [ ] 4.4 Add default configuration values
- [ ] 4.5 Implement `crook config show` command
- [ ] 4.6 Implement `crook config validate` command
- [ ] 4.7 Write unit tests for configuration loading and merging
- [ ] 4.8 Create example config file with documentation comments

## 5. Core Maintenance Logic (CLI only, no TUI yet)

- [ ] 5.1 Implement down phase orchestration in pkg/maintenance/down.go:
  - [ ] Pre-flight validation
  - [ ] Cordon node
  - [ ] Set Ceph noout flag
  - [ ] Scale down operator
  - [ ] Discover target deployments
  - [ ] Scale down deployments with waiting
  - [ ] Save state file
- [ ] 5.2 Implement up phase orchestration in pkg/maintenance/up.go:
  - [ ] Load and validate state file
  - [ ] Uncordon node
  - [ ] Restore deployment replicas with waiting
  - [ ] Scale up operator
  - [ ] Unset Ceph noout flag
- [ ] 5.3 Implement deployment discovery in pkg/maintenance/discovery.go:
  - [ ] Find pods on target node
  - [ ] Trace ownership chain
  - [ ] Filter by configured prefixes
  - [ ] Return unique deployment list
- [ ] 5.4 Implement pre-flight validation in pkg/maintenance/validator.go:
  - [ ] Check cluster connectivity
  - [ ] Check node exists
  - [ ] Check namespaces exist
  - [ ] Check rook-ceph-tools exists
  - [ ] Check RBAC permissions (best-effort)
- [ ] 5.5 Add comprehensive error handling
- [ ] 5.6 Write unit tests for maintenance orchestration
- [ ] 5.7 Write integration tests (requires test cluster or mocks)

## 6. Cluster Monitoring

- [ ] 6.1 Implement node status monitoring in pkg/monitoring/node.go:
  - [ ] Query node status
  - [ ] Extract ready condition
  - [ ] Extract scheduling status and taints
- [ ] 6.2 Implement Ceph health monitoring in pkg/monitoring/ceph.go:
  - [ ] Execute `ceph status --format json`
  - [ ] Parse health status
  - [ ] Extract OSD, monitor, PG information
- [ ] 6.3 Implement OSD monitoring:
  - [ ] Execute `ceph osd tree --format json`
  - [ ] Filter OSDs by node
  - [ ] Extract status (up/down, in/out)
- [ ] 6.4 Implement deployment monitoring in pkg/monitoring/deployments.go:
  - [ ] Monitor operator deployment
  - [ ] Monitor discovered deployments
  - [ ] Calculate aggregate status
- [ ] 6.5 Implement background refresh with goroutines
- [ ] 6.6 Add configurable refresh intervals
- [ ] 6.7 Write unit tests for monitoring functions

## 7. TUI Components (Bubble Tea)

- [ ] 7.1 Create theme and styles in pkg/tui/styles/theme.go:
  - [ ] Define color palette
  - [ ] Define text styles (heading, status, error, warning)
  - [ ] Define border styles
- [ ] 7.2 Create reusable components in pkg/tui/components/:
  - [ ] 7.2.1 Progress bar component (progress.go)
  - [ ] 7.2.2 Status indicator component (status.go)
  - [ ] 7.2.3 Confirmation prompt component (confirm.go)
  - [ ] 7.2.4 Table display component (table.go)
- [ ] 7.3 Create main app model in pkg/tui/models/app.go:
  - [ ] Define app state
  - [ ] Implement Init() method
  - [ ] Implement Update() method for global messages
  - [ ] Implement View() method for routing to sub-models
- [ ] 7.4 Create dashboard model in pkg/tui/models/dashboard.go:
  - [ ] Display cluster health summary
  - [ ] Display node status
  - [ ] Display Ceph health
  - [ ] Display OSD status
  - [ ] Handle background refresh updates
  - [ ] Handle user confirmation to proceed
- [ ] 7.5 Create down phase model in pkg/tui/models/down.go:
  - [ ] Define state machine states
  - [ ] Implement state transitions
  - [ ] Display current state and progress
  - [ ] Handle async operation updates
  - [ ] Handle errors with retry/abort options
- [ ] 7.6 Create up phase model in pkg/tui/models/up.go:
  - [ ] Define state machine states
  - [ ] Display restore plan
  - [ ] Implement state transitions
  - [ ] Display progress for each operation
- [ ] 7.7 Implement keyboard navigation and shortcuts
- [ ] 7.8 Add terminal size detection and responsive rendering
- [ ] 7.9 Add color support detection and fallback

## 8. CLI Commands (Cobra)

- [ ] 8.1 Create root command in cmd/crook/main.go:
  - [ ] Initialize Cobra app
  - [ ] Add global flags
  - [ ] Set up logging
  - [ ] Load configuration
- [ ] 8.2 Create `crook down` command:
  - [ ] Accept node name argument
  - [ ] Accept command-specific flags
  - [ ] Initialize TUI or run headless based on flag
  - [ ] Execute down phase
- [ ] 8.3 Create `crook up` command:
  - [ ] Accept node name argument
  - [ ] Accept command-specific flags
  - [ ] Initialize TUI or run headless based on flag
  - [ ] Execute up phase
- [ ] 8.4 Create `crook config` command group:
  - [ ] `crook config show` - Display effective config
  - [ ] `crook config validate` - Validate config file
- [ ] 8.5 Create `crook state` command group:
  - [ ] `crook state list` - List state files
  - [ ] `crook state clean` - Clean old backups
  - [ ] `crook state show <file>` - Display state file contents
- [ ] 8.6 Create `crook version` command
- [ ] 8.7 Add shell completion generation (bash, zsh, fish)

## 9. Integration and Testing

- [ ] 9.1 Write integration tests for complete workflows:
  - [ ] Test down phase end-to-end
  - [ ] Test up phase end-to-end
  - [ ] Test error recovery scenarios
- [ ] 9.2 Test with real Kubernetes cluster (if available) or kind
- [ ] 9.3 Test TUI in various terminals (xterm, tmux, screen)
- [ ] 9.4 Test configuration loading from all sources
- [ ] 9.5 Test state file backward compatibility with bash script files
- [ ] 9.6 Add benchmark tests for performance-critical operations
- [ ] 9.7 Test cancellation handling (Ctrl+C during operations)

## 10. Documentation and Polish

- [ ] 10.1 Write comprehensive README.md:
  - [ ] Installation instructions
  - [ ] Quick start guide
  - [ ] Configuration documentation
  - [ ] Command reference
  - [ ] Troubleshooting guide
- [ ] 10.2 Create EXAMPLES.md with common workflows
- [ ] 10.3 Add godoc comments to all exported functions
- [ ] 10.4 Create CONTRIBUTING.md with development setup
- [ ] 10.5 Add LICENSE file
- [ ] 10.6 Create justfile with common tasks:
  - [ ] build, test, lint, install targets
- [ ] 10.7 Set up CI/CD pipeline (GitHub Actions):
  - [ ] Run tests on PR
  - [ ] Build binaries for releases
  - [ ] Run linters (golangci-lint)
- [ ] 10.8 Create release process and versioning strategy

## 11. Migration and Cleanup

- [ ] 11.1 Test Go binary with existing bash script state files
- [ ] 11.2 Verify feature parity with bash script
- [ ] 11.3 Create migration guide for users
- [ ] 11.4 Remove osd-maintenance.sh
- [ ] 11.5 Update devenv.nix if needed (already has Go configured)
- [ ] 11.6 Create pre-built binaries for common platforms:
  - [ ] Linux amd64
  - [ ] Linux arm64
  - [ ] macOS amd64
  - [ ] macOS arm64
- [ ] 11.7 Update project.md to reflect Go implementation

## 12. Final Validation

- [ ] 12.1 Run `openspec validate add-go-tui-interface --strict`
- [ ] 12.2 Verify all requirements have corresponding code
- [ ] 12.3 Verify all scenarios are tested
- [ ] 12.4 Perform manual end-to-end testing
- [ ] 12.5 Collect feedback from early users (if applicable)
- [ ] 12.6 Address any issues found during validation
- [ ] 12.7 Tag v1.0.0 release

## Notes

- Tasks can be parallelized where indicated (e.g., 2.x and 3.x can be worked on concurrently)
- Integration tests (9.x) require tasks 2-8 to be complete
- TUI work (7.x) can start after monitoring (6.x) is partially complete
- Each checkbox represents a verifiable unit of work that can be demonstrated or tested
