package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
)

func TestListPodsOnNode(t *testing.T) {
	ctx := context.Background()

	// Create fake pods on different nodes
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
	}
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-3",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node-2",
		},
	}

	clientset := fake.NewClientset(pod1, pod2, pod3)
	client := newClientFromInterface(clientset)

	// Test cluster-wide listing (empty namespace)
	pods, err := client.ListPodsOnNode(ctx, "node-1", "")
	if err != nil {
		t.Fatalf("failed to list pods on node (cluster-wide): %v", err)
	}

	// Note: fake clientset doesn't support field selectors, so it returns all pods
	// In real usage, the field selector would filter by node
	// For this test, we just verify the API call succeeds
	if len(pods) == 0 {
		t.Error("expected at least some pods (cluster-wide)")
	}

	// Test namespace-scoped listing
	podsInNs, err := client.ListPodsOnNode(ctx, "node-1", "default")
	if err != nil {
		t.Fatalf("failed to list pods on node (namespace-scoped): %v", err)
	}

	if len(podsInNs) == 0 {
		t.Error("expected at least some pods in default namespace")
	}
}

func TestListPodsOnNode_NoPods(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	// Test with empty namespace (cluster-wide)
	pods, err := client.ListPodsOnNode(ctx, "nonexistent-node", "")
	if err != nil {
		t.Fatalf("failed to list pods (cluster-wide): %v", err)
	}

	if len(pods) != 0 {
		t.Errorf("expected 0 pods on nonexistent node (cluster-wide), got %d", len(pods))
	}

	// Test with specific namespace
	podsInNs, err := client.ListPodsOnNode(ctx, "nonexistent-node", "default")
	if err != nil {
		t.Fatalf("failed to list pods (namespace-scoped): %v", err)
	}

	if len(podsInNs) != 0 {
		t.Errorf("expected 0 pods on nonexistent node (namespace-scoped), got %d", len(podsInNs))
	}
}

func TestGetOwnerChain_PodWithoutOwner(t *testing.T) {
	ctx := context.Background()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-pod",
			Namespace: "default",
			UID:       "pod-uid",
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	chain, err := client.GetOwnerChain(ctx, pod)
	if err != nil {
		t.Fatalf("failed to get owner chain: %v", err)
	}

	if chain.Pod.Name != "standalone-pod" {
		t.Errorf("expected pod name 'standalone-pod', got %s", chain.Pod.Name)
	}
	if chain.ReplicaSet != nil {
		t.Error("expected no ReplicaSet owner")
	}
	if chain.Deployment != nil {
		t.Error("expected no Deployment owner")
	}
}

func TestGetOwnerChain_PodOwnedByReplicaSet(t *testing.T) {
	ctx := context.Background()

	// Create a deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			UID:       "deployment-uid",
		},
	}

	// Create a ReplicaSet owned by the deployment
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rs",
			Namespace: "default",
			UID:       "rs-uid",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "test-deployment",
					UID:        "deployment-uid",
					Controller: ptr.To(true),
				},
			},
		},
	}

	// Create a pod owned by the ReplicaSet
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "pod-uid",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       "test-rs",
					UID:        "rs-uid",
					Controller: ptr.To(true),
				},
			},
		},
	}

	clientset := fake.NewClientset(pod, replicaSet, deployment)
	client := newClientFromInterface(clientset)

	chain, err := client.GetOwnerChain(ctx, pod)
	if err != nil {
		t.Fatalf("failed to get owner chain: %v", err)
	}

	if chain.Pod.Name != "test-pod" {
		t.Errorf("expected pod name 'test-pod', got %s", chain.Pod.Name)
	}
	if chain.ReplicaSet == nil {
		t.Fatal("expected ReplicaSet owner")
	}
	if chain.ReplicaSet.Name != "test-rs" {
		t.Errorf("expected ReplicaSet name 'test-rs', got %s", chain.ReplicaSet.Name)
	}
	if chain.Deployment == nil {
		t.Fatal("expected Deployment owner")
	}
	if chain.Deployment.Name != "test-deployment" {
		t.Errorf("expected Deployment name 'test-deployment', got %s", chain.Deployment.Name)
	}
}

func TestGetOwnerChain_PodOwnedByStatefulSet(t *testing.T) {
	ctx := context.Background()

	// Create a pod owned by a StatefulSet
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "pod-uid",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "StatefulSet",
					Name:       "test-statefulset",
					UID:        "statefulset-uid",
					Controller: ptr.To(true),
				},
			},
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	chain, err := client.GetOwnerChain(ctx, pod)
	if err != nil {
		t.Fatalf("failed to get owner chain: %v", err)
	}

	if chain.StatefulSet == nil {
		t.Fatal("expected StatefulSet owner")
	}
	if chain.StatefulSet.Name != "test-statefulset" {
		t.Errorf("expected StatefulSet name 'test-statefulset', got %s", chain.StatefulSet.Name)
	}
}

func TestGetOwnerChain_PodOwnedByDaemonSet(t *testing.T) {
	ctx := context.Background()

	// Create a pod owned by a DaemonSet
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "pod-uid",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
					Name:       "test-daemonset",
					UID:        "daemonset-uid",
					Controller: ptr.To(true),
				},
			},
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	chain, err := client.GetOwnerChain(ctx, pod)
	if err != nil {
		t.Fatalf("failed to get owner chain: %v", err)
	}

	if chain.DaemonSet == nil {
		t.Fatal("expected DaemonSet owner")
	}
	if chain.DaemonSet.Name != "test-daemonset" {
		t.Errorf("expected DaemonSet name 'test-daemonset', got %s", chain.DaemonSet.Name)
	}
}

func TestListPodsInNamespace(t *testing.T) {
	ctx := context.Background()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "default",
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "default",
		},
	}
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-3",
			Namespace: "other",
		},
	}

	clientset := fake.NewClientset(pod1, pod2, pod3)
	client := newClientFromInterface(clientset)

	pods, err := client.ListPodsInNamespace(ctx, "default")
	if err != nil {
		t.Fatalf("failed to list pods: %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("expected 2 pods in default namespace, got %d", len(pods))
	}

	// Verify pod names
	names := make(map[string]bool)
	for _, pod := range pods {
		names[pod.Name] = true
	}

	if !names["pod-1"] || !names["pod-2"] {
		t.Errorf("expected pod-1 and pod-2, got %v", names)
	}
}

func TestGetPod(t *testing.T) {
	ctx := context.Background()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	retrieved, err := client.GetPod(ctx, "default", "test-pod")
	if err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}

	if retrieved.Name != "test-pod" {
		t.Errorf("expected pod name 'test-pod', got %s", retrieved.Name)
	}
}

func TestGetPod_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	_, err := client.GetPod(ctx, "default", "nonexistent-pod")
	if err == nil {
		t.Error("expected error when getting nonexistent pod, got nil")
	}
}

func TestExtractPodType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// Basic type extraction
		{"rook-ceph-osd-0-xyz", "osd"},
		{"rook-ceph-osd-10-abc", "osd"},
		{"rook-ceph-mon-a-xyz", "mon"},
		{"rook-ceph-mon-b-abc", "mon"},
		{"rook-ceph-mgr-a-xyz", "mgr"},
		{"rook-ceph-mds-a-xyz", "mds"},
		{"rook-ceph-rgw-a-xyz", "rgw"},
		{"rook-ceph-crashcollector-worker-1-xyz", "crashcollector"},
		{"rook-ceph-exporter-worker-1-xyz", "exporter"},
		{"rook-ceph-tools-xyz", "tools"},
		{"rook-ceph-operator-xyz", "operator"},
		{"rook-ceph-cleanup-xyz", "cleanup"},
		{"csi-cephfsplugin-xyz", "csi"},
		{"csi-rbdplugin-xyz", "csi"},
		{"unknown-pod", "other"},
		{"", "other"},

		// Overlapping prefix tests - "prepare" must match before "osd"
		// This is the key test case for the determinism fix
		{"rook-ceph-osd-prepare-worker-01-xyz", "prepare"},
		{"rook-ceph-osd-prepare-node-abc-123", "prepare"},
		{"rook-ceph-osd-prepare-rook", "prepare"},
		{"rook-ceph-osd-prepare-rook-m02", "prepare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractPodType(tt.name); got != tt.want {
				t.Errorf("extractPodType(%q) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtractPodType_Deterministic(t *testing.T) {
	// Run extractPodType many times on overlapping prefix names
	// to verify deterministic behavior (the original map-based implementation
	// would produce random results due to Go's map iteration order)
	testCases := []struct {
		name string
		want string
	}{
		{"rook-ceph-osd-prepare-worker-01", "prepare"},
		{"rook-ceph-osd-0", "osd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run 100 times to catch non-determinism
			for i := 0; i < 100; i++ {
				got := extractPodType(tc.name)
				if got != tc.want {
					t.Errorf("iteration %d: extractPodType(%q) = %s, want %s",
						i, tc.name, got, tc.want)
				}
			}
		})
	}
}
