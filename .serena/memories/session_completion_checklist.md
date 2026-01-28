# Session Completion Checklist

## MANDATORY WORKFLOW - NEVER SKIP

When ending ANY work session, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

## The Checklist

### 1. File Issues for Remaining Work
```bash
# Create issues for anything that needs follow-up
bd create --title "Follow-up: <description>" --type task

# Add to convoy if part of larger effort
gt convoy create "Project Name" <new-issue-id> <existing-ids>
```

### 2. Run Quality Gates (if code changed)

**For Bash/Shell**:
```bash
# Syntax check
bash -n script.sh

# Run shellcheck if available
shellcheck script.sh

# Test execution
./script.sh --dry-run  # If script supports it
```

**For Node.js projects**:
```bash
cd <rig>/mayor/rig/  # Or crew/ericfriday/

npm test
npm run lint
npm run build  # Verify builds
```

**For Go projects**:
```bash
go test ./...
go build ./...
```

### 3. Update Issue Status

```bash
# Close completed work
bd close <completed-id>

# Update in-progress items
bd update <in-progress-id> --notes "Current status: <details>"

# Add comments if needed
bd comments <id> add "Completion notes"
```

### 4. PUSH TO REMOTE (MANDATORY)

```bash
# Sync beads first
bd sync

# Pull latest changes
git pull --rebase origin main

# If conflicts, resolve them
git status
# ... fix conflicts ...
git add <files>
git rebase --continue

# PUSH - This is REQUIRED
git push origin main  # Or current branch

# VERIFY push succeeded
git status  # MUST show "Your branch is up to date with 'origin/main'"
```

### 5. Clean Up

```bash
# Check for stashes (should be empty)
git stash list
# If stashes exist, decide: apply or drop
git stash drop  # If no longer needed

# Check for unmerged branches
git branch --merged
# Prune local branches if safe
git branch -d <branch-name>

# Verify no uncommitted changes
git status  # Should be "working tree clean"
```

### 6. Verify Everything

```bash
# Verify all commits are pushed
git log origin/main..HEAD
# Should output: nothing (all commits pushed)

# Double-check working tree is clean
git status
# Should show: "nothing to commit, working tree clean"

# Verify latest tests pass
npm test  # Or go test ./... or appropriate command
```

### 7. Hand Off (if needed)

```bash
# If context is full or work needs continuation
gt handoff -s "Brief status summary" \
  -m "Detailed context:

Current state: <what's done>
Next steps: <what's needed>
Blockers: <any issues>
Key files: <important files>
Research: <findings>
"

# Or create comprehensive session notes
cat > ~/gt/harness/docs/sessions/$(date +%Y%m%d-%H%M)-handoff.md <<EOF
# Session Handoff - $(date)

## Work Completed
- ...

## Work Remaining
- ...

## Blockers & Issues
- ...

## Key Files Modified
- ...

## Research & Findings
- ...

## Next Steps
1. ...
EOF
```

## CRITICAL RULES

### The Three Musts
1. ✅ Work is NOT complete until `git push` succeeds
2. ✅ NEVER stop before pushing - that strands work locally
3. ✅ NEVER say "ready to push when you are" - YOU must push

### If Push Fails

```bash
# Don't give up! Resolve and retry

# Check what's wrong
git status
git remote -v
git log origin/main..HEAD

# Common fixes:
git pull --rebase origin main  # Get latest first
git push origin main           # Retry

# If still failing, check network/auth
git remote show origin
```

### Post-Push Verification

```bash
# All three should confirm success:

# 1. Git status shows up to date
git status
# Output: "Your branch is up to date with 'origin/main'"

# 2. No local commits ahead of remote
git log origin/main..HEAD
# Output: (empty)

# 3. Remote has your work
git log -1 --oneline
git log origin/main -1 --online
# Should show same commit
```

## For Polecats (Ephemeral Workers)

Polecats have a simplified completion:

```bash
# Work through molecule steps
bd ready
bd close <step-id>

# Implement and test
# ... work ...
git add <files>
gt commit -m "feat: description"
go test ./...  # Or npm test

# Push branch
git push -u origin $(git branch --show-current)
bd sync

# Submit and self-clean (this exits!)
gt done
# Pushes to merge queue, nukes sandbox, exits session
# Refinery handles merge and issue closure - DON'T do it yourself
```

## For Crew (Persistent Workers)

Crew must do full completion:

```bash
# Follow full 7-step checklist above
# Crew pushes directly to main
# Crew closes their own issues
# Crew manages their own cleanup
```

## Common Mistakes to Avoid

❌ **Stopping before push** - Work is stranded
❌ **Saying "ready to push when you are"** - YOU push
❌ **Skipping tests** - Quality gate failures
❌ **Not syncing beads** - Out of sync state
❌ **Leaving working tree dirty** - Uncommitted work
❌ **Polecats closing their own issues** - Refinery does it
❌ **Crew using `gt done`** - That's for polecats only

## Success Indicators

✅ `git status` shows "up to date with origin"
✅ `git log origin/main..HEAD` is empty
✅ Tests pass
✅ Build succeeds
✅ Issues updated
✅ Beads synced
✅ Working tree clean

## Emergency Procedures

If you absolutely must stop without completing:

1. Create interrupt request:
   ```bash
   echo "Emergency stop - work incomplete" > ~/gt/harness/state/interrupt-request.txt
   ```

2. Document current state:
   ```bash
   gt handoff -s "INCOMPLETE - emergency stop" -m "Work incomplete. Details: ..."
   ```

3. Notify overseer:
   ```bash
   gt mail send overseer -s "EMERGENCY: Incomplete session" -m "Details..."
   ```

But this should be EXTREMELY rare. The standard workflow is to complete all steps.
