package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultMonitorConfig(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := DefaultMonitorConfig(client, "test-node", "rook-ceph", []string{"deploy1"})

	if cfg.Client != client {
		t.Error("client not set correctly")
	}
	if cfg.NodeName != "test-node" {
		t.Errorf("expected NodeName=test-node, got %s", cfg.NodeName)
	}
	if cfg.CephNamespace != "rook-ceph" {
		t.Errorf("expected CephNamespace=rook-ceph, got %s", cfg.CephNamespace)
	}
	if len(cfg.DeploymentNames) != 1 || cfg.DeploymentNames[0] != "deploy1" {
		t.Errorf("expected DeploymentNames=[deploy1], got %v", cfg.DeploymentNames)
	}
	if cfg.NodeRefreshInterval != 2*time.Second {
		t.Errorf("expected 2s node refresh, got %s", cfg.NodeRefreshInterval)
	}
	if cfg.CephRefreshInterval != 5*time.Second {
		t.Errorf("expected 5s ceph refresh, got %s", cfg.CephRefreshInterval)
	}
	if cfg.DeploymentRefreshInterval != 2*time.Second {
		t.Errorf("expected 2s deployment refresh, got %s", cfg.DeploymentRefreshInterval)
	}
}

func TestNewMonitor(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}
	cfg := DefaultMonitorConfig(client, "test-node", "rook-ceph", []string{})

	monitor := NewMonitor(cfg)

	if monitor == nil {
		t.Fatal("expected monitor to be created")
	}
	if monitor.config != cfg {
		t.Error("config not set correctly")
	}
	if monitor.updates == nil {
		t.Error("updates channel not created")
	}
	if monitor.ctx == nil {
		t.Error("context not created")
	}
	if monitor.cancel == nil {
		t.Error("cancel function not created")
	}
	if monitor.latest == nil {
		t.Error("latest update not initialized")
	}
}

func TestMonitorGetLatest(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}
	cfg := DefaultMonitorConfig(client, "test-node", "rook-ceph", []string{})

	monitor := NewMonitor(cfg)

	latest := monitor.GetLatest()
	if latest == nil {
		t.Fatal("expected latest to not be nil")
	}
	if latest.UpdateTime.IsZero() {
		t.Error("expected UpdateTime to be set")
	}
}

func TestMonitorStartStop(t *testing.T) {
	// Setup node for monitoring
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

	// Setup deployment for monitoring
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "rook-ceph",
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
	}

	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node, deployment)
	client := &k8s.Client{Clientset: clientset}

	cfg := &MonitorConfig{
		Client:                    client,
		NodeName:                  "test-node",
		CephNamespace:             "rook-ceph",
		DeploymentNames:           []string{"test-deploy"},
		NodeRefreshInterval:       100 * time.Millisecond,
		CephRefreshInterval:       100 * time.Millisecond,
		DeploymentRefreshInterval: 100 * time.Millisecond,
	}

	monitor := NewMonitor(cfg)
	updates := monitor.Start()

	// Should receive updates - may take multiple to get all data
	updateReceived := false
	timeout := time.After(1 * time.Second)
	for !updateReceived {
		select {
		case update := <-updates:
			if update == nil {
				t.Fatal("received nil update")
			}
			// Check if we have at least some data
			if update.NodeStatus != nil || update.DeploymentsStatus != nil {
				updateReceived = true
			}
			if update.HealthSummary == nil {
				t.Error("expected HealthSummary to be set")
			}
		case <-timeout:
			t.Fatal("timeout waiting for update")
		}
	}

	// Stop the monitor
	monitor.Stop()

	// Channel should be closed
	time.Sleep(100 * time.Millisecond)
	select {
	case _, ok := <-updates:
		if ok {
			t.Error("expected channel to be closed")
		}
	default:
		t.Error("channel should be closed but appears to be blocking")
	}
}

func TestStartMonitoring(t *testing.T) {
	tests := []struct {
		name        string
		config      *MonitorConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil client",
			config: &MonitorConfig{
				Client:        nil,
				NodeName:      "test",
				CephNamespace: "rook-ceph",
			},
			expectError: true,
			errorMsg:    "k8s client is required",
		},
		{
			name: "empty node name",
			config: &MonitorConfig{
				Client:        &k8s.Client{},
				NodeName:      "",
				CephNamespace: "rook-ceph",
			},
			expectError: true,
			errorMsg:    "node name is required",
		},
		{
			name: "empty ceph namespace",
			config: &MonitorConfig{
				Client:        &k8s.Client{},
				NodeName:      "test",
				CephNamespace: "",
			},
			expectError: true,
			errorMsg:    "ceph namespace is required",
		},
		{
			name: "valid config",
			config: &MonitorConfig{
				//nolint:staticcheck // SA1019: NewClientset requires apply configurations
				Client:                    &k8s.Client{Clientset: fake.NewSimpleClientset()},
				NodeName:                  "test",
				CephNamespace:             "rook-ceph",
				NodeRefreshInterval:       1 * time.Second,
				CephRefreshInterval:       1 * time.Second,
				DeploymentRefreshInterval: 1 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			monitor, updates, err := StartMonitoring(ctx, tt.config)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
				if monitor != nil {
					t.Error("expected nil monitor on error")
				}
				if updates != nil {
					t.Error("expected nil updates on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if monitor == nil {
					t.Fatal("expected monitor to be created")
				}
				if updates == nil {
					t.Fatal("expected updates channel to be created")
				}
				monitor.Stop()
			}
		})
	}
}

func TestMonitorStartIndividualMonitors(t *testing.T) {
	// Setup test data
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

	cfg := &MonitorConfig{
		Client:                    client,
		NodeName:                  "test-node",
		CephNamespace:             "rook-ceph",
		DeploymentNames:           []string{},
		NodeRefreshInterval:       100 * time.Millisecond,
		CephRefreshInterval:       100 * time.Millisecond,
		DeploymentRefreshInterval: 100 * time.Millisecond,
	}

	monitor := NewMonitor(cfg)

	// Test individual monitor starters
	t.Run("startNodeMonitor", func(t *testing.T) {
		nodeChan := monitor.startNodeMonitor()
		if nodeChan == nil {
			t.Fatal("expected node channel to be created")
		}
	})

	t.Run("startCephMonitor", func(t *testing.T) {
		cephChan := monitor.startCephMonitor()
		if cephChan == nil {
			t.Fatal("expected ceph channel to be created")
		}
	})

	t.Run("startOSDMonitor", func(t *testing.T) {
		osdChan := monitor.startOSDMonitor()
		if osdChan == nil {
			t.Fatal("expected osd channel to be created")
		}
	})

	t.Run("startDeploymentsMonitor", func(t *testing.T) {
		deploymentsChan := monitor.startDeploymentsMonitor()
		if deploymentsChan == nil {
			t.Fatal("expected deployments channel to be created")
		}
	})

	// Cancel to clean up goroutines
	monitor.cancel()
}

func TestMonitorAggregator(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &MonitorConfig{
		Client:                    client,
		NodeName:                  "test-node",
		CephNamespace:             "rook-ceph",
		DeploymentNames:           []string{},
		NodeRefreshInterval:       100 * time.Millisecond,
		CephRefreshInterval:       100 * time.Millisecond,
		DeploymentRefreshInterval: 100 * time.Millisecond,
	}

	monitor := NewMonitor(cfg)

	// Create test channels
	nodeChan := make(chan *NodeStatus, 1)
	cephChan := make(chan *CephHealth, 1)
	osdChan := make(chan *OSDTreeStatus, 1)
	deploymentsChan := make(chan *DeploymentsStatus, 1)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodeChan, cephChan, osdChan, deploymentsChan)

	// Send test data
	nodeChan <- &NodeStatus{
		Name:        "test-node",
		Ready:       true,
		ReadyStatus: corev1.ConditionTrue,
	}

	// Wait for aggregation
	time.Sleep(50 * time.Millisecond)

	// Check latest was updated
	latest := monitor.GetLatest()
	if latest.NodeStatus == nil {
		t.Error("expected NodeStatus to be set in latest")
	}
	if latest.HealthSummary == nil {
		t.Error("expected HealthSummary to be set in latest")
	}

	// Cancel context to stop aggregator
	monitor.cancel()

	// Close channels to signal done
	close(nodeChan)
	close(cephChan)
	close(osdChan)
	close(deploymentsChan)

	// Wait for aggregator to finish with timeout
	done := make(chan struct{})
	go func() {
		monitor.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Aggregator finished successfully
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for aggregator to finish")
	}
}

func TestMonitorAggregatorAllChannels(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &MonitorConfig{
		Client:                    client,
		NodeName:                  "test-node",
		CephNamespace:             "rook-ceph",
		DeploymentNames:           []string{},
		NodeRefreshInterval:       100 * time.Millisecond,
		CephRefreshInterval:       100 * time.Millisecond,
		DeploymentRefreshInterval: 100 * time.Millisecond,
	}

	monitor := NewMonitor(cfg)

	// Create test channels
	nodeChan := make(chan *NodeStatus, 1)
	cephChan := make(chan *CephHealth, 1)
	osdChan := make(chan *OSDTreeStatus, 1)
	deploymentsChan := make(chan *DeploymentsStatus, 1)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodeChan, cephChan, osdChan, deploymentsChan)

	// Send data to all channels
	nodeChan <- &NodeStatus{
		Name:        "test-node",
		Ready:       true,
		ReadyStatus: corev1.ConditionTrue,
	}

	cephChan <- &CephHealth{
		OverallStatus: "HEALTH_OK",
		OSDCount:      3,
		OSDsUp:        3,
		OSDsIn:        3,
	}

	osdChan <- &OSDTreeStatus{
		OSDs: []OSDStatus{
			{ID: 0, Up: true, In: true},
		},
	}

	deploymentsChan <- &DeploymentsStatus{
		OverallStatus: DeploymentHealthy,
	}

	// Wait for aggregation
	time.Sleep(100 * time.Millisecond)

	// Check all data was aggregated
	latest := monitor.GetLatest()
	if latest.NodeStatus == nil {
		t.Error("expected NodeStatus to be set")
	}
	if latest.CephHealth == nil {
		t.Error("expected CephHealth to be set")
	}
	if latest.OSDStatus == nil {
		t.Error("expected OSDStatus to be set")
	}
	if latest.DeploymentsStatus == nil {
		t.Error("expected DeploymentsStatus to be set")
	}
	if latest.HealthSummary == nil {
		t.Error("expected HealthSummary to be set")
	}

	// Cancel and cleanup
	monitor.cancel()
	close(nodeChan)
	close(cephChan)
	close(osdChan)
	close(deploymentsChan)

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		monitor.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for aggregator")
	}
}

func TestMonitorAggregatorChannelFull(t *testing.T) {
	//nolint:staticcheck // SA1019: NewClientset requires apply configurations, using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &MonitorConfig{
		Client:                    client,
		NodeName:                  "test-node",
		CephNamespace:             "rook-ceph",
		DeploymentNames:           []string{},
		NodeRefreshInterval:       100 * time.Millisecond,
		CephRefreshInterval:       100 * time.Millisecond,
		DeploymentRefreshInterval: 100 * time.Millisecond,
	}

	monitor := NewMonitor(cfg)
	// Override with a non-buffered channel to test the default case
	monitor.updates = make(chan *MonitorUpdate)

	nodeChan := make(chan *NodeStatus, 10)
	cephChan := make(chan *CephHealth, 10)
	osdChan := make(chan *OSDTreeStatus, 10)
	deploymentsChan := make(chan *DeploymentsStatus, 10)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodeChan, cephChan, osdChan, deploymentsChan)

	// Send multiple updates quickly without draining
	for i := 0; i < 5; i++ {
		nodeChan <- &NodeStatus{
			Name:        "test-node",
			Ready:       true,
			ReadyStatus: corev1.ConditionTrue,
		}
	}

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Latest should still be updated
	latest := monitor.GetLatest()
	if latest.NodeStatus == nil {
		t.Error("expected NodeStatus to be set even with full channel")
	}

	// Cancel and cleanup
	monitor.cancel()
	close(nodeChan)
	close(cephChan)
	close(osdChan)
	close(deploymentsChan)

	done := make(chan struct{})
	go func() {
		monitor.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for aggregator")
	}
}
