package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCordonNode(t *testing.T) {
	ctx := context.Background()

	// Create a fake node that is not cordoned
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	// Cordon the node
	err := client.CordonNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to cordon node: %v", err)
	}

	// Verify the node was cordoned
	updated, err := clientset.CoreV1().Nodes().Get(ctx, "test-node", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if !updated.Spec.Unschedulable {
		t.Error("expected node to be cordoned (unschedulable=true)")
	}
}

func TestCordonNode_AlreadyCordoned(t *testing.T) {
	ctx := context.Background()

	// Create a fake node that is already cordoned
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	// Cordoning again should succeed (idempotent)
	err := client.CordonNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to cordon already-cordoned node: %v", err)
	}

	// Verify still cordoned
	updated, err := clientset.CoreV1().Nodes().Get(ctx, "test-node", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if !updated.Spec.Unschedulable {
		t.Error("expected node to remain cordoned")
	}
}

func TestCordonNode_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	err := client.CordonNode(ctx, "nonexistent-node")
	if err == nil {
		t.Error("expected error when cordoning nonexistent node, got nil")
	}
}

func TestUncordonNode(t *testing.T) {
	ctx := context.Background()

	// Create a fake node that is cordoned
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	// Uncordon the node
	err := client.UncordonNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to uncordon node: %v", err)
	}

	// Verify the node was uncordoned
	updated, err := clientset.CoreV1().Nodes().Get(ctx, "test-node", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if updated.Spec.Unschedulable {
		t.Error("expected node to be uncordoned (unschedulable=false)")
	}
}

func TestUncordonNode_AlreadyUncordoned(t *testing.T) {
	ctx := context.Background()

	// Create a fake node that is not cordoned
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	// Uncordoning again should succeed (idempotent)
	err := client.UncordonNode(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to uncordon already-uncordoned node: %v", err)
	}

	// Verify still uncordoned
	updated, err := clientset.CoreV1().Nodes().Get(ctx, "test-node", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if updated.Spec.Unschedulable {
		t.Error("expected node to remain uncordoned")
	}
}

func TestUncordonNode_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	err := client.UncordonNode(ctx, "nonexistent-node")
	if err == nil {
		t.Error("expected error when uncordoning nonexistent node, got nil")
	}
}

func TestGetNodeStatus(t *testing.T) {
	ctx := context.Background()

	// Create a fake node with ready condition
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: true,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	status, err := client.GetNodeStatus(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to get node status: %v", err)
	}

	if status.Name != "test-node" {
		t.Errorf("expected name 'test-node', got %s", status.Name)
	}
	if !status.Unschedulable {
		t.Error("expected unschedulable to be true")
	}
	if !status.Ready {
		t.Error("expected ready to be true")
	}
	if len(status.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(status.Conditions))
	}
}

func TestGetNodeStatus_NotReady(t *testing.T) {
	ctx := context.Background()

	// Create a fake node with not ready condition
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Unschedulable: false,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	clientset := fake.NewClientset(node)
	client := newClientFromInterface(clientset)

	status, err := client.GetNodeStatus(ctx, "test-node")
	if err != nil {
		t.Fatalf("failed to get node status: %v", err)
	}

	if status.Ready {
		t.Error("expected ready to be false")
	}
}

func TestGetNodeStatus_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	_, err := client.GetNodeStatus(ctx, "nonexistent-node")
	if err == nil {
		t.Error("expected error when getting status of nonexistent node, got nil")
	}
}

func TestListNodes(t *testing.T) {
	ctx := context.Background()

	// Create fake nodes
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-2",
		},
	}

	clientset := fake.NewClientset(node1, node2)
	client := newClientFromInterface(clientset)

	nodes, err := client.ListNodes(ctx)
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}

	// Verify node names
	names := make(map[string]bool)
	for _, node := range nodes {
		names[node.Name] = true
	}

	if !names["node-1"] || !names["node-2"] {
		t.Errorf("expected node-1 and node-2, got %v", names)
	}
}
