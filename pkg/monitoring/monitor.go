package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/andri/crook/pkg/k8s"
)

// MonitorConfig holds configuration for the monitoring system
type MonitorConfig struct {
	// K8s client for API access
	Client *k8s.Client

	// Node name to monitor
	NodeName string

	// Ceph namespace (typically "rook-ceph")
	CephNamespace string

	// Deployment names to monitor
	DeploymentNames []string

	// Refresh intervals
	NodeRefreshInterval       time.Duration
	CephRefreshInterval       time.Duration
	DeploymentRefreshInterval time.Duration
}

// DefaultMonitorConfig returns a config with default refresh intervals
func DefaultMonitorConfig(client *k8s.Client, nodeName, cephNamespace string, deploymentNames []string) *MonitorConfig {
	return &MonitorConfig{
		Client:                    client,
		NodeName:                  nodeName,
		CephNamespace:             cephNamespace,
		DeploymentNames:           deploymentNames,
		NodeRefreshInterval:       2 * time.Second,
		CephRefreshInterval:       5 * time.Second,
		DeploymentRefreshInterval: 2 * time.Second,
	}
}

// MonitorUpdate contains the latest monitoring data
type MonitorUpdate struct {
	NodeStatus        *NodeStatus
	CephHealth        *CephHealth
	OSDStatus         *OSDTreeStatus
	DeploymentsStatus *DeploymentsStatus
	HealthSummary     *HealthSummary
	UpdateTime        time.Time
	Error             error
}

// Monitor manages background monitoring of cluster components
type Monitor struct {
	config  *MonitorConfig
	ctx     context.Context
	cancel  context.CancelFunc
	updates chan *MonitorUpdate
	wg      sync.WaitGroup
	mu      sync.RWMutex
	latest  *MonitorUpdate
}

// NewMonitor creates a new monitoring instance
func NewMonitor(config *MonitorConfig) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		updates: make(chan *MonitorUpdate, 10),
		latest: &MonitorUpdate{
			UpdateTime: time.Now(),
		},
	}
}

// Start begins background monitoring of all components
func (m *Monitor) Start() <-chan *MonitorUpdate {
	// Start individual component monitors
	nodeChan := m.startNodeMonitor()
	cephChan := m.startCephMonitor()
	osdChan := m.startOSDMonitor()
	deploymentsChan := m.startDeploymentsMonitor()

	// Start aggregator that combines all updates
	m.wg.Add(1)
	go m.aggregator(nodeChan, cephChan, osdChan, deploymentsChan)

	return m.updates
}

// Stop gracefully stops all monitoring goroutines
func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
	close(m.updates)
}

// GetLatest returns the most recent monitoring data
func (m *Monitor) GetLatest() *MonitorUpdate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	return &MonitorUpdate{
		NodeStatus:        m.latest.NodeStatus,
		CephHealth:        m.latest.CephHealth,
		OSDStatus:         m.latest.OSDStatus,
		DeploymentsStatus: m.latest.DeploymentsStatus,
		HealthSummary:     m.latest.HealthSummary,
		UpdateTime:        m.latest.UpdateTime,
		Error:             m.latest.Error,
	}
}

// startNodeMonitor starts background node monitoring
func (m *Monitor) startNodeMonitor() <-chan *NodeStatus {
	return StartNodeMonitoring(m.ctx, m.config.Client, m.config.NodeName, m.config.NodeRefreshInterval)
}

// startCephMonitor starts background Ceph health monitoring
func (m *Monitor) startCephMonitor() <-chan *CephHealth {
	return StartCephHealthMonitoring(m.ctx, m.config.Client, m.config.CephNamespace, m.config.CephRefreshInterval)
}

// startOSDMonitor starts background OSD monitoring
func (m *Monitor) startOSDMonitor() <-chan *OSDTreeStatus {
	return StartOSDMonitoring(m.ctx, m.config.Client, m.config.CephNamespace, m.config.NodeName, m.config.CephRefreshInterval)
}

// startDeploymentsMonitor starts background deployments monitoring
func (m *Monitor) startDeploymentsMonitor() <-chan *DeploymentsStatus {
	return StartDeploymentsMonitoring(m.ctx, m.config.Client, m.config.CephNamespace, m.config.DeploymentNames, m.config.DeploymentRefreshInterval)
}

// aggregator combines updates from all monitors and produces aggregated health status
func (m *Monitor) aggregator(
	nodeChan <-chan *NodeStatus,
	cephChan <-chan *CephHealth,
	osdChan <-chan *OSDTreeStatus,
	deploymentsChan <-chan *DeploymentsStatus,
) {
	defer m.wg.Done()

	var (
		nodeStatus        *NodeStatus
		cephHealth        *CephHealth
		osdStatus         *OSDTreeStatus
		deploymentsStatus *DeploymentsStatus
	)

	// Helper to send aggregated update
	sendUpdate := func() {
		summary := AggregateHealth(nodeStatus, cephHealth, deploymentsStatus, osdStatus)

		update := &MonitorUpdate{
			NodeStatus:        nodeStatus,
			CephHealth:        cephHealth,
			OSDStatus:         osdStatus,
			DeploymentsStatus: deploymentsStatus,
			HealthSummary:     summary,
			UpdateTime:        time.Now(),
		}

		// Update latest
		m.mu.Lock()
		m.latest = update
		m.mu.Unlock()

		// Send to channel (non-blocking)
		select {
		case m.updates <- update:
		case <-m.ctx.Done():
			return
		default:
			// Channel full, skip this update
		}
	}

	// Process updates from all channels
	for {
		select {
		case <-m.ctx.Done():
			return

		case ns, ok := <-nodeChan:
			if !ok {
				nodeChan = nil
				continue
			}
			nodeStatus = ns
			sendUpdate()

		case ch, ok := <-cephChan:
			if !ok {
				cephChan = nil
				continue
			}
			cephHealth = ch
			sendUpdate()

		case os, ok := <-osdChan:
			if !ok {
				osdChan = nil
				continue
			}
			osdStatus = os
			sendUpdate()

		case ds, ok := <-deploymentsChan:
			if !ok {
				deploymentsChan = nil
				continue
			}
			deploymentsStatus = ds
			sendUpdate()
		}

		// Exit if all channels are closed
		if nodeChan == nil && cephChan == nil && osdChan == nil && deploymentsChan == nil {
			return
		}
	}
}

// StartMonitoring is a convenience function to start monitoring with a config
func StartMonitoring(ctx context.Context, config *MonitorConfig) (*Monitor, <-chan *MonitorUpdate, error) {
	if config.Client == nil {
		return nil, nil, fmt.Errorf("k8s client is required")
	}

	if config.NodeName == "" {
		return nil, nil, fmt.Errorf("node name is required")
	}

	if config.CephNamespace == "" {
		return nil, nil, fmt.Errorf("ceph namespace is required")
	}

	monitor := NewMonitor(config)
	updates := monitor.Start()

	return monitor, updates, nil
}
