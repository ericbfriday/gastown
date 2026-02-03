# Lifecycle Hooks

Lifecycle hooks provide event-driven extensibility for Gas Town infrastructure events.

## Overview

This hook system handles infrastructure lifecycle events (session start/stop, mail received, etc.), which is separate from the Claude Code session hooks managed by `hooks/registry.toml`.

## Configuration

Hooks are configured in JSON format at:
- `.gastown/hooks.json` (preferred)
- `.claude/hooks.json` (fallback)

### Example Configuration

```json
{
  "hooks": {
    "pre-shutdown": [
      {
        "type": "builtin",
        "name": "pre-shutdown-checks"
      },
      {
        "type": "command",
        "cmd": "./scripts/cleanup.sh"
      }
    ],
    "post-session-start": [
      {
        "type": "command",
        "cmd": "./scripts/notify-start.sh"
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

## Supported Events

| Event | Description | Can Block |
|-------|-------------|-----------|
| `pre-session-start` | Before starting a session | Yes |
| `post-session-start` | After starting a session | No |
| `pre-shutdown` | Before shutting down | Yes |
| `post-shutdown` | After shutting down | No |
| `on-pane-output` | On tmux pane output | No |
| `session-idle` | When session becomes idle | No |
| `mail-received` | When mail is received | No |
| `work-assigned` | When work is assigned | No |

## Hook Types

### Command Hooks

Execute external scripts or commands:

```json
{
  "type": "command",
  "cmd": "./scripts/my-hook.sh"
}
```

Environment variables passed to command hooks:
- `GT_HOOK_EVENT`: The event name
- `GT_HOOK_TIMESTAMP`: ISO 8601 timestamp
- `GT_HOOK_*`: Additional metadata from context

Exit codes:
- `0`: Success
- Non-zero: Failure (blocks operation for pre-* events)

### Builtin Hooks

Internal Go functions:

```json
{
  "type": "builtin",
  "name": "pre-shutdown-checks"
}
```

Available builtin hooks:
- `pre-shutdown-checks`: Composite check (git clean, commits pushed, beads synced, assigned issues)
- `verify-git-clean`: Ensure working directory is clean
- `check-uncommitted`: Check for uncommitted changes (alias for verify-git-clean)
- `check-commits-pushed`: Verify all commits are pushed to remote
- `check-beads-synced`: Check if beads database is synced (non-blocking warning)
- `check-assigned-issues`: Check for hooked issues that need handling

## CLI Commands

### List Hooks

```bash
# List all hooks
gt hooks lifecycle list

# List hooks for specific event
gt hooks lifecycle list pre-shutdown

# JSON output
gt hooks lifecycle list --json
```

### Fire Hooks

```bash
# Manually fire hooks for an event
gt hooks lifecycle fire pre-shutdown

# With verbose output
gt hooks lifecycle fire pre-shutdown --verbose

# JSON output
gt hooks lifecycle fire pre-shutdown --json
```

### Test Configuration

```bash
# Validate configuration
gt hooks lifecycle test

# Validate and test execution
gt hooks lifecycle test --all

# JSON output
gt hooks lifecycle test --json
```

## Programmatic Usage

```go
import "github.com/steveyegge/gastown/internal/hooks"

// Create hook runner
runner, err := hooks.NewHookRunner("/path/to/town")
if err != nil {
    return err
}

// Fire hooks
ctx := &hooks.HookContext{
    WorkingDir: "/path/to/town",
    Metadata: map[string]interface{}{
        "session_id": "gt-rig-polecat",
    },
}

results := runner.Fire(hooks.EventPreShutdown, ctx)

// Check results
for _, result := range results {
    if !result.Success {
        log.Printf("Hook failed: %s", result.Error)
    }
    if result.Block {
        return fmt.Errorf("operation blocked by hook")
    }
}
```

## Integration Points

### Session Lifecycle

In `internal/polecat/session_manager.go`:

```go
import "github.com/steveyegge/gastown/internal/hooks"

func (m *SessionManager) Start(polecat string, opts SessionStartOptions) error {
    // Fire pre-session-start hooks
    runner, _ := hooks.NewHookRunner(m.rig.Path)
    ctx := &hooks.HookContext{
        WorkingDir: m.rig.Path,
        Metadata: map[string]interface{}{
            "polecat": polecat,
            "rig": m.rig.Name,
        },
    }

    results := runner.Fire(hooks.EventPreSessionStart, ctx)
    for _, r := range results {
        if r.Block {
            return fmt.Errorf("session blocked by hook: %s", r.Message)
        }
    }

    // ... start session ...

    // Fire post-session-start hooks
    runner.Fire(hooks.EventPostSessionStart, ctx)

    return nil
}
```

### Mail Router

In `internal/mail/router.go`:

```go
import "github.com/steveyegge/gastown/internal/hooks"

func (r *Router) DeliverMail(msg *Mail) error {
    // ... deliver mail ...

    // Fire mail-received hook
    runner, _ := hooks.NewHookRunner(r.townRoot)
    ctx := &hooks.HookContext{
        WorkingDir: r.townRoot,
        Metadata: map[string]interface{}{
            "from": msg.From,
            "to": msg.To,
            "subject": msg.Subject,
        },
    }
    runner.Fire(hooks.EventMailReceived, ctx)

    return nil
}
```

## Built-in Hook Development

To add a new built-in hook:

1. Add the function to `builtin.go`:
```go
func myNewHook(ctx *HookContext) (*HookResult, error) {
    // Your logic here
    return &HookResult{
        Success: true,
        Message: "Hook executed successfully",
    }, nil
}
```

2. Register it in `registerBuiltinHooks()`:
```go
func registerBuiltinHooks(r *HookRunner) {
    r.RegisterBuiltin("my-new-hook", myNewHook)
    // ... other hooks ...
}
```

3. Use it in configuration:
```json
{
  "type": "builtin",
  "name": "my-new-hook"
}
```

## Testing

Example test file structure:

```go
func TestHookRunner(t *testing.T) {
    runner, err := hooks.NewHookRunner(".")
    require.NoError(t, err)

    ctx := &hooks.HookContext{
        WorkingDir: ".",
    }

    results := runner.Fire(hooks.EventPreShutdown, ctx)
    // Assertions...
}
```

## Performance Considerations

- Hooks execute sequentially for each event
- Pre-* hooks stop execution if one blocks
- Command hooks have a 30-second default timeout
- Set custom timeout: `runner.SetTimeout(60 * time.Second)`

## Security

- Command hooks run with the same permissions as the Gas Town process
- Scripts should be carefully vetted before adding to hooks
- Use absolute paths or relative to town root
- Avoid sensitive data in hook output (it's logged)

## Troubleshooting

### Hook not executing

1. Check configuration file exists and is valid JSON
2. Verify event name is correct (case-sensitive)
3. For command hooks, ensure script is executable
4. Enable debug logging: `GT_DEBUG_HOOKS=1 gt lifecycle fire <event>`

### Hook fails silently

- Check hook result in JSON output: `gt hooks lifecycle fire <event> --json`
- Review hook error messages
- For command hooks, check script exit code and stderr

### Integration not working

- Ensure `hooks.NewHookRunner()` is called with correct town root
- Verify `Fire()` is called with appropriate context
- Check that hook results are being processed (especially `Block` flag)
