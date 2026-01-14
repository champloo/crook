# Capability: Node Maintenance

Core orchestration of Rook-Ceph node maintenance operations (down and up phases).

## ADDED Requirements

### Requirement: Down Phase Orchestration

The system SHALL execute the node down phase by performing these steps in order: pre-flight validation, cordon node, set Ceph noout flag, scale down Rook operator, discover target deployments via nodeSelector, and scale down deployments.

#### Scenario: Successful down phase

- **WHEN** user initiates down phase for node "worker-01"
- **THEN** system runs pre-flight validation checks
- **THEN** system cordons the node
- **THEN** system sets Ceph `noout` flag via rook-ceph-tools
- **THEN** system scales rook-ceph-operator deployment to 0 replicas
- **THEN** system waits for operator to reach 0 ready replicas
- **THEN** system discovers all node-pinned deployments via nodeSelector
- **THEN** system scales each discovered deployment to 0 replicas
- **THEN** system waits for each deployment's readyReplicas to become 0
- **THEN** system displays success message

#### Scenario: Down phase with no matching deployments

- **WHEN** user initiates down phase for node with no Ceph pods
- **THEN** system completes cordon, noout, and operator scaling steps
- **THEN** system discovers zero matching deployments
- **THEN** system logs info that no deployments were found
- **THEN** system completes successfully

#### Scenario: Down phase with ordered deployment scaling

- **WHEN** multiple Ceph deployments are discovered on node
- **THEN** system scales deployments in controlled order for Ceph stability
- **THEN** scaling order is: rook-ceph-osd (first), rook-ceph-mon, rook-ceph-exporter, rook-ceph-crashcollector (last)
- **THEN** system waits for each deployment to reach 0 replicas before proceeding to next

### Requirement: Up Phase Orchestration

The system SHALL execute the node up phase by performing these steps in order: pre-flight validation, discover scaled-down deployments via nodeSelector, uncordon node, restore deployments with MON quorum gating, scale up Rook operator, and unset Ceph noout flag.

#### Scenario: Successful up phase

- **WHEN** user initiates up phase for node "worker-01"
- **THEN** system runs pre-flight validation checks
- **THEN** system discovers scaled-down deployments via nodeSelector (replicas=0)
- **THEN** system uncordons the node BEFORE scaling (allows pod scheduling)
- **THEN** system scales MON deployments first with quorum gating
- **THEN** system scales remaining deployments (OSDs, etc.) in order
- **THEN** system waits for each deployment's replicas to become ready
- **THEN** system scales rook-ceph-operator back to 1 replica
- **THEN** system unsets Ceph `noout` flag (LAST)
- **THEN** system displays success message confirming node is operational

#### Scenario: Up phase with ordered deployment scaling

- **WHEN** restoring deployments
- **THEN** system separates MON deployments from other deployments
- **THEN** system scales MON deployments first
- **THEN** system waits for Ceph monitor quorum to establish
- **THEN** system scales remaining deployments in order: rook-ceph-osd, rook-ceph-exporter, rook-ceph-crashcollector
- **THEN** system waits for each deployment to become ready before proceeding to next
- **THEN** MON quorum gating ensures safe cluster recovery

### Requirement: Pre-flight Validation

The system SHALL validate cluster prerequisites before allowing down or up phase operations.

#### Scenario: Pre-flight checks before down phase

- **WHEN** user initiates down phase
- **THEN** system validates cluster connectivity
- **THEN** system validates target node exists in cluster
- **THEN** system validates rook-ceph namespace exists
- **THEN** system validates rook-ceph-tools deployment exists and is ready
- **THEN** system validates current user has required RBAC permissions (via SelfSubjectAccessReview)
- **THEN** system proceeds to down phase only if all checks pass
- **THEN** system displays specific error if any check fails

#### Scenario: Pre-flight checks before up phase

- **WHEN** user initiates up phase
- **THEN** system validates cluster connectivity
- **THEN** system validates target node exists
- **THEN** system validates namespace exists
- **THEN** system proceeds to up phase only if all checks pass

### Requirement: Wait Operations

The system SHALL wait for asynchronous Kubernetes operations to complete before proceeding.

#### Scenario: Wait for deployment scale down

- **WHEN** deployment is scaled to 0 replicas
- **THEN** system polls deployment status every 5 seconds
- **THEN** system checks `.status.readyReplicas` field
- **THEN** system continues waiting while readyReplicas > 0
- **THEN** system proceeds when readyReplicas becomes 0

#### Scenario: Wait for deployment scale up

- **WHEN** deployment is scaled to N replicas
- **THEN** system polls deployment status every 5 seconds
- **THEN** system checks `.status.replicas` and `.status.readyReplicas` fields
- **THEN** system continues waiting while replicas != N OR readyReplicas != N
- **THEN** system proceeds when both match target

#### Scenario: Wait operation timeout

- **WHEN** wait operation exceeds configured timeout (default 300 seconds)
- **THEN** system displays timeout error with current state
- **THEN** system exits with error code
- **THEN** system provides guidance on manual inspection

### Requirement: Node-Pinned Deployment Discovery

The system SHALL discover node-pinned deployments by examining the `nodeSelector["kubernetes.io/hostname"]` field in deployment specs.

#### Scenario: Discover deployments via nodeSelector

- **WHEN** querying deployments for a target node
- **THEN** return all deployments where `spec.template.spec.nodeSelector["kubernetes.io/hostname"]` matches the target node name

#### Scenario: Discover deployments via nodeAffinity fallback

- **WHEN** a deployment has no nodeSelector but has `requiredDuringSchedulingIgnoredDuringExecution` nodeAffinity with `kubernetes.io/hostname` key
- **THEN** extract the target node from the first matching nodeAffinity value

#### Scenario: Ignore portable deployments

- **WHEN** a deployment has neither nodeSelector nor required nodeAffinity for hostname
- **THEN** exclude it from node-pinned deployment discovery

### Requirement: MON Quorum Gating

The system SHALL wait for Ceph monitor quorum before scaling up OSDs during up phase.

#### Scenario: Wait for monitor quorum

- **WHEN** restoring deployments that include MON deployments
- **THEN** system scales MON deployments first
- **THEN** system polls Ceph quorum status via rook-ceph-tools
- **THEN** system executes `ceph quorum_status` and parses JSON response
- **THEN** system proceeds to scale OSDs only after quorum is established

#### Scenario: Monitor quorum timeout

- **WHEN** quorum is not established within timeout
- **THEN** system displays timeout error with current quorum state
- **THEN** system shows monitors in quorum vs out of quorum
- **THEN** system suggests manual inspection via `ceph quorum_status`

### Requirement: Stateless Architecture

The system SHALL operate without external state files, using Kubernetes API as the single source of truth.

#### Scenario: No state configuration required

- **WHEN** running crook without state configuration in config file
- **THEN** node maintenance operations work correctly using nodeSelector discovery

#### Scenario: Restore to single replica

- **WHEN** restoring a scaled-down deployment
- **THEN** scale it to 1 replica (Rook-Ceph node-pinned deployments are always 1 replica by design)
