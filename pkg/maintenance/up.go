package maintenance

import (
	"context"
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// UpPhaseProgress tracks progress of the up phase operation
type UpPhaseProgress struct {
	Stage       string
	Description string
	Deployment  string // Optional: current deployment being processed
}

// UpPhaseOptions holds options for the up phase operation
type UpPhaseOptions struct {
	// ProgressCallback is called on each major step with progress updates
	// Optional - if nil, no progress updates are sent
	ProgressCallback func(progress UpPhaseProgress)

	// WaitOptions for deployment scaling operations
	WaitOptions WaitOptions

	// Deployments provides pre-discovered deployments to restore.
	// When set, ExecuteUpPhase uses these directly instead of re-discovering.
	// This ensures the confirmed plan matches the executed plan (avoiding TUI plan drift).
	// If nil, ExecuteUpPhase will discover deployments via ListScaledDownDeploymentsForNode.
	Deployments []appsv1.Deployment
}

// ExecuteUpPhase orchestrates the complete node up phase workflow
// Steps: pre-flight → discover scaled-down deployments → uncordon → restore deployments → scale operator → unset noout
func ExecuteUpPhase(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	nodeName string,
	opts UpPhaseOptions,
) error {
	// Step 1: Pre-flight validation
	sendUpProgress(opts.ProgressCallback, "pre-flight", "Running pre-flight validation checks", "")

	validationResults, err := ValidateUpPhase(ctx, client, cfg, nodeName)
	if err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}
	if !validationResults.AllPassed {
		return fmt.Errorf("pre-flight validation failed:\n%s", validationResults.String())
	}

	// Step 2: Use pre-discovered deployments or discover via nodeSelector
	var deployments []appsv1.Deployment
	if len(opts.Deployments) > 0 {
		// Use pre-discovered deployments (TUI confirmed plan)
		sendUpProgress(opts.ProgressCallback, "discover", fmt.Sprintf("Using %d pre-discovered deployments on %s", len(opts.Deployments), nodeName), "")
		deployments = opts.Deployments
	} else {
		// Discover deployments (CLI non-TUI mode)
		sendUpProgress(opts.ProgressCallback, "discover", fmt.Sprintf("Discovering scaled-down deployments on %s", nodeName), "")
		discovered, discoverErr := client.ListScaledDownDeploymentsForNode(ctx, cfg.Kubernetes.RookClusterNamespace, nodeName, cfg.DeploymentFilters.Prefixes)
		if discoverErr != nil {
			return fmt.Errorf("failed to discover scaled-down deployments: %w", discoverErr)
		}
		deployments = discovered
	}

	// Step 3: Uncordon node FIRST so pods can schedule when deployments scale up
	sendUpProgress(opts.ProgressCallback, "uncordon", fmt.Sprintf("Uncordoning node %s", nodeName), "")
	if uncordonErr := client.UncordonNode(ctx, nodeName); uncordonErr != nil {
		return fmt.Errorf("failed to uncordon node %s: %w", nodeName, uncordonErr)
	}

	// Step 4: Restore deployments in order
	if restoreErr := restoreDeployments(ctx, client, cfg, deployments, opts); restoreErr != nil {
		return restoreErr
	}

	// Step 5: Scale up rook-ceph-operator to 1
	if scaleErr := scaleOperator(ctx, client, cfg, opts); scaleErr != nil {
		return scaleErr
	}

	// Step 6: Finalize - unset noout flag to allow normal Ceph rebalancing
	if finalizeErr := finalizeUpPhase(ctx, client, cfg, opts); finalizeErr != nil {
		return finalizeErr
	}

	sendUpProgress(opts.ProgressCallback, "complete", fmt.Sprintf("Up phase completed successfully - node %s is operational", nodeName), "")
	return nil
}

// restoreDeployments scales up deployments in the correct order
// MON deployments are scaled first, then quorum is verified before scaling OSDs
// All deployments are scaled to 1 replica (Rook-Ceph node-pinned deployments always use 1)
func restoreDeployments(ctx context.Context, client *k8s.Client, cfg config.Config, deployments []appsv1.Deployment, opts UpPhaseOptions) error {
	if len(deployments) == 0 {
		sendUpProgress(opts.ProgressCallback, "skip", "No scaled-down deployments to restore", "")
		return nil
	}

	// Separate MON deployments from others
	monDeployments, otherDeployments := separateMonDeploymentsFromList(deployments)

	// First scale up MON deployments
	if len(monDeployments) > 0 {
		for _, deployment := range monDeployments {
			deploymentName := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)

			sendUpProgress(opts.ProgressCallback, "scale-up", fmt.Sprintf("Scaling up MON %s to 1 replica", deploymentName), deploymentName)

			if err := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 1); err != nil {
				return fmt.Errorf("failed to scale MON deployment %s to 1: %w", deploymentName, err)
			}

			if err := WaitForDeploymentScaleUp(ctx, client, deployment.Namespace, deployment.Name, 1, opts.WaitOptions); err != nil {
				return fmt.Errorf("failed waiting for MON deployment %s to scale up: %w", deploymentName, err)
			}
		}

		// Wait for MON quorum before proceeding to OSDs
		sendUpProgress(opts.ProgressCallback, "quorum", "Waiting for Ceph monitor quorum", "")
		if err := WaitForMonitorQuorum(ctx, client, cfg.Kubernetes.RookClusterNamespace, opts.WaitOptions); err != nil {
			return fmt.Errorf("failed waiting for monitor quorum: %w", err)
		}
		sendUpProgress(opts.ProgressCallback, "quorum", "Ceph monitor quorum established", "")
	}

	// Now scale up remaining deployments (OSDs and others) in order
	orderedOther := OrderDeploymentsForUp(otherDeployments)
	for _, deployment := range orderedOther {
		deploymentName := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)

		sendUpProgress(opts.ProgressCallback, "scale-up", fmt.Sprintf("Scaling up %s to 1 replica", deploymentName), deploymentName)

		if err := client.ScaleDeployment(ctx, deployment.Namespace, deployment.Name, 1); err != nil {
			return fmt.Errorf("failed to scale deployment %s to 1: %w", deploymentName, err)
		}

		if err := WaitForDeploymentScaleUp(ctx, client, deployment.Namespace, deployment.Name, 1, opts.WaitOptions); err != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale up: %w", deploymentName, err)
		}
	}

	return nil
}

// separateMonDeploymentsFromList separates MON deployments from other deployments
func separateMonDeploymentsFromList(deployments []appsv1.Deployment) (monDeployments, otherDeployments []appsv1.Deployment) {
	for _, deployment := range deployments {
		if strings.HasPrefix(deployment.Name, "rook-ceph-mon") {
			monDeployments = append(monDeployments, deployment)
		} else {
			otherDeployments = append(otherDeployments, deployment)
		}
	}
	return monDeployments, otherDeployments
}

// scaleOperator scales up the rook-ceph-operator deployment to 1
func scaleOperator(ctx context.Context, client *k8s.Client, cfg config.Config, opts UpPhaseOptions) error {
	sendUpProgress(opts.ProgressCallback, "operator", "Scaling up rook-ceph-operator to 1", "")

	operatorName := "rook-ceph-operator"
	operatorNamespace := cfg.Kubernetes.RookOperatorNamespace

	if err := client.ScaleDeployment(ctx, operatorNamespace, operatorName, 1); err != nil {
		return fmt.Errorf("failed to scale operator to 1: %w", err)
	}

	if err := WaitForDeploymentScaleUp(ctx, client, operatorNamespace, operatorName, 1, opts.WaitOptions); err != nil {
		return fmt.Errorf("failed waiting for operator to scale up: %w", err)
	}

	return nil
}

// finalizeUpPhase unsets the noout flag to allow normal Ceph rebalancing
func finalizeUpPhase(ctx context.Context, client *k8s.Client, cfg config.Config, opts UpPhaseOptions) error {
	sendUpProgress(opts.ProgressCallback, "unset-noout", "Unsetting Ceph noout flag", "")

	if err := client.UnsetNoOut(ctx, cfg.Kubernetes.RookClusterNamespace); err != nil {
		return fmt.Errorf("failed to unset noout flag: %w", err)
	}

	return nil
}

// sendUpProgress safely calls the progress callback if it's not nil
func sendUpProgress(callback func(UpPhaseProgress), stage, description, deployment string) {
	if callback != nil {
		callback(UpPhaseProgress{
			Stage:       stage,
			Description: description,
			Deployment:  deployment,
		})
	}
}
