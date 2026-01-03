package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CephStatus represents the parsed output of 'ceph status --format json'
type CephStatus struct {
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
	OSDMap struct {
		NumOSDs       int `json:"num_osds"`
		NumUpOSDs     int `json:"num_up_osds"`
		NumInOSDs     int `json:"num_in_osds"`
		Full          bool `json:"full"`
		NearFull      bool `json:"nearfull"`
	} `json:"osdmap"`
}

// CephOSDTree represents the parsed output of 'ceph osd tree --format json'
type CephOSDTree struct {
	Nodes []CephOSDNode `json:"nodes"`
}

// CephOSDNode represents a node in the OSD tree
type CephOSDNode struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Reweight float64 `json:"reweight"`
	Children []int   `json:"children,omitempty"`
}

// ExecuteCephCommand executes a Ceph command via the rook-ceph-tools pod
func (c *Client) ExecuteCephCommand(ctx context.Context, namespace string, command []string) (string, error) {
	// Find the rook-ceph-tools pod
	pod, err := c.findRookCephToolsPod(ctx, namespace)
	if err != nil {
		return "", err
	}

	// Execute the command in the pod
	output, err := c.ExecInPod(ctx, namespace, pod.Name, "", command)
	if err != nil {
		return "", fmt.Errorf("failed to execute ceph command: %w", err)
	}

	return output, nil
}

// ExecuteCephCommand is a package-level function that uses the global client
func ExecuteCephCommand(ctx context.Context, namespace string, command []string) (string, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return "", err
	}
	return client.ExecuteCephCommand(ctx, namespace, command)
}

// findRookCephToolsPod finds a ready rook-ceph-tools pod in the namespace
func (c *Client) findRookCephToolsPod(ctx context.Context, namespace string) (*corev1.Pod, error) {
	// List pods with label selector for rook-ceph-tools
	listOptions := metav1.ListOptions{
		LabelSelector: "app=rook-ceph-tools",
	}

	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list rook-ceph-tools pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf(
			"no rook-ceph-tools pod found in namespace %s. "+
				"Please ensure the rook-ceph-tools deployment is running. "+
				"See https://rook.io/docs/rook/latest/Troubleshooting/ceph-toolbox/",
			namespace,
		)
	}

	// Find a ready pod
	for _, pod := range podList.Items {
		if isPodReady(&pod) {
			return &pod, nil
		}
	}

	return nil, fmt.Errorf(
		"no ready rook-ceph-tools pod found in namespace %s. "+
			"Found %d pod(s) but none are ready",
		namespace,
		len(podList.Items),
	)
}

// isPodReady checks if a pod is in the Ready state
func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}

	return false
}

// SetNoOut sets the Ceph noout flag
func (c *Client) SetNoOut(ctx context.Context, namespace string) error {
	_, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "set", "noout"})
	if err != nil {
		return fmt.Errorf("failed to set noout flag: %w", err)
	}
	return nil
}

// SetNoOut is a package-level function that uses the global client
func SetNoOut(ctx context.Context, namespace string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.SetNoOut(ctx, namespace)
}

// UnsetNoOut unsets the Ceph noout flag
func (c *Client) UnsetNoOut(ctx context.Context, namespace string) error {
	_, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "unset", "noout"})
	if err != nil {
		return fmt.Errorf("failed to unset noout flag: %w", err)
	}
	return nil
}

// UnsetNoOut is a package-level function that uses the global client
func UnsetNoOut(ctx context.Context, namespace string) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.UnsetNoOut(ctx, namespace)
}

// GetCephStatus gets the Ceph cluster status
func (c *Client) GetCephStatus(ctx context.Context, namespace string) (*CephStatus, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "status", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph status: %w", err)
	}

	var status CephStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse ceph status JSON: %w", err)
	}

	return &status, nil
}

// GetCephStatus is a package-level function that uses the global client
func GetCephStatus(ctx context.Context, namespace string) (*CephStatus, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetCephStatus(ctx, namespace)
}

// GetOSDTree gets the Ceph OSD tree
func (c *Client) GetOSDTree(ctx context.Context, namespace string) (*CephOSDTree, error) {
	output, err := c.ExecuteCephCommand(ctx, namespace, []string{"ceph", "osd", "tree", "--format", "json"})
	if err != nil {
		return nil, fmt.Errorf("failed to get ceph osd tree: %w", err)
	}

	var tree CephOSDTree
	if err := json.Unmarshal([]byte(output), &tree); err != nil {
		return nil, fmt.Errorf("failed to parse ceph osd tree JSON: %w", err)
	}

	return &tree, nil
}

// GetOSDTree is a package-level function that uses the global client
func GetOSDTree(ctx context.Context, namespace string) (*CephOSDTree, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetOSDTree(ctx, namespace)
}

// IsHealthy checks if the Ceph cluster is healthy
func (s *CephStatus) IsHealthy() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_OK"
}

// IsWarning checks if the Ceph cluster has warnings
func (s *CephStatus) IsWarning() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_WARN"
}

// IsError checks if the Ceph cluster has errors
func (s *CephStatus) IsError() bool {
	return strings.ToUpper(s.Health.Status) == "HEALTH_ERR"
}
