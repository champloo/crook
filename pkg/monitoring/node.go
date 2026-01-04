package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/andri/crook/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeStatus represents the monitoring status of a node
type NodeStatus struct {
	Name           string
	Ready          bool
	ReadyStatus    corev1.ConditionStatus // True, False, Unknown
	Unschedulable  bool
	Taints         []corev1.Taint
	KubeletVersion string
	PodCount       int
	Conditions     []corev1.NodeCondition
	LastUpdateTime time.Time
}

// NodeStatusColor returns a color indicator for the node status
func (ns *NodeStatus) NodeStatusColor() string {
	switch ns.ReadyStatus {
	case corev1.ConditionTrue:
		return "green"
	case corev1.ConditionFalse:
		return "red"
	case corev1.ConditionUnknown:
		return "yellow"
	default:
		return "yellow"
	}
}

// MonitorNodeStatus retrieves the current status of a node
func MonitorNodeStatus(ctx context.Context, client *k8s.Client, nodeName string) (*NodeStatus, error) {
	// Get the node
	node, err := client.GetNode(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Get pod count for this node
	podCount, err := getPodCountForNode(ctx, client, nodeName)
	if err != nil {
		// Log error but don't fail - pod count is not critical
		podCount = -1
	}

	// Extract ready condition
	ready := false
	readyStatus := corev1.ConditionUnknown
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			ready = condition.Status == corev1.ConditionTrue
			readyStatus = condition.Status
			break
		}
	}

	status := &NodeStatus{
		Name:           node.Name,
		Ready:          ready,
		ReadyStatus:    readyStatus,
		Unschedulable:  node.Spec.Unschedulable,
		Taints:         node.Spec.Taints,
		KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		PodCount:       podCount,
		Conditions:     node.Status.Conditions,
		LastUpdateTime: time.Now(),
	}

	return status, nil
}

// getPodCountForNode counts the number of pods running on a node
func getPodCountForNode(ctx context.Context, client *k8s.Client, nodeName string) (int, error) {
	// List all pods in all namespaces
	pods, err := client.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list pods: %w", err)
	}

	return len(pods.Items), nil
}

// StartNodeMonitoring starts background monitoring of a node with the given refresh interval
func StartNodeMonitoring(ctx context.Context, client *k8s.Client, nodeName string, interval time.Duration) <-chan *NodeStatus {
	updates := make(chan *NodeStatus, 1)

	go func() {
		defer close(updates)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Send initial status
		if status, err := MonitorNodeStatus(ctx, client, nodeName); err == nil {
			select {
			case updates <- status:
			case <-ctx.Done():
				return
			}
		}

		// Send periodic updates
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, err := MonitorNodeStatus(ctx, client, nodeName)
				if err != nil {
					// Continue monitoring even if we get an error
					continue
				}

				select {
				case updates <- status:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return updates
}
