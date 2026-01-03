package k8s

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestScaleDeployment(t *testing.T) {
	ctx := context.Background()

	// Create a fake clientset with a test deployment
	initialReplicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	// Scale deployment to 0
	err := client.ScaleDeployment(ctx, "default", "test-deployment", 0)
	if err != nil {
		t.Fatalf("failed to scale deployment: %v", err)
	}

	// Verify the deployment was scaled
	updated, err := clientset.AppsV1().Deployments("default").Get(ctx, "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	if *updated.Spec.Replicas != 0 {
		t.Errorf("expected replicas to be 0, got %d", *updated.Spec.Replicas)
	}
}

func TestScaleDeployment_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromClientset(clientset)

	err := client.ScaleDeployment(ctx, "default", "nonexistent", 1)
	if err == nil {
		t.Error("expected error when scaling nonexistent deployment, got nil")
	}
}

func TestGetDeploymentStatus(t *testing.T) {
	ctx := context.Background()

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          3,
			ReadyReplicas:     2,
			AvailableReplicas: 2,
			UpdatedReplicas:   3,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	status, err := client.GetDeploymentStatus(ctx, "default", "test-deployment")
	if err != nil {
		t.Fatalf("failed to get deployment status: %v", err)
	}

	if status.Name != "test-deployment" {
		t.Errorf("expected name 'test-deployment', got %s", status.Name)
	}
	if status.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %s", status.Namespace)
	}
	if status.Replicas != 3 {
		t.Errorf("expected replicas 3, got %d", status.Replicas)
	}
	if status.ReadyReplicas != 2 {
		t.Errorf("expected ready replicas 2, got %d", status.ReadyReplicas)
	}
	if status.AvailableReplicas != 2 {
		t.Errorf("expected available replicas 2, got %d", status.AvailableReplicas)
	}
}

func TestListDeploymentsInNamespace(t *testing.T) {
	ctx := context.Background()

	replicas := int32(1)
	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-1",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	deployment2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-2",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
	deployment3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-3",
			Namespace: "other",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment1, deployment2, deployment3)
	client := newClientFromClientset(clientset)

	deployments, err := client.ListDeploymentsInNamespace(ctx, "default")
	if err != nil {
		t.Fatalf("failed to list deployments: %v", err)
	}

	if len(deployments) != 2 {
		t.Errorf("expected 2 deployments in default namespace, got %d", len(deployments))
	}

	// Verify names
	names := make(map[string]bool)
	for _, d := range deployments {
		names[d.Name] = true
	}

	if !names["deployment-1"] || !names["deployment-2"] {
		t.Errorf("expected deployment-1 and deployment-2, got %v", names)
	}
}

func TestFilterDeploymentsByPrefix(t *testing.T) {
	replicas := int32(1)
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-osd-0"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-mon-a"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other-deployment"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "rook-ceph-exporter"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
	}

	prefixes := []string{"rook-ceph-osd", "rook-ceph-mon", "rook-ceph-exporter"}
	filtered := FilterDeploymentsByPrefix(deployments, prefixes)

	if len(filtered) != 3 {
		t.Errorf("expected 3 filtered deployments, got %d", len(filtered))
	}

	// Verify the right ones were filtered
	for _, d := range filtered {
		if d.Name == "other-deployment" {
			t.Errorf("other-deployment should not be in filtered results")
		}
	}
}

func TestFilterDeploymentsByPrefix_EmptyPrefixes(t *testing.T) {
	replicas := int32(1)
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "deployment-1"},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		},
	}

	filtered := FilterDeploymentsByPrefix(deployments, []string{})
	if len(filtered) != len(deployments) {
		t.Errorf("expected all deployments when prefixes is empty, got %d", len(filtered))
	}
}

func TestWaitForReplicas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	ctx := context.Background()

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	opts := WaitForReplicasOptions{
		PollInterval: 100 * time.Millisecond,
		Timeout:      1 * time.Second,
	}

	// This should succeed immediately since replicas already match
	err := client.WaitForReplicas(ctx, "default", "test-deployment", 3, opts)
	if err != nil {
		t.Fatalf("failed to wait for replicas: %v", err)
	}
}

func TestWaitForReplicas_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	ctx := context.Background()

	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	clientset := fake.NewClientset(deployment)
	client := newClientFromClientset(clientset)

	opts := WaitForReplicasOptions{
		PollInterval: 100 * time.Millisecond,
		Timeout:      300 * time.Millisecond,
	}

	// This should timeout since we're waiting for 5 replicas but deployment has 1
	err := client.WaitForReplicas(ctx, "default", "test-deployment", 5, opts)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestDefaultWaitOptions(t *testing.T) {
	opts := DefaultWaitOptions()

	if opts.PollInterval != 5*time.Second {
		t.Errorf("expected poll interval 5s, got %v", opts.PollInterval)
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("expected timeout 5m, got %v", opts.Timeout)
	}
}
