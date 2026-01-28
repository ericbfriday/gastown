# Claude Automation Harness

**Version:** 2.0
**Status:** Phase 2 Complete - Production-Ready Claude Code Integration
**Created:** 2026-01-20
**Phase 2 Completed:** 2026-01-27
**Next Phase:** Parallel Agent Support (Phase 3)

## Overview

The Claude Automation Harness is a continuous loop system that spawns Claude Code agents to work on tasks in the Gastown multi-agent orchestration environment. It implements the "Ralph Wiggum loop" pattern where agents start with minimal context, build their understanding from documentation, execute work, and preserve knowledge for future iterations.

## Key Features

### Phase 1: Foundation ✅
- **Continuous Loop**: Automatically spawns agents in succession
- **Minimal Context Bootstrap**: Agents start lean and build what they need
- **Knowledge Preservation**: Research and findings persist across sessions
- **Human-in-the-Loop**: Interrupts when human attention is required
- **Work Queue Integration**: Pulls work from beads and Gastown
- **Status Tracking**: Visibility into agent activity and progress
- **Graceful Error Handling**: Recovers from failures and preserves state

### Phase 2: Claude Code Integration ✅ (NEW)
- **Production-Ready Spawning**: Actual Claude Code agent spawning via `claude -p`
- **Real-Time Monitoring**: Stream-JSON event parsing for immediate visibility
- **Session Lifecycle Management**: Complete tracking from spawn to completion
- **Comprehensive Metrics**: Token usage, tool calls, duration, and performance data
- **Background Processing**: Non-blocking output monitoring
- **Heartbeat Mechanism**: Liveness tracking and stall detection
- **Robust Error Recovery**: Handles crashes, timeouts, and failures gracefully
- **Event Analysis Tools**: Rich session analysis and debugging capabilities
- **Complete Audit Trail**: Filesystem-based state tracking for forensics
- **76 Integration Tests**: Comprehensive test coverage validates all functionality

## Quick Start

### Prerequisites

Required commands in your environment:
- `gt` (Gastown)
- `bd` (Beads)
- `git`
- `jq`
- `claude` (Claude Code CLI)

### Installation

The harness is already set up in `~/gt/harness/`. No installation needed.

### Basic Usage

```bash
# Navigate to harness directory
cd ~/gt/harness

# Start the harness (runs continuously with real Claude agents)
./loop.sh

# Start with iteration limit (for testing)
MAX_ITERATIONS=5 ./loop.sh

# Run in background
nohup ./loop.sh > loop.out 2>&1 &

# Check status (enhanced with real-time session data)
./scripts/report-status.sh

# Stop (Ctrl+C or send SIGTERM to process)
pkill -f "loop.sh"
```

### Monitoring Active Sessions (NEW in Phase 2)

```bash
# Watch session in real-time
SESSION_ID=$(jq -r '.session_id' state/current-session.json)
./scripts/parse-session-events.sh watch $SESSION_ID

# View session summary
./scripts/parse-session-events.sh latest

# Analyze tool usage
./scripts/parse-session-events.sh tools $SESSION_ID

# Check for errors
./scripts/parse-session-events.sh errors $SESSION_ID

# View event timeline
./scripts/parse-session-events.sh timeline $SESSION_ID

# Calculate metrics
./scripts/parse-session-events.sh metrics $SESSION_ID
```

### Configuration

Edit `harness/config.yaml` to customize:
- Iteration timing
- Interrupt conditions
- Work routing
- Quality gates
- Notifications

See [config.yaml](config.yaml) for all options.

## How It Works

### The Loop

```
1. Check work queue (from beads/gt)
2. If work available:
   a. Spawn Claude agent with bootstrap prompt
   b. Monitor agent session
   c. Check for interrupts periodically
   d. On completion, preserve context
   e. Move to next iteration
3. If no work, wait and retry
4. Repeat until max iterations or manual stop
```

### Agent Lifecycle

Each spawned agent:
1. **Loads minimal context** - Session ID, assigned work
2. **Builds understanding** - Reads relevant docs, explores codebase
3. **Executes work** - Follows molecule steps or work plan
4. **Preserves knowledge** - Saves research, documents decisions
5. **Completes or handoffs** - Closes work or signals for help

### Knowledge Accumulation

Research and findings are preserved in:
- `docs/research/` - Ad-hoc research notes
- `docs/sessions/` - Session context and summaries
- `docs/decisions/` - Decision logs
- Serena memories (via `gt serena write-memory`)

Future agents can leverage this accumulated knowledge.

### Interrupt Mechanism

Agents signal for human attention by creating:
```bash
echo "REASON" > ~/gt/harness/state/interrupt-request.txt
```

The harness:
1. Detects the interrupt
2. Preserves session context
3. Sends notification to overseer
4. Pauses the loop
5. Waits for human to remove interrupt file
6. Resumes automatically

## Directory Structure

```
harness/
├── loop.sh                    # Main loop script
├── config.yaml                # Configuration
├── README.md                  # This file
├── state/                     # Runtime state
│   ├── current-session.json   # Active session info
│   ├── iteration.log          # Loop activity log
│   ├── queue.json             # Work queue
│   └── interrupt-request.txt  # Interrupt signal (when present)
├── prompts/                   # Agent prompts
│   └── bootstrap.md           # Bootstrap prompt template
├── docs/                      # Generated documentation
│   ├── research/              # Research findings
│   ├── sessions/              # Session contexts & summaries
│   └── decisions/             # Decision logs
└── scripts/                   # Helper scripts
    ├── manage-queue.sh        # Work queue manager
    ├── check-interrupt.sh     # Interrupt detection
    ├── preserve-context.sh    # Context preservation
    └── report-status.sh       # Status reporting
```

## Scripts Reference

### Main Loop (`loop.sh`)

**Purpose:** Core harness loop

**Usage:**
```bash
./loop.sh

# With options
MAX_ITERATIONS=10 ./loop.sh
ITERATION_DELAY=2 ./loop.sh
```

**Environment Variables:**
- `MAX_ITERATIONS` - Stop after N iterations (0 = infinite, default)
- `ITERATION_DELAY` - Seconds between iterations (default: 5)
- `INTERRUPT_CHECK_INTERVAL` - Seconds between interrupt checks (default: 30)
- `AGENT_TYPE` - Model to use (default: claude-sonnet)
- `SESSION_TIMEOUT` - Max session duration (default: 3600s)

### Queue Manager (`scripts/manage-queue.sh`)

**Purpose:** Manage work queue

**Usage:**
```bash
# Refresh queue and show count
./scripts/manage-queue.sh check

# Get next work item
./scripts/manage-queue.sh next

# Mark item as claimed
./scripts/manage-queue.sh claim <issue-id>

# Add item to queue
./scripts/manage-queue.sh add '<json>'

# Show queue contents
./scripts/manage-queue.sh show

# Clear queue
./scripts/manage-queue.sh clear
```

### Interrupt Check (`scripts/check-interrupt.sh`)

**Purpose:** Detect interrupt conditions

**Returns:** Exit 0 if interrupt detected, exit 1 if none

**Checks:**
- Explicit interrupt request file
- Quality gate failures
- Blocked work
- Approval requirements
- Session timeout
- Error conditions

### Context Preservation (`scripts/preserve-context.sh`)

**Purpose:** Save session context during interrupts

**Output:**
- `docs/sessions/<session>-context.json` - Full context
- `docs/sessions/<session>-summary.md` - Human-readable summary
- `docs/sessions/<session>-memories.txt` - Serena memories list
- `docs/sessions/<session>-logs.txt` - Recent iteration logs
- `docs/sessions/<session>-beads.json` - Beads state

### Status Report (`scripts/report-status.sh`)

**Purpose:** Generate status report

**Usage:**
```bash
# Quick status
./scripts/report-status.sh

# Detailed status
./scripts/report-status.sh --detailed

# Watch continuously
watch -n 5 ./scripts/report-status.sh
```

### Session Event Parser (`scripts/parse-session-events.sh`) - NEW in Phase 2

**Purpose:** Analyze and monitor Claude Code sessions

**Commands:**
```bash
# Show session summary
./scripts/parse-session-events.sh summary <session_id>

# List all tool calls
./scripts/parse-session-events.sh tools <session_id>

# Show errors
./scripts/parse-session-events.sh errors <session_id>

# View event timeline
./scripts/parse-session-events.sh timeline <session_id>

# Calculate metrics (tokens, duration, etc.)
./scripts/parse-session-events.sh metrics <session_id>

# Export events to JSON
./scripts/parse-session-events.sh export <session_id>

# Watch session in real-time
./scripts/parse-session-events.sh watch <session_id>

# List all sessions
./scripts/parse-session-events.sh list

# Show latest session summary
./scripts/parse-session-events.sh latest
```

**Output includes:**
- Event counts (messages, tool calls, errors)
- Session state and timestamps
- API usage (input/output tokens)
- Tool usage breakdown
- Performance metrics

## Workflow Examples

### Starting the Harness

```bash
cd ~/gt/harness

# Check configuration
cat config.yaml

# Verify work available
bd ready

# Start harness
./loop.sh
```

### Monitoring

```bash
# Terminal 1: Watch overall status
watch -n 5 ./scripts/report-status.sh

# Terminal 2: Watch active session in real-time (NEW in Phase 2)
SESSION_ID=$(jq -r '.session_id' state/current-session.json)
./scripts/parse-session-events.sh watch $SESSION_ID

# Terminal 3: Monitor harness logs
tail -f state/iteration.log

# Or check recent sessions
ls -lt docs/sessions/

# View session metrics (NEW in Phase 2)
./scripts/parse-session-events.sh latest
```

### Interrupting

```bash
# Request interrupt (from agent or manually)
echo "Need manual code review" > state/interrupt-request.txt

# Harness will pause and preserve context

# Review the situation
cat docs/sessions/*-summary.md | tail -1

# When ready to resume
rm state/interrupt-request.txt

# Harness continues automatically
```

### Troubleshooting

```bash
# Check for errors in harness logs
tail state/iteration.log | grep ERROR

# Analyze latest session (NEW in Phase 2)
./scripts/parse-session-events.sh latest

# Check for session errors
SESSION_ID=$(jq -r '.session_id' state/current-session.json)
./scripts/parse-session-events.sh errors $SESSION_ID

# View session metrics
./scripts/parse-session-events.sh metrics $SESSION_ID

# Check work queue
./scripts/manage-queue.sh show

# Reset harness state
rm state/current-session.json
rm state/interrupt-request.txt

# Kill stalled agent (if needed)
PID=$(jq -r '.pid' state/current-session.json)
kill $PID
```

## Integration with Gastown

The harness integrates with Gastown commands:

### Work Discovery

```bash
# Harness automatically pulls from:
bd ready          # Beads ready work
gt ready          # Cross-rig ready work

# Prioritizes and queues in state/queue.json
```

### Work Dispatch

Agents can dispatch work to rigs:
```bash
gt sling <issue-id> <rig>
```

### Status Tracking

Work is tracked via:
- Beads issue status
- Gastown convoys
- Session documentation

### Knowledge Preservation

Findings are stored in:
- Serena memories (`gt serena write-memory`)
- Session documentation
- Research notes

## Quality Gates

The harness enforces quality gates:

### Pre-Work (Advisory)
- Check that main branch is healthy
- File issues if tests are already failing

### Post-Work (Required)
- All tests must pass
- Build must succeed
- Git working tree must be clean
- Changes must be pushed

Failure triggers interrupt for human resolution.

## What's New in Phase 2

### Completed Features

✅ **Real Claude Code Integration**
- Actual `claude -p` spawning (not placeholder)
- Stream-JSON output format for monitoring
- Bootstrap prompt injection
- Session lifecycle management

✅ **Real-Time Monitoring**
- Background output processor
- Event parsing and analysis
- Heartbeat mechanism
- Stall detection

✅ **Comprehensive State Tracking**
- Complete session lifecycle
- Filesystem audit trail
- Metrics collection
- Event logging

✅ **Production-Ready Error Handling**
- Spawn failure recovery
- Timeout detection
- Crash handling
- Graceful degradation

✅ **Event Analysis Tools**
- Session summaries
- Tool usage analysis
- Error investigation
- Timeline reconstruction
- Metrics calculation

### Test Coverage

- 7 test suites
- 76 integration tests
- 4,560 lines of test code
- Mock Claude CLI for isolation
- All tests passing

## Future Enhancements

### Phase 3: Parallel Agent Support (Next)

1. **N Concurrent Agents**
   - Lock-free work coordination
   - Git worktree isolation
   - Independent agent processes
   - Target: 2.5x throughput with 3 agents

2. **Agent Pool Management**
   - Dynamic scaling
   - Health monitoring
   - Failure recovery
   - Work reclamation

3. **Status Aggregation**
   - Unified view across agents
   - Performance metrics
   - Resource utilization
   - Throughput tracking

**Timeline:** 4 weeks
**Design Complete:** See `docs/research/parallel-coordination-design.md`

### Phase 4-6: Advanced Features

- Knowledge preservation automation
- Intelligent work orchestration
- Web dashboard with real-time metrics
- Alert system (Slack/email)
- Performance analytics

## Documentation

### Quick Reference

- **[ROADMAP.md](ROADMAP.md)** - Implementation phases and timeline
- **[config.yaml](config.yaml)** - Configuration reference
- **[WORKFLOW.md](docs/WORKFLOW.md)** - Detailed workflow guide

### Phase 2 Documentation (NEW)

- **[PHASE-2-SUMMARY.md](docs/PHASE-2-SUMMARY.md)** - Complete Phase 2 implementation summary
- **[PRODUCTION-ROLLOUT.md](docs/PRODUCTION-ROLLOUT.md)** - Production deployment guide
- **[claude-code-cli-research.md](docs/research/claude-code-cli-research.md)** - CLI capabilities research
- **[spawn-mechanism-architecture.md](docs/research/spawn-mechanism-architecture.md)** - Spawn design specification
- **[parallel-coordination-design.md](docs/research/parallel-coordination-design.md)** - Phase 3 design

### Architecture Documentation

- **Session State**: `state/current-session.json` - Active session tracking
- **Event Logs**: `docs/sessions/<session_id>.log` - Stream-JSON events
- **Session Archives**: `docs/sessions/<session_id>.json` - Completed sessions
- **Transcripts**: `~/.claude/transcripts/<session_id>.jsonl` - Full conversations
- **Metrics**: `state/sessions/<session_id>/metrics.json` - Performance data

### Testing

- **Test Suites**: `tests/test-*.sh` - Integration test suites
- **Test Library**: `tests/test-lib.sh` - Common test utilities
- **Mock CLI**: `tests/mocks/mock-claude.sh` - Mock Claude for testing

Run tests:
```bash
cd tests
./test-spawn.sh
./test-monitoring.sh
./test-lifecycle.sh
# ... or run all
for test in test-*.sh; do ./$test; done
```

### Known Limitations

**Current (Phase 2):**
- Single agent only (parallel support in Phase 3)
- No automatic work discovery (manual queue refresh)
- No web dashboard (CLI tools only)
- Manual interrupt resolution (no automatic notifications)
- No work stealing/reclamation yet

**Resolved in Phase 2:**
- ✅ Agent spawning now production-ready (was placeholder)
- ✅ Real-time monitoring implemented
- ✅ Comprehensive metrics collection added
- ✅ Automatic error recovery working

## Configuration Reference

See [config.yaml](config.yaml) for detailed configuration options.

### Key Settings

**Loop Control:**
- `max_iterations` - Stop after N iterations (0 = infinite)
- `iteration_delay` - Seconds between iterations
- `interrupt_check_interval` - Interrupt check frequency

**Agent:**
- `agent_type` - Model to use (claude-sonnet, claude-opus)
- `session_timeout` - Max session duration

**Interrupts:**
- Enable/disable specific interrupt conditions
- Configure notification recipients

**Quality Gates:**
- Pre-work and post-work checks
- Required vs advisory gates
- Actions on failure

## Best Practices

### For Harness Operators

1. **Monitor regularly** - Use `watch -n 5 ./scripts/report-status.sh`
2. **Review interrupts promptly** - Agents are waiting for human input
3. **Keep docs updated** - Agents rely on `docs/research/` and Serena memories
4. **Preserve knowledge** - Don't delete research notes or session archives
5. **Analyze failures** - Use `parse-session-events.sh` to investigate issues
6. **Watch resource usage** - Monitor disk space, CPU, and API costs
7. **Review metrics** - Track success rates and throughput over time

### For Agent Development

1. **Start minimal** - Don't load unnecessary context upfront
2. **Document decisions** - Future agents need context (use Serena)
3. **Preserve research** - Save what you learn in `docs/research/`
4. **Signal early** - Interrupt when blocked, don't waste API tokens
5. **Test continuously** - Don't defer testing until the end

### For System Health

1. **Keep main healthy** - Fix broken tests promptly
2. **Manage work queue** - Don't let it grow unbounded
3. **Archive old sessions** - Prevent disk bloat (compress logs >7 days)
4. **Update documentation** - Keep guides current as system evolves
5. **Review metrics** - Identify patterns and problems early

## Production Readiness

### Pre-Production Checklist

- [ ] Review [PRODUCTION-ROLLOUT.md](docs/PRODUCTION-ROLLOUT.md)
- [ ] Complete staging tests (all 6 stages)
- [ ] Verify resource requirements met
- [ ] Configure monitoring
- [ ] Set up backup procedures
- [ ] Train operators
- [ ] Define escalation path

### Deployment

See [PRODUCTION-ROLLOUT.md](docs/PRODUCTION-ROLLOUT.md) for complete deployment guide including:
- Prerequisites verification
- Installation steps
- Configuration
- Staging tests
- Production deployment
- Monitoring setup
- Troubleshooting
- Rollback procedures

### Support

**Documentation:**
- [PHASE-2-SUMMARY.md](docs/PHASE-2-SUMMARY.md) - What was built
- [PRODUCTION-ROLLOUT.md](docs/PRODUCTION-ROLLOUT.md) - Deployment guide
- [ROADMAP.md](ROADMAP.md) - Future phases
- [docs/research/](docs/research/) - Technical specifications

**Operational Runbook:**
See [PRODUCTION-ROLLOUT.md § Operational Runbook](docs/PRODUCTION-ROLLOUT.md#operational-runbook)

**Testing:**
```bash
cd tests && ./test-spawn.sh  # Test spawning
cd tests && ./test-monitoring.sh  # Test monitoring
```

## Troubleshooting

### Harness Won't Start

```bash
# Check required commands
which gt bd jq git

# Check directory structure
ls -la state/ prompts/ scripts/

# Check permissions
ls -l loop.sh scripts/*.sh

# Check configuration
cat config.yaml | grep -v "^#"
```

### No Work Being Processed

```bash
# Check queue
./scripts/manage-queue.sh show

# Check beads
bd ready

# Check gt
gt ready

# Manually refresh queue
./scripts/manage-queue.sh check
```

### Agent Not Spawning

```bash
# Check session file
cat state/current-session.json

# Check for errors in log
tail state/iteration.log | grep ERROR

# Verify Claude Code is available
which claude

# Check spawn_agent() function in loop.sh
# (Currently placeholder - needs implementation)
```

### Interrupts Not Working

```bash
# Verify interrupt file location
ls -la state/interrupt-request.txt

# Check interrupt detection
./scripts/check-interrupt.sh
echo $?  # Should be 0 if interrupt present

# Check configuration
grep -A10 "interrupts:" config.yaml
```

### Context Not Preserved

```bash
# Check preservation script
./scripts/preserve-context.sh

# Check output directory
ls -la docs/sessions/

# Verify permissions
ls -ld docs/sessions/
```

## FAQ

**Q: How do I stop the harness?**
A: Ctrl+C or `pkill -f loop.sh`. It will shutdown gracefully.

**Q: Can I run multiple harnesses?**
A: Not recommended yet. Parallel support is planned but not implemented.

**Q: How do I test the harness without actually working?**
A: Set `MAX_ITERATIONS=1` to run a single iteration for testing.

**Q: What happens if the harness crashes?**
A: Restart it. Context is preserved in session files. You may need to manually clean up `state/current-session.json`.

**Q: How do I integrate my own quality gates?**
A: Edit `config.yaml` under `quality_gates:` section.

**Q: Can agents spawn sub-agents?**
A: Not directly via harness. Agents can use `gt sling` to dispatch work.

**Q: How is this different from running agents manually?**
A: Harness provides automation, knowledge accumulation, interrupt handling, and continuous operation.

## Support & Contribution

**Issues:** File in Gastown repo or notify overseer via `gt mail`

**Documentation:** `~/gt/docs/CLAUDE-HARNESS-WORKFLOW.md` - Full implementation workflow

**Contact:** Eric Friday <ericfriday@gmail.com>

---

**Built for Gastown** - Multi-agent orchestration with complete attribution
**Powered by Claude Code** - Anthropic's official CLI for Claude
