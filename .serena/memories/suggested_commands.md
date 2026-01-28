# Suggested Commands Reference

## Session Initialization (ALWAYS RUN AT START)

```bash
# Load role and beads context - MANDATORY at session start
gt prime && bd prime

# Check your hook (assigned work)
gt hook

# Check mail
gt mail inbox
```

## Work Discovery & Management

```bash
# Find available work
bd ready              # Beads issues ready to work
gt ready              # Cross-rig work

# Show issue details
bd show <id>

# Claim work
bd update <id> --status in_progress

# Close completed work
bd close <id>

# Sync beads with git (ALWAYS before push)
bd sync
```

## Git Workflow

```bash
# Check status
git status

# Commit with agent identity (use gt, not git directly)
gt commit -m "feat: description (issue-id)"

# Pull and rebase
git pull --rebase origin main

# Push to remote (MANDATORY before ending session)
git push

# Verify clean state
git log origin/main..HEAD   # Should be empty (all pushed)
```

## Testing & Quality Gates

### For Bash/Shell Projects
```bash
# Run shell checks (if shellcheck installed)
shellcheck script.sh

# Test script execution
bash -n script.sh   # Syntax check

# Run integration tests (if present)
./test/run-tests.sh
```

### For Node.js Projects (duneagent, aardwolf_snd rigs)
```bash
cd <rig>/mayor/rig/  # Or crew/ericfriday/

# Install dependencies
npm install

# Run tests
npm test

# Run linter
npm run lint

# Build
npm run build
```

### For Go Projects (gt tool itself - if working on it)
```bash
# Run tests
go test ./...

# Build
go build ./...

# Run with race detector
go test -race ./...
```

## Harness Operations

```bash
cd ~/gt/harness

# Start harness (continuous loop)
./loop.sh

# Start with iteration limit (for testing)
MAX_ITERATIONS=5 ./loop.sh

# Check status
./scripts/report-status.sh

# Watch status (in separate terminal)
watch -n 5 ./scripts/report-status.sh

# Stop harness
pkill -f "loop.sh"
# Or Ctrl+C if in foreground

# Request interrupt (from agent or manually)
echo "Reason for interrupt" > state/interrupt-request.txt

# Resume from interrupt
rm state/interrupt-request.txt

# Check queue
./scripts/manage-queue.sh show

# Refresh queue
./scripts/manage-queue.sh check
```

## Communication & Coordination

```bash
# Send mail
gt mail send <recipient> -s "Subject" -m "Message"

# Examples
gt mail send aardwolf_snd/witness -s "Help needed"
gt mail send overseer -s "Escalation"

# Broadcast to all workers
gt broadcast "Message"

# Toggle Do Not Disturb
gt dnd
```

## Cross-Rig Work

```bash
# Create worktree in another rig (preserves identity)
gt worktree <rig>

# Example: work on aardwolf_snd from gastown context
gt worktree aardwolf_snd
cd ~/gt/aardwolf_snd/crew/<your-identity>/

# Dispatch work to another rig
bd create --prefix <rig-prefix> "Task description"
gt sling <issue-id> <rig>
```

## Knowledge & Memory Management

```bash
# Write memory (via Serena - if available)
# This is typically done through specialized tools
# See GASTOWN-CLAUDE.md for details

# Read research notes
cat ~/gt/harness/docs/research/*.md

# View session summaries
ls -lt ~/gt/harness/docs/sessions/
cat ~/gt/harness/docs/sessions/<session>-summary.md
```

## Utility Commands (macOS Darwin)

```bash
# File operations
ls -la                # List with details
find . -name "*.sh"   # Find files
grep -r "pattern" .   # Search content

# JSON processing
jq . file.json        # Pretty-print
jq '.field' file.json # Extract field

# Directory navigation
pwd                   # Current directory
cd <path>             # Change directory

# Environment
echo $PATH            # Check PATH
which <command>       # Find command location
env                   # Show all environment variables
```

## Session Completion (MANDATORY BEFORE ENDING)

```bash
# 1. Sync beads
bd sync

# 2. Pull latest
git pull --rebase origin main

# 3. PUSH (MANDATORY - work not complete until this succeeds)
git push

# 4. Verify
git status                    # Should show "up to date with origin"
git log origin/main..HEAD     # Should be empty

# 5. Check for stashes (should be empty)
git stash list

# 6. Close or update issues
bd close <completed-id>
bd update <in-progress-id> --notes "Status update"
```

## Troubleshooting

```bash
# Check required commands
which gt bd jq git

# Verify versions
gt --version   # Should be v0.4.0
bd --version   # Should be v0.47.1

# Check git identity
git config user.name   # Should be: Eric Friday
git config user.email  # Should be: ericfriday@gmail.com

# Fix push rejection
git pull --rebase origin main
# Resolve conflicts if any
git push

# Check for errors in harness
tail ~/gt/harness/state/iteration.log | grep ERROR

# View recent sessions
ls -lt ~/gt/harness/docs/sessions/
```

## Quick Reference Card

**Start Session**: `gt prime && bd prime && gt hook`
**Find Work**: `bd ready` or `gt ready`
**Run Tests**: `npm test` (Node) or `go test ./...` (Go) or custom test script
**Commit**: `gt commit -m "type: message"`
**Complete**: `bd sync && git pull --rebase && git push && git status`
