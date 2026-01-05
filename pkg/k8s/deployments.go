package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentStatus holds the status information for a deployment
type DeploymentStatus struct {
	Name              string
	Namespace         string
	Replicas          int32
	ReadyReplicas     int32
	AvailableReplicas int32
	UpdatedReplicas   int32
}

// ScaleDeployment scales a deployment to the specified number of replicas
func (c *Client) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	deploymentsClient := c.Clientset.AppsV1().Deployments(namespace)

	// Get the current deployment
	deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
	}

	// Update the replicas
	deployment.Spec.Replicas = &replicas

	// Update the deployment
	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment %s/%s to %d replicas: %w", namespace, name, replicas, err)
	}

	return nil
}

// ScaleDeployment is a package-level function that uses the global client
func ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.ScaleDeployment(ctx, namespace, name, replicas)
}

// GetDeploymentStatus returns the status of a deployment
func (c *Client) GetDeploymentStatus(ctx context.Context, namespace, name string) (*DeploymentStatus, error) {
	deployment, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
	}

	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	return &DeploymentStatus{
		Name:              deployment.Name,
		Namespace:         deployment.Namespace,
		Replicas:          replicas,
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
	}, nil
}

// GetDeploymentStatus is a package-level function that uses the global client
func GetDeploymentStatus(ctx context.Context, namespace, name string) (*DeploymentStatus, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetDeploymentStatus(ctx, namespace, name)
}

// ListDeploymentsInNamespace returns all deployments in a namespace
func (c *Client) ListDeploymentsInNamespace(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	deploymentList, err := c.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments in namespace %s: %w", namespace, err)
	}

	return deploymentList.Items, nil
}

// ListDeploymentsInNamespace is a package-level function that uses the global client
func ListDeploymentsInNamespace(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListDeploymentsInNamespace(ctx, namespace)
}

// FilterDeploymentsByPrefix returns deployments whose names start with any of the given prefixes
func FilterDeploymentsByPrefix(deployments []appsv1.Deployment, prefixes []string) []appsv1.Deployment {
	if len(prefixes) == 0 {
		return deployments
	}

	filtered := make([]appsv1.Deployment, 0)
	for _, deployment := range deployments {
		for _, prefix := range prefixes {
			if len(deployment.Name) >= len(prefix) && deployment.Name[:len(prefix)] == prefix {
				filtered = append(filtered, deployment)
				break
			}
		}
	}

	return filtered
}

// WaitForReplicasOptions holds options for waiting for replicas
type WaitForReplicasOptions struct {
	// PollInterval is how often to check deployment status
	PollInterval time.Duration
	// Timeout is the maximum time to wait
	Timeout time.Duration
}

// DefaultWaitOptions returns default wait options
func DefaultWaitOptions() WaitForReplicasOptions {
	return WaitForReplicasOptions{
		PollInterval: 5 * time.Second,
		Timeout:      5 * time.Minute,
	}
}

// WaitForReplicas waits until the deployment has the expected number of replicas
func (c *Client) WaitForReplicas(ctx context.Context, namespace, name string, expectedReplicas int32, opts WaitForReplicasOptions) error {
	return c.waitForCondition(ctx, namespace, name, opts, func(deployment *appsv1.Deployment) bool {
		actualReplicas := int32(0)
		if deployment.Spec.Replicas != nil {
			actualReplicas = *deployment.Spec.Replicas
		}
		return actualReplicas == expectedReplicas
	}, fmt.Sprintf("replicas to be %d", expectedReplicas))
}

// WaitForReplicas is a package-level function that uses the global client
func WaitForReplicas(ctx context.Context, namespace, name string, expectedReplicas int32, opts WaitForReplicasOptions) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.WaitForReplicas(ctx, namespace, name, expectedReplicas, opts)
}

// WaitForReadyReplicas waits until the deployment has the expected number of ready replicas
func (c *Client) WaitForReadyReplicas(ctx context.Context, namespace, name string, expectedReady int32, opts WaitForReplicasOptions) error {
	return c.waitForCondition(ctx, namespace, name, opts, func(deployment *appsv1.Deployment) bool {
		return deployment.Status.ReadyReplicas == expectedReady
	}, fmt.Sprintf("ready replicas to be %d", expectedReady))
}

// WaitForReadyReplicas is a package-level function that uses the global client
func WaitForReadyReplicas(ctx context.Context, namespace, name string, expectedReady int32, opts WaitForReplicasOptions) error {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return err
	}
	return client.WaitForReadyReplicas(ctx, namespace, name, expectedReady, opts)
}

// waitForCondition is a helper that waits for a deployment to meet a condition
func (c *Client) waitForCondition(
	ctx context.Context,
	namespace, name string,
	opts WaitForReplicasOptions,
	condition func(*appsv1.Deployment) bool,
	conditionDesc string,
) error {
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	deploymentsClient := c.Clientset.AppsV1().Deployments(namespace)

	for {
		select {
		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout waiting for deployment %s/%s %s", namespace, name, conditionDesc)
			}
			return fmt.Errorf("context cancelled while waiting for deployment %s/%s", namespace, name)

		case <-ticker.C:
			deployment, err := deploymentsClient.Get(timeoutCtx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
			}

			if condition(deployment) {
				return nil
			}
		}
	}
}

// GetDeployment returns a deployment by name
func (c *Client) GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	deployment, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
	}
	return deployment, nil
}

// GetDeployment is a package-level function that uses the global client
func GetDeployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.GetDeployment(ctx, namespace, name)
}

// DeploymentInfoForLS holds deployment information for the ls command view
type DeploymentInfoForLS struct {
	// Name is the deployment name
	Name string

	// Namespace is the deployment namespace
	Namespace string

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32

	// DesiredReplicas is the desired number of replicas
	DesiredReplicas int32

	// NodeName is the node where the deployment's pod runs
	NodeName string

	// Age is the time since the deployment was created
	Age time.Duration

	// Status is the deployment status (Ready/Scaling/Unavailable)
	Status string

	// Type is the deployment type (osd/mon/exporter/crashcollector)
	Type string

	// OsdID is the OSD ID (from label ceph-osd-id, if applicable)
	OsdID string
}

// ListCephDeployments returns Ceph deployments with detailed info
func (c *Client) ListCephDeployments(ctx context.Context, namespace string, prefixes []string) ([]DeploymentInfoForLS, error) {
	// Get all deployments in the namespace
	deployments, err := c.ListDeploymentsInNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Filter by prefixes
	filtered := FilterDeploymentsByPrefix(deployments, prefixes)

	// Get pods in namespace to map deployments to nodes
	podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	// Build deployment -> node map via pods
	deploymentNodes := make(map[string]string)
	for _, pod := range podList.Items {
		// Find deployment via owner references
		for _, ownerRef := range pod.OwnerReferences {
			if ownerRef.Kind == "ReplicaSet" {
				// ReplicaSet name format: <deployment-name>-<hash>
				// Find the best (longest) matching deployment name to handle cases
				// where one deployment name is a prefix of another (e.g., "rook-ceph-exporter-rook"
				// vs "rook-ceph-exporter-rook-m02")
				rsName := ownerRef.Name
				var bestMatch string
				for _, dep := range filtered {
					prefix := dep.Name + "-"
					if strings.HasPrefix(rsName, prefix) {
						// Keep the longest matching deployment name
						if len(dep.Name) > len(bestMatch) {
							bestMatch = dep.Name
						}
					}
				}
				if bestMatch != "" {
					deploymentNodes[bestMatch] = pod.Spec.NodeName
				}
			}
		}
	}

	// Build result
	result := make([]DeploymentInfoForLS, 0, len(filtered))
	now := time.Now()

	for _, dep := range filtered {
		info := DeploymentInfoForLS{
			Name:            dep.Name,
			Namespace:       dep.Namespace,
			ReadyReplicas:   dep.Status.ReadyReplicas,
			DesiredReplicas: getDeploymentDesiredReplicas(&dep),
			NodeName:        deploymentNodes[dep.Name],
			Age:             now.Sub(dep.CreationTimestamp.Time),
			Status:          getDeploymentStatusString(&dep),
			Type:            extractDeploymentType(dep.Name),
			OsdID:           extractOsdID(&dep),
		}
		result = append(result, info)
	}

	return result, nil
}

// ListCephDeployments is a package-level function that uses the global client
func ListCephDeployments(ctx context.Context, namespace string, prefixes []string) ([]DeploymentInfoForLS, error) {
	client, err := GetClient(ctx, ClientConfig{})
	if err != nil {
		return nil, err
	}
	return client.ListCephDeployments(ctx, namespace, prefixes)
}

// getDeploymentDesiredReplicas returns the desired replicas for a deployment
func getDeploymentDesiredReplicas(dep *appsv1.Deployment) int32 {
	if dep.Spec.Replicas != nil {
		return *dep.Spec.Replicas
	}
	return 1 // Default
}

// getDeploymentStatusString returns a human-readable status for a deployment
func getDeploymentStatusString(dep *appsv1.Deployment) string {
	desired := getDeploymentDesiredReplicas(dep)
	ready := dep.Status.ReadyReplicas

	if ready == desired && desired > 0 {
		return "Ready"
	}
	if ready == 0 {
		if desired == 0 {
			return "Scaled Down"
		}
		return "Unavailable"
	}
	return "Scaling"
}

// extractDeploymentType extracts the deployment type from the name
func extractDeploymentType(name string) string {
	typeMap := map[string]string{
		"rook-ceph-osd":                    "osd",
		"rook-ceph-mon":                    "mon",
		"rook-ceph-mgr":                    "mgr",
		"rook-ceph-mds":                    "mds",
		"rook-ceph-rgw":                    "rgw",
		"rook-ceph-exporter":               "exporter",
		"rook-ceph-crashcollector":         "crashcollector",
		"csi-cephfsplugin-provisioner":     "csi",
		"csi-rbdplugin-provisioner":        "csi",
		"rook-ceph-tools":                  "tools",
		"rook-ceph-operator":               "operator",
		"rook-ceph-detect-version":         "detect",
		"rook-ceph-csi-detect-version":     "detect",
		"rook-ceph-filesystem-mirror":      "mirror",
		"rook-ceph-mirror":                 "mirror",
		"rook-ceph-purge-osd":              "purge",
		"rook-ceph-remove-mon":             "remove",
		"rook-ceph-nfs":                    "nfs",
		"rook-ceph-object-realm":           "realm",
		"rook-ceph-object-store":           "store",
		"rook-ceph-object-zone":            "zone",
		"rook-ceph-osd-prepare":            "prepare",
		"rook-ceph-direct-mount":           "mount",
		"rook-ceph-cleanup":                "cleanup",
		"rook-ceph-csi-cephfs-provisioner": "csi",
		"rook-ceph-csi-rbd-provisioner":    "csi",
		"rook-ceph-csi-nfs-provisioner":    "csi",
		"rook-ceph-csi-addons-controller":  "csi",
		"ceph-volumemodechange":            "volumemode",
	}

	// Check for exact or prefix matches
	for prefix, typ := range typeMap {
		if strings.HasPrefix(name, prefix) {
			return typ
		}
	}

	return "other"
}

// extractOsdID extracts the OSD ID from deployment labels
func extractOsdID(dep *appsv1.Deployment) string {
	if dep.Labels != nil {
		if osdID, ok := dep.Labels["ceph-osd-id"]; ok {
			return osdID
		}
	}
	return ""
}
