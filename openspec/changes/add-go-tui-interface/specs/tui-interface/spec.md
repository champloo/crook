# Capability: TUI Interface

Bubble Tea-based terminal user interface providing interactive workflows and real-time feedback.

## ADDED Requirements

### Requirement: Down Phase Interactive Workflow

The system SHALL provide an interactive TUI for the down phase with state transitions and progress tracking.

#### Scenario: Down phase state machine progression

- **WHEN** user launches `crook down <node>` command
- **THEN** system displays cluster health dashboard (initial state)
- **THEN** system prompts for confirmation showing node and impacted deployments
- **THEN** system transitions through states: Cordoning → SettingNoOut → ScalingOperator → DiscoveringDeployments → ScalingDeployments → Complete
- **THEN** system displays current state name and description at top of screen
- **THEN** system shows progress indicator for each async operation
- **THEN** system displays final summary with state file path on completion

#### Scenario: Down phase user confirmation

- **WHEN** user reaches confirmation state
- **THEN** system displays summary table showing:
  - Target node name
  - Number of deployments to be scaled down
  - List of deployment names
  - Estimated impact duration
- **THEN** system shows prompt "Proceed with down phase? (y/N)"
- **THEN** system proceeds only if user types 'y' or 'Y'
- **THEN** system aborts and exits if user types 'n', 'N', or Ctrl+C

#### Scenario: Down phase error state

- **WHEN** any operation fails during down phase
- **THEN** system transitions to error state
- **THEN** system displays error message in red with error icon
- **THEN** system shows which operation failed and current cluster state
- **THEN** system offers options: Retry, Abort, View Logs
- **THEN** system allows user to retry failed operation or abort gracefully

### Requirement: Up Phase Interactive Workflow

The system SHALL provide an interactive TUI for the up phase with state transitions and progress tracking.

#### Scenario: Up phase state machine progression

- **WHEN** user launches `crook up <node>` command
- **THEN** system loads and validates state file
- **THEN** system displays restore plan showing deployments and target replica counts
- **THEN** system prompts for confirmation
- **THEN** system transitions through states: Uncordoning → RestoringDeployments → ScalingOperator → UnsettingNoOut → Complete
- **THEN** system displays progress for each operation
- **THEN** system shows final success summary on completion

#### Scenario: Up phase with detailed restore plan

- **WHEN** user reaches confirmation state
- **THEN** system displays table with columns: Namespace, Deployment, Current Replicas, Target Replicas
- **THEN** system shows state file timestamp and node name
- **THEN** system highlights any anomalies (deployments no longer exist, unexpected current replicas)
- **THEN** system prompts "Proceed with restoration? (y/N)"

### Requirement: Real-time Progress Tracking

The system SHALL display live progress bars for all asynchronous operations.

#### Scenario: Deployment scaling progress

- **WHEN** system is scaling multiple deployments
- **THEN** system displays progress bar for each deployment
- **THEN** system updates progress bar based on readyReplicas count
- **THEN** system shows percentage complete (e.g., "3/5 deployments scaled")
- **THEN** system marks completed deployments with checkmark
- **THEN** system refreshes display every 100ms

#### Scenario: Indeterminate progress for unknown duration

- **WHEN** operation has unknown completion time (e.g., setting noout flag)
- **THEN** system displays spinner animation
- **THEN** system shows elapsed time counter
- **THEN** system shows operation description
- **THEN** system transitions to completion state when operation finishes

#### Scenario: Progress bar visual representation

- **WHEN** progress bar is displayed
- **THEN** system renders bar using block characters (█, ░)
- **THEN** system shows percentage number on right side
- **THEN** system uses color coding: blue for in-progress, green for complete, red for error
- **THEN** system ensures bar fits within terminal width (responsive)

### Requirement: Cluster Health Dashboard

The system SHALL display cluster health information before initiating maintenance operations.

#### Scenario: Dashboard initial view

- **WHEN** TUI launches for down phase
- **THEN** system displays dashboard showing:
  - Target node name and status (Ready/NotReady/SchedulingDisabled)
  - Rook operator deployment status (replicas, ready replicas)
  - Ceph cluster health (HEALTH_OK, HEALTH_WARN, HEALTH_ERR)
  - Number of OSDs and their status
  - Pods running on target node (count and list of deployment names)
- **THEN** system refreshes data every 2 seconds while in dashboard view
- **THEN** system allows user to proceed with 'Enter' or cancel with 'Esc'

#### Scenario: Dashboard health warning indicators

- **WHEN** cluster health is not HEALTH_OK
- **THEN** system displays health status in yellow (WARN) or red (ERR)
- **THEN** system shows warning icon next to health status
- **THEN** system displays additional confirmation: "Cluster health is not OK. Are you sure you want to proceed?"

#### Scenario: Dashboard live updates

- **WHEN** dashboard is visible
- **THEN** system polls Kubernetes API every 2 seconds for node status
- **THEN** system executes `ceph status` command every 5 seconds for cluster health
- **THEN** system updates display without flickering (efficient rendering)
- **THEN** system shows timestamp of last update in bottom corner

### Requirement: Keyboard Navigation

The system SHALL support intuitive keyboard controls for navigation and actions.

#### Scenario: Global keyboard shortcuts

- **WHEN** user presses keys in TUI
- **THEN** system responds to:
  - `Ctrl+C` or `Esc` → Abort operation and exit gracefully
  - `Enter` → Confirm current action or proceed to next state
  - `↑` / `↓` → Navigate lists or menu options
  - `y` / `n` → Answer yes/no prompts
  - `r` → Retry failed operation (in error state)
  - `l` → Toggle log view (show/hide detailed logs)
  - `?` → Show help overlay with keyboard shortcuts

#### Scenario: Context-sensitive controls

- **WHEN** user is in confirmation state
- **THEN** system accepts 'y', 'Y', 'n', 'N', or 'Enter' (default No)
- **WHEN** user is in error state
- **THEN** system accepts 'r' (retry), 'a' (abort), 'l' (logs)
- **WHEN** operation is in progress
- **THEN** system accepts only 'Ctrl+C' to cancel (gracefully if possible)

### Requirement: Resource List View (ls command)

The system SHALL provide a `crook ls` command to display Ceph-related Kubernetes resources in an interactive TUI view.

#### Scenario: Launch ls command

- **WHEN** user runs `crook ls`
- **THEN** system displays interactive TUI showing Ceph resources
- **THEN** system shows resources grouped by type: Nodes, Deployments, Pods, OSDs
- **THEN** system highlights resources in the configured rook-cluster-namespace
- **THEN** system allows keyboard navigation between resource types

#### Scenario: Node list view

- **WHEN** user views nodes in ls mode
- **THEN** system displays all cluster nodes with columns:
  - Name
  - Status (Ready/NotReady)
  - Roles (control-plane, worker)
  - Scheduling status (Schedulable/Cordoned)
  - Ceph pod count (pods matching deployment-filter prefixes)
  - Age
- **THEN** system highlights nodes with Ceph workloads
- **THEN** system shows cordoned nodes with distinct visual indicator

#### Scenario: Deployment list view

- **WHEN** user views deployments in ls mode
- **THEN** system displays deployments matching configured prefixes with columns:
  - Name
  - Namespace
  - Ready replicas (current/desired)
  - Node (where pods are scheduled)
  - Age
  - Status indicator (Ready, Scaling, Unavailable)
- **THEN** system groups deployments by type (osd, mon, exporter, crashcollector)
- **THEN** system shows scaled-down deployments (0 replicas) with warning indicator

#### Scenario: OSD list view

- **WHEN** user views OSDs in ls mode
- **THEN** system executes `ceph osd tree --format json` via rook-ceph-tools
- **THEN** system displays OSDs with columns:
  - OSD ID (e.g., osd.0)
  - Node hostname
  - Status (up/down)
  - In/Out state
  - Weight
  - Associated deployment name
  - PGs (primary PG count if available)
- **THEN** system shows OSDs marked "out" or "down" with warning indicator
- **THEN** system displays noout flag status in header if set

#### Scenario: Ceph cluster summary header

- **WHEN** ls TUI is displayed
- **THEN** system shows cluster summary header containing:
  - Cluster health status (HEALTH_OK/WARN/ERR)
  - Total OSDs (up/total, in/total)
  - Monitors in quorum (X/Y)
  - noout flag status (set/unset)
  - Storage usage (used/total)
- **THEN** system refreshes summary every 5 seconds

#### Scenario: Automatic resource refresh

- **WHEN** ls TUI is displayed
- **THEN** system automatically refreshes resource data in the background
- **THEN** Kubernetes resources (nodes, deployments, pods) refresh every 2 seconds
- **THEN** Ceph resources (OSDs, cluster health) refresh every 5 seconds
- **THEN** each resource type updates independently when new data is available
- **THEN** refresh intervals are configurable via config file
- **THEN** user can force immediate refresh with 'r' key

#### Scenario: Keyboard navigation in ls mode

- **WHEN** user presses keys in ls TUI
- **THEN** system responds to:
  - `Tab` or `1-4` → Switch between resource views (Nodes, Deployments, OSDs, Pods)
  - `↑` / `↓` or `j` / `k` → Navigate within current list
  - `Enter` → Show detailed view of selected resource
  - `r` → Refresh data immediately
  - `/` → Filter/search within current view
  - `q` or `Esc` → Exit ls mode
  - `?` → Show help overlay

#### Scenario: Resource detail view

- **WHEN** user presses Enter on a selected resource
- **THEN** system displays detail panel showing:
  - Full resource metadata (name, namespace, labels, annotations)
  - Resource-specific details (node conditions, deployment status, OSD stats)
  - Related resources (pods on node, deployment's pods, etc.)
- **THEN** user can press `Esc` or `q` to return to list view

#### Scenario: Filter resources

- **WHEN** user presses `/` and types filter text
- **THEN** system filters current view to show only matching resources
- **THEN** filter matches against resource name (case-insensitive)
- **THEN** user can press `Esc` to clear filter
- **THEN** filter indicator shows "Filtered: <query>" in status bar

#### Scenario: ls with node argument

- **WHEN** user runs `crook ls <node-name>`
- **THEN** system pre-filters all views to show only resources related to specified node
- **THEN** system shows deployments with pods on that node
- **THEN** system shows OSDs hosted on that node
- **THEN** system displays node-specific header with node status

### Requirement: Terminal Compatibility

The system SHALL render correctly across common terminal emulators and multiplexers.

#### Scenario: Terminal size handling

- **WHEN** terminal width is less than 80 columns
- **THEN** system displays warning "Terminal too narrow, recommend 80+ columns"
- **THEN** system renders with horizontal scrolling if necessary
- **WHEN** terminal is resized during operation
- **THEN** system detects resize event
- **THEN** system redraws interface to fit new dimensions
- **THEN** system maintains current state and progress

#### Scenario: Color support detection

- **WHEN** terminal does not support 256 colors
- **THEN** system falls back to 16-color palette
- **WHEN** terminal does not support colors (TERM=dumb)
- **THEN** system uses ASCII symbols only (no colors)
- **THEN** system uses text-based progress indicators (e.g., "[=====>    ]")

#### Scenario: Multiplexer compatibility

- **WHEN** running inside tmux or screen
- **THEN** system detects terminal type from $TERM variable
- **THEN** system uses compatible rendering mode
- **THEN** system handles multiplexer-specific key sequences correctly
