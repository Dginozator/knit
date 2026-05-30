package yds

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
	}
}

// retryPolicy determines if an error should be retried.
// Retries on transient errors (rate limiting, server errors) but not on auth/client errors.
func (r *RetryConfig) retryPolicy(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Don't retry on auth errors or client errors
	if strings.Contains(msg, "AccessDeniedException") ||
		strings.Contains(msg, "InvalidArgumentException") ||
		strings.Contains(msg, "ResourceNotFoundException") {
		return false
	}
	// Retry on rate limiting and server errors
	return true
}

// exponentialBackoff calculates the delay for the given attempt.
func (r *RetryConfig) exponentialBackoff(attempt int) time.Duration {
	delay := r.BaseDelay * time.Duration(1<<uint(attempt))
	if delay > r.MaxDelay {
		return r.MaxDelay
	}
	return delay
}

// WithRetry executes a function with retry logic.
func WithRetry(ctx context.Context, cfg *RetryConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !cfg.retryPolicy(lastErr) || attempt == cfg.MaxRetries {
			return lastErr
		}

		delay := cfg.exponentialBackoff(attempt)
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
