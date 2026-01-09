package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
)

// NodeStatus holds the status information for a node
type NodeStatus struct {
	Name          string
	Unschedulable bool
	Ready         bool
	Conditions    []corev1.NodeCondition
}

// CordonNode marks a node as unschedulable
func (c *Client) CordonNode(ctx context.Context, nodeName string) error {
	// Get the node first to verify it exists
	node, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// If already cordoned, nothing to do
	if node.Spec.Unschedulable {
		return nil
	}

	// Patch the node to set unschedulable=true
	patch := []byte(`{"spec":{"unschedulable":true}}`)
	_, err = c.Clientset.CoreV1().Nodes().Patch(
		ctx,
		nodeName,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to cordon node %s: %w", nodeName, err)
	}

	return nil
}

// UncordonNode marks a node as schedulable
func (c *Client) UncordonNode(ctx context.Context, nodeName string) error {
	// Get the node first to verify it exists
	node, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	// If already uncordoned, nothing to do
	if !node.Spec.Unschedulable {
		return nil
	}

	// Patch the node to set unschedulable=false
	patch := []byte(`{"spec":{"unschedulable":false}}`)
	_, err = c.Clientset.CoreV1().Nodes().Patch(
		ctx,
		nodeName,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to uncordon node %s: %w", nodeName, err)
	}

	return nil
}

// GetNodeStatus returns the status of a node
func (c *Client) GetNodeStatus(ctx context.Context, nodeName string) (*NodeStatus, error) {
	node, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}

	status := &NodeStatus{
		Name:          node.Name,
		Unschedulable: node.Spec.Unschedulable,
		Ready:         false,
		Conditions:    node.Status.Conditions,
	}

	// Determine if node is ready
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			status.Ready = condition.Status == corev1.ConditionTrue
			break
		}
	}

	return status, nil
}

// GetNode returns a node by name
func (c *Client) GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	node, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}
	return node, nil
}

// ListNodes returns all nodes in the cluster
func (c *Client) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	return nodeList.Items, nil
}

// NodeInfo holds node information for display and serialization
type NodeInfo struct {
	// Name is the node name
	Name string `json:"name"`

	// Status is the node status (Ready/NotReady/Unknown)
	Status string `json:"status"`

	// Roles are the node roles (control-plane, worker, etc.)
	Roles []string `json:"roles"`

	// Schedulable indicates if the node accepts new pods
	Schedulable bool `json:"schedulable"`

	// Cordoned indicates if the node is cordoned (unschedulable)
	Cordoned bool `json:"cordoned"`

	// CephPodCount is the number of Ceph pods on this node
	CephPodCount int `json:"ceph_pod_count"`

	// Age is the human-readable time since the node was created
	Age string `json:"age"`

	// KubeletVersion is the kubelet version
	KubeletVersion string `json:"kubelet_version"`
}

// ListNodesWithCephPods returns all nodes with Ceph pod counts.
// Uses DefaultRookCephPrefixes() to filter pods.
func (c *Client) ListNodesWithCephPods(ctx context.Context, namespace string) ([]NodeInfo, error) {
	prefixes := DefaultRookCephPrefixes()

	// Get all nodes
	nodes, err := c.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get all pods in the namespace to count per-node Ceph pods
	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	// Build a map of node -> Ceph pod count
	nodePodCounts := make(map[string]int)
	for _, pod := range podList.Items {
		// Check if pod matches any of the prefixes
		if matchesAnyPrefix(pod.Name, prefixes) {
			nodePodCounts[pod.Spec.NodeName]++
		}
	}

	// Build result
	result := make([]NodeInfo, 0, len(nodes))
	now := time.Now()

	for _, node := range nodes {
		info := NodeInfo{
			Name:           node.Name,
			Status:         getNodeStatus(&node),
			Roles:          extractNodeRoles(&node),
			Schedulable:    !node.Spec.Unschedulable,
			Cordoned:       node.Spec.Unschedulable,
			CephPodCount:   nodePodCounts[node.Name],
			Age:            duration.HumanDuration(now.Sub(node.CreationTimestamp.Time)),
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		}
		result = append(result, info)
	}

	return result, nil
}

// getNodeStatus extracts the status string from a node
func getNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// extractNodeRoles extracts roles from node labels
func extractNodeRoles(node *corev1.Node) []string {
	var roles []string
	const rolePrefix = "node-role.kubernetes.io/"

	for label := range node.Labels {
		if strings.HasPrefix(label, rolePrefix) {
			role := strings.TrimPrefix(label, rolePrefix)
			if role != "" {
				roles = append(roles, role)
			}
		}
	}

	return roles
}

// matchesAnyPrefix checks if a string starts with any of the given prefixes
func matchesAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// NodeExists checks if a node exists in the cluster
func (c *Client) NodeExists(ctx context.Context, nodeName string) (bool, error) {
	_, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check node %s: %w", nodeName, err)
	}
	return true, nil
}
