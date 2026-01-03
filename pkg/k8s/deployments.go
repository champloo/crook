package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeploymentStatus holds the status information for a deployment
type DeploymentStatus struct {
	Name             string
	Namespace        string
	Replicas         int32
	ReadyReplicas    int32
	AvailableReplicas int32
	UpdatedReplicas  int32
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

// Helper function to create a clientset from a kubernetes.Interface (for testing)
func newClientFromClientset(clientset kubernetes.Interface) *Client {
	return &Client{
		Clientset: clientset,
	}
}
