package monitoring

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/components"
	"github.com/andri/crook/pkg/tui/views"
)

// LsMonitorConfig holds configuration for the ls monitor
type LsMonitorConfig struct {
	// Context is the parent context for all polling operations.
	// If nil, context.Background() is used.
	Context context.Context

	// Client is the Kubernetes client
	Client *k8s.Client

	// Namespace is the Rook-Ceph namespace
	Namespace string

	// Prefixes are the deployment name prefixes to filter
	Prefixes []string

	// NodeFilter optionally filters resources to a specific node
	NodeFilter string

	// Refresh intervals (independent per resource)
	NodesRefreshInterval       time.Duration
	DeploymentsRefreshInterval time.Duration
	PodsRefreshInterval        time.Duration
	OSDsRefreshInterval        time.Duration
	HeaderRefreshInterval      time.Duration
}

// DefaultLsMonitorConfig returns a config with default refresh intervals
func DefaultLsMonitorConfig(client *k8s.Client, namespace string, prefixes []string) *LsMonitorConfig {
	return &LsMonitorConfig{
		Client:                     client,
		Namespace:                  namespace,
		Prefixes:                   prefixes,
		NodesRefreshInterval:       2 * time.Second,
		DeploymentsRefreshInterval: 2 * time.Second,
		PodsRefreshInterval:        2 * time.Second,
		OSDsRefreshInterval:        5 * time.Second,
		HeaderRefreshInterval:      5 * time.Second,
	}
}

// LsMonitorUpdate contains the latest monitoring data for the ls TUI
type LsMonitorUpdate struct {
	// Nodes is the list of cluster nodes
	Nodes []views.NodeInfo

	// Deployments is the list of Rook-Ceph deployments
	Deployments []views.DeploymentInfo

	// Pods is the list of Rook-Ceph pods
	Pods []views.PodInfo

	// OSDs is the list of Ceph OSDs
	OSDs []views.OSDInfo

	// Header is the cluster header data
	Header *components.ClusterHeaderData

	// UpdateTime is when this update was created
	UpdateTime time.Time

	// Error holds any error that occurred during fetching
	Error error
}

// LsMonitor manages background polling of all ls resources
type LsMonitor struct {
	config   *LsMonitorConfig
	ctx      context.Context
	cancel   context.CancelFunc
	updates  chan *LsMonitorUpdate
	stopOnce sync.Once
	wg       sync.WaitGroup
	mu       sync.RWMutex
	latest   *LsMonitorUpdate
}

// NewLsMonitor creates a new ls monitoring instance
func NewLsMonitor(config *LsMonitorConfig) *LsMonitor {
	parentCtx := context.Background()
	if config != nil && config.Context != nil {
		parentCtx = config.Context
	}
	ctx, cancel := context.WithCancel(parentCtx)

	return &LsMonitor{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		updates: make(chan *LsMonitorUpdate, 10),
		latest: &LsMonitorUpdate{
			UpdateTime: time.Now(),
		},
	}
}

// Start begins background monitoring of all ls resources
func (m *LsMonitor) Start() <-chan *LsMonitorUpdate {
	// Start individual resource pollers
	nodesCh := m.startNodesPoller()
	deploymentsCh := m.startDeploymentsPoller()
	podsCh := m.startPodsPoller()
	osdsCh := m.startOSDsPoller()
	headerCh := m.startHeaderPoller()

	// Start aggregator that combines all updates
	m.wg.Add(1)
	go m.aggregator(nodesCh, deploymentsCh, podsCh, osdsCh, headerCh)

	return m.updates
}

// Stop gracefully stops all monitoring goroutines
func (m *LsMonitor) Stop() {
	m.stopOnce.Do(func() {
		m.cancel()
		m.wg.Wait()
		close(m.updates)
	})
}

// GetLatest returns the most recent monitoring data
func (m *LsMonitor) GetLatest() *LsMonitorUpdate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	return &LsMonitorUpdate{
		Nodes:       m.latest.Nodes,
		Deployments: m.latest.Deployments,
		Pods:        m.latest.Pods,
		OSDs:        m.latest.OSDs,
		Header:      m.latest.Header,
		UpdateTime:  m.latest.UpdateTime,
		Error:       m.latest.Error,
	}
}

// runPoller runs a polling loop with the given interval and fetch function.
// It handles initial fetch, tick-based updates, context cancellation, and error wrapping.
// This generic helper reduces code duplication across the 5 resource pollers.
func runPoller[T any](
	ctx context.Context,
	updates chan<- T,
	interval time.Duration,
	source string,
	fetch func() (T, error),
	onError func(error),
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// send attempts to send data to the updates channel (non-blocking)
	send := func(data T) {
		select {
		case updates <- data:
		case <-ctx.Done():
		default:
		}
	}

	// handleFetch fetches data and either sends it or reports an error
	handleFetch := func() {
		data, err := fetch()
		if err != nil {
			// Suppress context cancellation errors during shutdown
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			onError(fmt.Errorf("%s: %w", source, err))
		} else {
			send(data)
		}
	}

	// Initial fetch
	handleFetch()

	// Poll loop
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			handleFetch()
		}
	}
}

// startNodesPoller starts background node polling
func (m *LsMonitor) startNodesPoller() <-chan []views.NodeInfo {
	updates := make(chan []views.NodeInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.NodesRefreshInterval, "nodes", m.fetchNodes, m.handleError)
	}()
	return updates
}

// fetchNodes fetches all nodes with Ceph pods
func (m *LsMonitor) fetchNodes() ([]views.NodeInfo, error) {
	nodes, err := m.config.Client.ListNodesWithCephPods(m.ctx, m.config.Namespace, m.config.Prefixes)
	if err != nil {
		return nil, err
	}

	result := make([]views.NodeInfo, len(nodes))
	for i, n := range nodes {
		result[i] = views.NodeInfo{
			Name:           n.Name,
			Status:         n.Status,
			Roles:          n.Roles,
			Schedulable:    n.Schedulable,
			Cordoned:       n.Cordoned,
			CephPodCount:   n.CephPodCount,
			Age:            n.Age,
			KubeletVersion: n.KubeletVersion,
		}
	}
	return result, nil
}

// startDeploymentsPoller starts background deployment polling
func (m *LsMonitor) startDeploymentsPoller() <-chan []views.DeploymentInfo {
	updates := make(chan []views.DeploymentInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.DeploymentsRefreshInterval, "deployments", m.fetchDeployments, m.handleError)
	}()
	return updates
}

// fetchDeployments fetches all Ceph deployments
func (m *LsMonitor) fetchDeployments() ([]views.DeploymentInfo, error) {
	deployments, err := m.config.Client.ListCephDeployments(m.ctx, m.config.Namespace, m.config.Prefixes)
	if err != nil {
		return nil, err
	}

	result := make([]views.DeploymentInfo, 0)
	for _, d := range deployments {
		// Apply node filter if specified
		if m.config.NodeFilter != "" && d.NodeName != m.config.NodeFilter {
			continue
		}
		result = append(result, views.DeploymentInfo{
			Name:            d.Name,
			Namespace:       d.Namespace,
			ReadyReplicas:   d.ReadyReplicas,
			DesiredReplicas: d.DesiredReplicas,
			NodeName:        d.NodeName,
			Age:             d.Age,
			Status:          d.Status,
			Type:            d.Type,
			OsdID:           d.OsdID,
		})
	}
	return result, nil
}

// startPodsPoller starts background pod polling
func (m *LsMonitor) startPodsPoller() <-chan []views.PodInfo {
	updates := make(chan []views.PodInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.PodsRefreshInterval, "pods", m.fetchPods, m.handleError)
	}()
	return updates
}

// fetchPods fetches all Ceph pods
func (m *LsMonitor) fetchPods() ([]views.PodInfo, error) {
	pods, err := m.config.Client.ListCephPods(m.ctx, m.config.Namespace, m.config.Prefixes, m.config.NodeFilter)
	if err != nil {
		return nil, err
	}

	result := make([]views.PodInfo, len(pods))
	for i, p := range pods {
		result[i] = views.PodInfo{
			Name:            p.Name,
			Namespace:       p.Namespace,
			Status:          p.Status,
			Ready:           p.ReadyContainers == p.TotalContainers && p.TotalContainers > 0,
			ReadyContainers: p.ReadyContainers,
			TotalContainers: p.TotalContainers,
			Restarts:        p.Restarts,
			NodeName:        p.NodeName,
			Age:             p.Age,
			Type:            p.Type,
			IP:              p.IP,
			OwnerDeployment: p.OwnerDeployment,
		}
	}
	return result, nil
}

// startOSDsPoller starts background OSD polling
func (m *LsMonitor) startOSDsPoller() <-chan []views.OSDInfo {
	updates := make(chan []views.OSDInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.OSDsRefreshInterval, "osds", m.fetchOSDs, m.handleError)
	}()
	return updates
}

// fetchOSDs fetches all OSD info
func (m *LsMonitor) fetchOSDs() ([]views.OSDInfo, error) {
	osds, err := m.config.Client.GetOSDInfoList(m.ctx, m.config.Namespace)
	if err != nil {
		return nil, err
	}

	result := make([]views.OSDInfo, 0)
	for _, o := range osds {
		// Apply node filter if specified
		if m.config.NodeFilter != "" && o.Hostname != m.config.NodeFilter {
			continue
		}
		result = append(result, views.OSDInfo{
			ID:             o.ID,
			Name:           o.Name,
			Hostname:       o.Hostname,
			Status:         o.Status,
			InOut:          o.InOut,
			Weight:         o.Weight,
			Reweight:       o.Reweight,
			DeviceClass:    o.DeviceClass,
			DeploymentName: o.DeploymentName,
		})
	}
	return result, nil
}

// startHeaderPoller starts background cluster header polling
func (m *LsMonitor) startHeaderPoller() <-chan *components.ClusterHeaderData {
	updates := make(chan *components.ClusterHeaderData, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.HeaderRefreshInterval, "header", m.fetchHeader, m.handleError)
	}()
	return updates
}

// fetchHeader fetches cluster header data
func (m *LsMonitor) fetchHeader() (*components.ClusterHeaderData, error) {
	status, err := m.config.Client.GetCephStatus(m.ctx, m.config.Namespace)
	if err != nil {
		return nil, err
	}

	headerData := &components.ClusterHeaderData{
		Health:     status.Health.Status,
		OSDs:       status.OSDMap.NumOSDs,
		OSDsUp:     status.OSDMap.NumUpOSDs,
		OSDsIn:     status.OSDMap.NumInOSDs,
		LastUpdate: time.Now(),
	}

	// Fetch monitor status
	monStatus, monErr := m.config.Client.GetMonitorStatus(m.ctx, m.config.Namespace)
	if monErr == nil {
		headerData.MonsTotal = monStatus.TotalCount
		headerData.MonsInQuorum = monStatus.InQuorum
	}

	// Fetch flags
	flags, flagsErr := m.config.Client.GetCephFlags(m.ctx, m.config.Namespace)
	if flagsErr == nil {
		headerData.NooutSet = flags.NoOut
	}

	// Fetch storage usage
	storage, storageErr := m.config.Client.GetStorageUsage(m.ctx, m.config.Namespace)
	if storageErr == nil {
		headerData.UsedBytes = storage.UsedBytes
		headerData.TotalBytes = storage.TotalBytes
	}

	return headerData, nil
}

// aggregator combines updates from all pollers
func (m *LsMonitor) aggregator(
	nodesCh <-chan []views.NodeInfo,
	deploymentsCh <-chan []views.DeploymentInfo,
	podsCh <-chan []views.PodInfo,
	osdsCh <-chan []views.OSDInfo,
	headerCh <-chan *components.ClusterHeaderData,
) {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return

		case nodes, ok := <-nodesCh:
			if !ok {
				nodesCh = nil
				continue
			}
			m.updateNodes(nodes)
			m.sendUpdate()

		case deployments, ok := <-deploymentsCh:
			if !ok {
				deploymentsCh = nil
				continue
			}
			m.updateDeployments(deployments)
			m.sendUpdate()

		case pods, ok := <-podsCh:
			if !ok {
				podsCh = nil
				continue
			}
			m.updatePods(pods)
			m.sendUpdate()

		case osds, ok := <-osdsCh:
			if !ok {
				osdsCh = nil
				continue
			}
			m.updateOSDs(osds)
			m.sendUpdate()

		case header, ok := <-headerCh:
			if !ok {
				headerCh = nil
				continue
			}
			m.updateHeader(header)
			m.sendUpdate()
		}

		// Exit if all channels are closed
		if nodesCh == nil && deploymentsCh == nil && podsCh == nil && osdsCh == nil && headerCh == nil {
			return
		}
	}
}

// updateNodes updates the nodes in the latest cache
func (m *LsMonitor) updateNodes(nodes []views.NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Nodes = nodes
	m.latest.Error = nil // Clear error on successful update
	m.latest.UpdateTime = time.Now()
}

// updateDeployments updates the deployments in the latest cache
func (m *LsMonitor) updateDeployments(deployments []views.DeploymentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Deployments = deployments
	m.latest.Error = nil // Clear error on successful update
	m.latest.UpdateTime = time.Now()
}

// updatePods updates the pods in the latest cache
func (m *LsMonitor) updatePods(pods []views.PodInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Pods = pods
	m.latest.Error = nil // Clear error on successful update
	m.latest.UpdateTime = time.Now()
}

// updateOSDs updates the OSDs in the latest cache
func (m *LsMonitor) updateOSDs(osds []views.OSDInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.OSDs = osds
	m.latest.Error = nil // Clear error on successful update
	m.latest.UpdateTime = time.Now()
}

// updateHeader updates the header in the latest cache
func (m *LsMonitor) updateHeader(header *components.ClusterHeaderData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Header = header
	m.latest.Error = nil // Clear error on successful update
	m.latest.UpdateTime = time.Now()
}

// updateError updates the error in the latest cache
func (m *LsMonitor) updateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Error = err
	m.latest.UpdateTime = time.Now()
}

// handleError updates the error state and sends an update directly.
// This simplifies error handling by avoiding an intermediate channel.
func (m *LsMonitor) handleError(err error) {
	m.updateError(err)
	m.sendUpdate()
}

// sendUpdate sends the current state to the updates channel
func (m *LsMonitor) sendUpdate() {
	m.mu.RLock()
	update := &LsMonitorUpdate{
		Nodes:       m.latest.Nodes,
		Deployments: m.latest.Deployments,
		Pods:        m.latest.Pods,
		OSDs:        m.latest.OSDs,
		Header:      m.latest.Header,
		UpdateTime:  m.latest.UpdateTime,
		Error:       m.latest.Error,
	}
	m.mu.RUnlock()

	// Non-blocking send
	select {
	case m.updates <- update:
	case <-m.ctx.Done():
	default:
		// Channel full, skip this update
	}
}
