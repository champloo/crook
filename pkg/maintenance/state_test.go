package maintenance

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAllDeploymentsScaledDown(t *testing.T) {
	tests := []struct {
		name        string
		deployments []appsv1.Deployment
		want        bool
	}{
		{
			name:        "empty list returns true",
			deployments: []appsv1.Deployment{},
			want:        true,
		},
		{
			name:        "nil list returns true",
			deployments: nil,
			want:        true,
		},
		{
			name: "all at zero replicas returns true",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 0, 0),
				makeDeploymentWithStatus("dep2", 0, 0),
			},
			want: true,
		},
		{
			name: "one with non-zero spec replicas returns false",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 0, 0),
				makeDeploymentWithStatus("dep2", 1, 0),
			},
			want: false,
		},
		{
			name: "all scaled down but one has ready replicas returns false",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 0, 0),
				makeDeploymentWithStatus("dep2", 0, 1), // spec=0 but still has ready pod
			},
			want: false,
		},
		{
			name: "nil replicas treated as non-zero returns false",
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "dep1"},
					Spec:       appsv1.DeploymentSpec{Replicas: nil}, // nil defaults to 1
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllDeploymentsScaledDown(tt.deployments)
			if got != tt.want {
				t.Errorf("AllDeploymentsScaledDown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllDeploymentsScaledUp(t *testing.T) {
	tests := []struct {
		name        string
		deployments []appsv1.Deployment
		want        bool
	}{
		{
			name:        "empty list returns true",
			deployments: []appsv1.Deployment{},
			want:        true,
		},
		{
			name:        "nil list returns true",
			deployments: nil,
			want:        true,
		},
		{
			name: "all at one replica and ready returns true",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 1, 1),
				makeDeploymentWithStatus("dep2", 1, 1),
			},
			want: true,
		},
		{
			name: "one with zero spec replicas returns false",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 1, 1),
				makeDeploymentWithStatus("dep2", 0, 0),
			},
			want: false,
		},
		{
			name: "spec is 1 but not ready yet returns false",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 1, 1),
				makeDeploymentWithStatus("dep2", 1, 0), // spec=1 but no ready pods
			},
			want: false,
		},
		{
			name: "higher replica count all ready returns true",
			deployments: []appsv1.Deployment{
				makeDeploymentWithStatus("dep1", 3, 3),
				makeDeploymentWithStatus("dep2", 2, 2),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllDeploymentsScaledUp(tt.deployments)
			if got != tt.want {
				t.Errorf("AllDeploymentsScaledUp() = %v, want %v", got, tt.want)
			}
		})
	}
}

// makeDeploymentWithStatus creates a deployment with the given spec replicas and ready replicas
func makeDeploymentWithStatus(name string, specReplicas, readyReplicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &specReplicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: readyReplicas,
		},
	}
}
