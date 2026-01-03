package k8s

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// WithRetry wraps a function with retry logic for transient errors
func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt >= cfg.MaxRetries {
			break
		}

		// Handle rate limiting with Retry-After header
		if retryAfter := getRetryAfter(err); retryAfter > 0 {
			backoff = retryAfter
		}

		// Wait with backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Continue to next attempt
		}

		// Calculate next backoff (exponential)
		backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxRetries, lastErr)
}

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
	// Check for context cancellation - don't retry
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Check for Kubernetes API errors
	if statusErr, ok := err.(*errors.StatusError); ok {
		statusCode := statusErr.Status().Code

		// Don't retry client errors (4xx) except for rate limiting and timeout
		if statusCode >= 400 && statusCode < 500 {
			return statusCode == http.StatusTooManyRequests || // 429
				statusCode == http.StatusRequestTimeout // 408
		}

		// Retry server errors (5xx)
		if statusCode >= 500 {
			return true
		}

		// Don't retry other errors
		return false
	}

	// Retry on network-related errors (timeout, connection refused, etc.)
	// These are typically wrapped errors from net package
	// For simplicity, we retry all non-StatusError errors as they're likely transient
	return true
}

// getRetryAfter extracts the Retry-After duration from a rate limit error
func getRetryAfter(err error) time.Duration {
	statusErr, ok := err.(*errors.StatusError)
	if !ok {
		return 0
	}

	if statusErr.Status().Code != http.StatusTooManyRequests {
		return 0
	}

	// Try to extract Retry-After from details
	// Note: This is a simplified implementation. In production, you might want to
	// parse the Retry-After header from the HTTP response if available
	if statusErr.Status().Details != nil {
		if retryAfterSeconds := statusErr.Status().Details.RetryAfterSeconds; retryAfterSeconds > 0 {
			return time.Duration(retryAfterSeconds) * time.Second
		}
	}

	// Check if there's a Retry-After in the message (some errors include it)
	if msg := statusErr.Status().Message; msg != "" {
		// Try to parse "retry after X seconds" from message
		// This is a best-effort attempt
		_ = msg // TODO: parse retry-after from message if needed
	}

	return 0
}

// IsNotFound returns true if the error is a NotFound error
func IsNotFound(err error) bool {
	return errors.IsNotFound(err)
}

// IsAlreadyExists returns true if the error is an AlreadyExists error
func IsAlreadyExists(err error) bool {
	return errors.IsAlreadyExists(err)
}

// IsForbidden returns true if the error is a Forbidden error
func IsForbidden(err error) bool {
	return errors.IsForbidden(err)
}

// IsConflict returns true if the error is a Conflict error
func IsConflict(err error) bool {
	return errors.IsConflict(err)
}

// IsServerError returns true if the error is a server error (5xx)
func IsServerError(err error) bool {
	return errors.IsInternalError(err) ||
		errors.IsServerTimeout(err) ||
		errors.IsServiceUnavailable(err) ||
		errors.IsTimeout(err)
}

// ParseRetryAfterHeader parses the Retry-After header value
// It can be either a delay in seconds or an HTTP date
func ParseRetryAfterHeader(value string) time.Duration {
	// Try parsing as integer (seconds)
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}
