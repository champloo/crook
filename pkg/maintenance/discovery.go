package maintenance

import (
	"context"
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentInfo holds information about a discovered deployment
type DeploymentInfo struct {
	Namespace string
	Name      string
}

// DiscoverDeployments finds all deployments with pods on the target node
// matching the configured prefixes
func DiscoverDeployments(
	ctx context.Context,
	client *k8s.Client,
	nodeName string,
	namespace string,
	prefixes []string,
) ([]appsv1.Deployment, error) {
	// Get all pods on the node
	pods, err := client.ListPodsOnNode(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	// Track unique deployments using a map (key: namespace/name)
	uniqueDeployments := make(map[string]*DeploymentInfo)

	// Process each pod to find its deployment owner
	for _, pod := range pods {
		// Filter by namespace if specified
		if namespace != "" && pod.Namespace != namespace {
			continue
		}

		// Get ownership chain
		chain, err := client.GetOwnerChain(ctx, &pod)
		if err != nil {
			// Log but don't fail - some pods may not have owners
			continue
		}

		// Check if pod is owned by a deployment
		if chain.Deployment == nil {
			// Pod not owned by a deployment (could be StatefulSet, DaemonSet, etc.)
			continue
		}

		// Check if deployment name matches any prefix
		if !matchesPrefix(chain.Deployment.Name, prefixes) {
			continue
		}

		// Add to unique deployments
		key := fmt.Sprintf("%s/%s", pod.Namespace, chain.Deployment.Name)
		if _, exists := uniqueDeployments[key]; !exists {
			uniqueDeployments[key] = &DeploymentInfo{
				Namespace: pod.Namespace,
				Name:      chain.Deployment.Name,
			}
		}
	}

	// Fetch full deployment objects for each unique deployment
	deployments := make([]appsv1.Deployment, 0, len(uniqueDeployments))
	for _, info := range uniqueDeployments {
		deployment, err := client.GetDeployment(ctx, info.Namespace, info.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment %s/%s: %w", info.Namespace, info.Name, err)
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

// matchesPrefix checks if a deployment name matches any of the given prefixes
func matchesPrefix(name string, prefixes []string) bool {
	// If no prefixes specified, match all
	if len(prefixes) == 0 {
		return true
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// GroupDeploymentsByPrefix groups deployments by their prefix for ordered processing
// Returns a map where keys are prefixes and values are deployments matching that prefix
func GroupDeploymentsByPrefix(deployments []appsv1.Deployment, prefixes []string) map[string][]appsv1.Deployment {
	grouped := make(map[string][]appsv1.Deployment)

	for _, deployment := range deployments {
		for _, prefix := range prefixes {
			if strings.HasPrefix(deployment.Name, prefix) {
				grouped[prefix] = append(grouped[prefix], deployment)
				break // Only match first prefix
			}
		}
	}

	return grouped
}

// OrderDeploymentsForDown returns deployments ordered for safe down phase
// Order: rook-ceph-osd (first), rook-ceph-mon, rook-ceph-exporter, rook-ceph-crashcollector (last)
func OrderDeploymentsForDown(deployments []appsv1.Deployment) []appsv1.Deployment {
	downOrder := []string{
		"rook-ceph-osd",
		"rook-ceph-mon",
		"rook-ceph-exporter",
		"rook-ceph-crashcollector",
	}

	return orderByPrefixes(deployments, downOrder)
}

// OrderDeploymentsForUp returns deployments ordered for safe up phase
// Order: rook-ceph-mon (first), rook-ceph-osd, rook-ceph-exporter, rook-ceph-crashcollector (last)
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
