# Change: Migrate State Persistence from TSV to JSON Format

## Why

The current TSV (tab-separated values) format for state files is functional but limited for structured data representation. JSON provides better structure, native Go support, industry-standard tooling, and easier evolution for complex state tracking requirements.

**Key motivations:**
- **Better structure:** JSON naturally represents nested data, metadata, and arrays
- **Tooling ecosystem:** Standard tools like `jq`, better IDE support, validation schemas
- **Native Go support:** `encoding/json` package with struct tags, no custom parsing
- **Human readability:** More intuitive for operators inspecting state files
- **Schema evolution:** Easier to add new fields without breaking backward compatibility
- **Error handling:** Better parse error messages with field-level validation
- **Future extensibility:** Using generic `resources` array allows tracking other Kubernetes resource types beyond Deployments

## What Changes

- State file format changes from TSV to JSON
- File structure becomes nested JSON object with `resources` array
- Parser/writer implementation uses `encoding/json` instead of custom TSV logic
- Default file path template changes from `crook-state-{{.Node}}.tsv` to `crook-state-{{.Node}}.json`
- Data model uses `resources` array (not `deployments`) for future extensibility
- Each resource includes `kind` field to support multiple resource types

## Impact

**Affected specs:**
- `specs/state-persistence` - All requirements updated for JSON format with `resources` array

**Affected code:**
- `pkg/state/state.go` - Data structures use `Resources []Resource` with JSON struct tags
- `pkg/state/parser.go` - New JSON parser using `encoding/json`
- `pkg/state/writer.go` - New JSON writer with pretty-printing
- `pkg/config/defaults.go` - Default template path `.tsv` → `.json`

**Affected tasks (crook-z1v epic):**
- `crook-umx` - Define state data structure with `Resources` field and JSON tags
- `crook-dvs` - Implement JSON format parser
- `crook-p0b` - Implement JSON format writer
- `crook-im4` - Update path resolution for `.json` extension
- `crook-82i` - Update validation for JSON format
- `crook-9oh` - Update tests for JSON format
- `crook-h41` - Backup functionality works with `.json` files

**Breaking changes:**
- **BREAKING:** File format changes from TSV to JSON (no backward compatibility)
- **BREAKING:** Field name changes from `deployments` to `resources`
- **Note:** Since bash script is not in active use, no migration path needed
