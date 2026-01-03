package k8s

import (
	"context"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFindRookCephToolsPod(t *testing.T) {
	ctx := context.Background()

	// Create a ready rook-ceph-tools pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools-12345",
			Namespace: "rook-ceph",
			Labels: map[string]string{
				"app": "rook-ceph-tools",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	foundPod, err := client.findRookCephToolsPod(ctx, "rook-ceph")
	if err != nil {
		t.Fatalf("failed to find rook-ceph-tools pod: %v", err)
	}

	if foundPod.Name != "rook-ceph-tools-12345" {
		t.Errorf("expected pod name 'rook-ceph-tools-12345', got %s", foundPod.Name)
	}
}

func TestFindRookCephToolsPod_NotFound(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewClientset()
	client := newClientFromInterface(clientset)

	_, err := client.findRookCephToolsPod(ctx, "rook-ceph")
	if err == nil {
		t.Error("expected error when no rook-ceph-tools pod exists, got nil")
	}
}

func TestFindRookCephToolsPod_NotReady(t *testing.T) {
	ctx := context.Background()

	// Create a non-ready rook-ceph-tools pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools-12345",
			Namespace: "rook-ceph",
			Labels: map[string]string{
				"app": "rook-ceph-tools",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	clientset := fake.NewClientset(pod)
	client := newClientFromInterface(clientset)

	_, err := client.findRookCephToolsPod(ctx, "rook-ceph")
	if err == nil {
		t.Error("expected error when rook-ceph-tools pod is not ready, got nil")
	}
}

func TestIsPodReady(t *testing.T) {
	tests := []struct {
		name  string
		pod   *corev1.Pod
		ready bool
	}{
		{
			name: "ready pod",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			ready: true,
		},
		{
			name: "not ready - pending",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			ready: false,
		},
		{
			name: "not ready - running but condition false",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
				},
			},
			ready: false,
		},
		{
			name: "not ready - no conditions",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{},
				},
			},
			ready: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready := isPodReady(tt.pod)
			if ready != tt.ready {
				t.Errorf("expected ready=%v, got %v", tt.ready, ready)
			}
		})
	}
}

func TestCephStatus_ParseJSON(t *testing.T) {
	jsonData := `{
		"health": {
			"status": "HEALTH_OK"
		},
		"osdmap": {
			"num_osds": 3,
			"num_up_osds": 3,
			"num_in_osds": 3,
			"full": false,
			"nearfull": false
		}
	}`

	var status CephStatus
	err := json.Unmarshal([]byte(jsonData), &status)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if status.Health.Status != "HEALTH_OK" {
		t.Errorf("expected health status 'HEALTH_OK', got %s", status.Health.Status)
	}
	if status.OSDMap.NumOSDs != 3 {
		t.Errorf("expected 3 OSDs, got %d", status.OSDMap.NumOSDs)
	}
	if status.OSDMap.Full {
		t.Error("expected full=false")
	}
}

func TestCephStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		healthy bool
	}{
		{"healthy", "HEALTH_OK", true},
		{"healthy lowercase", "health_ok", true},
		{"warning", "HEALTH_WARN", false},
		{"error", "HEALTH_ERR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &CephStatus{}
			status.Health.Status = tt.status

			if status.IsHealthy() != tt.healthy {
				t.Errorf("expected IsHealthy()=%v for status %s", tt.healthy, tt.status)
			}
		})
	}
}

func TestCephStatus_IsWarning(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		warning bool
	}{
		{"warning", "HEALTH_WARN", true},
		{"warning lowercase", "health_warn", true},
		{"healthy", "HEALTH_OK", false},
		{"error", "HEALTH_ERR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &CephStatus{}
			status.Health.Status = tt.status

			if status.IsWarning() != tt.warning {
				t.Errorf("expected IsWarning()=%v for status %s", tt.warning, tt.status)
			}
		})
	}
}

func TestCephStatus_IsError(t *testing.T) {
	tests := []struct {
		name   string
		status string
		isErr  bool
	}{
		{"error", "HEALTH_ERR", true},
		{"error lowercase", "health_err", true},
		{"healthy", "HEALTH_OK", false},
		{"warning", "HEALTH_WARN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &CephStatus{}
			status.Health.Status = tt.status

			if status.IsError() != tt.isErr {
				t.Errorf("expected IsError()=%v for status %s", tt.isErr, tt.status)
			}
		})
	}
}

func TestCephOSDTree_ParseJSON(t *testing.T) {
	jsonData := `{
		"nodes": [
			{
				"id": 0,
				"name": "default",
				"type": "root",
				"status": "up",
				"reweight": 1.0,
				"children": [1, 2]
			},
			{
				"id": 1,
				"name": "osd.0",
				"type": "osd",
				"status": "up",
				"reweight": 1.0
			}
		]
	}`

	var tree CephOSDTree
	err := json.Unmarshal([]byte(jsonData), &tree)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(tree.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tree.Nodes))
	}
	if tree.Nodes[0].Name != "default" {
		t.Errorf("expected first node name 'default', got %s", tree.Nodes[0].Name)
	}
	if tree.Nodes[0].Type != "root" {
		t.Errorf("expected first node type 'root', got %s", tree.Nodes[0].Type)
	}
	if len(tree.Nodes[0].Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(tree.Nodes[0].Children))
	}
}
