package maintenance

import (
	"context"
	"fmt"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// DownPhaseProgress tracks progress of the down phase operation
type DownPhaseProgress struct {
	Stage       string
	Description string
	Deployment  string // Optional: current deployment being processed
}

// DownPhaseOptions holds options for the down phase operation
type DownPhaseOptions struct {
	// ProgressCallback is called on each major step with progress updates
	// Optional - if nil, no progress updates are sent
	ProgressCallback func(progress DownPhaseProgress)

	// WaitOptions for deployment scaling operations
	WaitOptions WaitOptions
}

// ExecuteDownPhase orchestrates the complete node down phase workflow
// Steps: pre-flight → cordon → set noout → scale operator → discover → scale deployments
func ExecuteDownPhase(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	nodeName string,
	opts DownPhaseOptions,
) error {
	// Step 1: Pre-flight validation
	updateProgress(opts.ProgressCallback, "pre-flight", "Running pre-flight validation checks", "")

	validationResults, err := ValidateDownPhase(ctx, client, cfg, nodeName)
	if err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}
	if !validationResults.AllPassed {
		return fmt.Errorf("pre-flight validation failed:\n%s", validationResults.String())
	}

	// Step 2: Cordon node
	updateProgress(opts.ProgressCallback, "cordon", fmt.Sprintf("Cordoning node %s", nodeName), "")

	if cordonErr := client.CordonNode(ctx, nodeName); cordonErr != nil {
		return fmt.Errorf("failed to cordon node %s: %w", nodeName, cordonErr)
	}

	// Step 3: Set Ceph noout flag
	updateProgress(opts.ProgressCallback, "noout", "Setting Ceph noout flag", "")

	if nooutErr := client.SetNoOut(ctx, cfg.Kubernetes.RookClusterNamespace); nooutErr != nil {
		return fmt.Errorf("failed to set noout flag: %w", nooutErr)
	}

	// Step 4: Scale down rook-ceph-operator
	updateProgress(opts.ProgressCallback, "operator", "Scaling down rook-ceph-operator to 0", "")

	operatorName := "rook-ceph-operator"
	operatorNamespace := cfg.Kubernetes.RookOperatorNamespace

	if scaleErr := client.ScaleDeployment(ctx, operatorNamespace, operatorName, 0); scaleErr != nil {
		return fmt.Errorf("failed to scale operator to 0: %w", scaleErr)
	}

	if waitErr := WaitForDeploymentScaleDown(ctx, client, operatorNamespace, operatorName, opts.WaitOptions); waitErr != nil {
		return fmt.Errorf("failed waiting for operator to scale down: %w", waitErr)
	}

	// Step 5: Discover node-pinned deployments via nodeSelector
	updateProgress(opts.ProgressCallback, "discover", fmt.Sprintf("Discovering node-pinned deployments on %s", nodeName), "")

	deployments, err := client.ListNodePinnedDeployments(ctx, cfg.Kubernetes.RookClusterNamespace, nodeName)
	if err != nil {
		return fmt.Errorf("failed to discover node-pinned deployments: %w", err)
	}

	if len(deployments) == 0 {
		updateProgress(opts.ProgressCallback, "complete", "No node-pinned deployments found - down phase complete", "")
		return nil
	}

	// Warn on unexpected replica counts (>1)
	ValidateDeploymentReplicas(deployments)

	// Step 6: Order deployments for safe down phase
	// Note: Unlike UP phase, DOWN does not require special MON separation.
	// The noout flag prevents rebalancing, and we're going offline anyway.
	// See OrderDeploymentsForDown documentation for the full rationale.
	orderedDeployments := OrderDeploymentsForDown(deployments)

	// Step 7: Scale down each deployment and wait
	for _, deployment := range orderedDeployments {
		deploymentName := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)

		// Skip deployments already at 0 replicas
		if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == 0 {
			logger.Debug("skipping deployment already at 0 replicas", "deployment", deploymentName)
			continue
		}

		updateProgress(opts.ProgressCallback, "scale-down", fmt.Sprintf("Scaling down %s to 0", deploymentName), deploymentName)

		if scaleErr := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 0); scaleErr != nil {
			return fmt.Errorf("failed to scale deployment %s to 0: %w", deploymentName, scaleErr)
		}

		if waitErr := WaitForDeploymentScaleDown(ctx, client, deployment.Namespace, deployment.Name, opts.WaitOptions); waitErr != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale down: %w", deploymentName, waitErr)
		}
	}

	// Step 8: Complete
	updateProgress(opts.ProgressCallback, "complete", "Down phase completed successfully", "")

	return nil
}

// ScaleDownDeployments scales down multiple deployments and waits for each
func ScaleDownDeployments(
	ctx context.Context,
	client *k8s.Client,
	deployments []appsv1.Deployment,
	opts WaitOptions,
	progressCallback func(deploymentName string),
) error {
	for _, deployment := range deployments {
		if progressCallback != nil {
			progressCallback(fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))
		}

		if err := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 0); err != nil {
			return fmt.Errorf("failed to scale deployment %s/%s: %w", deployment.Namespace, deployment.Name, err)
		}

		if err := WaitForDeploymentScaleDown(ctx, client, deployment.Namespace, deployment.Name, opts); err != nil {
			return fmt.Errorf("failed waiting for deployment %s/%s to scale down: %w", deployment.Namespace, deployment.Name, err)
		}
	}

	return nil
}

// updateProgress safely calls the progress callback if it's not nil
func updateProgress(callback func(DownPhaseProgress), stage, description, deployment string) {
	if callback != nil {
		callback(DownPhaseProgress{
			Stage:       stage,
			Description: description,
			Deployment:  deployment,
		})
	}
}

// ValidateDeploymentReplicas warns if any deployment has unexpected replica count.
// Rook-Ceph node-pinned deployments should always have 1 replica.
// This is a warning-only validation; it does not return an error.
func ValidateDeploymentReplicas(deployments []appsv1.Deployment) {
	var warnings []string
	for _, dep := range deployments {
		if dep.Spec.Replicas != nil && *dep.Spec.Replicas > 1 {
			warnings = append(warnings, fmt.Sprintf(
				"%s/%s has %d replicas (expected 1)",
				dep.Namespace, dep.Name, *dep.Spec.Replicas,
			))
		}
	}
	if len(warnings) > 0 {
		logger.Warn("unexpected replica counts detected",
			"deployments", warnings,
			"note", "Rook-Ceph node-pinned deployments should have 1 replica")
	}
}
