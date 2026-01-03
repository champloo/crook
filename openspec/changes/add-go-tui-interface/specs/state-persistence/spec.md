# Capability: State Persistence

State file management for preserving and restoring deployment replica counts across maintenance operations.

## ADDED Requirements

### Requirement: State File Format

The system SHALL use TSV (tab-separated values) format with optional metadata header for state files.

#### Scenario: Write state file with metadata

- **WHEN** system saves state after down phase
- **THEN** system writes file with:
  - Header line: `# crook-state v1`
  - Metadata lines: `# Node: <node-name>`, `# Timestamp: <RFC3339>`, `# OperatorReplicas: <count>`
  - Data lines: `Deployment\t<namespace>\t<name>\t<replicas>`
- **THEN** system ensures file is written atomically (write temp file, rename)
- **THEN** system sets file permissions to 0644 (readable by all, writable by owner)

#### Scenario: Parse state file with metadata

- **WHEN** system reads state file for up phase
- **THEN** system ignores lines starting with `#` (comments/metadata)
- **THEN** system parses data lines as: `kind\tnamespace\tname\treplicas`
- **THEN** system validates each line has exactly 4 tab-separated fields
- **THEN** system validates replica count is valid integer >= 0

#### Scenario: Parse legacy state file

- **WHEN** state file has no metadata header (created by old version/bash script)
- **THEN** system parses file as pure TSV data
- **THEN** system skips header line if it contains "Deployment\tNamespace\t..." (column headers)
- **THEN** system successfully loads deployment data
- **THEN** system logs warning: "Legacy state file format detected, consider regenerating"

### Requirement: State File Path Resolution

The system SHALL resolve state file paths using template with node name substitution.

#### Scenario: Default template path

- **WHEN** user initiates down phase for node "worker-01"
- **THEN** system uses template `./crook-state-{{.Node}}.tsv`
- **THEN** system substitutes `{{.Node}}` with "worker-01"
- **THEN** system resolves to `./crook-state-worker-01.tsv`

#### Scenario: Explicit path override

- **WHEN** user specifies `--state-file /tmp/my-state.tsv`
- **THEN** system uses explicit path without template substitution
- **THEN** system ignores template configuration

#### Scenario: Template path with directory

- **WHEN** template is `~/.local/state/crook/state-{{.Node}}.tsv`
- **THEN** system expands `~` to user home directory
- **THEN** system creates parent directories if they don't exist
- **THEN** system substitutes node name
- **THEN** system writes state file to resolved path

### Requirement: State File Backup

The system SHALL create backup copies of existing state files before overwriting.

#### Scenario: Backup before write

- **WHEN** state file already exists at target path
- **THEN** system creates backup with timestamp: `<original-name>.backup.<RFC3339>.tsv`
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
- **THEN** system uses filename: `crook-state-<node>.<timestamp>.tsv`

### Requirement: State File Validation

The system SHALL validate state file integrity before using it for restoration.

#### Scenario: Validate state file exists

- **WHEN** up phase reads state file
- **THEN** system checks file exists at expected path
- **THEN** system returns error if file not found: "State file not found: <path>. Cannot proceed with up phase."

#### Scenario: Validate state file format

- **WHEN** system parses state file
- **THEN** system validates version header (if present) is supported (`v1`)
- **THEN** system validates each data line has exactly 4 fields
- **THEN** system validates first field is "Deployment"
- **THEN** system validates namespace is non-empty string
- **THEN** system validates deployment name is non-empty string
- **THEN** system validates replica count is integer >= 0
- **THEN** system returns detailed parse error if any validation fails

#### Scenario: Validate state file age

- **WHEN** state file contains timestamp metadata
- **THEN** system parses timestamp
- **THEN** system calculates age (current time - timestamp)
- **THEN** system warns if age > 24 hours: "State file is X hours old. Cluster state may have changed."
- **THEN** system requires explicit confirmation to proceed with old state file

#### Scenario: Validate deployments exist

- **WHEN** up phase loads state file
- **THEN** system queries Kubernetes API for each deployment listed
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
  - Deployments: array of {Namespace, Name, Replicas}

#### Scenario: State serialization

- **WHEN** system serializes state object to file
- **THEN** system writes metadata as comment lines
- **THEN** system writes each deployment as TSV line
- **THEN** system ensures consistent ordering (sort by namespace, then name)

#### Scenario: State deserialization

- **WHEN** system deserializes state file
- **THEN** system populates state object from metadata and data lines
- **THEN** system defaults OperatorReplicas to 1 if not specified (backward compatibility)
- **THEN** system returns parsed state object or detailed error

### Requirement: State File Cleanup

The system SHALL provide mechanism to clean up old state files and backups.

#### Scenario: List state files

- **WHEN** user runs `crook state list`
- **THEN** system searches current directory and configured backup directory
- **THEN** system displays table with: Node, Timestamp, File Path, Size
- **THEN** system highlights state files older than 7 days

#### Scenario: Clean old backups

- **WHEN** user runs `crook state clean --older-than 7d`
- **THEN** system finds backup files older than 7 days
- **THEN** system displays list of files to be deleted
- **THEN** system prompts for confirmation
- **THEN** system deletes confirmed files
- **THEN** system logs deletion summary: "Removed X backup files"

#### Scenario: Clean with dry-run

- **WHEN** user runs `crook state clean --older-than 7d --dry-run`
- **THEN** system displays files that would be deleted
- **THEN** system does not delete any files
- **THEN** system exits with message: "Dry run complete. No files deleted."
