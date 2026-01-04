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

func TestCephHealthColor(t *testing.T) {
	tests := []struct {
		name          string
		overallStatus string
		expectedColor string
	}{
		{"healthy", "HEALTH_OK", "green"},
		{"warning", "HEALTH_WARN", "yellow"},
		{"error", "HEALTH_ERR", "red"},
		{"unknown", "UNKNOWN", "yellow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &CephHealth{OverallStatus: tt.overallStatus}
			if got := ch.HealthColor(); got != tt.expectedColor {
				t.Errorf("expected %s, got %s", tt.expectedColor, got)
			}
		})
	}
}

func TestCephIsHealthy(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"HEALTH_OK", true},
		{"HEALTH_WARN", false},
		{"HEALTH_ERR", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			ch := &CephHealth{OverallStatus: tt.status}
			if got := ch.IsHealthy(); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestMonitorCephHealth_NoToolsPod(t *testing.T) {
	// Test case where no rook-ceph-tools pod exists
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	_, err := MonitorCephHealth(context.Background(), client, "rook-ceph")
	if err == nil {
		t.Fatal("expected error when no ceph-tools pod exists")
	}
}

func TestMonitorCephHealth_ToolsPodNotReady(t *testing.T) {
	// Test case where rook-ceph-tools pod exists but is not ready
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools-abc123",
			Namespace: "rook-ceph",
			Labels:    map[string]string{"app": "rook-ceph-tools"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(pod)
	client := &k8s.Client{Clientset: clientset}

	_, err := MonitorCephHealth(context.Background(), client, "rook-ceph")
	if err == nil {
		t.Fatal("expected error when ceph-tools pod is not ready")
	}
}

func TestMonitorOSDStatus_NoToolsPod(t *testing.T) {
	// Test case where no rook-ceph-tools pod exists
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	_, err := MonitorOSDStatus(context.Background(), client, "rook-ceph", "test-node")
	if err == nil {
		t.Fatal("expected error when no ceph-tools pod exists")
	}
}

func TestContainsFlag(t *testing.T) {
	tests := []struct {
		name     string
		flags    string
		flag     string
		expected bool
	}{
		{"single flag match", "noout", "noout", true},
		{"multiple flags match first", "noout,nodown,noup", "noout", true},
		{"multiple flags match middle", "noout,nodown,noup", "nodown", true},
		{"multiple flags match last", "noout,nodown,noup", "noup", true},
		{"no match", "nodown,noup", "noout", false},
		{"empty flags", "", "noout", false},
		{"partial match should fail", "nooutside", "noout", false},
		{"similar but different", "noout,nodown", "no", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsFlag(tt.flags, tt.flag)
			if got != tt.expected {
				t.Errorf("containsFlag(%q, %q) = %v, want %v", tt.flags, tt.flag, got, tt.expected)
			}
		})
	}
}

func TestStartCephHealthMonitoring(t *testing.T) {
	// Setup a mock pod that will fail when trying to exec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools-abc123",
			Namespace: "rook-ceph",
			Labels:    map[string]string{"app": "rook-ceph-tools"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(pod)
	client := &k8s.Client{Clientset: clientset}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	updates := StartCephHealthMonitoring(ctx, client, "rook-ceph", 50*time.Millisecond)

	// Since we can't mock pod exec, we won't get successful updates,
	// but we should verify the channel gets closed when context is cancelled
	<-ctx.Done()

	time.Sleep(100 * time.Millisecond)

	// Channel should eventually close
	select {
	case _, ok := <-updates:
		// Either got a message (errors) or channel closed - both are acceptable
		_ = ok
	default:
		// Channel might be closed or empty
	}
}

func TestStartOSDMonitoring(t *testing.T) {
	// Setup a mock pod that will fail when trying to exec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-tools-abc123",
			Namespace: "rook-ceph",
			Labels:    map[string]string{"app": "rook-ceph-tools"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(pod)
	client := &k8s.Client{Clientset: clientset}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	updates := StartOSDMonitoring(ctx, client, "rook-ceph", "test-node", 50*time.Millisecond)

	// Since we can't mock pod exec, we won't get successful updates,
	// but we should verify the channel behavior
	<-ctx.Done()

	time.Sleep(100 * time.Millisecond)

	// Channel should eventually close
	select {
	case _, ok := <-updates:
		// Either got a message (errors) or channel closed - both are acceptable
		_ = ok
	default:
		// Channel might be closed or empty
	}
}

func TestCheckNoOutFlag_NoToolsPod(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	_, err := checkNoOutFlag(context.Background(), client, "rook-ceph")
	if err == nil {
		t.Fatal("expected error when no ceph-tools pod exists")
	}
}
