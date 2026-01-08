package maintenance

import (
	"context"
	"fmt"
	"time"

	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// WaitOptions holds configuration for wait operations
type WaitOptions struct {
	// PollInterval is how often to check deployment status (default: 5 seconds)
	PollInterval time.Duration

	// Timeout is the maximum time to wait (default: 300 seconds)
	Timeout time.Duration

	// APITimeout is the timeout for individual API calls (default: 30 seconds)
	APITimeout time.Duration

	// ProgressCallback is called on each poll with current status
	// Optional - if nil, no progress updates are sent
	ProgressCallback func(status *k8s.DeploymentStatus)
}

// DefaultWaitOptions returns wait options with spec-compliant defaults
func DefaultWaitOptions() WaitOptions {
	return WaitOptions{
		PollInterval:     5 * time.Second,
		Timeout:          300 * time.Second,
		APITimeout:       30 * time.Second,
		ProgressCallback: nil,
	}
}

// WaitForDeploymentScaleDown polls until readyReplicas becomes 0
// Returns error if timeout is exceeded or context is cancelled
func WaitForDeploymentScaleDown(ctx context.Context, client *k8s.Client, namespace, name string, opts WaitOptions) error {
	return waitForCondition(ctx, client, namespace, name, opts,
		func(status *k8s.DeploymentStatus) bool {
			return status.ReadyReplicas == 0
		},
		"scale down to 0 ready replicas",
	)
}

// WaitForDeploymentScaleUp polls until replicas equals targetReplicas
// Returns error if timeout is exceeded or context is cancelled
func WaitForDeploymentScaleUp(ctx context.Context, client *k8s.Client, namespace, name string, targetReplicas int32, opts WaitOptions) error {
	return waitForCondition(ctx, client, namespace, name, opts,
		func(status *k8s.DeploymentStatus) bool {
			return status.Replicas == targetReplicas && status.ReadyReplicas == targetReplicas
		},
		fmt.Sprintf("scale up to %d replicas", targetReplicas),
	)
}

// WaitForDeploymentReady polls until deployment has expected ready replicas
// This is useful for waiting on deployments to become healthy after scaling
func WaitForDeploymentReady(ctx context.Context, client *k8s.Client, namespace, name string, expectedReady int32, opts WaitOptions) error {
	return waitForCondition(ctx, client, namespace, name, opts,
		func(status *k8s.DeploymentStatus) bool {
			return status.ReadyReplicas >= expectedReady
		},
		fmt.Sprintf("reach %d ready replicas", expectedReady),
	)
}

// waitForCondition is a helper that polls deployment status until condition is met
func waitForCondition(
	ctx context.Context,
	client *k8s.Client,
	namespace, name string,
	opts WaitOptions,
	condition func(*k8s.DeploymentStatus) bool,
	conditionDesc string,
) error {
	// Apply defaults if not set
	if opts.PollInterval == 0 {
		opts.PollInterval = 5 * time.Second
	}
	if opts.Timeout == 0 {
		opts.Timeout = 300 * time.Second
	}
	if opts.APITimeout == 0 {
		opts.APITimeout = 30 * time.Second
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	// Check immediately before first poll
	callCtx, callCancel := context.WithTimeout(timeoutCtx, opts.APITimeout)
	status, err := client.GetDeploymentStatus(callCtx, namespace, name)
	callCancel()
	if err != nil {
		return fmt.Errorf("failed to get deployment %s/%s status: %w", namespace, name, err)
	}

	// Send initial progress callback
	if opts.ProgressCallback != nil {
		opts.ProgressCallback(status)
	}

	// Check if already satisfied
	if condition(status) {
		return nil
	}

	// Poll until condition is met or timeout
	for {
		select {
		case <-timeoutCtx.Done():
			// Use last known status for error message; avoid unbounded API calls
			// that could block indefinitely if API server is unhealthy
			finalStatus := status

			// Attempt a final status fetch with a bounded timeout
			// This is best-effort and won't block the timeout/cancellation return
			finalFetchCtx, finalFetchCancel := context.WithTimeout(context.Background(), opts.APITimeout)
			if fetchedStatus, fetchErr := client.GetDeploymentStatus(finalFetchCtx, namespace, name); fetchErr == nil {
				finalStatus = fetchedStatus
			}
			finalFetchCancel()

			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf(
					"timeout waiting for deployment %s/%s to %s after %v - current state: replicas=%d, ready=%d, available=%d, updated=%d",
					namespace, name, conditionDesc, opts.Timeout,
					finalStatus.Replicas, finalStatus.ReadyReplicas, finalStatus.AvailableReplicas, finalStatus.UpdatedReplicas,
				)
			}
			return fmt.Errorf("context cancelled while waiting for deployment %s/%s to %s", namespace, name, conditionDesc)

		case <-ticker.C:
			pollCtx, pollCancel := context.WithTimeout(timeoutCtx, opts.APITimeout)
			status, err = client.GetDeploymentStatus(pollCtx, namespace, name)
			pollCancel()
			if err != nil {
				return fmt.Errorf("failed to get deployment %s/%s status: %w", namespace, name, err)
			}

			// Send progress callback
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(status)
			}

			// Check condition
			if condition(status) {
				return nil
			}
		}
	}
}

// WaitForMultipleDeploymentsScaleDown waits for multiple deployments to scale down
// Returns error if any deployment fails to scale down within timeout
func WaitForMultipleDeploymentsScaleDown(
	ctx context.Context,
	client *k8s.Client,
	deployments []appsv1.Deployment,
	opts WaitOptions,
) error {
	for _, deployment := range deployments {
		if err := WaitForDeploymentScaleDown(ctx, client, deployment.Namespace, deployment.Name, opts); err != nil {
			return fmt.Errorf("failed waiting for deployment %s/%s: %w", deployment.Namespace, deployment.Name, err)
		}
	}
	return nil
}

// WaitForMultipleDeploymentsScaleUp waits for multiple deployments to scale up.
// Target replicas: uses spec.Replicas if set and non-zero, otherwise defaults to 1.
// For Rook-Ceph node-pinned deployments, the target is typically 1 replica.
func WaitForMultipleDeploymentsScaleUp(
	ctx context.Context,
	client *k8s.Client,
	deployments []appsv1.Deployment,
	opts WaitOptions,
) error {
	for _, deployment := range deployments {
		targetReplicas := int32(1)
		if deployment.Spec.Replicas != nil {
			targetReplicas = *deployment.Spec.Replicas
		}

		if err := WaitForDeploymentScaleUp(ctx, client, deployment.Namespace, deployment.Name, targetReplicas, opts); err != nil {
			return fmt.Errorf("failed waiting for deployment %s/%s: %w", deployment.Namespace, deployment.Name, err)
		}
	}
	return nil
}

// WaitForMonitorQuorum polls until Ceph monitors establish quorum
// Returns error if timeout is exceeded, context is cancelled, or quorum cannot be verified
func WaitForMonitorQuorum(ctx context.Context, client *k8s.Client, namespace string, opts WaitOptions) error {
	// Apply defaults if not set
	if opts.PollInterval == 0 {
		opts.PollInterval = 5 * time.Second
	}
	if opts.Timeout == 0 {
		opts.Timeout = 300 * time.Second
	}
	if opts.APITimeout == 0 {
		opts.APITimeout = 30 * time.Second
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()

	// Check immediately before first poll
	callCtx, callCancel := context.WithTimeout(timeoutCtx, opts.APITimeout)
	status, err := client.GetMonitorStatus(callCtx, namespace)
	callCancel()
	if err != nil {
		// First check may fail if tools pod isn't ready yet - continue polling
		status = nil
	}

	if status != nil && status.HasQuorum() {
		return nil
	}

	var lastStatus *k8s.MonitorStatus
	var lastErr error

	// Poll until quorum is established or timeout
	for {
		select {
		case <-timeoutCtx.Done():
			errMsg := "timeout waiting for Ceph monitor quorum"
			if lastErr != nil {
				errMsg = fmt.Sprintf("%s - last error: %v", errMsg, lastErr)
			} else if lastStatus != nil {
				errMsg = fmt.Sprintf(
					"%s after %v - monitors in quorum: %d/%d (%v), out of quorum: %v",
					errMsg, opts.Timeout,
					lastStatus.InQuorum, lastStatus.TotalCount, lastStatus.QuorumNames, lastStatus.OutOfQuorum,
				)
			}
			if timeoutCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("%s - inspect manually with 'ceph quorum_status' via rook-ceph-tools", errMsg)
			}
			return fmt.Errorf("context cancelled while waiting for monitor quorum")

		case <-ticker.C:
			pollCtx, pollCancel := context.WithTimeout(timeoutCtx, opts.APITimeout)
			status, err = client.GetMonitorStatus(pollCtx, namespace)
			pollCancel()
			if err != nil {
				lastErr = err
				continue // Keep polling on transient errors
			}
			lastErr = nil
			lastStatus = status

			if status.HasQuorum() {
				return nil
			}
		}
	}
}
