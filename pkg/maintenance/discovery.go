package maintenance

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
)

// OrderDeploymentsForDown returns deployments ordered for safe down phase.
// Order: rook-ceph-osd (first), rook-ceph-mon, rook-ceph-exporter, rook-ceph-crashcollector (last).
//
// MON handling asymmetry note: Unlike the UP phase, the DOWN phase does not
// require special MON separation or quorum checks. This is because:
//   - The noout flag is set BEFORE scaling down, preventing Ceph rebalancing
//   - OSDs are scaled down first to ensure clean shutdown before MONs
//   - No quorum check is needed since the cluster is going offline anyway
//
// The UP phase (see restoreDeployments in up.go) explicitly separates MONs
// and waits for quorum before scaling OSDs, since OSDs require MON quorum
// to recover properly.
func OrderDeploymentsForDown(deployments []appsv1.Deployment) []appsv1.Deployment {
	downOrder := []string{
		"rook-ceph-osd",
		"rook-ceph-mon",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	return orderByPrefixes(deployments, downOrder)
}

// OrderDeploymentsForUp returns non-MON deployments ordered for the UP phase.
// Order: rook-ceph-osd (first), rook-ceph-exporter, rook-ceph-crashcollector (last).
//
// IMPORTANT: This function is called on non-MON deployments only. MON deployments
// are separated and scaled BEFORE this ordering is applied (see restoreDeployments
// in up.go). The caller (restoreDeployments) handles MONs explicitly:
//  1. Scale up MON deployments first
//  2. Wait for Ceph monitor quorum
//  3. Then call OrderDeploymentsForUp for remaining deployments
//
// The "rook-ceph-mon" prefix in the order list below is kept for defensive
// ordering in case MONs are inadvertently passed to this function.
func OrderDeploymentsForUp(deployments []appsv1.Deployment) []appsv1.Deployment {
	upOrder := []string{
		"rook-ceph-mon",
		"rook-ceph-osd",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	return orderByPrefixes(deployments, upOrder)
}

// orderByPrefixes returns deployments ordered by the given prefix order
// Deployments not matching any prefix are appended at the end
func orderByPrefixes(deployments []appsv1.Deployment, prefixOrder []string) []appsv1.Deployment {
	ordered := make([]appsv1.Deployment, 0, len(deployments))

	// Add deployments in prefix order
	for _, prefix := range prefixOrder {
		for _, deployment := range deployments {
			if strings.HasPrefix(deployment.Name, prefix) {
				ordered = append(ordered, deployment)
			}
		}
	}

	// Add any remaining deployments that didn't match prefixes
	for _, deployment := range deployments {
		found := false
		for _, existing := range ordered {
			if existing.Namespace == deployment.Namespace && existing.Name == deployment.Name {
				found = true
				break
			}
		}
		if !found {
			ordered = append(ordered, deployment)
		}
	}

	return ordered
}

// GetDeploymentNames extracts deployment names from a list of deployments
func GetDeploymentNames(deployments []appsv1.Deployment) []string {
	names := make([]string, len(deployments))
	for i, deployment := range deployments {
		names[i] = fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	}
	return names
}
