# Polecat Package Error Handling Migration

## Overview

This document describes the migration of the polecat package from basic error handling to the comprehensive errors package (`internal/errors`).

## Migration Date

2026-02-03

## Objectives

1. Replace all `errors.New()` and `fmt.Errorf()` with the enhanced errors package
2. Add error categories (Transient/Permanent/User/System)
3. Add recovery hints for common failures
4. Implement retry logic for transient errors (beads operations, file I/O, tmux)
5. Add error severity levels
6. Improve error context and debugging information
7. Maintain backward compatibility (no API breaking changes)

## Changes Made

### 1. manager.go

#### Sentinel Errors
**Before:**
```go
var (
    ErrPolecatExists      = errors.New("polecat already exists")
    ErrPolecatNotFound    = errors.New("polecat not found")
    ErrHasChanges         = errors.New("polecat has uncommitted changes")
    ErrHasUncommittedWork = errors.New("polecat has uncommitted work")
    ErrShellInWorktree    = errors.New("shell working directory is inside polecat worktree")
)
```

**After:**
```go
var (
    ErrPolecatExists = errors.User("polecat.Exists", "polecat already exists").
        WithHint("Use a different name or remove the existing polecat with: gt polecat nuke <rig>/<name>")
    ErrPolecatNotFound = errors.Permanent("polecat.NotFound", nil).
        WithHint("Use 'gt polecat list' to see available polecats")
    ErrHasChanges = errors.User("polecat.HasChanges", "polecat has uncommitted changes").
        WithHint("Commit your changes with: gt polecat commit <rig>/<name>")
    ErrHasUncommittedWork = errors.User("polecat.HasUncommittedWork", "polecat has uncommitted work").
        WithHint("Review changes with 'git status' and commit or stash them before removal")
    ErrShellInWorktree = errors.User("polecat.ShellInWorktree", "shell working directory is inside polecat worktree").
        WithHint("Change directory outside the worktree before removing it")
)
```

#### AddWithOptions Method
- Enhanced polecat exists check with context (name, rig)
- Added error categorization for directory creation (system)
- Enhanced worktree creation with transient categorization and context (branch, start_point, path)
- Added recovery hints for file system and git operations
- Improved error messages with actionable hints

#### RemoveWithOptions Method
- Enhanced not-found check with context
- Added error categorization for clone path removal (system)
- Added recovery hints for file system permissions
- Improved beads error handling with proper `errors.Is()` checks

#### AllocateName Method
- Enhanced pool state save with transient categorization
- Added error context (rig) and hints for file system permissions

#### RepairWorktreeWithOptions Method
- Enhanced not-found check with context
- Added error categorization for old worktree removal (system)
- Enhanced directory creation with context and hints
- Added transient categorization for worktree creation

#### Get/SetState/AssignIssue/ClearIssue Methods
- Enhanced not-found checks with context
- Added transient categorization for beads operations
- Added error context (issue_id, polecat, status)
- Added recovery hints for beads availability

#### CleanupStaleBranches Method
- Enhanced repo base finding with system categorization
- Added transient categorization for branch listing
- Improved error messages with context

#### List Method
- Enhanced directory read errors with system categorization
- Added error context and recovery hints

### 2. namepool.go

#### SetTheme Method
- Enhanced unknown theme error with user categorization
- Added error context (theme) and available themes hint

#### GetThemeNames Function
- Enhanced unknown theme error with user categorization
- Added error context and recovery hints

#### RemoveCustomName Method
- Enhanced not-custom-name error with user categorization
- Added error context (name) and recovery hints

### 3. session_manager.go

#### Session Errors
**Before:**
```go
var (
    ErrSessionRunning  = errors.New("session already running")
    ErrSessionNotFound = errors.New("session not found")
    ErrIssueInvalid    = errors.New("issue not found or tombstoned")
)
```

**After:**
```go
var (
    ErrSessionRunning = errors.User("polecat.SessionRunning", "session already running").
        WithHint("Attach to the existing session with: gt polecat attach <rig>/<name>")
    ErrSessionNotFound = errors.Permanent("polecat.SessionNotFound", nil).
        WithHint("Check active sessions with: gt polecat list")
    ErrIssueInvalid = errors.User("polecat.IssueInvalid", "issue not found or tombstoned").
        WithHint("Verify the issue exists with: bd show <issue-id>")
)
```

#### Start Method
- Enhanced not-found check with context (polecat, rig)
- Added transient categorization for session checks
- Enhanced hook blocking with user categorization and context
- Added transient categorization for runtime settings
- Enhanced session creation with transient categorization and context
- Improved startup failure detection with permanent categorization
- Added recovery hints throughout

#### Stop Method
- Enhanced session checks with transient categorization
- Enhanced hook blocking with user categorization
- Added transient categorization for session kill

#### Status/Attach/Capture/CaptureSession/Inject Methods
- Enhanced session checks with transient categorization
- Added error context (session_id) throughout
- Improved not-found errors with context

#### validateIssue Method
- Enhanced issue validation with error context
- Added system categorization for parse errors
- Improved tombstone detection with context

#### hookIssue Method
- Added transient categorization for bd update
- Enhanced error with context (issue_id, agent_id)

### 4. pending.go

#### CheckInboxForSpawns Function
- Enhanced mailbox retrieval with system categorization
- Added transient categorization for message listing
- Added error context (mailbox) and recovery hints

#### TriggerPendingSpawns Function
- Enhanced session checks with transient categorization
- Added permanent categorization for dead sessions
- Enhanced session nudging with transient categorization
- Added error context (session_id) throughout
- Improved error messages with recovery hints

## Error Categories Used

### Transient
- Beads operations (update, queries)
- Tmux operations (session checks, kills, nudges)
- File I/O (pool state save)
- Worktree creation
- Directory reads

Category: `errors.Transient()`
Strategy: Automatic retry with exponential backoff (3 attempts by default)

### Permanent
- Polecat not found
- Session not found
- Branch not found
- Session startup failures
- Dead sessions

Category: `errors.Permanent()`
Strategy: Fail immediately, no retry

### User
- Polecat already exists
- Name conflicts
- Theme selection errors
- Session already running
- Issue invalid/tombstoned
- Uncommitted work blocking removal
- Shell inside worktree
- Hook blocking operations
- Custom name not found

Category: `errors.User()`
Strategy: Fail with clear recovery hints

### System
- Directory creation failures
- File system permission errors
- Repo base not found
- Mailbox access errors
- Runtime settings failures
- Clone path removal failures

Category: `errors.System()`
Strategy: System-level recovery hints

## Recovery Hints Added

### Polecat Operations
- "Use 'gt polecat list' to see available polecats"
- "Use a different name or remove the existing polecat with: gt polecat nuke <rig>/<name>"
- "Commit your changes with: gt polecat commit <rig>/<name>"
- "Review changes with 'git status' and commit or stash them before removal"
- "Change directory outside the worktree before removing it"

### Session Operations
- "Attach to the existing session with: gt polecat attach <rig>/<name>"
- "Check active sessions with: gt polecat list"
- "Check tmux status and try again"
- "Check the agent command configuration or runtime settings"
- "Check hook scripts in .runtime/hooks/ and resolve the issue"
- "Check hook scripts in .runtime/hooks/ or use --force to bypass"

### Git/Worktree Operations
- "Check that the start point exists with: git branch -r"
- "Check git repository status"
- "Check that .repo.git or mayor/rig exists in the rig directory"

### File System Operations
- "Check file system permissions and available disk space"
- "Check file system permissions"
- "Check file system permissions for .runtime directory"
- "Check file system permissions for polecats/ directory"

### Beads Operations
- "Check that beads is available with: bd status"
- "Verify the issue exists with: bd show <issue-id>"
- "Check beads installation with: bd status"

### Name Pool Operations
- "Use one of the available themes: mad-max, minerals, wasteland"
- "Check available custom names or use a themed name"

### Mail Operations
- "Check that the deacon mailbox exists in the town"
- "Check mail directory permissions"
- "Check tmux status and session availability"

## Backward Compatibility

- All exported functions maintain their signatures
- Sentinel errors (ErrPolecatExists, ErrPolecatNotFound, etc.) are still exported
- Error checking with `errors.Is()` still works
- Old code continues to work without changes
- Tests pass without modifications

## Benefits

1. **Automatic Retry**: Transient failures (beads, tmux, file I/O) retry automatically
2. **Better Debugging**: Errors include context (name, rig, session_id, issue_id, paths, etc.)
3. **Actionable Messages**: Recovery hints guide users to fix issues
4. **Reduced Failures**: Beads and tmux operations are more resilient
5. **Improved UX**: Clear error messages reduce confusion about polecat state
6. **Maintainability**: Consistent error handling across the package

## Testing

All tests pass with the new error handling:
- Unit tests verify error behavior
- Error checking works with `errors.Is()` for sentinel errors
- No breaking changes to existing test contracts

## Files Modified

- `/Users/ericfriday/gt/internal/polecat/manager.go` - Main polecat management logic
- `/Users/ericfriday/gt/internal/polecat/namepool.go` - Name pool management
- `/Users/ericfriday/gt/internal/polecat/session_manager.go` - Session lifecycle management
- `/Users/ericfriday/gt/internal/polecat/pending.go` - Pending spawn operations

## Files Not Modified

- `/Users/ericfriday/gt/internal/polecat/types.go` - Type definitions only (no error handling)

## Context Added

Common context fields added across errors:
- `name`: Polecat name
- `rig`: Rig name
- `session_id`: Tmux session identifier
- `issue_id`: Beads issue identifier
- `polecat`: Polecat name (in session operations)
- `theme`: Name pool theme
- `branch`: Git branch name
- `start_point`: Git start point for worktree
- `path`: File system paths
- `dir`: Directory paths
- `mailbox`: Mailbox identifier
- `agent_id`: Beads agent identifier
- `hook_error`: Hook error details
- `work_dir`: Working directory

## Lines Changed

Approximately 450-500 lines modified across 4 files:
- manager.go: ~250 lines
- session_manager.go: ~150 lines
- namepool.go: ~30 lines
- pending.go: ~30 lines

## Future Enhancements

1. Add structured logging for retry attempts
2. Add metrics for retry success/failure rates
3. Consider context-aware retry (with timeout propagation)
4. Add more specific error hints based on common failure patterns
5. Add telemetry for error categories to identify common issues
6. Consider adding error codes for programmatic handling in CLI

## References

- Errors Package README: `/Users/ericfriday/gt/internal/errors/README.md`
- Errors Package Implementation: `/Users/ericfriday/gt/internal/errors/`
- Swarm Migration: `/Users/ericfriday/gt/docs/swarm-errors-migration.md`
- Rig Migration: `/Users/ericfriday/gt/docs/rig-errors-migration.md`

## Co-Authors

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
