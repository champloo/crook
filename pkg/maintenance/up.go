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
// Steps: load state → validate → uncordon → restore deployments → scale operator → unset noout
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

	// Step 2: Load and validate state file
	maintenanceState, statePath, err := loadAndValidateState(cfg, opts, nodeName)
	if err != nil {
		return err
	}

	// Step 3: Validate deployments exist
	missingDeployments, err := validateDeploymentsExist(ctx, client, maintenanceState, opts)
	if err != nil {
		return err
	}

	// Step 4: Uncordon node FIRST so pods can schedule when deployments scale up
	sendUpProgress(opts.ProgressCallback, "uncordon", fmt.Sprintf("Uncordoning node %s", nodeName), "")
	if uncordonErr := client.UncordonNode(ctx, nodeName); uncordonErr != nil {
		return fmt.Errorf("failed to uncordon node %s: %w", nodeName, uncordonErr)
	}

	// Step 5: Restore deployments in order (pods can now schedule to uncordoned node)
	if restoreErr := restoreDeployments(ctx, client, cfg, maintenanceState, missingDeployments, opts); restoreErr != nil {
		return restoreErr
	}

	// Step 6: Scale up rook-ceph-operator
	if scaleErr := scaleOperator(ctx, client, cfg, maintenanceState, opts); scaleErr != nil {
		return scaleErr
	}

	// Step 7: Finalize - unset noout flag to allow normal Ceph rebalancing
	if finalizeErr := finalizeUpPhase(ctx, client, cfg, opts); finalizeErr != nil {
		return finalizeErr
	}

	sendUpProgress(opts.ProgressCallback, "complete", fmt.Sprintf("Up phase completed successfully - node %s is operational (state file: %s)", nodeName, statePath), "")
	return nil
}

// loadAndValidateState loads the state file and validates it matches the target node
func loadAndValidateState(cfg config.Config, opts UpPhaseOptions, nodeName string) (*state.State, string, error) {
	sendUpProgress(opts.ProgressCallback, "load-state", "Loading maintenance state file", "")

	statePath, err := resolveStatePath(cfg, opts.StateFilePath, nodeName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve state file path: %w", err)
	}

	maintenanceState, err := state.ParseFile(statePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load state file %s: %w", statePath, err)
	}

	if maintenanceState.Node != nodeName {
		return nil, "", fmt.Errorf("state file node mismatch: expected %s, got %s", nodeName, maintenanceState.Node)
	}

	return maintenanceState, statePath, nil
}

// validateDeploymentsExist checks that all deployments in state file exist in cluster
func validateDeploymentsExist(ctx context.Context, client *k8s.Client, maintenanceState *state.State, opts UpPhaseOptions) ([]string, error) {
	sendUpProgress(opts.ProgressCallback, "validate-deployments", "Validating deployments exist", "")

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

	if len(missingDeployments) > 0 && !opts.SkipMissingDeployments {
		return nil, fmt.Errorf("deployments missing from cluster: %v - cannot restore safely", missingDeployments)
	}

	if len(missingDeployments) > 0 {
		sendUpProgress(opts.ProgressCallback, "warning", fmt.Sprintf("Warning: %d deployments missing from cluster", len(missingDeployments)), "")
	}

	return missingDeployments, nil
}

// restoreDeployments scales up deployments in the correct order
// MON deployments are scaled first, then quorum is verified before scaling OSDs
func restoreDeployments(ctx context.Context, client *k8s.Client, cfg config.Config, maintenanceState *state.State, missingDeployments []string, opts UpPhaseOptions) error {
	deploymentResources := make([]state.Resource, 0)
	for _, resource := range maintenanceState.Resources {
		if resource.Kind == "Deployment" {
			deploymentResources = append(deploymentResources, resource)
		}
	}

	// Separate MON deployments from others
	monResources, otherResources := separateMonDeployments(deploymentResources)

	// First scale up MON deployments
	if len(monResources) > 0 {
		for _, resource := range monResources {
			deploymentName := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)

			if contains(missingDeployments, deploymentName) {
				sendUpProgress(opts.ProgressCallback, "skip", fmt.Sprintf("Skipping missing MON deployment %s", deploymentName), deploymentName)
				continue
			}

			sendUpProgress(opts.ProgressCallback, "scale-up", fmt.Sprintf("Scaling up MON %s to %d replicas", deploymentName, resource.Replicas), deploymentName)

			if err := client.ScaleDeployment(ctx, resource.Namespace, resource.Name, safeInt32(resource.Replicas)); err != nil {
				return fmt.Errorf("failed to scale MON deployment %s to %d: %w", deploymentName, resource.Replicas, err)
			}

			if err := WaitForDeploymentScaleUp(ctx, client, resource.Namespace, resource.Name, safeInt32(resource.Replicas), opts.WaitOptions); err != nil {
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
	orderedOther := orderResourcesForUp(otherResources)
	for _, resource := range orderedOther {
		deploymentName := fmt.Sprintf("%s/%s", resource.Namespace, resource.Name)

		if contains(missingDeployments, deploymentName) {
			sendUpProgress(opts.ProgressCallback, "skip", fmt.Sprintf("Skipping missing deployment %s", deploymentName), deploymentName)
			continue
		}

		sendUpProgress(opts.ProgressCallback, "scale-up", fmt.Sprintf("Scaling up %s to %d replicas", deploymentName, resource.Replicas), deploymentName)

		if err := client.ScaleDeployment(ctx, resource.Namespace, resource.Name, safeInt32(resource.Replicas)); err != nil {
			return fmt.Errorf("failed to scale deployment %s to %d: %w", deploymentName, resource.Replicas, err)
		}

		if err := WaitForDeploymentScaleUp(ctx, client, resource.Namespace, resource.Name, safeInt32(resource.Replicas), opts.WaitOptions); err != nil {
			return fmt.Errorf("failed waiting for deployment %s to scale up: %w", deploymentName, err)
		}
	}

	return nil
}

// separateMonDeployments separates MON deployments from other deployments
func separateMonDeployments(resources []state.Resource) (monResources, otherResources []state.Resource) {
	for _, resource := range resources {
		if startsWithPrefix(resource.Name, "rook-ceph-mon") {
			monResources = append(monResources, resource)
		} else {
			otherResources = append(otherResources, resource)
		}
	}
	return monResources, otherResources
}

// scaleOperator scales up the rook-ceph-operator deployment
func scaleOperator(ctx context.Context, client *k8s.Client, cfg config.Config, maintenanceState *state.State, opts UpPhaseOptions) error {
	sendUpProgress(opts.ProgressCallback, "operator", fmt.Sprintf("Scaling up rook-ceph-operator to %d", maintenanceState.OperatorReplicas), "")

	operatorName := "rook-ceph-operator"
	operatorNamespace := cfg.Kubernetes.RookOperatorNamespace

	if err := client.ScaleDeployment(ctx, operatorNamespace, operatorName, safeInt32(maintenanceState.OperatorReplicas)); err != nil {
		return fmt.Errorf("failed to scale operator to %d: %w", maintenanceState.OperatorReplicas, err)
	}

	if err := WaitForDeploymentScaleUp(ctx, client, operatorNamespace, operatorName, safeInt32(maintenanceState.OperatorReplicas), opts.WaitOptions); err != nil {
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

// safeInt32 safely converts an int to int32, clamping to valid range
func safeInt32(n int) int32 {
	if n < 0 {
		return 0
	}
	if n > 2147483647 {
		return 2147483647
	}
	return int32(n)
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

		if err := client.ScaleDeployment(ctx, resource.Namespace, resource.Name, safeInt32(resource.Replicas)); err != nil {
			return fmt.Errorf("failed to scale deployment %s to %d: %w", deploymentName, resource.Replicas, err)
		}

		if err := WaitForDeploymentScaleUp(ctx, client, resource.Namespace, resource.Name, safeInt32(resource.Replicas), opts); err != nil {
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
func sendUpProgress(callback func(UpPhaseProgress), stage, description, deployment string) {
	if callback != nil {
		callback(UpPhaseProgress{
			Stage:       stage,
			Description: description,
			Deployment:  deployment,
		})
	}
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
