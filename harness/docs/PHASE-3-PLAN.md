# Phase 3 Implementation Plan: Parallel Agent Support

**Version:** 1.0
**Status:** Planning Complete - Ready for Implementation
**Date:** 2026-01-28
**Timeline:** 4 weeks
**Target Completion:** 2026-02-25

## Executive Summary

Phase 3 extends the Claude automation harness from single-agent operation to parallel multi-agent coordination, enabling 3+ agents to work simultaneously on different tasks. This phase builds on the production-ready foundation established in Phase 2, adding lock-free coordination primitives, git worktree isolation, comprehensive health monitoring, and robust failure recovery.

**Expected Outcome:** 2.5x throughput improvement (10 items/hour vs 4 items/hour) with 3 concurrent agents, while maintaining the simplicity and observability of the Phase 1-2 design.

### Key Design Principles

1. **Lock-Free Coordination** - Atomic filesystem operations eliminate shared state
2. **Complete Isolation** - Each agent in dedicated worktree with own state directory
3. **Zero Shared Mutable State** - Agents coordinate via immutable filesystem artifacts
4. **Observable Operation** - Filesystem-based audit trail for all actions
5. **Graceful Degradation** - System continues with reduced capacity on agent failures
6. **Resource Limits** - Per-agent memory/CPU/disk constraints prevent resource exhaustion

### Core Architecture Changes

**Before Phase 3 (Single Agent):**
```
loop.sh → Queue → Spawn Agent → Monitor → Complete
```

**After Phase 3 (Parallel Agents):**
```
parallel-loop.sh (Coordinator)
    ├─→ Agent Pool Manager (spawn/monitor/terminate N agents)
    ├─→ Work Queue Manager (atomic claim mechanism)
    ├─→ Health Monitor (heartbeat + stall detection)
    └─→ Status Aggregator (unified view)
         │
         ├─→ Agent 1 (worktree-1/, state/agents/agent-1/)
         ├─→ Agent 2 (worktree-2/, state/agents/agent-2/)
         └─→ Agent N (worktree-N/, state/agents/agent-N/)
```

## Architecture Overview

### System Components

#### 1. Parallel Coordinator (`parallel-loop.sh`)

**Responsibilities:**
- Spawn and maintain N agent worker processes
- Monitor agent health via heartbeat mechanism
- Detect and recover from agent failures
- Aggregate status across all agents
- Manage coordinator lifecycle (start, run, stop, crash recovery)

**Key Functions:**
- `manage_agent_pool()` - Scale up/down to target agent count
- `spawn_new_agent()` - Create isolated agent worker
- `monitor_agent_health()` - Check heartbeats, detect stalls/crashes
- `handle_dead_agent()` - Recover from agent failures
- `recover_from_crash()` - Restore state after coordinator crash
- `aggregate_status()` - Unified status view

#### 2. Agent Worker Process (`scripts/spawn-agent-worker.sh`)

**Responsibilities:**
- Independent process running agent lifecycle
- Claim work from shared queue atomically
- Execute work in isolated worktree
- Maintain heartbeat for liveness
- Handle interrupts gracefully
- Release work on completion/failure

**Lifecycle:**
```
Start → Initialize → Claim Work → Execute → Release → Repeat
  ↓                    ↓              ↓
State: idle        working       completing
  ↓                    ↓              ↓
Heartbeat          Heartbeat      Heartbeat
```

#### 3. Work Queue Manager (`scripts/manage-queue.sh` - Enhanced)

**New Capabilities:**
- **Atomic Claims** - Lock-free work claiming via hard links
- **Claim Tracking** - Track which agent claimed which work
- **Stale Detection** - Reclaim work from dead/slow agents
- **Priority Ordering** - Agents pull highest-priority available work

**Claim Protocol:**
```bash
# Atomic claim via hard link
ln "$agent_marker" "$claim_file" 2>/dev/null
  → Success: Agent gets work
  → Failure: Already claimed by another agent
```

#### 4. Git Worktree Manager (New Functions)

**Responsibilities:**
- Create isolated worktrees per agent
- Configure per-worktree git identity
- Clean up abandoned worktrees
- Manage worktree lifecycle

**Functions:**
- `setup_agent_worktree(agent_id)` - Create worktree
- `cleanup_agent_worktree(agent_id)` - Remove worktree
- `verify_worktree_isolation()` - Check independence

#### 5. Health Monitor (New Functions)

**Monitoring Dimensions:**
- **Process Health** - Is agent process alive?
- **Heartbeat** - Recent activity detected?
- **Work Progress** - Is work advancing?
- **Resource Usage** - Within limits?

**Detection:**
- Process crash (PID no longer exists)
- Heartbeat timeout (no updates for 2 minutes)
- Stale work (claimed for >2 hours)
- Resource exhaustion (OOM, disk full)

#### 6. Status Aggregator (`scripts/aggregate-status.sh` - New)

**Output:**
- Coordinator health (PID, uptime, agent count)
- Per-agent status (state, work, heartbeat, health)
- Queue statistics (available, claimed, completed, failed)
- Aggregate metrics (throughput, success rate, duration)

**File:** `state/aggregate-status.json`

### Directory Structure

```
harness/
├── loop.sh                          # Single-agent mode (Phase 2)
├── parallel-loop.sh                 # Parallel coordinator (Phase 3 NEW)
├── config.yaml                      # Extended with parallel config
│
├── state/                           # Shared coordination state
│   ├── work-queue.json              # Central work queue
│   ├── work-claimed/                # Claimed work items (atomic locks)
│   │   ├── item-123.claimed        # Lock file: points to agent marker
│   │   ├── item-123.claimed.owner  # Text file: agent-2
│   │   └── item-123.claimed.timestamp # Claim timestamp
│   │
│   ├── agents/                      # Per-agent state (isolated)
│   │   ├── agent-1/
│   │   │   ├── status.json         # Current status
│   │   │   ├── heartbeat           # Last heartbeat timestamp (epoch)
│   │   │   ├── session.json        # Active session details
│   │   │   ├── metrics.json        # Agent-specific metrics
│   │   │   ├── agent-marker        # File for atomic claims
│   │   │   ├── pid                 # Worker process PID
│   │   │   └── logs/
│   │   │       ├── stdout.log      # Agent stdout
│   │   │       └── stderr.log      # Agent stderr
│   │   │
│   │   ├── agent-2/
│   │   │   └── ... (same structure)
│   │   └── agent-N/
│   │       └── ... (same structure)
│   │
│   ├── interrupts/                  # Per-agent interrupts
│   │   ├── agent-1.interrupt       # Interrupt request from agent-1
│   │   └── agent-2.interrupt       # Interrupt request from agent-2
│   │
│   ├── aggregate-status.json        # Unified status view
│   ├── coordinator.pid              # Main coordinator PID
│   └── coordinator.log              # Coordinator activity log
│
├── worktrees/                       # Git worktrees (isolation)
│   ├── agent-1/                     # Isolated working directory
│   │   └── .git → ../../.git/worktrees/agent-1/
│   ├── agent-2/
│   └── agent-N/
│
├── scripts/
│   ├── manage-queue.sh              # UPDATED: Atomic claim support
│   ├── spawn-agent-worker.sh       # NEW: Worker spawning
│   ├── monitor-agents.sh            # NEW: Health monitoring
│   ├── aggregate-status.sh          # NEW: Status aggregation
│   └── cleanup-agent.sh             # NEW: Agent cleanup
│
└── docs/sessions/                   # Session logs by agent
    ├── agent-1/
    │   ├── ses_xxx.log
    │   └── ses_xxx.json
    ├── agent-2/
    └── agent-N/
```

### Data Flow

#### Work Claiming Flow

```
Queue Manager
    ↓
Work Queue (JSON file with available items)
    ↓
Agent Worker calls claim_work_item(item_id, agent_id)
    ↓
Atomic Operation: ln agent_marker → claim_file
    ↓
    ├─ Success? → Update queue JSON, start work
    └─ Failure? → Already claimed, try next item
```

#### Health Monitoring Flow

```
Agent Worker (every 30s)
    ↓
Update heartbeat file: date +%s > heartbeat
    ↓
Health Monitor (every 10s)
    ↓
Read all agent heartbeat files
    ↓
Compare timestamps to current time
    ↓
If age > 120s → Declare agent dead → Trigger recovery
```

#### Failure Recovery Flow

```
Health Monitor detects dead agent
    ↓
Release agent's claimed work
    ↓
Preserve crash context (logs, state)
    ↓
Cleanup agent resources (worktree, state)
    ↓
Check failure count
    ↓
    ├─ < threshold? → Spawn replacement agent
    └─ ≥ threshold? → Notify overseer, don't respawn
```

## Integration with Phase 2

### Leveraging Existing Infrastructure

**Reused Components:**
1. **Session Management** - Agent worker uses `spawn_agent()` from Phase 2
2. **Monitoring** - Stream-JSON parsing and heartbeat mechanism
3. **State Tracking** - Session state files, metrics collection
4. **Error Handling** - Timeout, stall detection, recovery logic
5. **Event Parsing** - `parse-session-events.sh` for analysis

**Enhanced Components:**
1. **Queue Manager** - Add atomic claim mechanism
2. **Bootstrap** - Add agent ID for identification
3. **Status Reporting** - Aggregate across agents
4. **Interrupt Handling** - Per-agent interrupt files

### Backward Compatibility

**Maintained:**
- Single-agent mode via `loop.sh` (unchanged)
- All Phase 2 scripts work independently
- Configuration extends (not replaces)
- Monitoring tools compatible

**Migration Path:**
```
Single Agent (Phase 2)     Parallel Agents (Phase 3)
─────────────────────────  ─────────────────────────
./loop.sh               →  ./parallel-loop.sh
  ↓                          ↓
Spawn 1 agent              Spawn N agents
  ↓                          ↓
Sequential work            Parallel work
  ↓                          ↓
4 items/hour               10 items/hour (3 agents)
```

**Rollback:**
- Keep `loop.sh` unmodified for single-agent operation
- Feature flag in `config.yaml`: `parallel_mode: true/false`
- Can switch modes by changing entry script

## Configuration Extensions

### New Configuration Schema

```yaml
harness:
  # NEW: Parallel agent configuration
  parallel_mode: true                    # Enable parallel agents
  parallel_agents: 3                     # Number of concurrent agents
  agent_startup_stagger: 5               # Seconds between agent starts
  agent_restart_delay: 10                # Delay before restarting crashed agent

  # NEW: Resource limits per agent
  resource_limits:
    max_session_memory_mb: 8192          # Per-agent memory limit
    max_cpu_percent: 80                  # Max CPU per agent (nice value)
    max_worktree_size_gb: 10             # Max worktree disk usage
    max_session_duration: 3600           # 1 hour timeout (from Phase 2)

  # NEW: Health monitoring
  health_check:
    heartbeat_interval: 30               # Agent heartbeat frequency
    heartbeat_timeout: 120               # Declare dead after N seconds
    stale_work_timeout: 7200             # Reclaim work after 2 hours
    process_check_interval: 60           # Check process existence

  # NEW: Coordinator behavior
  coordinator:
    agent_poll_interval: 10              # Check agent health every N seconds
    queue_refresh_interval: 60           # Refresh work queue
    status_update_interval: 5            # Update aggregate status
    max_agent_failures: 3                # Stop respawning after N failures
    cleanup_dead_agents: true            # Auto-cleanup dead agent resources

  # NEW: Work queue coordination
  queue:
    enable_work_affinity: true           # Agents prefer same rig
    claim_timeout: 300                   # Release claim if no progress

  # UPDATED: Interrupt handling (per-agent)
  interrupts:
    per_agent: true                      # Each agent can interrupt independently
    notify_on_any_interrupt: true        # Notify if any agent interrupts

  # UPDATED: Safety limits
  safety:
    max_consecutive_failures: 5          # Per-agent failure threshold
    max_worktrees: 10                    # System-wide worktree limit
```

### Environment Variable Overrides

```bash
# Parallel mode
PARALLEL_MODE=true ./parallel-loop.sh

# Agent count
PARALLEL_AGENTS=5 ./parallel-loop.sh

# Resource limits
AGENT_MEMORY_LIMIT=4096 ./parallel-loop.sh  # 4GB per agent

# Monitoring intervals
HEARTBEAT_INTERVAL=60 ./parallel-loop.sh    # 1 minute heartbeats
HEARTBEAT_TIMEOUT=300 ./parallel-loop.sh    # 5 minute timeout
```

## Implementation Approach

### Development Strategy

**Incremental Build:**
1. Build atomic claim mechanism first (testable in isolation)
2. Add worktree management (testable with mock agents)
3. Create agent worker script (use Phase 2 spawn logic)
4. Build coordinator (integrate all pieces)
5. Add health monitoring (layer on top)
6. Implement failure recovery (test failure scenarios)

**Testing at Each Step:**
- Unit tests for each component
- Integration tests for component pairs
- End-to-end tests for full system
- Stress tests for edge cases

**Validation Checkpoints:**
- Week 1: Atomic claims verified, worktrees working
- Week 2: Agent workers functional, coordinator spawning
- Week 3: Failure recovery working, all scenarios tested
- Week 4: Documentation complete, production ready

### Quality Gates

**Before Proceeding to Next Week:**
1. All unit tests passing (100% pass rate)
2. Integration tests passing (>95% pass rate)
3. Code review completed (peer review)
4. Documentation updated (inline + docs/)
5. No critical bugs (P0/P1 issues resolved)

**Before Production Deployment:**
1. 24-hour stability test passed
2. Throughput target met (2.5x with 3 agents)
3. All failure scenarios tested and recovered
4. Runbook completed and reviewed
5. Rollback procedure tested

## Risk Mitigation

### Identified Risks and Mitigations

#### Risk 1: Race Conditions in Work Claiming
**Severity:** High
**Probability:** Medium
**Impact:** Work claimed by multiple agents, conflicts

**Mitigation:**
- Use atomic hard link operation (`ln` command)
- Comprehensive contention testing (10 parallel claims)
- 1000-iteration stress test to verify atomicity
- Fallback: Agent retries with different work item

#### Risk 2: Git Worktree Conflicts
**Severity:** High
**Probability:** Low
**Impact:** Git corruption, lost work

**Mitigation:**
- Worktrees are fully isolated by git design
- Each worktree has unique branch
- Test concurrent operations explicitly
- Verify independence with isolation tests

#### Risk 3: Resource Exhaustion
**Severity:** Medium
**Probability:** Medium
**Impact:** System slowdown, crashes

**Mitigation:**
- Per-agent memory limits via `ulimit`
- CPU throttling via `nice`/`renice`
- Disk space monitoring (max worktree size)
- Agent count limits in configuration

#### Risk 4: Coordinator Single Point of Failure
**Severity:** Medium
**Probability:** Low
**Impact:** All agents orphaned, work stalled

**Mitigation:**
- Coordinator crash recovery on restart
- Detect stale coordinator PID
- Cleanup stale claims on recovery
- Agents continue working if coordinator crashes

#### Risk 5: Stale Work Accumulation
**Severity:** Low
**Probability:** Medium
**Impact:** Work stuck, queue grows

**Mitigation:**
- Stale work timeout (2 hours)
- Automatic reclamation
- Dead agent detection via heartbeat
- Manual cleanup tools

#### Risk 6: Integration Complexity
**Severity:** Medium
**Probability:** Low
**Impact:** Phase 2 features break, regressions

**Mitigation:**
- Maintain Phase 2 test suite (regression tests)
- Feature flag for parallel mode
- Keep `loop.sh` unmodified (single-agent fallback)
- Extensive integration testing

### Rollback Plan

**Triggers:**
- Critical bugs affecting work completion
- Resource exhaustion causing system instability
- Data corruption or lost work
- Throughput worse than single-agent mode

**Rollback Procedure:**
1. Stop parallel coordinator: `pkill -f parallel-loop.sh`
2. Kill all agent workers: `pkill -f spawn-agent-worker.sh`
3. Switch config: `parallel_mode: false`
4. Cleanup worktrees: `scripts/cleanup-agent.sh all`
5. Verify state: Check queue, release stale claims
6. Restart single-agent: `./loop.sh`
7. Monitor: Watch logs for stability

**Recovery Time:** <5 minutes

**Data Loss:** Minimal (work released back to queue)

## Success Metrics

### Performance Targets

**Throughput:**
- Baseline (1 agent): 4 items/hour
- Target (3 agents): 10 items/hour (2.5x)
- Stretch (5 agents): 15 items/hour (3.75x)

**Efficiency:**
- 3 agents: >80% efficiency (2.4x minimum)
- 5 agents: >70% efficiency (3.5x minimum)

**Overhead:**
- Coordinator CPU: <5%
- Coordinator memory: <100MB
- Queue contention: <10% time spent claiming

### Reliability Targets

**Uptime:**
- System runs 24+ hours without intervention
- Agent failures <5% of total runtime
- Work loss: 0 (all work recovered)

**Recovery:**
- Dead agent detected within 2 minutes
- Work reclaimed within 5 minutes
- Replacement agent spawned within 1 minute

**Correctness:**
- Zero race conditions in work claims (1000-iteration test)
- Zero git conflicts (concurrent operation test)
- All work either completed or in queue (no lost items)

### Quality Targets

**Testing:**
- Unit test coverage: >90%
- Integration test coverage: >80%
- All critical paths tested

**Documentation:**
- All new scripts documented (inline comments)
- User guide complete (how to run)
- Operations runbook complete (how to operate)
- Troubleshooting guide complete (how to debug)

**Code Quality:**
- Shellcheck clean (no warnings)
- Consistent style (matches Phase 1-2)
- Error handling comprehensive (all paths covered)

## Dependencies

### Technical Dependencies

**Required:**
- Git 2.5+ (worktree support)
- Bash 4.0+ (associative arrays)
- jq 1.6+ (JSON manipulation)
- coreutils (ln, stat, date)

**System Requirements:**
- Disk space: 5GB + (500MB per worktree × N agents)
- Memory: 6GB + (2GB per agent × N agents)
- CPU: 1 core + (0.5 cores per agent × N agents)

**Phase 2 Dependencies:**
- All Phase 2 scripts functional
- `spawn_agent()` working correctly
- Stream-JSON monitoring operational
- Metrics collection accurate

### External Dependencies

**None** - System uses only local filesystem coordination.

**Optional:**
- Notification system (gt mail) for alerts
- Web dashboard (future Phase 6)

## Next Steps

### Immediate Actions (This Week)

1. **Review and Approve Plan**
   - Stakeholder review of architecture
   - Approve resource allocation
   - Confirm timeline feasibility

2. **Set Up Development Environment**
   - Create Phase 3 branch
   - Set up test harness
   - Prepare isolated test environment

3. **Begin Week 1 Implementation**
   - Start with atomic claim mechanism
   - Build worktree management
   - Create test suite

### Week 1 Deliverables

See [PHASE-3-MILESTONES.md](./PHASE-3-MILESTONES.md) for detailed breakdown.

**Summary:**
- Atomic work claim mechanism verified
- Git worktree isolation working
- Agent state directory structure created
- Status aggregation functional

### Communication Plan

**Daily:**
- Standup notes in `docs/sessions/phase-3-standup.md`
- Commit messages descriptive and linked to milestones

**Weekly:**
- Progress report in `docs/sessions/phase-3-week-N.md`
- Demo of completed milestones
- Adjust plan based on learnings

**Milestone Completion:**
- Tag releases: `phase-3-milestone-1`, etc.
- Update ROADMAP.md with progress
- Notify stakeholders of progress

---

**Document Status:** Final - Ready for Implementation
**Approval Required:** System Architect, Tech Lead
**Next Document:** [PHASE-3-MILESTONES.md](./PHASE-3-MILESTONES.md)

**Author:** System Architect
**Date:** 2026-01-28
**Version:** 1.0
