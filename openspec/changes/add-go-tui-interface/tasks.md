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

## 3. Configuration Management

- [ ] 3.1 Define configuration schema in pkg/config/config.go
- [ ] 3.2 Implement Viper-based config loading:
  - [ ] Load from default locations
  - [ ] Support explicit --config flag
  - [ ] Environment variable overrides (CROOK_*)
  - [ ] CLI flag overrides
- [ ] 3.3 Implement configuration validation
- [ ] 3.4 Add default configuration values
- [ ] 3.5 Write unit tests for configuration loading and merging
- [ ] 3.6 Create example config file with documentation comments

## 4. Core Maintenance Logic (CLI only, no TUI yet)

- [ ] 4.1 Implement down phase orchestration in pkg/maintenance/down.go:
  - [ ] Pre-flight validation
  - [ ] Cordon node
  - [ ] Set Ceph noout flag
  - [ ] Scale down operator
  - [ ] Discover target deployments via nodeSelector
  - [ ] Scale down deployments with waiting
- [ ] 4.2 Implement up phase orchestration in pkg/maintenance/up.go:
  - [ ] Discover scaled-down deployments via nodeSelector
  - [ ] Uncordon node
  - [ ] Restore deployment replicas with MON quorum gating
  - [ ] Scale up operator
  - [ ] Unset Ceph noout flag
- [ ] 4.3 Implement deployment discovery in pkg/maintenance/discovery.go:
  - [ ] Find deployments with nodeSelector matching target node
  - [ ] Support nodeAffinity fallback
  - [ ] Filter by replicas (for up phase)
- [ ] 4.4 Implement pre-flight validation in pkg/maintenance/validator.go:
  - [ ] Check cluster connectivity
  - [ ] Check node exists
  - [ ] Check namespaces exist
  - [ ] Check rook-ceph-tools exists
  - [ ] Check RBAC permissions (best-effort)
- [ ] 4.5 Add comprehensive error handling
- [ ] 4.6 Write unit tests for maintenance orchestration
- [ ] 4.7 Write integration tests (requires test cluster or mocks)

## 5. Cluster Monitoring

- [ ] 5.1 Implement node status monitoring in pkg/monitoring/node.go:
  - [ ] Query node status
  - [ ] Extract ready condition
  - [ ] Extract scheduling status and taints
- [ ] 5.2 Implement Ceph health monitoring in pkg/monitoring/ceph.go:
  - [ ] Execute `ceph status --format json`
  - [ ] Parse health status
  - [ ] Extract OSD, monitor, PG information
- [ ] 5.3 Implement OSD monitoring:
  - [ ] Execute `ceph osd tree --format json`
  - [ ] Filter OSDs by node
  - [ ] Extract status (up/down, in/out)
- [ ] 5.4 Implement deployment monitoring in pkg/monitoring/deployments.go:
  - [ ] Monitor operator deployment
  - [ ] Monitor discovered deployments
  - [ ] Calculate aggregate status
- [ ] 5.5 Implement background refresh with goroutines
- [ ] 5.6 Add configurable refresh intervals
- [ ] 5.7 Write unit tests for monitoring functions

## 6. TUI Components (Bubble Tea)

- [ ] 6.1 Create theme and styles in pkg/tui/styles/theme.go:
  - [ ] Define color palette
  - [ ] Define text styles (heading, status, error, warning)
  - [ ] Define border styles
- [ ] 6.2 Create reusable components in pkg/tui/components/:
  - [ ] 6.2.1 Progress bar component (progress.go)
  - [ ] 6.2.2 Status indicator component (status.go)
  - [ ] 6.2.3 Confirmation prompt component (confirm.go)
  - [ ] 6.2.4 Table display component (table.go)
- [ ] 6.3 Create main app model in pkg/tui/models/app.go:
  - [ ] Define app state
  - [ ] Implement Init() method
  - [ ] Implement Update() method for global messages
  - [ ] Implement View() method for routing to sub-models
- [ ] 6.5 Create down phase model in pkg/tui/models/down.go:
  - [ ] Define state machine states
  - [ ] Implement state transitions
  - [ ] Display current state and progress
  - [ ] Handle async operation updates
  - [ ] Handle errors with retry/abort options
- [ ] 6.6 Create up phase model in pkg/tui/models/up.go:
  - [ ] Define state machine states
  - [ ] Display restore plan
  - [ ] Implement state transitions
  - [ ] Display progress for each operation
- [ ] 6.7 Implement keyboard navigation and shortcuts
- [ ] 6.8 Add terminal size detection and responsive rendering
- [ ] 6.9 Add color support detection and fallback
- [ ] 6.10 `crook ls`: permanent Maintenance pane for up/down flows
  - [ ] 6.10.1 Make UpModel/DownModel support embedded rendering (no outer frame)
  - [ ] 6.10.2 Render Nodes + Maintenance panes side-by-side (top row)
  - [ ] 6.10.3 Route key input to embedded flow while active; preserve node selection on exit
  - [ ] 6.10.4 Make Nodes view responsive to reduced width (column sizing/degradation)
  - [ ] 6.10.5 Update/add TUI tests for the split layout and embedded flows

## 7. CLI Commands (Cobra)

- [ ] 7.1 Create root command in cmd/crook/main.go:
  - [ ] Initialize Cobra app
  - [ ] Add global flags
  - [ ] Set up logging
  - [ ] Load configuration
- [ ] 7.2 Create `crook down` command:
  - [ ] Accept node name argument
  - [ ] Accept command-specific flags
  - [ ] Initialize TUI or run headless based on flag
  - [ ] Execute down phase
- [ ] 7.3 Create `crook up` command:
  - [ ] Accept node name argument
  - [ ] Accept command-specific flags
  - [ ] Initialize TUI or run headless based on flag
  - [ ] Execute up phase
- [ ] 7.4 Create `crook version` command

## 8. Integration and Testing

- [ ] 8.1 Write integration tests for complete workflows:
  - [ ] Test down phase end-to-end
  - [ ] Test up phase end-to-end
  - [ ] Test error recovery scenarios
- [ ] 8.2 Test with real Kubernetes cluster (if available) or kind
- [ ] 8.3 Test TUI in various terminals (xterm, tmux, screen)
- [ ] 8.4 Test configuration loading from all sources
- [ ] 8.5 Test cancellation handling (Ctrl+C during operations)

## 9. Documentation and Polish

- [ ] 9.1 Write comprehensive README.md:
  - [ ] Installation instructions
  - [ ] Quick start guide
  - [ ] Configuration documentation
  - [ ] Command reference
  - [ ] Troubleshooting guide
- [ ] 9.2 Create EXAMPLES.md with common workflows
- [ ] 9.3 Add godoc comments to all exported functions
- [ ] 9.4 Create CONTRIBUTING.md with development setup
- [ ] 9.5 Add LICENSE file
- [ ] 9.6 Create justfile with common tasks:
  - [ ] build, test, lint, install targets
- [ ] 9.7 Set up CI/CD pipeline (GitHub Actions):
  - [ ] Run tests on PR
  - [ ] Build binaries for releases
  - [ ] Run linters (golangci-lint)
- [ ] 9.8 Create release process and versioning strategy

## 10. Final Validation

- [ ] 10.1 Run `openspec validate add-go-tui-interface --strict`
- [ ] 10.2 Verify all requirements have corresponding code
- [ ] 10.3 Verify all scenarios are tested
- [ ] 10.4 Perform manual end-to-end testing
- [ ] 10.5 Collect feedback from early users (if applicable)
- [ ] 10.6 Address any issues found during validation
- [ ] 10.7 Tag v1.0.0 release

## Notes

- Tasks can be parallelized where indicated (e.g., 2.x and 3.x can be worked on concurrently)
- Integration tests (8.x) require tasks 2-7 to be complete
- TUI work (6.x) can start after monitoring (5.x) is partially complete
- Each checkbox represents a verifiable unit of work that can be demonstrated or tested
