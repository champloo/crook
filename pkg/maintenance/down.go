package maintenance

import (
	"context"
	"fmt"

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
	if err := updateProgress(opts.ProgressCallback, "pre-flight", "Running pre-flight validation checks", ""); err != nil {
		return err
	}

	validationResults, err := ValidateDownPhase(ctx, client, cfg, nodeName)
	if err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}
	if !validationResults.AllPassed {
		return fmt.Errorf("pre-flight validation failed:\n%s", validationResults.String())
	}

	// Step 2: Cordon node
	if err := updateProgress(opts.ProgressCallback, "cordon", fmt.Sprintf("Cordoning node %s", nodeName), ""); err != nil {
		return err
	}

	if err := client.CordonNode(ctx, nodeName); err != nil {
		return fmt.Errorf("failed to cordon node %s: %w", nodeName, err)
	}

	// Step 3: Set Ceph noout flag
	if err := updateProgress(opts.ProgressCallback, "noout", "Setting Ceph noout flag", ""); err != nil {
		return err
	}

	if err := client.SetNoOut(ctx, cfg.Kubernetes.RookClusterNamespace); err != nil {
		return fmt.Errorf("failed to set noout flag: %w", err)
	}

	// Step 4: Scale down rook-ceph-operator
	if err := updateProgress(opts.ProgressCallback, "operator", "Scaling down rook-ceph-operator to 0", ""); err != nil {
		return err
	}

	operatorName := "rook-ceph-operator"
	operatorNamespace := cfg.Kubernetes.RookOperatorNamespace

	// Get current operator replicas before scaling down
	operatorStatus, err := client.GetDeploymentStatus(ctx, operatorNamespace, operatorName)
	if err != nil {
		return fmt.Errorf("failed to get operator status: %w", err)
	}

	if err := client.ScaleDeployment(ctx, operatorNamespace, operatorName, 0); err != nil {
		return fmt.Errorf("failed to scale operator to 0: %w", err)
	}

	if err := WaitForDeploymentScaleDown(ctx, client, operatorNamespace, operatorName, opts.WaitOptions); err != nil {
		return fmt.Errorf("failed waiting for operator to scale down: %w", err)
	}

	// Step 5: Discover deployments on node
	if err := updateProgress(opts.ProgressCallback, "discover", fmt.Sprintf("Discovering deployments on node %s", nodeName), ""); err != nil {
		return err
	}

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
		if err := updateProgress(opts.ProgressCallback, "complete", "No deployments found - creating empty state file", ""); err != nil {
			return err
		}

		emptyState := state.NewState(nodeName, int(operatorStatus.Replicas), []state.Resource{})
		statePath, err := resolveStatePath(cfg, opts.StateFilePath, nodeName)
		if err != nil {
			return fmt.Errorf("failed to resolve state file path: %w", err)
		}

		if err := state.WriteFile(statePath, emptyState); err != nil {
			return fmt.Errorf("failed to save empty state file: %w", err)
		}

		return nil
	}

	// Step 6: Order deployments for safe down phase
	orderedDeployments := OrderDeploymentsForDown(deployments)

	// Step 7: Scale down each deployment and wait
	for _, deployment := range orderedDeployments {
		deploymentName := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)

		if err := updateProgress(opts.ProgressCallback, "scale-down", fmt.Sprintf("Scaling down %s to 0", deploymentName), deploymentName); err != nil {
			return err
		}

		if err := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 0); err != nil {
			return fmt.Errorf("failed to scale deployment %s to 0: %w", deploymentName, err)
		}

		if err := WaitForDeploymentScaleDown(ctx, client, deployment.Namespace, deployment.Name, opts.WaitOptions); err != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale down: %w", deploymentName, err)
		}
	}

	// Step 8: Save state file (only after all operations succeed)
	if err := updateProgress(opts.ProgressCallback, "save-state", "Saving maintenance state file", ""); err != nil {
		return err
	}

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

	if err := state.WriteFile(statePath, maintenanceState); err != nil {
		return fmt.Errorf("failed to save state file: %w", err)
	}

	// Step 9: Complete
	if err := updateProgress(opts.ProgressCallback, "complete", fmt.Sprintf("Down phase completed successfully - state saved to %s", statePath), ""); err != nil {
		return err
	}

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
func updateProgress(callback func(DownPhaseProgress), stage, description, deployment string) error {
	if callback != nil {
		callback(DownPhaseProgress{
			Stage:       stage,
			Description: description,
			Deployment:  deployment,
		})
	}
	return nil
}
