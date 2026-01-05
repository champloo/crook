# Capability: State Persistence

State file management for preserving and restoring deployment replica counts across maintenance operations.

## ADDED Requirements

### Requirement: State File Format

The system SHALL use JSON format for state files with structured metadata and resource data.

#### Scenario: Write state file with structured JSON

- **WHEN** system saves state after down phase
- **THEN** system writes JSON file with:
  - Root object with fields: `version`, `node`, `timestamp`, `operatorReplicas`, `resources`
  - `version`: string value `"v1"`
  - `node`: string with node name
  - `timestamp`: RFC3339 formatted timestamp string
  - `operatorReplicas`: integer count
  - `resources`: array of objects with `kind`, `namespace`, `name`, `replicas` fields
- **THEN** system writes JSON with 2-space indentation (pretty-printed)
- **THEN** system writes a trailing newline at end of file
- **THEN** system ensures file is written atomically (write temp file, rename)
- **THEN** system sets file permissions to 0644 (readable by all, writable by owner)
- **THEN** system uses `.json` file extension

#### Scenario: Parse state JSON file

- **WHEN** system reads state file for up phase
- **THEN** system parses file as JSON
- **THEN** system validates root object has required fields: `version`, `resources`
- **THEN** system validates each resource has: `kind`, `namespace`, `name`, `replicas`
- **THEN** system validates replica count is integer >= 0
- **THEN** system returns structured error for malformed JSON

#### Scenario: Deterministic JSON output

- **WHEN** system writes the same logical state content multiple times
- **THEN** system produces byte-identical JSON output
- **THEN** system sorts `resources` by `namespace`, then `name` (ascending) before writing

#### Scenario: Example JSON state file structure

- **WHEN** viewing example state file
- **THEN** JSON structure is:
```json
{
  "version": "v1",
  "node": "worker-01",
  "timestamp": "2024-01-01T12:00:00Z",
  "operatorReplicas": 1,
  "resources": [
    {
      "kind": "Deployment",
      "namespace": "rook-ceph",
      "name": "rook-ceph-osd-0",
      "replicas": 1
    },
    {
      "kind": "Deployment",
      "namespace": "rook-ceph",
      "name": "rook-ceph-mon-a",
      "replicas": 1
    }
  ]
}
```

### Requirement: State File Path Resolution

The system SHALL resolve state file paths using template with node name substitution.

#### Scenario: Default template path

- **WHEN** user initiates down phase for node "worker-01"
- **THEN** system uses template `./crook-state-{{.Node}}.json`
- **THEN** system substitutes `{{.Node}}` with "worker-01"
- **THEN** system resolves to `./crook-state-worker-01.json`

#### Scenario: Explicit path override

- **WHEN** user specifies `--state-file /tmp/my-state.json`
- **THEN** system uses explicit path without template substitution
- **THEN** system ignores template configuration

#### Scenario: Template path with directory

- **WHEN** template is `~/.local/state/crook/state-{{.Node}}.json`
- **THEN** system expands `~` to user home directory
- **THEN** system creates parent directories if they don't exist
- **THEN** system substitutes node name
- **THEN** system writes state file to resolved path

### Requirement: State File Backup

The system SHALL create backup copies of existing state files before overwriting.

#### Scenario: Backup before write

- **WHEN** state file already exists at target path
- **THEN** system creates backup with timestamp: `<original-name>.backup.<RFC3339>.json`
- **THEN** system writes new state file to original path
- **THEN** system logs: "Backed up existing state file to <backup-path>"

#### Scenario: Backup disabled

- **WHEN** config has `state.backup-enabled: false`
- **THEN** system overwrites existing state file without backup
- **THEN** system logs warning: "Overwriting state file without backup (backup disabled in config)"

#### Scenario: Backup directory management

- **WHEN** config specifies `state.backup-directory`
- **THEN** system stores backups in specified directory
- **THEN** system creates backup directory if it doesn't exist
- **THEN** system uses filename: `crook-state-<node>.<timestamp>.json`

### Requirement: State File Validation

The system SHALL validate state file integrity before using it for restoration.

#### Scenario: Validate state file exists

- **WHEN** up phase reads state file
- **THEN** system checks file exists at expected path
- **THEN** system returns error if file not found: "State file not found: <path>. Cannot proceed with up phase."

#### Scenario: Validate JSON state file format

- **WHEN** system parses state file
- **THEN** system validates JSON is well-formed
- **THEN** system validates version field is present and supported (`v1`)
- **THEN** system validates resources is array
- **THEN** system validates each resource has required fields
- **THEN** system validates `kind` field is supported (initially: `Deployment`)
- **THEN** system validates namespace is non-empty string
- **THEN** system validates resource name is non-empty string
- **THEN** system validates replica count is integer >= 0
- **THEN** system returns detailed parse error if any validation fails

#### Scenario: Validate state file age

- **WHEN** state file contains timestamp field
- **THEN** system parses timestamp
- **THEN** system calculates age (current time - timestamp)
- **THEN** system warns if age > 24 hours: "State file is X hours old. Cluster state may have changed."
- **THEN** system requires explicit confirmation to proceed with old state file

#### Scenario: Validate deployments exist

- **WHEN** up phase loads state file
- **THEN** system queries Kubernetes API for each resource with `kind: "Deployment"`
- **THEN** system warns if any deployment no longer exists
- **THEN** system prompts: "Deployment <name> not found. Skip it and continue? (y/N)"
- **THEN** system proceeds only with user confirmation

### Requirement: Atomic State File Operations

The system SHALL ensure state file writes are atomic to prevent corruption.

#### Scenario: Atomic write

- **WHEN** system writes state file
- **THEN** system generates unique temporary filename: `<path>.tmp.<random>`
- **THEN** system writes complete state data to temporary file
- **THEN** system flushes data to disk (fsync)
- **THEN** system renames temporary file to target path (atomic operation)
- **THEN** system ensures old file is replaced atomically

#### Scenario: Write failure handling

- **WHEN** write operation fails (disk full, permissions)
- **THEN** system does not remove temporary file (for debugging)
- **THEN** system returns error with detailed message
- **THEN** system does not corrupt existing state file
- **THEN** system logs temporary file path for manual inspection

### Requirement: State Data Structure

The system SHALL maintain structured representation of state in memory.

#### Scenario: State object structure

- **WHEN** system captures state during down phase
- **THEN** system creates state object containing:
  - Version: "v1"
  - Node: node name string
  - Timestamp: RFC3339 formatted timestamp
  - OperatorReplicas: original operator replica count
  - Resources: array of {Kind, Namespace, Name, Replicas}

#### Scenario: State serialization

- **WHEN** system serializes state object to file
- **THEN** system writes the state as JSON
- **THEN** system ensures consistent ordering (sort by namespace, then name)

#### Scenario: State deserialization

- **WHEN** system deserializes state file
- **THEN** system populates state object from JSON fields
- **THEN** system defaults OperatorReplicas to 1 if not specified
- **THEN** system returns parsed state object or detailed error
