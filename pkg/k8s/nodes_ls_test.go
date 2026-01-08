package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListNodesWithCephPods(t *testing.T) {
	ctx := context.Background()

	// Create test nodes
	nodes := &corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker-1",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: false,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.28.0",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker-2",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-48 * time.Hour)},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: true, // Cordoned
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.28.0",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "control-plane-1",
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "",
					},
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-72 * time.Hour)},
				},
				Spec: corev1.NodeSpec{
					Unschedulable: false,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.28.0",
					},
				},
			},
		},
	}

	// Create test pods
	pods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook-ceph-osd-0-abc123",
					Namespace: "rook-ceph",
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook-ceph-osd-1-def456",
					Namespace: "rook-ceph",
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rook-ceph-mon-a-xyz789",
					Namespace: "rook-ceph",
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-2",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-pod",
					Namespace: "rook-ceph",
				},
				Spec: corev1.PodSpec{
					NodeName: "worker-1",
				},
			},
		},
	}

	// Create fake clientset
	clientset := fake.NewClientset(nodes, pods)
	client := &Client{Clientset: clientset}

	// Test ListNodesWithCephPods (uses default prefixes)
	result, err := client.ListNodesWithCephPods(ctx, "rook-ceph")

	if err != nil {
		t.Fatalf("ListNodesWithCephPods() error = %v", err)
	}

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}

	// Check node info
	nodeMap := make(map[string]NodeInfoForLS)
	for _, n := range result {
		nodeMap[n.Name] = n
	}

	// Check worker-1
	w1 := nodeMap["worker-1"]
	if w1.CephPodCount != 2 {
		t.Errorf("worker-1 CephPodCount = %d, want 2", w1.CephPodCount)
	}
	if w1.Status != "Ready" {
		t.Errorf("worker-1 Status = %s, want Ready", w1.Status)
	}
	if w1.Cordoned {
		t.Errorf("worker-1 Cordoned = true, want false")
	}

	// Check worker-2
	w2 := nodeMap["worker-2"]
	if w2.CephPodCount != 1 {
		t.Errorf("worker-2 CephPodCount = %d, want 1", w2.CephPodCount)
	}
	if !w2.Cordoned {
		t.Errorf("worker-2 Cordoned = false, want true")
	}

	// Check control-plane-1
	cp1 := nodeMap["control-plane-1"]
	if cp1.CephPodCount != 0 {
		t.Errorf("control-plane-1 CephPodCount = %d, want 0", cp1.CephPodCount)
	}
	if cp1.Status != "NotReady" {
		t.Errorf("control-plane-1 Status = %s, want NotReady", cp1.Status)
	}
}

func TestGetNodeStatusString(t *testing.T) {
	tests := []struct {
		name       string
		conditions []corev1.NodeCondition
		want       string
	}{
		{
			name: "ready node",
			conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			want: "Ready",
		},
		{
			name: "not ready node",
			conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
			want: "NotReady",
		},
		{
			name: "unknown status node",
			conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionUnknown},
			},
			want: "NotReady",
		},
		{
			name:       "no ready condition",
			conditions: []corev1.NodeCondition{},
			want:       "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: tt.conditions,
				},
			}
			if got := getNodeStatus(node); got != tt.want {
				t.Errorf("getNodeStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractNodeRoles(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   []string
	}{
		{
			name: "worker role",
			labels: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			want: []string{"worker"},
		},
		{
			name: "control-plane role",
			labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "",
			},
			want: []string{"control-plane"},
		},
		{
			name: "multiple roles",
			labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "",
				"node-role.kubernetes.io/master":        "",
			},
			want: []string{"control-plane", "master"},
		},
		{
			name: "no roles",
			labels: map[string]string{
				"other-label": "value",
			},
			want: []string{},
		},
		{
			name:   "nil labels",
			labels: nil,
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tt.labels,
				},
			}
			got := extractNodeRoles(node)

			// Convert to maps for easier comparison (order doesn't matter)
			gotMap := make(map[string]bool)
			for _, r := range got {
				gotMap[r] = true
			}
			wantMap := make(map[string]bool)
			for _, r := range tt.want {
				wantMap[r] = true
			}

			if len(gotMap) != len(wantMap) {
				t.Errorf("extractNodeRoles() = %v, want %v", got, tt.want)
				return
			}

			for r := range wantMap {
				if !gotMap[r] {
					t.Errorf("extractNodeRoles() missing role %s", r)
				}
			}
		})
	}
}

func TestMatchesAnyPrefix(t *testing.T) {
	tests := []struct {
		s        string
		prefixes []string
		want     bool
	}{
		{"rook-ceph-osd-0", []string{"rook-ceph-osd", "rook-ceph-mon"}, true},
		{"rook-ceph-mon-a", []string{"rook-ceph-osd", "rook-ceph-mon"}, true},
		{"other-pod", []string{"rook-ceph-osd", "rook-ceph-mon"}, false},
		{"rook-ceph-osd-0", []string{}, false},
		{"", []string{"rook-ceph-osd"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := matchesAnyPrefix(tt.s, tt.prefixes); got != tt.want {
				t.Errorf("matchesAnyPrefix(%q, %v) = %v, want %v", tt.s, tt.prefixes, got, tt.want)
			}
		})
	}
}
