# Capability: Cluster Monitoring

Real-time cluster health monitoring and status display for Rook-Ceph and Kubernetes resources.

## ADDED Requirements

### Requirement: Node Status Monitoring

The system SHALL monitor and display Kubernetes node status for the target node.

#### Scenario: Query node status

- **WHEN** system monitors node "worker-01"
- **THEN** system retrieves node object from API
- **THEN** system extracts and displays:
  - Node name
  - Ready condition status (True/False/Unknown)
  - Reason for not-ready state (if applicable)
  - Taints (especially "unschedulable" taint)
  - Schedulable status (from `.spec.unschedulable`)
  - Kubelet version
  - Number of pods currently running on node

#### Scenario: Node ready condition

- **WHEN** node Ready condition is True
- **THEN** system displays status as "Ready" in green
- **WHEN** node Ready condition is False
- **THEN** system displays status as "NotReady" in red with reason
- **WHEN** node Ready condition is Unknown
- **THEN** system displays status as "Unknown" in yellow with warning icon

#### Scenario: Node scheduling status

- **WHEN** node has `.spec.unschedulable = true`
- **THEN** system displays "SchedulingDisabled (Cordoned)" badge
- **WHEN** node has "NoSchedule" taint
- **THEN** system displays "SchedulingDisabled (Tainted)" badge
- **WHEN** node is schedulable
- **THEN** system displays "Schedulable" in normal text

### Requirement: Ceph Cluster Health Monitoring

The system SHALL monitor and display Ceph cluster health status.

#### Scenario: Query Ceph health

- **WHEN** system monitors Ceph cluster health
- **THEN** system executes `ceph status --format json` via rook-ceph-tools
- **THEN** system parses JSON output
- **THEN** system extracts:
  - Overall health (HEALTH_OK, HEALTH_WARN, HEALTH_ERR)
  - Health summary messages (if not HEALTH_OK)
  - Number of OSDs (total, up, in)
  - Number of monitors (total, in quorum)
  - Data usage (used, available, total)
  - PG states summary

#### Scenario: Healthy cluster

- **WHEN** Ceph status is HEALTH_OK
- **THEN** system displays "HEALTH_OK" in green with checkmark icon
- **THEN** system shows brief summary: "X OSDs up, Y monitors in quorum"

#### Scenario: Warning state

- **WHEN** Ceph status is HEALTH_WARN
- **THEN** system displays "HEALTH_WARN" in yellow with warning icon
- **THEN** system displays health summary messages
- **THEN** system highlights that maintenance may worsen cluster state

#### Scenario: Error state

- **WHEN** Ceph status is HEALTH_ERR
- **THEN** system displays "HEALTH_ERR" in red with error icon
- **THEN** system displays health summary messages
- **THEN** system warns that performing maintenance is risky
- **THEN** system requires explicit confirmation to proceed

### Requirement: OSD Status Monitoring

The system SHALL monitor and display OSD (Object Storage Daemon) status.

#### Scenario: List OSDs on node

- **WHEN** system monitors OSDs for node "worker-01"
- **THEN** system executes `ceph osd tree --format json`
- **THEN** system parses output to find OSDs on target node
- **THEN** system displays for each OSD:
  - OSD ID (e.g., osd.0)
  - Status (up/down, in/out)
  - Weight
  - Associated deployment name (e.g., rook-ceph-osd-0)

#### Scenario: OSD status indicators

- **WHEN** OSD is "up" and "in"
- **THEN** system displays status in green
- **WHEN** OSD is "down" or "out"
- **THEN** system displays status in red
- **WHEN** OSD has "noout" flag set
- **THEN** system displays "(noout)" badge next to status

### Requirement: Deployment Status Monitoring

The system SHALL monitor Rook-Ceph deployment status in configured namespaces.

#### Scenario: Monitor operator deployment

- **WHEN** system monitors rook-ceph-operator deployment
- **THEN** system retrieves deployment from configured namespace
- **THEN** system displays:
  - Desired replicas
  - Current replicas
  - Ready replicas
  - Available replicas
  - Deployment conditions (Available, Progressing)

#### Scenario: Monitor discovered deployments

- **WHEN** system discovers deployments to be scaled down
- **THEN** system monitors each deployment status
- **THEN** system displays table with columns:
  - Deployment name
  - Current replicas
  - Ready replicas
  - Status indicator (Ready, Scaling, Unavailable)

#### Scenario: Deployment not ready

- **WHEN** deployment has ready replicas < desired replicas
- **THEN** system displays status as "Scaling" in yellow
- **WHEN** deployment has 0 ready replicas but >0 desired
- **THEN** system displays status as "Unavailable" in red

### Requirement: Background Monitoring Refresh

The system SHALL refresh monitoring data at configurable intervals without blocking user interaction.

#### Scenario: Background data refresh

- **WHEN** monitoring is active
- **THEN** system spawns separate goroutines for K8s and Ceph resource polling
- **THEN** system refreshes Kubernetes resources (nodes, deployments, pods) at K8s interval
- **THEN** system refreshes Ceph resources (OSDs, cluster header) at Ceph interval
- **THEN** system aggregates updates from all pollers via channels
- **THEN** system sends combined update messages to Bubble Tea model
- **THEN** system updates display without blocking user input

#### Scenario: Refresh error handling

- **WHEN** API call fails during refresh
- **THEN** system logs error to log file
- **THEN** system displays stale data with last known values
- **THEN** system continues retry on next refresh cycle
- **THEN** system does not crash or stop refreshing other monitors
- **THEN** errors are tracked per resource type and combined for display

#### Scenario: Configurable refresh rates

- **WHEN** user configures refresh intervals in config file
- **THEN** system uses two categories of refresh intervals:
  - `ui.k8s-refresh-ms` (default: 2000ms) for nodes, deployments, pods
  - `ui.ceph-refresh-ms` (default: 5000ms) for OSDs, cluster header/status
- **THEN** system validates intervals are > 0
- **THEN** system applies configured intervals to respective resource pollers

### Requirement: Status Aggregation

The system SHALL aggregate individual statuses into overall health assessment.

#### Scenario: Calculate overall health

- **WHEN** system aggregates all monitoring data
- **THEN** system evaluates:
  - Node ready: critical
  - Ceph HEALTH_OK: warning if not OK, critical if ERR
  - Operator available: warning if not available
  - OSDs up: warning if any down
- **THEN** system determines overall status: Healthy, Degraded, Critical
- **THEN** system displays overall status prominently in the UI header

#### Scenario: Healthy state

- **WHEN** node is Ready, Ceph is HEALTH_OK, operator is available, all OSDs are up
- **THEN** system displays overall status "Healthy" in green
- **THEN** system shows checkmark icon

#### Scenario: Degraded state

- **WHEN** Ceph is HEALTH_WARN or some OSDs are down
- **THEN** system displays overall status "Degraded" in yellow
- **THEN** system shows warning icon
- **THEN** system lists degradation reasons

#### Scenario: Critical state

- **WHEN** node is NotReady or Ceph is HEALTH_ERR
- **THEN** system displays overall status "Critical" in red
- **THEN** system shows error icon
- **THEN** system strongly warns against proceeding with maintenance

### Requirement: Resource Listing for ls Command

The system SHALL provide comprehensive resource queries to support the `crook ls` command.

#### Scenario: List all cluster nodes

- **WHEN** system lists nodes for ls view
- **THEN** system queries all nodes via Kubernetes API
- **THEN** system returns for each node:
  - Name
  - Ready condition status
  - Roles (from labels `node-role.kubernetes.io/*`)
  - Schedulable status (from `.spec.unschedulable`)
  - Creation timestamp
  - Kubelet version
- **THEN** system annotates each node with count of Ceph pods running on it

#### Scenario: List Ceph deployments across namespaces

- **WHEN** system lists deployments for ls view
- **THEN** system queries deployments in configured rook-cluster-namespace
- **THEN** system filters by configured prefix patterns
- **THEN** system returns for each deployment:
  - Name and namespace
  - Replica counts (desired, current, ready, available)
  - Creation timestamp
  - Deployment conditions (Available, Progressing)
  - Labels (especially `ceph-osd-id` for OSD deployments)
- **THEN** system maps each deployment to its host node via pod spec

#### Scenario: List all OSDs with detailed status

- **WHEN** system lists OSDs for ls view
- **THEN** system executes `ceph osd tree --format json` via rook-ceph-tools
- **THEN** system parses JSON to extract OSD information
- **THEN** system returns for each OSD:
  - OSD ID (numeric and osd.N format)
  - Host/node name (from tree hierarchy)
  - Status (up/down)
  - In/Out state
  - Weight and reweight values
  - Device class (hdd, ssd, nvme)
  - Crush location (root, datacenter, rack, host)
- **THEN** system cross-references with Kubernetes deployments to map OSD ID to deployment name

#### Scenario: Query Ceph flags status

- **WHEN** system queries Ceph cluster flags
- **THEN** system executes `ceph osd dump --format json` via rook-ceph-tools
- **THEN** system extracts active flags (noout, noin, nodown, noup, etc.)
- **THEN** system returns list of currently set flags
- **THEN** system highlights maintenance-relevant flags (noout, norebalance)

#### Scenario: List pods by node

- **WHEN** system lists pods for a specific node
- **THEN** system queries pods with field selector `spec.nodeName=<node>`
- **THEN** system filters to Ceph-related pods (matching deployment prefixes)
- **THEN** system returns for each pod:
  - Name and namespace
  - Phase (Running, Pending, Succeeded, Failed)
  - Ready containers count
  - Restart count
  - Age
  - Owner deployment name (via ownership chain traversal)

#### Scenario: Aggregate storage usage

- **WHEN** system queries storage usage for ls view
- **THEN** system executes `ceph df --format json` via rook-ceph-tools
- **THEN** system extracts:
  - Total cluster capacity
  - Used storage (raw and percentage)
  - Available storage
  - Per-pool usage summary (optional, for detail view)

#### Scenario: Query monitor quorum status

- **WHEN** system queries monitor status for ls view
- **THEN** system executes `ceph mon stat --format json` via rook-ceph-tools
- **THEN** system extracts:
  - Total monitor count
  - Monitors in quorum (count and names)
  - Monitors out of quorum (if any)
  - Leader monitor name

### Requirement: Graceful Degradation on Ceph Failures

The system SHALL continue displaying available data when Ceph commands fail or timeout.

#### Scenario: Ceph commands timeout during ls

- **WHEN** system executes `crook ls` and Ceph commands timeout (cluster degraded)
- **THEN** system continues to display Kubernetes data (nodes, deployments, pods)
- **THEN** system omits OSD data from output (empty/null in JSON/YAML)
- **THEN** system omits cluster health from output header
- **THEN** system does not return error to user
- **THEN** system completes within timeout period (does not hang)

#### Scenario: Partial data in TUI mode

- **WHEN** TUI refreshes data and some Ceph commands fail
- **THEN** TUI displays available data (nodes, deployments, pods)
- **THEN** TUI shows empty or stale data for failed sections (OSDs, cluster health)
- **THEN** TUI continues to accept user input
- **THEN** TUI retries failed data on next refresh cycle

#### Scenario: Partial data in non-TUI output

- **WHEN** system executes `crook ls --output json` with degraded cluster
- **THEN** system outputs valid JSON with available data
- **THEN** system sets `cluster_health` to null if health fetch failed
- **THEN** system sets `osds` to null or empty array if OSD fetch failed
- **THEN** system includes `nodes`, `deployments`, `pods` from Kubernetes API
- **THEN** system exits with success (0) as partial data is valid output
