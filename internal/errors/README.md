# Errors Package

Comprehensive error handling with retry logic, recovery hints, and structured error types for Gas Town.

## Overview

This package provides:
- **Enhanced error types** with categorization, severity, and context
- **Automatic retry logic** with exponential backoff
- **Recovery hints** for actionable error messages
- **Domain-specific errors** for sessions, git, beads, etc.
- **Error enrichment** that automatically adds helpful hints

## Quick Start

### Basic Error Creation

```go
import "github.com/steveyegge/gastown/internal/errors"

// Simple error with operation context
err := errors.New("session.Start", baseErr)

// User-facing error with message
err := errors.User("polecat.Create", "polecat name already exists")

// Transient error that should be retried
err := errors.Transient("git.Push", networkErr)

// Critical error that prevents operation
err := errors.Critical("database.Open", dbErr)
```

### Error with Recovery Hints

```go
err := errors.User("polecat.Create", "polecat not found").
    WithHint("Use 'gt polecat list' to see available polecats")

// Display full message with hint
fmt.Println(err.FullMessage())
// Output:
// polecat.Create: polecat not found
//
// How to fix: Use 'gt polecat list' to see available polecats
```

### Automatic Retry

```go
// Retry with default config (3 attempts, 100ms initial delay)
err := errors.WithRetry(func() error {
    return gitPush()
})

// Network-optimized retry (5 attempts, 500ms initial, 30s max)
err := errors.WithNetworkRetry(func() error {
    return fetchFromRemote()
})

// Custom retry configuration
config := errors.RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0, // Exponential backoff
    OnRetry: func(attempt int, err error) {
        log.Printf("Retry %d: %v", attempt, err)
    },
}
err := errors.Retry(fn, config)
```

### Retry with Values

```go
// Retry function that returns a value
result, err := errors.RetryFunc(func() (string, error) {
    return fetchData()
}, errors.NetworkRetryConfig())
```

### Context-Aware Retry

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := errors.RetryWithContext(ctx, func() error {
    return performOperation()
}, errors.DefaultRetryConfig())
```

## Error Categories

Errors are categorized for appropriate handling:

### CategoryTransient
Temporary errors that may succeed on retry (network timeouts, lock contention, etc.)

```go
err := errors.Transient("network.Connect", networkErr)
```

### CategoryPermanent
Errors that will not succeed on retry (invalid configuration, missing resources)

```go
err := errors.Permanent("config.Load", configErr)
```

### CategoryUser
Errors caused by user input or actions (invalid arguments, resource conflicts)

```go
err := errors.User("polecat.Create", "invalid polecat name")
```

### CategorySystem
Internal system errors (resource exhaustion, system failures)

```go
err := errors.System("memory.Allocate", memErr)
```

## Error Severity

Severity indicates the impact level:

- **SeverityLow**: Minor errors that don't significantly impact functionality
- **SeverityMedium**: Moderate errors that may limit functionality (default)
- **SeverityHigh**: Serious errors that significantly impact functionality
- **SeverityCritical**: Fatal errors that prevent operation

```go
err := errors.Critical("database.Open", dbErr)
if err.IsFatal() {
    os.Exit(1)
}
```

## Domain-Specific Errors

### SessionError

```go
err := errors.NewSessionError("start", "Toast", tmuxErr).
    WithHint(errors.HintSessionStart)
```

### GitError

```go
err := errors.NewGitError("push", "/path/to/repo", "main", pushErr).
    WithHint(errors.HintGitPushFailed)
```

### BeadsError

```go
err := errors.NewBeadsError("show", "gt-123", notFoundErr).
    WithHint(errors.WithIssueNotFoundHint("gt-123"))
```

### RefineryError

```go
err := errors.NewRefineryError("process", "myrig", "mr-456", mergeErr)
```

### NetworkError

```go
err := errors.NewNetworkError("connect", "github.com", timeoutErr).
    WithHint(errors.HintNetworkError)
```

### FileError

```go
err := errors.NewFileError("read", "/path/to/file", osErr).
    WithHint(errors.HintFileNotFound)
```

### ConfigError

```go
err := errors.NewConfigError("parse", "config.json", "timeout", parseErr).
    WithHint(errors.HintConfigInvalid)
```

## Recovery Hints

### Built-in Hints

The package provides many pre-defined hints:

```go
errors.HintPolecatNotFound
errors.HintRigNotFound
errors.HintSessionNotFound
errors.HintGitPushFailed
errors.HintBeadsNotFound
errors.HintNetworkError
errors.HintFileNotFound
errors.HintConfigInvalid
// ... and many more
```

### Contextual Hints

Helper functions create context-specific hints:

```go
hint := errors.WithPolecatNotFoundHint("Toast")
// "Polecat 'Toast' not found. Use 'gt polecat list' to see available polecats."

hint := errors.WithCommandNotFoundHint("bd")
// "Install beads with: brew install beads"

hint := errors.SuggestRetry("git push", "network timeout")
// "The git push operation failed (network timeout) but can be retried. ..."
```

### HintBuilder

For complex hints:

```go
hint := errors.NewHintBuilder().
    Add("The operation failed.").
    AddIf(hasConflicts, "Resolve merge conflicts first.").
    AddFormatted("Run 'gt status %s' to check.", rigName).
    Build()
```

## Error Enrichment

Automatically add hints to errors based on patterns:

```go
err := errors.New("command not found: bd")
enriched := errors.EnrichErrorWithHint(err)
// Automatically adds hint about installing beads
```

Enrichment recognizes:
- Command not found errors (bd, tmux, git)
- Git operation failures (push, pull, merge conflicts)
- Network errors (connection refused, timeout)
- File errors (not found, permission denied)
- Beads errors

## Error Checking

### Check Error Type

```go
if errors.IsTransient(err) {
    // Retry the operation
}

if errors.IsRecoverable(err) {
    // Show recovery hint to user
}
```

### Extract Information

```go
category := errors.GetCategory(err)
severity := errors.GetSeverity(err)
hint := errors.GetHint(err)

if hint != "" {
    fmt.Println("Suggestion:", hint)
}
```

### Standard Error Checking

```go
// errors.Is and errors.As work as expected
if errors.Is(err, os.ErrNotExist) {
    // Handle file not found
}

var sessionErr *errors.SessionError
if errors.As(err, &sessionErr) {
    fmt.Printf("Session error for polecat: %s\n", sessionErr.Polecat)
}
```

## Retry Configurations

Pre-configured retry strategies for common scenarios:

### DefaultRetryConfig
- 3 attempts
- 100ms initial delay
- 10s max delay
- 2.0x multiplier

```go
errors.WithRetry(fn)
```

### NetworkRetryConfig
- 5 attempts
- 500ms initial delay
- 30s max delay
- 2.0x multiplier

```go
errors.WithNetworkRetry(fn)
```

### FileIORetryConfig
- 3 attempts
- 50ms initial delay
- 2s max delay
- 2.0x multiplier

```go
errors.WithFileIORetry(fn)
```

### DatabaseRetryConfig
- 4 attempts
- 100ms initial delay
- 5s max delay
- 2.0x multiplier

```go
errors.WithDatabaseRetry(fn)
```

## Custom Retry Logic

### Custom ShouldRetry Function

```go
config := errors.RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 100 * time.Millisecond,
    Multiplier:   2.0,
    ShouldRetry: func(err error) bool {
        // Custom logic: only retry specific errors
        return errors.Is(err, ErrConnectionRefused) ||
               errors.Is(err, ErrTimeout)
    },
}
```

### Retry Callback

```go
config := errors.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    Multiplier:   2.0,
    OnRetry: func(attempt int, err error) {
        log.Printf("Attempt %d failed: %v", attempt, err)
        // Could also update progress UI, send metrics, etc.
    },
}
```

## Best Practices

### 1. Use Appropriate Categories

```go
// Network operations → Transient
err := errors.Transient("git.Push", pushErr)

// User input validation → User
err := errors.User("validate", "invalid email format")

// Configuration loading → Permanent
err := errors.Permanent("config.Load", loadErr)
```

### 2. Add Recovery Hints

```go
err := errors.User("polecat.NotFound", "polecat not found").
    WithHint(errors.HintPolecatList)
```

### 3. Use Domain Errors

```go
// Instead of generic errors
err := errors.NewGitError("push", repo, branch, pushErr)

// Not just
err := fmt.Errorf("git push failed: %w", pushErr)
```

### 4. Enrich External Errors

```go
// When calling external libraries
result, err := externalLib.DoSomething()
if err != nil {
    return errors.EnrichErrorWithHint(err)
}
```

### 5. Retry Transient Operations

```go
err := errors.WithNetworkRetry(func() error {
    return gitPushToRemote()
})
```

### 6. Provide Context

```go
err := errors.New("session.Start", baseErr).
    WithContext("polecat", polecatName).
    WithContext("rig", rigName).
    WithContext("attempt", attemptCount)
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/errors -v
go test ./internal/errors -cover
```

Test coverage includes:
- Error creation and categorization
- Retry logic and backoff
- Context cancellation
- Error wrapping and unwrapping
- Hint generation
- Domain-specific errors

## Examples

### Example 1: Session Start with Retry

```go
func startSession(name string) error {
    return errors.WithRetry(func() error {
        t := tmux.NewTmux()
        err := t.NewSession(name)
        if err != nil {
            if isTransientTmuxError(err) {
                return errors.Transient("session.Start", err)
            }
            return errors.Permanent("session.Start", err).
                WithHint(errors.HintSessionStart)
        }
        return nil
    })
}
```

### Example 2: Git Push with Recovery

```go
func gitPush(repo, branch string) error {
    err := errors.WithNetworkRetry(func() error {
        cmd := exec.Command("git", "push", "origin", branch)
        cmd.Dir = repo
        return cmd.Run()
    })

    if err != nil {
        return errors.NewGitError("push", repo, branch, err).
            WithHint(errors.HintGitPushFailed)
    }
    return nil
}
```

### Example 3: Polecat Operation with Context

```go
func createPolecat(rig *rig.Rig, name string) error {
    exists, err := polecatExists(rig, name)
    if err != nil {
        return errors.System("polecat.Check", err)
    }
    if exists {
        return errors.User("polecat.Create", "polecat already exists").
            WithHint(errors.WithPolecatNotFoundHint(name))
    }

    err = errors.WithRetry(func() error {
        return createPolecatWorktree(rig, name)
    })

    if err != nil {
        return errors.New("polecat.Create", err).
            WithContext("rig", rig.Name).
            WithContext("polecat", name).
            WithHint("Check git worktree permissions and try again")
    }
    return nil
}
```

### Example 4: Beads Operation with Enrichment

```go
func getIssue(id string) (*Issue, error) {
    cmd := exec.Command("bd", "show", id, "--json")
    output, err := cmd.CombinedOutput()
    if err != nil {
        // Automatically adds hint about beads installation
        enriched := errors.EnrichErrorWithHint(err)
        return nil, errors.NewBeadsError("show", id, enriched).
            WithHint(errors.WithIssueNotFoundHint(id))
    }

    var issue Issue
    if err := json.Unmarshal(output, &issue); err != nil {
        return nil, errors.Permanent("beads.Parse", err)
    }
    return &issue, nil
}
```

## Migration Guide

### From Standard Errors

```go
// Before
return fmt.Errorf("failed to start session: %w", err)

// After
return errors.New("session.Start", err).
    WithHint(errors.HintSessionStart)
```

### From Domain Errors

```go
// Before
type SessionError struct {
    Op  string
    Err error
}

// After
return errors.NewSessionError("start", polecatName, err)
```

### Adding Retry Logic

```go
// Before
for i := 0; i < 3; i++ {
    err := gitPush()
    if err == nil {
        return nil
    }
    time.Sleep(time.Second * time.Duration(i+1))
}
return err

// After
return errors.WithNetworkRetry(gitPush)
```

## Package Structure

```
internal/errors/
├── README.md          # This file
├── errors.go          # Core error types and constructors
├── errors_test.go     # Core error tests
├── retry.go           # Retry logic and configurations
├── retry_test.go      # Retry tests
├── hints.go           # Recovery hints and suggestions
├── hints_test.go      # Hint tests
├── domain.go          # Domain-specific error types
└── domain_test.go     # Domain error tests
```

## Contributing

When adding new error scenarios:

1. Add appropriate error category and severity
2. Include a recovery hint
3. Add tests for the new error type
4. Update this README with examples
5. Consider if a domain-specific error type is needed

## See Also

- Standard library `errors` package documentation
- Gas Town error handling conventions
- Beads error codes and messages
