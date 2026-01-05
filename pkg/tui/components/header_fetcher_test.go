package components

import (
	"context"
	"testing"
	"time"
)

func TestNewHeaderFetcher(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace:       "rook-ceph",
		RefreshInterval: 10 * time.Second,
	})

	if f == nil {
		t.Fatal("NewHeaderFetcher returned nil")
	}

	if f.config.RefreshInterval != 10*time.Second {
		t.Errorf("expected RefreshInterval=10s, got %v", f.config.RefreshInterval)
	}
}

func TestNewHeaderFetcher_DefaultInterval(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace: "rook-ceph",
		// RefreshInterval not set, should default to 5s
	})

	if f.config.RefreshInterval != 5*time.Second {
		t.Errorf("expected default RefreshInterval=5s, got %v", f.config.RefreshInterval)
	}
}

func TestHeaderFetcher_Stop(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace:       "rook-ceph",
		RefreshInterval: 1 * time.Second,
	})

	// Start creates the context
	f.Start()

	// Context should be valid
	if f.ctx == nil {
		t.Fatal("expected ctx to be set after Start")
	}
	if f.ctx.Err() != nil {
		t.Fatal("expected ctx to not be cancelled immediately")
	}

	// Stop should cancel
	f.Stop()

	if f.ctx.Err() != context.Canceled {
		t.Errorf("expected ctx.Err()=context.Canceled after Stop, got %v", f.ctx.Err())
	}
}

func TestHeaderFetcher_HandleTick_AfterStop(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace:       "rook-ceph",
		RefreshInterval: 1 * time.Second,
	})

	f.Start()
	f.Stop()

	// HandleTick should return nil after Stop (context cancelled)
	cmd := f.HandleTick()
	if cmd != nil {
		t.Error("expected HandleTick to return nil after Stop")
	}
}

func TestHeaderFetcher_HandleTick_BeforeStart(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace:       "rook-ceph",
		RefreshInterval: 1 * time.Second,
	})

	// HandleTick before Start - ctx is nil, should still return commands
	// (because ctx == nil does not trigger the early return)
	cmd := f.HandleTick()

	// Should return batch of commands
	if cmd == nil {
		t.Error("expected HandleTick to return commands even before Start")
	}
}

func TestHeaderFetcher_Stop_BeforeStart(t *testing.T) {
	f := NewHeaderFetcher(HeaderFetcherConfig{
		Namespace: "rook-ceph",
	})

	// Stop before Start should not panic
	f.Stop()
}

func TestHeaderTickCmd(t *testing.T) {
	cmd := HeaderTickCmd(1 * time.Second)
	if cmd == nil {
		t.Error("HeaderTickCmd should return a non-nil command")
	}
}
