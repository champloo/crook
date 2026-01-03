# Capability: Kubernetes Client

Wrapper around Kubernetes client-go library providing domain-specific operations.

## ADDED Requirements

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

### Requirement: Deployment Operations

The system SHALL provide operations for scaling deployments and querying deployment status.

#### Scenario: Scale deployment

- **WHEN** system scales deployment "rook-ceph-osd-0" in namespace "rook-ceph" to 0 replicas
- **THEN** system retrieves deployment object
- **THEN** system updates `.spec.replicas = 0`
- **THEN** system patches deployment
- **THEN** system returns success immediately (does not wait)

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
- **THEN** system returns ownership chain: Pod → ReplicaSet → Deployment

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

### Requirement: Error Handling and Retries

The system SHALL handle transient Kubernetes API errors with exponential backoff retry.

#### Scenario: Transient network error

- **WHEN** API call fails with network timeout
- **THEN** system retries up to 3 times
- **THEN** system uses exponential backoff: 1s, 2s, 4s
- **THEN** system returns error if all retries fail
- **THEN** system includes original error and retry count in error message

#### Scenario: Permanent error

- **WHEN** API call fails with 404 Not Found
- **THEN** system does not retry
- **THEN** system returns error immediately

#### Scenario: API rate limiting

- **WHEN** API returns 429 Too Many Requests
- **THEN** system extracts Retry-After header
- **THEN** system waits specified duration before retry
- **THEN** system retries up to 3 times total

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
