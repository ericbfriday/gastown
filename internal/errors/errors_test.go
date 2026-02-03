package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "with op and message",
			err: &Error{
				Op:      "session.Start",
				Message: "failed to start session",
			},
			expected: "session.Start: failed to start session",
		},
		{
			name: "with op and wrapped error",
			err: &Error{
				Op:  "git.Push",
				Err: errors.New("connection refused"),
			},
			expected: "git.Push: connection refused",
		},
		{
			name: "message takes precedence over error",
			err: &Error{
				Op:      "test",
				Message: "custom message",
				Err:     errors.New("wrapped error"),
			},
			expected: "test: custom message",
		},
		{
			name: "only message",
			err: &Error{
				Message: "standalone error",
			},
			expected: "standalone error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_IsTransient(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected bool
	}{
		{
			name:     "transient error",
			err:      &Error{Category: CategoryTransient},
			expected: true,
		},
		{
			name:     "permanent error",
			err:      &Error{Category: CategoryPermanent},
			expected: false,
		},
		{
			name:     "user error",
			err:      &Error{Category: CategoryUser},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.IsTransient()
			if got != tt.expected {
				t.Errorf("IsTransient() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestError_IsRecoverable(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected bool
	}{
		{
			name:     "transient is recoverable",
			err:      &Error{Category: CategoryTransient},
			expected: true,
		},
		{
			name:     "user is recoverable",
			err:      &Error{Category: CategoryUser},
			expected: true,
		},
		{
			name:     "permanent is not recoverable",
			err:      &Error{Category: CategoryPermanent},
			expected: false,
		},
		{
			name:     "system is not recoverable",
			err:      &Error{Category: CategorySystem},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.IsRecoverable()
			if got != tt.expected {
				t.Errorf("IsRecoverable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestError_IsFatal(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected bool
	}{
		{
			name:     "critical severity",
			err:      &Error{Severity: SeverityCritical},
			expected: true,
		},
		{
			name:     "high severity",
			err:      &Error{Severity: SeverityHigh},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.IsFatal()
			if got != tt.expected {
				t.Errorf("IsFatal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestError_WithHint(t *testing.T) {
	err := &Error{Op: "test", Message: "test error"}
	hint := "Try running 'gt status'"

	err = err.WithHint(hint)

	if err.Hint != hint {
		t.Errorf("WithHint() did not set hint, got %q, want %q", err.Hint, hint)
	}
}

func TestError_WithContext(t *testing.T) {
	err := &Error{Op: "test", Message: "test error"}

	err = err.WithContext("attempt", 3)
	err = err.WithContext("delay", "1s")

	if err.Context["attempt"] != 3 {
		t.Errorf("WithContext() did not set attempt")
	}
	if err.Context["delay"] != "1s" {
		t.Errorf("WithContext() did not set delay")
	}
}

func TestError_FullMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "with hint",
			err: &Error{
				Op:      "test",
				Message: "operation failed",
				Hint:    "Try again later",
			},
			expected: "test: operation failed\n\nHow to fix: Try again later",
		},
		{
			name: "without hint",
			err: &Error{
				Op:      "test",
				Message: "operation failed",
			},
			expected: "test: operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.FullMessage()
			if got != tt.expected {
				t.Errorf("FullMessage() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTransient(t *testing.T) {
	baseErr := errors.New("connection timeout")
	err := Transient("network.Connect", baseErr)

	if err.Category != CategoryTransient {
		t.Errorf("Transient() category = %v, want %v", err.Category, CategoryTransient)
	}
	if err.Op != "network.Connect" {
		t.Errorf("Transient() op = %q, want %q", err.Op, "network.Connect")
	}
	if !errors.Is(err, baseErr) {
		t.Errorf("Transient() should wrap base error")
	}
}

func TestPermanent(t *testing.T) {
	baseErr := errors.New("invalid configuration")
	err := Permanent("config.Load", baseErr)

	if err.Category != CategoryPermanent {
		t.Errorf("Permanent() category = %v, want %v", err.Category, CategoryPermanent)
	}
	if err.Severity != SeverityHigh {
		t.Errorf("Permanent() severity = %v, want %v", err.Severity, SeverityHigh)
	}
}

func TestUser(t *testing.T) {
	err := User("polecat.Create", "polecat name already exists")

	if err.Category != CategoryUser {
		t.Errorf("User() category = %v, want %v", err.Category, CategoryUser)
	}
	if err.Severity != SeverityLow {
		t.Errorf("User() severity = %v, want %v", err.Severity, SeverityLow)
	}
}

func TestSystem(t *testing.T) {
	baseErr := errors.New("out of memory")
	err := System("process.Allocate", baseErr)

	if err.Category != CategorySystem {
		t.Errorf("System() category = %v, want %v", err.Category, CategorySystem)
	}
	if err.Severity != SeverityHigh {
		t.Errorf("System() severity = %v, want %v", err.Severity, SeverityHigh)
	}
}

func TestCritical(t *testing.T) {
	baseErr := errors.New("database corruption detected")
	err := Critical("database.Open", baseErr)

	if err.Severity != SeverityCritical {
		t.Errorf("Critical() severity = %v, want %v", err.Severity, SeverityCritical)
	}
	if !err.IsFatal() {
		t.Errorf("Critical() should be fatal")
	}
}

func TestIsTransient_Helper(t *testing.T) {
	transientErr := Transient("test", errors.New("temporary failure"))
	permanentErr := Permanent("test", errors.New("permanent failure"))

	if !IsTransient(transientErr) {
		t.Errorf("IsTransient() should return true for transient error")
	}
	if IsTransient(permanentErr) {
		t.Errorf("IsTransient() should return false for permanent error")
	}
	if IsTransient(nil) {
		t.Errorf("IsTransient() should return false for nil error")
	}
}

func TestIsRecoverable_Helper(t *testing.T) {
	userErr := User("test", "user error")
	systemErr := System("test", errors.New("system error"))

	if !IsRecoverable(userErr) {
		t.Errorf("IsRecoverable() should return true for user error")
	}
	if IsRecoverable(systemErr) {
		t.Errorf("IsRecoverable() should return false for system error")
	}
}

func TestGetHint(t *testing.T) {
	hint := "Try running 'gt status'"
	err := User("test", "error").WithHint(hint)

	got := GetHint(err)
	if got != hint {
		t.Errorf("GetHint() = %q, want %q", got, hint)
	}

	if GetHint(nil) != "" {
		t.Errorf("GetHint() should return empty string for nil error")
	}
	if GetHint(errors.New("plain error")) != "" {
		t.Errorf("GetHint() should return empty string for non-Error type")
	}
}

func TestGetCategory(t *testing.T) {
	err := Transient("test", errors.New("error"))

	if GetCategory(err) != CategoryTransient {
		t.Errorf("GetCategory() = %v, want %v", GetCategory(err), CategoryTransient)
	}
	if GetCategory(nil) != CategoryUnknown {
		t.Errorf("GetCategory() should return CategoryUnknown for nil")
	}
	if GetCategory(errors.New("plain error")) != CategoryUnknown {
		t.Errorf("GetCategory() should return CategoryUnknown for non-Error type")
	}
}

func TestGetSeverity(t *testing.T) {
	err := Critical("test", errors.New("error"))

	if GetSeverity(err) != SeverityCritical {
		t.Errorf("GetSeverity() = %v, want %v", GetSeverity(err), SeverityCritical)
	}
	if GetSeverity(nil) != SeverityMedium {
		t.Errorf("GetSeverity() should return SeverityMedium for nil")
	}
}

func TestError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := New("test", baseErr)

	unwrapped := errors.Unwrap(wrappedErr)
	if unwrapped != baseErr {
		t.Errorf("Unwrap() should return base error")
	}

	// Test with errors.Is
	if !errors.Is(wrappedErr, baseErr) {
		t.Errorf("errors.Is() should find wrapped error")
	}
}

func TestError_As(t *testing.T) {
	// Create a chain of wrapped errors
	baseErr := errors.New("base")
	wrappedErr := New("test", baseErr)
	doubleWrapped := fmt.Errorf("outer: %w", wrappedErr)

	// Test errors.As can find our Error type in the chain
	var e *Error
	if !errors.As(doubleWrapped, &e) {
		t.Errorf("errors.As() should find *Error in chain")
	}
	if e.Op != "test" {
		t.Errorf("errors.As() found wrong error, op = %q", e.Op)
	}
}
