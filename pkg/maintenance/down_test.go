package maintenance

import (
	"bytes"
	"testing"

	"github.com/andri/crook/internal/logger"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateDeploymentReplicas(t *testing.T) {
	tests := []struct {
		name             string
		deployments      []appsv1.Deployment
		expectWarningLog bool
	}{
		{
			name: "all deployments have 1 replica - no warning",
			deployments: []appsv1.Deployment{
				makeDeploymentWithReplicas("osd-0", 1),
				makeDeploymentWithReplicas("mon-a", 1),
			},
			expectWarningLog: false,
		},
		{
			name: "one deployment has 2 replicas - warning",
			deployments: []appsv1.Deployment{
				makeDeploymentWithReplicas("osd-0", 2),
				makeDeploymentWithReplicas("mon-a", 1),
			},
			expectWarningLog: true,
		},
		{
			name: "multiple deployments have >1 replicas - all listed",
			deployments: []appsv1.Deployment{
				makeDeploymentWithReplicas("osd-0", 3),
				makeDeploymentWithReplicas("mon-a", 2),
				makeDeploymentWithReplicas("exporter", 1),
			},
			expectWarningLog: true,
		},
		{
			name: "deployment with nil replicas - no warning",
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osd-nil",
						Namespace: "rook-ceph",
					},
					Spec: appsv1.DeploymentSpec{
						// Replicas is nil (defaults to 1)
					},
				},
			},
			expectWarningLog: false,
		},
		{
			name: "deployment with 0 replicas - no warning",
			deployments: []appsv1.Deployment{
				makeDeploymentWithReplicas("osd-scaled-down", 0),
			},
			expectWarningLog: false,
		},
		{
			name:             "empty deployments slice - no warning",
			deployments:      []appsv1.Deployment{},
			expectWarningLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			testLogger := logger.New(logger.Config{
				Level:  logger.LevelDebug,
				Format: logger.FormatText,
				Output: &buf,
			})
			originalLogger := logger.GetDefault()
			logger.SetDefault(testLogger)
			defer logger.SetDefault(originalLogger)

			// Call the function
			ValidateDeploymentReplicas(tt.deployments)

			// Check if warning was logged
			logOutput := buf.String()
			containsWarning := len(logOutput) > 0

			if tt.expectWarningLog && !containsWarning {
				t.Errorf("expected warning log, but got none")
			}
			if !tt.expectWarningLog && containsWarning {
				t.Errorf("expected no warning log, but got: %s", logOutput)
			}

			// If we expect a warning, verify it contains "unexpected replica counts"
			if tt.expectWarningLog && containsWarning {
				if !bytes.Contains(buf.Bytes(), []byte("unexpected replica counts")) {
					t.Errorf("warning log should contain 'unexpected replica counts', got: %s", logOutput)
				}
			}
		})
	}
}

// makeDeploymentWithReplicas creates a deployment with the specified replica count
func makeDeploymentWithReplicas(name string, replicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}
