# Hook System Integration Summary

## Overview

The lifecycle hook system (gt-69l) has been fully implemented and integrated into Gas Town infrastructure.

## Implementation Status

### ✅ Completed

#### Core Hook System (Commit: b6f901d4)
- **Package**: `internal/hooks/`
  - `types.go`: Event constants, HookConfig, HookResult structures
  - `runner.go`: HookRunner with Fire() method, configuration loading
  - `builtin.go`: Built-in hooks (pre-shutdown-checks, verify-git-clean, check-uncommitted)
  - `README.md`: Comprehensive documentation

#### CLI Commands (Commit: b6f901d4)
- `gt hooks lifecycle list [event]` - List registered hooks
- `gt hooks lifecycle fire <event>` - Manually fire hooks
- `gt hooks lifecycle test [--all]` - Validate configuration

#### Integration Points (Commit: 9f6eaf8d)

**Polecat Session Manager** (`internal/polecat/session_manager.go`):
- Pre-session-start hooks (can block startup)
- Post-session-start hooks (best-effort)
- Pre-shutdown hooks (can block shutdown)
- Post-shutdown hooks (best-effort)

**Mail Router** (`internal/mail/router.go`):
- Mail-received hooks fire after successful delivery
- Hook context includes: from, to, subject, thread_id, reply_to, cc, priority

#### Bug Fix (Commit: 9f6eaf8d)
- Fixed infinite loop in `findGitDir()` function
- Now uses `filepath.Dir()` and `filepath.Join()` for proper path traversal

#### Testing (Current)
- Added `integration_test.go` with comprehensive tests
- Tests verify configuration loading, hook firing, and event constants
- All tests passing

#### Documentation (Current)
- Example configuration: `.gastown/hooks.json.example`
- Integration summary: This document

## Supported Events

| Event | Description | Can Block | Integration Point |
|-------|-------------|-----------|------------------|
| `pre-session-start` | Before starting a session | Yes | Session Manager |
| `post-session-start` | After starting a session | No | Session Manager |
| `pre-shutdown` | Before shutting down | Yes | Session Manager |
| `post-shutdown` | After shutting down | No | Session Manager |
| `on-pane-output` | On tmux pane output | No | (Future) |
| `session-idle` | When session becomes idle | No | (Future) |
| `mail-received` | When mail is received | No | Mail Router |
| `work-assigned` | When work is assigned | No | (Future) |

## Hook Types

### Command Hooks
Execute external scripts with:
- Environment variables: `GT_HOOK_EVENT`, `GT_HOOK_TIMESTAMP`, `GT_HOOK_*`
- Exit code 0: Success
- Non-zero exit code: Failure (blocks operation for pre-* events)
- 30-second default timeout

### Builtin Hooks
Internal Go functions:
- `pre-shutdown-checks`: Composite check (git clean, uncommitted)
- `verify-git-clean`: Ensure working directory is clean
- `check-uncommitted`: Check for uncommitted changes

## Configuration

Configuration file locations (checked in order):
1. `.gastown/hooks.json` (preferred)
2. `.claude/hooks.json` (fallback)

### Example Configuration

```json
{
  "hooks": {
    "pre-session-start": [
      {
        "type": "builtin",
        "name": "verify-git-clean"
      }
    ],
    "post-session-start": [
      {
        "type": "command",
        "cmd": "./scripts/notify-session-started.sh"
      }
    ],
    "pre-shutdown": [
      {
        "type": "builtin",
        "name": "pre-shutdown-checks"
      }
    ],
    "mail-received": [
      {
        "type": "command",
        "cmd": "./scripts/process-mail.sh"
      }
    ]
  }
}
```

## Usage Examples

### List All Hooks
```bash
gt hooks lifecycle list
```

### List Hooks for Specific Event
```bash
gt hooks lifecycle list pre-shutdown
```

### Manually Fire Hooks
```bash
gt hooks lifecycle fire pre-shutdown --verbose
```

### Validate Configuration
```bash
gt hooks lifecycle test
```

### Test All Hooks
```bash
gt hooks lifecycle test --all
```

## Integration Behavior

### Session Start
1. Pre-session-start hooks fire **before** session creation
2. If any hook blocks (non-zero exit), session start is aborted
3. After session is verified running, post-session-start hooks fire (best-effort)

### Session Shutdown
1. Pre-shutdown hooks fire **before** termination (unless force flag)
2. If any hook blocks, shutdown is aborted (unless force flag)
3. After session is killed, post-shutdown hooks fire (best-effort)

### Mail Delivery
1. Mail is delivered to recipient's inbox
2. Mail-received hook fires (best-effort, doesn't block delivery)
3. Hook receives full message context in metadata

## Hook Context

Hooks receive context via:

**Environment Variables** (Command hooks):
- `GT_HOOK_EVENT`: Event name
- `GT_HOOK_TIMESTAMP`: ISO 8601 timestamp
- `GT_HOOK_*`: Additional metadata (uppercase)

**HookContext struct** (Builtin hooks):
```go
type HookContext struct {
    Event      Event
    Timestamp  time.Time
    WorkingDir string
    Metadata   map[string]interface{}
}
```

### Metadata by Integration Point

**Session Manager** (pre/post-session-start, pre/post-shutdown):
- `polecat`: Polecat name
- `rig`: Rig name
- `session_id`: Tmux session identifier
- `issue_id`: Issue ID (if provided)

**Mail Router** (mail-received):
- `from`: Sender address
- `to`: Recipient address
- `subject`: Message subject
- `thread_id`: Thread identifier (if present)
- `reply_to`: Reply-to address (if present)
- `cc`: CC addresses (if present)
- `priority`: Message priority (if present)

## Future Integration Points

### On-Pane-Output
- Trigger: Tmux pane content changes
- Use case: Log monitoring, error detection
- Integration: Tmux pane-died hook extension

### Session-Idle
- Trigger: No activity for N minutes
- Use case: Auto-shutdown, resource cleanup
- Integration: Tmux activity monitoring

### Work-Assigned
- Trigger: Issue assigned to agent
- Use case: Notifications, tracking, metrics
- Integration: Beads assignment logic

## Testing

Run the integration test:
```bash
go test ./internal/hooks -run TestHookIntegration -v
```

Run all hook tests:
```bash
go test ./internal/hooks -v
```

## Separation from Claude Code Hooks

This lifecycle hooks system is **separate** from Claude Code session hooks:

| Feature | Lifecycle Hooks | Claude Code Hooks |
|---------|----------------|-------------------|
| Config | `.gastown/hooks.json` | `hooks/registry.toml` |
| Events | Infrastructure (sessions, mail) | Tool execution (PreToolUse) |
| Language | Go | Node.js (MCP) |
| Commands | `gt hooks lifecycle` | `gt hooks list/install` |

Both systems coexist and serve different purposes.

## Performance

- Hooks execute **sequentially** per event
- Pre-* hooks stop on first block
- Command hooks have 30-second timeout (configurable)
- Best-effort hooks don't fail the operation

## Security

- Command hooks run with GT process permissions
- Scripts should be vetted before adding
- Use absolute or town-root relative paths
- Avoid sensitive data in hook output (gets logged)

## Troubleshooting

### Hook Not Executing
1. Verify config file exists: `gt hooks lifecycle list`
2. Check event name spelling (case-sensitive)
3. For command hooks, ensure script is executable
4. Enable debug: `GT_DEBUG_SESSION=1` (for session hooks)

### Hook Fails Silently
- Check JSON output: `gt hooks lifecycle fire <event> --json`
- Review error messages in output
- For command hooks, check script exit code and stderr

### Pre-Hook Blocks Unexpectedly
- Run hook manually: `gt hooks lifecycle fire <event> --verbose`
- Check for uncommitted changes if using git hooks
- Use `--force` flag to bypass pre-shutdown hooks

## Related Issues

- **gt-69l**: Hook system for event extensibility (CLOSED)
- **hq-cv-sgaa4**: Convoy tracking remaining work (if any)

## Files Modified

- `internal/hooks/types.go` (new)
- `internal/hooks/runner.go` (new)
- `internal/hooks/builtin.go` (new, bugfix)
- `internal/hooks/integration_test.go` (new)
- `internal/hooks/README.md` (new)
- `internal/cmd/hooks_cmd.go` (new)
- `internal/cmd/lifecycle_hooks.go` (new)
- `internal/polecat/session_manager.go` (integration)
- `internal/mail/router.go` (integration)
- `.gastown/hooks.json.example` (documentation)
- `docs/hooks-integration-summary.md` (this document)

## Commits

- `b6f901d4`: feat(hooks): implement lifecycle hooks system
- `9f6eaf8d`: feat(plan-to-epic): (includes hook integration + bugfix)
- (Current): docs: add example config and integration summary
