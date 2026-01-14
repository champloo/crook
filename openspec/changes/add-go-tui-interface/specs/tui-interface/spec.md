# Capability: TUI Interface

Bubble Tea-based terminal user interface providing interactive workflows and real-time feedback.

## ADDED Requirements

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
- **THEN** system transitions through states: Init → Confirm → PreFlight → Cordoning → SettingNoOut → ScalingOperator → DiscoveringDeployments → ScalingDeployments → Complete
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
- **THEN** system transitions through states: Init → Discovering → Confirm → PreFlight → Uncordoning → RestoringDeployments → ScalingOperator → UnsettingNoOut → Complete
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
  - `q` or `Esc` or `Ctrl+C` → Exit TUI
  - `↑` / `↓` or `j` / `k` → Navigate lists within active pane
  - `Tab` → Cycle to next pane (Nodes → Deployments → OSDs)
  - `Shift+Tab` → Cycle to previous pane
  - `1` / `2` / `3` → Jump to Nodes / Deployments / OSDs pane
  - `[` / `]` → Toggle Deployments/Pods view (when pane 2 is active)
  - `d` → Start down phase for selected node (Nodes pane only)
  - `u` → Start up phase for selected node (Nodes pane only)
  - `r` → Refresh data immediately

#### Scenario: Context-sensitive controls

- **WHEN** user is in confirmation state (down/up phase)
- **THEN** system accepts 'y' / 'Y' to confirm, 'Esc' to cancel
- **WHEN** user is in error state
- **THEN** system accepts 'r' (retry) or 'q'/'Esc' (quit)

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
