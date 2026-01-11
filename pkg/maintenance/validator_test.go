package maintenance

import (
	"context"
	"testing"

	"github.com/andri/crook/pkg/config"
	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktest "k8s.io/client-go/testing"
)

func TestValidateDownPhase_AllChecksPassed(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset with all required resources
	client := createTestClient(
		createNode("worker-01"),
		createNamespace("rook-ceph"),
		createReadyRookToolsDeployment("rook-ceph"),
	)

	// Add RBAC check support
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(_ ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	cfg := config.DefaultConfig()

	results, err := ValidateDownPhase(ctx, client, cfg, "worker-01")
	if err != nil {
		t.Fatalf("ValidateDownPhase failed: %v", err)
	}

	if !results.AllPassed {
		t.Errorf("Expected all checks to pass, but got failures: %v", results.FailedChecks())
	}
}

func TestValidateDownPhase_NodeNotFound(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset without the target node
	client := createTestClient(
		createNamespace("rook-ceph"),
		createReadyRookToolsDeployment("rook-ceph"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateDownPhase(ctx, client, cfg, "nonexistent-node")
	if err != nil {
		t.Fatalf("ValidateDownPhase failed: %v", err)
	}

	if results.AllPassed {
		t.Error("Expected node existence check to fail")
	}

	// Verify that the specific check failed
	nodeCheckFailed := false
	for _, r := range results.Results {
		if r.Check == "Node existence" && !r.Passed {
			nodeCheckFailed = true
			break
		}
	}
	if !nodeCheckFailed {
		t.Error("Expected node existence check to fail")
	}
}

func TestValidateDownPhase_NamespaceMissing(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset without the rook-ceph namespace
	client := createTestClient(
		createNode("worker-01"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateDownPhase(ctx, client, cfg, "worker-01")
	if err != nil {
		t.Fatalf("ValidateDownPhase failed: %v", err)
	}

	if results.AllPassed {
		t.Error("Expected namespace check to fail")
	}

	// Verify that namespace check failed
	namespaceCheckFailed := false
	for _, r := range results.Results {
		if r.Check == "Namespace" && !r.Passed {
			namespaceCheckFailed = true
			break
		}
	}
	if !namespaceCheckFailed {
		t.Error("Expected namespace check to fail")
	}
}

func TestValidateDownPhase_RookToolsNotReady(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset with unready rook-ceph-tools
	client := createTestClient(
		createNode("worker-01"),
		createNamespace("rook-ceph"),
		createUnreadyRookToolsDeployment("rook-ceph"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateDownPhase(ctx, client, cfg, "worker-01")
	if err != nil {
		t.Fatalf("ValidateDownPhase failed: %v", err)
	}

	if results.AllPassed {
		t.Error("Expected rook-ceph-tools check to fail")
	}

	// Verify that the specific check failed
	toolsCheckFailed := false
	for _, r := range results.Results {
		if r.Check == "rook-ceph-tools deployment" && !r.Passed {
			toolsCheckFailed = true
			break
		}
	}
	if !toolsCheckFailed {
		t.Error("Expected rook-ceph-tools deployment check to fail")
	}
}

func TestValidateDownPhase_RookToolsNotFound(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset without rook-ceph-tools
	client := createTestClient(
		createNode("worker-01"),
		createNamespace("rook-ceph"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateDownPhase(ctx, client, cfg, "worker-01")
	if err != nil {
		t.Fatalf("ValidateDownPhase failed: %v", err)
	}

	if results.AllPassed {
		t.Error("Expected rook-ceph-tools check to fail")
	}
}

func TestValidateUpPhase_AllChecksPassed(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset with all required resources
	client := createTestClient(
		createNode("worker-01"),
		createNamespace("rook-ceph"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateUpPhase(ctx, client, cfg, "worker-01")
	if err != nil {
		t.Fatalf("ValidateUpPhase failed: %v", err)
	}

	if !results.AllPassed {
		t.Errorf("Expected all checks to pass, but got failures: %v", results.FailedChecks())
	}
}

func TestValidateUpPhase_NodeNotFound(t *testing.T) {
	ctx := context.Background()

	// Create fake clientset without the target node
	client := createTestClient(
		createNamespace("rook-ceph"),
	)

	cfg := config.DefaultConfig()

	results, err := ValidateUpPhase(ctx, client, cfg, "nonexistent-node")
	if err != nil {
		t.Fatalf("ValidateUpPhase failed: %v", err)
	}

	if results.AllPassed {
		t.Error("Expected node existence check to fail")
	}
}

func TestValidationResults_String(t *testing.T) {
	results := &ValidationResults{
		Results: []ValidationResult{
			{Check: "Test 1", Passed: true, Message: "Success"},
			{Check: "Test 2", Passed: false, Message: "Failed"},
		},
		AllPassed: false,
	}

	str := results.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// Verify it contains check names
	if !containsSubstring(str, "Test 1") || !containsSubstring(str, "Test 2") {
		t.Error("String representation should contain check names")
	}
}

func TestValidationResults_FailedChecks(t *testing.T) {
	results := &ValidationResults{
		Results: []ValidationResult{
			{Check: "Test 1", Passed: true, Message: "Success"},
			{Check: "Test 2", Passed: false, Message: "Failed"},
			{Check: "Test 3", Passed: false, Message: "Failed"},
		},
		AllPassed: false,
	}

	failed := results.FailedChecks()
	if len(failed) != 2 {
		t.Errorf("Expected 2 failed checks, got %d", len(failed))
	}
}

func TestCheckPermission_Allowed(t *testing.T) {
	ctx := context.Background()

	client := createTestClient()
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(_ ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	ra := &authv1.ResourceAttributes{
		Resource: "nodes",
		Verb:     "patch",
	}
	allowed, err := checkPermission(ctx, client, ra)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if !allowed {
		t.Error("Expected permission to be allowed")
	}
}

func TestCheckPermission_Denied(t *testing.T) {
	ctx := context.Background()

	client := createTestClient()
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(_ ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: false,
			},
		}, nil
	})

	ra := &authv1.ResourceAttributes{
		Resource: "nodes",
		Verb:     "patch",
	}
	allowed, err := checkPermission(ctx, client, ra)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if allowed {
		t.Error("Expected permission to be denied")
	}
}

func TestCheckPermission_WithGroupAndSubresource(t *testing.T) {
	ctx := context.Background()

	client := createTestClient()

	var capturedSAR *authv1.SelfSubjectAccessReview
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(action ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		createAction, ok := action.(ktest.CreateAction)
		if ok {
			capturedSAR, _ = createAction.GetObject().(*authv1.SelfSubjectAccessReview)
		}
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	ra := &authv1.ResourceAttributes{
		Group:       "apps",
		Resource:    "deployments",
		Subresource: "scale",
		Verb:        "update",
		Namespace:   "rook-ceph",
	}
	_, err := checkPermission(ctx, client, ra)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	// Verify the SAR was constructed correctly
	if capturedSAR == nil {
		t.Fatal("SAR was not captured")
	}

	attrs := capturedSAR.Spec.ResourceAttributes
	if attrs.Group != "apps" {
		t.Errorf("Expected group 'apps', got '%s'", attrs.Group)
	}
	if attrs.Resource != "deployments" {
		t.Errorf("Expected resource 'deployments', got '%s'", attrs.Resource)
	}
	if attrs.Subresource != "scale" {
		t.Errorf("Expected subresource 'scale', got '%s'", attrs.Subresource)
	}
	if attrs.Verb != "update" {
		t.Errorf("Expected verb 'update', got '%s'", attrs.Verb)
	}
	if attrs.Namespace != "rook-ceph" {
		t.Errorf("Expected namespace 'rook-ceph', got '%s'", attrs.Namespace)
	}
}

func TestCheckPermission_ClusterScoped(t *testing.T) {
	ctx := context.Background()

	client := createTestClient()

	var capturedSAR *authv1.SelfSubjectAccessReview
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(action ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		createAction, ok := action.(ktest.CreateAction)
		if ok {
			capturedSAR, _ = createAction.GetObject().(*authv1.SelfSubjectAccessReview)
		}
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	// Nodes are cluster-scoped - namespace should be empty
	ra := &authv1.ResourceAttributes{
		Resource: "nodes",
		Verb:     "patch",
	}
	_, err := checkPermission(ctx, client, ra)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	// Verify the SAR was constructed correctly
	if capturedSAR == nil {
		t.Fatal("SAR was not captured")
	}

	attrs := capturedSAR.Spec.ResourceAttributes
	if attrs.Namespace != "" {
		t.Errorf("Expected empty namespace for cluster-scoped resource, got '%s'", attrs.Namespace)
	}
	if attrs.Resource != "nodes" {
		t.Errorf("Expected resource 'nodes', got '%s'", attrs.Resource)
	}
}

func TestCheckPermission_PodExecSubresource(t *testing.T) {
	ctx := context.Background()

	client := createTestClient()

	var capturedSAR *authv1.SelfSubjectAccessReview
	client.Clientset.(*fake.Clientset).PrependReactor("create", "selfsubjectaccessreviews", func(action ktest.Action) (bool, runtime.Object, error) { //nolint:errcheck // test helper
		createAction, ok := action.(ktest.CreateAction)
		if ok {
			capturedSAR, _ = createAction.GetObject().(*authv1.SelfSubjectAccessReview)
		}
		return true, &authv1.SelfSubjectAccessReview{
			Status: authv1.SubjectAccessReviewStatus{
				Allowed: true,
			},
		}, nil
	})

	// pods/exec should be resource=pods, subresource=exec
	ra := &authv1.ResourceAttributes{
		Resource:    "pods",
		Subresource: "exec",
		Verb:        "create",
		Namespace:   "rook-ceph",
	}
	_, err := checkPermission(ctx, client, ra)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	// Verify the SAR was constructed correctly
	if capturedSAR == nil {
		t.Fatal("SAR was not captured")
	}

	attrs := capturedSAR.Spec.ResourceAttributes
	if attrs.Resource != "pods" {
		t.Errorf("Expected resource 'pods', got '%s'", attrs.Resource)
	}
	if attrs.Subresource != "exec" {
		t.Errorf("Expected subresource 'exec', got '%s'", attrs.Subresource)
	}
}

func TestFormatPermissionCheck(t *testing.T) {
	tests := []struct {
		name     string
		ra       authv1.ResourceAttributes
		expected string
	}{
		{
			name:     "cluster-scoped resource",
			ra:       authv1.ResourceAttributes{Resource: "nodes", Verb: "patch"},
			expected: "patch nodes [cluster]",
		},
		{
			name:     "namespaced resource",
			ra:       authv1.ResourceAttributes{Resource: "pods", Verb: "list", Namespace: "rook-ceph"},
			expected: "list pods [rook-ceph]",
		},
		{
			name:     "resource with group",
			ra:       authv1.ResourceAttributes{Group: "apps", Resource: "deployments", Verb: "get", Namespace: "default"},
			expected: "get apps/deployments [default]",
		},
		{
			name:     "resource with subresource",
			ra:       authv1.ResourceAttributes{Resource: "pods", Subresource: "exec", Verb: "create", Namespace: "rook-ceph"},
			expected: "create pods/exec [rook-ceph]",
		},
		{
			name:     "resource with group and subresource",
			ra:       authv1.ResourceAttributes{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "update", Namespace: "rook-ceph"},
			expected: "update apps/deployments/scale [rook-ceph]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPermissionCheck(&tt.ra)
			if result != tt.expected {
				t.Errorf("formatPermissionCheck() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Helper functions

func createTestClient(objects ...runtime.Object) *k8s.Client {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(objects...)
	return &k8s.Client{
		Clientset: clientset,
	}
}

//nolint:unparam // test helper designed for flexibility
func createNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

//nolint:unparam // test helper designed for flexibility
func createNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func createReadyRookToolsDeployment(namespace string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
		},
	}
}

func createUnreadyRookToolsDeployment(namespace string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 0,
		},
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstringAt(s, substr, 0))
}

func containsSubstringAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
