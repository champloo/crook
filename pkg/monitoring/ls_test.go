package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/components"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultLsMonitorConfig(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := DefaultLsMonitorConfig(client, "rook-ceph")

	if cfg.Client != client {
		t.Error("client not set correctly")
	}
	if cfg.Namespace != "rook-ceph" {
		t.Errorf("expected Namespace=rook-ceph, got %s", cfg.Namespace)
	}
	if cfg.NodesRefreshInterval != 2*time.Second {
		t.Errorf("expected 2s nodes refresh, got %s", cfg.NodesRefreshInterval)
	}
	if cfg.DeploymentsRefreshInterval != 2*time.Second {
		t.Errorf("expected 2s deployments refresh, got %s", cfg.DeploymentsRefreshInterval)
	}
	if cfg.PodsRefreshInterval != 2*time.Second {
		t.Errorf("expected 2s pods refresh, got %s", cfg.PodsRefreshInterval)
	}
	if cfg.OSDsRefreshInterval != 5*time.Second {
		t.Errorf("expected 5s osds refresh, got %s", cfg.OSDsRefreshInterval)
	}
	if cfg.HeaderRefreshInterval != 5*time.Second {
		t.Errorf("expected 5s header refresh, got %s", cfg.HeaderRefreshInterval)
	}
}

func TestNewLsMonitor(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}
	cfg := DefaultLsMonitorConfig(client, "rook-ceph")

	monitor := NewLsMonitor(cfg)

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

func TestNewLsMonitor_UsesParentContext(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	parentCtx, cancel := context.WithCancel(context.Background())
	cfg := DefaultLsMonitorConfig(client, "rook-ceph")
	cfg.Context = parentCtx

	monitor := NewLsMonitor(cfg)

	cancel()

	select {
	case <-monitor.ctx.Done():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for monitor context to be cancelled")
	}
}

func TestLsMonitorGetLatest(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}
	cfg := DefaultLsMonitorConfig(client, "rook-ceph")

	monitor := NewLsMonitor(cfg)

	latest := monitor.GetLatest()
	if latest == nil {
		t.Fatal("expected latest to not be nil")
	}
	if latest.UpdateTime.IsZero() {
		t.Error("expected UpdateTime to be set")
	}
}

func TestLsMonitorGetLatest_ThreadSafe(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}
	cfg := DefaultLsMonitorConfig(client, "rook-ceph")

	monitor := NewLsMonitor(cfg)

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				latest := monitor.GetLatest()
				if latest == nil {
					t.Error("expected latest to not be nil")
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLsMonitorStartStop(t *testing.T) {
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
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-ceph-osd-0",
			Namespace: "rook-ceph",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	}

	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node, deployment)
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)
	updates := monitor.Start()

	// Should receive updates
	updateReceived := false
	timeout := time.After(1 * time.Second)
	for !updateReceived {
		select {
		case update := <-updates:
			if update == nil {
				t.Fatal("received nil update")
			}
			// Check if we have at least some data
			if update.Nodes != nil || update.Deployments != nil {
				updateReceived = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for update")
		}
	}

	// Stop the monitor
	monitor.Stop()

	// Drain any remaining updates and verify channel is closed
	for range updates {
		// Keep reading until channel is closed
	}
}

func TestLsMonitorPollers(t *testing.T) {
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

	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset(node)
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)

	// Test individual poller starters
	t.Run("startNodesPoller", func(t *testing.T) {
		nodesCh := monitor.startNodesPoller()
		if nodesCh == nil {
			t.Fatal("expected nodes channel to be created")
		}
	})

	t.Run("startDeploymentsPoller", func(t *testing.T) {
		deploymentsCh := monitor.startDeploymentsPoller()
		if deploymentsCh == nil {
			t.Fatal("expected deployments channel to be created")
		}
	})

	t.Run("startPodsPoller", func(t *testing.T) {
		podsCh := monitor.startPodsPoller()
		if podsCh == nil {
			t.Fatal("expected pods channel to be created")
		}
	})

	t.Run("startOSDsPoller", func(t *testing.T) {
		osdsCh := monitor.startOSDsPoller()
		if osdsCh == nil {
			t.Fatal("expected osds channel to be created")
		}
	})

	t.Run("startHeaderPoller", func(t *testing.T) {
		headerCh := monitor.startHeaderPoller()
		if headerCh == nil {
			t.Fatal("expected header channel to be created")
		}
	})

	// Cancel to clean up goroutines
	monitor.cancel()
}

func TestLsMonitorAggregator(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)

	// Create test channels
	nodesCh := make(chan []k8s.NodeInfo, 1)
	deploymentsCh := make(chan []k8s.DeploymentInfo, 1)
	podsCh := make(chan []k8s.PodInfo, 1)
	osdsCh := make(chan []k8s.OSDInfo, 1)
	headerCh := make(chan *components.ClusterHeaderData, 1)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodesCh, deploymentsCh, podsCh, osdsCh, headerCh)

	// Send test data
	nodesCh <- []k8s.NodeInfo{
		{Name: "test-node", Status: "Ready"},
	}

	// Wait for aggregation
	time.Sleep(50 * time.Millisecond)

	// Check latest was updated
	latest := monitor.GetLatest()
	if latest.Nodes == nil {
		t.Error("expected Nodes to be set in latest")
	}
	if len(latest.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(latest.Nodes))
	}

	// Cancel context to stop aggregator
	monitor.cancel()

	// Close channels to signal done
	close(nodesCh)
	close(deploymentsCh)
	close(podsCh)
	close(osdsCh)
	close(headerCh)

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

func TestLsMonitorAggregatorAllChannels(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)

	// Create test channels
	nodesCh := make(chan []k8s.NodeInfo, 1)
	deploymentsCh := make(chan []k8s.DeploymentInfo, 1)
	podsCh := make(chan []k8s.PodInfo, 1)
	osdsCh := make(chan []k8s.OSDInfo, 1)
	headerCh := make(chan *components.ClusterHeaderData, 1)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodesCh, deploymentsCh, podsCh, osdsCh, headerCh)

	// Send data to all channels
	nodesCh <- []k8s.NodeInfo{
		{Name: "test-node", Status: "Ready"},
	}

	deploymentsCh <- []k8s.DeploymentInfo{
		{Name: "test-deploy", Status: "Ready"},
	}

	podsCh <- []k8s.PodInfo{
		{Name: "test-pod", Status: "Running"},
	}

	osdsCh <- []k8s.OSDInfo{
		{ID: 0, Name: "osd.0", Status: "up"},
	}

	headerCh <- &components.ClusterHeaderData{
		Health: "HEALTH_OK",
		OSDs:   3,
		OSDsUp: 3,
		OSDsIn: 3,
	}

	// Wait for aggregation
	time.Sleep(100 * time.Millisecond)

	// Check all data was aggregated
	latest := monitor.GetLatest()
	if latest.Nodes == nil {
		t.Error("expected Nodes to be set")
	}
	if latest.Deployments == nil {
		t.Error("expected Deployments to be set")
	}
	if latest.Pods == nil {
		t.Error("expected Pods to be set")
	}
	if latest.OSDs == nil {
		t.Error("expected OSDs to be set")
	}
	if latest.Header == nil {
		t.Error("expected Header to be set")
	}

	// Cancel and cleanup
	monitor.cancel()
	close(nodesCh)
	close(deploymentsCh)
	close(podsCh)
	close(osdsCh)
	close(headerCh)

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

func TestLsMonitorAggregatorChannelFull(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)
	// Override with a non-buffered channel to test the default case
	monitor.updates = make(chan *LsMonitorUpdate)

	nodesCh := make(chan []k8s.NodeInfo, 10)
	deploymentsCh := make(chan []k8s.DeploymentInfo, 10)
	podsCh := make(chan []k8s.PodInfo, 10)
	osdsCh := make(chan []k8s.OSDInfo, 10)
	headerCh := make(chan *components.ClusterHeaderData, 10)

	// Start aggregator
	monitor.wg.Add(1)
	go monitor.aggregator(nodesCh, deploymentsCh, podsCh, osdsCh, headerCh)

	// Send multiple updates quickly without draining
	for i := 0; i < 5; i++ {
		nodesCh <- []k8s.NodeInfo{
			{Name: "test-node", Status: "Ready"},
		}
	}

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Latest should still be updated
	latest := monitor.GetLatest()
	if latest.Nodes == nil {
		t.Error("expected Nodes to be set even with full channel")
	}

	// Cancel and cleanup
	monitor.cancel()
	close(nodesCh)
	close(deploymentsCh)
	close(podsCh)
	close(osdsCh)
	close(headerCh)

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

func TestLsMonitorNodeFilter(t *testing.T) {
	//nolint:staticcheck // SA1019: using deprecated NewSimpleClientset
	clientset := fake.NewSimpleClientset()
	client := &k8s.Client{Clientset: clientset}

	cfg := &LsMonitorConfig{
		Client:                     client,
		Namespace:                  "rook-ceph",
		NodeFilter:                 "target-node",
		NodesRefreshInterval:       50 * time.Millisecond,
		DeploymentsRefreshInterval: 50 * time.Millisecond,
		PodsRefreshInterval:        50 * time.Millisecond,
		OSDsRefreshInterval:        50 * time.Millisecond,
		HeaderRefreshInterval:      50 * time.Millisecond,
	}

	monitor := NewLsMonitor(cfg)

	if monitor.config.NodeFilter != "target-node" {
		t.Errorf("expected NodeFilter=target-node, got %s", monitor.config.NodeFilter)
	}
}
