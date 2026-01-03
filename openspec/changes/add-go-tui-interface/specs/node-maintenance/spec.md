# Capability: Node Maintenance

Core orchestration of Rook-Ceph node maintenance operations (down and up phases).

## ADDED Requirements

### Requirement: Down Phase Orchestration

The system SHALL execute the node down phase by performing these steps in order: cordon node, set Ceph noout flag, scale down Rook operator, discover target deployments, scale down deployments, and save state to file.

#### Scenario: Successful down phase

- **WHEN** user initiates down phase for node "worker-01"
- **THEN** system cordons the node
- **THEN** system sets Ceph `noout` flag via rook-ceph-tools
- **THEN** system scales rook-ceph-operator deployment to 0 replicas
- **THEN** system discovers all deployments with pods on target node matching configured prefixes
- **THEN** system scales each discovered deployment to 0 replicas
- **THEN** system waits for each deployment's readyReplicas to become 0
- **THEN** system saves deployment state (namespace, name, original replica count) to JSON state file
- **THEN** system displays success message with state file location

#### Scenario: Down phase with no matching deployments

- **WHEN** user initiates down phase for node with no Ceph pods
- **THEN** system completes cordon, noout, and operator scaling steps
- **THEN** system discovers zero matching deployments
- **THEN** system logs warning that no deployments were found
- **THEN** system creates empty state file for consistency
- **THEN** system completes successfully

#### Scenario: Down phase interrupted during deployment scaling

- **WHEN** deployment scaling fails or is interrupted (network error, timeout)
- **THEN** system captures error details
- **THEN** system displays current state (which deployments succeeded, which failed)
- **THEN** system exits with error code
- **THEN** system does NOT create state file if any operation failed

#### Scenario: Down phase with ordered deployment scaling

- **WHEN** multiple Ceph deployments are discovered on node
- **THEN** system scales deployments in controlled order for Ceph stability
- **THEN** scaling order is: rook-ceph-osd (first), rook-ceph-mon, rook-ceph-exporter, rook-ceph-crashcollector (last)
- **THEN** system waits for each deployment to reach 0 replicas before proceeding to next
- **THEN** system records all deployments in state file

### Requirement: Up Phase Orchestration

The system SHALL execute the node up phase by performing these steps in order: validate state file exists, restore deployment replicas, scale up Rook operator, unset Ceph noout flag, and uncordon node.

#### Scenario: Successful up phase

- **WHEN** user initiates up phase for node "worker-01"
- **THEN** system validates state file exists at expected path
- **THEN** system parses state file and validates format
- **THEN** system scales each deployment listed in state file back to original replica count
- **THEN** system waits for each deployment's replicas to reach desired count
- **THEN** system scales rook-ceph-operator back to 1 replica
- **THEN** system unsets Ceph `noout` flag
- **THEN** system uncordons the node
- **THEN** system displays success message confirming node is operational

#### Scenario: Up phase with missing state file

- **WHEN** user initiates up phase but state file does not exist
- **THEN** system displays error message indicating state file not found
- **THEN** system shows expected state file path
- **THEN** system exits with error code without modifying cluster

#### Scenario: Up phase with corrupted state file

- **WHEN** state file exists but has invalid format or corrupted data
- **THEN** system displays error message with parse details
- **THEN** system exits with error code without modifying cluster
- **THEN** system suggests manual inspection of state file

#### Scenario: Up phase with ordered deployment scaling

- **WHEN** restoring deployments from state file
- **THEN** system scales deployments in controlled order for Ceph stability
- **THEN** scaling order is: rook-ceph-mon (first), rook-ceph-osd, rook-ceph-exporter, rook-ceph-crashcollector (last)
- **THEN** system waits for each deployment to become ready before proceeding to next
- **THEN** system ensures monitors establish quorum before scaling OSDs
- **THEN** order is different from down phase to ensure safe cluster recovery

### Requirement: Pre-flight Validation

The system SHALL validate cluster prerequisites before allowing down or up phase operations.

#### Scenario: Pre-flight checks before down phase

- **WHEN** user initiates down phase
- **THEN** system validates kubectl connectivity to cluster
- **THEN** system validates target node exists in cluster
- **THEN** system validates rook-operator-namespace exists
- **THEN** system validates rook-cluster-namespace exists
- **THEN** system validates rook-ceph-tools deployment exists and is ready
- **THEN** system validates current user has required RBAC permissions (cordon, scale, exec)
- **THEN** system proceeds to down phase only if all checks pass
- **THEN** system displays specific error if any check fails

#### Scenario: Pre-flight checks before up phase

- **WHEN** user initiates up phase
- **THEN** system validates state file exists and is readable
- **THEN** system validates kubectl connectivity to cluster
- **THEN** system validates all deployments in state file still exist
- **THEN** system proceeds to up phase only if all checks pass

### Requirement: Deployment Discovery

The system SHALL discover target deployments by querying pods on the target node and filtering by configured prefixes.

#### Scenario: Discovery with multiple matching deployments

- **WHEN** node has pods from deployments: rook-ceph-osd-0, rook-ceph-mon-a, rook-ceph-exporter
- **THEN** system queries pods with field selector `spec.nodeName=<node>`
- **THEN** system traces ownership: Pod → ReplicaSet → Deployment
- **THEN** system filters deployments matching regex `^(rook-ceph-osd|rook-ceph-mon|rook-ceph-exporter|rook-ceph-crashcollector)`
- **THEN** system returns unique list of deployment names
- **THEN** system includes deployments from configured namespace only

#### Scenario: Discovery with ownership chain traversal

- **WHEN** pod is owned by ReplicaSet which is owned by Deployment
- **THEN** system reads pod's ownerReferences for ReplicaSet
- **THEN** system reads ReplicaSet's ownerReferences for Deployment
- **THEN** system returns deployment name if prefix matches

### Requirement: Wait Operations

The system SHALL wait for asynchronous Kubernetes operations to complete before proceeding.

#### Scenario: Wait for deployment scale down

- **WHEN** deployment is scaled to 0 replicas
- **THEN** system polls deployment status every 5 seconds
- **THEN** system checks `.status.readyReplicas` field
- **THEN** system continues waiting while readyReplicas is non-empty
- **THEN** system proceeds when readyReplicas becomes empty or null

#### Scenario: Wait for deployment scale up

- **WHEN** deployment is scaled to N replicas
- **THEN** system polls deployment status every 5 seconds
- **THEN** system checks `.status.replicas` field
- **THEN** system continues waiting while replicas != N
- **THEN** system proceeds when replicas equals desired count

#### Scenario: Wait operation timeout

- **WHEN** wait operation exceeds configured timeout (default 300 seconds)
- **THEN** system displays timeout error with current state
- **THEN** system exits with error code
- **THEN** system provides guidance on manual inspection
