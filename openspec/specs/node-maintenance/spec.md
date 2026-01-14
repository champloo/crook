# node-maintenance Specification

## Purpose
Core orchestration of Rook-Ceph node maintenance operations (down and up phases), providing safe, stateless node management using nodeSelector-based deployment discovery.

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

### Requirement: Down Phase Orchestration
The system SHALL execute the node down phase by performing these steps in order: pre-flight validation, cordon node, set Ceph noout flag, scale down Rook operator, discover target deployments, and scale down deployments.

#### Scenario: Successful down phase
- **WHEN** user initiates down phase for node "worker-01"
- **THEN** system runs pre-flight validation checks
- **THEN** system cordons the node (marks unschedulable)
- **THEN** system sets Ceph `noout` flag via rook-ceph-tools
- **THEN** system scales rook-ceph-operator deployment to 0 replicas
- **THEN** system waits for rook-ceph-operator to reach 0 ready replicas
- **THEN** system discovers all node-pinned deployments via nodeSelector
- **THEN** system scales each discovered deployment to 0 replicas in order
- **THEN** system waits for each deployment's readyReplicas to become 0
- **THEN** system displays success message

#### Scenario: Down phase with no matching deployments
- **WHEN** user initiates down phase for node with no Ceph pods
- **THEN** system completes cordon, noout, and operator scaling steps
- **THEN** system discovers zero matching deployments
- **THEN** system completes successfully with "no node-pinned deployments found" message

#### Scenario: Down phase with ordered deployment scaling
- **WHEN** multiple Ceph deployments are discovered on node
- **THEN** system scales deployments in controlled order for Ceph stability
- **THEN** scaling order is: rook-ceph-osd (first), rook-ceph-mon, rook-ceph-exporter, rook-ceph-crashcollector (last)
- **THEN** system waits for each deployment to reach 0 replicas before proceeding to next
- **THEN** deployments not matching known prefixes are scaled last

#### Scenario: Skip already-scaled deployments
- **WHEN** a discovered deployment already has 0 replicas
- **THEN** system logs debug message and skips scaling that deployment
- **THEN** system does not wait for that deployment

### Requirement: Up Phase Orchestration
The system SHALL execute the node up phase by performing these steps in order: pre-flight validation, discover scaled-down deployments, uncordon node, restore deployments (with MON quorum gating), scale up Rook operator, and unset Ceph noout flag.

#### Scenario: Successful up phase
- **WHEN** user initiates up phase for node "worker-01"
- **THEN** system runs pre-flight validation checks
- **THEN** system discovers scaled-down deployments via nodeSelector (replicas=0)
- **THEN** system uncordons the node BEFORE scaling (allows pods to schedule)
- **THEN** system scales MON deployments first and waits for each to become ready
- **THEN** system waits for Ceph monitor quorum to establish
- **THEN** system scales remaining deployments in order (OSD, exporter, crashcollector)
- **THEN** system waits for each deployment to become ready before proceeding
- **THEN** system scales rook-ceph-operator back to 1 replica
- **THEN** system unsets Ceph `noout` flag (LAST - allows normal rebalancing)
- **THEN** system displays success message confirming node is operational

#### Scenario: Up phase with no deployments to restore
- **WHEN** user initiates up phase but no scaled-down deployments are found
- **THEN** system proceeds with uncordon, operator scaling, and noout unsetting
- **THEN** system completes with "no scaled-down deployments to restore" message

#### Scenario: Up phase with MON quorum gating
- **WHEN** restoring deployments that include MON deployments
- **THEN** system separates MON deployments from other deployments
- **THEN** system scales MON deployments first
- **THEN** system waits for each MON deployment to become ready (replicas=1, readyReplicas=1)
- **THEN** system polls Ceph quorum status via rook-ceph-tools
- **THEN** system proceeds to scale OSDs only after MON quorum is established
- **THEN** this ensures OSDs can recover properly (OSDs require active MON quorum)

#### Scenario: Up phase with pre-discovered deployments
- **WHEN** up phase is called with pre-discovered deployments (TUI confirmed plan)
- **THEN** system uses provided deployments directly without re-discovering
- **THEN** this ensures executed plan matches confirmed plan (no drift)

### Requirement: Pre-flight Validation
The system SHALL validate cluster prerequisites before allowing down or up phase operations.

#### Scenario: Pre-flight checks before down phase
- **WHEN** user initiates down phase
- **THEN** system validates cluster connectivity (implicit via client)
- **THEN** system validates target node exists in cluster
- **THEN** system validates rook-ceph namespace exists
- **THEN** system validates rook-ceph-tools deployment exists and has ready replicas
- **THEN** system validates RBAC permissions via SelfSubjectAccessReview
- **THEN** system proceeds to down phase only if all checks pass
- **THEN** system displays specific error message for any failed check

#### Scenario: Pre-flight checks before up phase
- **WHEN** user initiates up phase
- **THEN** system validates cluster connectivity
- **THEN** system validates target node exists in cluster
- **THEN** system validates rook-ceph namespace exists
- **THEN** system proceeds to up phase only if all checks pass

#### Scenario: RBAC permission validation
- **WHEN** validating permissions before down phase
- **THEN** system checks nodes get/patch permissions (cluster-scoped)
- **THEN** system checks deployments get/list permissions (namespaced)
- **THEN** system checks deployments/scale get/update permissions (namespaced)
- **THEN** system checks pods list and pods/exec create permissions (namespaced)
- **THEN** permission failures are reported with "Permission denied - contact cluster admin"
- **THEN** permission check errors assume allowed (best-effort validation)

### Requirement: Wait Operations
The system SHALL wait for asynchronous Kubernetes operations to complete before proceeding.

#### Scenario: Wait for deployment scale down
- **WHEN** deployment is scaled to 0 replicas
- **THEN** system polls deployment status every 5 seconds (default)
- **THEN** system checks `.status.readyReplicas` field
- **THEN** system continues waiting while readyReplicas > 0
- **THEN** system proceeds when readyReplicas becomes 0

#### Scenario: Wait for deployment scale up
- **WHEN** deployment is scaled to N replicas
- **THEN** system polls deployment status every 5 seconds (default)
- **THEN** system checks `.status.replicas` and `.status.readyReplicas` fields
- **THEN** system continues waiting while replicas != N OR readyReplicas != N
- **THEN** system proceeds when both replicas and readyReplicas equal N

#### Scenario: Wait for monitor quorum
- **WHEN** waiting for Ceph MON quorum during up phase
- **THEN** system polls quorum status via rook-ceph-tools every 5 seconds
- **THEN** system executes `ceph quorum_status` and parses JSON response
- **THEN** system checks if quorum_names matches expected MON count
- **THEN** system proceeds when quorum is established
- **THEN** system continues polling on transient errors

#### Scenario: Wait operation timeout
- **WHEN** wait operation exceeds configured timeout (default 300 seconds)
- **THEN** system displays timeout error with current deployment/quorum state
- **THEN** system provides detailed status (replicas, readyReplicas, availableReplicas)
- **THEN** system exits with error code
- **THEN** system suggests manual inspection command

#### Scenario: Wait operation context cancellation
- **WHEN** context is cancelled during wait operation
- **THEN** system returns immediately with cancellation error
- **THEN** system does not block on final status fetch

#### Scenario: Progress callbacks during wait
- **WHEN** ProgressCallback is configured in WaitOptions
- **THEN** system calls callback on each poll with current deployment status
- **THEN** this enables TUI/CLI to show real-time progress updates
