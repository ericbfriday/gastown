# Claude Automation Harness - Implementation Workflow

## Overview

This document outlines the comprehensive workflow for implementing a Claude-based automation harness for the Gastown multi-agent orchestration system. The harness implements a "Ralph Wiggum loop" pattern where Claude agents are spawned in a continuous cycle, each starting with minimal context and building their understanding from documentation and research.

## Core Principles

### 1. Ralph Wiggum Loop Pattern

The harness operates on a continuous cycle:
```
Initialize → Spawn Agent → Agent Works → Agent Completes/Hands Off → Next Iteration
```

Each iteration:
- Starts with minimal bootstrap context
- Agent builds understanding from documentation
- Agent completes discrete work units
- State/knowledge preserved for next iteration
- Cycle repeats automatically

### 2. Minimal Context Bootstrapping

Agents start with only:
- Single prompt file defining their role
- Pointer to documentation directory
- Current work queue/hook
- Session identifier

Everything else is discovered/built during execution.

### 3. Knowledge Accumulation

Research and findings are preserved:
- Serena memory integration
- Session documentation
- Research notes
- Decision logs
- Context handoffs

### 4. Human-in-the-Loop

Harness detects when human attention needed:
- Ambiguous decisions
- Blocked work
- Quality gate failures
- Escalation triggers
- Manual approval gates

## Architecture

### Directory Structure

```
~/gt/
├── harness/                          # Harness root
│   ├── loop.sh                       # Main loop script
│   ├── config.yaml                   # Harness configuration
│   ├── state/                        # Runtime state
│   │   ├── current-session.json      # Active session info
│   │   ├── iteration.log             # Loop iteration log
│   │   └── queue.json                # Work queue
│   ├── prompts/                      # Agent prompts
│   │   ├── bootstrap.md              # Minimal startup context
│   │   ├── worker.md                 # Worker agent prompt
│   │   └── supervisor.md             # Supervisor agent prompt
│   ├── docs/                         # Generated documentation
│   │   ├── research/                 # Ad-hoc research findings
│   │   ├── sessions/                 # Session summaries
│   │   └── decisions/                # Decision logs
│   └── scripts/                      # Helper scripts
│       ├── spawn-agent.sh            # Agent spawning
│       ├── check-interrupt.sh        # Interrupt detection
│       ├── preserve-context.sh       # Context preservation
│       └── report-status.sh          # Status reporting
```

### Component Interactions

```
┌─────────────────────────────────────────────────────────────┐
│                      Harness Loop (loop.sh)                  │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Check      │→ │   Spawn      │→ │   Monitor    │      │
│  │   Queue      │  │   Agent      │  │   Progress   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         ↓                  │                   │             │
│  ┌──────────────┐         │                   │             │
│  │   Human      │←────────┴───────────────────┘             │
│  │   Interrupt? │                                            │
│  └──────────────┘                                            │
│         │                                                     │
│         ↓                                                     │
│  ┌──────────────┐  ┌──────────────┐                         │
│  │   Preserve   │→ │   Next       │                         │
│  │   Context    │  │   Iteration  │                         │
│  └──────────────┘  └──────────────┘                         │
└─────────────────────────────────────────────────────────────┘
         │
         ↓
┌─────────────────────────────────────────────────────────────┐
│                   Claude Agent Session                       │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Load       │→ │   Build      │→ │   Execute    │      │
│  │   Bootstrap  │  │   Context    │  │   Work       │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                           ↓                  │               │
│                    ┌──────────────┐         │               │
│                    │   Research   │         │               │
│                    │   & Learn    │         │               │
│                    └──────────────┘         │               │
│                                              ↓               │
│                                       ┌──────────────┐      │
│                                       │   Complete   │      │
│                                       │   or Handoff │      │
│                                       └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
         │
         ↓
┌─────────────────────────────────────────────────────────────┐
│                    Knowledge Storage                         │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Serena     │  │   Session    │  │   Research   │      │
│  │   Memories   │  │   Docs       │  │   Notes      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)

**Tasks:**
1. Create harness directory structure
2. Implement basic loop script
3. Create bootstrap prompt system
4. Build state management
5. Implement session logging

**Deliverables:**
- `harness/loop.sh` - Basic loop with iteration control
- `harness/prompts/bootstrap.md` - Minimal startup context
- `harness/state/` - State management files
- `harness/scripts/spawn-agent.sh` - Agent spawning script

**Success Criteria:**
- Loop can spawn Claude session
- Session receives bootstrap prompt
- Loop can detect session completion
- Basic state persists between iterations

### Phase 2: Context Building System (Week 1-2)

**Tasks:**
1. Design documentation discovery mechanism
2. Implement research preservation
3. Create Serena memory integration
4. Build context handoff system
5. Add session summarization

**Deliverables:**
- Documentation indexing system
- Research note templates
- Serena memory helpers
- Context handoff protocol
- Session summary generator

**Success Criteria:**
- Agents can discover relevant docs
- Research is automatically preserved
- Context flows between sessions
- Knowledge accumulates over iterations

### Phase 3: Work Orchestration (Week 2)

**Tasks:**
1. Integrate with beads issue tracking
2. Implement work queue management
3. Create rig-aware dispatching
4. Build convoy tracking integration
5. Add parallel work support

**Deliverables:**
- Work queue manager
- Beads integration layer
- Rig dispatch system
- Convoy status tracking
- Parallel agent support

**Success Criteria:**
- Harness can pull work from beads
- Work is dispatched to appropriate rigs
- Multiple agents can work in parallel
- Status is tracked via convoys

### Phase 4: Human Interrupt System (Week 2-3)

**Tasks:**
1. Define interrupt conditions
2. Implement interrupt detection
3. Create pause/resume mechanism
4. Build notification system
5. Add approval gates

**Deliverables:**
- Interrupt condition definitions
- Detection and pause logic
- Resume capability
- Notification hooks (mail integration)
- Approval gate system

**Success Criteria:**
- Harness detects interrupt conditions
- Loop pauses gracefully
- Context preserved during pause
- Human can review and approve
- Work resumes smoothly

### Phase 5: Status & Monitoring (Week 3)

**Tasks:**
1. Build status dashboard
2. Implement progress tracking
3. Create reporting system
4. Add metrics collection
5. Build visualization

**Deliverables:**
- Status dashboard script
- Progress tracking system
- Report generator
- Metrics database
- Terminal UI or web view

**Success Criteria:**
- Real-time status visibility
- Progress reports available
- Metrics tracked over time
- Visual representation of work

### Phase 6: Integration & Testing (Week 3-4)

**Tasks:**
1. End-to-end workflow testing
2. Integration with existing gt/bd commands
3. Error handling and recovery
4. Performance optimization
5. Documentation and runbooks

**Deliverables:**
- Test suite
- Integration tests
- Error recovery procedures
- Performance tuning
- Complete documentation

**Success Criteria:**
- Full workflow operates end-to-end
- Graceful error handling
- Acceptable performance
- Complete documentation
- Ready for production use

## Detailed Component Specifications

### 1. Main Loop Script (`harness/loop.sh`)

```bash
#!/usr/bin/env bash
# Claude Automation Harness - Main Loop
# Implements Ralph Wiggum pattern for continuous agent spawning

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
ITERATION_FILE="$STATE_DIR/iteration.log"
SESSION_FILE="$STATE_DIR/current-session.json"
QUEUE_FILE="$STATE_DIR/queue.json"

# Configuration
MAX_ITERATIONS=${MAX_ITERATIONS:-0}  # 0 = infinite
ITERATION_DELAY=${ITERATION_DELAY:-5}  # seconds between iterations
INTERRUPT_CHECK_INTERVAL=${INTERRUPT_CHECK_INTERVAL:-30}  # seconds

# Core loop functions
init_harness() { ... }
check_work_queue() { ... }
spawn_agent() { ... }
monitor_session() { ... }
check_interrupt() { ... }
preserve_context() { ... }
next_iteration() { ... }

# Main loop
main() {
  init_harness

  iteration=0
  while true; do
    iteration=$((iteration + 1))
    log_iteration "$iteration"

    # Check for available work
    if ! check_work_queue; then
      log "No work available, waiting..."
      sleep "$ITERATION_DELAY"
      continue
    fi

    # Spawn agent for work
    spawn_agent

    # Monitor agent session
    while monitor_session; do
      # Check for interrupt conditions
      if check_interrupt; then
        log "Interrupt detected, pausing..."
        preserve_context
        wait_for_resume
      fi
      sleep "$INTERRUPT_CHECK_INTERVAL"
    done

    # Session complete, prepare next iteration
    next_iteration

    # Check iteration limit
    if [[ $MAX_ITERATIONS -gt 0 && $iteration -ge $MAX_ITERATIONS ]]; then
      log "Reached max iterations ($MAX_ITERATIONS), exiting"
      break
    fi

    sleep "$ITERATION_DELAY"
  done
}

main "$@"
```

### 2. Bootstrap Prompt (`harness/prompts/bootstrap.md`)

```markdown
# Claude Agent Bootstrap

You are a worker agent in the Gastown multi-agent orchestration system. This session is part of an automated harness that spawns agents to work on tasks continuously.

## Your Mission

Work on the assigned task from your hook. Build context as needed from available documentation.

## Starting Context (Minimal)

**Session ID:** {{SESSION_ID}}
**Iteration:** {{ITERATION}}
**Assigned Work:** {{HOOK_BEAD}}

## Documentation Sources

Available documentation (build your context from these):
- `~/gt/GASTOWN-CLAUDE.md` - Complete system guide
- `~/gt/AGENTS.md` - Agent workflow reference
- `~/gt/.beads/` - Beads issue tracking
- `{{RIG}}/AGENTS.md` - Rig-specific guidance
- `.serena/memories/` - Previous research and findings

## Your Workflow

1. **Load context** - Run priming commands:
   ```bash
   gt prime && bd prime
   gt hook  # See your assignment
   ```

2. **Build understanding** - Read relevant docs, research as needed

3. **Execute work** - Follow molecule steps or work plan

4. **Preserve knowledge** - Document findings, research, decisions

5. **Complete or handoff**:
   - If complete: close issue, push changes
   - If need human: use interrupt mechanism
   - If context full: use `gt handoff`

## Interrupt Conditions

Signal for human attention by creating file:
```bash
echo "REASON" > ~/gt/harness/state/interrupt-request.txt
```

Reasons include:
- Ambiguous requirement
- Multiple valid approaches
- Quality gate failure
- Escalation needed
- Manual approval required

## Research Preservation

Save any research or findings:
```bash
# Serena memory
gt serena write-memory <name> <content>

# Session documentation
echo "Findings..." > ~/gt/harness/docs/research/{{SESSION_ID}}-<topic>.md
```

## Key Principles

- Start minimal, build context as needed
- Preserve all research and findings
- Signal clearly when human needed
- Document decisions and rationale
- Follow existing patterns and conventions

**Remember:** You are part of a continuous loop. Your work contributes to the overall system, and your findings help future iterations.
```

### 3. Interrupt Detection (`harness/scripts/check-interrupt.sh`)

```bash
#!/usr/bin/env bash
# Check for interrupt conditions

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INTERRUPT_FILE="$HARNESS_ROOT/state/interrupt-request.txt"
STATE_DIR="$HARNESS_ROOT/state"

# Check for explicit interrupt request
if [[ -f "$INTERRUPT_FILE" ]]; then
  reason=$(cat "$INTERRUPT_FILE")
  echo "INTERRUPT: $reason"
  exit 0
fi

# Check for quality gate failures
if [[ -f "$STATE_DIR/quality-gate-failed" ]]; then
  echo "INTERRUPT: Quality gate failed"
  exit 0
fi

# Check for blocked work
if gt hook | grep -q "blocked"; then
  echo "INTERRUPT: Work is blocked"
  exit 0
fi

# Check for manual approval gates
# ... additional checks ...

# No interrupt detected
exit 1
```

### 4. Work Queue Manager (`harness/scripts/manage-queue.sh`)

```bash
#!/usr/bin/env bash
# Manage work queue for harness

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QUEUE_FILE="$HARNESS_ROOT/state/queue.json"

check_queue() {
  # Get ready work from beads
  ready_work=$(bd ready --json 2>/dev/null || echo "[]")

  # Get work from gt across rigs
  gt_ready=$(gt ready --json 2>/dev/null || echo "[]")

  # Merge and prioritize
  jq -s '.[0] + .[1] | sort_by(.priority) | reverse' \
    <(echo "$ready_work") \
    <(echo "$gt_ready") \
    > "$QUEUE_FILE"

  # Return count
  jq 'length' "$QUEUE_FILE"
}

get_next_work() {
  # Get highest priority work item
  jq -r '.[0] | @json' "$QUEUE_FILE"
}

mark_claimed() {
  local issue_id="$1"
  # Remove from queue
  jq "del(.[0])" "$QUEUE_FILE" > "$QUEUE_FILE.tmp"
  mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"
}

case "${1:-}" in
  check) check_queue ;;
  next) get_next_work ;;
  claim) mark_claimed "$2" ;;
  *) echo "Usage: $0 {check|next|claim <id>}" >&2; exit 1 ;;
esac
```

### 5. Context Preservation (`harness/scripts/preserve-context.sh`)

```bash
#!/usr/bin/env bash
# Preserve context during interrupts or handoffs

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"
SESSION_ID="${SESSION_ID:-unknown}"

preserve_context() {
  local context_file="$DOCS_DIR/sessions/${SESSION_ID}-context.json"

  # Capture current state
  jq -n \
    --arg session "$SESSION_ID" \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson hook "$(gt hook --json 2>/dev/null || echo '{}')" \
    --argjson status "$(git -C ~/gt status --porcelain --json 2>/dev/null || echo '{}')" \
    '{
      session: $session,
      timestamp: $timestamp,
      hook: $hook,
      git_status: $status,
      rig: env.BD_RIG,
      actor: env.BD_ACTOR
    }' > "$context_file"

  # Capture Serena memories
  if command -v gt &>/dev/null; then
    gt serena list-memories > "$DOCS_DIR/sessions/${SESSION_ID}-memories.txt"
  fi

  # Capture recent logs
  tail -n 100 "$STATE_DIR/iteration.log" > "$DOCS_DIR/sessions/${SESSION_ID}-logs.txt"

  echo "Context preserved to $context_file"
}

preserve_context
```

### 6. Status Reporting (`harness/scripts/report-status.sh`)

```bash
#!/usr/bin/env bash
# Generate status report for harness

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"

generate_report() {
  cat <<EOF
# Claude Harness Status Report
Generated: $(date)

## Current State
$(cat "$STATE_DIR/current-session.json" 2>/dev/null | jq -r '
  "Session: \(.session_id)\n" +
  "Iteration: \(.iteration)\n" +
  "Started: \(.started_at)\n" +
  "Status: \(.status)"
')

## Work Queue
$(harness/scripts/manage-queue.sh check) items ready

## Recent Activity
$(tail -n 10 "$STATE_DIR/iteration.log")

## Active Rigs
$(gt rig list 2>/dev/null)

## Convoys in Progress
$(gt convoy list --active 2>/dev/null)

## Interrupts (Last 24h)
$(find "$DOCS_DIR/sessions" -name "*context.json" -mtime -1 | wc -l) interrupts

EOF
}

generate_report
```

## Integration Points

### Beads Integration

The harness integrates with beads for work tracking:

```bash
# Pull work from beads ready queue
bd ready --json

# Claim work
bd update <id> --status in_progress

# Track progress via convoys
gt convoy create "Harness Batch $(date +%Y%m%d)" <issue-ids...>

# Close on completion
bd close <id>
bd sync
```

### Gastown (gt) Integration

Integration with gt commands:

```bash
# Check agent hooks
gt hook

# Dispatch work to rigs
gt sling <issue-id> <rig>

# Status across rigs
gt ready

# Mail for interrupts
gt mail send overseer -s "INTERRUPT: ..." -m "Details..."
```

### Serena Memory Integration

Knowledge preservation via Serena:

```bash
# Write research findings
gt serena write-memory "research-<topic>" "Content..."

# List available memories
gt serena list-memories

# Read previous research
gt serena read-memory "research-<topic>"
```

## Configuration

Harness configuration (`harness/config.yaml`):

```yaml
harness:
  # Loop control
  max_iterations: 0  # 0 = infinite
  iteration_delay: 5  # seconds
  interrupt_check_interval: 30  # seconds

  # Agent spawning
  agent_type: claude-sonnet  # Model to use
  session_timeout: 3600  # Max session duration (seconds)
  parallel_agents: 1  # Number of concurrent agents

  # Work routing
  default_rig: aardwolf_snd
  rig_priority:
    - aardwolf_snd
    - duneagent

  # Interrupt conditions
  interrupts:
    quality_gate_failure: true
    blocked_work: true
    approval_required: true
    ambiguous_requirement: true

  # Knowledge preservation
  preserve_research: true
  serena_integration: true
  session_summaries: true

  # Notifications
  notify_on_interrupt: overseer
  notify_on_completion: false
  notify_on_error: overseer
```

## Usage Examples

### Starting the Harness

```bash
# Start with default configuration
cd ~/gt/harness
./loop.sh

# Start with custom iteration limit
MAX_ITERATIONS=10 ./loop.sh

# Start with faster cycle
ITERATION_DELAY=2 ./loop.sh

# Run in background
nohup ./loop.sh > loop.out 2>&1 &
```

### Checking Status

```bash
# Quick status
./scripts/report-status.sh

# Detailed status
./scripts/report-status.sh --detailed

# Watch status
watch -n 5 ./scripts/report-status.sh
```

### Interrupting

```bash
# Request interrupt
echo "Need manual review of PR" > state/interrupt-request.txt

# Check interrupt status
cat state/interrupt-request.txt

# Resume after interrupt
rm state/interrupt-request.txt
```

### Viewing Logs

```bash
# Iteration log
tail -f state/iteration.log

# Session logs
ls -lt docs/sessions/

# Recent session
cat docs/sessions/$(ls -t docs/sessions/ | head -1)
```

## Error Handling

### Common Error Scenarios

1. **Agent Spawn Failure**
   - Log error details
   - Preserve work item in queue
   - Continue to next iteration
   - Notify on repeated failures

2. **Quality Gate Failure**
   - Trigger interrupt
   - Preserve context
   - Notify human
   - Wait for manual fix

3. **Work Queue Empty**
   - Sleep and retry
   - Check upstream systems (beads, gt)
   - Log idle time
   - Continue loop

4. **Context Overflow**
   - Trigger handoff
   - Preserve research
   - Spawn fresh agent
   - Continue work

5. **Git Conflicts**
   - Attempt auto-resolution
   - If fails, trigger interrupt
   - Preserve state
   - Wait for manual resolution

### Recovery Procedures

```bash
# Reset harness state
./scripts/reset-harness.sh

# Recover from crash
./scripts/recover.sh

# Re-queue failed work
./scripts/requeue.sh <issue-id>
```

## Monitoring & Observability

### Metrics to Track

- Iterations per hour
- Work items completed
- Average session duration
- Interrupt frequency
- Research documents created
- Memory entries added
- Success vs failure rate

### Log Files

- `state/iteration.log` - Main loop iterations
- `state/sessions/*.log` - Individual session logs
- `state/errors.log` - Error tracking
- `docs/sessions/*-context.json` - Session contexts

### Dashboards

Future enhancements could include:
- Web-based status dashboard
- Real-time metrics visualization
- Work queue visualization
- Rig health monitoring
- Agent performance analytics

## Future Enhancements

### Planned Features

1. **Multi-Agent Parallelism**
   - Run multiple agents concurrently
   - Work stealing/balancing
   - Coordination between agents

2. **Learning & Optimization**
   - Track success patterns
   - Optimize work routing
   - Improve context building
   - Adaptive interrupt thresholds

3. **Advanced Orchestration**
   - Dependency-aware scheduling
   - Critical path optimization
   - Resource-aware dispatching

4. **Enhanced Monitoring**
   - Web dashboard
   - Real-time notifications
   - Performance analytics
   - Predictive alerts

5. **Integration Expansion**
   - GitHub integration
   - Slack notifications
   - Metrics export (Prometheus/Grafana)
   - CI/CD integration

## Success Metrics

The harness is successful when:

- ✅ Runs continuously without manual intervention
- ✅ Agents successfully build context from minimal prompts
- ✅ Work progresses across multiple rigs
- ✅ Research and knowledge is preserved
- ✅ Human interrupts are timely and appropriate
- ✅ No work is lost or stranded
- ✅ Status is always visible and accurate
- ✅ System recovers gracefully from errors

## References

- [Gastown Architecture](GASTOWN-CLAUDE.md)
- [Agent Guidelines](AGENTS.md)
- [Beads Documentation](.beads/README.md)
- [Ralph Wiggum Loop Pattern](https://github.com/anthropics/claude-code/docs/ralph-wiggum.md) (reference)

---

**Document Status:** Draft - Implementation Plan
**Created:** 2026-01-27
**Version:** 1.0
**Author:** Claude (Architecture Design Session)
