# Mayor CLI Commands

CLI commands for managing the Mayor global coordinator session.

## Overview

The Mayor is the top-level coordinator agent in Gas Town. It manages infrastructure, coordinates between rigs, and provides global oversight. One Mayor per machine - multi-town setups require containers/VMs for isolation.

The Mayor session uses the `hq-mayor` session name (HQ prefix for town-level services).

## Commands

### gt mayor start

Start the Mayor session.

```bash
gt mayor start                  # Start with default agent (Claude)
gt mayor start --agent claude   # Start with specific agent type
gt mayor start --continue       # Resume from handoff mail (future)
```

**Flags:**
- `--agent <type>`: Specify agent type (default: claude)
- `--continue`: Resume from handoff mail (not yet implemented)

**Behavior:**
- Creates tmux session `hq-mayor`
- Working directory: town root
- Checks for existing/zombie sessions
- Applies Mayor theme (🎩)
- Waits for agent to start
- Sets up environment variables

### gt mayor attach

Attach to the running Mayor session.

```bash
gt mayor attach
gt mayor at        # Alias
```

**Aliases:** `at`

**Behavior:**
- Checks if Mayor is running
- Attaches to `hq-mayor` tmux session
- Detach with Ctrl-B D

### gt mayor stop

Stop the Mayor session.

```bash
gt mayor stop
gt mayor stop --grace-period 5
```

**Flags:**
- `--grace-period <seconds>`: Wait N seconds before shutdown

**Behavior:**
- Checks if Mayor is running
- Optional grace period delay
- Sends Ctrl-C for graceful shutdown
- Kills tmux session

### gt mayor status

Show Mayor session status.

```bash
gt mayor status
gt mayor status --json
```

**Flags:**
- `--json`: Output as JSON

**Output (human-readable):**
```
🎩 Mayor Session

  State: ● running
  Session ID: hq-mayor
  Attached: no
  Created: Mon Jan 27 10:30:45 2025
  Windows: 1

Attach with: gt mayor attach
```

**Output (JSON):**
```json
{
  "running": true,
  "session_id": "hq-mayor",
  "windows": 1,
  "created": "Mon Jan 27 10:30:45 2025",
  "attached": false,
  "activity": "10:45:23"
}
```

## Architecture

### Session Management

The Mayor commands use the `internal/mayor` package which provides:
- `mayor.Manager`: Manages Mayor lifecycle
- `mayor.Start()`: Creates and starts Mayor session
- `mayor.Stop()`: Stops Mayor session
- `mayor.IsRunning()`: Checks Mayor status
- `mayor.Status()`: Returns detailed session info

### Session Name

The Mayor uses a fixed session name defined in `internal/session`:
- `session.MayorSessionName()` → `"hq-mayor"`
- HQ prefix indicates town-level service
- One Mayor per machine (not per town)

### Working Directory

The Mayor session runs from the town root:
- `<town>/` (not `<town>/mayor/`)
- Ensures all `gt` commands work correctly
- Matches behavior of `gt handoff` for Mayor

### Environment Variables

The Mayor session sets:
- `GT_ROLE=mayor`: Identifies role
- `GT_TOWN_ROOT`: Absolute path to town root
- Other environment as needed by runtime

### Theme

The Mayor uses a dedicated theme:
- Icon: 🎩 (top hat - coordinator)
- Color scheme: Distinctive from other agents
- Status bar shows: role, unread mail, time
- Mouse mode enabled for clickable status

## Integration

### With Other Commands

- `gt start`: Can optionally start Mayor
- `gt stop`: Stops Mayor along with other agents
- `gt status`: Shows Mayor in town-level agents
- `gt nudge mayor <msg>`: Send message to Mayor
- `gt mail`: Mayor can send/receive mail
- `gt handoff mayor`: Hand off work to Mayor

### Session Discovery

Mayor sessions are discovered by:
- Session name pattern: `hq-mayor`
- BeadID pattern: `hq-mayor` (town beads)
- Environment variable: `GT_ROLE=mayor`

### Hooks

Mayor lifecycle supports hooks:
- Pre-session-start hooks
- Post-session-start hooks
- Pre-shutdown hooks
- Crash detection (pane-died hook)

## Implementation

### File Structure

```
internal/cmd/mayor.go         # CLI commands
internal/cmd/mayor_test.go    # Unit tests
internal/mayor/manager.go     # Session manager
internal/session/names.go     # Session name constants
docs/mayor-cli.md            # This documentation
```

### Key Functions

**internal/cmd/mayor.go:**
- `getMayorSessionName()`: Returns "hq-mayor"
- `runMayorStart()`: Implements start command
- `runMayorAttach()`: Implements attach command
- `runMayorStop()`: Implements stop command
- `runMayorStatus()`: Implements status command

**internal/mayor/manager.go:**
- `NewManager(townRoot)`: Creates manager
- `SessionName()`: Returns session name
- `Start(agentOverride)`: Starts session
- `Stop()`: Stops session
- `IsRunning()`: Check if running
- `Status()`: Get detailed status

## Examples

### Start Mayor and attach

```bash
cd ~/gt
gt mayor start
gt mayor attach
```

### Check status without attaching

```bash
gt mayor status
gt mayor status --json | jq .
```

### Stop with grace period

```bash
gt mayor stop --grace-period 10
```

### Integration with town lifecycle

```bash
# Start entire town (including Mayor)
gt start

# Check Mayor is running
gt mayor status

# Send message to Mayor
gt nudge mayor "Review infrastructure status"

# Stop entire town (including Mayor)
gt stop
```

## Future Enhancements

### --continue flag

The `--continue` flag is planned for resuming from handoff mail:

```bash
gt mayor start --continue
```

This will:
1. Check Mayor's inbox for handoff messages
2. Load most recent handoff context
3. Resume work from where predecessor left off
4. Similar to polecat `--issue` flag

### Health Checks

Future health monitoring:
- Periodic health checks via Deacon
- Automatic restart on crashes
- Alert on extended downtime

### Multi-Mayor

For multi-town setups:
- Container/VM isolation per town
- Separate `hq-mayor` per container
- Coordinated via external orchestration

## Testing

### Unit Tests

```bash
go test ./internal/cmd -run TestMayor
```

Tests verify:
- Session name format
- Command structure (subcommands)
- Flag definitions and types
- Alias configuration

### Manual Testing

```bash
# Test start/attach/stop cycle
gt mayor start
tmux list-sessions | grep hq-mayor
gt mayor status
gt mayor attach  # Ctrl-B D to detach
gt mayor stop

# Test error cases
gt mayor attach  # Should fail (not running)
gt mayor start
gt mayor start   # Should fail (already running)
gt mayor stop
gt mayor stop    # Should fail (not running)
```

### Integration Testing

The Mayor commands integrate with:
- Tmux session management
- Workspace detection
- Theme application
- Environment setup
- Mail routing
- Status monitoring

## Troubleshooting

### Mayor won't start

Check:
1. Are you in a Gas Town workspace? (`gt status`)
2. Is Mayor already running? (`gt mayor status`)
3. Is there a zombie session? (Kill it with `tmux kill-session -t hq-mayor`)

### Can't attach

Check:
1. Is Mayor running? (`gt mayor status`)
2. Is tmux installed? (`which tmux`)
3. Is tmux server running? (`tmux list-sessions`)

### Mayor keeps crashing

Check:
1. Town logs for crash reports
2. Claude process status in tmux
3. Disk space and permissions
4. Claude configuration validity

## See Also

- [Deacon CLI](./deacon-cli.md) - Town-level watchdog
- [Session Management](./sessions.md) - Tmux session patterns
- [Town Architecture](./architecture.md) - Mayor's role
- [gt start](./start.md) - Starting full infrastructure
