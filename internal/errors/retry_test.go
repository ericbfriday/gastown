package errors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return Transient("test", errors.New("temporary failure"))
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := Retry(fn, config)
	if err != nil {
		t.Errorf("Retry() should succeed, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Retry() attempts = %d, want 3", attempts)
	}
}

func TestRetry_PermanentFailure(t *testing.T) {
	attempts := 0
	permanentErr := Permanent("test", errors.New("permanent failure"))

	fn := func() error {
		attempts++
		return permanentErr
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := Retry(fn, config)
	if err == nil {
		t.Error("Retry() should fail with permanent error")
	}
	// Should fail immediately without retrying permanent errors
	if attempts != 1 {
		t.Errorf("Retry() should not retry permanent errors, attempts = %d", attempts)
	}
}

func TestRetry_MaxAttempts(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		return Transient("test", errors.New("always fails"))
	}

	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := Retry(fn, config)
	if err == nil {
		t.Error("Retry() should fail after max attempts")
	}
	if attempts != 3 {
		t.Errorf("Retry() attempts = %d, want 3", attempts)
	}
}

func TestRetry_OnRetryCallback(t *testing.T) {
	retryCount := 0
	var lastErr error

	fn := func() error {
		if retryCount < 2 {
			return Transient("test", errors.New("temporary failure"))
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
		OnRetry: func(attempt int, err error) {
			retryCount++
			lastErr = err
		},
	}

	err := Retry(fn, config)
	if err != nil {
		t.Errorf("Retry() should succeed, got error: %v", err)
	}
	if retryCount != 2 {
		t.Errorf("OnRetry called %d times, want 2", retryCount)
	}
	if lastErr == nil {
		t.Error("OnRetry should have received error")
	}
}

func TestRetryWithContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	fn := func() error {
		attempts++
		if attempts == 2 {
			cancel() // Cancel after second attempt
		}
		return Transient("test", errors.New("temporary failure"))
	}

	config := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 50 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := RetryWithContext(ctx, fn, config)
	if err == nil {
		t.Error("RetryWithContext() should fail when context is canceled")
	}
	// Should stop retrying after cancellation
	if attempts > 3 {
		t.Errorf("RetryWithContext() should stop after cancellation, attempts = %d", attempts)
	}
}

func TestRetryWithContext_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	attempts := 0
	fn := func() error {
		attempts++
		time.Sleep(50 * time.Millisecond) // Slow operation
		return Transient("test", errors.New("temporary failure"))
	}

	config := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := RetryWithContext(ctx, fn, config)
	if err == nil {
		t.Error("RetryWithContext() should fail when context times out")
	}
	// Should stop before max attempts due to timeout
	if attempts >= 10 {
		t.Errorf("RetryWithContext() should stop before max attempts, attempts = %d", attempts)
	}
}

func TestRetryFunc_WithValue(t *testing.T) {
	attempts := 0
	fn := func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", Transient("test", errors.New("temporary failure"))
		}
		return "success", nil
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	result, err := RetryFunc(fn, config)
	if err != nil {
		t.Errorf("RetryFunc() should succeed, got error: %v", err)
	}
	if result != "success" {
		t.Errorf("RetryFunc() result = %q, want %q", result, "success")
	}
	if attempts != 3 {
		t.Errorf("RetryFunc() attempts = %d, want 3", attempts)
	}
}

func TestRetryFunc_Failure(t *testing.T) {
	fn := func() (int, error) {
		return 0, Permanent("test", errors.New("permanent failure"))
	}

	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
	}

	result, err := RetryFunc(fn, config)
	if err == nil {
		t.Error("RetryFunc() should fail")
	}
	if result != 0 {
		t.Errorf("RetryFunc() should return zero value on error, got %d", result)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("DefaultRetryConfig() MaxAttempts = %d, want 3", config.MaxAttempts)
	}
	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("DefaultRetryConfig() InitialDelay = %v, want 100ms", config.InitialDelay)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("DefaultRetryConfig() Multiplier = %f, want 2.0", config.Multiplier)
	}
}

func TestNetworkRetryConfig(t *testing.T) {
	config := NetworkRetryConfig()

	if config.MaxAttempts != 5 {
		t.Errorf("NetworkRetryConfig() MaxAttempts = %d, want 5", config.MaxAttempts)
	}
	if config.InitialDelay != 500*time.Millisecond {
		t.Errorf("NetworkRetryConfig() InitialDelay = %v, want 500ms", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("NetworkRetryConfig() MaxDelay = %v, want 30s", config.MaxDelay)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name       string
		attempt    int
		initial    time.Duration
		multiplier float64
		max        time.Duration
		expected   time.Duration
	}{
		{
			name:       "first attempt",
			attempt:    1,
			initial:    100 * time.Millisecond,
			multiplier: 2.0,
			max:        10 * time.Second,
			expected:   100 * time.Millisecond,
		},
		{
			name:       "second attempt",
			attempt:    2,
			initial:    100 * time.Millisecond,
			multiplier: 2.0,
			max:        10 * time.Second,
			expected:   200 * time.Millisecond,
		},
		{
			name:       "third attempt",
			attempt:    3,
			initial:    100 * time.Millisecond,
			multiplier: 2.0,
			max:        10 * time.Second,
			expected:   400 * time.Millisecond,
		},
		{
			name:       "exceeds max",
			attempt:    10,
			initial:    100 * time.Millisecond,
			multiplier: 2.0,
			max:        1 * time.Second,
			expected:   1 * time.Second,
		},
		{
			name:       "zero attempt",
			attempt:    0,
			initial:    100 * time.Millisecond,
			multiplier: 2.0,
			max:        10 * time.Second,
			expected:   100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBackoff(tt.attempt, tt.initial, tt.multiplier, tt.max)
			if got != tt.expected {
				t.Errorf("CalculateBackoff() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithRetry_Convenience(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return Transient("test", errors.New("temporary failure"))
		}
		return nil
	}

	err := WithRetry(fn)
	if err != nil {
		t.Errorf("WithRetry() should succeed, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("WithRetry() attempts = %d, want 2", attempts)
	}
}

func TestWithNetworkRetry(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 2 {
			return Transient("network", errors.New("connection timeout"))
		}
		return nil
	}

	err := WithNetworkRetry(fn)
	if err != nil {
		t.Errorf("WithNetworkRetry() should succeed, got error: %v", err)
	}
}

func TestShouldRetryAny(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	wrapped1 := New("test", err1)
	wrapped2 := New("test", err2)

	if !ShouldRetryAny(wrapped1, err1, err2) {
		t.Error("ShouldRetryAny() should find err1")
	}
	if !ShouldRetryAny(wrapped2, err1, err2) {
		t.Error("ShouldRetryAny() should find err2")
	}
	if ShouldRetryAny(wrapped1, err3) {
		t.Error("ShouldRetryAny() should not match err3")
	}
}

func TestRetry_CustomShouldRetry(t *testing.T) {
	attempts := 0
	customErr := errors.New("custom retryable error")

	fn := func() error {
		attempts++
		if attempts < 3 {
			return customErr
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			return errors.Is(err, customErr)
		},
	}

	err := Retry(fn, config)
	if err != nil {
		t.Errorf("Retry() with custom ShouldRetry should succeed, got error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("Retry() attempts = %d, want 3", attempts)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	attempts := 0
	delays := []time.Duration{}
	lastTime := time.Now()

	fn := func() error {
		now := time.Now()
		if attempts > 0 {
			delays = append(delays, now.Sub(lastTime))
		}
		lastTime = now
		attempts++
		if attempts < 4 {
			return Transient("test", errors.New("temporary failure"))
		}
		return nil
	}

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	err := Retry(fn, config)
	if err != nil {
		t.Errorf("Retry() should succeed, got error: %v", err)
	}

	// Verify delays are increasing (exponential backoff)
	if len(delays) < 2 {
		t.Fatal("Not enough delays recorded")
	}

	for i := 1; i < len(delays); i++ {
		if delays[i] <= delays[i-1] {
			t.Errorf("Delay[%d] (%v) should be greater than Delay[%d] (%v)", i, delays[i], i-1, delays[i-1])
		}
	}
}
