package state

import (
	"sort"
	"time"
)

const (
	// VersionV1 is the current state file version.
	VersionV1 = "v1"
)

// Resource represents a single tracked Kubernetes resource in the state file.
type Resource struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Replicas  int    `json:"replicas"`
}

// State captures the maintenance state for a node.
type State struct {
	Version          string     `json:"version"`
	Node             string     `json:"node,omitempty"`
	Timestamp        time.Time  `json:"timestamp,omitempty"`
	OperatorReplicas int        `json:"operatorReplicas"`
	Resources        []Resource `json:"resources"`
}

// NewState returns a state instance populated with the current timestamp.
func NewState(node string, operatorReplicas int, resources []Resource) State {
	return State{
		Version:          VersionV1,
		Node:             node,
		Timestamp:        time.Now().UTC(),
		OperatorReplicas: operatorReplicas,
		Resources:        resources,
	}
}

// SortedResources returns a copy of resources sorted by namespace, then name.
func SortedResources(resources []Resource) []Resource {
	sorted := append([]Resource(nil), resources...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Kind < sorted[j].Kind
	})
	return sorted
}
