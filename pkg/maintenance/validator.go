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

	// Check 3: Operator namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Kubernetes.RookOperatorNamespace); err != nil {
		results.addResult("Operator namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Kubernetes.RookOperatorNamespace))
	} else {
		results.addResult("Operator namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Kubernetes.RookOperatorNamespace))
	}

	// Check 4: Cluster namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Kubernetes.RookClusterNamespace); err != nil {
		results.addResult("Cluster namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Kubernetes.RookClusterNamespace))
	} else {
		results.addResult("Cluster namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Kubernetes.RookClusterNamespace))
	}

	// Check 5: rook-ceph-tools deployment exists and is ready
	if err := validateRookToolsDeployment(ctx, client, cfg.Kubernetes.RookClusterNamespace); err != nil {
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

	// Check 3: Operator namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Kubernetes.RookOperatorNamespace); err != nil {
		results.addResult("Operator namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Kubernetes.RookOperatorNamespace))
	} else {
		results.addResult("Operator namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Kubernetes.RookOperatorNamespace))
	}

	// Check 4: Cluster namespace exists
	if err := validateNamespaceExists(ctx, client, cfg.Kubernetes.RookClusterNamespace); err != nil {
		results.addResult("Cluster namespace", false, err, fmt.Sprintf("Namespace %s not found", cfg.Kubernetes.RookClusterNamespace))
	} else {
		results.addResult("Cluster namespace", true, nil, fmt.Sprintf("Namespace %s exists", cfg.Kubernetes.RookClusterNamespace))
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

	// Required permissions for down phase
	permissions := []struct {
		resource string
		verb     string
		check    string
	}{
		{"nodes", "patch", "RBAC: cordon nodes"},
		{"deployments", "get", "RBAC: get deployments"},
		{"deployments", "update", "RBAC: scale deployments"},
		{"pods", "list", "RBAC: list pods"},
		{"pods/exec", "create", "RBAC: exec in pods"},
	}

	for _, perm := range permissions {
		allowed, err := checkPermission(ctx, client, perm.resource, perm.verb, cfg.Kubernetes.RookClusterNamespace)
		if err != nil {
			// Best-effort check - don't fail on errors
			results = append(results, ValidationResult{
				Check:   perm.check,
				Passed:  true, // Assume allowed if we can't check
				Error:   nil,
				Message: fmt.Sprintf("Unable to verify %s permission (assuming allowed)", perm.verb),
			})
		} else if !allowed {
			results = append(results, ValidationResult{
				Check:   perm.check,
				Passed:  false,
				Error:   fmt.Errorf("missing %s permission on %s", perm.verb, perm.resource),
				Message: fmt.Sprintf("Missing %s permission on %s - contact cluster admin", perm.verb, perm.resource),
			})
		} else {
			results = append(results, ValidationResult{
				Check:   perm.check,
				Passed:  true,
				Error:   nil,
				Message: fmt.Sprintf("Permission %s on %s verified", perm.verb, perm.resource),
			})
		}
	}

	return results
}

// checkPermission uses SelfSubjectAccessReview to check if current user has permission
func checkPermission(ctx context.Context, client *k8s.Client, resource, verb, namespace string) (bool, error) {
	sar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Resource:  resource,
			},
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
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n", status, r.Check, r.Message))
	}
	if vr.AllPassed {
		sb.WriteString("\nAll checks passed - ready to proceed\n")
	} else {
		sb.WriteString("\nSome checks failed - resolve issues before proceeding\n")
	}
	return sb.String()
}

// FailedChecks returns a slice of failed validation results
func (vr *ValidationResults) FailedChecks() []ValidationResult {
	failed := make([]ValidationResult, 0)
	for _, r := range vr.Results {
		if !r.Passed {
			failed = append(failed, r)
		}
	}
	return failed
}
