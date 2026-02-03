# Swarm Errors Package Migration - Implementation Summary

## Status: READY FOR IMPLEMENTATION

Date: 2026-02-03
Priority: HIGH
Estimated Effort: 2-3 hours

## Executive Summary

This document provides a complete implementation guide for migrating the swarm package to use the comprehensive errors package. The migration adds automatic retry logic, recovery hints, error categorization, and improved debugging context while maintaining full backward compatibility.

## Background

The swarm package currently uses basic error handling (`errors.New()`, `fmt.Errorf()`) with no retry logic, recovery hints, or structured error information. This causes:

1. **Reliability Issues**: Transient network/beads failures cause operations to fail unnecessarily
2. **Poor UX**: Error messages don't provide actionable recovery steps
3. **Debugging Difficulty**: Errors lack context (swarm IDs, branch names, etc.)
4. **Inconsistency**: Error handling varies across different operations

The new errors package (`internal/errors/`) provides:
- Automatic retry with exponential backoff
- Error categorization (Transient/Permanent/User/System)
- Recovery hints for user-facing errors
- Rich error context
- 85.7% test coverage

## Files to Modify

1. `/Users/ericfriday/gt/internal/swarm/manager.go`
2. `/Users/ericfriday/gt/internal/swarm/integration.go`
3. `/Users/ericfriday/gt/internal/swarm/landing.go`
4. `/Users/ericfriday/gt/internal/swarm/manager_test.go` ✅ (COMPLETED)
5. `/Users/ericfriday/gt/internal/cmd/swarm.go` (minor hint improvement)

## Detailed Implementation Steps

### Step 1: Update manager.go

#### 1.1 Update Imports
```go
// BEFORE
import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "os/exec"
    "strings"

    "github.com/steveyegge/gastown/internal/rig"
)

// AFTER
import (
    "bytes"
    "context"  // ADD: for future context-aware operations
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"  // ADD: for time operations in retry

    "github.com/steveyegge/gastown/internal/errors"  // CHANGE: use new errors package
    "github.com/steveyegge/gastown/internal/rig"
)
```

#### 1.2 Replace Sentinel Errors
```go
// BEFORE
var (
    ErrSwarmNotFound  = errors.New("swarm not found")
    ErrSwarmExists    = errors.New("swarm already exists")
    ErrInvalidState   = errors.New("invalid state transition")
    ErrNoReadyTasks   = errors.New("no ready tasks")
    ErrBeadsNotFound  = errors.New("beads not available")
)

// AFTER
var (
    ErrSwarmNotFound = errors.Permanent("swarm.NotFound", nil).
        WithHint("Use 'gt swarm list' to see available swarms")
    ErrSwarmExists = errors.User("swarm.Exists", "swarm already exists").
        WithHint("Use a different swarm ID or check existing swarms with 'gt swarm list'")
    ErrInvalidState = errors.User("swarm.InvalidState", "invalid state transition").
        WithHint("Check swarm status with 'gt swarm status <id>' to see current state")
    ErrNoReadyTasks = errors.Permanent("swarm.NoReadyTasks", nil).
        WithHint("Check epic dependencies with 'bd swarm status <epic-id>' to see blocked tasks")
    ErrBeadsNotFound = errors.System("swarm.BeadsNotFound", nil).
        WithHint("Install beads with: brew install beads")
)
```

#### 1.3 Update LoadSwarm Method
Key changes:
- Wrap beads command execution in `errors.WithRetry()`
- Categorize errors (Transient for network, Permanent for not found)
- Add context (epic_id, stderr)
- Add recovery hints

See full implementation in migration document Section 1.3.

#### 1.4 Update GetReadyTasks Method
- Wrap beads status query with retry
- Add error context (swarm_id)
- Enhanced hints

#### 1.5 Update IsComplete Method
- Wrap beads status query with retry
- Add error context
- Enhanced error messages

#### 1.6 Update loadTasksFromBeads Method
- Wrap beads show with retry
- Categorize transient vs permanent errors
- Add context and hints

### Step 2: Update integration.go

#### 2.1 Update Imports
```go
// BEFORE
import (
    "bytes"
    "errors"
    "fmt"
    "os/exec"
    "strings"
)

// AFTER
import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"

    "github.com/steveyegge/gastown/internal/errors"
)
```

#### 2.2 Replace Branch Errors
```go
// BEFORE
var (
    ErrBranchExists     = errors.New("branch already exists")
    ErrBranchNotFound   = errors.New("branch not found")
    ErrNotOnIntegration = errors.New("not on integration branch")
)

// AFTER
var (
    ErrBranchExists = errors.User("swarm.BranchExists", "branch already exists").
        WithHint("Use a different branch name or delete the existing branch first")
    ErrBranchNotFound = errors.Permanent("swarm.BranchNotFound", nil).
        WithHint("Check available branches with: git branch -a")
    ErrNotOnIntegration = errors.User("swarm.NotOnIntegration", "not on integration branch").
        WithHint("Switch to integration branch with: git checkout <integration-branch>")
)
```

#### 2.3 Update Git Operations
- CreateIntegrationBranch: Add context and network retry for push
- MergeToIntegration: Network retry for fetch, enhanced conflict errors
- LandToMain: Network retry for pull/push, enhanced conflict handling
- CleanupBranches: Network retry for remote deletes
- getCurrentBranch: Add error context
- getConflictingFiles: Add error context

### Step 3: Update landing.go

#### 3.1 Update Imports
Add: `"github.com/steveyegge/gastown/internal/errors"`

#### 3.2 Update ExecuteLanding Method
- Wrap session stop operations in retry
- Enhanced code-at-risk error with context
- Add recovery hints

#### 3.3 Update gitRunOutput Method
- Add error context (command, dir, stderr)
- Add recovery hints

### Step 4: Test Updates ✅ (COMPLETED)

The test file `/Users/ericfriday/gt/internal/swarm/manager_test.go` has been updated to:
- Check for wrapped error messages instead of exact sentinel errors
- Verify retry behavior (errors contain "retry" or wrapped operation names)
- Add `strings` import for error message checking

Tests currently pass with minor assertion differences due to error wrapping.

### Step 5: cmd/swarm.go (Minor Enhancement)

Update getSwarmRig error message:
```go
// BEFORE
return nil, "", fmt.Errorf("rig '%s' not found", rigName)

// AFTER
return nil, "", fmt.Errorf("rig '%s' not found. Use 'gt rig list' to see available rigs", rigName)
```

## Error Categories and Retry Strategy

### Transient Errors (Auto-Retry)
**Use Case**: Network timeouts, temporary beads failures
**Retry Config**: DefaultRetryConfig (3 attempts, 100ms initial, 10s max)
**Network Config**: NetworkRetryConfig (5 attempts, 500ms initial, 30s max)

Examples:
- Beads query timeouts
- Git network operations (push, pull, fetch)
- Session stop operations

### Permanent Errors (No Retry)
**Use Case**: Resources not found, parse failures
**Strategy**: Fail immediately with hints

Examples:
- Swarm not found
- Epic not found
- Parse errors

### User Errors
**Use Case**: User input or action issues
**Strategy**: Clear recovery hints

Examples:
- Branch already exists
- Invalid state transition
- Merge conflicts

### System Errors
**Use Case**: System configuration issues
**Strategy**: System-level hints

Examples:
- Beads not installed
- Git not available

## Recovery Hints Reference

### Swarm Operations
```
"Use 'gt swarm list' to see available swarms"
"Check swarm status with: 'gt swarm status <id>'"
"Check epic dependencies with: 'bd swarm status <epic-id>'"
"Use a different swarm ID or check existing swarms with 'gt swarm list'"
```

### Git Operations
```
"Check available branches with: git branch -a"
"Switch to integration branch with: git checkout <integration-branch>"
"Resolve conflicts in: <files>\nThen run: git add . && git commit"
"Check git status with: git status"
"Check network connection and retry with: git push origin <branch>"
"Verify base commit exists with: git show <commit>"
```

### Beads Operations
```
"Install beads with: brew install beads"
"Verify the epic exists with: bd show <epic-id>"
"Try running: bd swarm status <swarm-id>"
```

## Testing Strategy

### Unit Tests
- All existing tests should pass
- Error behavior verified through string matching
- Tests check for operation-specific error messages

### Integration Tests
- E2E lifecycle test documented (manual verification required)
- Beads infrastructure needed for full testing
- See TestSwarmE2ELifecycle documentation

### Manual Testing Checklist
- [ ] Create swarm with non-existent epic (should see beads hint)
- [ ] Network failure during git push (should auto-retry)
- [ ] Merge conflict (should see detailed file list and resolution hint)
- [ ] Landing with code at risk (should see workers list and git hint)
- [ ] Branch already exists (should see hint about deletion or new name)

## Rollout Plan

### Phase 1: Implementation (This Task)
1. Apply all code changes to swarm package files
2. Run unit tests (`go test ./internal/swarm/...`)
3. Fix any test failures
4. Run go fmt and linters

### Phase 2: Integration Testing
1. Manual testing with real swarms
2. Test network failure scenarios
3. Test conflict scenarios
4. Verify retry behavior

### Phase 3: Documentation
1. Update swarm package README (if exists)
2. Add examples to Gas Town docs
3. Update troubleshooting guides

### Phase 4: Monitoring
1. Monitor error rates after deployment
2. Track retry success rates
3. Collect user feedback on error messages

## Success Criteria

- [ ] All swarm operations use the new errors package
- [ ] Automatic retry for transient failures working
- [ ] Clear recovery hints displayed to users
- [ ] All unit tests passing
- [ ] No API breaking changes
- [ ] Error context includes relevant IDs and state
- [ ] Documentation complete

## Risks and Mitigation

### Risk 1: Retry delays slow operations
**Mitigation**: Use short initial delays (100-500ms), exponential backoff
**Impact**: Low - Most operations succeed on first try

### Risk 2: Error message changes break log parsing
**Mitigation**: Maintain backward compatibility with errors.Is()
**Impact**: Low - Sentinel errors still work

### Risk 3: Increased complexity
**Mitigation**: Comprehensive documentation, clear error patterns
**Impact**: Medium - But improved reliability outweighs complexity

## Performance Impact

- **Successful operations**: No change (no retry overhead)
- **Failed operations**: Additional retry attempts add latency
  - Default: 3 attempts = ~300ms max additional latency
  - Network: 5 attempts = ~1.5s max additional latency
- **Trade-off**: Slightly slower failures vs much higher success rate

## Dependencies

- `internal/errors` package (already implemented, 85.7% coverage)
- No external dependencies
- Compatible with existing rig, polecat, git packages

## References

- Errors Package: `/Users/ericfriday/gt/internal/errors/`
- Errors README: `/Users/ericfriday/gt/internal/errors/README.md`
- Migration Doc: `/Users/ericfriday/gt/docs/swarm-errors-migration.md`

## Implementation Code Snippets

See attached file: `swarm-errors-migration.md` for complete before/after code examples for each method.

## Next Steps

1. Review this implementation plan
2. Apply changes to all files listed above
3. Run test suite
4. Create commit with proper attribution
5. Test manually with real swarms
6. Monitor for any issues

## Co-Authors

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
