package k8s

import (
	"context"
	"fmt"

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
