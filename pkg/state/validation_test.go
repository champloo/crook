package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateFileNotFound(t *testing.T) {
	t.Parallel()

	_, _, err := ValidateFile(filepath.Join(t.TempDir(), "missing.json"), ValidationOptions{})
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestValidateStateWarningsForAge(t *testing.T) {
	t.Parallel()

	state := &State{
		Version:   VersionV1,
		Node:      "worker-01",
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Resources: []Resource{{Kind: "Deployment", Namespace: "rook-ceph", Name: "osd", Replicas: 1}},
	}

	now := time.Date(2024, 1, 3, 1, 0, 0, 0, time.UTC)
	warnings, err := ValidateState(state, "", ValidationOptions{
		MaxAge: 24 * time.Hour,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("validate state: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected warning, got %d", len(warnings))
	}
	if !warnings[0].RequiresConfirmation {
		t.Fatalf("expected confirmation warning")
	}
}

func TestValidateFileParsesAndValidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	payload := `{
  "version": "v1",
  "node": "worker-01",
  "timestamp": "2024-01-01T00:00:00Z",
  "operatorReplicas": 1,
  "resources": [
    {"kind":"Deployment","namespace":"rook-ceph","name":"osd","replicas":1}
  ]
}`

	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	parsed, warnings, err := ValidateFile(path, ValidationOptions{
		Now: func() time.Time { return time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("validate file: %v", err)
	}
	if parsed == nil {
		t.Fatalf("expected parsed state")
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings")
	}
}

func TestFindMissingResources(t *testing.T) {
	t.Parallel()

	state := &State{
		Version:   VersionV1,
		Node:      "worker-01",
		Resources: []Resource{{Kind: "Deployment", Namespace: "rook-ceph", Name: "osd", Replicas: 1}},
	}

	missing, err := FindMissingResources(context.Background(), state, func(ctx context.Context, resource Resource) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("find missing: %v", err)
	}
	if len(missing) != 1 {
		t.Fatalf("expected missing resource")
	}
}
