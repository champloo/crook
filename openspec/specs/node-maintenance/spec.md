# node-maintenance Specification

## Purpose
TBD - created by archiving change remove-state-files. Update Purpose after archive.
## Requirements
### Requirement: Node-Pinned Deployment Discovery
The system SHALL discover node-pinned deployments by examining the `nodeSelector["kubernetes.io/hostname"]` field in deployment specs, regardless of current replica count.

#### Scenario: Discover deployments via nodeSelector
- **WHEN** querying deployments for a target node
- **THEN** return all deployments where `spec.template.spec.nodeSelector["kubernetes.io/hostname"]` matches the target node name

#### Scenario: Discover deployments via nodeAffinity fallback
- **WHEN** a deployment has no nodeSelector but has `requiredDuringSchedulingIgnoredDuringExecution` nodeAffinity with `kubernetes.io/hostname` key
- **THEN** extract the target node from the first matching nodeAffinity value

#### Scenario: Ignore portable deployments
- **WHEN** a deployment has neither nodeSelector nor required nodeAffinity for hostname
- **THEN** exclude it from node-pinned deployment discovery (it will auto-migrate)

### Requirement: Stateless Down Phase
The system SHALL scale down node-pinned deployments without writing external state files.

#### Scenario: Scale down node-pinned deployments
- **WHEN** executing the down phase for a node
- **THEN** discover node-pinned deployments via nodeSelector and scale each to 0 replicas

#### Scenario: Warn on unexpected replica count
- **WHEN** a node-pinned deployment has more than 1 replica during down phase
- **THEN** log a warning indicating unexpected configuration (Rook-Ceph deployments should have 1 replica)

### Requirement: Stateless Up Phase
The system SHALL restore node-pinned deployments by discovering scaled-down deployments via nodeSelector, without requiring state files.

#### Scenario: Discover scaled-down deployments
- **WHEN** executing the up phase for a node
- **THEN** discover deployments where nodeSelector matches the node AND replicas equals 0

#### Scenario: Restore to single replica
- **WHEN** restoring a scaled-down deployment
- **THEN** scale it to 1 replica (Rook-Ceph node-pinned deployments are always 1 replica by design)

### Requirement: Simplified Configuration
The system SHALL NOT require state file or deployment filter configuration for node maintenance operations.

#### Scenario: No state configuration required
- **WHEN** running crook without state configuration in config file
- **THEN** node maintenance operations work correctly using nodeSelector discovery

#### Scenario: No deployment filter required
- **WHEN** running crook without deployment-filters configuration
- **THEN** node maintenance operations work correctly using nodeSelector as the filter

