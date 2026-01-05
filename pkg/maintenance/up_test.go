package maintenance

import (
	"testing"

	"github.com/andri/crook/pkg/state"
)

func TestSeparateMonDeployments(t *testing.T) {
	t.Parallel()

	resources := []state.Resource{
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-mon-a", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-mon-b", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-osd-0", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-osd-1", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-exporter-worker-01", Replicas: 1},
	}

	monResources, otherResources := separateMonDeployments(resources)

	if len(monResources) != 2 {
		t.Errorf("expected 2 MON resources, got %d", len(monResources))
	}

	if len(otherResources) != 3 {
		t.Errorf("expected 3 other resources, got %d", len(otherResources))
	}

	// Verify MON resources
	for _, r := range monResources {
		if !startsWithPrefix(r.Name, "rook-ceph-mon") {
			t.Errorf("expected MON resource, got %s", r.Name)
		}
	}

	// Verify other resources don't include MONs
	for _, r := range otherResources {
		if startsWithPrefix(r.Name, "rook-ceph-mon") {
			t.Errorf("unexpected MON resource in others: %s", r.Name)
		}
	}
}

func TestSeparateMonDeployments_EmptyInput(t *testing.T) {
	t.Parallel()

	resources := []state.Resource{}
	monResources, otherResources := separateMonDeployments(resources)

	if len(monResources) != 0 {
		t.Errorf("expected 0 MON resources, got %d", len(monResources))
	}
	if len(otherResources) != 0 {
		t.Errorf("expected 0 other resources, got %d", len(otherResources))
	}
}

func TestSeparateMonDeployments_NoMons(t *testing.T) {
	t.Parallel()

	resources := []state.Resource{
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-osd-0", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-exporter-worker-01", Replicas: 1},
	}

	monResources, otherResources := separateMonDeployments(resources)

	if len(monResources) != 0 {
		t.Errorf("expected 0 MON resources, got %d", len(monResources))
	}
	if len(otherResources) != 2 {
		t.Errorf("expected 2 other resources, got %d", len(otherResources))
	}
}

func TestSeparateMonDeployments_OnlyMons(t *testing.T) {
	t.Parallel()

	resources := []state.Resource{
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-mon-a", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-mon-b", Replicas: 1},
	}

	monResources, otherResources := separateMonDeployments(resources)

	if len(monResources) != 2 {
		t.Errorf("expected 2 MON resources, got %d", len(monResources))
	}
	if len(otherResources) != 0 {
		t.Errorf("expected 0 other resources, got %d", len(otherResources))
	}
}

func TestOrderResourcesForUp_ExcludesMonitors(t *testing.T) {
	t.Parallel()

	// orderResourcesForUp should properly order non-MON resources
	// (MONs are handled separately now)
	resources := []state.Resource{
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-crashcollector-worker-01", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-osd-0", Replicas: 1},
		{Kind: "Deployment", Namespace: "rook-ceph", Name: "rook-ceph-exporter-worker-01", Replicas: 1},
	}

	ordered := orderResourcesForUp(resources)

	if len(ordered) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(ordered))
	}

	// Expected order: osd, exporter, crashcollector (mon is first but not in this list)
	expectedOrder := []string{
		"rook-ceph-osd-0",
		"rook-ceph-exporter-worker-01",
		"rook-ceph-crashcollector-worker-01",
	}

	for i, expected := range expectedOrder {
		if ordered[i].Name != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, ordered[i].Name)
		}
	}
}
