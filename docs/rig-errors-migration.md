# Rig Package Error Handling Migration

## Overview

This document describes the migration of the rig package from basic error handling to the comprehensive errors package (`internal/errors`).

## Migration Date

2026-02-03

## Objectives

1. Replace all `errors.New()` and `fmt.Errorf()` with the enhanced errors package
2. Add error categories (Transient/Permanent/User/System)
3. Add recovery hints for common failures
4. Implement retry logic for transient errors (network operations, file I/O)
5. Add error severity levels
6. Improve error context and debugging information
7. Maintain backward compatibility (no API breaking changes)

## Changes Made

### 1. manager.go

#### Sentinel Errors
**Before:**
```go
var (
    ErrRigNotFound = errors.New("rig not found")
    ErrRigExists   = errors.New("rig already exists")
)
```

**After:**
```go
var (
    ErrRigNotFound = errors.Permanent("rig.NotFound", nil).
        WithHint("Use 'gt rig list' to see available rigs")
    ErrRigExists = errors.User("rig.Exists", "rig already exists").
        WithHint("Use a different rig name or check existing rigs with 'gt rig list'")
)
```

#### wrapCloneError Function
- Enhanced GitHub auth failure detection with detailed recovery hints
- Added SSH URL conversion suggestions for HTTPS URLs
- Categorized network errors as transient (retry-able)
- Added error context (git_url, ssh_url)
- Enhanced error messages with specific recovery steps

#### loadRig Method
- Enhanced directory validation with context
- Distinguished between not-exist (permanent) and access errors (system)
- Added helpful error messages with paths and recovery hints
- Categorized "not a directory" as user error

#### AddRig Method
- Enhanced rig name validation with context and suggested alternatives
- Added retry logic with `errors.WithNetworkRetry()` for git clone operations
- Enhanced bare repository cloning with network retry
- Enhanced mayor clone creation with network retry
- Added error context for all directory creation failures
- Improved branch checkout error messages
- Added context for beads prefix mismatch errors
- Enhanced file system operation errors with hints

#### saveRigConfig Method
- Added error categorization for marshal failures (system)
- Added file I/O retry for transient filesystem issues
- Enhanced write failures with context (config_path)

#### LoadRigConfig Function
- Added file I/O retry for transient read issues
- Distinguished between not-exist (permanent) and read errors (system)
- Enhanced JSON unmarshal errors with hints about corruption
- Added error context (config_path)

#### initBeads Method
- Enhanced prefix validation with detailed hints
- Added error categorization for redirect file creation (system)
- Enhanced beads directory creation errors with context
- Added error context for config write failures

#### initAgentBeads Method
- Enhanced agent bead creation errors with context
- Added hints to check beads installation and database status

#### ensureGitignoreEntry Method
- Enhanced file read/write errors with context
- Added error categorization for permission failures

#### createPatrolHooks Method
- Enhanced settings directory creation with error context

#### createPluginDirectories Method
- Enhanced town and rig plugin directory creation with context
- Added error categorization for permission failures

### 2. convenience.go

#### Load Function
- Enhanced rigs config loading with error context
- Added hints to verify town configuration
- Enhanced "not found in config" error with recovery hints

#### FindFromCwd Function
- Enhanced current directory resolution with error context

#### FindRigFromPath Function
- Enhanced absolute path resolution with error context
- Improved "not in a town" error with detailed hints
- Added recovery suggestion to initialize a new town
- Enhanced rigs config loading with context
- Improved "no rig found" error with path context

### 3. overlay.go

#### CopyOverlay Function
- Enhanced overlay directory read errors with context
- Added error categorization for system failures

#### EnsureGitignorePatterns Function
- Added file I/O retry for gitignore reads
- Enhanced gitignore file operations with error context
- Added error categorization for write failures

#### copyFilePreserveMode Function
- Enhanced source file stat errors with context
- Enhanced file open/create errors with detailed hints
- Added file I/O retry for copy operations
- Enhanced copy failure errors with disk space hints

### 4. Test Updates

#### manager_test.go
- Updated TestGetRigNotFound to use `errors.Is()` for wrapped error checking
- Updated TestRemoveRigNotFound to use `errors.Is()` for wrapped error checking
- Tests now check for error presence and wrapping rather than exact sentinel match
- Added errors import for error checking utilities

## Error Categories Used

### Transient
- Network timeouts
- Connection failures (DNS resolution, connection refused)
- File I/O transient failures
- Temporary filesystem issues

Category: `errors.Transient()`
Strategy: Automatic retry with exponential backoff (3 attempts by default, 5 for network)

### Permanent
- Rig not found
- Directory not found
- Config file not found
- Config file corrupted (JSON unmarshal errors)
- Branch not found
- Worktree creation failures

Category: `errors.Permanent()`
Strategy: Fail immediately, no retry

### User
- Rig already exists
- Directory already exists
- Invalid rig name
- Not a directory
- Beads prefix mismatch
- Invalid beads prefix
- Not in a town (outside Gas Town directory)
- Authentication failures (GitHub password auth)

Category: `errors.User()`
Strategy: Fail with clear recovery hints

### System
- File system permission errors
- Directory creation failures
- File read/write failures
- Config marshal/unmarshal failures
- Agent bead creation failures
- Working directory resolution failures

Category: `errors.System()`
Strategy: System-level recovery hints

## Retry Configurations

### Network Retry (git clone, push, fetch)
- Max Attempts: 5
- Initial Delay: 500ms
- Max Delay: 30s
- Multiplier: 2.0x (exponential backoff)

### File I/O Retry (config files, gitignore, overlays)
- Max Attempts: 3
- Initial Delay: 50ms
- Max Delay: 2s
- Multiplier: 2.0x (exponential backoff)

## Recovery Hints Added

### Rig Operations
- "Use 'gt rig list' to see available rigs"
- "Navigate to a rig directory or use 'gt rig list' to see available rigs"
- "Try removing and re-adding the rig: gt rig remove <name> && gt rig add <name> <git-url>"
- "Remove the existing directory or choose a different rig name"

### Git Operations
- "GitHub no longer supports password authentication. Try using SSH instead: gt rig add <name> <ssh-url>"
- "Check your network connection and try again"
- "Verify the repository URL is correct and you have access"
- "The branch may not exist in the repository. Verify with: git branch -a"

### File System Operations
- "Check file system permissions"
- "Check file system permissions and available disk space"
- "Check disk space and file system integrity"

### Beads Operations
- "The source repository uses beads prefix '<prefix>'. Use --prefix <prefix> to match existing issues"
- "Beads prefix must be alphanumeric with optional hyphens, start with letter, and be at most 20 characters"
- "Check that beads is installed and the database is accessible with: bd status"

### Town Operations
- "Navigate to a Gas Town directory (containing config/) or initialize a new town with 'gt install'"
- "Verify the town configuration exists and is valid"

## Backward Compatibility

- All exported functions maintain their signatures
- Sentinel errors (ErrRigNotFound, ErrRigExists) are still exported
- Error checking with `errors.Is()` still works
- Old code continues to work without changes
- Tests updated to use `errors.Is()` for wrapped error checking

## Benefits

1. **Automatic Retry**: Transient failures (network, file I/O) retry automatically
2. **Better Debugging**: Errors include context (rig_name, git_url, paths, etc.)
3. **Actionable Messages**: Recovery hints guide users to fix issues
4. **Reduced Failures**: Network and file operations are more resilient
5. **Improved UX**: Clear error messages reduce confusion
6. **Maintainability**: Consistent error handling across the package

## Testing

All tests pass with the new error handling:
- Unit tests verify error behavior with `errors.Is()` checks
- Tests check for error presence and proper wrapping
- No breaking changes to existing test contracts

## Files Modified

- `/Users/ericfriday/gt/internal/rig/manager.go` - Main rig management logic
- `/Users/ericfriday/gt/internal/rig/convenience.go` - Convenience functions
- `/Users/ericfriday/gt/internal/rig/overlay.go` - Overlay and gitignore operations
- `/Users/ericfriday/gt/internal/rig/manager_test.go` - Test updates for wrapped errors

## Files Not Modified

- `/Users/ericfriday/gt/internal/rig/config.go` - No error handling changes needed
- `/Users/ericfriday/gt/internal/rig/types.go` - Type definitions only

## Future Enhancements

1. Add structured logging for retry attempts
2. Add metrics for retry success/failure rates
3. Consider context-aware retry (with timeout propagation)
4. Add more specific error hints based on common failure patterns
5. Add telemetry for error categories to identify common issues

## References

- Errors Package README: `/Users/ericfriday/gt/internal/errors/README.md`
- Errors Package Implementation: `/Users/ericfriday/gt/internal/errors/`
- Swarm Migration: `/Users/ericfriday/gt/docs/swarm-errors-migration.md`

## Co-Authors

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
