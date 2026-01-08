package maintenance

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestOrderDeploymentsForDown(t *testing.T) {
	deployments := []appsv1.Deployment{
		*createDeployment("rook-ceph-crashcollector"),
		*createDeployment("rook-ceph-mon-a"),
		*createDeployment("rook-ceph-exporter"),
		*createDeployment("rook-ceph-osd-0"),
	}

	ordered := OrderDeploymentsForDown(deployments)

	// Verify order: OSD first, then MON, then exporter, then crashcollector
	expectedOrder := []string{
		"rook-ceph-osd-0",
		"rook-ceph-mon-a",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	if len(ordered) != len(expectedOrder) {
		t.Fatalf("Expected %d deployments, got %d", len(expectedOrder), len(ordered))
	}

	for i, expected := range expectedOrder {
		if ordered[i].Name != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, ordered[i].Name)
		}
	}
}

func TestOrderDeploymentsForUp(t *testing.T) {
	deployments := []appsv1.Deployment{
		*createDeployment("rook-ceph-osd-0"),
		*createDeployment("rook-ceph-crashcollector"),
		*createDeployment("rook-ceph-exporter"),
		*createDeployment("rook-ceph-mon-a"),
	}

	ordered := OrderDeploymentsForUp(deployments)

	// Verify order: MON first, then OSD, then exporter, then crashcollector
	expectedOrder := []string{
		"rook-ceph-mon-a",
		"rook-ceph-osd-0",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	if len(ordered) != len(expectedOrder) {
		t.Fatalf("Expected %d deployments, got %d", len(expectedOrder), len(ordered))
	}

	for i, expected := range expectedOrder {
		if ordered[i].Name != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, ordered[i].Name)
		}
	}
}

func TestOrderDeploymentsForDown_UnrecognizedPrefix(t *testing.T) {
	// Test that deployments without recognized prefix are appended at the end
	deployments := []appsv1.Deployment{
		*createDeployment("unknown-deployment"),
		*createDeployment("rook-ceph-osd-0"),
	}

	ordered := OrderDeploymentsForDown(deployments)

	if len(ordered) != 2 {
		t.Fatalf("Expected 2 deployments, got %d", len(ordered))
	}

	// OSD should be first, unknown should be last
	if ordered[0].Name != "rook-ceph-osd-0" {
		t.Errorf("Expected rook-ceph-osd-0 first, got %s", ordered[0].Name)
	}
	if ordered[1].Name != "unknown-deployment" {
		t.Errorf("Expected unknown-deployment last, got %s", ordered[1].Name)
	}
}

func TestOrderDeploymentsForUp_UnrecognizedPrefix(t *testing.T) {
	// Test that deployments without recognized prefix are appended at the end
	deployments := []appsv1.Deployment{
		*createDeployment("unknown-deployment"),
		*createDeployment("rook-ceph-mon-a"),
	}

	ordered := OrderDeploymentsForUp(deployments)

	if len(ordered) != 2 {
		t.Fatalf("Expected 2 deployments, got %d", len(ordered))
	}

	// MON should be first, unknown should be last
	if ordered[0].Name != "rook-ceph-mon-a" {
		t.Errorf("Expected rook-ceph-mon-a first, got %s", ordered[0].Name)
	}
	if ordered[1].Name != "unknown-deployment" {
		t.Errorf("Expected unknown-deployment last, got %s", ordered[1].Name)
	}
}

func TestOrderDeploymentsForDown_EmptyInput(t *testing.T) {
	var deployments []appsv1.Deployment
	ordered := OrderDeploymentsForDown(deployments)

	if len(ordered) != 0 {
		t.Errorf("Expected empty result for empty input, got %d deployments", len(ordered))
	}
}

func TestOrderDeploymentsForUp_EmptyInput(t *testing.T) {
	var deployments []appsv1.Deployment
	ordered := OrderDeploymentsForUp(deployments)

	if len(ordered) != 0 {
		t.Errorf("Expected empty result for empty input, got %d deployments", len(ordered))
	}
}

// Helper functions

func createDeployment(name string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "rook-ceph",
			UID:       types.UID("deployment-" + name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}
