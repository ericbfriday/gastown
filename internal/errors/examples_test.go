package errors_test

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/steveyegge/gastown/internal/errors"
)

// Example_basicError demonstrates creating a basic error with context.
func Example_basicError() {
	err := errors.New("session.Start", fmt.Errorf("tmux session not found"))
	fmt.Println(err.Error())
	// Output:
	// session.Start: tmux session not found
}

// Example_errorWithHint demonstrates adding a recovery hint to an error.
func Example_errorWithHint() {
	err := errors.User("polecat.Create", "polecat already exists").
		WithHint("Use 'gt polecat list' to see existing polecats")

	fmt.Println(err.FullMessage())
	// Output:
	// polecat.Create: polecat already exists
	//
	// How to fix: Use 'gt polecat list' to see existing polecats
}

// Example_transientError demonstrates creating a transient error for retry.
func Example_transientError() {
	networkErr := fmt.Errorf("connection timeout")
	err := errors.Transient("network.Connect", networkErr)

	if errors.IsTransient(err) {
		fmt.Println("Error is transient and can be retried")
	}
	// Output:
	// Error is transient and can be retried
}

// Example_retry demonstrates basic retry functionality.
func Example_retry() {
	attempts := 0

	err := errors.WithRetry(func() error {
		attempts++
		if attempts < 3 {
			return errors.Transient("operation", fmt.Errorf("temporary failure"))
		}
		return nil // Success on third attempt
	})

	if err == nil {
		fmt.Printf("Operation succeeded after %d attempts\n", attempts)
	}
	// Output:
	// Operation succeeded after 3 attempts
}

// Example_retryWithContext demonstrates context-aware retry.
func Example_retryWithContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := errors.RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   2.0,
	}

	err := errors.RetryWithContext(ctx, func() error {
		// Simulate work that might timeout
		return nil
	}, config)

	if err == nil {
		fmt.Println("Operation completed within timeout")
	}
	// Output:
	// Operation completed within timeout
}

// Example_retryFunc demonstrates retry with a return value.
func Example_retryFunc() {
	attempts := 0

	result, err := errors.RetryFunc(func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.Transient("fetch", fmt.Errorf("network error"))
		}
		return "success", nil
	}, errors.NetworkRetryConfig())

	if err == nil {
		fmt.Printf("Got result: %s after %d attempts\n", result, attempts)
	}
	// Output:
	// Got result: success after 2 attempts
}

// Example_domainError demonstrates using domain-specific errors.
func Example_domainError() {
	gitErr := errors.NewGitError("push", "/path/to/repo", "main", fmt.Errorf("rejected"))
	// gitErr is already *errors.GitError, no type assertion needed

	fmt.Println(gitErr.Error())
	// Output will vary based on error wrapping
}

// Example_errorEnrichment demonstrates automatic error enrichment.
func Example_errorEnrichment() {
	// Simulate a command not found error
	cmdErr := fmt.Errorf("executable file not found in $PATH: bd")
	enriched := errors.EnrichErrorWithHint(cmdErr)

	hint := errors.GetHint(enriched)
	if hint != "" {
		fmt.Println("Hint provided for missing command")
	}
	// Output:
	// Hint provided for missing command
}

// Example_hintBuilder demonstrates building complex hints.
func Example_hintBuilder() {
	hint := errors.NewHintBuilder().
		Add("The operation failed.").
		Add("Check your network connection.").
		AddFormatted("Run '%s' to check status.", "gt status").
		Build()

	fmt.Println(hint)
	// Output:
	// The operation failed.
	// Check your network connection.
	// Run 'gt status' to check status.
}

// Example_customRetry demonstrates custom retry logic with callback.
func Example_customRetry() {
	retryCount := 0

	config := errors.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Multiplier:   2.0,
		OnRetry: func(attempt int, err error) {
			retryCount++
			fmt.Printf("Retry %d: %v\n", attempt, err)
		},
	}

	err := errors.Retry(func() error {
		if retryCount < 1 {
			return errors.Transient("test", fmt.Errorf("temporary failure"))
		}
		return nil
	}, config)

	if err == nil {
		fmt.Printf("Operation succeeded after %d retries\n", retryCount)
	}
	// Output will vary due to timing
}

// Example_networkRetry demonstrates network-optimized retry.
func Example_networkRetry() {
	err := errors.WithNetworkRetry(func() error {
		// Simulate a network operation
		cmd := exec.Command("echo", "network simulation")
		return cmd.Run()
	})

	if err == nil {
		fmt.Println("Network operation succeeded")
	}
	// Output:
	// Network operation succeeded
}

// Example_errorCategories demonstrates checking error categories.
func Example_errorCategories() {
	userErr := errors.User("validation", "invalid input")
	systemErr := errors.System("database", fmt.Errorf("connection failed"))

	if errors.GetCategory(userErr) == errors.CategoryUser {
		fmt.Println("User error detected")
	}

	if errors.IsRecoverable(userErr) {
		fmt.Println("Error is recoverable")
	}

	if !errors.IsRecoverable(systemErr) {
		fmt.Println("System error is not recoverable")
	}
	// Output:
	// User error detected
	// Error is recoverable
	// System error is not recoverable
}

// Example_errorSeverity demonstrates checking error severity.
func Example_errorSeverity() {
	criticalErr := errors.Critical("database.Open", fmt.Errorf("corruption detected"))

	if criticalErr.IsFatal() {
		fmt.Println("Fatal error - immediate action required")
	}

	severity := errors.GetSeverity(criticalErr)
	fmt.Printf("Severity: %s\n", severity)
	// Output:
	// Fatal error - immediate action required
	// Severity: critical
}

// Example_contextualHints demonstrates creating context-specific hints.
func Example_contextualHints() {
	polecatHint := errors.WithPolecatNotFoundHint("Toast")
	fmt.Println(polecatHint)

	cmdHint := errors.WithCommandNotFoundHint("bd")
	fmt.Println(cmdHint)

	retryHint := errors.SuggestRetry("git push", "network timeout")
	fmt.Println(retryHint)
	// Output will show the generated hints
}

// Example_errorChaining demonstrates error wrapping and unwrapping.
func Example_errorChaining() {
	baseErr := fmt.Errorf("base error")
	wrappedErr := errors.New("operation", baseErr)
	doubleWrapped := errors.New("outer", wrappedErr)

	// Check if base error is in the chain
	if errors.Is(doubleWrapped, baseErr) {
		fmt.Println("Base error found in chain")
	}

	// Extract our Error type from the chain
	var e *errors.Error
	if errors.As(doubleWrapped, &e) {
		fmt.Printf("Found error with operation: %s\n", e.Op)
	}
	// Output will demonstrate error chain traversal
}
