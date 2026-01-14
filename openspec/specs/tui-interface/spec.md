# tui-interface Specification

## Purpose
TBD - created by archiving change add-go-tui-interface. Update Purpose after archive.
## Requirements
### Requirement: TUI Launch

The system SHALL launch the interactive TUI when invoked without arguments.

#### Scenario: Launch TUI

- **WHEN** user runs `crook` with no arguments
- **THEN** interactive TUI is displayed with multi-pane layout

#### Scenario: Non-interactive ls command

- **WHEN** user runs `crook ls`
- **THEN** system outputs table or JSON format to stdout (NOT interactive TUI)
- **THEN** system exits after displaying data

### Requirement: Down Phase Interactive Workflow

The system SHALL provide an interactive TUI for the down phase with state transitions and progress tracking.

#### Scenario: Down phase state machine progression

- **WHEN** user initiates down phase for node via `d` key or `crook down <node>`
- **THEN** system transitions through states: Init -> Confirm -> PreFlight -> Cordoning -> SettingNoOut -> ScalingOperator -> DiscoveringDeployments -> ScalingDeployments -> Complete
- **THEN** NothingToDo state used when all deployments already at 0 replicas
- **THEN** Error state used on any failure
- **THEN** system displays current state name and description
- **THEN** system shows progress for each async operation

#### Scenario: Down phase user confirmation

- **WHEN** user reaches confirmation state
- **THEN** system displays summary showing:
  - Target node name
  - Deployment table with Name, Current replicas, Target replicas (0), Status
- **THEN** system shows prompt "Press y to confirm, Esc to cancel"
- **THEN** system proceeds only if user presses 'y' or 'Y'
- **THEN** system aborts if user presses 'n', 'N', or Esc

### Requirement: Up Phase Interactive Workflow

The system SHALL provide an interactive TUI for the up phase with state transitions and progress tracking.

#### Scenario: Up phase state machine progression

- **WHEN** user initiates up phase for node via `u` key or `crook up <node>`
- **THEN** system transitions through states: Init -> Discovering -> Confirm -> PreFlight -> Uncordoning -> RestoringDeployments -> ScalingOperator -> UnsettingNoOut -> Complete
- **THEN** NothingToDo state used when all deployments already at 1 replica
- **THEN** Error state used on any failure
- **THEN** system discovers scaled-down deployments via nodeSelector

#### Scenario: Up phase with detailed restore plan

- **WHEN** user reaches confirmation state
- **THEN** system displays table with: Deployment Name, Current Replicas (0), Target Replicas (1)
- **THEN** system prompts "Press y to confirm, Esc to cancel"

### Requirement: Real-time Progress Tracking

The system SHALL display live progress for all asynchronous operations.

#### Scenario: Deployment scaling progress

- **WHEN** system is scaling multiple deployments
- **THEN** system displays status icon per deployment (spinner, checkmark, X)
- **THEN** system shows X/Y count (e.g., "Scaling 3/5 deployments")
- **THEN** system marks completed deployments with checkmark
- **THEN** system refreshes display based on poll interval

#### Scenario: Indeterminate progress for unknown duration

- **WHEN** operation has unknown completion time (e.g., setting noout flag)
- **THEN** system displays spinner animation
- **THEN** system shows operation description
- **THEN** system transitions to completion state when operation finishes

### Requirement: Keyboard Navigation

The system SHALL support intuitive keyboard controls for navigation and actions.

#### Scenario: Global keyboard shortcuts

- **WHEN** user presses keys in TUI
- **THEN** system responds to:
  - `q` or `Esc` or `Ctrl+C` -> Exit TUI
  - Up/Down or `j` / `k` -> Navigate lists within active pane
  - `Tab` -> Cycle to next pane (Nodes -> Deployments -> OSDs)
  - `Shift+Tab` -> Cycle to previous pane
  - `1` / `2` / `3` -> Jump to Nodes / Deployments / OSDs pane
  - `[` / `]` -> Toggle Deployments/Pods view (when pane 2 is active)
  - `d` -> Start down phase for selected node (Nodes pane only)
  - `u` -> Start up phase for selected node (Nodes pane only)
  - `r` -> Refresh data immediately

#### Scenario: Context-sensitive controls

- **WHEN** user is in confirmation state (down/up phase)
- **THEN** system accepts 'y' / 'Y' to confirm, 'Esc' to cancel
- **WHEN** user is in error state
- **THEN** system accepts 'r' (retry) or 'q'/'Esc' (quit)

#### Scenario: Navigation during maintenance flows

- **WHEN** maintenance flow (down/up) is active
- **THEN** navigation keys remain functional:
  - `Tab` / `Shift+Tab` -> Cycle through panes
  - `1` / `2` / `3` -> Jump to specific pane
  - `[` / `]` -> Toggle Deployments/Pods view (when pane 2 active)
  - `j` / `k` / Up / Down -> Scroll within active pane
- **THEN** action keys are disabled (d, u, r for refresh, q)
- **THEN** flow-specific keys are handled by the maintenance flow:
  - Confirmation: 'y', 'n', 'Esc'
  - Running: 'Ctrl+C' (interrupt)
  - Error: 'r' (retry), 'q' (quit)
  - Complete: 'Enter', 'q', 'Esc' (exit)

### Requirement: Resource List View (Multi-pane Layout)

The system SHALL provide a multi-pane layout showing Ceph-related Kubernetes resources.

#### Scenario: Multi-pane ls layout

- **WHEN** user runs `crook` (launches TUI)
- **THEN** system displays three resource panes simultaneously:
  - Top: Nodes pane
  - Middle: Deployments pane (toggleable to Pods view)
  - Bottom: OSDs pane
- **THEN** each pane displays title with shortcut key in border
- **THEN** active pane has highlighted border
- **THEN** cluster header remains visible at top
- **THEN** status bar shows context-sensitive keyboard hints at bottom

#### Scenario: Deployments/Pods toggle in middle pane

- **WHEN** middle pane (Deployments) is active
- **THEN** user can press `[` to show Deployments view
- **THEN** user can press `]` to show Pods view
- **THEN** pane title updates to reflect current view

#### Scenario: Automatic resource refresh

- **WHEN** TUI is displayed
- **THEN** Kubernetes resources (nodes, deployments, pods) refresh every 2 seconds (configurable via `ui.k8s-refresh-ms`)
- **THEN** Ceph resources (OSDs, cluster health) refresh every 5 seconds (configurable via `ui.ceph-refresh-ms`)
- **THEN** user can force immediate refresh with 'r' key

### Requirement: Embedded Maintenance Flows

The system SHALL embed maintenance flows in the main TUI when initiated via keyboard shortcuts.

#### Scenario: Node maintenance via d/u keys

- **WHEN** user is in Nodes pane and presses `d`
- **THEN** system displays Maintenance pane with Down Phase confirmation UI
- **THEN** key input is routed to embedded flow
- **THEN** `Esc` cancels and returns Maintenance pane to idle
- **THEN** `y` confirms and starts workflow
- **WHEN** user presses `u`
- **THEN** same flow for Up Phase

### Requirement: Terminal Compatibility

The system SHALL render correctly across common terminal emulators.

#### Scenario: Terminal size handling

- **WHEN** terminal width is less than 80 columns
- **THEN** system displays warning about narrow terminal
- **WHEN** terminal is resized during operation
- **THEN** system redraws interface to fit new dimensions
- **THEN** system maintains current state

### Requirement: Down CLI Command

The system SHALL provide a `crook down` command for non-interactive node maintenance preparation.

#### Scenario: Down with confirmation

- **WHEN** user runs `crook down worker-01`
- **THEN** system displays summary of affected deployments
- **THEN** system prompts for confirmation
- **AND** on "y", executes down phase
- **AND** on "n", aborts operation

#### Scenario: Down with --yes flag

- **WHEN** user runs `crook down worker-01 --yes`
- **THEN** system executes down phase without prompting

#### Scenario: Down with timeout

- **WHEN** user runs `crook down worker-01 --timeout 5m`
- **THEN** system applies 5 minute timeout to entire operation
- **AND** returns error if timeout exceeded

#### Scenario: Down idempotent when already down

- **WHEN** user runs `crook down worker-01` on an already-prepared node
- **THEN** system reports success without making changes

#### Scenario: Down validation failure

- **WHEN** user runs `crook down nonexistent-node`
- **THEN** system returns error "node not found in cluster"
- **AND** exit code is 1

### Requirement: Up CLI Command

The system SHALL provide a `crook up` command for non-interactive node restoration.

#### Scenario: Up with confirmation

- **WHEN** user runs `crook up worker-01`
- **THEN** system discovers scaled-down deployments via nodeSelector
- **THEN** system displays summary of deployments to restore
- **THEN** system prompts for confirmation
- **AND** on "y", executes up phase

#### Scenario: Up with --yes flag

- **WHEN** user runs `crook up worker-01 --yes`
- **THEN** system executes up phase without prompting

#### Scenario: Up with timeout

- **WHEN** user runs `crook up worker-01 --timeout 15m`
- **THEN** system applies 15 minute timeout to entire operation

#### Scenario: Up idempotent when already up

- **WHEN** user runs `crook up worker-01` on an already-operational node
- **THEN** system reports success without making changes

### Requirement: Ls CLI Command

The system SHALL provide a `crook ls` command for non-interactive resource listing.

#### Scenario: List all resources in table format

- **WHEN** user runs `crook ls`
- **THEN** system outputs nodes, deployments, OSDs, and pods in table format
- **AND** system exits after displaying data

#### Scenario: List with JSON output

- **WHEN** user runs `crook ls --output json`
- **THEN** system outputs resources as JSON for automation
- **AND** JSON includes all resource types

#### Scenario: Filter by node name

- **WHEN** user runs `crook ls worker-01`
- **THEN** system shows only resources related to worker-01

#### Scenario: Show specific resource types

- **WHEN** user runs `crook ls --show nodes,osds`
- **THEN** system displays only nodes and OSDs
- **AND** omits deployments and pods

### Requirement: Global CLI Flags

The system SHALL provide global flags available to all commands.

#### Scenario: Namespace override

- **WHEN** user runs `crook ls --namespace custom-rook`
- **THEN** system operates against the custom-rook namespace
- **AND** default namespace is rook-ceph

#### Scenario: Config file specification

- **WHEN** user runs `crook down worker-01 --config /path/to/config.yaml`
- **THEN** system loads configuration from specified file

#### Scenario: Log level configuration

- **WHEN** user runs `crook down worker-01 --log-level debug`
- **THEN** system outputs debug-level log messages

#### Scenario: Log file output

- **WHEN** user runs `crook down worker-01 --log-file /var/log/crook.log`
- **THEN** system writes logs to specified file instead of stderr

### Requirement: CLI Exit Codes

The system SHALL use standard exit codes for CLI commands.

#### Scenario: Successful operation

- **WHEN** command completes successfully
- **THEN** exit code is 0

#### Scenario: Operation failure

- **WHEN** command fails due to error
- **THEN** exit code is 1
- **AND** error message is written to stderr

