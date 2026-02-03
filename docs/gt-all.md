# gt all - Batch Polecat Operations

Run commands on multiple polecats simultaneously with pattern matching and filtering.

## Synopsis

```bash
gt all <command> [specs...] [flags]
```

## Description

The `gt all` command provides batch operations across multiple polecats. It supports flexible pattern matching, filtering, and parallel execution for operations like status checking, starting/stopping sessions, and running commands.

## Spec Patterns

Spec patterns allow you to target polecats flexibly:

- `Toast` - Specific polecat (requires `--rig` flag for disambiguation)
- `gastown/Toast` - Specific rig/polecat
- `gastown/*` - All polecats in a rig
- `*` - All polecats everywhere

## Global Flags

- `--rig <name>` - Filter by specific rig
- `--status <state>` - Filter by polecat status (working/idle/stuck/done)
- `--pattern <regex>` - Filter by name pattern (substring match)
- `--dry-run` - Show what would happen without executing
- `--parallel <n>` - Number of parallel workers (default: 5)

## Subcommands

### gt all status

Show status of multiple polecats in a summary table.

```bash
gt all status [specs...] [--json]
```

**Examples:**
```bash
# Status of all polecats
gt all status

# Status of gastown polecats
gt all status gastown/*

# Status filtered by rig and state
gt all status --rig gastown --status working

# JSON output
gt all status --json
```

**Output:**
- Rig/polecat name
- Current state (working, stuck, done)
- Session status (running/stopped)
- Assigned issue

### gt all stop

Stop tmux sessions for multiple polecats.

```bash
gt all stop [specs...] [--force]
```

**Options:**
- `--force` - Kill sessions immediately without graceful shutdown
- `--dry-run` - Preview what would be stopped

**Examples:**
```bash
# Stop all gastown polecats
gt all stop gastown/*

# Force kill all working polecats
gt all stop --status working --force

# Dry-run to preview
gt all stop --dry-run
```

**Behavior:**
- Confirms destructive operations (unless `--force`)
- Runs in parallel for efficiency
- Reports successes and failures separately
- Skips polecats that aren't running

### gt all start

Start tmux sessions for multiple polecats.

```bash
gt all start [specs...]
```

**Examples:**
```bash
# Start all gastown polecats
gt all start gastown/*

# Start specific polecats
gt all start gastown/Toast gastown/Furiosa

# Start idle polecats
gt all start --status idle
```

**Behavior:**
- Skips polecats that are already running
- Starts sessions in parallel
- Uses each polecat's configured working directory
- Resumes work on assigned issues (if any)

### gt all attach

Attach to multiple polecat sessions.

```bash
gt all attach [specs...]
```

**Examples:**
```bash
# Attach to all gastown polecats
gt all attach gastown/*

# Attach to working polecats
gt all attach --status working
```

**Behavior:**
- If one polecat: attaches directly to that session
- If multiple: lists all sessions with attach commands
- Only attaches to running sessions

### gt all run

Run a command in multiple polecat sessions.

```bash
gt all run <command> [specs...]
```

**Examples:**
```bash
# Check git status in all sessions
gt all run "git status" gastown/*

# Run gt status in all working polecats
gt all run "gt status" --status working

# Nudge all polecats
gt all run "# Resume work" gastown/*
```

**Behavior:**
- Sends command to each session's tmux pane
- Runs in parallel for efficiency
- Only targets running sessions
- Reports which sessions succeeded/failed

## Filtering

Filters can be combined for precise targeting:

```bash
# Working polecats in gastown
gt all status --rig gastown --status working

# Polecats matching pattern
gt all status --pattern "Toast.*"

# Combined filters
gt all stop --rig gastown --status working --pattern "Test.*"
```

## Parallel Execution

Operations run in parallel using a worker pool for efficiency:

- Default: 5 parallel workers
- Adjust with `--parallel <n>` flag
- Prevents overwhelming the system
- Shows progress for long operations

## Safety Features

### Confirmation Prompts

Destructive operations confirm before executing (unless `--force`):

```bash
gt all stop gastown/*
# Prompt: About to stop 5 polecat session(s). Continue? (y/N):
```

### Dry-Run Mode

Preview changes before executing:

```bash
gt all stop --dry-run gastown/*
# Would stop 5 polecat session(s):
#   - gastown/Toast
#   - gastown/Furiosa
#   ...
```

### Error Aggregation

Failed operations are reported separately:

```bash
✓ Stopped 3 session(s)

⚠ Warning: Some operations failed:
  - gastown/Stuck: session not found
  - greenplace/Max: permission denied
```

## Use Cases

### Monitor All Polecats

```bash
# Quick status check
gt all status --json | jq '.[] | select(.session_running == false)'
```

### Emergency Stop

```bash
# Kill all sessions immediately
gt all stop --force
```

### Selective Operations

```bash
# Stop only stuck polecats
gt all stop --status stuck

# Start idle polecats
gt all start --status idle
```

### Bulk Commands

```bash
# Pull latest changes in all sessions
gt all run "git pull" gastown/*

# Check completion status
gt all run "gt status" --status working
```

## Performance

Batch operations are optimized for speed:

- **Parallel execution**: Multiple operations run concurrently
- **Worker pools**: Prevents resource exhaustion
- **Non-blocking**: Operations don't wait for each other
- **Progress reporting**: Shows status for long operations

## Error Handling

Operations handle errors gracefully:

1. **Per-polecat errors**: Don't stop other operations
2. **Error aggregation**: All errors reported at end
3. **Partial success**: Shows what succeeded and what failed
4. **Exit codes**: Non-zero if any operation failed

## Exit Codes

- `0` - All operations succeeded
- `1` - One or more operations failed

## Examples

### Daily Workflow

```bash
# Morning: Start all polecats
gt all start gastown/*

# Check status during day
gt all status --status working

# Evening: Stop all sessions
gt all stop gastown/*
```

### Troubleshooting

```bash
# Find stuck polecats
gt all status --status stuck

# Check what's running
gt all status --json | jq '.[] | select(.session_running == true)'

# Force restart all
gt all stop --force gastown/*
gt all start gastown/*
```

### Bulk Management

```bash
# Update all working polecats
gt all run "git pull" --status working

# Nudge stalled polecats
gt all run "# Continue work" --status working

# Check progress across all
gt all run "gt status" gastown/*
```

## See Also

- `gt polecat` - Individual polecat management
- `gt polecat list` - List polecats with filtering
- `gt polecat status` - Detailed single polecat status
- `gt crew` - Batch operations for crew workers
