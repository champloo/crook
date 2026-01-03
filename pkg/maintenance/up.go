package maintenance

import (
	"context"
	"fmt"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/state"
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

	// StateFilePath optionally overrides the config template path
	StateFilePath string

	// SkipMissingDeployments continues execution even if some deployments in state file don't exist
	// When false (default), the operation fails if any deployment is missing
	SkipMissingDeployments bool
}

// ExecuteUpPhase orchestrates the complete node up phase workflow
// Steps: load state → validate → restore deployments → scale operator → unset noout → uncordon
func ExecuteUpPhase(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	nodeName string,
	opts UpPhaseOptions,
) error {
	// Step 1: Pre-flight validation
	if err := sendUpProgress(opts.ProgressCallback, "pre-flight", "Running pre-flight validation checks", ""); err != nil {
		return err
	}

	validationResults, err := ValidateUpPhase(ctx, client, cfg, nodeName)
	if err != nil {
		return fmt.Errorf("pre-flight validation failed: %w", err)
	}
	if !validationResults.AllPassed {
		return fmt.Errorf("pre-flight validation failed:\n%s", validationResults.String())
	}

	// Step 2: Load state file
	if err := sendUpProgress(opts.ProgressCallback, "load-state", "Loading maintenance state file", ""); err != nil {
		return err
	}

	statePath, err := resolveStatePath(cfg, opts.StateFilePath, nodeName)
	if err != nil {
		return fmt.Errorf("failed to resolve state file path: %w", err)
	}

	maintenanceState, err := state.ParseFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to load state file %s: %w", statePath, err)
	}

	// Validate state file node matches
	if maintenanceState.Node != nodeName {
		return fmt.Errorf("state file node mismatch: expected %s, got %s", nodeName, maintenanceState.Node)
	}

	// Step 3: Validate deployments exist
	if err := sendUpProgress(opts.ProgressCallback, "validate-deployments", "Validating deployments exist", ""); err != nil {
		return err
	}

	missingDeployments := make([]string, 0)
	for _, resource := range maintenanceState.Resources {
		if resource.Kind != "Deployment" {
			continue
		}

		_, err := client.GetDeployment(ctx, resource.Namespace, resource.Name)
		if err != nil {
			deploymentName := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)
			missingDeployments = append(missingDeployments, deploymentName)
		}
	}

	if len(missingDeployments) > 0 {
		if !opts.SkipMissingDeployments {
			return fmt.Errorf("deployments missing from cluster: %v - cannot restore safely", missingDeployments)
		}
		// Log warning but continue
		if err := sendUpProgress(opts.ProgressCallback, "warning", fmt.Sprintf("Warning: %d deployments missing from cluster", len(missingDeployments)), ""); err != nil {
			return err
		}
	}

	// Step 4: Restore deployments in order
	deploymentResources := make([]state.Resource, 0)
	for _, resource := range maintenanceState.Resources {
		if resource.Kind == "Deployment" {
			deploymentResources = append(deploymentResources, resource)
		}
	}

	// Order deployments for safe up phase
	orderedResources := orderResourcesForUp(deploymentResources)

	for _, resource := range orderedResources {
		deploymentName := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)

		// Skip if deployment is missing and we're allowed to skip
		if contains(missingDeployments, deploymentName) {
			if err := sendUpProgress(opts.ProgressCallback, "skip", fmt.Sprintf("Skipping missing deployment %s", deploymentName), deploymentName); err != nil {
				return err
			}
			continue
		}

		if err := sendUpProgress(opts.ProgressCallback, "scale-up", fmt.Sprintf("Scaling up %s to %d replicas", deploymentName, resource.Replicas), deploymentName); err != nil {
			return err
		}

		if err := client.ScaleDeployment(ctx, resource.Namespace, resource.Name, int32(resource.Replicas)); err != nil {
			return fmt.Errorf("failed to scale deployment %s to %d: %w", deploymentName, resource.Replicas, err)
		}

		if err := WaitForDeploymentScaleUp(ctx, client, resource.Namespace, resource.Name, int32(resource.Replicas), opts.WaitOptions); err != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale up: %w", deploymentName, err)
		}
	}

	// Step 5: Scale up rook-ceph-operator
	if err := sendUpProgress(opts.ProgressCallback, "operator", fmt.Sprintf("Scaling up rook-ceph-operator to %d", maintenanceState.OperatorReplicas), ""); err != nil {
		return err
	}

	operatorName := "rook-ceph-operator"
	operatorNamespace := cfg.Kubernetes.RookOperatorNamespace

	if err := client.ScaleDeployment(ctx, operatorNamespace, operatorName, int32(maintenanceState.OperatorReplicas)); err != nil {
		return fmt.Errorf("failed to scale operator to %d: %w", maintenanceState.OperatorReplicas, err)
	}

	if err := WaitForDeploymentScaleUp(ctx, client, operatorNamespace, operatorName, int32(maintenanceState.OperatorReplicas), opts.WaitOptions); err != nil {
		return fmt.Errorf("failed waiting for operator to scale up: %w", err)
	}

	// Step 6: Unset Ceph noout flag
	if err := sendUpProgress(opts.ProgressCallback, "unset-noout", "Unsetting Ceph noout flag", ""); err != nil {
		return err
	}

	if err := client.UnsetNoOut(ctx, cfg.Kubernetes.RookClusterNamespace); err != nil {
		return fmt.Errorf("failed to unset noout flag: %w", err)
	}

	// Step 7: Uncordon node
	if err := sendUpProgress(opts.ProgressCallback, "uncordon", fmt.Sprintf("Uncordoning node %s", nodeName), ""); err != nil {
		return err
	}

	if err := client.UncordonNode(ctx, nodeName); err != nil {
		return fmt.Errorf("failed to uncordon node %s: %w", nodeName, err)
	}

	// Step 8: Complete
	if err := sendUpProgress(opts.ProgressCallback, "complete", fmt.Sprintf("Up phase completed successfully - node %s is operational", nodeName), ""); err != nil {
		return err
	}

	return nil
}

// ScaleUpDeployments scales up multiple deployments from state resources and waits for each
func ScaleUpDeployments(
	ctx context.Context,
	client *k8s.Client,
	resources []state.Resource,
	opts WaitOptions,
	progressCallback func(deploymentName string),
) error {
	for _, resource := range resources {
		if resource.Kind != "Deployment" {
			continue
		}

		deploymentName := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)

		if progressCallback != nil {
			progressCallback(deploymentName)
		}

		if err := client.ScaleDeployment(ctx, resource.Namespace, resource.Name, int32(resource.Replicas)); err != nil {
			return fmt.Errorf("failed to scale deployment %s to %d: %w", deploymentName, resource.Replicas, err)
		}

		if err := WaitForDeploymentScaleUp(ctx, client, resource.Namespace, resource.Name, int32(resource.Replicas), opts); err != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale up: %w", deploymentName, err)
		}
	}

	return nil
}

// orderResourcesForUp orders state resources for safe up phase
// Order: rook-ceph-mon (first), rook-ceph-osd, rook-ceph-exporter, rook-ceph-crashcollector (last)
func orderResourcesForUp(resources []state.Resource) []state.Resource {
	upOrder := []string{
		"rook-ceph-mon",
		"rook-ceph-osd",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	ordered := make([]state.Resource, 0, len(resources))

	// Add resources in prefix order
	for _, prefix := range upOrder {
		for _, resource := range resources {
			if startsWithPrefix(resource.Name, prefix) {
				ordered = append(ordered, resource)
			}
		}
	}

	// Add any remaining resources that didn't match prefixes
	for _, resource := range resources {
		found := false
		for _, existing := range ordered {
			if existing.Namespace == resource.Namespace && existing.Name == resource.Name {
				found = true
				break
			}
		}
		if !found {
			ordered = append(ordered, resource)
		}
	}

	return ordered
}

// startsWithPrefix checks if a string starts with the given prefix
func startsWithPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// sendUpProgress safely calls the progress callback if it's not nil
func sendUpProgress(callback func(UpPhaseProgress), stage, description, deployment string) error {
	if callback != nil {
		callback(UpPhaseProgress{
			Stage:       stage,
			Description: description,
			Deployment:  deployment,
		})
	}
	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
