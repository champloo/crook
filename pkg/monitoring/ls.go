package monitoring

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andri/crook/pkg/k8s"
	"github.com/andri/crook/pkg/tui/components"
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

	// NodeFilter optionally filters resources to a specific node
	NodeFilter string

	// K8sRefreshInterval is the refresh interval for Kubernetes API resources (nodes, deployments, pods)
	K8sRefreshInterval time.Duration

	// CephRefreshInterval is the refresh interval for Ceph CLI operations (OSDs, header)
	CephRefreshInterval time.Duration
}

// DefaultLsMonitorConfig returns a config with default refresh intervals
func DefaultLsMonitorConfig(client *k8s.Client, namespace string) *LsMonitorConfig {
	return &LsMonitorConfig{
		Client:              client,
		Namespace:           namespace,
		K8sRefreshInterval:  2 * time.Second,
		CephRefreshInterval: 5 * time.Second,
	}
}

// LsMonitorUpdate contains the latest monitoring data for the ls TUI
type LsMonitorUpdate struct {
	// Nodes is the list of cluster nodes
	Nodes []k8s.NodeInfo

	// Deployments is the list of Rook-Ceph deployments
	Deployments []k8s.DeploymentInfo

	// Pods is the list of Rook-Ceph pods
	Pods []k8s.PodInfo

	// OSDs is the list of Ceph OSDs
	OSDs []k8s.OSDInfo

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
	errors   map[string]error
}

// NewLsMonitor creates a new ls monitoring instance
func NewLsMonitor(config *LsMonitorConfig) (*LsMonitor, error) {
	if err := validateLsMonitorConfig(config); err != nil {
		return nil, err
	}
	parentCtx := context.Background()
	if config.Context != nil {
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
		errors: make(map[string]error),
	}, nil
}

func validateLsMonitorConfig(config *LsMonitorConfig) error {
	if config == nil {
		return fmt.Errorf("ls monitor config is nil")
	}
	if config.Client == nil {
		return fmt.Errorf("ls monitor client is nil")
	}
	if strings.TrimSpace(config.Namespace) == "" {
		return fmt.Errorf("ls monitor namespace is empty")
	}
	if config.K8sRefreshInterval <= 0 {
		return fmt.Errorf("ls monitor k8s refresh interval must be > 0")
	}
	if config.CephRefreshInterval <= 0 {
		return fmt.Errorf("ls monitor ceph refresh interval must be > 0")
	}
	return nil
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

	// Return a copy to avoid callers mutating internal state.
	return cloneUpdate(m.latest)
}

func cloneUpdate(update *LsMonitorUpdate) *LsMonitorUpdate {
	if update == nil {
		return nil
	}
	return &LsMonitorUpdate{
		Nodes:       cloneNodes(update.Nodes),
		Deployments: append([]k8s.DeploymentInfo(nil), update.Deployments...),
		Pods:        append([]k8s.PodInfo(nil), update.Pods...),
		OSDs:        append([]k8s.OSDInfo(nil), update.OSDs...),
		Header:      cloneHeader(update.Header),
		UpdateTime:  update.UpdateTime,
		Error:       update.Error,
	}
}

func cloneNodes(nodes []k8s.NodeInfo) []k8s.NodeInfo {
	if nodes == nil {
		return nil
	}
	cloned := make([]k8s.NodeInfo, len(nodes))
	copy(cloned, nodes)
	for i := range cloned {
		if nodes[i].Roles != nil {
			roles := make([]string, len(nodes[i].Roles))
			copy(roles, nodes[i].Roles)
			cloned[i].Roles = roles
		}
	}
	return cloned
}

func cloneHeader(header *components.ClusterHeaderData) *components.ClusterHeaderData {
	if header == nil {
		return nil
	}
	cloned := *header
	return &cloned
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
	onError func(string, error),
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
			onError(source, fmt.Errorf("%s: %w", source, err))
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
func (m *LsMonitor) startNodesPoller() <-chan []k8s.NodeInfo {
	updates := make(chan []k8s.NodeInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.K8sRefreshInterval, "nodes", m.fetchNodes, m.handleError)
	}()
	return updates
}

// fetchNodes fetches all nodes with Ceph pods
func (m *LsMonitor) fetchNodes() ([]k8s.NodeInfo, error) {
	return m.config.Client.ListNodesWithCephPods(m.ctx, m.config.Namespace)
}

// startDeploymentsPoller starts background deployment polling
func (m *LsMonitor) startDeploymentsPoller() <-chan []k8s.DeploymentInfo {
	updates := make(chan []k8s.DeploymentInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.K8sRefreshInterval, "deployments", m.fetchDeployments, m.handleError)
	}()
	return updates
}

// fetchDeployments fetches all Ceph deployments
func (m *LsMonitor) fetchDeployments() ([]k8s.DeploymentInfo, error) {
	deployments, err := m.config.Client.ListCephDeployments(m.ctx, m.config.Namespace)
	if err != nil {
		return nil, err
	}

	// Apply node filter if specified
	if m.config.NodeFilter == "" {
		return deployments, nil
	}

	result := make([]k8s.DeploymentInfo, 0, len(deployments))
	for _, d := range deployments {
		if d.NodeName == m.config.NodeFilter {
			result = append(result, d)
		}
	}
	return result, nil
}

// startPodsPoller starts background pod polling
func (m *LsMonitor) startPodsPoller() <-chan []k8s.PodInfo {
	updates := make(chan []k8s.PodInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.K8sRefreshInterval, "pods", m.fetchPods, m.handleError)
	}()
	return updates
}

// fetchPods fetches all Ceph pods
func (m *LsMonitor) fetchPods() ([]k8s.PodInfo, error) {
	return m.config.Client.ListCephPods(m.ctx, m.config.Namespace, m.config.NodeFilter)
}

// startOSDsPoller starts background OSD polling
func (m *LsMonitor) startOSDsPoller() <-chan []k8s.OSDInfo {
	updates := make(chan []k8s.OSDInfo, 1)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer close(updates)
		runPoller(m.ctx, updates, m.config.CephRefreshInterval, "osds", m.fetchOSDs, m.handleError)
	}()
	return updates
}

// fetchOSDs fetches all OSD info
func (m *LsMonitor) fetchOSDs() ([]k8s.OSDInfo, error) {
	osds, err := m.config.Client.GetOSDInfoList(m.ctx, m.config.Namespace)
	if err != nil {
		return nil, err
	}

	// Apply node filter if specified
	if m.config.NodeFilter == "" {
		return osds, nil
	}

	result := make([]k8s.OSDInfo, 0, len(osds))
	for _, o := range osds {
		if o.Hostname == m.config.NodeFilter {
			result = append(result, o)
		}
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
		runPoller(m.ctx, updates, m.config.CephRefreshInterval, "header", m.fetchHeader, m.handleError)
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
	nodesCh <-chan []k8s.NodeInfo,
	deploymentsCh <-chan []k8s.DeploymentInfo,
	podsCh <-chan []k8s.PodInfo,
	osdsCh <-chan []k8s.OSDInfo,
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
func (m *LsMonitor) updateNodes(nodes []k8s.NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Nodes = nodes
	m.clearErrorLocked("nodes")
	m.latest.UpdateTime = time.Now()
}

// updateDeployments updates the deployments in the latest cache
func (m *LsMonitor) updateDeployments(deployments []k8s.DeploymentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Deployments = deployments
	m.clearErrorLocked("deployments")
	m.latest.UpdateTime = time.Now()
}

// updatePods updates the pods in the latest cache
func (m *LsMonitor) updatePods(pods []k8s.PodInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Pods = pods
	m.clearErrorLocked("pods")
	m.latest.UpdateTime = time.Now()
}

// updateOSDs updates the OSDs in the latest cache
func (m *LsMonitor) updateOSDs(osds []k8s.OSDInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.OSDs = osds
	m.clearErrorLocked("osds")
	m.latest.UpdateTime = time.Now()
}

// updateHeader updates the header in the latest cache
func (m *LsMonitor) updateHeader(header *components.ClusterHeaderData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latest.Header = header
	m.clearErrorLocked("header")
	m.latest.UpdateTime = time.Now()
}

func (m *LsMonitor) clearErrorLocked(source string) {
	if m.errors == nil {
		return
	}
	delete(m.errors, source)
	m.latest.Error = combineErrors(m.errors)
}

// updateError updates the error in the latest cache
func (m *LsMonitor) updateError(source string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.errors == nil {
		m.errors = make(map[string]error)
	}
	m.errors[source] = err
	m.latest.Error = combineErrors(m.errors)
	m.latest.UpdateTime = time.Now()
}

func combineErrors(errs map[string]error) error {
	if len(errs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(errs))
	for key := range errs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	list := make([]error, 0, len(keys))
	for _, key := range keys {
		if errs[key] != nil {
			list = append(list, errs[key])
		}
	}
	if len(list) == 0 {
		return nil
	}
	return errors.Join(list...)
}

// handleError updates the error state and sends an update directly.
// This simplifies error handling by avoiding an intermediate channel.
func (m *LsMonitor) handleError(source string, err error) {
	m.updateError(source, err)
	m.sendUpdate()
}

// sendUpdate sends the current state to the updates channel
func (m *LsMonitor) sendUpdate() {
	m.mu.RLock()
	update := cloneUpdate(m.latest)
	m.mu.RUnlock()

	// Non-blocking send
	select {
	case m.updates <- update:
	case <-m.ctx.Done():
	default:
		// Channel full, skip this update
	}
}
