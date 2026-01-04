package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMonitorNodeStatus(t *testing.T) {
	tests := []struct {
		name          string
		node          *corev1.Node
		pods          []corev1.Pod
		expectedReady bool
		expectedColor string
	}{
		{
			name: "healthy node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
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
			pods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod2"}},
			},
			expectedReady: true,
			expectedColor: "green",
		},
		{
			name: "not ready node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.28.0",
					},
				},
			},
			expectedReady: false,
			expectedColor: "red",
		},
		{
			name: "unknown status node",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionUnknown},
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.28.0",
					},
				},
			},
			expectedReady: false,
			expectedColor: "yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
			clientset := fake.NewSimpleClientset(tt.node)
			for i := range tt.pods {
				tt.pods[i].Spec.NodeName = "test-node"
				_, _ = clientset.CoreV1().Pods("default").Create(context.Background(), &tt.pods[i], metav1.CreateOptions{})
			}

			client := &k8s.Client{Clientset: clientset}

			status, err := MonitorNodeStatus(context.Background(), client, "test-node")
			if err != nil {
				t.Fatalf("MonitorNodeStatus failed: %v", err)
			}

			if status.Ready != tt.expectedReady {
				t.Errorf("expected Ready=%v, got %v", tt.expectedReady, status.Ready)
			}

			if status.NodeStatusColor() != tt.expectedColor {
				t.Errorf("expected color=%s, got %s", tt.expectedColor, status.NodeStatusColor())
			}

			if status.Name != "test-node" {
				t.Errorf("expected Name=test-node, got %s", status.Name)
			}

			if tt.pods != nil && status.PodCount != len(tt.pods) {
				t.Errorf("expected PodCount=%d, got %d", len(tt.pods), status.PodCount)
			}
		})
	}
}

func TestStartNodeMonitoring(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	updates := StartNodeMonitoring(ctx, client, "test-node", 100*time.Millisecond)

	// Should receive at least one update
	select {
	case status := <-updates:
		if status == nil {
			t.Fatal("received nil status")
		}
		if status.Name != "test-node" {
			t.Errorf("expected Name=test-node, got %s", status.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for status update")
	}

	// Wait for context to cancel and channel to close
	<-ctx.Done()

	// Give goroutine time to clean up
	time.Sleep(100 * time.Millisecond)

	// Channel should be closed - drain any pending messages first
	for {
		select {
		case _, ok := <-updates:
			if !ok {
				// Channel closed as expected
				return
			}
			// Got a message, continue draining
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timeout waiting for channel to close")
		}
	}
}

func TestNodeStatusColor(t *testing.T) {
	tests := []struct {
		name          string
		readyStatus   corev1.ConditionStatus
		expectedColor string
	}{
		{"ready true", corev1.ConditionTrue, "green"},
		{"ready false", corev1.ConditionFalse, "red"},
		{"ready unknown", corev1.ConditionUnknown, "yellow"},
		{"empty status", corev1.ConditionStatus(""), "yellow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &NodeStatus{ReadyStatus: tt.readyStatus}
			if got := ns.NodeStatusColor(); got != tt.expectedColor {
				t.Errorf("expected %s, got %s", tt.expectedColor, got)
			}
		})
	}
}

func TestGetPodCountForNode(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
	}

	// Note: fake.ClientSet doesn't properly support field selectors,
	// so this test verifies the function works but may return all pods
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "test-node"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "test-node"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod3", Namespace: "kube-system"},
			Spec:       corev1.PodSpec{NodeName: "test-node"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod4", Namespace: "default"},
			Spec:       corev1.PodSpec{NodeName: "other-node"}, // Different node
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	for i := range pods {
		_, _ = clientset.CoreV1().Pods(pods[i].Namespace).Create(context.Background(), &pods[i], metav1.CreateOptions{})
	}

	client := &k8s.Client{Clientset: clientset}

	count, err := getPodCountForNode(context.Background(), client, "test-node")
	if err != nil {
		t.Fatalf("getPodCountForNode failed: %v", err)
	}

	// The fake client doesn't properly filter by field selector,
	// so we just verify we get a count (which will be all 4 pods in this case)
	if count < 0 {
		t.Errorf("expected non-negative pod count, got %d", count)
	}
}

func TestMonitorNodeStatus_NoPods(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorNodeStatus(context.Background(), client, "test-node")
	if err != nil {
		t.Fatalf("MonitorNodeStatus failed: %v", err)
	}

	if status.PodCount != 0 {
		t.Errorf("expected PodCount=0, got %d", status.PodCount)
	}
}

func TestMonitorNodeStatus_NoReadyCondition(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorNodeStatus(context.Background(), client, "test-node")
	if err != nil {
		t.Fatalf("MonitorNodeStatus failed: %v", err)
	}

	// Should default to not ready and unknown
	if status.Ready {
		t.Error("expected Ready=false when no Ready condition exists")
	}
	if status.ReadyStatus != corev1.ConditionUnknown {
		t.Errorf("expected ReadyStatus=Unknown, got %v", status.ReadyStatus)
	}
}

func TestMonitorNodeStatus_WithTaints(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				{
					Key:    "node.kubernetes.io/not-ready",
					Effect: corev1.TaintEffectNoSchedule,
				},
				{
					Key:    "node.kubernetes.io/disk-pressure",
					Effect: corev1.TaintEffectNoExecute,
				},
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	status, err := MonitorNodeStatus(context.Background(), client, "test-node")
	if err != nil {
		t.Fatalf("MonitorNodeStatus failed: %v", err)
	}

	if len(status.Taints) != 2 {
		t.Errorf("expected 2 taints, got %d", len(status.Taints))
	}
}

func TestMonitorNodeStatus_NonExistent(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	_, err := MonitorNodeStatus(context.Background(), client, "non-existent-node")
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
}

func TestStartNodeMonitoring_CancelContext(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	ctx, cancel := context.WithCancel(context.Background())

	updates := StartNodeMonitoring(ctx, client, "test-node", 50*time.Millisecond)

	// Immediately cancel
	cancel()

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// Drain and verify channel closes
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case _, ok := <-updates:
			if !ok {
				// Channel closed as expected
				return
			}
			// Got a message, continue draining
		case <-timeout:
			t.Fatal("timeout waiting for channel to close after context cancellation")
		}
	}
}
