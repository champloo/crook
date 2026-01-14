package maintenance

import (
	"context"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

const operatorDeploymentName = "rook-ceph-operator"

// IsInDownState checks if the node is fully in the "down" maintenance state.
// This includes:
//   - Node is cordoned (unschedulable)
//   - Ceph noout flag is set
//   - rook-ceph-operator is scaled to 0 and has no ready replicas
//   - All provided deployments are scaled to 0 and have no ready replicas
//
// Returns true only if ALL conditions are met. On any error, returns false
// so the caller proceeds with the maintenance operation (fail-safe behavior).
func IsInDownState(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	nodeName string,
	deployments []appsv1.Deployment,
) bool {
	// Check deployments first (no API call needed)
	if !AllDeploymentsScaledDown(deployments) {
		return false
	}

	// Check node is cordoned
	nodeStatus, err := client.GetNodeStatus(ctx, nodeName)
	if err != nil || !nodeStatus.Unschedulable {
		return false
	}

	// Check noout flag is set
	flags, err := client.GetCephFlags(ctx, cfg.Namespace)
	if err != nil || !flags.NoOut {
		return false
	}

	// Check operator is scaled down
	opStatus, err := client.GetDeploymentStatus(ctx, cfg.Namespace, operatorDeploymentName)
	if err != nil || opStatus.Replicas != 0 || opStatus.ReadyReplicas != 0 {
		return false
	}

	return true
}

// IsInUpState checks if the node is fully in the "up" operational state.
// This includes:
//   - Node is schedulable (not cordoned)
//   - Ceph noout flag is unset
//   - rook-ceph-operator is scaled to 1 and has 1 ready replica
//   - No deployments need to be restored (empty list means all are up)
//
// Returns true only if ALL conditions are met. On any error, returns false
// so the caller proceeds with the maintenance operation (fail-safe behavior).
func IsInUpState(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	nodeName string,
	deployments []appsv1.Deployment,
) bool {
	// Check deployments first (no API call needed)
	// For up state, we expect the deployments list to be empty
	// (ListScaledDownDeploymentsForNode returns only scaled-down deployments)
	if len(deployments) > 0 {
		return false
	}

	// Check node is schedulable
	nodeStatus, err := client.GetNodeStatus(ctx, nodeName)
	if err != nil || nodeStatus.Unschedulable {
		return false
	}

	// Check noout flag is unset
	flags, err := client.GetCephFlags(ctx, cfg.Namespace)
	if err != nil || flags.NoOut {
		return false
	}

	// Check operator is running
	opStatus, err := client.GetDeploymentStatus(ctx, cfg.Namespace, operatorDeploymentName)
	if err != nil || opStatus.Replicas != 1 || opStatus.ReadyReplicas != 1 {
		return false
	}

	return true
}

// AllDeploymentsScaledDown checks if all deployments are fully scaled down.
// Returns true if the list is empty or all deployments have:
//   - Spec.Replicas == 0 (or nil, treated as 0 for this check)
//   - Status.ReadyReplicas == 0
//
// This is stricter than just checking spec - it ensures pods are actually gone.
func AllDeploymentsScaledDown(deployments []appsv1.Deployment) bool {
	for _, d := range deployments {
		// Check spec replicas
		specReplicas := int32(1) // default if nil
		if d.Spec.Replicas != nil {
			specReplicas = *d.Spec.Replicas
		}
		if specReplicas > 0 {
			return false
		}

		// Check ready replicas (stricter - ensures pods are actually gone)
		if d.Status.ReadyReplicas > 0 {
			return false
		}
	}
	return true
}
