package errors

import (
	"errors"
	"fmt"
)

// ErrorCategory defines the type of error.
type ErrorCategory int

const (
	// CategoryUnknown represents an uncategorized error.
	CategoryUnknown ErrorCategory = iota
	// CategoryTransient represents a temporary error that may succeed on retry.
	CategoryTransient
	// CategoryPermanent represents an error that will not succeed on retry.
	CategoryPermanent
	// CategoryUser represents an error caused by user input or actions.
	CategoryUser
	// CategorySystem represents an internal system error.
	CategorySystem
)

// String returns the string representation of an ErrorCategory.
func (c ErrorCategory) String() string {
	switch c {
	case CategoryTransient:
		return "transient"
	case CategoryPermanent:
		return "permanent"
	case CategoryUser:
		return "user"
	case CategorySystem:
		return "system"
	default:
		return "unknown"
	}
}

// Severity defines the impact level of an error.
type Severity int

const (
	// SeverityLow represents a minor error that doesn't significantly impact functionality.
	SeverityLow Severity = iota
	// SeverityMedium represents a moderate error that may limit functionality.
	SeverityMedium
	// SeverityHigh represents a serious error that significantly impacts functionality.
	SeverityHigh
	// SeverityCritical represents a fatal error that prevents operation.
	SeverityCritical
)

// String returns the string representation of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Error represents an enhanced error with context and metadata.
type Error struct {
	// Op is the operation that failed (e.g., "session.Start", "git.Push").
	Op string
	// Code is a machine-readable error code for programmatic handling.
	Code string
	// Category classifies the error type for retry and handling decisions.
	Category ErrorCategory
	// Severity indicates the impact level of the error.
	Severity Severity
	// Err is the underlying error that caused this error.
	Err error
	// Message is a user-friendly error message.
	Message string
	// Hint provides actionable recovery suggestions for the user.
	Hint string
	// Context provides additional contextual information.
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Op == "" {
		if e.Message != "" {
			return e.Message
		}
		if e.Err != nil {
			return e.Err.Error()
		}
		return "unknown error"
	}

	msg := e.Op
	if e.Message != "" {
		msg = fmt.Sprintf("%s: %s", msg, e.Message)
	} else if e.Err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Err)
	}

	return msg
}

// Unwrap returns the underlying error for errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}

// IsTransient returns true if the error is transient and may succeed on retry.
func (e *Error) IsTransient() bool {
	return e.Category == CategoryTransient
}

// IsRecoverable returns true if the error is recoverable (transient or user).
func (e *Error) IsRecoverable() bool {
	return e.Category == CategoryTransient || e.Category == CategoryUser
}

// IsFatal returns true if the error is critical severity.
func (e *Error) IsFatal() bool {
	return e.Severity == SeverityCritical
}

// WithHint adds or updates the recovery hint.
func (e *Error) WithHint(hint string) *Error {
	e.Hint = hint
	return e
}

// WithContext adds contextual information.
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// FullMessage returns the complete error message with hint.
func (e *Error) FullMessage() string {
	msg := e.Error()
	if e.Hint != "" {
		msg = fmt.Sprintf("%s\n\nHow to fix: %s", msg, e.Hint)
	}
	return msg
}

// New creates a new Error with the given operation and underlying error.
func New(op string, err error) *Error {
	return &Error{
		Op:       op,
		Err:      err,
		Category: CategoryUnknown,
		Severity: SeverityMedium,
	}
}

// Newf creates a new Error with a formatted message.
func Newf(op string, format string, args ...interface{}) *Error {
	return &Error{
		Op:       op,
		Message:  fmt.Sprintf(format, args...),
		Category: CategoryUnknown,
		Severity: SeverityMedium,
	}
}

// Transient creates a new transient error that should be retried.
func Transient(op string, err error) *Error {
	return &Error{
		Op:       op,
		Err:      err,
		Category: CategoryTransient,
		Severity: SeverityMedium,
	}
}

// Permanent creates a new permanent error that should not be retried.
func Permanent(op string, err error) *Error {
	return &Error{
		Op:       op,
		Err:      err,
		Category: CategoryPermanent,
		Severity: SeverityHigh,
	}
}

// User creates a new user error caused by invalid input or actions.
func User(op string, message string) *Error {
	return &Error{
		Op:       op,
		Message:  message,
		Category: CategoryUser,
		Severity: SeverityLow,
	}
}

// System creates a new system error for internal failures.
func System(op string, err error) *Error {
	return &Error{
		Op:       op,
		Err:      err,
		Category: CategorySystem,
		Severity: SeverityHigh,
	}
}

// Critical creates a new critical error that prevents operation.
func Critical(op string, err error) *Error {
	return &Error{
		Op:       op,
		Err:      err,
		Category: CategoryPermanent,
		Severity: SeverityCritical,
	}
}

// As is a helper for errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Is is a helper for errors.Is.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// Unwrap is a helper for errors.Unwrap.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// IsTransient checks if any error in the chain is transient.
func IsTransient(err error) bool {
	var e *Error
	if As(err, &e) {
		return e.IsTransient()
	}
	return false
}

// IsRecoverable checks if any error in the chain is recoverable.
func IsRecoverable(err error) bool {
	var e *Error
	if As(err, &e) {
		return e.IsRecoverable()
	}
	return false
}

// GetHint extracts the recovery hint from an error, if available.
func GetHint(err error) string {
	var e *Error
	if As(err, &e) {
		return e.Hint
	}
	return ""
}

// GetCategory extracts the error category, defaulting to CategoryUnknown.
func GetCategory(err error) ErrorCategory {
	var e *Error
	if As(err, &e) {
		return e.Category
	}
	return CategoryUnknown
}

// GetSeverity extracts the error severity, defaulting to SeverityMedium.
func GetSeverity(err error) Severity {
	var e *Error
	if As(err, &e) {
		return e.Severity
	}
	return SeverityMedium
}
