package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteFileDeterministic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	state := State{
		Version:          VersionV1,
		Node:             "worker-01",
		Timestamp:        time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		OperatorReplicas: 1,
		Resources: []Resource{
			{Kind: "Deployment", Namespace: "rook-ceph", Name: "b", Replicas: 1},
			{Kind: "Deployment", Namespace: "rook-ceph", Name: "a", Replicas: 2},
		},
	}

	if err := WriteFile(path, state); err != nil {
		t.Fatalf("write state: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}

	if err := WriteFile(path, state); err != nil {
		t.Fatalf("write state again: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state again: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("expected deterministic output")
	}

	if !strings.HasSuffix(string(first), "\n") {
		t.Fatalf("expected trailing newline")
	}
}

func TestParseDefaultsOperatorReplicas(t *testing.T) {
	t.Parallel()

	data := []byte(`{
  "version": "v1",
  "node": "worker-01",
  "timestamp": "2024-01-01T12:00:00Z",
  "resources": [
    {
      "kind": "Deployment",
      "namespace": "rook-ceph",
      "name": "rook-ceph-osd-0",
      "replicas": 1
    }
  ]
}`)

	state, err := Parse(data)
	if err != nil {
		t.Fatalf("parse state: %v", err)
	}

	if state.OperatorReplicas != 1 {
		t.Fatalf("expected operator replicas default 1, got %d", state.OperatorReplicas)
	}
}

func TestParseValidationErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload string
	}{
		{
			name:    "missing version",
			payload: `{"resources":[]}`,
		},
		{
			name:    "invalid version",
			payload: `{"version":"v2","resources":[]}`,
		},
		{
			name:    "missing resources",
			payload: `{"version":"v1"}`,
		},
		{
			name:    "negative replicas",
			payload: `{"version":"v1","resources":[{"kind":"Deployment","namespace":"rook","name":"a","replicas":-1}]}`,
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse([]byte(testCase.payload)); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
