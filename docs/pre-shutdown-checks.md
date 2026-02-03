# Pre-Shutdown Checks

Session pre-shutdown checks prevent data loss by verifying clean state before shutting down polecat sessions.

## Overview

When you run `gt session stop`, the system performs several safety checks to ensure no work is lost:

1. **Git Working Tree Clean** - No uncommitted changes
2. **Commits Pushed** - All commits pushed to remote
3. **Beads Synced** - Beads database is synchronized
4. **Assigned Issues Handled** - No hooked issues pending

If any check fails, shutdown is blocked and you receive clear feedback on what needs attention.

## Usage

### Normal Shutdown (with checks)

```bash
gt session stop wyvern/Toast
```

If checks fail, you'll see output like:

```
Error: pre-shutdown hook blocked: hook blocked shutdown: Pre-shutdown checks failed:
  - git-clean: Working directory has uncommitted changes
  - commits-pushed: Branch main has 2 unpushed commit(s)
```

### Force Shutdown (skip checks)

```bash
gt session stop wyvern/Toast --force
```

This bypasses all pre-shutdown checks. Use with caution as it may result in data loss.

## Pre-Shutdown Checks Details

### 1. Git Working Tree Clean

**What it checks:** Verifies the working directory has no uncommitted changes using `git status --porcelain`.

**Blocks shutdown:** Yes

**Example failure:**
```
git-clean: Working directory has uncommitted changes
```

**How to fix:**
```bash
# Commit your changes
git add .
git commit -m "Save work"

# Or stash them
git stash
```

### 2. Commits Pushed

**What it checks:** Ensures all local commits have been pushed to the remote tracking branch using `git log @{u}..HEAD`.

**Blocks shutdown:** Yes

**Example failure:**
```
commits-pushed: Branch main has 3 unpushed commit(s)
```

**How to fix:**
```bash
git push
```

**Note:** If no upstream branch is configured (e.g., new branch), this check passes.

### 3. Beads Synced

**What it checks:** Verifies the beads database is in sync using `bd sync --status`.

**Blocks shutdown:** No (warning only)

**Example warning:**
```
beads-synced: Beads database may not be in sync
```

**How to fix:**
```bash
bd sync
```

**Note:** If beads sync is not configured, this check passes.

### 4. Assigned Issues Handled

**What it checks:** Verifies the polecat has no hooked issues that need handling using `bd list --assignee=<agent> --status=hooked`.

**Blocks shutdown:** Yes

**Example failure:**
```
assigned-issues: Polecat has 2 hooked issue(s) that need handling: gt-abc, gt-xyz
```

**How to fix:**
```bash
# Close completed issues
bd close gt-abc

# Or reassign if not done
bd update gt-xyz --assignee=someone-else
```

## Configuration

Pre-shutdown checks are configured via the hooks system. By default, they are **not enabled** unless you create a hooks configuration file.

### Enable Pre-Shutdown Checks

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

### Enable Individual Checks

You can enable specific checks instead of the composite `pre-shutdown-checks`:

```json
{
  "hooks": {
    "pre-shutdown": [
      {
        "type": "builtin",
        "name": "verify-git-clean"
      },
      {
        "type": "builtin",
        "name": "check-commits-pushed"
      }
    ]
  }
}
```

### Custom Checks

You can add custom pre-shutdown checks using command hooks:

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
        "cmd": "./scripts/custom-check.sh"
      }
    ]
  }
}
```

Custom command hooks should:
- Exit with code 0 on success
- Exit with non-zero on failure (blocks shutdown)
- Write error messages to stderr

## Testing Hooks

### List Configured Hooks

```bash
gt hooks lifecycle list pre-shutdown
```

### Test Hook Execution

```bash
gt hooks lifecycle fire pre-shutdown --verbose
```

This runs the pre-shutdown hooks without actually stopping the session.

### Validate Configuration

```bash
gt hooks lifecycle test
```

## Troubleshooting

### Hook not running

1. Check if `.gastown/hooks.json` exists:
   ```bash
   cat .gastown/hooks.json
   ```

2. Validate JSON syntax:
   ```bash
   gt hooks lifecycle test
   ```

3. Enable debug logging:
   ```bash
   GT_DEBUG_SESSION=1 gt session stop <rig>/<polecat>
   ```

### False positives

If a check is incorrectly blocking shutdown:

1. Use `--force` to bypass (temporary):
   ```bash
   gt session stop <rig>/<polecat> --force
   ```

2. Disable the problematic check in `.gastown/hooks.json`

3. Report the issue if it's a bug in the check logic

### Check passes but should fail

This may indicate:
- Git repo not configured correctly
- Beads not initialized
- Check needs improvement

Please file an issue with details about the scenario.

## Implementation Details

### Hook Integration

Pre-shutdown checks are implemented as lifecycle hooks in `internal/hooks/builtin.go`:

- `pre-shutdown-checks`: Composite check (runs all checks)
- `verify-git-clean`: Git status check
- `check-commits-pushed`: Unpushed commits check
- `check-beads-synced`: Beads sync check (non-blocking)
- `check-assigned-issues`: Hooked issues check

### Session Manager Integration

The `SessionManager.Stop()` method in `internal/polecat/session_manager.go` fires `EventPreShutdown` hooks before terminating the session:

```go
func (m *SessionManager) Stop(polecat string, force bool) error {
    // ...
    if !force {
        if err := m.firePreShutdownHooks(townRoot, polecat, workDir); err != nil {
            return fmt.Errorf("pre-shutdown hook blocked: %w", err)
        }
    }
    // ...
}
```

### Blocking vs Non-Blocking

Checks can be blocking or non-blocking:

- **Blocking:** Prevents shutdown if check fails (e.g., uncommitted changes)
- **Non-blocking:** Shows warning but allows shutdown (e.g., beads sync)

This is controlled by the `Block` field in `HookResult`.

## Best Practices

1. **Enable checks in production**: Always use pre-shutdown checks in production environments to prevent data loss.

2. **Test after enabling**: Run `gt hooks lifecycle fire pre-shutdown` to ensure checks work correctly.

3. **Document custom checks**: If you add custom command hooks, document their purpose and expected behavior.

4. **Use --force sparingly**: Only use `--force` when you're certain it's safe to bypass checks.

5. **Fix issues promptly**: When shutdown is blocked, fix the underlying issue rather than forcing shutdown.

## Related Documentation

- [Lifecycle Hooks](../internal/hooks/README.md)
- [Session Management](./sessions.md)
- [Git Workflow](./git-workflow.md)
- [Beads Database](./beads.md)
