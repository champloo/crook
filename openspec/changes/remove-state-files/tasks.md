# Tasks: Remove State Files

## 1. Add Node Extraction Functions
- [ ] 1.1 Add `GetDeploymentTargetNode(dep)` function to `pkg/k8s/deployments.go`
- [ ] 1.2 Add `ListNodePinnedDeployments(ctx, namespace, nodeName)` function
- [ ] 1.3 Add `ListScaledDownDeploymentsForNode(ctx, namespace, nodeName)` function
- [ ] 1.4 Add unit tests for `GetDeploymentTargetNode` (nodeSelector and nodeAffinity cases)
- [ ] 1.5 Add unit tests for `ListNodePinnedDeployments`
- [ ] 1.6 Add unit tests for `ListScaledDownDeploymentsForNode`

## 2. Update DOWN Phase
- [ ] 2.1 Add `validateDeploymentReplicas()` warning function to `pkg/maintenance/down.go`
- [ ] 2.2 Replace pod-based discovery with `ListNodePinnedDeployments` in down phase
- [ ] 2.3 Remove state file writing from down phase
- [ ] 2.4 Update down phase tests

## 3. Update UP Phase
- [ ] 3.1 Replace state file loading with `ListScaledDownDeploymentsForNode` in up phase
- [ ] 3.2 Default to scaling deployments to 1 replica
- [ ] 3.3 Remove state file validation from up phase
- [ ] 3.4 Update up phase tests

## 4. Remove State Package
- [ ] 4.1 Delete `pkg/state/state.go`
- [ ] 4.2 Delete `pkg/state/format.go`
- [ ] 4.3 Delete `pkg/state/backup.go`
- [ ] 4.4 Delete `pkg/state/path.go`
- [ ] 4.5 Delete `pkg/state/validation.go`
- [ ] 4.6 Delete `pkg/state/errors.go`
- [ ] 4.7 Remove any remaining state package tests

## 5. Simplify Configuration
- [ ] 5.1 Remove `StateConfig` struct from `pkg/config/config.go`
- [ ] 5.2 Remove `DeploymentFilterConfig` struct from `pkg/config/config.go`
- [ ] 5.3 Remove `DefaultStateFileTemplate`, `DefaultStateBackupDirectory` constants
- [ ] 5.4 Remove `DefaultDeploymentPrefixes` constant
- [ ] 5.5 Update `Config` struct to remove `State` and `DeploymentFilters` fields
- [ ] 5.6 Update config tests

## 6. Update Commands
- [ ] 6.1 Remove `--state-file` flag from `cmd/crook/commands/down.go`
- [ ] 6.2 Remove `--state-file` flag from `cmd/crook/commands/up.go`
- [ ] 6.3 Remove deployment filter flags if any
- [ ] 6.4 Update command help text

## 7. Clean Up Discovery
- [ ] 7.1 Review `pkg/maintenance/discovery.go` for unused code
- [ ] 7.2 Remove or simplify pod-based discovery functions if no longer needed
- [ ] 7.3 Update discovery tests

## 8. Final Validation
- [ ] 8.1 Run `go build ./...` - ensure clean build
- [ ] 8.2 Run `go test ./...` - all tests pass
- [ ] 8.3 Run `golangci-lint run` - no lint errors
- [ ] 8.4 Manual test: `crook down <node>` discovers deployments via nodeSelector
- [ ] 8.5 Manual test: `crook up <node>` restores deployments without state file
