package maintenance

import (
	"context"
	"fmt"
	"strings"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	authv1 "k8s.io/api/authorization/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidationResult holds the result of a single pre-flight check
type ValidationResult struct {
	Check   string
	Passed  bool
	Error   error
	Message string
}

// ValidationResults holds all validation results
type ValidationResults struct {
	Results   []ValidationResult
	AllPassed bool
}

// ValidateDownPhase performs comprehensive pre-flight checks before down phase
func ValidateDownPhase(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string) (*ValidationResults, error) {
	results := &ValidationResults{
		Results:   make([]ValidationResult, 0),
		AllPassed: true,
	}

	// Check 1: Cluster connectivity (implicit - client creation validates this)
	results.addResult("Cluster connectivity", true, nil, "Successfully connected to Kubernetes API")

	// Check 2: Node exists
	if err := validateNodeExists(ctx, client, nodeName); err != nil {
		results.addResult("Node existence", false, err, fmt.Sprintf("Node %s not found", nodeName))
	} else {
		results.addResult("Node existence", true, nil, fmt.Sprintf("Node %s exists", nodeName))
	}

	// Check 3: Namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Namespace); err != nil {
		results.addResult("Namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Namespace))
	} else {
		results.addResult("Namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Namespace))
	}

	// Check 4: rook-ceph-tools deployment exists and is ready
	if err := validateRookToolsDeployment(ctx, client, cfg.Namespace); err != nil {
		results.addResult("rook-ceph-tools deployment", false, err, "rook-ceph-tools deployment not ready")
	} else {
		results.addResult("rook-ceph-tools deployment", true, nil, "rook-ceph-tools deployment is ready")
	}

	// Check 6: RBAC permissions (best-effort)
	rbacResults := validateRBACPermissions(ctx, client, cfg)
	for _, r := range rbacResults {
		results.addResult(r.Check, r.Passed, r.Error, r.Message)
	}

	return results, nil
}

// ValidateUpPhase performs pre-flight checks before up phase
func ValidateUpPhase(ctx context.Context, client *k8s.Client, cfg config.Config, nodeName string) (*ValidationResults, error) {
	results := &ValidationResults{
		Results:   make([]ValidationResult, 0),
		AllPassed: true,
	}

	// Check 1: Cluster connectivity
	results.addResult("Cluster connectivity", true, nil, "Successfully connected to Kubernetes API")

	// Check 2: Node exists
	if err := validateNodeExists(ctx, client, nodeName); err != nil {
		results.addResult("Node existence", false, err, fmt.Sprintf("Node %s not found", nodeName))
	} else {
		results.addResult("Node existence", true, nil, fmt.Sprintf("Node %s exists", nodeName))
	}

	// Check 3: Namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Namespace); err != nil {
		results.addResult("Namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Namespace))
	} else {
		results.addResult("Namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Namespace))
	}

	return results, nil
}

// validateNodeExists checks if the specified node exists in the cluster
func validateNodeExists(ctx context.Context, client *k8s.Client, nodeName string) error {
	_, err := client.GetNode(ctx, nodeName)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("node not found - verify node name is correct")
		}
		return fmt.Errorf("failed to get node: %w", err)
	}
	return nil
}

// validateNamespaceExists checks if the specified namespace exists
func validateNamespaceExists(ctx context.Context, client *k8s.Client, namespace string) error {
	_, err := client.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("namespace not found - create it or update configuration")
		}
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	return nil
}

// validateRookToolsDeployment checks if rook-ceph-tools deployment exists and is ready
func validateRookToolsDeployment(ctx context.Context, client *k8s.Client, namespace string) error {
	deployment, err := client.GetDeployment(ctx, namespace, "rook-ceph-tools")
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("deployment not found - deploy rook-ceph-tools to continue")
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Check if deployment has at least one ready replica
	if deployment.Status.ReadyReplicas == 0 {
		return fmt.Errorf("deployment has no ready replicas - wait for rook-ceph-tools to become ready")
	}

	return nil
}

// validateRBACPermissions performs best-effort validation of required permissions
func validateRBACPermissions(ctx context.Context, client *k8s.Client, cfg config.Config) []ValidationResult {
	results := make([]ValidationResult, 0)

	// Required permissions for maintenance operations
	permissions := []authv1.ResourceAttributes{
		// Cluster-scoped: nodes (patch for cordon/uncordon, get for validation)
		{Resource: "nodes", Verb: "patch"},
		{Resource: "nodes", Verb: "get"},
		// Namespaced: deployments in cluster namespace (for node-pinned deployments)
		{Group: "apps", Resource: "deployments", Verb: "get", Namespace: cfg.Namespace},
		{Group: "apps", Resource: "deployments", Verb: "list", Namespace: cfg.Namespace},
		// Scale subresource for scaling deployments (least-privilege)
		{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "get", Namespace: cfg.Namespace},
		{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "update", Namespace: cfg.Namespace},
		// Namespaced: deployments in operator namespace (for rook-ceph-operator)
		{Group: "apps", Resource: "deployments", Verb: "get", Namespace: cfg.Namespace},
		{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "get", Namespace: cfg.Namespace},
		{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "update", Namespace: cfg.Namespace},
		// Namespaced: pods (for exec to rook-ceph-tools)
		{Resource: "pods", Verb: "list", Namespace: cfg.Namespace},
		{Resource: "pods", Subresource: "exec", Verb: "create", Namespace: cfg.Namespace},
	}

	for _, perm := range permissions {
		checkName := formatPermissionCheck(&perm)
		allowed, err := checkPermission(ctx, client, &perm)
		if err != nil {
			// Best-effort check - don't fail on errors
			results = append(results, ValidationResult{
				Check:   checkName,
				Passed:  true, // Assume allowed if we can't check
				Error:   nil,
				Message: "Unable to verify (assuming allowed)",
			})
		} else if !allowed {
			results = append(results, ValidationResult{
				Check:   checkName,
				Passed:  false,
				Error:   fmt.Errorf("missing permission: %s", checkName),
				Message: "Permission denied - contact cluster admin",
			})
		} else {
			results = append(results, ValidationResult{
				Check:   checkName,
				Passed:  true,
				Error:   nil,
				Message: "Permission verified",
			})
		}
	}

	return results
}

// formatPermissionCheck generates a display name from ResourceAttributes
func formatPermissionCheck(ra *authv1.ResourceAttributes) string {
	resource := ra.Resource
	if ra.Subresource != "" {
		resource += "/" + ra.Subresource
	}
	if ra.Group != "" {
		resource = ra.Group + "/" + resource
	}
	scope := "cluster"
	if ra.Namespace != "" {
		scope = ra.Namespace
	}
	return fmt.Sprintf("%s %s [%s]", ra.Verb, resource, scope)
}

// checkPermission uses SelfSubjectAccessReview to check if current user has permission
func checkPermission(ctx context.Context, client *k8s.Client, ra *authv1.ResourceAttributes) (bool, error) {
	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: ra,
		},
	}

	result, err := client.Clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return result.Status.Allowed, nil
}

// addResult adds a validation result and updates AllPassed flag
func (vr *ValidationResults) addResult(check string, passed bool, err error, message string) {
	vr.Results = append(vr.Results, ValidationResult{
		Check:   check,
		Passed:  passed,
		Error:   err,
		Message: message,
	})
	if !passed {
		vr.AllPassed = false
	}
}

// String returns a formatted string representation of validation results
func (vr *ValidationResults) String() string {
	var sb strings.Builder
	sb.WriteString("Pre-flight validation results:\n")
	for _, r := range vr.Results {
		status := "✓"
		if !r.Passed {
			status = "✗"
		}
		fmt.Fprintf(&sb, "  %s %s: %s\n", status, r.Check, r.Message)
	}
	if vr.AllPassed {
		sb.WriteString("\nAll checks passed - ready to proceed\n")
	} else {
		sb.WriteString("\nSome checks failed - resolve issues before proceeding\n")
	}
	return sb.String()
}

// MaintenanceStatus describes the maintenance state of a node
type MaintenanceStatus struct {
	// NodeName is the name of the node
	NodeName string
	// Cordoned indicates the node is cordoned (unschedulable)
	Cordoned bool
	// HasScaledDownDeployments indicates the node has rook-ceph deployments scaled to 0
	HasScaledDownDeployments bool
}

// OtherNodesMaintenanceInfo holds information about other nodes currently in maintenance
type OtherNodesMaintenanceInfo struct {
	// NodesInMaintenance lists nodes that appear to be in maintenance
	NodesInMaintenance []MaintenanceStatus
	// NoOutFlagSet indicates the Ceph noout flag is already set
	NoOutFlagSet bool
}

// HasWarning returns true if there are any warnings to display
func (info *OtherNodesMaintenanceInfo) HasWarning() bool {
	return len(info.NodesInMaintenance) > 0 || info.NoOutFlagSet
}

// WarningMessage returns a formatted warning message about nodes in maintenance
func (info *OtherNodesMaintenanceInfo) WarningMessage() string {
	if !info.HasWarning() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("WARNING: Another node may be in maintenance!\n\n")

	if info.NoOutFlagSet {
		sb.WriteString("  • Ceph 'noout' flag is already set\n")
	}

	for _, node := range info.NodesInMaintenance {
		reasons := []string{}
		if node.Cordoned {
			reasons = append(reasons, "cordoned")
		}
		if node.HasScaledDownDeployments {
			reasons = append(reasons, "has scaled-down deployments")
		}
		fmt.Fprintf(&sb, "  • Node '%s' is %s\n", node.NodeName, strings.Join(reasons, " and "))
	}

	sb.WriteString("\nTaking multiple nodes down simultaneously risks Ceph cluster availability\n")
	sb.WriteString("and data redundancy. Only proceed if you understand the risks.\n")

	return sb.String()
}

// CheckOtherNodesInMaintenance checks if any other nodes are currently in maintenance.
// It returns information about:
// - Other cordoned nodes (excluding the target node)
// - Whether the noout flag is already set (indicating ongoing maintenance)
// - Other nodes with scaled-down rook-ceph deployments
func CheckOtherNodesInMaintenance(
	ctx context.Context,
	client *k8s.Client,
	cfg config.Config,
	targetNodeName string,
) (*OtherNodesMaintenanceInfo, error) {
	info := &OtherNodesMaintenanceInfo{
		NodesInMaintenance: make([]MaintenanceStatus, 0),
	}

	// Check Ceph noout flag (best-effort - don't fail if ceph is unreachable)
	cephFlags, cephErr := client.GetCephFlags(ctx, cfg.Namespace)
	if cephErr == nil && cephFlags != nil {
		info.NoOutFlagSet = cephFlags.NoOut
	}

	// Get all nodes
	nodes, err := client.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Check each node (except target) for maintenance indicators
	for i := range nodes {
		node := &nodes[i]
		if node.Name == targetNodeName {
			continue
		}

		status := MaintenanceStatus{
			NodeName: node.Name,
			Cordoned: node.Spec.Unschedulable,
		}

		// Check if node has scaled-down deployments
		if status.Cordoned {
			// Only check for scaled-down deployments if the node is cordoned
			// to avoid expensive API calls for every node
			deployments, listErr := client.ListNodePinnedDeployments(ctx, cfg.Namespace, node.Name)
			if listErr == nil {
				for j := range deployments {
					dep := &deployments[j]
					if dep.Spec.Replicas != nil && *dep.Spec.Replicas == 0 {
						status.HasScaledDownDeployments = true
						break
					}
				}
			}
		}

		// Add to list if node shows signs of maintenance
		if status.Cordoned || status.HasScaledDownDeployments {
			info.NodesInMaintenance = append(info.NodesInMaintenance, status)
		}
	}

	return info, nil
}
