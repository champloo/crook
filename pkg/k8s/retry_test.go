package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	err := WithRetry(ctx, cfg, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	err := WithRetry(ctx, cfg, func() error {
		callCount++
		if callCount < 3 {
			// Return a retryable error (500)
			return kerrors.NewInternalError(errors.New("server error"))
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error after retries, got: %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWithRetry_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	cfg := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	err := WithRetry(ctx, cfg, func() error {
		callCount++
		return kerrors.NewInternalError(errors.New("persistent server error"))
	})

	if err == nil {
		t.Fatal("expected error after max retries, got nil")
	}
	// Should be called MaxRetries + 1 times (initial + retries)
	if callCount != 3 {
		t.Errorf("expected 3 calls (1 initial + 2 retries), got %d", callCount)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultRetryConfig()

	callCount := 0
	err := WithRetry(ctx, cfg, func() error {
		callCount++
		return kerrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test-pod")
	})

	if err == nil {
		t.Fatal("expected error for non-retryable error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (no retries for 404), got %d", callCount)
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	errCh := make(chan error, 1)

	go func() {
		errCh <- WithRetry(ctx, cfg, func() error {
			callCount++
			return kerrors.NewInternalError(errors.New("server error"))
		})
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-errCh
	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}

	// Should have attempted once, then cancelled during backoff
	if callCount < 1 {
		t.Errorf("expected at least 1 call before cancellation, got %d", callCount)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "500 Internal Server Error",
			err:       kerrors.NewInternalError(errors.New("internal error")),
			retryable: true,
		},
		{
			name:      "503 Service Unavailable",
			err:       kerrors.NewServiceUnavailable("service unavailable"),
			retryable: true,
		},
		{
			name:      "429 Too Many Requests",
			err:       kerrors.NewTooManyRequests("rate limited", 1),
			retryable: true,
		},
		{
			name:      "404 Not Found",
			err:       kerrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test"),
			retryable: false,
		},
		{
			name:      "403 Forbidden",
			err:       kerrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "test", errors.New("forbidden")),
			retryable: false,
		},
		{
			name:      "400 Bad Request",
			err:       kerrors.NewBadRequest("bad request"),
			retryable: false,
		},
		{
			name:      "context.Canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context.DeadlineExceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "generic error (network)",
			err:       errors.New("connection refused"),
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("expected isRetryable=%v for %s, got %v", tt.retryable, tt.name, result)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("expected InitialBackoff=1s, got %v", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("expected MaxBackoff=30s, got %v", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("expected BackoffMultiplier=2.0, got %v", cfg.BackoffMultiplier)
	}
}

func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "seconds",
			value:    "5",
			expected: 5 * time.Second,
		},
		{
			name:     "zero",
			value:    "0",
			expected: 0,
		},
		{
			name:     "invalid",
			value:    "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRetryAfterHeader(tt.value)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		err := kerrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test")
		if !IsNotFound(err) {
			t.Error("expected IsNotFound to return true")
		}

		err = kerrors.NewInternalError(errors.New("internal error"))
		if IsNotFound(err) {
			t.Error("expected IsNotFound to return false")
		}
	})

	t.Run("IsForbidden", func(t *testing.T) {
		err := kerrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "test", errors.New("forbidden"))
		if !IsForbidden(err) {
			t.Error("expected IsForbidden to return true")
		}
	})

	t.Run("IsServerError", func(t *testing.T) {
		err := kerrors.NewInternalError(errors.New("internal error"))
		if !IsServerError(err) {
			t.Error("expected IsServerError to return true for internal error")
		}

		err = kerrors.NewServiceUnavailable("unavailable")
		if !IsServerError(err) {
			t.Error("expected IsServerError to return true for service unavailable")
		}

		err = kerrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test")
		if IsServerError(err) {
			t.Error("expected IsServerError to return false for not found")
		}
	})
}

func TestWithRetry_ExponentialBackoff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backoff timing test in short mode")
	}

	ctx := context.Background()
	cfg := RetryConfig{
		MaxRetries:        2,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	start := time.Now()
	callCount := 0

	_ = WithRetry(ctx, cfg, func() error {
		callCount++
		return kerrors.NewInternalError(errors.New("server error"))
	})

	duration := time.Since(start)

	// Should have waited ~100ms + ~200ms = ~300ms total
	// Allow some tolerance for timing
	expectedMin := 250 * time.Millisecond
	expectedMax := 400 * time.Millisecond

	if duration < expectedMin || duration > expectedMax {
		t.Errorf("expected duration between %v and %v, got %v", expectedMin, expectedMax, duration)
	}
}
