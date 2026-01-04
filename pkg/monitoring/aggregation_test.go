package monitoring

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func TestAggregateHealth(t *testing.T) {
	tests := []struct {
		name              string
		nodeStatus        *NodeStatus
		cephHealth        *CephHealth
		deploymentsStatus *DeploymentsStatus
		osdStatus         *OSDTreeStatus
		expectedStatus    OverallHealthStatus
		expectedColor     string
	}{
		{
			name: "all healthy",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
				OSDCount:      3,
				OSDsUp:        3,
				OSDsIn:        3,
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedStatus: HealthStatusHealthy,
			expectedColor:  "green",
		},
		{
			name: "node not ready - critical",
			nodeStatus: &NodeStatus{
				Ready:       false,
				ReadyStatus: corev1.ConditionFalse,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedStatus: HealthStatusCritical,
			expectedColor:  "red",
		},
		{
			name: "ceph warning - degraded",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus:  "HEALTH_WARN",
				HealthMessages: []string{"PGs degraded"},
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedStatus: HealthStatusDegraded,
			expectedColor:  "yellow",
		},
		{
			name: "ceph error - critical",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus:  "HEALTH_ERR",
				HealthMessages: []string{"OSDs down"},
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedStatus: HealthStatusCritical,
			expectedColor:  "red",
		},
		{
			name: "deployments unavailable - critical",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentUnavailable,
			},
			expectedStatus: HealthStatusCritical,
			expectedColor:  "red",
		},
		{
			name: "deployments scaling - degraded",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentScaling,
			},
			expectedStatus: HealthStatusDegraded,
			expectedColor:  "yellow",
		},
		{
			name: "node cordoned - degraded",
			nodeStatus: &NodeStatus{
				Ready:         true,
				ReadyStatus:   corev1.ConditionTrue,
				Unschedulable: true,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedStatus: HealthStatusDegraded,
			expectedColor:  "yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := AggregateHealth(tt.nodeStatus, tt.cephHealth, tt.deploymentsStatus, tt.osdStatus)

			if summary.Status != tt.expectedStatus {
				t.Errorf("expected Status=%v, got %v", tt.expectedStatus, summary.Status)
			}

			if summary.StatusColor() != tt.expectedColor {
				t.Errorf("expected color=%s, got %s", tt.expectedColor, summary.StatusColor())
			}

			if summary.LastUpdateTime.IsZero() {
				t.Error("expected LastUpdateTime to be set")
			}

			// Verify health info is populated
			if tt.nodeStatus != nil && summary.NodeHealth == nil {
				t.Error("expected NodeHealth to be populated")
			}

			if tt.cephHealth != nil && summary.CephHealth == nil {
				t.Error("expected CephHealth to be populated")
			}

			if tt.deploymentsStatus != nil && summary.DeploymentHealth == nil {
				t.Error("expected DeploymentHealth to be populated")
			}
		})
	}
}

func TestEvaluateNodeHealth(t *testing.T) {
	tests := []struct {
		name        string
		nodeStatus  *NodeStatus
		wantHealthy bool
		wantReasons bool
	}{
		{
			name: "healthy node",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			wantHealthy: true,
			wantReasons: false,
		},
		{
			name: "not ready node",
			nodeStatus: &NodeStatus{
				Ready:       false,
				ReadyStatus: corev1.ConditionFalse,
			},
			wantHealthy: false,
			wantReasons: true,
		},
		{
			name: "cordoned node",
			nodeStatus: &NodeStatus{
				Ready:         true,
				ReadyStatus:   corev1.ConditionTrue,
				Unschedulable: true,
			},
			wantHealthy: false,
			wantReasons: true,
		},
		{
			name:        "nil node status",
			nodeStatus:  nil,
			wantHealthy: false,
			wantReasons: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &HealthSummary{
				Reasons:        []string{},
				LastUpdateTime: time.Now(),
			}

			healthy := evaluateNodeHealth(tt.nodeStatus, summary)

			if healthy != tt.wantHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.wantHealthy, healthy)
			}

			if tt.wantReasons && len(summary.Reasons) == 0 {
				t.Error("expected reasons to be populated")
			}

			if !tt.wantReasons && len(summary.Reasons) > 0 {
				t.Errorf("expected no reasons, got %v", summary.Reasons)
			}

			if summary.NodeHealth == nil {
				t.Error("expected NodeHealth to be populated")
			}
		})
	}
}

func TestEvaluateCephHealthEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		cephHealth  *CephHealth
		osdStatus   *OSDTreeStatus
		wantHealthy bool
	}{
		{
			name:        "nil ceph health",
			cephHealth:  nil,
			osdStatus:   nil,
			wantHealthy: false,
		},
		{
			name: "OSDs down on specific node",
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
				OSDCount:      3,
				OSDsUp:        3,
				OSDsIn:        3,
			},
			osdStatus: &OSDTreeStatus{
				OSDs: []OSDStatus{
					{ID: 0, Up: true, In: true},
					{ID: 1, Up: false, In: true}, // One down
					{ID: 2, Up: true, In: true},
				},
			},
			wantHealthy: false,
		},
		{
			name: "cluster-wide OSD down issues",
			cephHealth: &CephHealth{
				OverallStatus:  "HEALTH_WARN",
				OSDCount:       6,
				OSDsUp:         5,
				OSDsIn:         6,
				HealthMessages: []string{"1 OSD down"},
			},
			osdStatus:   nil,
			wantHealthy: false,
		},
		{
			name: "cluster-wide OSD out issues",
			cephHealth: &CephHealth{
				OverallStatus:  "HEALTH_WARN",
				OSDCount:       6,
				OSDsUp:         6,
				OSDsIn:         5,
				HealthMessages: []string{"1 OSD out"},
			},
			osdStatus:   nil,
			wantHealthy: false,
		},
		{
			name: "all OSDs up and in on node",
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
				OSDCount:      3,
				OSDsUp:        3,
				OSDsIn:        3,
			},
			osdStatus: &OSDTreeStatus{
				OSDs: []OSDStatus{
					{ID: 0, Up: true, In: true},
					{ID: 1, Up: true, In: true},
					{ID: 2, Up: true, In: true},
				},
			},
			wantHealthy: true,
		},
		{
			name: "OSD out on specific node",
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
				OSDCount:      3,
				OSDsUp:        3,
				OSDsIn:        3,
			},
			osdStatus: &OSDTreeStatus{
				OSDs: []OSDStatus{
					{ID: 0, Up: true, In: true},
					{ID: 1, Up: true, In: false}, // One out
					{ID: 2, Up: true, In: true},
				},
			},
			wantHealthy: false,
		},
		{
			name: "empty OSD list",
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
				OSDCount:      0,
				OSDsUp:        0,
				OSDsIn:        0,
			},
			osdStatus: &OSDTreeStatus{
				OSDs: []OSDStatus{},
			},
			wantHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &HealthSummary{Reasons: []string{}}
			healthy := evaluateCephHealth(tt.cephHealth, tt.osdStatus, summary)
			if healthy != tt.wantHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.wantHealthy, healthy)
			}
			if summary.CephHealth == nil {
				t.Error("expected CephHealth to be populated")
			}
		})
	}
}

func TestEvaluateDeploymentsHealthEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		deploymentsStatus *DeploymentsStatus
		wantHealthy       bool
	}{
		{
			name:              "nil deployments",
			deploymentsStatus: nil,
			wantHealthy:       false,
		},
		{
			name: "mixed deployment states",
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentScaling,
				Deployments: []DeploymentStatus{
					{Status: DeploymentHealthy},
					{Status: DeploymentScaling},
					{Status: DeploymentHealthy},
				},
			},
			wantHealthy: false,
		},
		{
			name: "all progressing",
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentProgressing,
				Deployments: []DeploymentStatus{
					{Status: DeploymentProgressing},
					{Status: DeploymentProgressing},
				},
			},
			wantHealthy: false,
		},
		{
			name: "some unavailable",
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentUnavailable,
				Deployments: []DeploymentStatus{
					{Status: DeploymentHealthy},
					{Status: DeploymentUnavailable},
				},
			},
			wantHealthy: false,
		},
		{
			name: "all healthy",
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
				Deployments: []DeploymentStatus{
					{Status: DeploymentHealthy},
					{Status: DeploymentHealthy},
					{Status: DeploymentHealthy},
				},
			},
			wantHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &HealthSummary{Reasons: []string{}}
			healthy := evaluateDeploymentsHealth(tt.deploymentsStatus, summary)
			if healthy != tt.wantHealthy {
				t.Errorf("expected healthy=%v, got %v", tt.wantHealthy, healthy)
			}
			if summary.DeploymentHealth == nil {
				t.Error("expected DeploymentHealth to be populated")
			}
		})
	}
}

func TestHealthSummaryStatusColor(t *testing.T) {
	tests := []struct {
		name          string
		status        OverallHealthStatus
		expectedColor string
	}{
		{"healthy", HealthStatusHealthy, "green"},
		{"degraded", HealthStatusDegraded, "yellow"},
		{"critical", HealthStatusCritical, "red"},
		{"unknown", HealthStatusUnknown, "yellow"},
		{"invalid", OverallHealthStatus("invalid"), "yellow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := &HealthSummary{Status: tt.status}
			if got := hs.StatusColor(); got != tt.expectedColor {
				t.Errorf("expected %s, got %s", tt.expectedColor, got)
			}
		})
	}
}

func TestHasAnyCriticalCondition(t *testing.T) {
	tests := []struct {
		name              string
		nodeStatus        *NodeStatus
		cephHealth        *CephHealth
		deploymentsStatus *DeploymentsStatus
		expectedCritical  bool
	}{
		{
			name: "node not ready is critical",
			nodeStatus: &NodeStatus{
				Ready:       false,
				ReadyStatus: corev1.ConditionFalse,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedCritical: true,
		},
		{
			name: "ceph error is critical",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_ERR",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedCritical: true,
		},
		{
			name: "deployments unavailable is critical",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentUnavailable,
			},
			expectedCritical: true,
		},
		{
			name: "all healthy is not critical",
			nodeStatus: &NodeStatus{
				Ready:       true,
				ReadyStatus: corev1.ConditionTrue,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_OK",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentHealthy,
			},
			expectedCritical: false,
		},
		{
			name: "warnings are not critical",
			nodeStatus: &NodeStatus{
				Ready:         true,
				ReadyStatus:   corev1.ConditionTrue,
				Unschedulable: true,
			},
			cephHealth: &CephHealth{
				OverallStatus: "HEALTH_WARN",
			},
			deploymentsStatus: &DeploymentsStatus{
				OverallStatus: DeploymentScaling,
			},
			expectedCritical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAnyCriticalCondition(tt.nodeStatus, tt.cephHealth, tt.deploymentsStatus)
			if result != tt.expectedCritical {
				t.Errorf("expected critical=%v, got %v", tt.expectedCritical, result)
			}
		})
	}
}
