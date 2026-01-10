package components

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/andri/crook/pkg/k8s"
)

// HeaderFetcherConfig holds configuration for the header fetcher
type HeaderFetcherConfig struct {
	// Client is the Kubernetes client
	Client *k8s.Client

	// Namespace is the Ceph cluster namespace
	Namespace string

	// RefreshInterval is how often to refresh the header data (default: 5s)
	RefreshInterval time.Duration
}

// HeaderFetcher fetches cluster header data in the background
type HeaderFetcher struct {
	config HeaderFetcherConfig
	ctx    context.Context
	cancel context.CancelFunc
}

// NewHeaderFetcher creates a new header fetcher
func NewHeaderFetcher(cfg HeaderFetcherConfig) *HeaderFetcher {
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = 5 * time.Second
	}
	return &HeaderFetcher{
		config: cfg,
	}
}

// HeaderFetchCmd returns a command that fetches header data.
// The context is used for cancellation - pass a cancelable context
// to allow stopping in-flight fetches.
func HeaderFetchCmd(ctx context.Context, client *k8s.Client, namespace string) tea.Cmd {
	return func() tea.Msg {
		data, err := FetchClusterHeaderData(ctx, client, namespace)
		return HeaderUpdateMsg{Data: data, Error: err}
	}
}

// HeaderTickMsg triggers a periodic header refresh
type HeaderTickMsg struct{}

// HeaderTickCmd returns a command that schedules the next header tick
func HeaderTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return HeaderTickMsg{}
	})
}

// FetchClusterHeaderData fetches all cluster data needed for the header
func FetchClusterHeaderData(ctx context.Context, client *k8s.Client, namespace string) (*ClusterHeaderData, error) {
	data := &ClusterHeaderData{
		LastUpdate: time.Now(),
	}

	// Fetch Ceph status (health + OSD counts)
	status, err := client.GetCephStatus(ctx, namespace)
	if err != nil {
		return nil, err
	}

	data.Health = status.Health.Status
	data.OSDs = status.OSDMap.NumOSDs
	data.OSDsUp = status.OSDMap.NumUpOSDs
	data.OSDsIn = status.OSDMap.NumInOSDs

	// Fetch monitor status
	monStatus, err := client.GetMonitorStatus(ctx, namespace)
	if err != nil {
		// Non-fatal: continue with partial data
		data.MonsTotal = 0
		data.MonsInQuorum = 0
	} else {
		data.MonsTotal = monStatus.TotalCount
		data.MonsInQuorum = monStatus.InQuorum
	}

	// Fetch Ceph flags
	flags, err := client.GetCephFlags(ctx, namespace)
	if err != nil {
		// Non-fatal: continue with partial data
		data.NooutSet = false
	} else {
		data.NooutSet = flags.NoOut
	}

	// Fetch storage usage
	storage, err := client.GetStorageUsage(ctx, namespace)
	if err != nil {
		// Non-fatal: continue with partial data
		data.UsedBytes = 0
		data.TotalBytes = 0
	} else {
		data.UsedBytes = storage.UsedBytes
		data.TotalBytes = storage.TotalBytes
	}

	return data, nil
}

// Start begins background data fetching (returns initial fetch command and tick command)
func (f *HeaderFetcher) Start() tea.Cmd {
	f.ctx, f.cancel = context.WithCancel(context.Background())
	return tea.Batch(
		HeaderFetchCmd(f.ctx, f.config.Client, f.config.Namespace),
		HeaderTickCmd(f.config.RefreshInterval),
	)
}

// Stop stops background data fetching
func (f *HeaderFetcher) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
}

// HandleTick handles a tick message and returns the next fetch + tick commands
func (f *HeaderFetcher) HandleTick() tea.Cmd {
	if f.ctx != nil && f.ctx.Err() != nil {
		// Context cancelled, don't schedule more ticks
		return nil
	}
	return tea.Batch(
		HeaderFetchCmd(f.ctx, f.config.Client, f.config.Namespace),
		HeaderTickCmd(f.config.RefreshInterval),
	)
}

// FetchOnce performs a single fetch and returns the result
func (f *HeaderFetcher) FetchOnce() tea.Cmd {
	// Use a background context for one-off fetches
	return HeaderFetchCmd(context.Background(), f.config.Client, f.config.Namespace)
}
