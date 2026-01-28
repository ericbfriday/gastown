# Session Cycling Demo

## Quick Start

### Basic Session Cycle

The simplest way to cycle a session:

```bash
gt session cycle wyvern/Toast
```

This will:
1. Display TUI with current session status
2. Stop the existing session gracefully
3. Preserve context (last output, current issue)
4. Start a new session
5. Run all lifecycle hooks
6. Show completion status

### Cycle to Specific Issue

Restart and immediately work on a new issue:

```bash
gt session cycle wyvern/Toast --issue gt-abc123
```

The TUI will show:
- Previous issue being closed
- New issue being assigned
- Context transition

### Quick Restart (No TUI)

For scripts or automation:

```bash
gt session cycle wyvern/Toast --no-tui
```

Output:
```
Cycling session for wyvern/Toast...
  Stopping current session...
  Starting new session...
✓ Session cycled. Attach with: gt session at wyvern/Toast
```

## Interactive Session Management

### Starting Fresh Session

If no session exists, cycle command starts one:

```bash
gt session cycle wyvern/NewPolecat --issue gt-xyz789
```

TUI skips shutdown phases and goes directly to startup.

### Manual Stop Then Start

For more control:

```bash
# Stop with TUI feedback
gt session cycle wyvern/Toast --no-start

# Later, start with TUI
gt session cycle wyvern/Toast --no-stop --issue gt-next
```

## Advanced Usage

### Cycle and Attach

Automatically attach after cycling:

```bash
gt session cycle wyvern/Toast --attach
```

After transition completes, you're immediately in the session.

### Force Restart

If session is unresponsive:

```bash
gt session cycle wyvern/Toast --force
```

Skips graceful shutdown, kills session immediately.

### Batch Operations

Cycle multiple sessions:

```bash
for polecat in Toast Max Furiosa; do
    gt session cycle wyvern/$polecat --no-tui
done
```

## TUI Interaction

### During Transition

While the TUI is running, you can:

**View Help**: Press `?`
```
Session Cycling Commands

r   restart session      s   stop session
f   force stop          ↵   start session

?   toggle help         q   quit
```

**Monitor Progress**:
```
╭─────────────────────────────────────╮
│ ● shutting-down                     │
│ Stopping session and preserving...  │
│ Step 2/7                            │
╰─────────────────────────────────────╯

[██████░░░░░░░░░░░░░░░░] 29%
Elapsed: 1.2s

╭─────────────────────────────────────╮
│ Preserved Context:                  │
│   Previous issue: gt-abc123         │
│   Last output: ✓ Tests passed (2... │
╰─────────────────────────────────────╯
```

**Cancel Transition**: Press `q` or `Ctrl-C`
- Only works in idle/complete states
- During transition, must wait for completion

### After Completion

When complete, TUI shows:

```
╭─────────────────────────────────────╮
│ ✓ Transition Complete               │
│                                     │
│ Total time: 3.8s                    │
│                                     │
│ Transition:                         │
│   gt-abc123 → gt-xyz789             │
│                                     │
│ Preserved 3 context items           │
╰─────────────────────────────────────╯

✓ Session ready
```

Press `q` to exit or `r` to cycle again.

## Context Preservation

### What Gets Preserved

The TUI captures and displays:

1. **Previous Issue**: The issue that was hooked
2. **Next Issue**: The issue being assigned
3. **Last Output**: Terminal capture (50 lines)
4. **Custom Data**: Hook-provided context

### Viewing Preserved Context

During transition:
```
╭─────────────────────────────────────╮
│ Preserved Context:                  │
│   Previous issue: gt-abc123         │
│   Next issue: gt-xyz789             │
│   Last output: Successfully comp... │
│   work_dir: /path/to/workspace      │
│   branch: feature/new-work          │
╰─────────────────────────────────────╯
```

### Using Preserved Context

Context is available to hooks via `HookContext.Metadata`:

```bash
#!/bin/bash
# .claude/hooks/pre-start.sh

# Read preserved context
PREV_ISSUE=$(echo "$HOOK_METADATA" | jq -r '.previous_issue')
NEXT_ISSUE=$(echo "$HOOK_METADATA" | jq -r '.next_issue')

echo "Transitioning from $PREV_ISSUE to $NEXT_ISSUE"
```

## Hook Integration

### Lifecycle Hooks

The TUI visualizes hook execution:

1. **Pre-Shutdown** (`pre-shutdown` event)
   ```
   ● pre-shutdown
   Running pre-shutdown checks...
   ```

2. **Post-Shutdown** (`post-shutdown` event)
   ```
   ● post-shutdown-hook
   Running post-shutdown hooks...
   ```

3. **Pre-Start** (`pre-session-start` event)
   ```
   ● pre-start
   Running pre-start checks...
   ```

4. **Post-Start** (`post-session-start` event)
   ```
   ● post-startup-hook
   Running post-startup hooks...
   ```

### Hook Failures

If a hook fails, TUI shows detailed error:

```
╭─────────────────────────────────────╮
│ Error:                              │
│ Hook blocked startup: cleanup       │
│ required                            │
│                                     │
│ Pre-session-start hook blocked      │
╰─────────────────────────────────────╯

! Hook script exited with code 1
! Please resolve issues and retry
```

## Error Scenarios

### Session Not Running

```bash
gt session cycle wyvern/Idle
```

TUI detects and adapts:
```
ℹ Session not running
  Switching to startup flow...

● pre-start
Starting new session...
```

### Tmux Server Down

```
╭─────────────────────────────────────╮
│ Error:                              │
│ Failed to check session: tmux       │
│ server not running                  │
│                                     │
│ Check if tmux is running:           │
│   tmux ls                           │
╰─────────────────────────────────────╯
```

### Session Stuck

If session won't stop:

```bash
# Force stop with TUI feedback
gt session cycle wyvern/Stuck --force
```

TUI shows:
```
⚠ force-shutdown
Force stopping session...

✓ Session terminated
```

## Scripting Examples

### Automated Cycling

```bash
#!/bin/bash
# cycle-all-polecats.sh

for polecat in $(gt polecat list --rig wyvern --quiet); do
    echo "Cycling $polecat..."
    if gt session cycle wyvern/$polecat --no-tui; then
        echo "  ✓ Success"
    else
        echo "  ✗ Failed"
    fi
done
```

### Conditional Cycling

```bash
#!/bin/bash
# cycle-if-stale.sh

POLECAT="wyvern/Toast"
UPTIME=$(gt session status $POLECAT --format json | jq -r '.uptime_seconds')

if [ "$UPTIME" -gt 3600 ]; then
    echo "Session stale (uptime: ${UPTIME}s), cycling..."
    gt session cycle $POLECAT --no-tui
fi
```

### Issue-Based Cycling

```bash
#!/bin/bash
# cycle-to-issue.sh

POLECAT="wyvern/Toast"
NEXT_ISSUE=$(bd ready --limit 1 --format id)

if [ -n "$NEXT_ISSUE" ]; then
    echo "Cycling to issue: $NEXT_ISSUE"
    gt session cycle $POLECAT --issue $NEXT_ISSUE --attach
else
    echo "No ready work, keeping current session"
fi
```

## Best Practices

### When to Use TUI

✅ **Use TUI for**:
- Interactive session management
- Debugging transition issues
- Monitoring hook execution
- Learning session lifecycle

❌ **Skip TUI for**:
- Automated scripts
- Batch operations
- CI/CD pipelines
- Background jobs

### Performance Tips

1. **Use `--no-tui` for scripts**: Faster, less resource usage
2. **Avoid cycling too frequently**: Preserves session state
3. **Use `--force` sparingly**: Only for truly stuck sessions
4. **Monitor hook execution**: Slow hooks delay transitions

### Troubleshooting

**TUI won't start**:
```bash
# Check terminal capabilities
echo $TERM

# Ensure tmux is running
tmux ls

# Try without TUI
gt session cycle <rig>/<polecat> --no-tui
```

**Transition hangs**:
- Press `Ctrl-C` to cancel
- Check hook scripts for infinite loops
- Verify tmux session is responsive
- Use `--force` if needed

**Context not preserved**:
- Check tmux capture is working: `gt session capture <rig>/<polecat>`
- Verify hooks have access to metadata
- Ensure session manager has read permissions

## See Also

- [Session Management Documentation](../SESSION-MANAGEMENT.md)
- [Hook System Guide](../HOOKS.md)
- [Lifecycle Events Reference](../LIFECYCLE.md)
- [TUI Components](../TUI-COMPONENTS.md)
