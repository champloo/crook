# Implementation Tasks: Migrate State to JSON

## 1. Update State Data Structures

- [ ] 1.1 Rename `Deployments` field to `Resources` in `State` struct in `pkg/state/state.go`
- [ ] 1.2 Rename `DeploymentState` struct to `Resource`
- [ ] 1.3 Add JSON struct tags to all fields (use camelCase: `operatorReplicas`, `resources`)
- [ ] 1.4 Add `json:",omitempty"` tags for optional fields
- [ ] 1.5 Update validation methods to work with JSON unmarshaled data
- [ ] 1.6 Add unit tests for JSON marshaling/unmarshaling
- [ ] 1.7 Update String() methods to reflect new field names

## 2. Implement JSON Parser

- [ ] 2.1 Create `pkg/state/parser.go` with JSON parsing logic
- [ ] 2.2 Implement `ParseJSON(reader io.Reader) (*State, error)` function
- [ ] 2.3 Add JSON schema validation (required fields, types)
- [ ] 2.4 Add detailed error messages with JSON field paths
- [ ] 2.5 Write unit tests for valid JSON parsing
- [ ] 2.6 Write unit tests for malformed JSON handling
- [ ] 2.7 Write unit tests for missing required fields
- [ ] 2.8 Write unit tests for invalid field types

## 3. Implement JSON Writer

- [ ] 3.1 Create `pkg/state/writer.go` with JSON writing logic
- [ ] 3.2 Implement `WriteJSON(writer io.Writer, state *State) error` function
- [ ] 3.3 Use 2-space indentation for pretty printing
- [ ] 3.4 Sort resources array before serialization (by namespace, then name)
- [ ] 3.5 Add support for compact JSON via environment variable `CROOK_STATE_COMPACT_JSON`
- [ ] 3.6 Write unit tests for JSON writing
- [ ] 3.7 Write unit tests for pretty printing
- [ ] 3.8 Write unit tests for deterministic output (sorted resources)
- [ ] 3.9 Write unit tests for compact JSON mode

## 4. Update File Path Resolution

- [ ] 4.1 Change default template from `.tsv` to `.json` in `pkg/config/defaults.go`
- [ ] 4.2 Update path resolution logic if needed
- [ ] 4.3 Update unit tests for path template resolution
- [ ] 4.4 Update documentation for new default extension

## 5. Update State File Operations

- [ ] 5.1 Update atomic write logic to use JSON writer
- [ ] 5.2 Update backup filename pattern to use `.json` extension
- [ ] 5.3 Update cleanup logic to handle `.json` files
- [ ] 5.4 Update state file listing to show JSON files
- [ ] 5.5 Write integration tests for atomic JSON writes
- [ ] 5.6 Write integration tests for backup functionality

## 6. Update Validation Logic

- [ ] 6.1 Update validation to work with JSON parsed data
- [ ] 6.2 Add JSON-specific validation errors (field paths)
- [ ] 6.3 Update age validation to parse RFC3339 timestamps
- [ ] 6.4 Update resource existence check (validate each resource's kind)
- [ ] 6.5 Write unit tests for JSON validation
- [ ] 6.6 Write unit tests for validation error messages

## 7. Update Tests

- [ ] 7.1 Update existing state persistence tests for JSON format
- [ ] 7.2 Add test fixtures with sample JSON state files
- [ ] 7.3 Write integration tests for down→up workflow with JSON
- [ ] 7.4 Update table-driven tests to use JSON format
- [ ] 7.5 Add benchmarks for JSON marshal/unmarshal performance
- [ ] 7.6 Ensure test coverage >85%

## 8. Update Configuration and Documentation

- [ ] 8.1 Update default config template to use `.json` extension
- [ ] 8.2 Update example config files
- [ ] 8.3 Update README with JSON format documentation
- [ ] 8.4 Add example JSON state file to documentation
- [ ] 8.5 Update CLI help text to mention JSON format
- [ ] 8.6 Update project.md to reflect JSON format and `resources` field

## 9. Update Beads Issues

- [ ] 9.1 Update crook-umx acceptance criteria for `Resources` field and JSON tags
- [ ] 9.2 Update crook-dvs to specify JSON parser implementation
- [ ] 9.3 Update crook-p0b to specify JSON writer implementation
- [ ] 9.4 Update crook-82i for JSON validation specifics
- [ ] 9.5 Update crook-9oh test descriptions for JSON format
- [ ] 9.6 Update crook-h41 backup to specify `.json` files
