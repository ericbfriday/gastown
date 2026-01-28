# Codebase Structure

## Repository Layout

```
~/gt/                              # Gas Town HQ (Town root)
├── .beads/                        # Town-level beads (hq- prefix)
│   ├── formulas/                  # 32+ workflow formulas (TOML)
│   ├── interactions.jsonl         # Town-level activity log
│   ├── issues.jsonl               # Issue database
│   └── config.json                # Beads configuration
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
├── docs/                          # Documentation
│   └── CLAUDE-HARNESS-WORKFLOW.md # Harness implementation spec
├── harness/                       # Claude Automation Harness (NEW)
│   ├── loop.sh                    # Main loop script
│   ├── config.yaml                # Harness configuration
│   ├── README.md                  # Complete harness guide
│   ├── GETTING-STARTED.md         # Quick start
│   ├── ROADMAP.md                 # Implementation phases
│   ├── SUMMARY.md                 # Phase 1 summary
│   ├── state/                     # Runtime state
│   │   ├── queue.json             # Work queue
│   │   ├── iteration.log          # Loop activity log
│   │   └── current-session.json   # Active session (when running)
│   ├── prompts/                   # Agent prompts
│   │   └── bootstrap.md           # Bootstrap prompt template
│   ├── docs/                      # Generated documentation
│   │   ├── research/              # Research findings
│   │   ├── sessions/              # Session contexts & summaries
│   │   └── decisions/             # Decision logs
│   └── scripts/                   # Helper scripts
│       ├── manage-queue.sh        # Work queue manager
│       ├── check-interrupt.sh     # Interrupt detection
│       ├── preserve-context.sh    # Context preservation
│       └── report-status.sh       # Status reporting
├── aardwolf_snd/                  # Rig: Mudlet SND package
│   ├── .repo.git/                 # Shared bare repository
│   ├── .beads/                    # Rig-level beads (as- prefix)
│   ├── config.json                # Rig configuration
│   ├── mayor/rig/                 # Canonical clone (read-only)
│   ├── refinery/rig/              # Main branch worktree (merge queue)
│   │   └── .claude/settings.json  # Refinery hooks
│   ├── witness/                   # Polecat lifecycle manager
│   │   └── .claude/settings.json  # Witness hooks
│   ├── crew/                      # Persistent workers
│   │   └── ericfriday/            # Owner's workspace (git clone)
│   └── polecats/                  # Ephemeral workers (gitignored)
├── duneagent/                     # Rig: Dune MUD agent (similar structure)
├── GASTOWN-CLAUDE.md              # Comprehensive agent guide
├── AGENTS.md                      # Basic agent workflow
├── EBF-NOTES.md                   # Setup notes
└── .gitignore                     # Git ignore rules
```

## Key Directories Explained

### Town-Level (`~/gt/`)

**Purpose**: Global coordination and infrastructure

- `.beads/` - Town-wide issue tracking and workflows
- `mayor/` - Global coordinator configuration
- `deacon/` - Background supervisor
- `settings/` - Town-wide agent settings
- `harness/` - Automation harness (NEW)

### Harness (`~/gt/harness/`)

**Purpose**: Continuous agent spawning and orchestration

**Current Status**: Phase 1 complete (foundation)
- Core loop and infrastructure: ✅
- Helper scripts: ✅
- Documentation: ✅
- Claude Code integration: ⏳ (Phase 2)

**Key Files**:
- `loop.sh` - Main automation loop (423 lines)
- `config.yaml` - Configuration (190 lines)
- `scripts/*.sh` - Helper utilities
- `prompts/bootstrap.md` - Agent bootstrap template

### Rig-Level (`~/gt/<rig>/`)

**Purpose**: Project containers with full git workflow

**Structure**:
- `mayor/rig/` - Canonical read-only clone
- `refinery/rig/` - Merge queue processor
- `witness/` - Polecat lifecycle manager
- `crew/<name>/` - Persistent worker clones
- `polecats/<name>/` - Ephemeral worker worktrees (gitignored)

### Rig: aardwolf_snd

**Technology**: JavaScript/TypeScript, Node.js, Mudlet Lua
**Content**: `mudlet-snd-flattened/` package
**Tests**: `npm test`
**Build**: `npm run build`

### Rig: duneagent

**Technology**: TypeScript monorepo
**Structure**:
- `packages/` - Core packages
- `apps/` - Application code
**Tests**: `npm test`
**Build**: `npm run build`

## Important Files

### Configuration Files

- `harness/config.yaml` - Harness configuration
- `mayor/town.json` - Town metadata
- `mayor/rigs.json` - Rig registry
- `settings/config.json` - Agent config
- `.beads/config.json` - Beads settings

### Documentation Files

- `GASTOWN-CLAUDE.md` - **ESSENTIAL** - Comprehensive agent guide
- `AGENTS.md` - Quick agent reference
- `harness/README.md` - Complete harness guide
- `harness/GETTING-STARTED.md` - Quick start
- `docs/CLAUDE-HARNESS-WORKFLOW.md` - Full implementation spec

### State & Runtime Files

- `harness/state/queue.json` - Work queue
- `harness/state/iteration.log` - Loop activity
- `harness/state/current-session.json` - Active session
- `.beads/interactions.jsonl` - Event log
- `.beads/issues.jsonl` - Issue database

## Gitignored Items

**Never committed**:
- `**/polecats/` - Ephemeral worker worktrees
- `**/crew/` - Persistent worker clones
- `**/mayor/rig/` - Canonical clones
- `**/refinery/rig/` - Merge queue worktrees
- `**/state.json` - Runtime state
- `**/*.lock` - Lock files
- `**/registry.json` - Runtime registries

**Always committed**:
- `.beads/` - Issue tracking data
- Configuration files
- Documentation
- Scripts and source code
- Formulas and workflows

## Navigation Tips

### Finding Work
```bash
# Town-wide
cd ~/gt
bd ready                    # Beads ready work
gt ready                    # Cross-rig work

# Rig-specific
cd ~/gt/<rig>/crew/ericfriday
bd ready
```

### Working Directories

```bash
# Crew member (persistent)
cd ~/gt/<rig>/crew/ericfriday/

# Harness operations
cd ~/gt/harness/

# Rig root (for rig-level operations)
cd ~/gt/<rig>/
```

### Reading Documentation

```bash
# Essential agent guide
cat ~/gt/GASTOWN-CLAUDE.md

# Quick reference
cat ~/gt/AGENTS.md

# Harness guide
cat ~/gt/harness/README.md

# Implementation details
cat ~/gt/docs/CLAUDE-HARNESS-WORKFLOW.md
```

## File Naming Conventions

**Scripts**: `kebab-case.sh`
**Documentation**: `UPPERCASE.md` or `Title-Case.md`
**Configuration**: `config.yaml`, `settings.json`
**State files**: `descriptive-name.json`
**Logs**: `*.log` or `*.jsonl`

## Where Things Live

**Issue Data**: `.beads/issues.jsonl`, `.beads/interactions.jsonl`
**Work Queue**: `harness/state/queue.json`
**Session Logs**: `harness/state/iteration.log`
**Research Notes**: `harness/docs/research/`
**Session Context**: `harness/docs/sessions/`
**Decision Logs**: `harness/docs/decisions/`
**Formulas**: `.beads/formulas/`
**Agent Settings**: `<component>/.claude/settings.json`
