package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andri/crook/internal/logger"
)

type stateJSON struct {
	Version          *string         `json:"version"`
	Node             *string         `json:"node"`
	Timestamp        *string         `json:"timestamp"`
	OperatorReplicas *int            `json:"operatorReplicas"`
	Resources        *[]resourceJSON `json:"resources"`
}

type resourceJSON struct {
	Kind      *string `json:"kind"`
	Namespace *string `json:"namespace"`
	Name      *string `json:"name"`
	Replicas  *int    `json:"replicas"`
}

// Parse parses state data from JSON bytes.
func Parse(data []byte) (*State, error) {
	return parseState(data, "")
}

// ParseFile parses a JSON state file from disk.
func ParseFile(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state file %s: %w", path, err)
	}
	return parseState(data, path)
}

// WriteFile writes state data to disk as deterministic JSON.
func WriteFile(path string, state State) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("state file path is required")
	}

	normalized := state
	if normalized.Version == "" {
		normalized.Version = VersionV1
	}
	if normalized.Version != VersionV1 {
		return fmt.Errorf("unsupported state version: %s", normalized.Version)
	}
	if strings.TrimSpace(normalized.Node) == "" {
		return errors.New("state node is required")
	}
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = time.Now().UTC()
	}
	if normalized.OperatorReplicas < 0 {
		return fmt.Errorf("operator replicas must be >= 0, got %d", normalized.OperatorReplicas)
	}

	normalized.Resources = SortedResources(normalized.Resources)
	for i, resource := range normalized.Resources {
		if strings.TrimSpace(resource.Kind) == "" {
			return fmt.Errorf("resource %d kind is required", i)
		}
		if strings.TrimSpace(resource.Namespace) == "" {
			return fmt.Errorf("resource %d namespace is required", i)
		}
		if strings.TrimSpace(resource.Name) == "" {
			return fmt.Errorf("resource %d name is required", i)
		}
		if resource.Replicas < 0 {
			return fmt.Errorf("resource %d replicas must be >= 0, got %d", i, resource.Replicas)
		}
	}

	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state file: %w", err)
	}
	payload = append(payload, '\n')

	return writeFileAtomic(path, payload)
}

func parseState(data []byte, path string) (*State, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, &ParseError{Path: path, Err: errors.New("empty state file")}
	}

	var raw stateJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&raw); err != nil {
		return nil, &ParseError{Path: path, Err: err}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, &ParseError{Path: path, Err: errors.New("unexpected trailing data")}
	}

	version := ""
	if raw.Version != nil {
		version = strings.TrimSpace(*raw.Version)
	}
	if version == "" {
		return nil, &ValidationError{Path: path, Field: "version", Message: "missing"}
	}
	if version != VersionV1 {
		return nil, &ValidationError{Path: path, Field: "version", Message: fmt.Sprintf("unsupported value %q", version)}
	}

	if raw.Resources == nil {
		return nil, &ValidationError{Path: path, Field: "resources", Message: "missing"}
	}

	resources := make([]Resource, 0, len(*raw.Resources))
	for i, resource := range *raw.Resources {
		fieldPrefix := fmt.Sprintf("resources[%d]", i)

		if resource.Kind == nil || strings.TrimSpace(*resource.Kind) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".kind", Message: "missing"}
		}
		if resource.Namespace == nil || strings.TrimSpace(*resource.Namespace) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".namespace", Message: "missing"}
		}
		if resource.Name == nil || strings.TrimSpace(*resource.Name) == "" {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".name", Message: "missing"}
		}
		if resource.Replicas == nil {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".replicas", Message: "missing"}
		}
		if *resource.Replicas < 0 {
			return nil, &ValidationError{Path: path, Field: fieldPrefix + ".replicas", Message: "must be >= 0"}
		}

		resources = append(resources, Resource{
			Kind:      *resource.Kind,
			Namespace: *resource.Namespace,
			Name:      *resource.Name,
			Replicas:  *resource.Replicas,
		})
	}

	parsed := &State{
		Version:   version,
		Resources: resources,
	}
	if raw.Node != nil {
		parsed.Node = *raw.Node
	}
	if raw.Timestamp != nil && strings.TrimSpace(*raw.Timestamp) != "" {
		parsedTimestamp, err := time.Parse(time.RFC3339Nano, *raw.Timestamp)
		if err != nil {
			return nil, &ValidationError{Path: path, Field: "timestamp", Message: "invalid RFC3339 timestamp"}
		}
		parsed.Timestamp = parsedTimestamp
	}
	if raw.OperatorReplicas != nil {
		if *raw.OperatorReplicas < 0 {
			return nil, &ValidationError{Path: path, Field: "operatorReplicas", Message: "must be >= 0"}
		}
		parsed.OperatorReplicas = *raw.OperatorReplicas
	} else {
		parsed.OperatorReplicas = 1
	}

	return parsed, nil
}

func writeFileAtomic(path string, payload []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create state directory %s: %w", dir, err)
		}
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.")
	if err != nil {
		return fmt.Errorf("create temp state file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		logger.Error("failed to write state file", "path", path, "temp_path", tmpName, "error", err)
		return fmt.Errorf("write state file %s: %w", path, err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		logger.Error("failed to sync state file", "path", path, "temp_path", tmpName, "error", err)
		return fmt.Errorf("sync state file %s: %w", path, err)
	}
	if err := tmpFile.Close(); err != nil {
		logger.Error("failed to close state temp file", "path", path, "temp_path", tmpName, "error", err)
		return fmt.Errorf("close state temp file %s: %w", path, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		logger.Error("failed to set state file permissions", "path", path, "temp_path", tmpName, "error", err)
		return fmt.Errorf("chmod state file %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		logger.Error("failed to replace state file", "path", path, "temp_path", tmpName, "error", err)
		return fmt.Errorf("replace state file %s: %w", path, err)
	}

	return nil
}
