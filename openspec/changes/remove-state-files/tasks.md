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

## 4. Update TUI Models
- [ ] 4.1 Remove `pkg/state` import from `pkg/tui/models/up.go`
- [ ] 4.2 Remove `loadedState *state.State` field from up model
- [ ] 4.3 Replace `state.ParseFile()` with nodeSelector discovery in up TUI
- [ ] 4.4 Remove `stateFilePath` field from down model
- [ ] 4.5 Update `DownPhaseCompleteMsg` to not include state file path
- [ ] 4.6 Remove state file path display from completion views
- [ ] 4.7 Update `pkg/tui/models/up_test.go`
- [ ] 4.8 Update `pkg/tui/models/down_test.go`

## 5. Remove State Package
- [ ] 5.1 Verify no imports remain: `rg "pkg/state" --type go`
- [ ] 5.2 Delete `pkg/state/state.go`
- [ ] 5.3 Delete `pkg/state/format.go`
- [ ] 5.4 Delete `pkg/state/backup.go`
- [ ] 5.5 Delete `pkg/state/path.go`
- [ ] 5.6 Delete `pkg/state/validation.go`
- [ ] 5.7 Delete `pkg/state/errors.go`
- [ ] 5.8 Remove any remaining state package tests

## 6. Simplify Configuration
- [ ] 6.1 Remove `StateConfig` struct from `pkg/config/config.go`
- [ ] 6.2 Remove `DeploymentFilterConfig` struct from `pkg/config/config.go`
- [ ] 6.3 Remove `DefaultStateFileTemplate`, `DefaultStateBackupDirectory` constants
- [ ] 6.4 Remove `DefaultDeploymentPrefixes` constant
- [ ] 6.5 Update `Config` struct to remove `State` and `DeploymentFilters` fields
- [ ] 6.6 Update `pkg/config/loader_test.go`

## 7. Update Commands
- [ ] 7.1 Remove `--state-file` flag from `cmd/crook/commands/down.go`
- [ ] 7.2 Remove `--state-file` flag from `cmd/crook/commands/up.go`
- [ ] 7.3 Remove deployment filter flags if any
- [ ] 7.4 Update command help text

## 8. Clean Up Discovery
- [ ] 8.1 Review `pkg/maintenance/discovery.go` for unused code
- [ ] 8.2 Remove or simplify pod-based discovery functions if no longer needed
- [ ] 8.3 Update discovery tests

## 9. Update ls Command (Optional Enhancement)
- [ ] 9.1 Update `ListCephDeployments` to use `GetDeploymentTargetNode` as primary
- [ ] 9.2 Keep pod-based detection as fallback for portable deployments
- [ ] 9.3 Verify `ls` shows node for 0-replica deployments

## 10. Final Validation
- [ ] 10.1 Run `go build ./...` - ensure clean build
- [ ] 10.2 Run `go test ./...` - all tests pass
- [ ] 10.3 Run `golangci-lint run` - no lint errors
- [ ] 10.4 Manual test: `crook down <node>` discovers deployments via nodeSelector
- [ ] 10.5 Manual test: `crook up <node>` restores deployments without state file
- [ ] 10.6 Manual test: Multi-node scenario works correctly
