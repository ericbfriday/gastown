# Workspace Cleanup System

Automated workspace cleanup with preflight and postflight hooks for maintaining clean workspace state.

## Overview

The workspace cleanup system provides:
- **Preflight checks**: Verify workspace is ready before starting work
- **Postflight cleanup**: Remove temp files, clear caches after work sessions
- **Hook integration**: Integrate with lifecycle hooks (gt-69l)
- **Configuration**: Define cleanup rules per workspace type
- **Safety**: Never delete uncommitted work, always backup important state

## Workspace Types

Each workspace type has its own cleanup configuration:

| Type | Description | Location | Cleanup Strategy |
|------|-------------|----------|------------------|
| `crew` | Persistent worker clone | `<rig>/crew/<name>/` | Conservative - safety first |
| `polecat` | Ephemeral worker worktree | `<rig>/polecats/<name>/` | Aggressive - full cleanup |
| `mayor` | Canonical read-only clone | `<rig>/mayor/rig/` | Minimal - keep pristine |
| `refinery` | Merge queue worktree | `<rig>/refinery/rig/` | Moderate - clear artifacts |
| `town` | Town-level workspace | `~/gt/` | Light - top-level only |

## CLI Usage

### Clean Workspace

```bash
# Postflight cleanup (default)
gt workspace clean

# Dry-run to preview changes
gt workspace clean --dry-run

# Preflight checks before starting work
gt workspace clean --preflight

# Clean specific workspace type
gt workspace clean --type crew

# Verbose output
gt workspace clean --verbose

# JSON output
gt workspace clean --json
```

### Workspace Status

```bash
# Show workspace status
gt workspace status

# Show detailed status
gt workspace status --verbose

# JSON output
gt workspace status --json

# Check specific workspace type
gt workspace status --type polecat
```

### Configuration Management

```bash
# Show current configuration
gt workspace config --show

# Show config for specific type
gt workspace config --show --type crew

# Export default configurations
gt workspace config --export --output cleanup.json
```

## Configuration

Cleanup configurations are stored in:
- `.gastown/cleanup.json` (preferred)
- `.claude/cleanup.json` (fallback)

### Configuration Format

```json
{
  "crew": {
    "workspace_type": "crew",
    "enabled": true,
    "dry_run": false,
    "preflight": [
      {
        "action": "verify-git-clean",
        "enabled": true,
        "safety_check": true,
        "description": "Verify git working directory is clean"
      },
      {
        "action": "remove-ds-store",
        "enabled": true,
        "patterns": [".DS_Store"],
        "recursive": true,
        "description": "Remove macOS metadata files"
      }
    ],
    "postflight": [
      {
        "action": "backup-uncommitted",
        "enabled": true,
        "safety_check": true,
        "backup_first": true,
        "description": "Backup uncommitted changes"
      },
      {
        "action": "remove-temp-files",
        "enabled": true,
        "patterns": ["*.tmp", "*.temp", "*.swp", "*.swo", "*~"],
        "recursive": true,
        "max_depth": 5,
        "description": "Remove temporary files"
      }
    ]
  }
}
```

## Cleanup Actions

| Action | Description | Safety |
|--------|-------------|--------|
| `verify-git-clean` | Check git working directory is clean | High |
| `backup-uncommitted` | Backup uncommitted changes to `.gastown/backups/` | High |
| `remove-temp-files` | Remove temp files (*.tmp, *.swp, etc.) | Medium |
| `remove-logs` | Remove log files (with exclusions) | Medium |
| `remove-ds-store` | Remove macOS .DS_Store files | Low |
| `clear-build-cache` | Clear build caches (dist/, .cache/, etc.) | Medium |
| `clean-git-worktree` | Run `git clean -fd` | High |
| `clean-node-modules` | Remove node_modules directory | Medium |

## Hook Integration

### Lifecycle Hooks Configuration

Add to `.gastown/hooks.json`:

```json
{
  "hooks": {
    "pre-session-start": [
      {
        "type": "builtin",
        "name": "workspace-preflight"
      }
    ],
    "post-shutdown": [
      {
        "type": "builtin",
        "name": "workspace-postflight"
      }
    ],
    "session-idle": [
      {
        "type": "builtin",
        "name": "workspace-postflight"
      }
    ]
  }
}
```

### Available Builtin Hooks

- `workspace-preflight`: Run preflight checks
- `workspace-postflight`: Run postflight cleanup
- `workspace-verify-clean`: Verify workspace is clean
- `workspace-backup`: Backup uncommitted changes

### Programmatic Usage

```go
import (
    "github.com/steveyegge/gastown/internal/workspace/cleanup"
    "github.com/steveyegge/gastown/internal/hooks"
)

// Register cleanup hooks
runner, _ := hooks.NewHookRunner("/path/to/workspace")
cleanup.RegisterCleanupHooks(runner)

// Fire preflight hook
ctx := &hooks.HookContext{
    WorkingDir: "/path/to/workspace",
    Metadata: map[string]interface{}{
        "workspace_type": "crew",
    },
}
results := runner.Fire(hooks.EventPreSessionStart, ctx)

// Check for blocking issues
for _, r := range results {
    if r.Block {
        log.Fatalf("Preflight blocked: %s", r.Message)
    }
}
```

## Default Configurations

### Crew Workspace (Conservative)

**Preflight:**
- Verify git is clean (blocks if dirty)
- Remove .DS_Store files

**Postflight:**
- Backup uncommitted changes
- Remove temp files (*.tmp, *.swp, etc.)
- Remove log files (with exclusions)
- Remove .DS_Store files

**On Idle:**
- Clear build caches

### Polecat Workspace (Aggressive)

**Preflight:**
- Clean git worktree

**Postflight:**
- Remove all temp files and logs
- Clear all build artifacts
- Remove .DS_Store files

**On Idle:** None (destroyed after use)

### Mayor Workspace (Pristine)

**Preflight:**
- Verify git is clean (strict)

**Postflight:**
- Remove .DS_Store files only

### Refinery Workspace (Moderate)

**Preflight:**
- Verify git is clean
- Remove temp files

**Postflight:**
- Clear build caches
- Remove .DS_Store files

### Town Workspace (Light)

**Preflight:**
- Remove .DS_Store files

**Postflight:**
- Remove top-level log files (exclude harness/, daemon/)
- Remove .DS_Store files

## Safety Features

### Backup Protection

Uncommitted changes are automatically backed up to:
```
.gastown/backups/backup-YYYYMMDD-HHMMSS.patch
```

Apply backup:
```bash
git apply .gastown/backups/backup-20260128-143000.patch
```

### Dry-Run Mode

Preview all changes before applying:
```bash
gt workspace clean --dry-run
```

### Safety Checks

Rules with `safety_check: true`:
- Verify before deletion
- Block operation on failure for preflight
- Warn but continue for postflight

### Exclusion Patterns

Protect important files:
```json
{
  "action": "remove-logs",
  "patterns": ["*.log"],
  "exclude": ["**/important.log", "**/keep/**"]
}
```

## Polecat Lifecycle Integration

Polecats are ephemeral workers with automatic cleanup:

```bash
# Create polecat (automatic preflight)
gt polecat create rust

# Work in polecat
cd ~/gt/duneagent/polecats/rust/duneagent/
# ... do work ...

# Destroy polecat (automatic postflight)
gt polecat destroy rust
```

Cleanup happens automatically:
1. **Pre-creation**: Clean git worktree
2. **Post-destruction**: Aggressive cleanup (all temp files, caches, artifacts)

## Monitoring & Reporting

### Cleanup Results

```json
{
  "action": "remove-temp-files",
  "success": true,
  "files_found": 45,
  "files_removed": 45,
  "bytes_freed": 2097152,
  "duration": "125ms",
  "details": [
    {
      "path": "/path/to/file.tmp",
      "action": "removed",
      "size": 1024
    }
  ]
}
```

### Preflight Check Results

```json
{
  "passed": true,
  "git_clean": true,
  "no_uncommitted": true,
  "build_clean": true,
  "can_proceed": true,
  "requires_backup": false
}
```

## Best Practices

### For Crew Members (Persistent Workers)

1. Run preflight before starting work:
   ```bash
   gt workspace clean --preflight
   ```

2. Run postflight after finishing:
   ```bash
   gt workspace clean --postflight
   ```

3. Check status regularly:
   ```bash
   gt workspace status
   ```

### For Polecats (Ephemeral Workers)

Cleanup is automatic via lifecycle hooks. No manual intervention needed.

### For Town-Level Work

```bash
# Clean town workspace
cd ~/gt
gt workspace clean --type town
```

## Troubleshooting

### "Git working directory is not clean"

**Cause:** Uncommitted changes in workspace

**Solution:**
```bash
# Backup changes
gt workspace clean --preflight  # Will create backup

# Or commit changes
git add .
git commit -m "Save work"
```

### "Safety check failed"

**Cause:** Critical safety check blocked operation

**Solution:**
1. Review the error message
2. Fix the underlying issue
3. Re-run cleanup

### Cleanup Not Running

**Cause:** Cleanup disabled or hooks not configured

**Solution:**
```bash
# Check configuration
gt workspace config --show

# Verify hooks
gt hooks lifecycle list
```

## Performance

Typical cleanup times:

- **Preflight checks**: 100-500ms
- **Postflight cleanup**: 500ms-2s (depends on file count)
- **Backup creation**: 50-200ms

Cleanup runs in foreground and blocks session lifecycle events to ensure clean state.

## Future Enhancements

Planned improvements:
- [ ] Background cleanup during idle time
- [ ] Configurable file size thresholds
- [ ] Integration with tmux session monitoring
- [ ] Cleanup scheduling (daily, weekly)
- [ ] Custom cleanup scripts
- [ ] Workspace health scoring
- [ ] Automatic backup rotation

## Related Systems

- **Lifecycle Hooks** (gt-69l): Event-driven hook system
- **Agent Monitoring** (gt-3yj): Status tracking and idle detection
- **Polecat Manager**: Ephemeral worker lifecycle
- **Session Manager**: Persistent worker sessions
