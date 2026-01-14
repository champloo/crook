# Known Implementation Gaps

This document lists features that were specified in the original OpenSpec specifications but intentionally not implemented, or were implemented differently than originally designed.

## Purpose

This document serves as:
1. Historical record of design decisions and intentional divergences
2. Potential future work backlog for features that could be added later
3. Transparency about spec vs implementation differences
4. Onboarding aid for new contributors

## TUI Features Not Implemented

| Feature | Original Spec Location | Status | Notes |
|---------|------------------------|--------|-------|
| Help overlay (`?` key) | tui-interface | Not implemented | Help shown in footer/status bar instead |
| Detail view on Enter | tui-interface | Component exists, not wired | `pkg/tui/components/detail.go` exists but not connected |
| g/G top/bottom navigation | tui-interface | Not in keybindings | Only j/k/arrows implemented |
| Log toggle (`l` key) | tui-interface | Removed | Commit 92c29ec |
| Progress bars per deployment | tui-interface | Different implementation | Uses status icons and X/Y counts instead |
| View Logs option in error state | tui-interface | Not implemented | Only retry/quit options available |

## Kubernetes Client Features Not Implemented

| Feature | Original Spec Location | Status | Notes |
|---------|------------------------|--------|-------|
| Exponential backoff retry | kubernetes-client | Not implemented | Task crook-8nb.3 tracks future work |
| Rate limit (429) handling | kubernetes-client | Not implemented | No Retry-After header handling |

## Configuration Features Changed or Removed

| Feature | Original Spec Location | Status | Notes |
|---------|------------------------|--------|-------|
| Theme configuration | configuration | Removed | Commit 205a5a2 |
| Per-resource refresh rates | configuration | Consolidated | Now only `ui.k8s-refresh-ms` and `ui.ceph-refresh-ms` |
| State file configuration | configuration | Removed | Stateless architecture uses nodeSelector discovery |
| Deployment filter config | configuration | Removed | Uses hardcoded Rook-Ceph prefixes in `DefaultRookCephPrefixes()` |
| Separate operator/cluster namespaces | configuration | Consolidated | Single `namespace` field for all Rook-Ceph resources |

## Architectural Decisions

### Stateless Architecture

**Decision:** Remove state file persistence in favor of nodeSelector-based discovery.

**Original Design:** Save deployment states to file, restore from file during up phase.

**Current Implementation:**
- Down phase: Discover deployments via nodeSelector, scale to 0
- Up phase: Discover scaled-down deployments via nodeSelector + replicas=0, restore to 1

**Rationale:**
- Eliminates state file corruption risk
- Works across node reboots without state persistence
- Simpler operational model
- No stale state issues

**Related:** Epic crook-bfs (Remove State Files)

### Replica Restoration

**Decision:** Always restore deployments to 1 replica.

**Original Design:** Save original replica count, restore to saved value.

**Current Implementation:** Restore all scaled-down deployments to exactly 1 replica.

**Rationale:**
- Rook-Ceph node-pinned deployments (OSDs, MONs, etc.) always have 1 replica
- Simplifies implementation without losing functionality
- Avoids edge cases with stale replica counts

### Deployment Discovery

**Decision:** Use nodeSelector/nodeAffinity instead of pod ownership traversal.

**Original Design:** Traverse Pod -> ReplicaSet -> Deployment ownership chain.

**Current Implementation:**
- Check `deployment.spec.template.spec.nodeSelector["kubernetes.io/hostname"]`
- Fallback to nodeAffinity requiredDuringScheduling

**Rationale:**
- Works for 0-replica deployments (no pods to traverse)
- More reliable for the up phase when pods don't exist
- Directly matches how Rook-Ceph pins deployments to nodes

## Future Work Candidates

These features could be implemented in future versions:

### Low Effort
- **g/G navigation** - Simple keybinding addition for top/bottom of lists
- **Help overlay** - Modal component showing all keybindings

### Medium Effort
- **Detail view wiring** - Connect existing detail component to Enter key
- **Exponential backoff** - Add retry logic to k8s client operations

### Considered But Unlikely
- **State file persistence** - Architectural decision to stay stateless
- **Per-resource refresh rates** - Complexity not justified; k8s/ceph split is sufficient
- **Theme configuration** - Low priority; terminal colors work well

## References

- Original specs: `openspec/changes/add-go-tui-interface/specs/`
- Current specs: `openspec/specs/`
- Related epics:
  - crook-bfs: Remove State Files
  - crook-j1j: Code review fixes
  - crook-8nb: Retry classification
