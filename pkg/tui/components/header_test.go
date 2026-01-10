package components

import (
	"strings"
	"testing"
	"time"
)

func TestNewClusterHeader(t *testing.T) {
	h := NewClusterHeader()
	if h == nil {
		t.Fatal("NewClusterHeader returned nil")
	}
	if !h.loading {
		t.Error("expected loading=true for new header")
	}
}

func TestClusterHeader_SetData(t *testing.T) {
	h := NewClusterHeader()

	data := &ClusterHeaderData{
		Health:       "HEALTH_OK",
		OSDs:         5,
		OSDsUp:       5,
		OSDsIn:       5,
		MonsTotal:    3,
		MonsInQuorum: 3,
		NooutSet:     false,
		UsedBytes:    1024 * 1024 * 1024,     // 1 GiB
		TotalBytes:   4 * 1024 * 1024 * 1024, // 4 GiB
		LastUpdate:   time.Now(),
	}

	h.SetData(data)

	if h.loading {
		t.Error("expected loading=false after SetData")
	}
	if h.GetData() != data {
		t.Error("expected GetData to return set data")
	}
	if h.HasError() {
		t.Error("expected no error after SetData")
	}
}

func TestClusterHeader_View_Loading(t *testing.T) {
	h := NewClusterHeader()
	view := h.Render()

	if !strings.Contains(view, "loading") {
		t.Errorf("expected 'loading' in view, got: %s", view)
	}
}

func TestClusterHeader_View_HealthOK(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		Health:       "HEALTH_OK",
		OSDs:         5,
		OSDsUp:       5,
		OSDsIn:       5,
		MonsTotal:    3,
		MonsInQuorum: 3,
		NooutSet:     false,
		UsedBytes:    1024 * 1024 * 1024,
		TotalBytes:   4 * 1024 * 1024 * 1024,
		LastUpdate:   time.Now(),
	})
	h.SetWidth(100)

	view := h.Render()

	if !strings.Contains(view, "HEALTH_OK") {
		t.Errorf("expected 'HEALTH_OK' in view, got: %s", view)
	}
	if !strings.Contains(view, "OSDs") {
		t.Errorf("expected 'OSDs' in view, got: %s", view)
	}
	if !strings.Contains(view, "5/5") {
		t.Errorf("expected '5/5' in view for full OSDs, got: %s", view)
	}
	if !strings.Contains(view, "MONs") {
		t.Errorf("expected 'MONs' in view, got: %s", view)
	}
	if !strings.Contains(view, "3/3") {
		t.Errorf("expected '3/3' in view for full monitors, got: %s", view)
	}
	if !strings.Contains(view, "Storage") {
		t.Errorf("expected 'Storage' in view, got: %s", view)
	}
}

func TestClusterHeader_View_HealthWarn(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		Health:       "HEALTH_WARN",
		OSDs:         5,
		OSDsUp:       4, // One down
		OSDsIn:       5,
		MonsTotal:    3,
		MonsInQuorum: 3,
		NooutSet:     true,
		UsedBytes:    1024 * 1024 * 1024,
		TotalBytes:   4 * 1024 * 1024 * 1024,
		LastUpdate:   time.Now(),
	})
	h.SetWidth(100)

	view := h.Render()

	if !strings.Contains(view, "HEALTH_WARN") {
		t.Errorf("expected 'HEALTH_WARN' in view, got: %s", view)
	}
	if !strings.Contains(view, "4/5") {
		t.Errorf("expected '4/5' in view for degraded OSDs, got: %s", view)
	}
}

func TestClusterHeader_View_NooutSet(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		Health:       "HEALTH_OK",
		OSDs:         5,
		OSDsUp:       5,
		OSDsIn:       5,
		MonsTotal:    3,
		MonsInQuorum: 3,
		NooutSet:     true,
		UsedBytes:    1024 * 1024 * 1024,
		TotalBytes:   4 * 1024 * 1024 * 1024,
		LastUpdate:   time.Now(),
	})
	h.SetWidth(100)

	view := h.Render()

	// Should contain noout with checkmark
	if !strings.Contains(view, "noout") {
		t.Errorf("expected 'noout' in view when set, got: %s", view)
	}
}

func TestClusterHeader_View_Compact(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		Health:       "HEALTH_OK",
		OSDs:         5,
		OSDsUp:       5,
		OSDsIn:       5,
		MonsTotal:    3,
		MonsInQuorum: 3,
		NooutSet:     false,
		UsedBytes:    1024 * 1024 * 1024,
		TotalBytes:   4 * 1024 * 1024 * 1024,
		LastUpdate:   time.Now(),
	})
	h.SetWidth(60) // Narrow terminal

	view := h.Render()

	// Compact view should be single line (no newlines, or just one for wrapping)
	if strings.Count(view, "\n") > 0 {
		t.Errorf("expected compact view to be single line, got: %s", view)
	}
	if !strings.Contains(view, "HEALTH_OK") {
		t.Errorf("expected 'HEALTH_OK' in compact view, got: %s", view)
	}
}

func TestClusterHeader_Error(t *testing.T) {
	h := NewClusterHeader()
	h.SetError(errTestError)

	if !h.HasError() {
		t.Error("expected HasError()=true after SetError")
	}
	if h.GetError() != errTestError {
		t.Error("expected GetError to return set error")
	}

	view := h.Render()
	if !strings.Contains(view, "error") {
		t.Errorf("expected 'error' in view, got: %s", view)
	}
}

var errTestError = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestClusterHeader_renderLastUpdated(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  time.Duration
		contains string
	}{
		{"just now", 5 * time.Second, "5s ago"},
		{"minutes", 2 * time.Minute, "2m ago"},
		{"hours", 2 * time.Hour, "2h ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewClusterHeader()
			h.SetData(&ClusterHeaderData{
				LastUpdate: time.Now().Add(-tt.elapsed),
			})
			h.SetWidth(100)

			result := h.renderLastUpdated()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected %q in result, got: %s", tt.contains, result)
			}
		})
	}
}

func TestClusterHeader_renderLastUpdated_Never(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		// LastUpdate is zero value
	})

	result := h.renderLastUpdated()
	if !strings.Contains(result, "never") {
		t.Errorf("expected 'never' in result for zero time, got: %s", result)
	}
}

func TestClusterHeader_renderStorageUsage_ZeroBytes(t *testing.T) {
	h := NewClusterHeader()
	h.SetData(&ClusterHeaderData{
		TotalBytes: 0,
	})

	result := h.renderStorageUsage()
	if !strings.Contains(result, "N/A") {
		t.Errorf("expected 'N/A' for zero storage, got: %s", result)
	}
}

func TestClusterHeader_renderMonStats_Degraded(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		inQuorum  int
		isWarning bool
		isError   bool
	}{
		{"all healthy", 3, 3, false, false},
		{"one down", 3, 2, true, false},
		{"no quorum", 3, 1, false, true},
		{"none up", 3, 0, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewClusterHeader()
			h.SetData(&ClusterHeaderData{
				MonsTotal:    tt.total,
				MonsInQuorum: tt.inQuorum,
			})

			result := h.renderMonStats()
			// Just verify it renders without panic
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestHeaderUpdateMsg(t *testing.T) {
	h := NewClusterHeader()

	// Test updating with data
	msg := HeaderUpdateMsg{
		Data: &ClusterHeaderData{
			Health: "HEALTH_OK",
		},
	}

	newModel, _ := h.Update(msg)
	header, ok := newModel.(*ClusterHeader)
	if !ok {
		t.Fatal("expected *ClusterHeader type")
	}

	if header.loading {
		t.Error("expected loading=false after update")
	}
	if header.data == nil {
		t.Error("expected data to be set")
	}
	if header.data.Health != "HEALTH_OK" {
		t.Errorf("expected Health=HEALTH_OK, got %s", header.data.Health)
	}
}

func TestHeaderUpdateMsg_WithError(t *testing.T) {
	h := NewClusterHeader()

	msg := HeaderUpdateMsg{
		Error: errTestError,
	}

	newModel, _ := h.Update(msg)
	header, ok := newModel.(*ClusterHeader)
	if !ok {
		t.Fatal("expected *ClusterHeader type")
	}

	if !header.HasError() {
		t.Error("expected HasError()=true after error update")
	}
}
