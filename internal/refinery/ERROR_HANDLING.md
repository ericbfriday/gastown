# Refinery Error Handling Migration

## Overview

The refinery package has been migrated from basic error handling (fmt.Errorf, errors.New) to the comprehensive errors package (`internal/errors`), providing better reliability, retry logic, and user experience through recovery hints.

## Changes Summary

### Files Modified

1. **types.go** - State transition errors with recovery hints
2. **manager.go** - Lifecycle and queue operation errors with retry logic
3. **engineer.go** - Git and beads operation errors with network retry

### Key Improvements

#### 1. Error Categories
All errors are now categorized for appropriate handling:
- **User Errors**: Invalid states, missing MRs, configuration issues
- **Transient Errors**: Network failures, temporary git issues
- **System Errors**: Internal failures requiring investigation

#### 2. Recovery Hints
Every error includes actionable guidance for users:
```go
// Before
return fmt.Errorf("refinery not running")

// After
return errors.User("refinery.status", "refinery not running").
    WithHint("Start the refinery with 'gt refinery start'")
```

#### 3. Automatic Retry Logic
Network and transient operations now retry automatically:

**Manager Operations:**
- Session creation (tmux) - 3 retries with backoff
- Session termination - 3 retries with backoff
- Beads queue queries - 3 retries with backoff

**Engineer Operations:**
- Git pull operations - 5 retries (network retry config)
- Git push operations - 5 retries (network retry config)
- Beads queries (list operations) - 3 retries with backoff

#### 4. Domain-Specific Context
Errors now include relevant context:
```go
errors.NewGitError("push", workDir, branch, err)
errors.NewBeadsError("list", issueID, err)
errors.NewRefineryError("start", rigName, mrID, err)
```

## Error Handling Patterns

### State Transition Errors (types.go)

```go
// Invalid transition with hint
if from == MRClosed {
    return errors.User("refinery.transition", "cannot change status from closed").
        WithHint("Closed merge requests are immutable. Create a new MR if needed.")
}

// Specific operation errors with context
func (mr *MergeRequest) Claim() error {
    if mr.Status != MROpen {
        return errors.User("refinery.claim",
            fmt.Sprintf("can only claim from open, current status is %s", mr.Status)).
            WithHint("MRs can only be claimed when they are in the open state.")
    }
    mr.Status = MRInProgress
    return nil
}
```

### Manager Errors (manager.go)

```go
// Session status with recovery hints
func (m *Manager) Status() (*tmux.SessionInfo, error) {
    running, err := t.HasSession(sessionID)
    if err != nil {
        return nil, errors.NewRefineryError("status", m.rig.Name, "", err).
            WithHint("Ensure tmux is installed and running: 'brew install tmux'")
    }
    // ... rest of method
}

// Queue operations with retry
func (m *Manager) Queue() ([]QueueItem, error) {
    var issues []*beads.Issue
    err := errors.WithRetry(func() error {
        var queryErr error
        issues, queryErr = b.List(beads.ListOptions{...})
        return queryErr
    })
    if err != nil {
        return nil, errors.NewBeadsError("list", "", err).
            WithHint("Failed to query merge queue. Ensure beads is initialized: 'bd init'")
    }
    // ... rest of method
}
```

### Engineer Git Operations (engineer.go)

```go
// Git operations with transient retry and recovery hints
func (e *Engineer) doMerge(ctx context.Context, branch, target, sourceIssue string) ProcessResult {
    // Network operations use automatic retry
    err := errors.WithNetworkRetry(func() error {
        return e.git.Pull("origin", target)
    })

    // User-facing errors with actionable hints
    if !exists {
        return ProcessResult{
            Success: false,
            Error:   errors.User("refinery.merge",
                fmt.Sprintf("branch %s not found locally", branch)).
                WithHint("The polecat may need to push the branch: 'git push origin " + branch + "'").
                Error(),
        }
    }

    // Transient push failures with retry
    err = errors.WithNetworkRetry(func() error {
        return e.git.Push("origin", target, false)
    })
    if err != nil {
        return ProcessResult{
            Success: false,
            Error:   errors.Transient("refinery.merge",
                errors.NewGitError("push", e.workDir, target, err)).
                WithHint("Push failed. Check network connection and remote permissions.").
                Error(),
        }
    }
}
```

### Beads Operations with Retry

```go
// List operations with automatic retry for transient failures
func (e *Engineer) ListReadyMRs() ([]*MRInfo, error) {
    var issues []*beads.Issue
    err := errors.WithRetry(func() error {
        var queryErr error
        issues, queryErr = e.beads.ReadyWithType("merge-request")
        return queryErr
    })
    if err != nil {
        return nil, errors.New("refinery.list",
            errors.NewBeadsError("list", "", err)).
            WithHint("Failed to query ready MRs. Ensure beads is initialized: 'bd init'")
    }
    // ... conversion logic
}
```

## Error Types Reference

### Common Refinery Errors

| Error | Category | Hint |
|-------|----------|------|
| `ErrNotRunning` | User | "Start the refinery with 'gt refinery start'" |
| `ErrAlreadyRunning` | User | "Use 'gt refinery status' to check the current refinery state" |
| `ErrNoQueue` | User | "Create merge requests with 'gt mr create' or check if MRs are blocked" |
| `ErrMRNotFound` | User | "Use 'gt refinery queue' to list all merge requests" |
| `ErrInvalidTransition` | User | "MR state transitions must follow the allowed flow: open → in_progress → closed" |
| `ErrClosedImmutable` | User | "Closed MRs cannot be modified. Create a new MR if needed." |

### Retry Configurations

**Default Retry** (WithRetry):
- Max attempts: 3
- Initial delay: 100ms
- Max delay: 10s
- Multiplier: 2.0x (exponential backoff)

**Network Retry** (WithNetworkRetry):
- Max attempts: 5
- Initial delay: 500ms
- Max delay: 30s
- Multiplier: 2.0x (exponential backoff)

## Testing

All existing tests pass without modification. The error changes are backward compatible:
- Error checking with `errors.Is()` works as expected
- Error types can be extracted with `errors.As()`
- String representations remain clear and informative

```bash
$ go test ./internal/refinery/... -v
PASS
ok      github.com/steveyegge/gastown/internal/refinery    0.067s
```

## Benefits

### 1. Reduced Transient Failures
- Network operations retry automatically (git push/pull)
- Temporary beads database locks are handled gracefully
- Tmux session operations retry on transient failures

### 2. Better User Experience
- Every error provides clear guidance on how to resolve it
- Users can distinguish between their errors and system errors
- Recovery hints reduce support burden

### 3. Improved Reliability
- Transient network issues don't cause merge failures
- Operations complete successfully despite temporary glitches
- Exponential backoff prevents thundering herd problems

### 4. Better Debugging
- Errors include operation context (rig name, MR ID, branch, etc.)
- Error categories help identify root causes
- Structured error information aids monitoring

## Migration Notes

### No Breaking Changes
- All public APIs remain unchanged
- Error messages are more informative but compatible
- Existing error checking code continues to work

### Future Improvements
Consider for future work:
1. Error metrics collection for monitoring
2. Circuit breakers for repeated failures
3. Contextual timeouts for long-running operations
4. Structured logging integration

## Examples

### Error with Full Context
```go
err := errors.NewRefineryError("start", "myrig", "mr-123", baseErr).
    WithHint("Refinery agent failed to start. View logs: 'tmux attach -t gt-myrig-refinery'")
```

### Automatic Retry with Logging
```go
err := errors.WithRetry(func() error {
    return performOperation()
})
// Automatically retries 3 times with exponential backoff
```

### User-Friendly Output
```
refinery.start: refinery already running

How to fix: Use 'gt refinery status' to check the current refinery state
```

## See Also

- [Errors Package README](/Users/ericfriday/gt/internal/errors/README.md)
- Refinery architecture documentation
- Gas Town error handling conventions
