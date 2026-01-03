package maintenance

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestDiscoverDeployments_MultipleMatchingDeployments(t *testing.T) {
	ctx := context.Background()

	// Create test data with matching deployments
	osdDeployment := createDeployment("rook-ceph", "rook-ceph-osd-0")
	monDeployment := createDeployment("rook-ceph", "rook-ceph-mon-a")

	osdRS := createReplicaSet("rook-ceph", "rook-ceph-osd-0-abc123", "rook-ceph-osd-0")
	monRS := createReplicaSet("rook-ceph", "rook-ceph-mon-a-def456", "rook-ceph-mon-a")

	osdPod := createPodWithOwner("rook-ceph", "rook-ceph-osd-0-abc123-xyz", "worker-01", "ReplicaSet", "rook-ceph-osd-0-abc123")
	monPod := createPodWithOwner("rook-ceph", "rook-ceph-mon-a-def456-xyz", "worker-01", "ReplicaSet", "rook-ceph-mon-a-def456")

	client := createTestClient(osdDeployment, monDeployment, osdRS, monRS, osdPod, monPod)

	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon"}
	deployments, err := DiscoverDeployments(ctx, client, "worker-01", "rook-ceph", prefixes)
	if err != nil {
		t.Fatalf("DiscoverDeployments failed: %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("Expected 2 deployments, got %d", len(deployments))
	}

	// Verify deployment names
	names := GetDeploymentNames(deployments)
	if !containsDeployment(names, "rook-ceph/rook-ceph-osd-0") {
		t.Error("Expected to find rook-ceph-osd-0 deployment")
	}
	if !containsDeployment(names, "rook-ceph/rook-ceph-mon-a") {
		t.Error("Expected to find rook-ceph-mon-a deployment")
	}
}

func TestDiscoverDeployments_NoMatchingPrefix(t *testing.T) {
	ctx := context.Background()

	// Create test data with non-matching deployment
	nginxDeployment := createDeployment("default", "nginx")
	nginxRS := createReplicaSet("default", "nginx-abc123", "nginx")
	nginxPod := createPodWithOwner("default", "nginx-abc123-xyz", "worker-01", "ReplicaSet", "nginx-abc123")

	client := createTestClient(nginxDeployment, nginxRS, nginxPod)

	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon"}
	deployments, err := DiscoverDeployments(ctx, client, "worker-01", "", prefixes)
	if err != nil {
		t.Fatalf("DiscoverDeployments failed: %v", err)
	}

	if len(deployments) != 0 {
		t.Errorf("Expected 0 deployments, got %d", len(deployments))
	}
}

func TestDiscoverDeployments_PodWithoutOwner(t *testing.T) {
	ctx := context.Background()

	// Create pod without owner
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "worker-01",
		},
	}

	client := createTestClient(pod)

	prefixes := []string{"rook-ceph-osd"}
	deployments, err := DiscoverDeployments(ctx, client, "worker-01", "", prefixes)
	if err != nil {
		t.Fatalf("DiscoverDeployments failed: %v", err)
	}

	if len(deployments) != 0 {
		t.Errorf("Expected 0 deployments for pod without owner, got %d", len(deployments))
	}
}

func TestDiscoverDeployments_DuplicateDeployments(t *testing.T) {
	ctx := context.Background()

	// Create deployment with multiple pods (should return unique deployment)
	osdDeployment := createDeployment("rook-ceph", "rook-ceph-osd-0")
	osdRS := createReplicaSet("rook-ceph", "rook-ceph-osd-0-abc123", "rook-ceph-osd-0")

	osdPod1 := createPodWithOwner("rook-ceph", "rook-ceph-osd-0-abc123-xyz1", "worker-01", "ReplicaSet", "rook-ceph-osd-0-abc123")
	osdPod2 := createPodWithOwner("rook-ceph", "rook-ceph-osd-0-abc123-xyz2", "worker-01", "ReplicaSet", "rook-ceph-osd-0-abc123")

	client := createTestClient(osdDeployment, osdRS, osdPod1, osdPod2)

	prefixes := []string{"rook-ceph-osd"}
	deployments, err := DiscoverDeployments(ctx, client, "worker-01", "rook-ceph", prefixes)
	if err != nil {
		t.Fatalf("DiscoverDeployments failed: %v", err)
	}

	if len(deployments) != 1 {
		t.Errorf("Expected 1 unique deployment, got %d", len(deployments))
	}
}

func TestDiscoverDeployments_FilterByNamespace(t *testing.T) {
	ctx := context.Background()

	// Create deployments in different namespaces
	osdDeployment1 := createDeployment("rook-ceph", "rook-ceph-osd-0")
	osdDeployment2 := createDeployment("other-ns", "rook-ceph-osd-0")

	osdRS1 := createReplicaSet("rook-ceph", "rook-ceph-osd-0-abc123", "rook-ceph-osd-0")
	osdRS2 := createReplicaSet("other-ns", "rook-ceph-osd-0-def456", "rook-ceph-osd-0")

	osdPod1 := createPodWithOwner("rook-ceph", "rook-ceph-osd-0-abc123-xyz", "worker-01", "ReplicaSet", "rook-ceph-osd-0-abc123")
	osdPod2 := createPodWithOwner("other-ns", "rook-ceph-osd-0-def456-xyz", "worker-01", "ReplicaSet", "rook-ceph-osd-0-def456")

	client := createTestClient(osdDeployment1, osdDeployment2, osdRS1, osdRS2, osdPod1, osdPod2)

	prefixes := []string{"rook-ceph-osd"}
	deployments, err := DiscoverDeployments(ctx, client, "worker-01", "rook-ceph", prefixes)
	if err != nil {
		t.Fatalf("DiscoverDeployments failed: %v", err)
	}

	if len(deployments) != 1 {
		t.Errorf("Expected 1 deployment in rook-ceph namespace, got %d", len(deployments))
	}

	if deployments[0].Namespace != "rook-ceph" {
		t.Errorf("Expected deployment in rook-ceph namespace, got %s", deployments[0].Namespace)
	}
}

func TestMatchesPrefix_WithPrefixes(t *testing.T) {
	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon"}

	tests := []struct {
		name     string
		expected bool
	}{
		{"rook-ceph-osd-0", true},
		{"rook-ceph-mon-a", true},
		{"rook-ceph-exporter", false},
		{"nginx", false},
	}

	for _, tt := range tests {
		result := matchesPrefix(tt.name, prefixes)
		if result != tt.expected {
			t.Errorf("matchesPrefix(%s) = %v, expected %v", tt.name, result, tt.expected)
		}
	}
}

func TestMatchesPrefix_NoPrefixes(t *testing.T) {
	// When no prefixes specified, should match all
	result := matchesPrefix("any-deployment", []string{})
	if !result {
		t.Error("Expected matchesPrefix to return true when no prefixes specified")
	}
}

func TestOrderDeploymentsForDown(t *testing.T) {
	deployments := []appsv1.Deployment{
		*createDeployment("rook-ceph", "rook-ceph-crashcollector"),
		*createDeployment("rook-ceph", "rook-ceph-mon-a"),
		*createDeployment("rook-ceph", "rook-ceph-exporter"),
		*createDeployment("rook-ceph", "rook-ceph-osd-0"),
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
		*createDeployment("rook-ceph", "rook-ceph-osd-0"),
		*createDeployment("rook-ceph", "rook-ceph-crashcollector"),
		*createDeployment("rook-ceph", "rook-ceph-exporter"),
		*createDeployment("rook-ceph", "rook-ceph-mon-a"),
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

func TestGroupDeploymentsByPrefix(t *testing.T) {
	deployments := []appsv1.Deployment{
		*createDeployment("rook-ceph", "rook-ceph-osd-0"),
		*createDeployment("rook-ceph", "rook-ceph-osd-1"),
		*createDeployment("rook-ceph", "rook-ceph-mon-a"),
		*createDeployment("rook-ceph", "rook-ceph-exporter"),
	}

	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon", "rook-ceph-exporter"}
	grouped := GroupDeploymentsByPrefix(deployments, prefixes)

	if len(grouped["rook-ceph-osd"]) != 2 {
		t.Errorf("Expected 2 OSD deployments, got %d", len(grouped["rook-ceph-osd"]))
	}

	if len(grouped["rook-ceph-mon"]) != 1 {
		t.Errorf("Expected 1 MON deployment, got %d", len(grouped["rook-ceph-mon"]))
	}

	if len(grouped["rook-ceph-exporter"]) != 1 {
		t.Errorf("Expected 1 exporter deployment, got %d", len(grouped["rook-ceph-exporter"]))
	}
}

func TestGetDeploymentNames(t *testing.T) {
	deployments := []appsv1.Deployment{
		*createDeployment("ns1", "deploy1"),
		*createDeployment("ns2", "deploy2"),
	}

	names := GetDeploymentNames(deployments)

	if len(names) != 2 {
		t.Fatalf("Expected 2 names, got %d", len(names))
	}

	if names[0] != "ns1/deploy1" {
		t.Errorf("Expected ns1/deploy1, got %s", names[0])
	}

	if names[1] != "ns2/deploy2" {
		t.Errorf("Expected ns2/deploy2, got %s", names[1])
	}
}

// Helper functions

func createDeployment(namespace, name string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("deployment-" + name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func createReplicaSet(namespace, name, deploymentName string) *appsv1.ReplicaSet {
	controller := true
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("rs-" + name),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       deploymentName,
					UID:        types.UID("deployment-" + deploymentName),
					Controller: &controller,
				},
			},
		},
	}
}

func createPodWithOwner(namespace, name, nodeName, ownerKind, ownerName string) *corev1.Pod {
	controller := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("pod-" + name),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       ownerKind,
					Name:       ownerName,
					UID:        types.UID("rs-" + ownerName),
					Controller: &controller,
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func containsDeployment(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
