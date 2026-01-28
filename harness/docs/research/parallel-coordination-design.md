# Parallel Agent Coordination Framework Design

**Document Version:** 1.0
**Status:** Draft
**Date:** 2026-01-27
**Author:** System Architect
**Phase:** 3 Planning (follows Phase 2 agent spawning)

## Executive Summary

This document specifies a lock-free, filesystem-based coordination framework for running N parallel Claude agents within the automation harness. The design prioritizes **simplicity**, **observability**, and **failure isolation** while maintaining the shell-based philosophy of Phase 1.

**Key Design Principles:**
- Lock-free coordination via atomic filesystem operations
- Zero shared mutable state between agents
- Each agent operates in isolated git worktree
- Filesystem-based status tracking (audit-friendly)
- Graceful degradation on agent failure
- Resource limits enforced per agent

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Harness Coordinator                          │
│  (Main Process - loop.sh with --parallel mode)                      │
│                                                                       │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │ Work Queue     │  │ Agent Pool     │  │ Health Monitor │        │
│  │ Manager        │  │ Manager        │  │                │        │
│  └────────────────┘  └────────────────┘  └────────────────┘        │
└───────────────────────────────────┬─────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────────┐
        │                           │                               │
        ▼                           ▼                               ▼
┌───────────────┐           ┌───────────────┐           ┌───────────────┐
│  Agent 1      │           │  Agent 2      │           │  Agent N      │
│  (Worker      │           │  (Worker      │           │  (Worker      │
│   Process)    │           │   Process)    │           │   Process)    │
│               │           │               │           │               │
│ ┌───────────┐ │           │ ┌───────────┐ │           │ ┌───────────┐ │
│ │ Worktree  │ │           │ │ Worktree  │ │           │ │ Worktree  │ │
│ │   1/      │ │           │ │   2/      │ │           │ │   N/      │ │
│ └───────────┘ │           │ └───────────┘ │           │ └───────────┘ │
│               │           │               │           │               │
│ ┌───────────┐ │           │ ┌───────────┐ │           │ ┌───────────┐ │
│ │ State     │ │           │ │ State     │ │           │ │ State     │ │
│ │ agent-1/  │ │           │ │ agent-2/  │ │           │ │ agent-N/  │ │
│ └───────────┘ │           │ └───────────┘ │           │ └───────────┘ │
└───────────────┘           │───────────────┘           └───────────────┘
        │                           │                               │
        └───────────────────────────┼───────────────────────────────┘
                                    │
                                    ▼
                        ┌───────────────────────┐
                        │   Shared Resources    │
                        │  (Read-Only Access)   │
                        │                       │
                        │ • config.yaml         │
                        │ • prompts/            │
                        │ • docs/research/      │
                        │ • work-queue.json     │
                        └───────────────────────┘
```

### Key Components

1. **Harness Coordinator** - Single main process managing agent lifecycle
2. **Work Queue** - Centralized queue with atomic claim mechanism
3. **Agent Workers** - Independent processes in isolated environments
4. **Health Monitor** - Watchdog for detecting dead/hung agents
5. **Status Aggregator** - Unified view across all agents

## Directory Structure

```
harness/
├── loop.sh                         # Single-agent mode (Phase 1)
├── parallel-loop.sh                # Parallel coordinator (Phase 3)
├── config.yaml                     # Extended with parallel config
├── state/                          # Shared coordination state
│   ├── work-queue.json             # Central work queue
│   ├── work-claimed/               # Claimed work items (atomic)
│   │   ├── item-123.claimed       # Lock file: agent-2
│   │   └── item-456.claimed       # Lock file: agent-1
│   ├── agents/                     # Per-agent state
│   │   ├── agent-1/
│   │   │   ├── status.json        # Current status
│   │   │   ├── heartbeat          # Last heartbeat timestamp
│   │   │   ├── session.json       # Active session details
│   │   │   └── metrics.json       # Agent-specific metrics
│   │   ├── agent-2/
│   │   │   └── ... (same structure)
│   │   └── agent-N/
│   │       └── ... (same structure)
│   ├── interrupts/                 # Per-agent interrupts
│   │   ├── agent-1.interrupt      # Interrupt request from agent-1
│   │   └── agent-2.interrupt      # Interrupt request from agent-2
│   ├── aggregate-status.json       # Unified status view
│   └── coordinator.pid             # Main coordinator PID
├── worktrees/                      # Git worktrees for isolation
│   ├── agent-1/                    # Isolated working directory
│   │   └── .git -> ../../.git/worktrees/agent-1/
│   ├── agent-2/
│   └── agent-N/
├── scripts/
│   ├── manage-queue.sh             # Enhanced with parallel support
│   ├── spawn-agent-worker.sh      # NEW: Worker spawning
│   ├── monitor-agents.sh           # NEW: Health monitoring
│   ├── aggregate-status.sh         # NEW: Status aggregation
│   └── cleanup-agent.sh            # NEW: Agent cleanup
└── docs/
    └── sessions/                   # Session logs by agent
        ├── agent-1/
        ├── agent-2/
        └── agent-N/
```

## Work Queue Partitioning Strategy

### Queue Data Structure

```json
{
  "version": "1.0",
  "last_refresh": "2026-01-27T10:30:00Z",
  "items": [
    {
      "id": "item-123",
      "type": "issue",
      "source": "beads",
      "rig": "aardwolf_snd",
      "priority": "high",
      "title": "Fix authentication bug",
      "claimed_by": null,
      "claimed_at": null,
      "status": "available"
    },
    {
      "id": "item-456",
      "type": "feature",
      "source": "gt",
      "rig": "duneagent",
      "priority": "medium",
      "title": "Add user settings page",
      "claimed_by": "agent-2",
      "claimed_at": "2026-01-27T10:28:15Z",
      "status": "claimed"
    }
  ]
}
```

### Atomic Claim Mechanism

**Approach: File-Based Locking with Link Creation**

Why `ln` instead of `touch` or `echo`?
- `ln` (hard link) creation is **atomic** at filesystem level
- Either succeeds (link created) or fails (link exists)
- No race condition window
- Works reliably on macOS HFS+/APFS

**Claim Protocol:**

```bash
# In manage-queue.sh

claim_work_item() {
  local item_id="$1"
  local agent_id="$2"
  local claim_dir="$STATE_DIR/work-claimed"
  local claim_file="$claim_dir/${item_id}.claimed"
  local agent_marker="$STATE_DIR/agents/${agent_id}/agent-marker"

  mkdir -p "$claim_dir"

  # Create agent marker if doesn't exist
  touch "$agent_marker"

  # Atomic claim via hard link
  if ln "$agent_marker" "$claim_file" 2>/dev/null; then
    # SUCCESS: We got the lock
    echo "$agent_id" > "${claim_file}.owner"
    date -u +%Y-%m-%dT%H:%M:%SZ > "${claim_file}.timestamp"

    # Update queue JSON (non-atomic, but only writer is this agent)
    update_queue_claimed "$item_id" "$agent_id"

    echo "claimed"
    return 0
  else
    # FAILURE: Already claimed by another agent
    echo "already-claimed"
    return 1
  fi
}

release_work_item() {
  local item_id="$1"
  local claim_dir="$STATE_DIR/work-claimed"

  # Release claim
  rm -f "$claim_dir/${item_id}.claimed"
  rm -f "$claim_dir/${item_id}.claimed.owner"
  rm -f "$claim_dir/${item_id}.claimed.timestamp"

  # Update queue
  update_queue_released "$item_id"
}
```

### Work Distribution Strategies

**Strategy 1: Priority-Based Pull (Recommended)**

Each agent independently pulls highest-priority available work:

```bash
get_next_work() {
  local agent_id="$1"

  # Get available items sorted by priority
  local items
  items=$(jq -r '.items[]
    | select(.status == "available")
    | {id, priority, title}
    | @json' "$QUEUE_FILE" \
    | sort -t: -k2 -r)  # Sort by priority

  # Try to claim items in priority order
  while IFS= read -r item; do
    local item_id
    item_id=$(echo "$item" | jq -r '.id')

    if claim_work_item "$item_id" "$agent_id"; then
      echo "$item"
      return 0
    fi
  done <<< "$items"

  return 1  # No work available
}
```

**Advantages:**
- Simple to implement
- Naturally load-balances (idle agents grab work)
- Respects priority ordering
- No central scheduler needed

**Strategy 2: Rig Affinity (Optional Enhancement)**

Agents prefer work for rigs they recently worked on:

```bash
get_next_work_with_affinity() {
  local agent_id="$1"
  local last_rig
  last_rig=$(jq -r '.last_rig // ""' "$STATE_DIR/agents/$agent_id/metrics.json")

  # Try same rig first (context still fresh)
  if [[ -n "$last_rig" ]]; then
    try_claim_rig_work "$agent_id" "$last_rig" && return 0
  fi

  # Fall back to any available work
  get_next_work "$agent_id"
}
```

## Agent Instance Isolation

### Git Worktree Setup

Each agent gets isolated working directory via `git worktree`:

```bash
# In parallel-loop.sh

setup_agent_worktree() {
  local agent_id="$1"
  local worktree_dir="$HARNESS_ROOT/worktrees/$agent_id"
  local branch="harness-agent-${agent_id}-$(date +%Y%m%d)"

  log "Setting up worktree for $agent_id"

  # Create worktree from main
  if [[ ! -d "$worktree_dir" ]]; then
    git worktree add "$worktree_dir" -b "$branch" main
    log_success "Created worktree: $worktree_dir"
  else
    log "Worktree exists: $worktree_dir"
  fi

  # Configure worktree git
  cd "$worktree_dir"
  git config user.name "Claude Agent ${agent_id}"
  git config user.email "agent-${agent_id}@gastown.local"
  cd - > /dev/null
}

cleanup_agent_worktree() {
  local agent_id="$1"
  local worktree_dir="$HARNESS_ROOT/worktrees/$agent_id"
  local branch="harness-agent-${agent_id}-"

  log "Cleaning up worktree for $agent_id"

  # Remove worktree
  if [[ -d "$worktree_dir" ]]; then
    git worktree remove "$worktree_dir" --force
    log_success "Removed worktree: $worktree_dir"
  fi

  # Prune old branches (keep last 3)
  git branch --list "${branch}*" \
    | sort -r \
    | tail -n +4 \
    | xargs -r git branch -D
}
```

### State Isolation

Each agent has completely isolated state:

```bash
# Agent state directory structure
state/agents/agent-1/
├── status.json          # Current status
├── heartbeat            # Timestamp of last heartbeat
├── session.json         # Active session details
├── metrics.json         # Performance metrics
├── agent-marker         # For atomic claims
└── logs/
    ├── stdout.log       # Agent stdout
    └── stderr.log       # Agent stderr
```

**No Shared Mutable State:**
- Each agent writes only to its own `state/agents/agent-X/` directory
- Queue updates via atomic claim mechanism
- Read-only access to shared resources (config, docs)

## Lock-Free Coordination

### Coordination Primitives

**1. Atomic Claim (via hard links)**
```bash
# Claim work item
ln "$agent_marker" "$claim_file"
```

**2. Heartbeat (via timestamp files)**
```bash
# Agent writes heartbeat
date +%s > "$STATE_DIR/agents/$agent_id/heartbeat"

# Coordinator checks liveness
check_agent_alive() {
  local agent_id="$1"
  local heartbeat_file="$STATE_DIR/agents/$agent_id/heartbeat"
  local now
  now=$(date +%s)
  local last_heartbeat
  last_heartbeat=$(cat "$heartbeat_file" 2>/dev/null || echo 0)

  local age=$((now - last_heartbeat))

  if [[ $age -gt 120 ]]; then
    # No heartbeat for 2+ minutes
    return 1
  fi
  return 0
}
```

**3. Status Updates (via atomic file writes)**
```bash
# Agent updates own status
update_agent_status() {
  local agent_id="$1"
  local status="$2"
  local status_file="$STATE_DIR/agents/$agent_id/status.json"

  jq -n \
    --arg status "$status" \
    --arg updated "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{status: $status, updated_at: $updated}' \
    > "${status_file}.tmp"

  mv "${status_file}.tmp" "$status_file"
}
```

**4. Interrupt Signaling (via presence files)**
```bash
# Agent requests interrupt
signal_interrupt() {
  local agent_id="$1"
  local reason="$2"

  echo "$reason" > "$STATE_DIR/interrupts/${agent_id}.interrupt"
}

# Coordinator detects interrupts
check_interrupts() {
  local interrupts
  interrupts=($(ls "$STATE_DIR/interrupts/"*.interrupt 2>/dev/null || true))

  if [[ ${#interrupts[@]} -gt 0 ]]; then
    for interrupt_file in "${interrupts[@]}"; do
      local agent_id
      agent_id=$(basename "$interrupt_file" .interrupt)
      local reason
      reason=$(cat "$interrupt_file")

      handle_agent_interrupt "$agent_id" "$reason"
    done
  fi
}
```

### Coordination Protocol

```
┌──────────────┐
│ Agent Starts │
└──────┬───────┘
       │
       ▼
┌──────────────────────┐
│ Write Status:        │
│   "initializing"     │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Claim Work Item      │  ◄──── Atomic (ln command)
│ (via hard link)      │
└──────┬───────────────┘
       │
       │ success?
       ├─ No ──► Back to queue
       │
       ▼ Yes
┌──────────────────────┐
│ Update Status:       │
│   "working"          │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Start Heartbeat      │  ◄──── Every 30s
│ Background Task      │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Execute Work         │
└──────┬───────────────┘
       │
       │ interrupt?
       ├─ Yes ──► Signal Interrupt
       │           Wait for Clear
       │
       ▼ No
┌──────────────────────┐
│ Release Work Item    │  ◄──── Remove claim file
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Update Status:       │
│   "idle"             │
└──────┬───────────────┘
       │
       └──► Back to Claim Work
```

## Resource Management

### Configuration Extensions

```yaml
# config.yaml additions

harness:
  # Parallel agent configuration
  parallel_agents: 3                    # Number of concurrent agents
  agent_startup_stagger: 5              # Seconds between agent starts

  # Resource limits per agent
  resource_limits:
    max_session_memory_mb: 8192         # Per-agent memory limit
    max_cpu_percent: 80                 # Max CPU per agent
    max_worktree_size_gb: 10            # Max worktree disk usage
    max_session_duration: 3600          # 1 hour timeout

  # Health monitoring
  health_check:
    heartbeat_interval: 30              # Agent heartbeat frequency
    heartbeat_timeout: 120              # Declare dead after N seconds
    stale_work_timeout: 7200            # Reclaim work after 2 hours

  # Coordinator behavior
  coordinator:
    agent_poll_interval: 10             # Check agent health every N seconds
    queue_refresh_interval: 60          # Refresh work queue every N seconds
    status_update_interval: 5           # Update aggregate status every N seconds
```

### Resource Enforcement

**Memory Limits (via ulimit):**

```bash
spawn_agent_worker() {
  local agent_id="$1"
  local memory_limit_mb
  memory_limit_mb=$(get_config "harness.resource_limits.max_session_memory_mb")

  # Convert MB to KB for ulimit
  local memory_limit_kb=$((memory_limit_mb * 1024))

  # Launch agent with resource limits
  (
    # Set memory limit
    ulimit -v "$memory_limit_kb"

    # Set CPU time limit (10 hours)
    ulimit -t 36000

    # Execute agent worker
    exec "$SCRIPTS_DIR/agent-worker.sh" "$agent_id" \
      >> "$STATE_DIR/agents/$agent_id/logs/stdout.log" 2>&1
  ) &

  local pid=$!
  echo "$pid" > "$STATE_DIR/agents/$agent_id/pid"
}
```

**CPU Throttling (via nice/renice):**

```bash
# Set lower priority for agent processes
spawn_agent_worker() {
  # ... setup ...

  # Launch with lower priority
  nice -n 10 "$SCRIPTS_DIR/agent-worker.sh" "$agent_id" &

  local pid=$!

  # Further throttle if needed
  renice +5 -p "$pid" > /dev/null
}
```

**Disk Space Monitoring:**

```bash
check_worktree_size() {
  local agent_id="$1"
  local worktree_dir="$HARNESS_ROOT/worktrees/$agent_id"
  local max_size_gb
  max_size_gb=$(get_config "harness.resource_limits.max_worktree_size_gb")

  local current_size_gb
  current_size_gb=$(du -sg "$worktree_dir" | cut -f1)

  if [[ $current_size_gb -gt $max_size_gb ]]; then
    log_warn "Agent $agent_id worktree exceeds size limit: ${current_size_gb}GB > ${max_size_gb}GB"
    return 1
  fi

  return 0
}
```

### Agent Pool Management

```bash
# In parallel-loop.sh

manage_agent_pool() {
  local target_count
  target_count=$(get_config "harness.parallel_agents")
  local current_count
  current_count=$(count_active_agents)

  if [[ $current_count -lt $target_count ]]; then
    # Scale up
    local needed=$((target_count - current_count))
    log "Scaling up: need $needed more agents"

    for ((i=1; i<=needed; i++)); do
      spawn_new_agent
      sleep "$AGENT_STARTUP_STAGGER"
    done

  elif [[ $current_count -gt $target_count ]]; then
    # Scale down (graceful)
    local excess=$((current_count - target_count))
    log "Scaling down: removing $excess agents"

    terminate_idle_agents "$excess"
  fi
}

spawn_new_agent() {
  local agent_id
  agent_id=$(find_available_agent_id)

  log "Spawning agent: $agent_id"

  # Setup isolation
  setup_agent_worktree "$agent_id"
  mkdir -p "$STATE_DIR/agents/$agent_id/logs"

  # Spawn worker process
  spawn_agent_worker "$agent_id"

  log_success "Agent $agent_id spawned (PID: $(cat "$STATE_DIR/agents/$agent_id/pid"))"
}
```

## Failure Isolation

### Failure Categories

1. **Agent Crash** - Process dies unexpectedly
2. **Agent Hang** - No heartbeat for timeout period
3. **Work Failure** - Work item fails quality gates
4. **Resource Exhaustion** - OOM, disk full, etc.
5. **Coordinator Failure** - Main process crashes

### Detection Mechanisms

**1. Process Monitoring:**

```bash
check_agent_process() {
  local agent_id="$1"
  local pid_file="$STATE_DIR/agents/$agent_id/pid"

  if [[ ! -f "$pid_file" ]]; then
    return 1
  fi

  local pid
  pid=$(cat "$pid_file")

  if ! kill -0 "$pid" 2>/dev/null; then
    # Process not running
    return 1
  fi

  return 0
}
```

**2. Heartbeat Monitoring:**

```bash
monitor_agent_health() {
  local timeout
  timeout=$(get_config "harness.health_check.heartbeat_timeout")

  for agent_dir in "$STATE_DIR/agents"/*; do
    local agent_id
    agent_id=$(basename "$agent_dir")

    if ! check_agent_alive "$agent_id"; then
      log_error "Agent $agent_id appears dead (no heartbeat)"
      handle_dead_agent "$agent_id"
    fi
  done
}
```

**3. Stale Work Detection:**

```bash
check_stale_work() {
  local stale_timeout
  stale_timeout=$(get_config "harness.health_check.stale_work_timeout")
  local now
  now=$(date +%s)

  for claim_file in "$STATE_DIR/work-claimed"/*.claimed; do
    [[ -f "$claim_file" ]] || continue

    local claim_time
    claim_time=$(stat -f %m "$claim_file")
    local age=$((now - claim_time))

    if [[ $age -gt $stale_timeout ]]; then
      local item_id
      item_id=$(basename "$claim_file" .claimed)

      log_warn "Work item $item_id is stale (${age}s old)"
      reclaim_stale_work "$item_id"
    fi
  done
}
```

### Recovery Strategies

**Agent Crash Recovery:**

```bash
handle_dead_agent() {
  local agent_id="$1"

  log "Handling dead agent: $agent_id"

  # 1. Release any claimed work
  release_agent_work "$agent_id"

  # 2. Preserve crash context
  preserve_agent_context "$agent_id" "crashed"

  # 3. Cleanup resources
  cleanup_agent "$agent_id"

  # 4. Spawn replacement (if within failure threshold)
  local failures
  failures=$(get_agent_failure_count "$agent_id")

  if [[ $failures -lt 3 ]]; then
    log "Respawning agent $agent_id (failure count: $failures)"
    spawn_new_agent
  else
    log_error "Agent $agent_id exceeded failure threshold, not respawning"
    notify_overseer "Agent $agent_id failed repeatedly"
  fi
}

release_agent_work() {
  local agent_id="$1"

  # Find work claimed by this agent
  for claim_file in "$STATE_DIR/work-claimed"/*.claimed.owner; do
    [[ -f "$claim_file" ]] || continue

    local owner
    owner=$(cat "$claim_file")

    if [[ "$owner" == "$agent_id" ]]; then
      local item_id
      item_id=$(basename "$claim_file" .claimed.owner)

      log "Releasing work item $item_id from dead agent $agent_id"
      release_work_item "$item_id"
    fi
  done
}
```

**Work Item Failure Handling:**

```bash
handle_work_failure() {
  local agent_id="$1"
  local item_id="$2"
  local failure_reason="$3"

  log_error "Work item $item_id failed on agent $agent_id: $failure_reason"

  # Update work item status
  jq --arg id "$item_id" --arg reason "$failure_reason" \
    '.items |= map(
      if .id == $id then
        .status = "failed" |
        .failure_reason = $reason |
        .failed_at = now
      else . end
    )' "$QUEUE_FILE" > "$QUEUE_FILE.tmp"
  mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"

  # Decide retry strategy
  local retry_count
  retry_count=$(get_work_retry_count "$item_id")

  if [[ $retry_count -lt 2 ]]; then
    # Retry on different agent
    log "Marking work item $item_id for retry (attempt $((retry_count + 1)))"
    mark_work_for_retry "$item_id"
  else
    # Park for human review
    log "Work item $item_id exceeded retry limit, parking"
    park_work_item "$item_id"
    notify_overseer "Work item $item_id failed after retries"
  fi
}

park_work_item() {
  local item_id="$1"
  local park_dir="$STATE_DIR/parked-work"

  mkdir -p "$park_dir"

  # Move work item to parked queue
  jq --arg id "$item_id" \
    '.items[] | select(.id == $id)' \
    "$QUEUE_FILE" > "$park_dir/${item_id}.json"

  # Remove from active queue
  jq --arg id "$item_id" \
    '.items |= map(select(.id != $id))' \
    "$QUEUE_FILE" > "$QUEUE_FILE.tmp"
  mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"
}
```

**Coordinator Failure Recovery:**

```bash
# On coordinator startup, recover from crash

recover_from_crash() {
  log "Checking for previous coordinator crash..."

  local coordinator_pid_file="$STATE_DIR/coordinator.pid"

  if [[ -f "$coordinator_pid_file" ]]; then
    local old_pid
    old_pid=$(cat "$coordinator_pid_file")

    if kill -0 "$old_pid" 2>/dev/null; then
      log_error "Coordinator already running (PID: $old_pid)"
      exit 1
    fi

    log_warn "Previous coordinator crashed (PID: $old_pid)"

    # Cleanup stale state
    cleanup_stale_claims
    reset_agent_states

    log_success "Crash recovery complete"
  fi

  # Write new coordinator PID
  echo "$$" > "$coordinator_pid_file"
}

cleanup_stale_claims() {
  log "Cleaning up stale work claims..."

  # Release all claims (agents will reclaim if still working)
  for claim_file in "$STATE_DIR/work-claimed"/*.claimed; do
    [[ -f "$claim_file" ]] || continue

    local item_id
    item_id=$(basename "$claim_file" .claimed)

    log "Releasing stale claim: $item_id"
    rm -f "$STATE_DIR/work-claimed/${item_id}."*
  done
}
```

## Progress Aggregation

### Status Aggregation

**Aggregate Status Schema:**

```json
{
  "timestamp": "2026-01-27T10:45:00Z",
  "coordinator": {
    "pid": 12345,
    "started_at": "2026-01-27T08:00:00Z",
    "uptime_seconds": 9900
  },
  "agents": [
    {
      "id": "agent-1",
      "status": "working",
      "pid": 12346,
      "work_item": "item-123",
      "started_at": "2026-01-27T10:28:15Z",
      "last_heartbeat": "2026-01-27T10:44:45Z",
      "health": "healthy"
    },
    {
      "id": "agent-2",
      "status": "idle",
      "pid": 12347,
      "work_item": null,
      "last_heartbeat": "2026-01-27T10:44:50Z",
      "health": "healthy"
    },
    {
      "id": "agent-3",
      "status": "working",
      "pid": 12348,
      "work_item": "item-789",
      "started_at": "2026-01-27T10:35:20Z",
      "last_heartbeat": "2026-01-27T10:44:40Z",
      "health": "healthy"
    }
  ],
  "work_queue": {
    "total": 15,
    "available": 12,
    "claimed": 3,
    "completed_today": 8,
    "failed_today": 1,
    "parked": 2
  },
  "metrics": {
    "agents_active": 3,
    "agents_working": 2,
    "agents_idle": 1,
    "agents_interrupted": 0,
    "throughput_per_hour": 4.2,
    "average_work_duration": 850,
    "success_rate": 0.89
  }
}
```

**Aggregation Script:**

```bash
#!/usr/bin/env bash
# scripts/aggregate-status.sh

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
AGGREGATE_FILE="$STATE_DIR/aggregate-status.json"

aggregate_agent_status() {
  local agents_json="[]"

  for agent_dir in "$STATE_DIR/agents"/*; do
    [[ -d "$agent_dir" ]] || continue

    local agent_id
    agent_id=$(basename "$agent_dir")

    # Read agent state
    local status_file="$agent_dir/status.json"
    local heartbeat_file="$agent_dir/heartbeat"
    local session_file="$agent_dir/session.json"
    local pid_file="$agent_dir/pid"

    # Build agent status object
    local agent_status
    agent_status=$(jq -n \
      --arg id "$agent_id" \
      --arg status "$(jq -r '.status // "unknown"' "$status_file" 2>/dev/null || echo "unknown")" \
      --arg pid "$(cat "$pid_file" 2>/dev/null || echo "null")" \
      --arg work "$(jq -r '.work_item // null' "$session_file" 2>/dev/null || echo "null")" \
      --arg started "$(jq -r '.started_at // null' "$session_file" 2>/dev/null || echo "null")" \
      --arg heartbeat "$(date -r "$(cat "$heartbeat_file" 2>/dev/null || echo 0)" -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "null")" \
      --arg health "$(check_agent_health "$agent_id")" \
      '{
        id: $id,
        status: $status,
        pid: ($pid | tonumber),
        work_item: $work,
        started_at: $started,
        last_heartbeat: $heartbeat,
        health: $health
      }')

    agents_json=$(echo "$agents_json" | jq --argjson agent "$agent_status" '. += [$agent]')
  done

  echo "$agents_json"
}

aggregate_queue_stats() {
  jq '{
    total: (.items | length),
    available: ([.items[] | select(.status == "available")] | length),
    claimed: ([.items[] | select(.status == "claimed")] | length),
    completed_today: ([.items[] | select(.status == "completed" and (.completed_at | startswith("'$(date +%Y-%m-%d)'")))] | length),
    failed_today: ([.items[] | select(.status == "failed" and (.failed_at | startswith("'$(date +%Y-%m-%d)'")))] | length)
  }' "$STATE_DIR/work-queue.json"
}

# Generate aggregate status
jq -n \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson coordinator "$(get_coordinator_info)" \
  --argjson agents "$(aggregate_agent_status)" \
  --argjson queue "$(aggregate_queue_stats)" \
  --argjson metrics "$(calculate_metrics)" \
  '{
    timestamp: $timestamp,
    coordinator: $coordinator,
    agents: $agents,
    work_queue: $queue,
    metrics: $metrics
  }' > "$AGGREGATE_FILE"

echo "Aggregate status updated: $AGGREGATE_FILE"
```

### Metrics Collection

```bash
calculate_metrics() {
  local now
  now=$(date +%s)
  local one_hour_ago=$((now - 3600))

  # Count completed work in last hour
  local completed_last_hour
  completed_last_hour=$(jq --arg threshold "$one_hour_ago" \
    '[.items[] | select(.status == "completed" and (.completed_at | fromdate) > ($threshold | tonumber))] | length' \
    "$STATE_DIR/work-queue.json")

  # Calculate success rate
  local total_finished
  total_finished=$(jq '[.items[] | select(.status == "completed" or .status == "failed")] | length' "$STATE_DIR/work-queue.json")
  local total_completed
  total_completed=$(jq '[.items[] | select(.status == "completed")] | length' "$STATE_DIR/work-queue.json")

  local success_rate
  if [[ $total_finished -gt 0 ]]; then
    success_rate=$(echo "scale=2; $total_completed / $total_finished" | bc)
  else
    success_rate="null"
  fi

  # Average work duration
  local avg_duration
  avg_duration=$(jq '[.items[] | select(.status == "completed" and .duration != null) | .duration] | add / length' "$STATE_DIR/work-queue.json" 2>/dev/null || echo "null")

  jq -n \
    --argjson active "$(count_active_agents)" \
    --argjson working "$(count_working_agents)" \
    --argjson idle "$(count_idle_agents)" \
    --argjson interrupted "$(count_interrupted_agents)" \
    --argjson throughput "$completed_last_hour" \
    --argjson avg_duration "$avg_duration" \
    --argjson success_rate "$success_rate" \
    '{
      agents_active: $active,
      agents_working: $working,
      agents_idle: $idle,
      agents_interrupted: $interrupted,
      throughput_per_hour: $throughput,
      average_work_duration: $avg_duration,
      success_rate: $success_rate
    }'
}
```

### Status Display

```bash
# Enhanced scripts/report-status.sh

display_parallel_status() {
  local status_file="$STATE_DIR/aggregate-status.json"

  if [[ ! -f "$status_file" ]]; then
    echo "No status available"
    return 1
  fi

  echo "═══════════════════════════════════════════════════════════"
  echo "  Claude Automation Harness - Parallel Status"
  echo "═══════════════════════════════════════════════════════════"
  echo ""

  # Coordinator info
  echo "Coordinator:"
  jq -r '.coordinator | "  PID: \(.pid) | Uptime: \(.uptime_seconds)s"' "$status_file"
  echo ""

  # Agents table
  echo "Agents:"
  echo "  ID       Status      Work Item    Health    Last Heartbeat"
  echo "  ───────  ──────────  ───────────  ────────  ──────────────"
  jq -r '.agents[] | "  \(.id)  \(.status | ljust(10))  \(.work_item // "none" | ljust(11))  \(.health | ljust(8))  \(.last_heartbeat)"' "$status_file"
  echo ""

  # Queue stats
  echo "Work Queue:"
  jq -r '.work_queue | "  Total: \(.total) | Available: \(.available) | Claimed: \(.claimed) | Completed Today: \(.completed_today) | Failed: \(.failed_today)"' "$status_file"
  echo ""

  # Metrics
  echo "Metrics:"
  jq -r '.metrics | "  Active: \(.agents_active) | Working: \(.agents_working) | Idle: \(.agents_idle) | Throughput: \(.throughput_per_hour)/hr | Success Rate: \(.success_rate * 100)%"' "$status_file"
  echo ""

  echo "═══════════════════════════════════════════════════════════"
}
```

## Implementation Roadmap

### Phase 3.1: Coordination Infrastructure (Week 1)

**Tasks:**
1. Extend `config.yaml` with parallel configuration
2. Implement `manage-queue.sh` atomic claim mechanism
3. Create agent state directory structure
4. Build `aggregate-status.sh` script
5. Test atomic claim under contention

**Deliverables:**
- ✅ Atomic work claiming verified
- ✅ Agent state isolation working
- ✅ Status aggregation functional

**Success Criteria:**
- 10 parallel claim attempts = exactly 1 success
- No race conditions observed in 1000 iterations

### Phase 3.2: Worktree Isolation (Week 1)

**Tasks:**
1. Implement `setup_agent_worktree()` function
2. Test worktree creation/cleanup
3. Verify git isolation between worktrees
4. Handle worktree cleanup on failures

**Deliverables:**
- ✅ Worktree creation automated
- ✅ Agent isolation verified
- ✅ Cleanup handles edge cases

**Success Criteria:**
- N agents can work simultaneously without git conflicts
- Worktree cleanup leaves no orphans

### Phase 3.3: Agent Worker Process (Week 2)

**Tasks:**
1. Create `scripts/spawn-agent-worker.sh`
2. Implement agent lifecycle (claim → work → release)
3. Add heartbeat mechanism
4. Build interrupt handling
5. Test resource limits (ulimit)

**Deliverables:**
- ✅ Agent worker script functional
- ✅ Heartbeat monitoring working
- ✅ Resource limits enforced

**Success Criteria:**
- Agent runs complete lifecycle without crashes
- Heartbeat detects dead agents within 2 minutes
- Memory limits prevent OOM on host

### Phase 3.4: Parallel Coordinator (Week 2)

**Tasks:**
1. Create `parallel-loop.sh` main coordinator
2. Implement agent pool management
3. Build health monitoring daemon
4. Add scale-up/scale-down logic
5. Implement coordinator crash recovery

**Deliverables:**
- ✅ Coordinator spawns N agents
- ✅ Health monitoring detects failures
- ✅ Crash recovery restores state

**Success Criteria:**
- Coordinator maintains N agents continuously
- Dead agents replaced within 1 minute
- Coordinator recovers from crash without data loss

### Phase 3.5: Failure Handling (Week 3)

**Tasks:**
1. Implement stale work detection
2. Build agent crash recovery
3. Add work item retry logic
4. Create parked work queue
5. Test failure scenarios

**Deliverables:**
- ✅ Crash recovery working
- ✅ Stale work reclaimed
- ✅ Failed work handled gracefully

**Success Criteria:**
- Agent crashes don't lose work
- Stale work reclaimed within timeout
- Failed work items park after 3 retries

### Phase 3.6: Integration Testing (Week 3)

**Tasks:**
1. Test 1 agent (baseline)
2. Test 3 agents (normal operation)
3. Test 10 agents (stress test)
4. Simulate agent crashes
5. Simulate coordinator crash
6. Measure throughput gains

**Deliverables:**
- ✅ Test suite covering failure modes
- ✅ Performance benchmarks documented
- ✅ Known issues documented

**Success Criteria:**
- 3 agents = 2.5x throughput vs 1 agent (allowing overhead)
- System runs 24 hours without intervention
- All failure modes recover automatically

### Phase 3.7: Documentation & Rollout (Week 4)

**Tasks:**
1. Update main README with parallel mode
2. Document configuration options
3. Create troubleshooting guide
4. Write runbook for operations
5. Staged rollout plan

**Deliverables:**
- ✅ Complete parallel mode documentation
- ✅ Operations runbook
- ✅ Troubleshooting guide

## Testing Strategy

### Unit Tests

**1. Atomic Claim Mechanism:**

```bash
# test/test-atomic-claim.sh

test_atomic_claim() {
  echo "Testing atomic work claim..."

  # Setup: 1 work item, 10 parallel claimers
  setup_test_queue "item-test-001"

  # Launch 10 parallel claim attempts
  for i in {1..10}; do
    (claim_work_item "item-test-001" "agent-$i" > "claim-result-$i.txt") &
  done

  wait

  # Verify: exactly 1 success
  local success_count=0
  for i in {1..10}; do
    if grep -q "claimed" "claim-result-$i.txt"; then
      ((success_count++))
    fi
  done

  if [[ $success_count -eq 1 ]]; then
    echo "✅ Atomic claim test passed"
    return 0
  else
    echo "❌ Atomic claim test failed: $success_count successes"
    return 1
  fi
}
```

**2. Heartbeat Monitoring:**

```bash
test_heartbeat_detection() {
  echo "Testing heartbeat detection..."

  # Start agent with heartbeat
  start_test_agent "agent-test-hb"

  # Wait 30 seconds
  sleep 30

  # Verify agent alive
  if check_agent_alive "agent-test-hb"; then
    echo "✅ Agent detected as alive"
  else
    echo "❌ Agent incorrectly detected as dead"
    return 1
  fi

  # Stop heartbeat
  kill_agent_heartbeat "agent-test-hb"

  # Wait 130 seconds (past timeout)
  sleep 130

  # Verify agent dead
  if ! check_agent_alive "agent-test-hb"; then
    echo "✅ Dead agent detected"
    return 0
  else
    echo "❌ Dead agent not detected"
    return 1
  fi
}
```

**3. Worktree Isolation:**

```bash
test_worktree_isolation() {
  echo "Testing worktree isolation..."

  # Create 2 agent worktrees
  setup_agent_worktree "agent-wt-1"
  setup_agent_worktree "agent-wt-2"

  # Make changes in worktree 1
  cd "$HARNESS_ROOT/worktrees/agent-wt-1"
  echo "test" > test-file.txt
  git add test-file.txt
  git commit -m "Test commit from agent-wt-1"

  # Verify worktree 2 unaffected
  cd "$HARNESS_ROOT/worktrees/agent-wt-2"
  if [[ -f test-file.txt ]]; then
    echo "❌ Worktrees not isolated"
    return 1
  fi

  echo "✅ Worktrees properly isolated"
  return 0
}
```

### Integration Tests

**1. Parallel Work Processing:**

```bash
test_parallel_processing() {
  echo "Testing parallel work processing..."

  # Setup: 9 work items, 3 agents
  setup_test_queue_multi 9

  # Start coordinator with 3 agents
  start_coordinator 3

  # Wait for completion (max 10 minutes)
  local timeout=600
  local elapsed=0

  while [[ $elapsed -lt $timeout ]]; do
    local remaining
    remaining=$(jq '[.items[] | select(.status == "available")] | length' "$QUEUE_FILE")

    if [[ $remaining -eq 0 ]]; then
      echo "✅ All work completed in ${elapsed}s"
      return 0
    fi

    sleep 10
    ((elapsed += 10))
  done

  echo "❌ Timeout: work not completed in ${timeout}s"
  return 1
}
```

**2. Failure Recovery:**

```bash
test_agent_crash_recovery() {
  echo "Testing agent crash recovery..."

  # Start 3 agents with work
  start_coordinator 3

  # Wait for agents to claim work
  sleep 30

  # Kill agent 2
  local pid
  pid=$(cat "$STATE_DIR/agents/agent-2/pid")
  kill -9 "$pid"

  echo "Killed agent-2 (PID: $pid)"

  # Wait for coordinator to detect and recover
  sleep 150  # heartbeat timeout + recovery time

  # Verify:
  # 1. Agent 2's work was released
  # 2. New agent was spawned
  # 3. Work was reclaimed

  local new_agent_count
  new_agent_count=$(count_active_agents)

  if [[ $new_agent_count -eq 3 ]]; then
    echo "✅ Agent crash recovery successful"
    return 0
  else
    echo "❌ Recovery failed: only $new_agent_count agents active"
    return 1
  fi
}
```

**3. Throughput Measurement:**

```bash
test_throughput_scaling() {
  echo "Testing throughput scaling..."

  # Baseline: 10 work items with 1 agent
  setup_test_queue_multi 10
  start_coordinator 1

  local start_time
  start_time=$(date +%s)

  wait_for_queue_empty

  local end_time
  end_time=$(date +%s)
  local baseline_duration=$((end_time - start_time))

  echo "Baseline (1 agent): ${baseline_duration}s"

  # Test: 10 work items with 3 agents
  setup_test_queue_multi 10
  start_coordinator 3

  start_time=$(date +%s)
  wait_for_queue_empty
  end_time=$(date +%s)

  local parallel_duration=$((end_time - start_time))

  echo "Parallel (3 agents): ${parallel_duration}s"

  # Calculate speedup
  local speedup
  speedup=$(echo "scale=2; $baseline_duration / $parallel_duration" | bc)

  echo "Speedup: ${speedup}x"

  # Expect at least 2x speedup (allowing for overhead)
  if (( $(echo "$speedup >= 2.0" | bc -l) )); then
    echo "✅ Throughput scaling achieved"
    return 0
  else
    echo "❌ Insufficient speedup: ${speedup}x < 2.0x"
    return 1
  fi
}
```

### Stress Tests

**1. High Contention:**

```bash
test_high_contention() {
  echo "Testing high contention scenario..."

  # 100 work items, 10 agents
  setup_test_queue_multi 100
  start_coordinator 10

  # Monitor for race conditions
  local errors=0

  while ! queue_empty; do
    # Check for duplicate claims
    if detect_duplicate_claims; then
      echo "❌ Race condition detected: duplicate claims"
      ((errors++))
    fi

    sleep 5
  done

  if [[ $errors -eq 0 ]]; then
    echo "✅ No race conditions under high contention"
    return 0
  else
    echo "❌ Detected $errors race conditions"
    return 1
  fi
}
```

**2. Long-Running Stability:**

```bash
test_24_hour_stability() {
  echo "Testing 24-hour stability..."

  # Start coordinator with continuous work stream
  start_coordinator 3
  start_work_generator  # Adds new work periodically

  # Run for 24 hours
  local duration=$((24 * 3600))
  sleep "$duration"

  # Check health
  local agents_alive
  agents_alive=$(count_active_agents)

  local work_completed
  work_completed=$(count_completed_work)

  if [[ $agents_alive -eq 3 ]] && [[ $work_completed -gt 0 ]]; then
    echo "✅ 24-hour stability test passed"
    echo "   Agents alive: $agents_alive"
    echo "   Work completed: $work_completed"
    return 0
  else
    echo "❌ Stability test failed"
    return 1
  fi
}
```

## Configuration Reference

### Complete Parallel Configuration

```yaml
# config.yaml - Full parallel configuration

harness:
  # Loop control
  max_iterations: 0                    # 0 = infinite
  iteration_delay: 5                   # seconds between coordinator iterations
  interrupt_check_interval: 30         # seconds between interrupt checks

  # Agent spawning
  agent_type: claude-sonnet            # Model to use
  session_timeout: 3600                # Max session duration (1 hour)

  # Parallel configuration
  parallel_mode: true                  # Enable parallel agents
  parallel_agents: 3                   # Number of concurrent agents
  agent_startup_stagger: 5             # Seconds between agent starts
  agent_restart_delay: 10              # Delay before restarting crashed agent

  # Work routing
  default_rig: aardwolf_snd
  rig_priority:
    - aardwolf_snd
    - duneagent

  # Work queue
  queue:
    refresh_interval: 60               # Refresh from sources every N seconds
    max_queue_size: 100
    priority_weights:
      high: 10
      medium: 5
      low: 1
    enable_work_affinity: true         # Agents prefer same rig

  # Resource limits per agent
  resource_limits:
    max_session_memory_mb: 8192        # 8GB per agent
    max_cpu_percent: 80                # Max 80% CPU per agent
    max_worktree_size_gb: 10           # 10GB max per worktree
    max_session_duration: 3600         # 1 hour timeout
    max_concurrent_git_ops: 5          # Limit parallel git operations

  # Health monitoring
  health_check:
    heartbeat_interval: 30             # Agent writes heartbeat every 30s
    heartbeat_timeout: 120             # Declare dead after 120s no heartbeat
    stale_work_timeout: 7200           # Reclaim work after 2 hours
    process_check_interval: 60         # Check process existence every 60s

  # Coordinator behavior
  coordinator:
    agent_poll_interval: 10            # Check agent health every 10s
    queue_refresh_interval: 60         # Refresh work queue every 60s
    status_update_interval: 5          # Update aggregate status every 5s
    max_agent_failures: 3              # Stop respawning after N failures
    cleanup_dead_agents: true          # Auto-cleanup dead agent resources

  # Interrupts
  interrupts:
    quality_gate_failure: true
    blocked_work: true
    approval_required: true
    ambiguous_requirement: true
    session_timeout: true
    error_threshold: 3                 # Per-agent error threshold

  # Work failure handling
  work_retry:
    max_retries: 2                     # Retry failed work 2 times
    retry_delay: 300                   # Wait 5 minutes before retry
    park_after_failure: true           # Park work after max retries
    notify_on_park: true               # Notify overseer when work parked

  # Logging (per-agent)
  logging:
    level: info
    coordinator_log: state/coordinator.log
    agent_logs: state/agents/{agent_id}/logs/
    keep_logs_days: 30
    rotate_logs: true

  # Safety limits
  safety:
    max_consecutive_failures: 5        # Per-agent
    max_session_memory_mb: 8192
    require_tests_pass: true
    require_clean_git: true
    max_worktrees: 10                  # System-wide limit

# Monitoring & observability
monitoring:
  metrics:
    enabled: true
    aggregate_metrics: state/aggregate-metrics.json
    per_agent_metrics: true

  track:
    - agents_active
    - agents_working
    - agents_idle
    - work_items_completed
    - work_items_failed
    - average_session_duration
    - throughput_per_hour
    - success_rate
    - resource_utilization

  dashboard:
    enabled: false                     # Future: web dashboard
    port: 8080
    refresh_interval: 10

  alerts:
    enable_alerts: true
    alert_on_all_agents_dead: true
    alert_on_queue_stall: true         # No progress for 1 hour
    alert_on_high_failure_rate: true   # >50% failure rate
```

## Security Considerations

### Isolation Guarantees

1. **Filesystem Isolation**
   - Each agent has dedicated worktree (no shared working directory)
   - Agent state directories have restrictive permissions (700)
   - Claim files prevent cross-agent work conflicts

2. **Process Isolation**
   - Each agent runs as separate process
   - Resource limits via ulimit prevent resource exhaustion
   - Process crash doesn't affect other agents

3. **Git Isolation**
   - Worktrees prevent branch conflicts
   - Each agent has own git identity
   - Push operations are serialized via git's internal locking

### Attack Surface

**Minimal:**
- No network services (filesystem-only coordination)
- No shared memory segments
- No IPC mechanisms (signals, sockets, pipes)
- Coordinator runs with same privileges as user

**Potential Issues:**
- Disk exhaustion from worktrees (mitigated by size limits)
- Fork bomb from infinite respawning (mitigated by failure threshold)
- Race conditions in claim mechanism (mitigated by atomic operations)

## Performance Characteristics

### Expected Throughput

**Baseline (1 agent):**
- ~4 work items/hour (15min average per item)

**Parallel (3 agents):**
- ~10 work items/hour (2.5x throughput)
- 83% efficiency (overhead: queue contention, coordinator, git ops)

**Parallel (5 agents):**
- ~15 work items/hour (3.75x throughput)
- 75% efficiency (diminishing returns due to shared resources)

### Resource Usage

**Per Agent:**
- Memory: ~4-6GB (Claude Code + context)
- CPU: ~60% single core (bursty during git operations)
- Disk: ~500MB worktree + ~200MB state

**Coordinator:**
- Memory: ~50MB (mostly bash + jq)
- CPU: <5% (mostly idle, periodic status checks)
- Disk: ~10MB state files

**Total (3 agents):**
- Memory: ~15GB
- CPU: ~2 cores (average)
- Disk: ~2.5GB

### Bottlenecks

1. **Git Operations** - Pushes are serialized by remote
2. **Queue Refresh** - Beads/gt CLI calls are sequential
3. **Coordinator** - Single point of coordination (minimal impact)

**Mitigations:**
- Git: Batched pushes, local work before pushing
- Queue: Cached queue with periodic refresh
- Coordinator: Async health checks, minimal blocking operations

## Future Enhancements

### Phase 4: Advanced Coordination

1. **Work Stealing**
   - Idle agents steal work from busy agents' queues
   - Enables better load balancing

2. **Dependency-Aware Scheduling**
   - Agents coordinate on dependent work items
   - Prevents blocking on dependencies

3. **Dynamic Scaling**
   - Auto-scale agent count based on queue depth
   - Scale down during idle periods

### Phase 5: Performance Optimization

1. **Persistent Worktrees**
   - Reuse worktrees across sessions
   - Reduces setup overhead

2. **Context Caching**
   - Share read-only context between agents
   - Reduces memory footprint

3. **Batched Git Operations**
   - Agents accumulate changes, push in batches
   - Reduces git overhead

### Phase 6: Observability Enhancements

1. **Web Dashboard**
   - Real-time agent status visualization
   - Interactive queue management
   - Performance metrics graphs

2. **Structured Logging**
   - JSON logs for machine parsing
   - Integration with log aggregators (Loki, etc.)

3. **Metrics Export**
   - Prometheus exporter
   - Grafana dashboards
   - Alert manager integration

## Conclusion

This parallel coordination framework provides a robust foundation for scaling the Claude automation harness from 1 to N agents while maintaining the simplicity and observability of the Phase 1 design.

**Key Strengths:**
- ✅ Lock-free coordination (atomic filesystem operations)
- ✅ Complete agent isolation (worktrees + state directories)
- ✅ Comprehensive failure handling (crash recovery, work retry, parking)
- ✅ Observable operation (aggregate status, per-agent metrics)
- ✅ Resource management (memory limits, CPU throttling, disk monitoring)
- ✅ Graceful scaling (dynamic agent pool management)

**Design Philosophy Maintained:**
- Simple, debuggable bash implementation
- File-based state and coordination
- Human-readable formats (JSON, logs)
- Minimal dependencies
- Easy to inspect and troubleshoot

**Implementation Timeline:**
- Phase 3.1-3.2: Week 1 (infrastructure + isolation)
- Phase 3.3-3.4: Week 2 (worker + coordinator)
- Phase 3.5-3.6: Week 3 (failure handling + testing)
- Phase 3.7: Week 4 (documentation + rollout)

**Total Effort:** 4 weeks from Phase 3 start to production deployment

---

**Document Status:** Ready for implementation
**Next Steps:**
1. Review design with stakeholders
2. Begin Phase 3.1 implementation
3. Set up testing environment
4. Create detailed implementation tickets

**Questions/Feedback:** Contact System Architect team
