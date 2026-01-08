# Design: Remove State Files

## Context

Crook manages Rook-Ceph node maintenance by scaling down deployments before node maintenance and restoring them after. Currently, this requires:
1. State files to remember which deployments were scaled down
2. Prefix-based filtering to identify Rook-Ceph deployments
3. Pod-based discovery to map deployments to nodes

This complexity exists because the original design assumed we couldn't determine deployment-to-node mapping when deployments have 0 replicas.

**Key Discovery**: Rook-Ceph sets `nodeSelector["kubernetes.io/hostname"]` on all node-pinned deployments (OSDs, MONs, crashcollectors, exporters). This field persists in the deployment spec even when `replicas=0`.

## Goals / Non-Goals

**Goals:**
- Eliminate external state file dependency
- Simplify multi-node maintenance scenarios
- Reduce codebase complexity (~500 lines removed)
- Make system more resilient to failures

**Non-Goals:**
- Support non-Rook-Ceph deployments that don't use nodeSelector
- Preserve original replica counts (Rook always uses 1)
- Maintain backwards compatibility with old state files

## Decisions

### Decision 1: Use nodeSelector as primary filter
**What**: Query deployments by `nodeSelector["kubernetes.io/hostname"]` instead of prefix matching.

**Why**:
- Directly answers "which deployments belong to this node?"
- Works at 0 replicas (unlike pod-based discovery)
- Automatically handles all node-pinned Rook components
- Ignores portable components (MGR, MDS, RGW) that auto-migrate

**Alternatives considered**:
- Annotations: Requires modifying deployments, could conflict with operator
- Ceph OSD tree: Only works for OSDs, not MONs/crashcollectors

### Decision 2: Default to 1 replica on scale-up
**What**: Always scale deployments to 1 replica when bringing node back up.

**Why**:
- Rook-Ceph node-pinned deployments are always 1 replica by design
- One OSD per device, one MON per deployment
- `mon.count: 3` creates 3 deployments, not 1 deployment with 3 replicas

**Alternatives considered**:
- Store in annotation: Adds complexity, Rook always uses 1 anyway
- Query before scale-down: Unnecessary complexity

### Decision 3: Remove prefix-based filtering entirely
**What**: Remove `deployment-filters.prefixes` config option.

**Why**:
- nodeSelector provides more accurate filtering
- Prefix matching could miss new Rook components or catch non-Rook deployments
- Simpler configuration for users

### Decision 4: Add replica count warning
**What**: Warn (don't fail) if any deployment has >1 replica during down phase.

**Why**:
- Rook-Ceph should always have 1 replica for node-pinned deployments
- >1 indicates unexpected configuration that user should investigate
- Warning is informative without blocking legitimate use cases

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Deployments without nodeSelector won't be managed | These are portable and will auto-migrate when node is cordoned - this is correct behavior |
| Breaking change for users with state files | Clear migration path: delete state files, they're no longer needed |
| Breaking change for users with deployment-filters config | Config section simply ignored, can be removed |
| Edge case: deployment with 0 replicas before down | Will be scaled to 1 on up - acceptable for Rook components |

## Migration Plan

1. Users update to new crook version
2. Old state files are ignored (can be deleted)
3. Old config sections (`state`, `deployment-filters`) are ignored
4. System works immediately with new nodeSelector-based discovery

**Rollback**: Revert to previous crook version, state files still work.

## Open Questions

None - all questions resolved during research phase.
