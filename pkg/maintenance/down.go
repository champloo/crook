package maintenance

import (
	"context"
	"fmt"

	"github.com/andri/crook/internal/logger"
	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/state"
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

	// StateFilePath optionally overrides the config template path
	StateFilePath string
}

// ExecuteDownPhase orchestrates the complete node down phase workflow
// Steps: pre-flight → cordon → set noout → scale operator → discover → scale deployments → save state
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

	// Get current operator replicas before scaling down
	operatorStatus, err := client.GetDeploymentStatus(ctx, operatorNamespace, operatorName)
	if err != nil {
		return fmt.Errorf("failed to get operator status: %w", err)
	}

	if scaleErr := client.ScaleDeployment(ctx, operatorNamespace, operatorName, 0); scaleErr != nil {
		return fmt.Errorf("failed to scale operator to 0: %w", scaleErr)
	}

	if waitErr := WaitForDeploymentScaleDown(ctx, client, operatorNamespace, operatorName, opts.WaitOptions); waitErr != nil {
		return fmt.Errorf("failed waiting for operator to scale down: %w", waitErr)
	}

	// Step 5: Discover deployments on node
	updateProgress(opts.ProgressCallback, "discover", fmt.Sprintf("Discovering deployments on node %s", nodeName), "")

	deployments, err := DiscoverDeployments(
		ctx,
		client,
		nodeName,
		cfg.Kubernetes.RookClusterNamespace,
		cfg.DeploymentFilters.Prefixes,
	)
	if err != nil {
		return fmt.Errorf("failed to discover deployments: %w", err)
	}

	if len(deployments) == 0 {
		// No deployments found - create empty state file for consistency
		updateProgress(opts.ProgressCallback, "complete", "No deployments found - creating empty state file", "")

		emptyState := state.NewState(nodeName, int(operatorStatus.Replicas), []state.Resource{})
		statePath, resolveErr := resolveStatePath(cfg, opts.StateFilePath, nodeName)
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve state file path: %w", resolveErr)
		}

		if writeErr := state.WriteFile(statePath, emptyState); writeErr != nil {
			return fmt.Errorf("failed to save empty state file: %w", writeErr)
		}

		return nil
	}

	// Step 6: Order deployments for safe down phase
	orderedDeployments := OrderDeploymentsForDown(deployments)

	// Step 7: Scale down each deployment and wait
	for _, deployment := range orderedDeployments {
		deploymentName := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)

		updateProgress(opts.ProgressCallback, "scale-down", fmt.Sprintf("Scaling down %s to 0", deploymentName), deploymentName)

		if scaleErr := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 0); scaleErr != nil {
			return fmt.Errorf("failed to scale deployment %s to 0: %w", deploymentName, scaleErr)
		}

		if waitErr := WaitForDeploymentScaleDown(ctx, client, deployment.Namespace, deployment.Name, opts.WaitOptions); waitErr != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale down: %w", deploymentName, waitErr)
		}
	}

	// Step 8: Save state file (only after all operations succeed)
	updateProgress(opts.ProgressCallback, "save-state", "Saving maintenance state file", "")

	resources := make([]state.Resource, 0, len(deployments))
	for _, deployment := range deployments {
		replicas := int32(1)
		if deployment.Spec.Replicas != nil {
			replicas = *deployment.Spec.Replicas
		}

		resources = append(resources, state.Resource{
			Kind:      "Deployment",
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
			Replicas:  int(replicas),
		})
	}

	maintenanceState := state.NewState(nodeName, int(operatorStatus.Replicas), resources)

	statePath, err := resolveStatePath(cfg, opts.StateFilePath, nodeName)
	if err != nil {
		return fmt.Errorf("failed to resolve state file path: %w", err)
	}

	if writeErr := state.WriteFile(statePath, maintenanceState); writeErr != nil {
		return fmt.Errorf("failed to save state file: %w", writeErr)
	}

	// Step 9: Complete
	updateProgress(opts.ProgressCallback, "complete", fmt.Sprintf("Down phase completed successfully - state saved to %s", statePath), "")

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

// resolveStatePath resolves the state file path from config or override
func resolveStatePath(cfg config.Config, overridePath, nodeName string) (string, error) {
	return state.ResolvePathWithOverride(overridePath, cfg.State.FilePathTemplate, nodeName)
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
