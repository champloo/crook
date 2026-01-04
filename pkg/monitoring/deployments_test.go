package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMonitorDeployment(t *testing.T) {
	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		expectedStatus DeploymentHealthStatus
		expectedColor  string
	}{
		{
			name: "healthy deployment",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentAvailable, Status: "True"},
					},
				},
			},
			expectedStatus: DeploymentHealthy,
			expectedColor:  "green",
		},
		{
			name: "unavailable deployment",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          0,
					ReadyReplicas:     0,
					AvailableReplicas: 0,
					UpdatedReplicas:   0,
				},
			},
			expectedStatus: DeploymentUnavailable,
			expectedColor:  "red",
		},
		{
			name: "scaling deployment",
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(5),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: "True", Reason: "NewReplicaSetAvailable"},
					},
				},
			},
			expectedStatus: DeploymentScaling,
			expectedColor:  "yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
			clientset := fake.NewSimpleClientset(tt.deployment)
			client := &k8s.Client{Clientset: clientset}

			status, err := MonitorDeployment(context.Background(), client, "default", "test-deploy")
			if err != nil {
				t.Fatalf("MonitorDeployment failed: %v", err)
			}

			if status.Status != tt.expectedStatus {
				t.Errorf("expected Status=%v, got %v", tt.expectedStatus, status.Status)
			}

			if status.StatusColor() != tt.expectedColor {
				t.Errorf("expected color=%s, got %s", tt.expectedColor, status.StatusColor())
			}
		})
	}
}

func TestMonitorDeployments(t *testing.T) {
	deployments := []*appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "deploy1", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: "True"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "deploy2", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(2)},
			Status: appsv1.DeploymentStatus{
				Replicas:          2,
				ReadyReplicas:     2,
				AvailableReplicas: 2,
				UpdatedReplicas:   2,
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: "True"},
				},
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	for _, d := range deployments {
		_, _ = clientset.AppsV1().Deployments("default").Create(context.Background(), d, metav1.CreateOptions{})
	}

	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorDeployments(context.Background(), client, "default", []string{"deploy1", "deploy2"})
	if err != nil {
		t.Fatalf("MonitorDeployments failed: %v", err)
	}

	if len(status.Deployments) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(status.Deployments))
	}

	if status.OverallStatus != DeploymentHealthy {
		t.Errorf("expected overall status to be Healthy, got %v", status.OverallStatus)
	}
}

func TestStartDeploymentsMonitoring(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
		Status: appsv1.DeploymentStatus{
			Replicas:          3,
			ReadyReplicas:     3,
			AvailableReplicas: 3,
			UpdatedReplicas:   3,
			Conditions: []appsv1.DeploymentCondition{
				{Type: appsv1.DeploymentAvailable, Status: "True"},
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(deployment)
	client := &k8s.Client{Clientset: clientset}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	updates := StartDeploymentsMonitoring(ctx, client, "default", []string{"test-deploy"}, 100*time.Millisecond)

	// Should receive at least one update
	select {
	case status := <-updates:
		if status == nil {
			t.Fatal("received nil status")
		}
		if len(status.Deployments) != 1 {
			t.Errorf("expected 1 deployment, got %d", len(status.Deployments))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for status update")
	}
}

func TestDeploymentStatusColor(t *testing.T) {
	tests := []struct {
		name          string
		status        DeploymentHealthStatus
		expectedColor string
	}{
		{"healthy", DeploymentHealthy, "green"},
		{"scaling", DeploymentScaling, "yellow"},
		{"progressing", DeploymentProgressing, "yellow"},
		{"unavailable", DeploymentUnavailable, "red"},
		{"invalid", DeploymentHealthStatus("invalid"), "yellow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &DeploymentStatus{Status: tt.status}
			if got := ds.StatusColor(); got != tt.expectedColor {
				t.Errorf("expected %s, got %s", tt.expectedColor, got)
			}
		})
	}
}

func TestDetermineDeploymentHealth(t *testing.T) {
	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		expectedStatus DeploymentHealthStatus
	}{
		{
			name: "healthy with all conditions",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentAvailable, Status: "True"},
					},
				},
			},
			expectedStatus: DeploymentHealthy,
		},
		{
			name: "zero replicas unavailable",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          0,
					ReadyReplicas:     0,
					AvailableReplicas: 0,
					UpdatedReplicas:   0,
				},
			},
			expectedStatus: DeploymentUnavailable,
		},
		{
			name: "progressing with true status",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
					UpdatedReplicas:   2,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: "True", Reason: "ReplicaSetUpdated"},
					},
				},
			},
			expectedStatus: DeploymentProgressing,
		},
		{
			name: "scaling with new replica set",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(5),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentProgressing, Status: "True", Reason: "NewReplicaSetAvailable"},
					},
				},
			},
			expectedStatus: DeploymentScaling,
		},
		{
			name: "replica mismatch",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(5),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
					UpdatedReplicas:   3,
				},
			},
			expectedStatus: DeploymentScaling,
		},
		{
			name: "nil replicas spec",
			deployment: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          0,
					ReadyReplicas:     0,
					AvailableReplicas: 0,
					UpdatedReplicas:   0,
				},
			},
			expectedStatus: DeploymentUnavailable, // 0 available replicas = unavailable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := determineDeploymentHealth(tt.deployment)
			if status != tt.expectedStatus {
				t.Errorf("expected %v, got %v", tt.expectedStatus, status)
			}
		})
	}
}

func TestMonitorDeploymentsError(t *testing.T) {
	// Test with non-existent deployment
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorDeployments(context.Background(), client, "default", []string{"non-existent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty list if deployment doesn't exist
	if len(status.Deployments) != 0 {
		t.Errorf("expected 0 deployments, got %d", len(status.Deployments))
	}
}

func TestMonitorDeploymentsMixedStates(t *testing.T) {
	deployments := []*appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "healthy", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: "True"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "unavailable", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(2)},
			Status: appsv1.DeploymentStatus{
				Replicas:          0,
				ReadyReplicas:     0,
				AvailableReplicas: 0,
				UpdatedReplicas:   0,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "scaling", Namespace: "default"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(5)},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
				UpdatedReplicas:   3,
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	for _, d := range deployments {
		_, _ = clientset.AppsV1().Deployments("default").Create(context.Background(), d, metav1.CreateOptions{})
	}

	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorDeployments(context.Background(), client, "default", []string{"healthy", "unavailable", "scaling"})
	if err != nil {
		t.Fatalf("MonitorDeployments failed: %v", err)
	}

	if len(status.Deployments) != 3 {
		t.Errorf("expected 3 deployments, got %d", len(status.Deployments))
	}

	// Overall status should be Unavailable since one deployment is unavailable
	if status.OverallStatus != DeploymentUnavailable {
		t.Errorf("expected overall status to be Unavailable, got %v", status.OverallStatus)
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
