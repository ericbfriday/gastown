# Gastown Agent Guide for Claude

> **Essential Reading**: This document provides comprehensive guidance for Claude and other AI agents working within the Gastown multi-agent orchestration system.

## Table of Contents

1. [System Overview](#system-overview)
2. [Architecture & Directory Structure](#architecture--directory-structure)
3. [Runtime Environment](#runtime-environment)
4. [Core Concepts](#core-concepts)
5. [Role Taxonomy](#role-taxonomy)
6. [Essential Commands](#essential-commands)
7. [Workflow Patterns](#workflow-patterns)
8. [Working with Beads](#working-with-beads)
9. [Formulas & Molecules](#formulas--molecules)
10. [Identity & Attribution](#identity--attribution)
11. [Cross-Rig Work](#cross-rig-work)
12. [Session Management](#session-management)
13. [Common Mistakes](#common-mistakes)
14. [Troubleshooting](#troubleshooting)

---

## System Overview

**Gastown** is a multi-agent orchestration framework that manages AI agents as structured work units with complete attribution and provenance tracking. The system addresses:

- **Accountability**: Determining which agent caused which outcomes
- **Quality metrics**: Evaluating agent reliability and performance
- **Work routing**: Directing tasks to appropriate agents
- **Scale coordination**: Managing agents across multiple repositories

**Town Location**: `~/gt/`

**Key Binaries**:
- `gt` (Gastown): `/Users/ericfriday/go/bin/gt` (v0.4.0)
- `bd` (Beads): `/Users/ericfriday/.local/bin/bd` (v0.47.1)

---

## Architecture & Directory Structure

```
~/gt/                              # Town root (Gas Town HQ)
├── .beads/                        # Town-level beads (hq- prefix)
│   ├── formulas/                  # 32+ workflow formulas
│   └── interactions.jsonl         # Town-level activity
├── .git/                          # Town git repository
├── mayor/                         # Mayor configuration
│   ├── town.json                  # Town metadata
│   ├── rigs.json                  # Registered rigs
│   ├── overseer.json              # Owner info (Eric Friday)
│   └── .claude/settings.json      # Mayor Claude settings
├── deacon/                        # Background supervisor daemon
│   └── .claude/settings.json      # Deacon Claude settings
├── daemon/                        # Runtime daemon state
├── settings/                      # Town-wide settings
│   ├── config.json                # Agent configuration
│   └── escalation.json            # Escalation rules
├── plugins/                       # Town-level plugins
└── <rig>/                         # Project containers (e.g., aardwolf_snd, duneagent)
    ├── .repo.git/                 # Shared bare repository
    ├── .beads/                    # Rig-level beads
    ├── config.json                # Rig configuration
    ├── mayor/rig/                 # Canonical clone (read-only)
    ├── refinery/rig/              # Main branch worktree (merge queue processor)
    │   └── .claude/settings.json  # Refinery hooks (gt prime, gt mail)
    ├── witness/                   # Polecat lifecycle manager
    │   └── .claude/settings.json  # Witness hooks
    ├── crew/                      # Persistent workers
    │   ├── ericfriday/            # Owner's workspace (git clone)
    │   └── <cross-rig-worktree>/  # Worktrees from other rigs
    └── polecats/                  # Ephemeral workers
        └── <name>/                # Individual polecat (worktree)
```

### Current Rigs

1. **aardwolf_snd**
   - Git: `git@github.com:ericbfriday/mudlet-snd-full.git`
   - Prefix: `as`
   - Branch: `main`

2. **duneagent**
   - Git: `git@github.com:ericbfriday/duneagent.git`
   - Prefix: `du`
   - Branch: `main`

---

## Runtime Environment

### System Configuration

**Platform**: macOS Darwin 25.2.0 (Apple Silicon)

**Shell**: `bash` (with zsh compatibility)

**PATH Priority** (in order):
1. `/opt/homebrew/bin` (Homebrew packages)
2. `/Users/ericfriday/.volta/bin` (Node.js via Volta)
3. `/Users/ericfriday/.local/bin` (Python/uv, bd)
4. `/Users/ericfriday/go/bin` (Go binaries, gt)
5. System paths
6. GNU tools (findutils, coreutils, gnu-sed)

### Python Environment

**System Python**: `3.14.2` (Homebrew)
- Location: `/opt/homebrew/bin/python3`
- Site packages: `/opt/homebrew/lib/python3.14/site-packages`

**Package Manager**: `uv 0.9.11`
- Location: `/Users/ericfriday/.local/bin/uv`
- Managed Pythons: `/Users/ericfriday/.local/share/uv/python/`
- Available versions: 3.14.2 (default), 3.14.0, 3.12.12

**Usage**:
```bash
# Create virtual environment
uv venv

# Install packages
uv pip install <package>

# Run with specific Python
uv run python script.py

# Generate completions (already configured in shell)
uv generate-shell-completion zsh
```

### Go Environment

**Version**: `1.25.6`
- Location: `/opt/homebrew/bin/go`
- GOROOT: `/opt/homebrew/Cellar/go/1.25.6/libexec`
- GOPATH: `/Users/ericfriday/go`
- Go binaries: `/Users/ericfriday/go/bin/` (includes `gt`)

**Usage**:
```bash
# Build Go projects
go build ./...

# Run tests
go test ./...

# Install tools
go install <package>@latest
```

### Node.js Environment (Volta)

**Manager**: Volta.sh
- Location: `/Users/ericfriday/.volta/bin`
- VOLTA_HOME: `$HOME/.volta`

**Installed Runtimes**:
- Node.js: `v20.19.6` (default), `v22.21.1`, `v24.4.1`, `v24.12.0`
- npm: `10.8.2` (built-in)
- Yarn: `4.11.0`

**Usage**:
```bash
# Check current version
volta list

# Install specific version (if needed)
volta install node@20

# Pin version for project (creates package.json entry)
volta pin node@20

# Run with specific version
node --version
npm --version
```

### Git Configuration

**User**:
- Name: Eric Friday
- Email: ericfriday@gmail.com

**Important**: All commits are attributed to the agent identity, not the git user directly.

---

## Core Concepts

### The Propulsion Principle

> **"If you find something on your hook, YOU RUN IT."**

Gastown agents execute assigned work **immediately** without waiting for confirmation. The system functions like a steam engine—agents are pistons propelled by their assignments.

**Do NOT**:
- Ask "should I proceed?"
- Wait for user confirmation to start work
- Delay execution when work is assigned

**DO**:
- Check your hook: `gt hook`
- Read your assignment
- Execute immediately

### Hooks

A **hook** is persistent storage for agent state, surviving crashes and restarts. It's implemented as a git worktree, ensuring durability.

**Check your hook**:
```bash
gt hook
```

Output shows:
- Your pinned molecule (workflow formula)
- Your hook_bead (assigned issue/task)

### Convoys

A **convoy** (🚚) is a batched work tracker for monitoring progress across issues.

```bash
gt convoy create "Feature X" gt-abc gt-def --notify overseer
gt convoy status hq-cv-abc
gt convoy list
```

Convoys provide:
- Single visibility into in-flight work
- Cross-rig tracking
- Auto-notification on completion
- Historical records

### Molecules & Formulas

**Formulas** are workflow templates (TOML files in `.beads/formulas/`).

**Molecules** are instantiated formulas with step beads created for tracking.

**Common formulas**:
- `mol-polecat-work`: Full polecat work lifecycle
- `mol-deacon-patrol`: Deacon background supervision
- `mol-witness-patrol`: Witness polecat monitoring
- `mol-refinery-patrol`: Refinery merge queue processing
- `code-review`: Code review workflow
- `design`: Design workflow

---

## Role Taxonomy

### Infrastructure Roles (Persistent)

| Role | Function | Lifecycle | Location |
|------|----------|-----------|----------|
| **Mayor** | Global coordinator | Singleton, persistent | `~/gt/mayor/` |
| **Deacon** | Background supervisor | Singleton, persistent | `~/gt/deacon/` |
| **Witness** | Per-rig polecat manager | One per rig | `~/gt/<rig>/witness/` |
| **Refinery** | Per-rig merge queue | One per rig | `~/gt/<rig>/refinery/` |

### Worker Roles (Project Work)

| Role | Description | Lifecycle | Location |
|------|-------------|-----------|----------|
| **Polecat** | Ephemeral worker, one task then nuked | Transient, Witness-managed | `~/gt/<rig>/polecats/<name>/` |
| **Crew** | Persistent worker with clone | Long-lived, user-managed | `~/gt/<rig>/crew/<name>/` |
| **Dog** | Deacon infrastructure helper | Ephemeral, Deacon-managed | `~/gt/deacon/dogs/<name>/` |

### Role Comparison: Crew vs Polecats

| Aspect | Crew | Polecat |
|--------|------|---------|
| **Lifecycle** | Persistent (user controls) | Transient (Witness controls) |
| **Monitoring** | None | Witness watches and recycles |
| **Work Assignment** | Human-directed or self-assigned | Assigned via `gt sling` |
| **Git Workflow** | Direct push to main | Branch → Refinery merge queue |
| **Cleanup** | Manual | Automatic on completion |
| **Use Cases** | Exploratory work, long projects | Discrete tasks, batch work |

**Use Crew for**:
- Exploratory work
- Long-running projects
- Tasks requiring judgment
- Direct control scenarios

**Use Polecats for**:
- Discrete well-defined tasks
- Batch work
- Parallelizable projects
- Work benefiting from supervision

### Dogs vs Crew: Critical Distinction

**Dogs are NOT workers**—they're narrow infrastructure utilities owned by Deacon (e.g., Boot handles health triage).

- **Crew**: Do actual project work, human-controlled
- **Dogs**: Execute system-level tasks only, Deacon-controlled

---

## Essential Commands

### Session Initialization

```bash
# Load context (run at session start)
gt prime                    # Load role context
bd prime                    # Load beads context

# Check your hook
gt hook                     # Shows pinned molecule + hook_bead

# Check mail
gt mail inbox               # Read messages
gt mail check --inject      # Auto-injected by hooks
```

### Work Discovery

```bash
# Find available work
bd ready                    # Issues ready to work on
gt ready                    # Work ready across town

# Show issue details
bd show <id>                # Full issue with dependencies

# List rigs
gt rig list                 # All registered rigs
```

### Work Assignment

```bash
# Assign work to an agent (THE unified dispatch command)
gt sling <issue-id> <rig> [--model=<model>]

# Examples
gt sling gt-abc aardwolf_snd
gt sling du-xyz duneagent --model=claude-sonnet

# Remove work from hook
gt unsling <issue-id>
```

### Issue Management (Beads)

```bash
# Claim work
bd update <id> --status in_progress

# Close work
bd close <id>

# Create new issue
bd create --title "Task description" --type feature

# Update issue
bd update <id> --notes "Completion details"

# Sync with git
bd sync
```

### Git Workflow

```bash
# Commit with agent identity
gt commit -m "feat: description"

# Check status
git status

# Push to remote
git push

# Create branch
git checkout -b feature/<name>
```

### Polecat Lifecycle (Self-Cleaning Model)

```bash
# 1. Load context
gt prime && bd prime

# 2. Check hook
gt hook

# 3. Work through molecule steps
bd ready                    # Find next step
bd close <step-id>          # Mark step complete

# 4. Submit and self-clean
gt done                     # Push, submit to MQ, nuke sandbox, exit
```

**After `gt done`**:
- Work is pushed to merge queue
- Sandbox is nuked (worktree removed)
- Session exits immediately
- Refinery handles merge and issue closure

### Cross-Rig Work

```bash
# Create worktree in another rig (preserves identity)
gt worktree <rig>
# Creates ~/gt/<rig>/crew/<your-town>-<your-name>/

# Dispatch work to target rig
bd create --prefix <rig-prefix> "Task description"
gt sling <issue-id> <rig>
```

### Communication

```bash
# Send mail
gt mail send <recipient> -s "Subject" -m "Message"

# Recipients format
gt mail send aardwolf_snd/witness -s "Help needed"
gt mail send duneagent/crew/ericfriday -s "Update"
gt mail send overseer -s "Escalation"

# Broadcast to all workers
gt broadcast "Nudge message"

# Toggle Do Not Disturb
gt dnd
```

### Merge Queue

```bash
# View merge queue
gt mq list

# Mark work ready for merge
gt done                     # From polecat (auto-submits)

# Check merge status
gt mq status
```

### Convoy Management

```bash
# Create convoy
gt convoy create "Project Name" issue1 issue2 --notify overseer

# Check convoy status
gt convoy status <convoy-id>

# List all convoys
gt convoy list
```

### Session Management

```bash
# Hand off to fresh session (context filling)
gt handoff -s "Brief status" -m "Detailed context"

# Resume from handoff
gt resume

# Park work on gate for async resumption
gt park <gate-id>

# Resume parked work
gt resume
```

---

## Workflow Patterns

### 1. Starting a New Session

```bash
# Always start with priming
gt prime                    # Load role context
bd prime                    # Load beads context

# Check for assignments
gt hook                     # Your assigned work
gt mail inbox               # Any messages

# If working as crew member in rig
cd ~/gt/<rig>/crew/ericfriday/
git status                  # Check current state
```

### 2. Polecat Work Cycle (Self-Cleaning)

```bash
# 1. Load context
gt prime && bd prime
gt hook                     # See your assignment

# 2. Read your issue
bd show <issue-id>

# 3. Set up branch
git checkout -b polecat/<name>
git fetch origin
git rebase origin/main

# 4. Work through molecule steps
bd ready                    # Find next step
# ... do the work ...
bd close <step-id>          # Mark complete

# 5. Implement
# ... make changes ...
git add <files>
gt commit -m "feat: description (<issue-id>)"

# 6. Test
go test ./...               # Or appropriate test command

# 7. Clean up
git status                  # Must be clean
git push -u origin $(git branch --show-current)
bd sync

# 8. Submit and exit (self-clean)
gt done                     # Push, MQ submit, nuke sandbox, exit
```

**Critical**: After `gt done`, you are GONE. Don't wait for merge, don't close the issue yourself. Refinery handles everything.

### 3. Crew Work Cycle (Persistent)

```bash
# 1. Start in crew directory
cd ~/gt/<rig>/crew/ericfriday/

# 2. Find or claim work
bd ready
bd update <id> --status in_progress

# 3. Create branch (optional)
git checkout -b feature/<name>

# 4. Implement
# ... make changes ...
git add <files>
gt commit -m "feat: description"

# 5. Test
go test ./...

# 6. Push directly to main
git pull --rebase
git push origin main

# 7. Close issue
bd close <id>
bd sync
```

### 4. Cross-Rig Work (Worktree Pattern)

```bash
# From your home rig (e.g., gastown)
gt worktree aardwolf_snd
# Creates ~/gt/aardwolf_snd/crew/gastown-ericfriday/

# Work preserves your identity
cd ~/gt/aardwolf_snd/crew/gastown-ericfriday/
# BD_ACTOR = gastown/crew/ericfriday

# Work normally
git checkout -b feature/<name>
# ... implement ...
gt commit -m "feat: description"
git push
```

### 5. Handling Context Filling

```bash
# When context is filling up
gt handoff -s "Brief status summary" \
  -m "Detailed context for next session

Current state: <what's done>
Next steps: <what's needed>
Blockers: <any issues>
Files: <key files>"

# Next session (or fresh agent) continues
gt resume                   # Reads handoff message
```

### 6. Landing the Plane (Session Completion)

**MANDATORY WORKFLOW** when ending a session:

```bash
# 1. File issues for remaining work
bd create --title "Follow-up: <description>"

# 2. Run quality gates (if code changed)
go test ./...
# Or: npm test, pytest, etc.

# 3. Update issue status
bd close <completed-id>
bd update <in-progress-id> --notes "Current status"
bd sync

# 4. PUSH TO REMOTE (MANDATORY)
git pull --rebase
git push
git status                  # MUST show "up to date with origin"

# 5. Clean up
git stash list              # Should be empty
git branch --merged         # Prune if needed

# 6. Verify
git status                  # Clean working tree
git log origin/main..HEAD   # Should be empty (all pushed)

# 7. Hand off (if needed)
# Provide context for next session
```

**CRITICAL RULES**:
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing (strands work locally)
- NEVER say "ready to push when you are" (YOU must push)
- If push fails, resolve and retry until success

---

## Working with Beads

Beads (`bd`) is the lightweight issue tracker integrated with Gastown.

### Issue Structure

Each issue has:
- **ID**: `<prefix>-<number>` (e.g., `gt-123`, `as-456`)
- **Type**: feature, bug, task, epic
- **Status**: pending, in_progress, completed, blocked
- **Dependencies**: blocks/blocked_by relationships
- **Labels**: Tags for organization
- **State**: Operational state dimensions
- **Comments**: Discussion thread

### Common Operations

```bash
# Create issue
bd create --title "Task description" --type feature --priority 1

# Quick capture (returns only ID)
bd q "Task description"

# Show issue with full dependency graph
bd show <id>

# Update status
bd update <id> --status in_progress
bd update <id> --notes "Progress update"

# Close issue
bd close <id>

# Reopen issue
bd reopen <id>

# Add comment
bd comments <id> add "Comment text"

# View comments
bd comments <id> list

# Search issues
bd search "keyword"

# List issues with filters
bd list --status pending --type bug

# Dependency management
bd dep add <issue-id> blocks <other-id>
bd dep add <issue-id> blocked-by <other-id>

# Sync with git
bd sync                     # Always do this before push
```

### Beads Best Practices

1. **Always sync before push**:
   ```bash
   bd sync
   git push
   ```

2. **Close issues when done** (crew only):
   ```bash
   bd close <id>
   ```

3. **Don't close issues as polecat** (Refinery does this after merge)

4. **Use dependencies** to track blockers:
   ```bash
   bd dep add gt-123 blocked-by gt-122
   ```

5. **File discovered work**:
   ```bash
   bd create --title "Found: bug in auth" --type bug
   ```

---

## Formulas & Molecules

### What are Formulas?

**Formulas** are workflow templates stored in `.beads/formulas/`. They define:
- Step-by-step workflow
- Entry/exit criteria
- Commands to run
- Dependencies between steps
- Variables and their sources

### What are Molecules?

**Molecules** are instantiated formulas with:
- Step beads created for tracking
- Variables populated from context
- Progress state maintained

### Common Formulas

**Infrastructure**:
- `mol-deacon-patrol`: Deacon background supervision
- `mol-witness-patrol`: Witness polecat monitoring
- `mol-refinery-patrol`: Refinery merge queue processing
- `mol-boot-triage`: Boot health triage

**Polecat Work**:
- `mol-polecat-work`: Full work lifecycle (load → implement → test → submit)
- `mol-polecat-code-review`: Code review workflow
- `mol-polecat-conflict-resolve`: Conflict resolution
- `mol-polecat-review-pr`: Pull request review

**Utility**:
- `code-review`: General code review
- `design`: Design workflow
- `security-audit`: Security audit workflow

### Working with Molecules

```bash
# Check your pinned molecule
gt hook

# Find molecule steps
bd ready                    # Shows next available step

# Complete a step
bd close <step-id>

# View molecule structure
bd show <molecule-id>
```

### Molecule Step Pattern

Each molecule has steps with:

1. **Load context**
   - Prime environment
   - Read assignment
   - Check for blockers

2. **Branch setup**
   - Create/verify feature branch
   - Sync with main

3. **Preflight tests**
   - Verify main is healthy
   - File issues if not

4. **Implement**
   - Do the work
   - Commit frequently

5. **Self-review**
   - Review changes
   - Check for issues

6. **Run tests**
   - Full test suite
   - Verify coverage

7. **Cleanup workspace**
   - Clean working tree
   - Push branch

8. **Prepare for review**
   - Update issue
   - Sync beads

9. **Submit and exit**
   - Push to merge queue
   - Self-clean (polecats only)

---

## Identity & Attribution

All work is attributed to the performer:

```
Git commits:
  Author: gastown/crew/joe <owner@example.com>

Beads issues:
  created_by: gastown/crew/joe

Events:
  actor: gastown/crew/joe
```

### Identity Format

**Town/Role/Name**:
- `gastown/crew/ericfriday`
- `aardwolf_snd/polecats/Toast`
- `duneagent/witness`

### Identity Persistence

Identity persists across rigs:
- Work on `gastown/crew/joe` appears on Joe's resume regardless of repository
- Worktrees preserve identity from origin rig
- Metrics are aggregated by full identity path

### Checking Your Identity

```bash
# Your identity is in environment
echo $BD_ACTOR

# Or inferred from directory structure
pwd
# ~/gt/<rig>/<role>/<name>/ → <town>/<role>/<name>
```

---

## Cross-Rig Work

### Decision Matrix

**Use Worktree When**:
- Need quick fixes
- Want work on your CV
- Working from your context
- Preserving your identity matters

**Use Dispatch When**:
- Target rig should own work
- Creating native issues
- Delegating to rig's workers
- System/infrastructure tasks

### Worktree Pattern (Preferred)

```bash
# Create worktree in target rig
gt worktree <rig>

# Example
gt worktree aardwolf_snd
# Creates ~/gt/aardwolf_snd/crew/gastown-ericfriday/

# Work preserves your identity
cd ~/gt/aardwolf_snd/crew/gastown-ericfriday/
# BD_ACTOR = gastown/crew/ericfriday

# Work normally
git checkout -b feature/<name>
# ... implement ...
gt commit -m "feat: description"
git push
```

### Dispatch Pattern

```bash
# Create issue in target rig
bd create --prefix as "Fix authentication bug"
# Creates as-123

# Add to convoy (optional)
gt convoy create "Auth fix" as-123

# Assign to target rig
gt sling as-123 aardwolf_snd
```

---

## Session Management

### Priming

**Always** start sessions with:

```bash
gt prime                    # Load role context
bd prime                    # Load beads context
```

This loads:
- Role-specific context
- Rig configuration
- Active molecules
- Mail messages (via hooks)

### Context Filling

When context fills up:

```bash
gt handoff -s "Brief status" \
  -m "Full context for next session

Current: <state>
Next: <steps>
Blockers: <issues>
Files: <list>"
```

Next session resumes:

```bash
gt resume                   # Reads handoff
```

### Hooks (Claude Settings)

Refinery, Witness, and infrastructure roles have Claude hooks configured:

**SessionStart**:
```bash
gt prime && gt mail check --inject
```

**PreCompact**:
```bash
gt prime
```

**UserPromptSubmit**:
```bash
gt mail check --inject
```

These ensure context is loaded and mail is checked automatically.

---

## Common Mistakes

1. **Using dogs for user work**
   - Dogs are infrastructure-only
   - Use crew or polecats for project work

2. **Confusing crew with polecats**
   - Different lifecycles
   - Crew: persistent, self-managed
   - Polecats: ephemeral, Witness-managed

3. **Working in wrong directories**
   - Affects identity detection
   - Always check: `pwd` and `git status`

4. **Waiting for confirmation when work is assigned**
   - Violates Propulsion Principle
   - Execute immediately when work is on your hook

5. **Creating worktrees when dispatch is appropriate**
   - Consider ownership
   - Use dispatch for work native to target rig

6. **Closing issues as polecat**
   - Refinery closes after merge
   - Only crew should close their own issues

7. **Not pushing before session end**
   - Work is NOT complete until `git push` succeeds
   - NEVER say "ready to push when you are"

8. **Ignoring pre-existing test failures**
   - Run preflight tests on main
   - File issues or fix before proceeding

9. **Scope creep**
   - Keep changes focused
   - File new issues for discovered work

10. **Not syncing beads before push**
    ```bash
    bd sync                 # ALWAYS before push
    git push
    ```

---

## Troubleshooting

### "gt command not found"

```bash
# Check if gt is in PATH
which gt
# Should be: /Users/ericfriday/go/bin/gt

# If not, check Go bin is in PATH
echo $PATH | grep go/bin

# Add to PATH if missing (should be in shell config)
export PATH=$PATH:/Users/ericfriday/go/bin
```

### "bd command not found"

```bash
# Check if bd is in PATH
which bd
# Should be: /Users/ericfriday/.local/bin/bd

# If not, check .local/bin is in PATH
echo $PATH | grep .local/bin

# Add to PATH if missing
export PATH=$HOME/.local/bin:$PATH
```

### Python Version Issues

```bash
# Check active Python
python3 --version
# Should be: Python 3.14.2

# Use uv to manage Python versions
uv python list              # See available versions
uv venv                     # Create venv with default Python
uv venv --python 3.12       # Create venv with specific version
```

### Node Version Issues

```bash
# Check active Node
node --version
# Should match volta default: v20.19.6

# List installed versions
volta list

# Switch version (creates package.json pin)
volta pin node@20

# Or use specific version
volta install node@22
```

### Git Identity Issues

```bash
# Check git config
git config user.name
git config user.email

# Should be:
# Eric Friday
# ericfriday@gmail.com

# Set if missing
git config --global user.name "Eric Friday"
git config --global user.email "ericfriday@gmail.com"
```

### "Hook is empty" / No work assigned

```bash
# Check your hook
gt hook
# If empty, no work is assigned

# Find available work
bd ready
gt ready

# Claim work (as crew)
bd update <id> --status in_progress

# Or wait for assignment (as polecat)
# Witness assigns work via gt sling
```

### Tests Failing

```bash
# Check if failure is pre-existing
git stash
git checkout origin/main
go test ./...               # Or appropriate test command
git checkout -
git stash pop

# If pre-existing:
bd create --title "Pre-existing test failure: <description>" --type bug
gt mail send witness -s "NOTICE: Main has failing tests"

# If your change caused it:
# Fix the issue, don't proceed with failures
```

### Context Filling Up

```bash
# Hand off to fresh session
gt handoff -s "Brief status" \
  -m "Full context:

Issue: <id>
Current: <what's done>
Next: <what's needed>
Files: <key files>
Blockers: <any issues>"

# Next session resumes
gt resume
```

### Push Rejected

```bash
# Pull and rebase
git pull --rebase origin main

# If conflicts, resolve them
git status                  # See conflicted files
# ... fix conflicts ...
git add <files>
git rebase --continue

# Retry push
git push origin main
```

### Beads Sync Conflicts

```bash
# Usually happens with concurrent edits
bd sync

# If conflicts in .beads/interactions.jsonl
git status
# Resolve manually (JSONL is append-only, can concatenate)
git add .beads/
git commit -m "chore: resolve beads sync conflict"
git push
```

### Permission Denied (SSH)

```bash
# Check SSH keys
ssh -T git@github.com
# Should authenticate as ericbfriday

# If not, check SSH config
cat ~/.ssh/config

# Test connection
ssh -vT git@github.com
```

---

## Quick Reference Card

### Session Start
```bash
gt prime && bd prime         # Load context
gt hook                      # Check assignment
gt mail inbox                # Read messages
```

### Work Discovery
```bash
bd ready                     # Available work
gt ready                     # Work across town
bd show <id>                 # Issue details
```

### Git Workflow
```bash
git checkout -b feature/<name>
# ... implement ...
git add <files>
gt commit -m "feat: description"
go test ./...                # Or npm test, pytest, etc.
git push
```

### Issue Management
```bash
bd update <id> --status in_progress
bd close <id>
bd sync
```

### Session End (MANDATORY)
```bash
bd sync                      # Sync beads
git pull --rebase            # Get latest
git push                     # MUST succeed
git status                   # Verify clean
```

### Cross-Rig Work
```bash
gt worktree <rig>            # Create worktree
cd ~/gt/<rig>/crew/<town>-<name>/
```

### Communication
```bash
gt mail send <recipient> -s "Subject" -m "Message"
gt broadcast "Nudge to all workers"
```

### Polecat Self-Clean
```bash
gt done                      # Submit, nuke, exit
```

---

## Additional Resources

### Documentation
- GitHub: https://github.com/ericbfriday/gastown
- Overview: https://github.com/ericbfriday/gastown/blob/main/docs/overview.md

### Local Files
- `~/gt/AGENTS.md`: Basic agent workflow
- `~/gt/mayor/CLAUDE.md`: Mayor-specific context
- `~/gt/EBF-NOTES.md`: Setup notes

### Formulas Location
- Town: `~/gt/.beads/formulas/`
- Rig: `~/gt/<rig>/.beads/formulas/`

### Getting Help

**From Witness** (rig issues):
```bash
gt mail send <rig>/witness -s "HELP: <topic>" -m "Details"
```

**From Deacon** (town issues):
```bash
gt mail send deacon -s "HELP: <topic>" -m "Details"
```

**From Mayor** (coordination):
```bash
gt mail send mayor -s "Question: <topic>" -m "Details"
```

**To Overseer** (escalation):
```bash
gt mail send overseer -s "ESCALATION: <issue>" -m "Details"
```

---

## System Quirks & Notes

### macOS Specific

- GNU tools are available with `g` prefix: `gfind`, `gsed`, `gawk`
- Or use unprefixed versions from Homebrew paths (higher priority in PATH)
- Ghostty terminal available at `/Applications/Ghostty.app/Contents/MacOS`

### Shell Configuration

- Primary shell is bash (`.bash_profile`)
- zsh completions configured (`.zshrc`)
- uv and goose have shell completions enabled

### Custom Tools

- `.opencode/bin`: OpenCode antigravity auth
- `.antigravity/antigravity/bin`: Antigravity tools
- `.lmstudio/bin`: LM Studio binaries

### Homebrew Notes

- Location: `/opt/homebrew` (Apple Silicon)
- PostgreSQL client: `/opt/homebrew/opt/libpq/bin`

---

**Version**: 1.0
**Created**: 2026-01-27
**Author**: Claude (analyzing system for Eric Friday)
**Town**: gt (~/gt/)
**Overseer**: Eric Friday <ericfriday@gmail.com>
