package errors

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig defines the retry behavior configuration.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (including the initial attempt).
	// Default: 3
	MaxAttempts int
	// InitialDelay is the delay before the first retry.
	// Default: 100ms
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	// Default: 10s
	MaxDelay time.Duration
	// Multiplier is the backoff multiplier for exponential backoff.
	// Default: 2.0 (doubles the delay each time)
	Multiplier float64
	// ShouldRetry is a custom function to determine if an error should be retried.
	// If nil, only transient errors are retried.
	ShouldRetry func(error) bool
	// OnRetry is called before each retry attempt with attempt number and error.
	OnRetry func(attempt int, err error)
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		ShouldRetry:  nil, // Use default: retry transient errors
	}
}

// NetworkRetryConfig returns retry configuration optimized for network operations.
func NetworkRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		ShouldRetry:  nil,
	}
}

// FileIORetryConfig returns retry configuration optimized for file I/O operations.
func FileIORetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   2.0,
		ShouldRetry:  nil,
	}
}

// DatabaseRetryConfig returns retry configuration optimized for database operations.
func DatabaseRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		ShouldRetry:  nil,
	}
}

// Retry executes fn with retry logic according to the config.
// Returns the result of fn or the last error encountered.
func Retry(fn func() error, config RetryConfig) error {
	return RetryWithContext(context.Background(), fn, config)
}

// RetryWithContext executes fn with retry logic and context support.
// The function stops retrying if the context is canceled.
func RetryWithContext(ctx context.Context, fn func() error, config RetryConfig) error {
	// Apply defaults
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 10 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.ShouldRetry == nil {
		config.ShouldRetry = IsTransient
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("operation canceled after %d attempts: %w", attempt-1, lastErr)
			}
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if attempt == config.MaxAttempts {
			// Last attempt failed
			return &Error{
				Op:       "retry",
				Err:      lastErr,
				Category: CategoryPermanent,
				Severity: SeverityHigh,
				Message:  fmt.Sprintf("operation failed after %d attempts", config.MaxAttempts),
				Hint:     "The operation failed repeatedly. Check system resources, network connectivity, or service status.",
				Context: map[string]interface{}{
					"attempts": config.MaxAttempts,
				},
			}
		}

		// Check if error is retryable
		if !config.ShouldRetry(err) {
			// Non-retryable error, fail immediately
			return err
		}

		// Call retry callback if provided
		if config.OnRetry != nil {
			config.OnRetry(attempt, err)
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation canceled during retry delay: %w", lastErr)
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	// Should never reach here, but return last error as fallback
	return lastErr
}

// RetryFunc is a helper for operations that return a value.
// It retries the function and returns the result or error.
func RetryFunc[T any](fn func() (T, error), config RetryConfig) (T, error) {
	return RetryFuncWithContext(context.Background(), fn, config)
}

// RetryFuncWithContext is like RetryFunc but with context support.
func RetryFuncWithContext[T any](ctx context.Context, fn func() (T, error), config RetryConfig) (T, error) {
	var result T
	var resultErr error

	err := RetryWithContext(ctx, func() error {
		r, e := fn()
		if e == nil {
			result = r
		}
		resultErr = e
		return e
	}, config)

	if err != nil {
		return result, err
	}
	return result, resultErr
}

// WithRetry wraps a function with default retry logic.
// This is a convenience function for simple retry scenarios.
func WithRetry(fn func() error) error {
	return Retry(fn, DefaultRetryConfig())
}

// WithNetworkRetry wraps a function with network-optimized retry logic.
func WithNetworkRetry(fn func() error) error {
	return Retry(fn, NetworkRetryConfig())
}

// WithFileIORetry wraps a function with file I/O optimized retry logic.
func WithFileIORetry(fn func() error) error {
	return Retry(fn, FileIORetryConfig())
}

// WithDatabaseRetry wraps a function with database-optimized retry logic.
func WithDatabaseRetry(fn func() error) error {
	return Retry(fn, DatabaseRetryConfig())
}

// CalculateBackoff calculates the backoff delay for a given attempt.
// This is useful for manual retry implementations.
func CalculateBackoff(attempt int, initial time.Duration, multiplier float64, max time.Duration) time.Duration {
	if attempt <= 0 {
		return initial
	}

	delay := float64(initial) * math.Pow(multiplier, float64(attempt-1))
	if delay > float64(max) {
		return max
	}
	return time.Duration(delay)
}

// ShouldRetryAny returns true if the error matches any of the target errors.
// This is useful for custom ShouldRetry functions.
func ShouldRetryAny(err error, targets ...error) bool {
	for _, target := range targets {
		if Is(err, target) {
			return true
		}
	}
	return false
}
