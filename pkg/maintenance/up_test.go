package maintenance

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSeparateMonDeploymentsFromList(t *testing.T) {
	t.Parallel()

	deployments := []appsv1.Deployment{
		makeTestDeployment("rook-ceph-mon-a", "rook-ceph"),
		makeTestDeployment("rook-ceph-mon-b", "rook-ceph"),
		makeTestDeployment("rook-ceph-osd-0", "rook-ceph"),
		makeTestDeployment("rook-ceph-osd-1", "rook-ceph"),
		makeTestDeployment("rook-ceph-exporter-worker-01", "rook-ceph"),
	}

	monDeployments, otherDeployments := separateMonDeploymentsFromList(deployments)

	if len(monDeployments) != 2 {
		t.Errorf("expected 2 MON deployments, got %d", len(monDeployments))
	}

	if len(otherDeployments) != 3 {
		t.Errorf("expected 3 other deployments, got %d", len(otherDeployments))
	}

	// Verify MON deployments
	for _, d := range monDeployments {
		if !startsWithPrefixString(d.Name, "rook-ceph-mon") {
			t.Errorf("expected MON deployment, got %s", d.Name)
		}
	}

	// Verify other deployments don't include MONs
	for _, d := range otherDeployments {
		if startsWithPrefixString(d.Name, "rook-ceph-mon") {
			t.Errorf("unexpected MON deployment in others: %s", d.Name)
		}
	}
}

func TestSeparateMonDeploymentsFromList_EmptyInput(t *testing.T) {
	t.Parallel()

	deployments := []appsv1.Deployment{}
	monDeployments, otherDeployments := separateMonDeploymentsFromList(deployments)

	if len(monDeployments) != 0 {
		t.Errorf("expected 0 MON deployments, got %d", len(monDeployments))
	}
	if len(otherDeployments) != 0 {
		t.Errorf("expected 0 other deployments, got %d", len(otherDeployments))
	}
}

func TestSeparateMonDeploymentsFromList_NoMons(t *testing.T) {
	t.Parallel()

	deployments := []appsv1.Deployment{
		makeTestDeployment("rook-ceph-osd-0", "rook-ceph"),
		makeTestDeployment("rook-ceph-exporter-worker-01", "rook-ceph"),
	}

	monDeployments, otherDeployments := separateMonDeploymentsFromList(deployments)

	if len(monDeployments) != 0 {
		t.Errorf("expected 0 MON deployments, got %d", len(monDeployments))
	}
	if len(otherDeployments) != 2 {
		t.Errorf("expected 2 other deployments, got %d", len(otherDeployments))
	}
}

func TestSeparateMonDeploymentsFromList_OnlyMons(t *testing.T) {
	t.Parallel()

	deployments := []appsv1.Deployment{
		makeTestDeployment("rook-ceph-mon-a", "rook-ceph"),
		makeTestDeployment("rook-ceph-mon-b", "rook-ceph"),
	}

	monDeployments, otherDeployments := separateMonDeploymentsFromList(deployments)

	if len(monDeployments) != 2 {
		t.Errorf("expected 2 MON deployments, got %d", len(monDeployments))
	}
	if len(otherDeployments) != 0 {
		t.Errorf("expected 0 other deployments, got %d", len(otherDeployments))
	}
}

func TestOrderDeploymentsForUp_ExcludesMonitors(t *testing.T) {
	t.Parallel()

	// OrderDeploymentsForUp should properly order non-MON deployments
	// (MONs are handled separately now)
	deployments := []appsv1.Deployment{
		makeTestDeployment("rook-ceph-crashcollector-worker-01", "rook-ceph"),
		makeTestDeployment("rook-ceph-osd-0", "rook-ceph"),
		makeTestDeployment("rook-ceph-exporter-worker-01", "rook-ceph"),
	}

	ordered := OrderDeploymentsForUp(deployments)

	if len(ordered) != 3 {
		t.Fatalf("expected 3 deployments, got %d", len(ordered))
	}

	// Expected order: mon, osd, exporter, crashcollector
	// Since there are no mons, order is: osd, exporter, crashcollector
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

// makeTestDeployment creates a test deployment for testing
func makeTestDeployment(name, _ string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "rook-ceph",
		},
	}
}

// startsWithPrefixString checks if a string starts with a prefix (test helper)
func startsWithPrefixString(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
