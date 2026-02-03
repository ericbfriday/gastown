# Implementation Summary: Session Pre-Shutdown Checks (gt-7o7)

## Overview

Implemented comprehensive pre-shutdown safety checks to prevent data loss when stopping polecat sessions. The system verifies clean state before shutdown and provides clear feedback on issues that need attention.

## Components Implemented

### 1. Built-in Hook Functions (`internal/hooks/builtin.go`)

Added four new built-in hook functions:

#### `checkCommitsPushed()`
- Verifies all local commits have been pushed to remote tracking branch
- Uses `git log @{u}..HEAD` to detect unpushed commits
- **Blocks shutdown:** Yes
- Handles edge cases: no upstream branch, empty repo, detached HEAD

#### `checkBeadsSynced()`
- Verifies beads database is synchronized
- Uses `bd sync --status` to check sync state
- **Blocks shutdown:** No (warning only)
- Gracefully handles missing bd command

#### `checkAssignedIssues()`
- Checks for hooked issues assigned to the polecat
- Uses `bd list --assignee=<agent> --status=hooked` to find pending work
- **Blocks shutdown:** Yes
- Shows issue IDs that need handling

#### Enhanced `preShutdownChecks()`
- Composite hook that runs all pre-shutdown checks
- Separates blocking failures from warnings
- Provides comprehensive feedback with clear error messages
- Now includes:
  - Git working tree clean
  - Commits pushed to remote
  - Beads database synced
  - Assigned issues handled

### 2. Integration Points

#### Session Manager (`internal/polecat/session_manager.go`)
- Already had `firePreShutdownHooks()` integration
- Pre-shutdown checks run automatically when `force=false`
- Hooks are bypassed when using `--force` flag

#### CLI (`internal/cmd/session.go`)
- Updated help text to document pre-shutdown checks
- Clarified `--force` flag behavior
- Listed all checks that are performed

### 3. Testing (`internal/hooks/builtin_test.go`)

Comprehensive test coverage for:
- `TestCheckCommitsPushed()` - Git push verification
- `TestCheckBeadsSynced()` - Beads sync check
- `TestCheckAssignedIssues()` - Issue assignment check
- `TestVerifyGitClean()` - Uncommitted changes check
- `TestPreShutdownChecks()` - Composite check integration
- `TestFindGitDir()` - Git directory discovery
- `TestJoinMessages()` - Message formatting

All tests pass successfully.

### 4. Documentation

#### `/docs/pre-shutdown-checks.md`
Comprehensive user-facing documentation including:
- Overview and usage examples
- Detailed explanation of each check
- Configuration instructions
- Troubleshooting guide
- Best practices

#### `internal/hooks/README.md`
Updated with:
- New built-in hooks documentation
- Testing instructions
- CLI usage examples for pre-shutdown checks

#### `.gastown/hooks.json.example`
Example configuration file showing how to enable pre-shutdown checks.

## Features Delivered

### 1. Pre-Shutdown Validation ✓
- ✅ Check for uncommitted work (git status)
- ✅ Check for unpushed commits
- ✅ Check for unsaved files (via git)
- ✅ Verify beads sync status
- ✅ Check for assigned issues

### 2. Integration with Hooks System ✓
- ✅ Uses `EventPreShutdown` lifecycle hook
- ✅ Hooks can block shutdown when unsafe
- ✅ Clear feedback on blocking issues
- ✅ Non-blocking warnings for advisory checks

### 3. CLI Support ✓
- ✅ Integrated with `gt session stop` command
- ✅ `--force` flag to override checks
- ✅ Clear error messages showing what needs attention
- ✅ Manual testing via `gt hooks lifecycle fire pre-shutdown`

### 4. Safety Mechanisms ✓
- ✅ Prevents shutdown if uncommitted changes exist
- ✅ Prevents shutdown if commits not pushed
- ✅ Prevents shutdown if issues are hooked but not handled
- ✅ Warns about beads sync issues
- ✅ Provides clear instructions on how to fix issues

## Usage Examples

### Normal Shutdown (with checks)
```bash
gt session stop wyvern/Toast
```

If checks fail:
```
Error: pre-shutdown hook blocked: Pre-shutdown checks failed:
  - git-clean: Working directory has uncommitted changes
  - commits-pushed: Branch main has 2 unpushed commit(s)
  - assigned-issues: Polecat has 1 hooked issue(s): gt-abc
```

### Force Shutdown (skip checks)
```bash
gt session stop wyvern/Toast --force
```

### Test Checks Manually
```bash
gt hooks lifecycle fire pre-shutdown --verbose
```

### Enable Checks
Create `.gastown/hooks.json`:
```json
{
  "hooks": {
    "pre-shutdown": [
      {
        "type": "builtin",
        "name": "pre-shutdown-checks"
      }
    ]
  }
}
```

## Technical Details

### Hook Execution Flow

1. User runs `gt session stop <rig>/<polecat>`
2. `SessionManager.Stop()` is called
3. If `force=false`, `firePreShutdownHooks()` is called
4. Hook runner loads configuration from `.gastown/hooks.json`
5. Each registered pre-shutdown hook executes in sequence
6. If any hook sets `Block=true`, shutdown is prevented
7. User receives clear error message with all failures

### Error Handling

- **Blocking Checks:** Return `Block=true` in HookResult
- **Non-Blocking Warnings:** Return `Success=false, Block=false`
- **Check Unavailable:** Return `Success=true` (don't block on missing tools)
- **Graceful Degradation:** Missing git/bd commands don't crash, just skip checks

### Edge Cases Handled

1. **Empty Git Repository:** Passes (can't get current branch)
2. **No Upstream Branch:** Passes (new branches are OK)
3. **Detached HEAD:** Passes (can't determine branch)
4. **No Beads Configured:** Passes (check skipped)
5. **No Polecat Metadata:** Passes (check skipped)

## Files Modified

1. `internal/hooks/builtin.go` - Added 4 new check functions
2. `internal/cmd/session.go` - Updated help text
3. `internal/hooks/README.md` - Updated documentation

## Files Created

1. `internal/hooks/builtin_test.go` - Comprehensive tests
2. `docs/pre-shutdown-checks.md` - User documentation
3. `.gastown/hooks.json.example` - Example configuration

## Testing

All tests pass:
```bash
go test ./internal/hooks -v
# PASS (10 tests, 0 failures)
```

## Dependencies

- Existing hooks system (`internal/hooks/`)
- Session manager (`internal/polecat/session_manager.go`)
- Git command-line tool
- Beads CLI (`bd`) - optional

## Backward Compatibility

✅ **Fully backward compatible**

- Pre-shutdown checks are opt-in via hooks configuration
- Without `.gastown/hooks.json`, behavior is unchanged
- `--force` flag continues to work as before
- No breaking changes to existing APIs

## Next Steps

1. **User Adoption:** Users need to create `.gastown/hooks.json` to enable checks
2. **Documentation:** Link to pre-shutdown-checks.md from main docs
3. **Integration Testing:** Test in real polecat workflows
4. **Monitoring:** Track how often checks block shutdown vs false positives

## Issue Resolution

This implementation fully addresses **gt-7o7** requirements:

✅ Pre-shutdown validation
✅ Integration with hooks system
✅ CLI support with --force flag
✅ Safety mechanisms
✅ Clear error messages
✅ Recovery options via fix instructions

The system provides robust protection against data loss while maintaining flexibility through the `--force` escape hatch.
