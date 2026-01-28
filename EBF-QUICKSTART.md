# Gas Town Quick Start Guide

**Your AI Agent Orchestration System**

Gas Town coordinates Claude agents to work on tasks across multiple repositories with complete attribution and provenance tracking.

---

## Quick Start: Using Claude with Gastown

### 1. Essential Reading First

**ALWAYS read this first when starting a session:**
```bash
cat ~/gt/GASTOWN-CLAUDE.md
```

This is your comprehensive agent guide with everything you need.

### 2. Session Initialization (MANDATORY)

Every time you start working, run:
```bash
# Load role context and beads
gt prime && bd prime

# Check your hook (assigned work)
gt hook

# Check mail
gt mail inbox
```

### 3. Find Available Work

```bash
# From beads (local rig issues)
bd ready

# From Gastown (cross-rig work)
gt ready

# Show specific issue details
bd show <issue-id>
```

### 4. Claim and Work on an Issue

```bash
# Claim work
bd update <issue-id> --status in_progress

# View issue details
bd show <issue-id>

# Make your changes
# ... code changes ...

# Commit with agent identity
gt commit -m "feat: description (<issue-id>)"

# Test your changes
npm test        # For Node.js rigs (duneagent, aardwolf_snd)
go test ./...   # For Go projects

# Complete work
bd close <issue-id>
bd sync
git push
```

### 5. Using the Automation Harness (NEW)

The harness can spawn Claude agents automatically:

```bash
# Navigate to harness
cd ~/gt/harness

# Check harness status
./scripts/report-status.sh

# Start harness (spawns agents for available work)
./loop.sh

# Monitor active session (in another terminal)
./scripts/parse-session-events.sh watch <session-id>

# Run tests
./tests/integration-suite.sh
```

---

## Common Workflows

### Creating New Work

```bash
# Create new issue
bd create --title "Task description" --type feature --priority 1

# Quick capture (returns only ID)
issue_id=$(bd q "Quick task description")

# Assign to harness
gt sling $issue_id <rig-name>
```

### Cross-Rig Work

```bash
# Create worktree in another rig (preserves your identity)
gt worktree aardwolf_snd
cd ~/gt/aardwolf_snd/crew/<your-identity>/

# Or dispatch work to target rig
bd create --prefix as "Fix in aardwolf_snd"
gt sling as-123 aardwolf_snd
```

### Handling Interrupts

```bash
# When you need human attention (from agent or manually)
echo "Need code review" > ~/gt/harness/state/interrupt-request.txt

# Harness pauses and preserves context

# When ready to resume
rm ~/gt/harness/state/interrupt-request.txt
```

### Session Completion (MANDATORY)

**Before ending ANY session, you MUST:**

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

---

## Essential Commands Reference

### Gastown Commands

```bash
# Context loading
gt prime                          # Load role context
bd prime                          # Load beads context

# Work discovery
gt hook                           # Check assigned work
gt ready                          # Cross-rig ready work
bd ready                          # Local ready work
gt mail inbox                     # Read messages

# Work assignment
gt sling <issue-id> <rig>         # Assign work to rig
gt unsling <issue-id>             # Remove from hook

# Communication
gt mail send <recipient> -s "Subject" -m "Message"
gt broadcast "Message to all"
gt dnd                            # Toggle do not disturb

# Rig operations
gt rig list                       # List rigs
gt worktree <rig>                 # Create worktree in rig
```

### Beads Commands

```bash
# Issue management
bd show <id>                      # View issue
bd create --title "..." --type feature
bd update <id> --status in_progress
bd close <id>                     # Complete issue
bd reopen <id>                    # Reopen issue
bd sync                           # Sync with git (ALWAYS before push)

# Dependencies
bd dep add <id> blocks <other-id>
bd dep add <id> blocked-by <other-id>

# Comments
bd comments <id> add "Comment"
bd comments <id> list

# Search & list
bd search "keyword"
bd list --status pending --type bug
```

### Git Workflow

```bash
# Create branch
git checkout -b feature/<name>

# Commit with agent identity (use gt, not git directly)
gt commit -m "feat: description"

# Push
git pull --rebase origin main
git push
```

### Harness Commands (Phase 2)

```bash
cd ~/gt/harness

# Run harness
./loop.sh                                    # Continuous
MAX_ITERATIONS=5 ./loop.sh                   # Limited run

# Monitor
./scripts/report-status.sh                   # Current status
watch -n 5 ./scripts/report-status.sh        # Live updates

# Session analysis
./scripts/parse-session-events.sh watch <session-id>
./scripts/parse-session-events.sh summary <session-id>
./scripts/parse-session-events.sh metrics <session-id>

# Work queue
./scripts/manage-queue.sh check
./scripts/manage-queue.sh show

# Testing
./tests/integration-suite.sh                 # All tests
./tests/integration-suite.sh --parallel      # Fast
```

---

## Directory Structure Quick Reference

```
~/gt/                              # Gas Town HQ
├── GASTOWN-CLAUDE.md              # ⭐ READ THIS FIRST
├── AGENTS.md                      # Agent workflow basics
├── EBF-QUICKSTART.md              # This file
├── harness/                       # Automation harness (Phase 2 complete)
│   ├── loop.sh                    # Main harness loop
│   ├── config.yaml                # Configuration
│   ├── README.md                  # Harness guide
│   ├── scripts/                   # Helper scripts
│   ├── tests/                     # Integration tests
│   └── docs/                      # Documentation
├── aardwolf_snd/                  # Rig: Mudlet SND
│   ├── mayor/rig/                 # Canonical clone
│   ├── crew/ericfriday/           # Your workspace
│   └── refinery/rig/              # Merge queue
├── duneagent/                     # Rig: Dune agent
│   └── (same structure)
├── mayor/                         # Global coordinator
├── deacon/                        # Background supervisor
└── settings/                      # Town settings
```

---

## Rig-Specific Commands

### For aardwolf_snd (Node.js/Mudlet)

```bash
cd ~/gt/aardwolf_snd/crew/ericfriday/

# Install dependencies
npm install

# Run tests
npm test

# Build
npm run build

# Lint
npm run lint
```

### For duneagent (TypeScript monorepo)

```bash
cd ~/gt/duneagent/crew/ericfriday/

# Install dependencies
npm install

# Run tests
npm test

# Build
npm run build
```

---

## Troubleshooting Quick Fixes

### "gt command not found"
```bash
# Add to PATH if needed
export PATH=$PATH:/Users/ericfriday/go/bin
```

### "bd command not found"
```bash
# Add to PATH if needed
export PATH=$HOME/.local/bin:$PATH
```

### Push Rejected
```bash
git pull --rebase origin main
# Fix conflicts if any
git add <files>
git rebase --continue
git push origin main
```

### Tests Failing
```bash
# Check if pre-existing
git stash
git checkout origin/main
npm test  # or go test ./...
git checkout -
git stash pop

# If pre-existing, file issue
bd create --title "Pre-existing test failure: <description>" --type bug
```

### Harness Not Working
```bash
cd ~/gt/harness

# Check dependencies
which gt bd jq git claude

# Check logs
tail -f state/iteration.log

# Verify configuration
cat config.yaml | grep -v "^#"

# Run tests
./tests/integration-suite.sh --quick
```

---

## Advanced: Claude Orchestration Patterns

### Pattern 1: Batch Work Processing

```bash
# Create multiple issues
for task in task1 task2 task3; do
    bd create --title "$task" --type feature
done

# Start harness to process them
cd ~/gt/harness
./loop.sh
```

### Pattern 2: Convoy Tracking

```bash
# Create convoy for batch work
gt convoy create "Feature X" gt-abc gt-def gt-ghi --notify overseer

# Check convoy status
gt convoy status <convoy-id>

# List all convoys
gt convoy list
```

### Pattern 3: Cross-Rig Coordination

```bash
# Create work in target rig
bd create --prefix du "Task for duneagent"

# Assign to rig
gt sling du-123 duneagent

# Monitor from here
gt convoy create "Cross-rig work" du-123
```

### Pattern 4: Manual Agent Spawning

```bash
# For crew work (persistent)
cd ~/gt/<rig>/crew/ericfriday/

# Prime context
gt prime && bd prime

# Find work
bd ready

# Claim and work
bd update <id> --status in_progress
# ... do work ...
bd close <id>
bd sync
git push
```

---

## Key Principles

### The Propulsion Principle
> **"If you find something on your hook, YOU RUN IT."**

Gastown agents execute assigned work **immediately** without waiting for confirmation.

### Landing the Plane
**NEVER** end a session without:
1. ✅ Filing issues for remaining work
2. ✅ Running quality gates (tests)
3. ✅ Updating issue status
4. ✅ **PUSHING TO REMOTE** (MANDATORY)
5. ✅ Verifying clean git state

### Session Completion Rules
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that strands work locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until success

---

## Quick Links

**Documentation:**
- Full Agent Guide: `~/gt/GASTOWN-CLAUDE.md`
- Harness README: `~/gt/harness/README.md`
- Harness Phase 2 Summary: `~/gt/harness/docs/PHASE-2-SUMMARY.md`
- Production Rollout: `~/gt/harness/docs/PRODUCTION-ROLLOUT.md`

**Serena Memories** (if using Serena MCP):
- `phase2_implementation_complete` - Phase 2 session summary
- `harness_commands_reference` - Harness operations guide
- `project_purpose` - Gas Town overview
- `suggested_commands` - Essential commands

**Getting Help:**
```bash
# From witness (rig issues)
gt mail send <rig>/witness -s "HELP: topic"

# From deacon (town issues)
gt mail send deacon -s "HELP: topic"

# Escalate to overseer
gt mail send overseer -s "ESCALATION: issue"
```

---

## Initial Setup (For Reference)

This was the output when Gas Town HQ was created:

```
🏭 Creating Gas Town HQ at /Users/ericfriday/gt

   ✓ Created mayor/
   ✓ Created mayor/town.json
   ✓ Created mayor/rigs.json
   ✓ Created mayor/.claude/settings.json
   ✓ Created deacon/.claude/settings.json
   ✓ Created .gitignore
   ✓ Initialized git repository
   ✓ Initialized .beads/ (town-level beads with hq- prefix)
   ✓ Provisioned 32 formulas
   ✓ Created role beads (7 roles)
   ✓ Created agent beads (mayor, deacon)
   ✓ Detected overseer: Eric Friday <ericfriday@gmail.com>
   ✓ Created settings/escalation.json
   ✓ Created .claude/commands/ (slash commands for all agents)

✓ HQ created successfully!
```

**Registered Rigs:**
1. **aardwolf_snd** - `git@github.com:ericbfriday/mudlet-snd-full.git`
2. **duneagent** - `git@github.com:ericbfriday/duneagent.git`

---

**Quick Start Checklist:**

- [ ] Read `GASTOWN-CLAUDE.md`
- [ ] Run `gt prime && bd prime` at session start
- [ ] Check `gt hook` and `gt mail inbox`
- [ ] Find work with `bd ready` or `gt ready`
- [ ] Complete work and **push before ending**
- [ ] Use harness for automated agent spawning

**Remember:** Always push your work! `git push` is MANDATORY before session end.
