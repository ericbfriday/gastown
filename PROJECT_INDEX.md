# Project Index: Gas Town (gt)

**Generated**: 2026-02-04
**Repository**: github.com/steveyegge/gastown
**Language**: Go 1.24.2
**Total Lines of Code**: ~195,000

---

## 📋 Executive Summary

**Gas Town** is a multi-agent orchestration system for Claude Code with persistent work tracking. It enables coordinating 20-30+ AI coding agents working on different tasks through git-backed hooks, preventing context loss on restart and enabling reliable multi-agent workflows.

**Core Value Proposition**: Turn chaos of 4-10 agents into organized workflows with persistent state, automated coordination, and git-backed work tracking.

---

## 📁 Project Structure

```
gt/
├── cmd/gt/                    # CLI entry point
├── internal/                  # Core implementation (60+ packages)
│   ├── beads/                # Beads integration & work tracking
│   ├── convoy/               # Work distribution system
│   ├── crew/                 # Crew member management
│   ├── daemon/               # Background daemon processes
│   ├── hooks/                # Git hook integration
│   ├── mail/                 # Agent messaging system
│   ├── mayor/                # Mayor AI coordinator
│   ├── polecat/              # Worker agent management
│   ├── rig/                  # Project container system
│   └── [50+ more packages]
├── .beads/                    # Beads issue tracking
│   ├── formulas/             # 32 workflow formulas
│   ├── issues.jsonl          # Issue database
│   └── routes.jsonl          # Routing configuration
├── docs/                      # Comprehensive documentation
├── templates/                 # Agent templates
├── scripts/                   # Utility scripts
├── harness/                   # Test harness
└── [workspaces]/             # Mayor, crew, polecats directories
```

---

## 🚀 Entry Points

### CLI Entry Point
- **Path**: `cmd/gt/main.go`
- **Purpose**: Main CLI application entry point
- **Commands**: 100+ subcommands via Cobra framework

### Key Commands
```bash
gt install <path>          # Initialize workspace
gt mayor attach            # Start Mayor coordinator (PRIMARY INTERFACE)
gt sling <bead-id> <rig>   # Assign work to agent
gt convoy create <name>    # Create work convoy
gt crew add <name>         # Add crew member workspace
gt rig add <name> <repo>   # Add project to workspace
```

---

## 📦 Core Modules

### Agent Management
- **internal/polecat** - Worker agent lifecycle & session management
- **internal/mayor** - AI coordinator implementation
- **internal/crew** - Crew member workspace management
- **internal/agent** - Agent state tracking

### Work Orchestration
- **internal/convoy** - Work distribution & convoy tracking
- **internal/beads** - Beads integration (issue tracking, formulas, molecules)
- **internal/mail** - Inter-agent messaging system
- **internal/mq** - Message queue for agent communication

### Persistence & Git
- **internal/hooks** - Git hook-based persistence
- **internal/git** - Git operations & worktree management
- **internal/rig** - Project container & worktree orchestration
- **internal/workspace** - Workspace initialization & management

### Runtime Integration
- **internal/runtime** - Multi-runtime support (Claude, Codex, Cursor, Gemini)
- **internal/session** - Session management & lifecycle
- **internal/tmux** - Tmux integration for agent sessions

### Daemon & Background
- **internal/daemon** - Background daemon processes
- **internal/deacon** - Heartbeat monitoring & stuck detection
- **internal/dog** - Watchdog for agent health
- **internal/witness** - Audit trail & event logging

### Configuration & State
- **internal/config** - Configuration management
- **internal/state** - Persistent state tracking
- **internal/checkpoint** - Session checkpointing

### Developer Tools
- **internal/doctor** - Health checks & diagnostics
- **internal/feed** - Activity feed generation
- **internal/monitoring** - System monitoring
- **internal/web** - Dashboard web interface

### Formula System
- **internal/formula** - Formula resolution & execution
- **internal/planconvert** - Plan to formula conversion
- **internal/planoracle** - Plan analysis & optimization
- **internal/mergeoracle** - Merge conflict resolution

### UI & Display
- **internal/ui** - Terminal UI components (Bubble Tea)
- **internal/tui** - Text-based UI helpers
- **internal/style** - Styling & colors (Lipgloss)
- **internal/townlog** - Structured logging

### Utilities
- **internal/errors** - Error handling & types
- **internal/events** - Event system
- **internal/filelock** - File locking primitives
- **internal/lock** - Distributed locking
- **internal/util** - Common utilities
- **internal/version** - Version management

---

## 🔧 Configuration Files

### Workspace Configuration
- **settings/config.json** - Global workspace settings
- **.beads/formulas/*.toml** - 32 workflow formulas
- **mayor/rigs.json** - Rig registry

### Runtime Configuration
- **.claude/settings.json** - Claude Code hook configuration
- **~/.codex/config.toml** - Codex CLI settings
- **settings/agents.json** - Agent aliases & commands

### Git Configuration
- **.git/** - Git repository (if initialized with --git)
- **.githooks/** - Custom git hooks
- **.gitignore** - Git ignore patterns

---

## 📚 Documentation

### Getting Started
- **README.md** - Main documentation & quick start
- **docs/INSTALLING.md** - Installation guide
- **docs/overview.md** - System overview
- **docs/glossary.md** - Terminology reference

### Architecture
- **docs/architecture/** - Architecture documentation
- **docs/design/** - Design decisions
- **docs/concepts/** - Core concepts explained

### Integration Guides
- **docs/hooks-integration-summary.md** - Hooks system
- **docs/MAIL_ORCHESTRATOR_SUMMARY.md** - Mail system
- **docs/FILELOCK_INTEGRATION_SUMMARY.md** - File locking
- **docs/ERROR_HANDLING.md** - Error handling guide

### Implementation Details
- **docs/implementation/** - Implementation notes
- **docs/examples/** - Usage examples
- **docs/reference.md** - Complete command reference

### Workflows
- **docs/CLAUDE-HARNESS-WORKFLOW.md** - Harness workflow
- **docs/SESSION-CYCLING-UX.md** - Session management
- **docs/prompt-system.md** - Prompt engineering
- **docs/mayor-cli.md** - Mayor interface guide

---

## 🧪 Test Coverage

### Test Organization
- **Unit tests**: `*_test.go` files (~100+ test files)
- **Integration tests**: `*_integration_test.go` files (~20+ files)
- **E2E tests**: `*_e2e_test.go` files

### Test Helpers
- **internal/cmd/test_helpers_test.go** - Common test utilities
- **testdata/** - Test fixtures & data
- **harness/** - Testing harness for agent workflows

### Key Test Suites
- Agent lifecycle tests (polecat, mayor, crew)
- Convoy & work distribution tests
- Mail system integration tests
- Hooks & persistence tests
- Doctor diagnostics tests
- Formula execution tests

---

## 🔗 Key Dependencies

### Core Framework
- **spf13/cobra** (v1.10.2) - CLI framework
- **BurntSushi/toml** (v1.6.0) - TOML parsing

### Terminal UI
- **charmbracelet/bubbletea** (v1.3.10) - TUI framework
- **charmbracelet/lipgloss** (v1.1.1) - Styling library
- **charmbracelet/glamour** (v0.10.0) - Markdown rendering
- **charmbracelet/bubbles** (v0.21.0) - TUI components

### System Integration
- **gofrs/flock** (v0.13.0) - File locking
- **google/uuid** (v1.6.0) - UUID generation
- **go-rod/rod** (v0.116.2) - Browser automation
- **golang.org/x/sys** (v0.39.0) - System calls
- **golang.org/x/term** (v0.38.0) - Terminal control

---

## 📝 Quick Start

### Installation
```bash
# Install from source
go install github.com/steveyegge/gastown/cmd/gt@latest

# Initialize workspace
gt install ~/gt --git
cd ~/gt

# Add first project
gt rig add myproject https://github.com/you/repo.git

# Create crew workspace
gt crew add yourname --rig myproject

# Start the Mayor (PRIMARY INTERFACE)
gt mayor attach
```

### Basic Workflow
```bash
# In Mayor session
gt convoy create "Feature X" gt-abc12 gt-def34 --notify
gt sling gt-abc12 myproject

# Monitor progress
gt convoy list
gt agents
```

---

## 🏗️ Architectural Patterns

### The Propulsion Principle
Git hooks as propulsion mechanism:
1. Persistent state survives restarts
2. Version control for all changes
3. Rollback capability to any state
4. Multi-agent coordination via git

### MEOW Pattern (Mayor-Enhanced Orchestration Workflow)
1. Tell the Mayor what you want
2. Mayor analyzes & breaks down tasks
3. Mayor creates convoy with beads
4. Mayor spawns appropriate agents
5. Beads slung to agents via hooks
6. Track progress through convoy status
7. Mayor summarizes results

### Work Distribution
```
User → Mayor → Convoy → Beads → Polecats → Hooks → Git
```

### Agent Roles
- **Mayor**: AI coordinator (primary interface)
- **Crew**: Human workspace
- **Polecat**: Worker agent (ephemeral)
- **Hook**: Persistent storage (git worktree)

---

## 🎯 Key Features

### Multi-Agent Orchestration
- Scale to 20-30+ agents comfortably
- Persistent work state in git hooks
- Automated coordination & handoffs
- Built-in mailboxes & identities

### Work Tracking
- Git-backed Beads issue tracking
- Convoy system for work distribution
- Formula-based repeatable workflows
- Real-time progress monitoring

### Runtime Flexibility
- Claude Code (default)
- Codex CLI
- Cursor
- Gemini
- Custom runtimes via config

### Developer Experience
- Web dashboard for monitoring
- Rich terminal UI (Bubble Tea)
- Shell completions (bash/zsh/fish)
- Comprehensive diagnostics (`gt doctor`)

---

## 🔍 Common Operations

### Health Checks
```bash
gt doctor              # Run all diagnostics
gt doctor --fix        # Auto-fix issues
```

### Monitoring
```bash
gt agents              # List active agents
gt convoy list         # Show all convoys
gt feed                # Activity feed
gt dashboard           # Web interface
```

### Maintenance
```bash
gt hooks list          # List hooks
gt hooks repair        # Repair broken hooks
gt convoy refresh      # Refresh convoy state
```

---

## 📊 Repository Statistics

- **Total Go Files**: ~600+
- **Total Lines of Code**: ~195,000
- **Internal Packages**: 60+
- **Test Files**: 100+
- **Formulas**: 32
- **Documentation Files**: 35+
- **Templates**: 4+

---

## 🎓 Learning Path

1. **Start here**: `README.md` - Understand the problem & solution
2. **Core concepts**: `docs/glossary.md` - Learn terminology
3. **Architecture**: `docs/architecture/` - System design
4. **Try it**: `gt install ~/gt --git` - Hands-on experience
5. **Advanced**: `docs/implementation/` - Deep dives

---

## 🚨 Critical Patterns

### Always Use the Mayor
The Mayor is your primary interface - it orchestrates everything else.

### Beads for Work Tracking
Beads (issue IDs) use format: `prefix-abc12` (e.g., `gt-x7k2m`)

### Hooks Provide Persistence
Work survives crashes/restarts via git-backed hooks.

### Convoys Coordinate Work
Bundle related beads into convoys for visibility.

---

## 📞 Support Resources

### Troubleshooting
- Run `gt doctor` for diagnostics
- Check `docs/ERROR_HANDLING.md` for error patterns
- Review `logs/` directory for debug info

### Memory Files
The Serena MCP has 45+ memory files documenting:
- Past sessions & learnings
- Implementation patterns
- Build fixes & solutions
- Ralph loop iterations

---

**Token Savings**: Reading this index (~3,000 tokens) vs. full codebase (~58,000+ tokens) = **94% reduction**

**Last Updated**: 2026-02-04
