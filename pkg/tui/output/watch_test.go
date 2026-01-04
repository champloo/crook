package output_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/andri/crook/pkg/tui/output"
)

func TestWatchRunner_NewWatchRunner(t *testing.T) {
	opts := output.WatchOptions{
		Interval: 2 * time.Second,
		Format:   output.FormatTable,
	}

	wr := output.NewWatchRunner(opts)
	if wr == nil {
		t.Fatal("NewWatchRunner() returned nil")
	}
	if wr.IsRunning() {
		t.Error("NewWatchRunner() should not be running initially")
	}
}

func TestWatchHeader_Format(t *testing.T) {
	var buf bytes.Buffer

	// Use a mock fetch function
	fetchCalled := 0
	opts := output.WatchOptions{
		Interval: 1 * time.Second,
		Format:   output.FormatTable,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			fetchCalled++
			return &output.Data{
				FetchedAt: time.Now(),
			}, nil
		},
		Writer:  &buf,
		Command: "crook ls",
	}

	// Run for just one iteration using a cancelled context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Ignore the context cancelled error
	_ = output.RunWatch(ctx, opts)

	result := buf.String()

	// Check header format
	if !strings.Contains(result, "Every 1.0s") {
		t.Error("Watch header should contain interval")
	}
	if !strings.Contains(result, "crook ls") {
		t.Error("Watch header should contain command")
	}
}

func TestWatch_FetchesData(t *testing.T) {
	var buf bytes.Buffer

	fetchCalled := 0
	opts := output.WatchOptions{
		Interval: 100 * time.Millisecond,
		Format:   output.FormatTable,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			fetchCalled++
			return &output.Data{
				ClusterHealth: &output.ClusterHealth{
					Status: "HEALTH_OK",
				},
				FetchedAt: time.Now(),
			}, nil
		},
		Writer:  &buf,
		Command: "crook ls",
	}

	// Run for a short time to get multiple iterations
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	_ = output.RunWatch(ctx, opts)

	// Should have called fetch at least twice (initial + 1 tick)
	if fetchCalled < 2 {
		t.Errorf("Expected at least 2 fetch calls, got %d", fetchCalled)
	}
}

func TestWatch_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	opts := output.WatchOptions{
		Interval: 1 * time.Second,
		Format:   output.FormatJSON,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			return &output.Data{
				ClusterHealth: &output.ClusterHealth{
					Status: "HEALTH_OK",
				},
				FetchedAt: time.Now(),
			}, nil
		},
		Writer:  &buf,
		Command: "crook ls -o json",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = output.RunWatch(ctx, opts)

	result := buf.String()

	// Should contain JSON structure
	if !strings.Contains(result, "cluster_health") {
		t.Error("JSON output should contain cluster_health")
	}
	if !strings.Contains(result, "HEALTH_OK") {
		t.Error("JSON output should contain health status")
	}
}

func TestWatch_YAMLFormat(t *testing.T) {
	var buf bytes.Buffer

	opts := output.WatchOptions{
		Interval: 1 * time.Second,
		Format:   output.FormatYAML,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			return &output.Data{
				ClusterHealth: &output.ClusterHealth{
					Status: "HEALTH_OK",
				},
				FetchedAt: time.Now(),
			}, nil
		},
		Writer:  &buf,
		Command: "crook ls -o yaml",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = output.RunWatch(ctx, opts)

	result := buf.String()

	// Should contain YAML structure
	if !strings.Contains(result, "cluster_health") {
		t.Error("YAML output should contain cluster_health")
	}
}

func TestWatch_HandlesContextCancellation(t *testing.T) {
	var buf bytes.Buffer

	opts := output.WatchOptions{
		Interval: 10 * time.Second, // Long interval
		Format:   output.FormatTable,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			return &output.Data{FetchedAt: time.Now()}, nil
		},
		Writer:  &buf,
		Command: "crook ls",
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start watch in goroutine
	done := make(chan error)
	go func() {
		done <- output.RunWatch(ctx, opts)
	}()

	// Cancel after a short time
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should complete within reasonable time
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Watch should return nil on context cancellation, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Watch did not exit on context cancellation")
	}
}

func TestWatchRunner_IsRunning(t *testing.T) {
	opts := output.WatchOptions{
		Interval: 100 * time.Millisecond,
		Format:   output.FormatTable,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			return &output.Data{FetchedAt: time.Now()}, nil
		},
		Writer:  &bytes.Buffer{},
		Command: "crook ls",
	}

	wr := output.NewWatchRunner(opts)

	if wr.IsRunning() {
		t.Error("Should not be running before Run()")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start in goroutine
	done := make(chan struct{})
	go func() {
		_ = wr.Run(ctx)
		close(done)
	}()

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	if !wr.IsRunning() {
		t.Error("Should be running after Run()")
	}

	cancel()

	// Wait for completion
	<-done

	if wr.IsRunning() {
		t.Error("Should not be running after completion")
	}
}

func TestWatch_WithNodes(t *testing.T) {
	var buf bytes.Buffer

	opts := output.WatchOptions{
		Interval: 1 * time.Second,
		Format:   output.FormatTable,
		FetchFunc: func(_ context.Context) (*output.Data, error) {
			return &output.Data{
				Nodes: []output.NodeOutput{
					{Name: "worker-1", Status: "Ready"},
					{Name: "worker-2", Status: "Ready"},
				},
				FetchedAt: time.Now(),
			}, nil
		},
		Writer:  &buf,
		Command: "crook ls",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = output.RunWatch(ctx, opts)

	result := buf.String()

	if !strings.Contains(result, "worker-1") {
		t.Error("Output should contain node data")
	}
	if !strings.Contains(result, "NODES") {
		t.Error("Output should contain NODES section")
	}
}
