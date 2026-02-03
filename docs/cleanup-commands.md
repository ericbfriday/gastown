# Cleanup Commands

The `gt cleanup` command provides tools for cleaning up stale state files, orphaned processes, and abandoned resources in Gas Town workspaces.

## Overview

Gas Town workspaces can accumulate stale state over time due to:
- Sessions terminated without proper cleanup
- Claude processes that outlive their parent sessions
- Lock files from crashed processes
- Abandoned git worktrees
- Temporary files and caches

The cleanup commands help maintain workspace health by safely removing these artifacts.

## Commands

### `gt cleanup processes` (default)

Clean up orphaned Claude processes that are not associated with any active tmux session.

```bash
gt cleanup processes              # Clean with confirmation
gt cleanup processes --dry-run    # Preview what would be cleaned
gt cleanup processes --force      # Clean without confirmation
gt cleanup                        # Shorthand (processes is default)
```

**What it cleans:**
- Claude/codex processes without a controlling terminal
- Processes not belonging to any active Gas Town tmux session
- Processes older than 60 seconds (prevents false positives)

**Signal escalation:**
1. First encounter: Send SIGTERM, wait for grace period
2. Still alive after 60s: Send SIGKILL
3. Survives SIGKILL: Report as unkillable

### `gt cleanup stale`

Clean stale lock files, PID files, and abandoned git worktrees.

```bash
gt cleanup stale                  # Clean with confirmation
gt cleanup stale --dry-run        # Preview what would be cleaned
gt cleanup stale --force          # Clean without confirmation
```

**What it cleans:**
- Lock files older than 1 hour in `.runtime/` and `.beads/`
- PID files referencing non-existent processes
- Git worktrees pointing to non-existent `.git` directories

**Safety features:**
- Abandoned worktrees always require individual confirmation (unless `--force`)
- Lock and PID files only removed if clearly stale
- Dry-run mode to preview before deletion

### `gt cleanup temp`

Remove temporary files and caches.

```bash
gt cleanup temp                   # Clean with confirmation
gt cleanup temp --dry-run         # Preview what would be cleaned
gt cleanup temp --force           # Clean without confirmation
```

**What it cleans:**
- Temporary files in `.runtime/tmp/`
- Gas Town temp files in system temp directory (`/tmp/gastown-*`)
- Only files older than 1 hour

**Reports:**
- Total number of files/directories cleaned
- Total bytes freed

### `gt cleanup sessions`

Clean zombie and disconnected tmux sessions.

```bash
gt cleanup sessions               # Clean with confirmation
gt cleanup sessions --dry-run     # Preview what would be cleaned
gt cleanup sessions --force       # Clean without confirmation
```

**What it cleans:**
- Tmux sessions with no active panes
- Disconnected sessions (client gone)
- Stale session state files

**Note:** Currently this is a placeholder implementation. Full session cleanup will be implemented in a future update.

## Common Options

### `--dry-run`

Preview what would be cleaned without making any changes.

```bash
gt cleanup stale --dry-run
gt cleanup processes --dry-run
```

Output shows:
- What files/processes would be affected
- Total size that would be freed
- No actual deletion or termination occurs

### `--force` / `-f`

Skip confirmation prompts and proceed with cleanup.

```bash
gt cleanup processes -f
gt cleanup stale --force
```

**Warning:** Use with caution, especially for `cleanup stale` which can remove worktrees.

### `--all`

Clean all stale state (runs all cleanup commands).

```bash
gt cleanup --all                  # Clean everything with confirmations
gt cleanup --all --dry-run        # Preview all cleanup
gt cleanup --all --force          # Clean everything without confirmation
```

Equivalent to running:
```bash
gt cleanup stale
gt cleanup temp
gt cleanup sessions
gt cleanup processes
```

## Examples

### Routine cleanup after development work

```bash
# Preview what needs cleaning
gt cleanup --all --dry-run

# Clean processes and temp files
gt cleanup processes -f
gt cleanup temp -f
```

### Recover from crashed sessions

```bash
# Find stale lock files and PIDs
gt cleanup stale --dry-run

# Clean them up
gt cleanup stale
```

### Before shutting down Gas Town

```bash
# Full cleanup with preview
gt cleanup --all --dry-run

# Execute if results look safe
gt cleanup --all -f
```

### Manual cleanup of specific worktree

```bash
# Find abandoned worktrees
gt cleanup stale --dry-run

# Review the list, then clean
gt cleanup stale
# (will prompt for each worktree)
```

## Integration with Doctor

The `gt doctor` command includes checks for stale state and orphaned processes:

- `orphan-processes` check detects orphaned Claude processes
- `orphan-sessions` check detects zombie tmux sessions
- `crew-worktrees` check detects stale cross-rig worktrees

Use `gt doctor --fix` to automatically fix some issues, or use specific `gt cleanup` commands for more targeted cleanup.

## Safety Guarantees

1. **Dry-run by default**: All cleanup operations support `--dry-run` to preview changes
2. **Confirmation prompts**: Dangerous operations (worktree removal) always prompt unless `--force`
3. **Age checks**: Processes and files must be stale before cleanup (prevents false positives)
4. **Graceful escalation**: Process cleanup starts with SIGTERM before SIGKILL
5. **Detailed reporting**: Shows exactly what was cleaned and any errors encountered

## Return Codes

- `0`: Success, cleanup completed
- `1`: Error occurred during cleanup

## Related Commands

- `gt doctor` - Diagnose workspace health issues
- `gt doctor --fix` - Automatically fix detected issues
- `gt status` - View current workspace state

## Implementation Notes

### Lock File Detection
- Looks in `.runtime/` and `.beads/` directories
- Only removes locks older than 1 hour
- Prevents race conditions with active operations

### PID File Detection
- Reads PID from file
- Checks if process exists using signal 0
- Removes if process is dead or PID is invalid

### Worktree Detection
- Checks `.git` file contents
- Follows `gitdir:` reference
- Removes if reference is broken

### Process Detection
- Uses tmux session verification
- Cross-references with ps output
- Minimum age requirement (60s) prevents false positives

## Troubleshooting

### "No orphaned processes found" but processes exist

The cleanup command only targets processes without tmux sessions. If processes have a TTY but their session is dead, they may not be detected. Try:

```bash
gt doctor --fix  # More comprehensive check
```

### Lock files keep coming back

If lock files reappear immediately, a process may be legitimately using them. Check:

```bash
gt status        # View active sessions
gt doctor        # Check for stuck processes
```

### Worktree removal fails

If a worktree can't be removed:

```bash
# Check if it's in use
cd /path/to/worktree
git status

# Force removal if truly abandoned
rm -rf /path/to/worktree
git worktree prune
```

### Cannot kill process (unkillable)

Some processes may survive SIGKILL (rare). These typically indicate:
- System processes (shouldn't be targeted)
- Zombie processes (already dead, just waiting for parent)
- Permission issues

Check process details:
```bash
ps -p <PID> -o pid,ppid,state,comm
```

If truly stuck, a system reboot may be needed.
