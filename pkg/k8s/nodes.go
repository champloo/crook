package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// CordonNode is a package-level function that uses the global client
func CordonNode(ctx context.Context, nodeName string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.CordonNode(ctx, nodeName)
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

// UncordonNode is a package-level function that uses the global client
func UncordonNode(ctx context.Context, nodeName string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.UncordonNode(ctx, nodeName)
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

// GetNodeStatus is a package-level function that uses the global client
func GetNodeStatus(ctx context.Context, nodeName string) (*NodeStatus, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetNodeStatus(ctx, nodeName)
}

// GetNode returns a node by name
func (c *Client) GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	node, err := c.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}
	return node, nil
}

// GetNode is a package-level function that uses the global client
func GetNode(ctx context.Context, nodeName string) (*corev1.Node, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetNode(ctx, nodeName)
}

// ListNodes returns all nodes in the cluster
func (c *Client) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	return nodeList.Items, nil
}

// ListNodes is a package-level function that uses the global client
func ListNodes(ctx context.Context) ([]corev1.Node, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListNodes(ctx)
}

// NodeInfoForLS holds node information for the ls command view
type NodeInfoForLS struct {
	// Name is the node name
	Name string

	// Status is the node status (Ready/NotReady/Unknown)
	Status string

	// Roles are the node roles (control-plane, worker, etc.)
	Roles []string

	// Schedulable indicates if the node accepts new pods
	Schedulable bool

	// Cordoned indicates if the node is cordoned (unschedulable)
	Cordoned bool

	// CephPodCount is the number of Ceph pods on this node
	CephPodCount int

	// Age is the time since the node was created
	Age time.Duration

	// KubeletVersion is the kubelet version
	KubeletVersion string
}

// ListNodesWithCephPods returns all nodes with Ceph pod counts
func (c *Client) ListNodesWithCephPods(ctx context.Context, namespace string, prefixes []string) ([]NodeInfoForLS, error) {
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
	result := make([]NodeInfoForLS, 0, len(nodes))
	now := time.Now()

	for _, node := range nodes {
		info := NodeInfoForLS{
			Name:           node.Name,
			Status:         getNodeStatus(&node),
			Roles:          extractNodeRoles(&node),
			Schedulable:    !node.Spec.Unschedulable,
			Cordoned:       node.Spec.Unschedulable,
			CephPodCount:   nodePodCounts[node.Name],
			Age:            now.Sub(node.CreationTimestamp.Time),
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		}
		result = append(result, info)
	}

	return result, nil
}

// ListNodesWithCephPods is a package-level function that uses the global client
func ListNodesWithCephPods(ctx context.Context, namespace string, prefixes []string) ([]NodeInfoForLS, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListNodesWithCephPods(ctx, namespace, prefixes)
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
	roles := make([]string, 0)
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

// NodeExists is a package-level function that uses the global client
func NodeExists(ctx context.Context, nodeName string) (bool, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return false, err
	}
	return client.NodeExists(ctx, nodeName)
}
