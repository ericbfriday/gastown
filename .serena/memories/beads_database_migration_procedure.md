# Beads Database Migration Procedure

**Created:** 2026-01-28  
**Context:** Resolved critical database mismatch in gastown  
**Status:** Verified working

## Problem Description

### Symptoms
- Beads daemon fails to start
- Warning message: "DATABASE MISMATCH DETECTED!"
- Error shows two different repo IDs:
  - Database repo ID: (old ID)
  - Current repo ID: (new ID)
- Risk of incorrect issue deletion during sync

### Common Causes
1. Copied `.beads` directory from another repository
2. Git remote URL changed
3. `bd` tool upgraded with URL canonicalization changes
4. Database corruption

## Solution: Database Migration

### When to Use Migration (vs Reinit)
**Use migration when:**
- Database contains important data (issues, formulas, session notes)
- This is the primary/authoritative clone
- No other active clones need to sync (or they can also be migrated)

**Use reinit when:**
- Database is corrupted beyond repair
- Data can be safely discarded
- Starting fresh is acceptable

### Migration Procedure

#### Step 1: Verify Current State
```bash
cd ~/gt
git remote -v                    # Check remote URL
ls -la .beads/                   # Verify database exists
cat .beads/daemon-error          # Review error details
```

#### Step 2: Execute Migration
```bash
cd ~/gt
echo "y" | bd migrate --update-repo-id
```

**What happens:**
- Prompts for confirmation (auto-answered with "y")
- Updates repo ID in database to match current repository
- Reports old and new repo IDs
- Preserves all existing data

**Expected output:**
```
WARNING: Changing repository ID can break sync if other clones exist.

Current repo ID: 505d71b2
New repo ID:     ed232c4e

Continue? [y/N] ✓ Repository ID updated

  Old: 505d71b2
  New: ed232c4e
```

#### Step 3: Restart Daemon
The daemon may still show old error due to stale process:

```bash
# Kill stale daemon process
pkill -f 'bd.*daemon'

# Wait for process to exit
sleep 1

# Test with any bd command (will auto-start daemon)
bd ready
```

#### Step 4: Verify Success
```bash
# Should show ready work without errors
bd ready

# Should list all issues without warnings
bd list | head -20

# Check daemon status
cat .beads/daemon-error          # Should be empty or show no errors
```

### Expected Results

**Before migration:**
```
Warning: Daemon failed to start:

DATABASE MISMATCH DETECTED!
...
```

**After migration:**
```
📋 Ready work (10 issues with no blockers):
...
```

## Alternative: Reinitialize (Nuclear Option)

**CAUTION:** This destroys all existing beads data!

### When to Use
- Database is corrupted
- Migration failed
- Data loss is acceptable
- Starting completely fresh

### Procedure
```bash
cd ~/gt
rm -rf .beads
bd init --prefix gt
```

**Consequences:**
- All issues deleted
- All formulas removed
- All session notes lost
- Issue history gone
- Fresh start with clean database

## Troubleshooting

### Migration Completes but Daemon Still Errors

**Symptom:** Migration reports success but `bd` commands still show mismatch

**Cause:** Stale daemon process still running with old configuration

**Solution:**
```bash
pkill -f 'bd.*daemon'
sleep 2
bd list  # Will restart daemon with new config
```

### Migration Command Hangs at Prompt

**Symptom:** `bd migrate --update-repo-id` waits for input indefinitely

**Solution:** Provide input via pipe:
```bash
echo "y" | bd migrate --update-repo-id
```

### Multiple Clones Warning

**Warning Message:**
> WARNING: Changing repository ID can break sync if other clones exist.

**Evaluation:**
- **Low Risk:** If `~/gt/` is your only working directory
- **Medium Risk:** If you have other clones on same machine
- **High Risk:** If other team members have clones

**Mitigation:**
- For multiple clones on same machine: Migrate each one
- For team scenarios: Coordinate migration timing
- Document the change in team communication

### Database File Locked

**Symptom:** Migration fails with "database locked" error

**Cause:** Daemon or another process has database open

**Solution:**
```bash
pkill -f 'bd.*daemon'
lsof .beads/beads.db             # Check for other processes
bd migrate --update-repo-id
```

## Technical Details

### What Gets Changed
- **Updated:** Repository ID in database metadata
- **Preserved:** All issues, metadata, formulas, session notes
- **Preserved:** Issue dependencies, labels, status
- **Preserved:** Git history and sync state

### Database Files
- `beads.db` - Main SQLite database (updated)
- `beads.db-shm` - Shared memory file (auto-updated)
- `beads.db-wal` - Write-ahead log (auto-updated)
- Other `.beads/` files - Not modified

### Repo ID Calculation
The repo ID is derived from the git remote URL:
- Git remote URL is canonicalized (normalized)
- Hash is computed to create unique ID
- bd upgrade may change canonicalization logic
- This causes "new" ID for same URL

## Prevention

### Best Practices
1. **Don't copy `.beads` between repos**
   - Each repository should have its own beads database
   - Use `bd init` for each new repo

2. **Document remote URL changes**
   - If changing git remote, plan for migration
   - Communicate with team about timing

3. **Keep bd updated**
   - Update bd on all clones simultaneously when possible
   - Test migration on non-critical clone first

4. **Regular backups**
   ```bash
   # Backup beads database before major changes
   cp -r .beads .beads.backup.$(date +%Y%m%d)
   ```

## Related Commands

### Database Inspection
```bash
# View database metadata
sqlite3 .beads/beads.db "SELECT * FROM metadata;"

# Check repo ID directly
sqlite3 .beads/beads.db "SELECT value FROM metadata WHERE key='repo_id';"

# List all issues
bd list --limit 0
```

### Daemon Management
```bash
# Check daemon status
cat .beads/daemon.lock            # PID and status
cat .beads/daemon.log             # Recent activity
cat .beads/daemon-error           # Current error

# Manual daemon control
bd daemon stop
bd daemon start
bd daemon restart
```

## Recovery Scenarios

### Scenario 1: Migration Failed Halfway
**Symptoms:** Migration started but errored out

**Recovery:**
1. Check database integrity: `sqlite3 .beads/beads.db "PRAGMA integrity_check;"`
2. If corrupt: Restore from backup or reinit
3. If intact: Retry migration

### Scenario 2: Wrong Repo ID Applied
**Symptoms:** Migrated but used wrong repository

**Recovery:**
1. If backup exists: Restore `.beads` from backup
2. If no backup: Re-migrate to correct repo
3. Worst case: Reinit and lose data

### Scenario 3: Multiple Clones Out of Sync
**Symptoms:** Some clones migrated, others not

**Recovery:**
1. Identify canonical/authoritative clone
2. Migrate all clones to same new repo ID
3. Test sync between clones
4. Resolve any conflicts manually

## Success Criteria

✅ **Migration successful when:**
- `bd ready` shows work items without errors
- `bd list` shows all issues
- `.beads/daemon-error` is empty or shows no mismatch
- Daemon starts automatically on bd commands
- No warning messages about repo ID

## References

- **Gastown docs:** `~/gt/GASTOWN-CLAUDE.md`
- **Beads README:** `~/gt/.beads/README.md`
- **Session notes:** `gastown_status_2026-01-28` memory
- **Migration docs:** Official bd documentation (if available)

## Validation Checklist

After migration, verify:
- [ ] Daemon starts without errors
- [ ] `bd list` shows expected issues
- [ ] `bd ready` returns work items
- [ ] Issue details accessible via `bd show <id>`
- [ ] New issues can be created
- [ ] Status updates work
- [ ] Sync with git works (`bd sync`)
- [ ] No error messages in daemon log
- [ ] Database file not corrupted

**Last Tested:** 2026-01-28 on gastown (~/gt/)  
**Result:** ✅ Complete success, all functions operational
