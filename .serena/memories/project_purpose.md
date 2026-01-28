# Project Purpose

**Gas Town (gt)** is a multi-agent orchestration framework that manages AI agents as structured work units with complete attribution and provenance tracking.

## Core Mission

The system addresses:
- **Accountability**: Determining which agent caused which outcomes
- **Quality metrics**: Evaluating agent reliability and performance
- **Work routing**: Directing tasks to appropriate agents
- **Scale coordination**: Managing agents across multiple repositories

## Key Components

### 1. **Gastown Framework** (`gt` command)
Multi-agent orchestration system that provides:
- Rig management (project containers)
- Agent lifecycle management
- Work assignment and tracking
- Mail and communication system
- Cross-rig work coordination

### 2. **Beads Issue Tracker** (`bd` command)
Lightweight issue tracking integrated with Gastown:
- Issue creation and management
- Dependency tracking
- Status management
- Git integration

### 3. **Claude Automation Harness** (NEW - Phase 1 Complete)
Located in `~/gt/harness/`, this is a continuous loop system that:
- Spawns Claude Code agents automatically
- Implements the "Ralph Wiggum loop" pattern
- Builds knowledge from minimal context
- Preserves research across sessions
- Handles interrupts for human attention
- Integrates with beads and Gastown work queues

**Status**: Foundation complete (Phase 1), Claude Code integration pending (Phase 2)

## Registered Rigs

1. **aardwolf_snd** - Mudlet SND package (JavaScript/Lua)
   - Git: `git@github.com:ericbfriday/mudlet-snd-full.git`
   - Branch: `main`
   - Tech: Node.js, Mudlet Lua

2. **duneagent** - Dune MUD agent system (TypeScript)
   - Git: `git@github.com:ericbfriday/duneagent.git`
   - Branch: `main`
   - Tech: Node.js, TypeScript monorepo

## Project Owner

**Eric Friday** (Overseer)
- Email: ericfriday@gmail.com
- Git user: Eric Friday <ericfriday@gmail.com>
