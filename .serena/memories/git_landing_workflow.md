# Git Landing Workflow - Pattern Reference

**Purpose:** Standard git workflow for landing completed work in Gastown projects

## Core Principles

### The Propulsion Principle
> "If you find something on your hook, YOU RUN IT."

Gastown agents execute assigned work immediately and MUST complete the full landing workflow.

### Landing the Plane
Work is NOT complete until:
1. ✅ Quality gates pass (tests)
2. ✅ Changes committed
3. ✅ **Push succeeds** (MANDATORY)
4. ✅ Git status clean and up to date

**CRITICAL:** Never end a session before pushing. Work stranded locally is incomplete work.

## Standard Landing Workflow

### 1. Pre-Commit Checks

```bash
# Verify git status
git status --porcelain

# Run tests (if applicable)
npm test              # Node.js projects
go test ./...         # Go projects
./tests/suite.sh      # Custom test suites

# Check diff statistics
git diff --stat
```

### 2. Stage Changes

```bash
# Stage specific files (preferred - avoids secrets)
git add <specific-files>

# Example: Stage harness changes
git add harness/loop.sh harness/README.md harness/tests/

# AVOID: git add -A or git add . (can include secrets)
```

### 3. Create Commit

**Format:** Use heredoc for proper multi-line formatting

```bash
git commit -m "$(cat <<'EOF'
<type>: <short description>

<detailed description of changes>

<bullet points of key changes>
- Feature 1
- Feature 2
- Documentation updates

<technical details if needed>

Files Modified:
- file1.sh (+100 lines)
- file2.md (updated)

Files Created:
- new-feature.sh
- docs/guide.md

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

**Commit Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `refactor:` - Code refactoring
- `test:` - Test changes
- `chore:` - Maintenance tasks

### 4. Pull with Rebase

```bash
# For feature branches
git pull --rebase origin <branch-name>

# For main branch
git pull --rebase origin main

# Handle new branches (expected failure)
# If error: "fatal: couldn't find remote ref <branch>"
# This is normal for new feature branches - proceed to push
```

### 5. Push Changes

```bash
# For existing branches
git push

# For new branches (first push)
git push -u origin <branch-name>

# Example outputs:
# Success (existing): "Everything up-to-date" or commit list
# Success (new): "[new branch] <branch> -> <branch>"
```

### 6. Verify Final State

```bash
# Must show "up to date with origin"
git status

# Expected output:
# "Your branch is up to date with 'origin/<branch>'"
# Clean working directory (or only expected untracked files)
```

## Special Cases

### New Feature Branch

```bash
# 1. Stage changes
git add <files>

# 2. Commit
git commit -m "feat: description"

# 3. Pull will fail (expected)
git pull --rebase origin feature/new-branch
# Result: "fatal: couldn't find remote ref" - This is EXPECTED

# 4. Push with upstream tracking
git push -u origin feature/new-branch
# Result: "[new branch] feature/new-branch -> feature/new-branch"

# 5. Verify
git status
# Result: "up to date with 'origin/feature/new-branch'"
```

### Handling Conflicts

```bash
# 1. Pull with rebase
git pull --rebase origin main

# 2. If conflicts occur
# Fix conflicts in files
git add <resolved-files>
git rebase --continue

# 3. Push
git push
```

### Push Rejected (Non-Fast-Forward)

```bash
# 1. Pull with rebase
git pull --rebase origin <branch>

# 2. Resolve any conflicts
git add <files>
git rebase --continue

# 3. Force push (only if necessary and safe)
git push --force-with-lease origin <branch>
```

## Quality Gates

### Pre-Commit Gates
- [ ] Tests pass
- [ ] Linting passes (if applicable)
- [ ] No secrets in staged files
- [ ] Commit message follows format

### Pre-Push Gates
- [ ] Rebase successful (or expected failure for new branch)
- [ ] Working directory clean or acceptable state
- [ ] Branch points to correct remote

### Post-Push Gates
- [ ] Git status shows "up to date with origin"
- [ ] Remote shows expected commits
- [ ] Working directory clean

## Gastown-Specific Patterns

### Using gt commit

```bash
# Gastown wrapper for git commit with agent identity
gt commit -m "feat: description"

# This automatically adds agent identity to commit
```

### Beads Sync Before Push

```bash
# MANDATORY before pushing
bd sync

# Then proceed with git workflow
git pull --rebase origin main
git push
```

### Session Completion Workflow

```bash
# 1. Sync beads
bd sync

# 2. Pull latest
git pull --rebase origin main

# 3. PUSH (MANDATORY - work not complete until this succeeds)
git push

# 4. Verify
git status  # Must show "up to date with origin"
```

## Common Errors and Solutions

### Error: "fatal: couldn't find remote ref"
**Cause:** New branch doesn't exist on remote yet  
**Solution:** Use `git push -u origin <branch-name>`  
**Status:** Expected behavior for new branches

### Error: "rejected - non-fast-forward"
**Cause:** Remote has commits you don't have locally  
**Solution:** `git pull --rebase origin <branch>` then push  
**Status:** Requires conflict resolution

### Error: "nothing to commit, working tree clean"
**Cause:** No changes staged or all changes committed  
**Solution:** Verify this is expected state, proceed to push  
**Status:** May be correct if only pushing existing commits

### Error: "remote contains work that you do not have"
**Cause:** Remote branch updated since last pull  
**Solution:** `git pull --rebase origin <branch>` first  
**Status:** Standard sync required

## Checklist Template

```markdown
## Landing Checklist

### Pre-Commit
- [ ] Tests passing
- [ ] No secrets in changes
- [ ] Diff reviewed
- [ ] Commit message prepared

### Commit
- [ ] Changes staged
- [ ] Commit created
- [ ] Message follows format
- [ ] Co-authored attribution included

### Pre-Push
- [ ] Beads synced (if applicable)
- [ ] Rebase attempted/completed
- [ ] Conflicts resolved (if any)
- [ ] Working directory acceptable

### Push
- [ ] Push executed
- [ ] Push succeeded
- [ ] Remote confirmed

### Verify
- [ ] Git status clean
- [ ] Up to date with origin
- [ ] No pending work
```

## Anti-Patterns to Avoid

❌ **Never:**
- End session before pushing
- Say "ready to push when you are" (YOU must push)
- Use `git add -A` or `git add .` (can include secrets)
- Skip rebase attempt
- Ignore push failures
- Leave working directory dirty without reason

✅ **Always:**
- Complete full landing workflow
- Push before ending session
- Verify final state
- Use specific file paths in git add
- Attempt rebase before push
- Resolve push failures

## Session End Rules

**MANDATORY before ending ANY session:**

1. `bd sync` - Sync beads (if using beads)
2. `git pull --rebase origin <branch>` - Get latest
3. `git push` - **MUST SUCCEED**
4. `git status` - Verify "up to date with origin"

**Work is incomplete until push succeeds.**

## Examples from Phase 2 Landing

### Successful New Branch Landing

```bash
# Stage all Phase 2 changes
git add EBF-NOTES.md EBF-QUICKSTART.md harness/

# Commit with comprehensive message
git commit -m "$(cat <<'EOF'
feat: implement Claude automation harness Phase 2 (Claude Code Integration)
...
EOF
)"

# Attempt rebase (expected failure for new branch)
git pull --rebase origin feature/claude-automation-harness
# Result: fatal: couldn't find remote ref (EXPECTED)

# Push with upstream tracking
git push -u origin feature/claude-automation-harness
# Result: [new branch] feature/claude-automation-harness -> feature/claude-automation-harness

# Verify
git status
# Result: Your branch is up to date with 'origin/feature/claude-automation-harness'
```

---

**Reference:** This workflow is based on Gastown session completion requirements and standard git best practices.

**Related Memories:**
- `session_completion_checklist.md` - Session end requirements
- `phase2_landing_complete.md` - Example successful landing
- `project_purpose.md` - Gastown overview
