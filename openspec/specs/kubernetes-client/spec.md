# kubernetes-client Specification

## Purpose
TBD - created by archiving change add-go-tui-interface. Update Purpose after archive.
## Requirements
### Requirement: Client Initialization

The system SHALL initialize Kubernetes client from kubeconfig with fallback to in-cluster configuration.

#### Scenario: Initialization from default kubeconfig

- **WHEN** system initializes Kubernetes client
- **THEN** system attempts to load kubeconfig from standard locations in order:
  1. Path specified in `--kubeconfig` flag
  2. Path from `KUBECONFIG` environment variable
  3. `~/.kube/config` (default location)
- **THEN** system uses current context from loaded kubeconfig
- **THEN** system creates clientset for API access
- **THEN** system validates connectivity by calling `/version` endpoint

#### Scenario: Initialization in-cluster

- **WHEN** system runs as pod inside Kubernetes cluster
- **THEN** system detects in-cluster environment (ServiceAccount token exists)
- **THEN** system loads config from `/var/run/secrets/kubernetes.io/serviceaccount/`
- **THEN** system uses in-cluster namespace from mounted secret
- **THEN** system creates clientset with in-cluster credentials

#### Scenario: Initialization failure

- **WHEN** kubeconfig is not found and not running in-cluster
- **THEN** system displays error: "Unable to load kubeconfig. Ensure kubectl is configured or KUBECONFIG is set."
- **THEN** system exits with error code 1

### Requirement: Node Operations

The system SHALL provide operations for cordoning and uncordoning nodes.

#### Scenario: Cordon node

- **WHEN** system cordons node "worker-01"
- **THEN** system retrieves node object from API
- **THEN** system sets `.spec.unschedulable = true`
- **THEN** system patches node with updated spec
- **THEN** system verifies node status shows "SchedulingDisabled"
- **THEN** system returns success

#### Scenario: Uncordon node

- **WHEN** system uncordons node "worker-01"
- **THEN** system retrieves node object from API
- **THEN** system sets `.spec.unschedulable = false`
- **THEN** system patches node with updated spec
- **THEN** system verifies node status does not show "SchedulingDisabled"
- **THEN** system returns success

#### Scenario: Node not found

- **WHEN** target node does not exist in cluster
- **THEN** system returns error: "Node '<name>' not found in cluster"
- **THEN** system does not retry

### Requirement: Deployment Scaling

The system SHALL scale deployments using the Kubernetes scale subresource for least-privilege RBAC.

#### Scenario: Scale deployment via scale subresource

- **WHEN** system scales deployment "rook-ceph-osd-0" in namespace "rook-ceph" to 0 replicas
- **THEN** system calls GetScale to retrieve current scale subresource
- **THEN** system updates scale.spec.replicas to 0
- **THEN** system calls UpdateScale to apply the change
- **THEN** system returns success immediately (does not wait)
- **THEN** system requires only `deployments/scale` permission (not full `deployments` update)

#### Scenario: Get deployment status

- **WHEN** system queries deployment status
- **THEN** system retrieves deployment object
- **THEN** system returns fields:
  - `.spec.replicas` (desired count)
  - `.status.replicas` (current count)
  - `.status.readyReplicas` (ready count)
  - `.status.updatedReplicas` (updated count)
  - `.status.availableReplicas` (available count)

#### Scenario: List deployments in namespace

- **WHEN** system lists deployments in "rook-ceph" namespace
- **THEN** system calls List API with namespace filter
- **THEN** system returns array of deployment names and replica counts

### Requirement: Node-Pinned Deployment Discovery

The system SHALL discover deployments pinned to specific nodes via nodeSelector or nodeAffinity.

#### Scenario: List node-pinned deployments

- **WHEN** system lists deployments pinned to node "worker-01"
- **THEN** system lists all deployments in namespace
- **THEN** system checks each deployment for nodeSelector with `kubernetes.io/hostname=worker-01`
- **THEN** system checks nodeAffinity requiredDuringScheduling as fallback
- **THEN** system returns deployments matching the target node

#### Scenario: List scaled-down deployments for node

- **WHEN** system lists scaled-down deployments for node "worker-01"
- **THEN** system finds node-pinned deployments
- **THEN** system filters to deployments with spec.replicas == 0
- **THEN** system returns filtered list for up phase restoration

#### Scenario: Get deployment target node

- **WHEN** system extracts target node from deployment spec
- **THEN** system checks nodeSelector for `kubernetes.io/hostname` first
- **THEN** system falls back to nodeAffinity requiredDuringScheduling
- **THEN** system returns hostname or empty string if not node-pinned

### Requirement: Pod Operations

The system SHALL provide operations for querying pods and their ownership.

#### Scenario: List pods on node

- **WHEN** system queries pods on node "worker-01"
- **THEN** system calls List API with field selector `spec.nodeName=worker-01`
- **THEN** system returns array of pod objects with:
  - Pod name
  - Namespace
  - Owner references (kind, name, UID)
  - Status (Running, Pending, etc.)

#### Scenario: Get pod owner chain

- **WHEN** pod has owner reference to ReplicaSet
- **THEN** system extracts ReplicaSet name from `.metadata.ownerReferences[0].name`
- **THEN** system retrieves ReplicaSet object
- **THEN** system extracts Deployment name from ReplicaSet's `.metadata.ownerReferences[0].name`
- **THEN** system returns ownership chain: Pod -> ReplicaSet -> Deployment

#### Scenario: Pod without owner

- **WHEN** pod has no owner references
- **THEN** system returns ownership chain with only Pod
- **THEN** system does not attempt further traversal

### Requirement: Ceph Command Execution

The system SHALL execute Ceph CLI commands via rook-ceph-tools deployment.

#### Scenario: Execute ceph command

- **WHEN** system executes `ceph osd set noout`
- **THEN** system finds rook-ceph-tools deployment in configured namespace
- **THEN** system selects a ready pod from the deployment
- **THEN** system executes command in pod: `ceph osd set noout`
- **THEN** system captures stdout and stderr
- **THEN** system returns command output and exit code

#### Scenario: Ceph tools not available

- **WHEN** rook-ceph-tools deployment does not exist
- **THEN** system returns error: "rook-ceph-tools deployment not found. Please install it following Rook documentation."
- **THEN** system provides documentation link

#### Scenario: Ceph command failure

- **WHEN** Ceph command exits with non-zero code
- **THEN** system captures stderr output
- **THEN** system returns error with command output
- **THEN** system includes exit code in error message

#### Scenario: Ceph command timeout

- **WHEN** Ceph command does not respond within configured timeout
- **THEN** system cancels the command execution
- **THEN** system returns error: "ceph command timed out after <timeout> (cluster may be degraded)"
- **THEN** system does not hang indefinitely
- **THEN** caller can handle timeout gracefully and continue with partial data
- **THEN** default timeout is 20 seconds (configurable via `timeouts.ceph-command-timeout-seconds`)

### Requirement: Context-based Cancellation

The system SHALL support cancellation of in-flight operations via Go context.

#### Scenario: User cancels operation

- **WHEN** user presses Ctrl+C during API call
- **THEN** system propagates context cancellation to client-go
- **THEN** system aborts in-flight HTTP requests
- **THEN** system cleans up resources
- **THEN** system returns context.Canceled error

#### Scenario: Operation timeout

- **WHEN** API call exceeds configured timeout (30 seconds default)
- **THEN** system cancels context
- **THEN** system aborts operation
- **THEN** system returns context.DeadlineExceeded error

