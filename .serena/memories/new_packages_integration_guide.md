# New Packages Integration Guide

**Created:** Iterations 1-3 (2026-01-28)
**Status:** Ready for Integration
**Packages:** 8 new packages delivered

## Package Overview

### 1. errors - Error Handling Package ⭐ HIGH PRIORITY

**Location:** `/Users/ericfriday/gt/internal/errors/`

**Purpose:** Comprehensive error handling with categorization, retry logic, and recovery hints

**Files:**
- `errors.go` (6.7KB) - Core error types and categorization
- `retry.go` (6.5KB) - Automatic retry with exponential backoff
- `hints.go` (9.1KB) - 20+ recovery hints
- `domain.go` (4.5KB) - Domain-specific errors
- `README.md` (13KB) - Complete documentation
- 5 test files (85.7% coverage)

**Key Features:**
```go
// Error categorization
type Category int
const (
    Transient   // Retry automatically
    Permanent   // Don't retry
    UserError   // User input issue
    SystemError // System/config issue
)

// Automatic retry
err := errors.Retry(ctx, errors.NetworkStrategy, func() error {
    return httpClient.Get(url)
})

// Recovery hints
err := errors.New("git push failed").
    WithHint(errors.HintGitPush).
    WithSeverity(errors.SeverityHigh)
```

**Integration Priority:** HIGH
- Migrate refinery error handling
- Migrate swarm error handling
- Add to all network operations
- Add to file operations
- Add to git operations

**Benefits:**
- Automatic retry for transient failures
- Consistent error messages across codebase
- Recovery hints for users
- Reduced support burden

**Integration Effort:** 2-3 days
**Breaking Changes:** None (additive only)

### 2. prompt - Interactive Prompt System ⭐ HIGH PRIORITY

**Location:** `/Users/ericfriday/gt/internal/prompt/`

**Purpose:** Unified interactive confirmation system with global override

**Files:**
- `prompt.go` (324 lines) - 5 prompt functions
- `prompt_test.go` (127 lines) - Test coverage
- `README.md` (292 lines) - Documentation
- `docs/prompt-system.md` (242 lines) - System guide

**Key Features:**
```go
// Simple confirmation
if !prompt.Confirm("Delete worktree?") {
    return nil
}

// Danger prompt (red text)
if !prompt.ConfirmDanger("This will delete ALL data") {
    return nil
}

// Batch confirmation
if !prompt.ConfirmBatch("Stop 15 workers?", 15) {
    return nil
}

// Choice selection
choice := prompt.Choice("Select action:", []string{"start", "stop", "restart"})

// Text input
name := prompt.Input("Enter name:", "default")
```

**Global Override:**
- `--yes` / `-y` flag on all commands
- `GT_YES=1` environment variable
- Non-interactive mode for automation

**Integration Status:**
- ✅ Already integrated in 6 commands:
  - `gt cleanup` - All subcommands
  - `gt orphans remove`
  - `gt start` - Confirmation before start
  - `gt uninstall`
  - `gt formula uninstall`
  - `gt all` - Batch operations

**Integration Priority:** MEDIUM
- Add to remaining destructive commands
- Ensure consistency across CLI
- Document override usage

**Integration Effort:** 1 day
**Breaking Changes:** None (opt-in)

### 3. filelock - File Locking System ⭐ CRITICAL

**Location:** `/Users/ericfriday/gt/internal/filelock/`

**Purpose:** Concurrency-safe file access with retry and stale lock cleanup

**Files:**
- `filelock.go` - Exclusive and shared locks
- `filelock_test.go` - Comprehensive tests
- `examples_test.go` (316 lines) - Usage examples
- `INTEGRATION.md` (551 lines) - Integration guide

**Key Features:**
```go
// Exclusive lock with retry
err := filelock.WithWriteLock(path, func() error {
    return os.WriteFile(path, data, 0644)
})

// Shared lock for reads
err := filelock.WithReadLock(path, func() error {
    data, err := os.ReadFile(path)
    return err
})

// Manual locking
lock := filelock.New(path)
if err := lock.Lock(); err != nil {
    return err
}
defer lock.Unlock()
```

**Features:**
- Exclusive and shared locks (flock on Unix)
- Exponential backoff retry (10ms → 50ms → 100ms)
- Stale lock detection (dead process cleanup)
- Cross-platform (Unix/Windows)

**Integration Status:**
- ✅ Integrated: `internal/state/state.go` (fixed import path)
- ❌ TODO: beads database operations
- ❌ TODO: registry operations
- ❌ TODO: queue operations

**Integration Priority:** CRITICAL
- Beads database MUST use locking (concurrent access issues)
- Registry operations need protection
- Mail queue needs protection
- Any shared file access needs locking

**Integration Effort:** 3-4 days
**Breaking Changes:** None (wrapper functions)

**Migration Path:**
1. Identify all shared file access points
2. Wrap with filelock.WithWriteLock() / WithReadLock()
3. Test concurrent access scenarios
4. Monitor for stale locks

### 4. tui/session - Session Cycling TUI

**Location:** `/Users/ericfriday/gt/internal/tui/session/`

**Purpose:** Bubbletea TUI for smooth session cycling with visual feedback

**Files:**
- `model.go` (416 lines) - State machine
- `view.go` (270 lines) - Lipgloss rendering
- `keys.go` (60 lines) - Keyboard shortcuts
- `model_test.go` (182 lines) - Tests
- `README.md` (415 lines) - Documentation

**Key Features:**
- 9-phase state machine (idle → stopping → starting → complete)
- Visual spinners and progress bar
- Hook integration with visual feedback
- Context preservation between sessions
- Interactive controls (?, q, r, s, Enter, f)

**Integration Status:**
- ✅ Integrated: `gt session cycle` command
- Supports both TUI and non-TUI modes

**Integration Priority:** LOW (feature complete)
**Integration Effort:** None needed (already integrated)

### 5. workspace/cleanup - Workspace Cleanup

**Location:** `/Users/ericfriday/gt/internal/workspace/cleanup/`

**Purpose:** Automated preflight/postflight workspace cleanup

**Files:**
- 6 Go files (~1,700 lines total)
- Test files with coverage
- Documentation

**Key Features:**
- 5 workspace type configurations (crew, polecat, mayor, refinery, town)
- Preflight checks (git clean, dependencies, permissions)
- Postflight cleanup (temp files, logs, stale state)
- Hook integration (pre-session-start, post-shutdown)

**Integration Status:**
- ✅ Integrated with hook system
- Ready for use

**Integration Priority:** LOW (feature complete)
**Integration Effort:** None needed (hook-based activation)

### 6. planconvert - Plan-to-Epic Converter

**Location:** `/Users/ericfriday/gt/internal/planconvert/`

**Purpose:** Convert planning documents to executable work items

**Files:**
- `parser.go` - Markdown parser (enhanced in Iteration 2)
- `parser_test.go` - Comprehensive tests
- Test fixtures

**Key Features:**
- YAML frontmatter parsing (version, status, date, author)
- Checkbox task extraction (- [ ], - [x], ✅, ☐)
- Section header recognition
- 4 output formats (json, jsonl, pretty, shell)

**Integration Status:**
- ✅ Fixed in Iteration 2 (was broken)
- ✅ Tested with real documents (24-41 tasks extracted)
- ✅ Used by plan-oracle plugin

**Integration Priority:** LOW (already working)
**Integration Effort:** None needed

### 7. daemon/mail_orchestrator - Mail Daemon

**Location:** `/Users/ericfriday/gt/internal/daemon/mail_orchestrator.go`

**Purpose:** Async mail processing with 3-queue architecture

**Files:**
- `mail_orchestrator.go` (650+ lines)
- `mail_orchestrator_test.go` (250+ lines)
- CLI command integration
- Documentation

**Key Features:**
- 3 queues: inbound, outbound, dead letter
- Priority-based processing (urgent > high > normal > low)
- Retry with exponential backoff (3 attempts, 5min delay)
- CLI: `gt mail daemon start/stop/status/logs/queue`

**Integration Status:**
- ✅ Complete implementation
- ✅ CLI commands ready
- ✅ Hook integration

**Integration Priority:** MEDIUM (when async mail needed)
**Integration Effort:** None needed (daemon-based activation)

### 8. hooks/builtin - Enhanced Hooks

**Location:** `/Users/ericfriday/gt/internal/hooks/builtin.go`

**Purpose:** Lifecycle hooks for infrastructure events

**Enhancement:** Pre-shutdown checks added in Iteration 3

**Key Features:**
- 8 event types (pre/post-session-start, pre/post-shutdown, mail-received, etc.)
- 4 builtin pre-shutdown checks:
  - checkCommitsPushed() - Verify commits pushed
  - checkBeadsSynced() - Check beads sync
  - checkAssignedIssues() - Verify no pending hooked issues
  - preShutdownChecks() - Composite validation

**Integration Status:**
- ✅ Fully integrated
- ✅ CLI commands: `gt hooks`, `gt hooks lifecycle`
- ✅ Used by cleanup system

**Integration Priority:** LOW (already integrated)
**Integration Effort:** None needed

## Integration Priorities

### CRITICAL (Do First)
1. **filelock** - Prevent data corruption in beads, registry, queues
   - Integration effort: 3-4 days
   - Impact: HIGH (prevents data loss)
   - Risk: HIGH if not done (concurrent access corruption)

### HIGH (Do Soon)
2. **errors** - Consistent error handling and retry logic
   - Integration effort: 2-3 days
   - Impact: MEDIUM (better UX, fewer failures)
   - Risk: LOW (additive only)

### MEDIUM (Do When Needed)
3. **prompt** - Remaining destructive commands
   - Integration effort: 1 day
   - Impact: LOW (UX improvement)
   - Risk: NONE (opt-in)

4. **mail_orchestrator** - When async mail processing needed
   - Integration effort: None (daemon activation)
   - Impact: HIGH (for async workflows)
   - Risk: NONE (optional daemon)

### LOW (Already Complete)
5. **tui/session** - Already integrated in `gt session cycle`
6. **workspace/cleanup** - Hook-based, already active
7. **planconvert** - Already working in plan-oracle
8. **hooks/builtin** - Already integrated

## Integration Checklist

### Phase 1: Critical Safety (Week 1)
- [ ] Audit all shared file access (beads, registry, queues)
- [ ] Wrap beads database operations with filelock
- [ ] Wrap registry operations with filelock
- [ ] Wrap queue operations with filelock
- [ ] Test concurrent access scenarios
- [ ] Monitor for stale locks

### Phase 2: Error Handling (Week 2)
- [ ] Migrate refinery to errors package
- [ ] Migrate swarm to errors package
- [ ] Add retry logic to network operations
- [ ] Add recovery hints to common errors
- [ ] Update error messages for consistency

### Phase 3: UX Improvements (Week 3)
- [ ] Add prompts to remaining destructive commands
- [ ] Document --yes / GT_YES override usage
- [ ] Ensure CLI consistency
- [ ] Test automation scenarios

### Phase 4: Validation (Week 4)
- [ ] Run full test suite
- [ ] Test concurrent scenarios
- [ ] Test error recovery paths
- [ ] Test prompt overrides
- [ ] Document integration status

## Testing Strategy

### filelock Integration Testing
```bash
# Concurrent beads access
go test -race ./internal/beads/... -v

# Stress test with parallel operations
for i in {1..10}; do
  (bd new "test-$i" &)
done
wait

# Stale lock cleanup
# (kill process mid-lock, verify cleanup)
```

### errors Integration Testing
```bash
# Retry logic
go test ./internal/errors/... -v -run TestRetry

# Network errors
# (simulate network failures, verify retry)

# Recovery hints
# (trigger errors, verify hints displayed)
```

### prompt Integration Testing
```bash
# Interactive mode
gt cleanup stale

# Non-interactive mode
gt cleanup stale --yes

# Environment override
GT_YES=1 gt cleanup stale
```

## Migration Path

### For filelock
1. Identify shared file access: `rg 'os\.(Read|Write)File'`
2. Audit for concurrent access risks
3. Wrap with appropriate lock type
4. Test with race detector: `go test -race`
5. Monitor for stale locks in production

### For errors
1. Find error creation: `rg 'errors\.New|fmt\.Errorf'`
2. Replace with errors package
3. Add appropriate category/severity
4. Add recovery hints where helpful
5. Test retry logic for transient errors

### For prompt
1. Find destructive operations
2. Add prompt.Confirm() or prompt.ConfirmDanger()
3. Document --yes flag usage
4. Test both interactive and non-interactive modes

## Documentation Updates Needed

- [ ] Update README with new packages
- [ ] Add migration guides to docs/
- [ ] Document --yes flag consistently
- [ ] Add filelock best practices
- [ ] Document error categorization strategy
- [ ] Update ARCHITECTURE.md

## Breaking Changes

**NONE** - All packages are additive only:
- errors: New package, doesn't replace existing
- prompt: Opt-in for new commands
- filelock: Wrapper functions, backward compatible
- All others: Feature additions

## Success Criteria

### filelock
- ✅ Zero data corruption from concurrent access
- ✅ Stale locks cleaned up automatically
- ✅ No deadlocks in stress testing
- ✅ Race detector passes

### errors
- ✅ Consistent error messages across codebase
- ✅ Retry logic reduces transient failures
- ✅ Recovery hints reduce support burden
- ✅ Error categorization aids debugging

### prompt
- ✅ All destructive commands have confirmations
- ✅ --yes flag works consistently
- ✅ Non-interactive mode works in automation
- ✅ No accidental data loss from commands

---

**Next Steps:** Begin Phase 1 (Critical Safety) with filelock integration
**Estimated Timeline:** 4 weeks for complete integration
**Risk Level:** LOW (all additive, no breaking changes)
