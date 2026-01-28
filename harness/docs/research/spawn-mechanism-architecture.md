# Agent Spawning Mechanism Architecture
**Date**: 2026-01-27
**Version**: 1.0
**Status**: Architecture Design

## Executive Summary

This document specifies the complete architecture for the agent spawning mechanism in the Claude Automation Harness. The design centers on a shell-based `spawn_agent()` function that uses `claude -p` with filesystem-based state tracking for auditability and minimal context burn.

**Key Design Decisions**:
- Shell-based implementation (bash) for consistency with Phase 1
- `claude -p` with `--output-format stream-json` for real-time monitoring
- Filesystem state for all coordination (no in-memory state)
- Bootstrap prompt injection via `--append-system-prompt-file`
- Session tracking via UUID + state files
- Graceful error handling with interrupt mechanism

---

## 1. spawn_agent() Function Design

### 1.1 Function Signature

```bash
spawn_agent() {
  # Returns: 0 on success, 1 on failure
  # Side effects:
  #   - Creates session state file
  #   - Spawns Claude Code process
  #   - Creates session log file
  #   - Updates work queue
}
```

### 1.2 Complete Implementation

```bash
#!/usr/bin/env bash
# spawn_agent() - Core agent spawning function

spawn_agent() {
  local session_id="ses_$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"
  local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  local start_epoch=$(date +%s)

  log "Spawning agent for session: $session_id"

  # ============================================================
  # STEP 1: Get work assignment
  # ============================================================

  local work_item=""
  local work_json="{}"

  if [[ -x "$SCRIPTS_DIR/manage-queue.sh" ]]; then
    work_json=$("$SCRIPTS_DIR/manage-queue.sh" next || echo '{}')

    if [[ "$work_json" == "{}" ]] || [[ -z "$work_json" ]]; then
      log_warn "No work available in queue"
      return 1
    fi

    # Extract work ID for tracking
    work_item=$(echo "$work_json" | jq -r '.id // .title // "unknown"')
    log "Assigned work: $work_item"

    # Claim work (remove from queue)
    "$SCRIPTS_DIR/manage-queue.sh" claim "$work_item" || true
  else
    log_error "Queue manager not found"
    return 1
  fi

  # ============================================================
  # STEP 2: Prepare bootstrap prompt
  # ============================================================

  local bootstrap_template="$PROMPTS_DIR/bootstrap.md"
  local bootstrap_file="/tmp/harness-bootstrap-${session_id}.md"

  if [[ ! -f "$bootstrap_template" ]]; then
    log_error "Bootstrap template not found: $bootstrap_template"
    return 1
  fi

  # Perform variable substitution
  sed \
    -e "s|{{SESSION_ID}}|${session_id}|g" \
    -e "s|{{ITERATION}}|${iteration:-0}|g" \
    -e "s|{{WORK_ITEM}}|${work_item}|g" \
    -e "s|{{RIG}}|${BD_RIG:-unknown}|g" \
    "$bootstrap_template" > "$bootstrap_file"

  log "Bootstrap prepared: $bootstrap_file"

  # ============================================================
  # STEP 3: Set up session state
  # ============================================================

  # Create initial session state file
  jq -n \
    --arg sid "$session_id" \
    --arg started "$timestamp" \
    --arg work_id "$work_item" \
    --argjson work "$work_json" \
    --arg status "spawning" \
    --argjson start_epoch "$start_epoch" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: $status,
      work: {
        id: $work_id,
        details: $work
      },
      pid: null,
      exit_code: null,
      ended_at: null,
      logs: {
        stdout: ("docs/sessions/" + $sid + ".log"),
        stderr: ("docs/sessions/" + $sid + ".err"),
        transcript: ("~/.claude/transcripts/" + $sid + ".jsonl")
      }
    }' > "$SESSION_FILE"

  log "Session state initialized: $SESSION_FILE"

  # ============================================================
  # STEP 4: Prepare environment for agent
  # ============================================================

  # Environment variables passed to Claude Code
  export SESSION_ID="$session_id"
  export HARNESS_SESSION="true"
  export INTERRUPT_FILE="$INTERRUPT_FILE"
  export BD_ACTOR="${BD_ACTOR:-harness-agent}"
  export ITERATION="${iteration:-0}"

  # Set working directory (cd to gt root)
  local work_dir="${GT_ROOT:-$HOME/gt}"

  if [[ ! -d "$work_dir" ]]; then
    log_error "Working directory not found: $work_dir"
    return 1
  fi

  # ============================================================
  # STEP 5: Build Claude Code command
  # ============================================================

  # Initial prompt for agent (what to do)
  local initial_prompt="You are spawned by the harness. Check your session context and begin work."

  # Build command array
  local -a claude_cmd=(
    claude -p "$initial_prompt"
    --session-id "$session_id"
    --output-format stream-json
    --append-system-prompt-file "$bootstrap_file"
    --allowedTools "Bash,Read,Edit,Write,Glob,Grep,mcp__serena__*"
    --max-turns 50
    --max-budget-usd 10.00
    --verbose
  )

  # Optional: Add rig-specific settings
  if [[ -f "$work_dir/.claude/settings.json" ]]; then
    claude_cmd+=(--settings "$work_dir/.claude/settings.json")
  fi

  log "Claude command prepared: ${claude_cmd[*]}"

  # ============================================================
  # STEP 6: Spawn Claude Code process
  # ============================================================

  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local err_file="$DOCS_DIR/sessions/${session_id}.err"
  local pid_file="$STATE_DIR/${session_id}.pid"

  # Ensure log directory exists
  mkdir -p "$DOCS_DIR/sessions"

  # Spawn in background with output capture
  (
    cd "$work_dir" || exit 1

    # Run Claude Code
    "${claude_cmd[@]}" \
      > "$log_file" 2> "$err_file"

    # Capture exit code
    echo $? > "$STATE_DIR/${session_id}.exit"

  ) &

  local agent_pid=$!
  echo "$agent_pid" > "$pid_file"

  log "Agent spawned: PID=$agent_pid, Session=$session_id"

  # ============================================================
  # STEP 7: Update session state with PID
  # ============================================================

  jq \
    --argjson pid "$agent_pid" \
    --arg status "running" \
    '.pid = $pid | .status = $status' \
    "$SESSION_FILE" > "$SESSION_FILE.tmp"
  mv "$SESSION_FILE.tmp" "$SESSION_FILE"

  log_success "Session running: $session_id (PID: $agent_pid)"

  # ============================================================
  # STEP 8: Return session metadata
  # ============================================================

  # Export for caller
  export LAST_SESSION_ID="$session_id"
  export LAST_SESSION_PID="$agent_pid"

  return 0
}
```

### 1.3 Helper Functions

```bash
# Check if spawned agent is still running
is_agent_running() {
  local session_id="$1"
  local pid_file="$STATE_DIR/${session_id}.pid"

  if [[ ! -f "$pid_file" ]]; then
    return 1  # No PID file = not running
  fi

  local pid
  pid=$(cat "$pid_file")

  # Check if process exists
  if kill -0 "$pid" 2>/dev/null; then
    return 0  # Running
  else
    return 1  # Not running
  fi
}

# Get agent exit code
get_agent_exit_code() {
  local session_id="$1"
  local exit_file="$STATE_DIR/${session_id}.exit"

  if [[ -f "$exit_file" ]]; then
    cat "$exit_file"
  else
    echo ""  # Unknown
  fi
}

# Kill agent gracefully
kill_agent() {
  local session_id="$1"
  local pid_file="$STATE_DIR/${session_id}.pid"

  if [[ ! -f "$pid_file" ]]; then
    log_warn "No PID file for session: $session_id"
    return 1
  fi

  local pid
  pid=$(cat "$pid_file")

  if kill -0 "$pid" 2>/dev/null; then
    log "Sending SIGTERM to agent PID $pid"
    kill -TERM "$pid"

    # Wait up to 30 seconds for graceful shutdown
    local count=0
    while kill -0 "$pid" 2>/dev/null && [[ $count -lt 30 ]]; do
      sleep 1
      count=$((count + 1))
    done

    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
      log_warn "Agent did not stop gracefully, sending SIGKILL"
      kill -KILL "$pid"
    fi

    log_success "Agent stopped: $session_id"
    return 0
  else
    log "Agent already stopped: $session_id"
    return 0
  fi
}
```

---

## 2. Session Tracking Data Structure

### 2.1 Session State File Schema

**Location**: `state/current-session.json`

```json
{
  "session_id": "ses_a1b2c3d4",
  "started_at": "2026-01-27T15:30:00Z",
  "start_epoch": 1738000200,
  "status": "running",
  "work": {
    "id": "issue-123",
    "details": {
      "id": "issue-123",
      "title": "Implement authentication module",
      "priority": 10,
      "rig": "aardwolf_snd",
      "actor": "claude-harness"
    }
  },
  "pid": 12345,
  "exit_code": null,
  "ended_at": null,
  "logs": {
    "stdout": "docs/sessions/ses_a1b2c3d4.log",
    "stderr": "docs/sessions/ses_a1b2c3d4.err",
    "transcript": "~/.claude/transcripts/ses_a1b2c3d4.jsonl"
  },
  "heartbeat": {
    "last_check": "2026-01-27T15:35:00Z",
    "message_count": 15,
    "tool_calls": 8
  },
  "interrupt": {
    "requested": false,
    "reason": null,
    "timestamp": null
  }
}
```

### 2.2 Session Lifecycle States

```
┌─────────────┐
│  spawning   │  Initial state, creating session
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   running   │  Agent actively executing
└──────┬──────┘
       │
       ├──────────────────┐
       │                  │
       ▼                  ▼
┌─────────────┐    ┌─────────────┐
│ completing  │    │   failed    │  Error occurred
└──────┬──────┘    └─────────────┘
       │
       ▼
┌─────────────┐
│  completed  │  Work finished successfully
└─────────────┘

Additional states:
┌─────────────┐
│ interrupted │  Human attention needed, paused
└─────────────┘

┌─────────────┐
│   timeout   │  Session exceeded time limit
└─────────────┘

┌─────────────┐
│   killed    │  Forcefully terminated
└─────────────┘
```

### 2.3 State Transition Logic

```bash
# Update session status
update_session_status() {
  local session_id="$1"
  local new_status="$2"
  local reason="${3:-}"

  local session_file="$STATE_DIR/current-session.json"

  if [[ ! -f "$session_file" ]]; then
    log_error "Session file not found: $session_file"
    return 1
  fi

  local current_status
  current_status=$(jq -r '.status' "$session_file")

  # Validate state transition
  case "$current_status -> $new_status" in
    "spawning -> running"|"spawning -> failed")
      ;;
    "running -> completing"|"running -> failed"|"running -> interrupted"|"running -> timeout")
      ;;
    "completing -> completed"|"completing -> failed")
      ;;
    "interrupted -> running"|"interrupted -> failed")
      ;;
    *)
      log_warn "Invalid state transition: $current_status -> $new_status"
      ;;
  esac

  # Update status
  jq \
    --arg status "$new_status" \
    --arg reason "$reason" \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '.status = $status |
     .status_reason = $reason |
     .status_updated_at = $timestamp |
     (if $status == "completed" or $status == "failed" or $status == "timeout" then
       .ended_at = $timestamp
     else . end)' \
    "$session_file" > "$session_file.tmp"

  mv "$session_file.tmp" "$session_file"

  log "Session status updated: $current_status -> $new_status"
}
```

### 2.4 Session Metadata Files

All session-related files follow this naming convention:

```
state/
  current-session.json         # Active session state
  ses_a1b2c3d4.pid            # Process ID
  ses_a1b2c3d4.exit           # Exit code (after completion)

docs/sessions/
  ses_a1b2c3d4.log            # stdout capture (stream-json events)
  ses_a1b2c3d4.err            # stderr capture (errors, warnings)
  ses_a1b2c3d4-context.json   # Preserved context (interrupts)
  ses_a1b2c3d4-summary.md     # Human-readable summary
  ses_a1b2c3d4.json           # Archived session state (after completion)

~/.claude/transcripts/
  ses_a1b2c3d4.jsonl          # Full conversation transcript (JSONL)
```

---

## 3. Bootstrap Prompt Injection Strategy

### 3.1 Variable Substitution Mechanism

The bootstrap template (`prompts/bootstrap.md`) contains placeholders that are substituted at spawn time:

**Template Variables**:
- `{{SESSION_ID}}` - Unique session identifier
- `{{ITERATION}}` - Current harness iteration number
- `{{WORK_ITEM}}` - Assigned work item ID/description
- `{{RIG}}` - Current rig name (e.g., aardwolf_snd)

**Substitution Implementation**:

```bash
# Prepare bootstrap with variable substitution
prepare_bootstrap() {
  local session_id="$1"
  local work_item="$2"
  local iteration="$3"
  local rig="$4"

  local template="$PROMPTS_DIR/bootstrap.md"
  local output="/tmp/harness-bootstrap-${session_id}.md"

  sed \
    -e "s|{{SESSION_ID}}|${session_id}|g" \
    -e "s|{{ITERATION}}|${iteration}|g" \
    -e "s|{{WORK_ITEM}}|${work_item}|g" \
    -e "s|{{RIG}}|${rig}|g" \
    "$template" > "$output"

  echo "$output"
}
```

### 3.2 Append vs Replace System Prompt

**Decision**: Use `--append-system-prompt-file` instead of `--system-prompt-file`

**Rationale**:
- Preserves Claude Code's default capabilities and tool awareness
- Adds harness-specific context on top of defaults
- Reduces risk of breaking Claude Code's built-in features
- Allows Claude Code to maintain its persona while adding harness role

**Example**:

```bash
# ✅ GOOD: Append to system prompt (recommended)
claude -p "Task" \
  --append-system-prompt-file /tmp/harness-bootstrap-ses_abc.md

# ❌ AVOID: Replace entire system prompt (risky)
claude -p "Task" \
  --system-prompt-file /tmp/harness-bootstrap-ses_abc.md
```

### 3.3 Initial Prompt Strategy

The initial prompt passed as argument to `claude -p` should be minimal and directive:

```bash
# Initial prompt (short, action-oriented)
initial_prompt="You are spawned by the harness. Check your session context and begin work."
```

**Why minimal?**
- Full context is in the appended system prompt
- Initial prompt triggers the session start
- Agent reads bootstrap to understand full assignment
- Keeps command line clean and focused

### 3.4 Bootstrap Content Structure

The bootstrap prompt should include:

1. **Identity & Role** - Who the agent is, what harness they're in
2. **Session Metadata** - Session ID, iteration, work assignment
3. **Context Building** - Where to find documentation, previous research
4. **Workflow Steps** - How to execute work (prime, check hook, etc.)
5. **Preservation Rules** - How to save research and findings
6. **Completion Criteria** - What constitutes "done"
7. **Interrupt Mechanism** - How to request human attention
8. **Environment Variables** - Available session context

---

## 4. Output Capture Approach

### 4.1 Stream-JSON Parsing

**Choice**: `--output-format stream-json` for real-time monitoring

**Stream-JSON Event Types**:

```json
{"type":"message_start","message":{"id":"...","type":"message",...},"timestamp":"..."}
{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""},"timestamp":"..."}
{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"},"timestamp":"..."}
{"type":"tool_use","id":"...","name":"Read","input":{"file_path":"..."},"timestamp":"..."}
{"type":"tool_result","tool_use_id":"...","content":"...","timestamp":"..."}
{"type":"content_block_stop","index":0,"timestamp":"..."}
{"type":"message_delta","delta":{"stop_reason":"end_turn","usage":{...}},"timestamp":"..."}
{"type":"message_stop","timestamp":"..."}
```

### 4.2 Real-Time Event Processor

```bash
# Monitor agent output in real-time
monitor_agent_output() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  tail -f "$log_file" | while IFS= read -r line; do
    # Parse JSON event
    local event_type
    event_type=$(echo "$line" | jq -r '.type // "unknown"')

    case "$event_type" in
      "message_start")
        log "Agent: Starting new message"
        ;;

      "tool_use")
        local tool_name
        tool_name=$(echo "$line" | jq -r '.name')
        log "Agent: Using tool: $tool_name"

        # Track tool usage
        echo "$line" >> "$STATE_DIR/${session_id}-tools.jsonl"
        ;;

      "message_stop")
        log "Agent: Message complete"

        # Update heartbeat
        update_heartbeat "$session_id"
        ;;

      "error")
        local error_msg
        error_msg=$(echo "$line" | jq -r '.error.message // "Unknown error"')
        log_error "Agent error: $error_msg"

        # Record error
        echo "$line" >> "$STATE_DIR/${session_id}-errors.jsonl"
        ;;

      *)
        # Ignore other event types (content_block_delta, etc.)
        ;;
    esac
  done
}
```

### 4.3 Session Log Structure

**stdout** (`ses_<id>.log`):
- Contains stream-json events, one per line
- Parseable with `jq` for real-time monitoring
- Complete record of Claude's responses and tool use

**stderr** (`ses_<id>.err`):
- Errors from Claude Code CLI itself
- Permission prompts (if any)
- Warning messages
- Diagnostic output (--verbose flag)

**transcript** (`~/.claude/transcripts/ses_<id>.jsonl`):
- Maintained by Claude Code automatically
- Full conversation history with timestamps
- Includes user messages, assistant responses, tool calls, tool results
- Survives beyond harness (persisted by Claude Code)

### 4.4 Log Aggregation Functions

```bash
# Extract all tool calls from session
get_session_tools() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  grep '"type":"tool_use"' "$log_file" | \
    jq -r '[.name, .timestamp] | @tsv'
}

# Count messages in session
count_session_messages() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  grep '"type":"message_stop"' "$log_file" | wc -l
}

# Extract errors from session
get_session_errors() {
  local session_id="$1"
  local err_file="$DOCS_DIR/sessions/${session_id}.err"

  if [[ -f "$err_file" ]]; then
    cat "$err_file"
  fi
}

# Get usage stats from transcript
get_session_usage() {
  local session_id="$1"
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"

  if [[ -f "$transcript" ]]; then
    jq -s 'map(select(.usage != null)) |
           map(.usage) |
           {
             total_input: map(.input_tokens // 0) | add,
             total_output: map(.output_tokens // 0) | add
           }' "$transcript"
  else
    echo '{"total_input":0,"total_output":0}'
  fi
}
```

---

## 5. Error Handling Patterns

### 5.1 Error Categories

```
┌─────────────────────────┬──────────────────────────┬───────────────────┐
│ Error Category          │ Example                  │ Handling Strategy │
├─────────────────────────┼──────────────────────────┼───────────────────┤
│ Spawn Failure           │ claude not in PATH       │ Log, abort spawn  │
│ Bootstrap Missing       │ bootstrap.md not found   │ Log, abort spawn  │
│ Work Queue Empty        │ No ready work            │ Log, retry later  │
│ Agent Crash             │ Segfault, OOM kill       │ Mark failed, log  │
│ Timeout                 │ Session > timeout limit  │ Kill, interrupt   │
│ Quality Gate Fail       │ Tests fail               │ Interrupt human   │
│ Explicit Error          │ Agent writes error file  │ Interrupt human   │
│ API Error               │ Rate limit, auth failure │ Pause, retry      │
│ Permission Denied       │ Tool not allowed         │ Log, continue     │
│ Interrupt Request       │ Agent needs human        │ Preserve, pause   │
└─────────────────────────┴──────────────────────────┴───────────────────┘
```

### 5.2 Error Handling Implementation

```bash
# Spawn error handling
spawn_agent() {
  # ... (initialization) ...

  # Check prerequisites
  if ! command -v claude &>/dev/null; then
    log_error "claude command not found in PATH"
    update_session_status "$session_id" "failed" "claude not installed"
    return 1
  fi

  if [[ -z "$ANTHROPIC_API_KEY" ]] && ! claude auth status &>/dev/null; then
    log_error "Not authenticated with Claude (no API key, no subscription)"
    update_session_status "$session_id" "failed" "authentication required"
    return 1
  fi

  # Try to spawn
  if ! "${claude_cmd[@]}" > "$log_file" 2> "$err_file" & then
    log_error "Failed to spawn Claude process"
    update_session_status "$session_id" "failed" "spawn failed"
    return 1
  fi

  local agent_pid=$!

  # Verify process started
  sleep 1
  if ! kill -0 "$agent_pid" 2>/dev/null; then
    log_error "Agent process died immediately after spawn"
    update_session_status "$session_id" "failed" "immediate crash"

    # Log any error output
    if [[ -f "$err_file" ]]; then
      log_error "Agent stderr: $(cat "$err_file")"
    fi

    return 1
  fi

  # Success
  echo "$agent_pid" > "$pid_file"
  update_session_status "$session_id" "running"
  return 0
}
```

### 5.3 Agent Failure Detection

```bash
# Check if agent failed
check_agent_health() {
  local session_id="$1"
  local pid_file="$STATE_DIR/${session_id}.pid"
  local exit_file="$STATE_DIR/${session_id}.exit"

  # Check if process is running
  if [[ -f "$pid_file" ]]; then
    local pid
    pid=$(cat "$pid_file")

    if ! kill -0 "$pid" 2>/dev/null; then
      # Process died
      log_warn "Agent process died: $session_id"

      # Check exit code
      if [[ -f "$exit_file" ]]; then
        local exit_code
        exit_code=$(cat "$exit_file")

        if [[ "$exit_code" -ne 0 ]]; then
          log_error "Agent exited with code: $exit_code"
          update_session_status "$session_id" "failed" "exit code $exit_code"
          return 1
        fi
      else
        log_error "Agent crashed (no exit code)"
        update_session_status "$session_id" "failed" "crash"
        return 1
      fi
    fi
  fi

  # Check for explicit error marker
  if [[ -f "$STATE_DIR/error" ]]; then
    local error_msg
    error_msg=$(cat "$STATE_DIR/error")
    log_error "Agent reported error: $error_msg"
    update_session_status "$session_id" "failed" "$error_msg"
    return 1
  fi

  # Check timeout
  local session_file="$SESSION_FILE"
  if [[ -f "$session_file" ]]; then
    local start_epoch
    start_epoch=$(jq -r '.start_epoch' "$session_file")
    local now_epoch=$(date +%s)
    local duration=$((now_epoch - start_epoch))
    local timeout=${SESSION_TIMEOUT:-3600}

    if [[ $duration -gt $timeout ]]; then
      log_warn "Session timeout: ${duration}s > ${timeout}s"
      update_session_status "$session_id" "timeout" "exceeded time limit"

      # Kill agent
      kill_agent "$session_id"

      return 1
    fi
  fi

  # Healthy
  return 0
}
```

### 5.4 Graceful Degradation

```bash
# Handle failures gracefully
handle_spawn_failure() {
  local session_id="$1"
  local failure_reason="$2"

  log_error "Spawn failure: $failure_reason"

  # Increment failure counter
  local failure_count=0
  if [[ -f "$STATE_DIR/failure-count" ]]; then
    failure_count=$(cat "$STATE_DIR/failure-count")
  fi
  failure_count=$((failure_count + 1))
  echo "$failure_count" > "$STATE_DIR/failure-count"

  # Check failure threshold
  local max_failures=${MAX_CONSECUTIVE_FAILURES:-5}
  if [[ $failure_count -ge $max_failures ]]; then
    log_error "Too many consecutive failures ($failure_count), requesting interrupt"
    echo "HARNESS: Too many spawn failures ($failure_count)" > "$INTERRUPT_FILE"
    return 1
  fi

  # Exponential backoff
  local backoff=$((2 ** failure_count))
  backoff=$((backoff > 300 ? 300 : backoff))  # Cap at 5 minutes

  log_warn "Backing off for ${backoff}s before retry"
  sleep "$backoff"

  return 0
}

# Reset failure counter on success
reset_failure_counter() {
  rm -f "$STATE_DIR/failure-count"
  log "Failure counter reset"
}
```

---

## 6. Filesystem-Based Coordination

### 6.1 State File Overview

```
state/
├── current-session.json        # Active session state
├── queue.json                  # Work queue
├── interrupt-request.txt       # Human attention needed (presence = interrupt)
├── ses_<id>.pid               # Process IDs
├── ses_<id>.exit              # Exit codes
├── ses_<id>-tools.jsonl       # Tool usage log
├── ses_<id>-errors.jsonl      # Error log
├── failure-count              # Consecutive spawn failures
├── metrics.json               # Harness metrics
└── iteration.log              # Main harness log

docs/sessions/
├── ses_<id>.log               # Session stdout (stream-json)
├── ses_<id>.err               # Session stderr
├── ses_<id>-context.json      # Preserved context
├── ses_<id>-summary.md        # Human-readable summary
└── ses_<id>.json              # Archived session state

~/.claude/transcripts/
└── ses_<id>.jsonl             # Full transcript (Claude Code managed)
```

### 6.2 Work Item State Tracking

Each work item can have an associated state file:

```
state/work/
└── issue-123.json

{
  "id": "issue-123",
  "status": "in_progress",
  "assigned_session": "ses_a1b2c3d4",
  "started_at": "2026-01-27T15:30:00Z",
  "last_updated": "2026-01-27T15:35:00Z",
  "progress": {
    "steps_completed": 3,
    "steps_total": 10,
    "current_step": "Implementing authentication logic"
  }
}
```

### 6.3 Heartbeat Mechanism

```bash
# Update heartbeat (called by monitor loop)
update_heartbeat() {
  local session_id="$1"
  local session_file="$SESSION_FILE"
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"

  if [[ ! -f "$session_file" ]]; then
    return 1
  fi

  # Count messages in transcript
  local message_count=0
  local tool_count=0

  if [[ -f "$transcript" ]]; then
    message_count=$(grep -c '"type":"assistant"' "$transcript" || echo 0)
    tool_count=$(grep -c '"type":"tool_use"' "$transcript" || echo 0)
  fi

  # Update session state
  jq \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson msg_count "$message_count" \
    --argjson tool_count "$tool_count" \
    '.heartbeat.last_check = $timestamp |
     .heartbeat.message_count = $msg_count |
     .heartbeat.tool_calls = $tool_count' \
    "$session_file" > "$session_file.tmp"

  mv "$session_file.tmp" "$session_file"
}
```

### 6.4 Completion Detection

```bash
# Detect session completion
detect_completion() {
  local session_id="$1"
  local exit_file="$STATE_DIR/${session_id}.exit"

  # Check if process has exited
  if is_agent_running "$session_id"; then
    return 1  # Still running
  fi

  # Process has stopped, check exit code
  if [[ -f "$exit_file" ]]; then
    local exit_code
    exit_code=$(cat "$exit_file")

    if [[ "$exit_code" -eq 0 ]]; then
      log_success "Session completed successfully: $session_id"
      update_session_status "$session_id" "completed"
      return 0
    else
      log_error "Session failed with exit code: $exit_code"
      update_session_status "$session_id" "failed" "exit $exit_code"
      return 1
    fi
  else
    log_warn "Session stopped but no exit code found: $session_id"
    update_session_status "$session_id" "failed" "unknown exit"
    return 1
  fi
}
```

### 6.5 Audit Trail

All operations are logged to `state/iteration.log` with timestamps:

```
[2026-01-27 15:30:00] ====== ITERATION 1 START ======
[2026-01-27 15:30:01] Checking work queue...
[2026-01-27 15:30:02] SUCCESS: Found 3 work items in queue
[2026-01-27 15:30:03] Spawning agent for session: ses_a1b2c3d4
[2026-01-27 15:30:04] Assigned work: issue-123
[2026-01-27 15:30:05] Bootstrap prepared: /tmp/harness-bootstrap-ses_a1b2c3d4.md
[2026-01-27 15:30:06] Session state initialized: state/current-session.json
[2026-01-27 15:30:07] Agent spawned: PID=12345, Session=ses_a1b2c3d4
[2026-01-27 15:30:08] SUCCESS: Session running: ses_a1b2c3d4 (PID: 12345)
...
```

**Audit Benefits**:
- Complete record of all harness operations
- Debugging failed sessions
- Performance analysis
- Compliance and accountability

---

## 7. Integration with Existing Harness

### 7.1 Integration Points

```
loop.sh (main loop)
    │
    ├─> check_work_queue()  ──> manage-queue.sh check
    │
    ├─> spawn_agent()  ──────────┐
    │   │                        │
    │   ├─> manage-queue.sh next │  (get work)
    │   ├─> prepare_bootstrap()  │  (inject context)
    │   ├─> spawn claude -p      │  (launch agent)
    │   └─> update_session_file()│  (track state)
    │                            │
    ├─> monitor_session()        │
    │   │                        │
    │   ├─> is_agent_running()   │  (check PID)
    │   ├─> update_heartbeat()   │  (track progress)
    │   └─> detect_completion()  │  (check exit)
    │                            │
    ├─> check_interrupt()  ──────┼─> check-interrupt.sh
    │   │                        │
    │   ├─> interrupt-request.txt│  (explicit request)
    │   ├─> quality-gate-failed  │  (test failures)
    │   └─> timeout detection    │  (session too long)
    │                            │
    ├─> preserve_context()  ─────┼─> preserve-context.sh
    │   │                        │
    │   └─> session-context.json │  (state snapshot)
    │                            │
    └─> next_iteration()         │
        │                        │
        └─> archive session      │  (move to docs/sessions/)
```

### 7.2 Modified loop.sh Flow

```bash
# Main loop (updated with full spawn integration)
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

    # Spawn agent (full implementation)
    if ! spawn_agent; then
      log_error "Failed to spawn agent"

      # Handle spawn failure
      if ! handle_spawn_failure "$LAST_SESSION_ID" "spawn failed"; then
        # Too many failures, interrupt
        wait_for_resume
      fi

      sleep "$ITERATION_DELAY"
      continue
    fi

    # Reset failure counter on successful spawn
    reset_failure_counter

    local session_id="$LAST_SESSION_ID"

    # Monitor agent session
    while is_agent_running "$session_id"; do
      # Update heartbeat
      update_heartbeat "$session_id"

      # Check health
      if ! check_agent_health "$session_id"; then
        log_error "Agent health check failed"
        break
      fi

      # Check for interrupt conditions
      if check_interrupt; then
        log_warn "Interrupt detected, pausing harness"

        # Kill agent gracefully
        kill_agent "$session_id"
        update_session_status "$session_id" "interrupted"

        # Preserve context
        preserve_context

        # Wait for human resolution
        wait_for_resume

        log "Resuming after interrupt"
        break  # Exit monitor loop, start next iteration
      fi

      sleep "$INTERRUPT_CHECK_INTERVAL"
    done

    # Session ended, detect completion status
    detect_completion "$session_id"

    # Archive session
    next_iteration

    # Check iteration limit
    if [[ $MAX_ITERATIONS -gt 0 && $iteration -ge $MAX_ITERATIONS ]]; then
      log_success "Reached max iterations ($MAX_ITERATIONS)"
      break
    fi

    sleep "$ITERATION_DELAY"
  done
}
```

### 7.3 Queue Integration

The `manage-queue.sh` script provides the work feed:

```bash
# In spawn_agent()
work_json=$("$SCRIPTS_DIR/manage-queue.sh" next)
work_item=$(echo "$work_json" | jq -r '.id')

# Claim work (removes from queue)
"$SCRIPTS_DIR/manage-queue.sh" claim "$work_item"
```

### 7.4 Interrupt Integration

The `check-interrupt.sh` script provides interrupt detection:

```bash
# In monitor loop
if check_interrupt; then
  # Interrupt detected
  preserve_context
  wait_for_resume
fi
```

Agents can also trigger interrupts:

```bash
# Agent writes interrupt request
echo "Need clarification on authentication approach" > "$INTERRUPT_FILE"

# Harness detects on next check_interrupt() call
```

---

## 8. Testing Strategy

### 8.1 Unit Tests

**Test spawn_agent() in isolation**:

```bash
#!/usr/bin/env bash
# test-spawn.sh

test_spawn_basic() {
  echo "Test: Basic spawn"

  # Setup
  export HARNESS_ROOT="$(pwd)/test-harness"
  mkdir -p "$HARNESS_ROOT"/{state,scripts,prompts,docs/sessions}

  # Mock queue
  echo '[{"id":"test-1","title":"Test work"}]' > "$HARNESS_ROOT/state/queue.json"

  # Mock bootstrap
  echo "# Test Bootstrap {{SESSION_ID}}" > "$HARNESS_ROOT/prompts/bootstrap.md"

  # Run spawn
  source ../loop.sh
  if spawn_agent; then
    echo "✓ Spawn succeeded"

    # Verify session file created
    if [[ -f "$HARNESS_ROOT/state/current-session.json" ]]; then
      echo "✓ Session file created"
    else
      echo "✗ Session file missing"
      return 1
    fi

    # Verify PID file
    session_id="$LAST_SESSION_ID"
    if [[ -f "$HARNESS_ROOT/state/${session_id}.pid" ]]; then
      echo "✓ PID file created"
    else
      echo "✗ PID file missing"
      return 1
    fi

  else
    echo "✗ Spawn failed"
    return 1
  fi
}

test_spawn_no_work() {
  echo "Test: Spawn with empty queue"

  # Empty queue
  echo '[]' > "$HARNESS_ROOT/state/queue.json"

  if spawn_agent; then
    echo "✗ Should have failed with empty queue"
    return 1
  else
    echo "✓ Correctly failed with empty queue"
  fi
}

test_spawn_missing_bootstrap() {
  echo "Test: Spawn with missing bootstrap"

  # Remove bootstrap
  rm -f "$HARNESS_ROOT/prompts/bootstrap.md"

  # Mock queue
  echo '[{"id":"test-2","title":"Test work"}]' > "$HARNESS_ROOT/state/queue.json"

  if spawn_agent; then
    echo "✗ Should have failed with missing bootstrap"
    return 1
  else
    echo "✓ Correctly failed with missing bootstrap"
  fi
}

# Run tests
test_spawn_basic
test_spawn_no_work
test_spawn_missing_bootstrap
```

### 8.2 Integration Tests

**Test full loop with mock agent**:

```bash
#!/usr/bin/env bash
# test-integration.sh

# Mock Claude Code CLI
mock_claude() {
  cat > /tmp/mock-claude <<'EOF'
#!/usr/bin/env bash
# Mock Claude CLI for testing

# Simulate agent behavior
echo '{"type":"message_start","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'
sleep 2
echo '{"type":"tool_use","name":"Read","input":{"file_path":"test.txt"}}'
sleep 2
echo '{"type":"message_stop","timestamp":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}'

exit 0
EOF
  chmod +x /tmp/mock-claude
  export PATH="/tmp:$PATH"
}

test_full_iteration() {
  echo "Test: Full iteration with mock agent"

  mock_claude

  # Setup harness
  export MAX_ITERATIONS=1
  export ITERATION_DELAY=1

  # Run loop
  timeout 30 ./loop.sh

  # Check results
  if [[ -f "docs/sessions/ses_*.json" ]]; then
    echo "✓ Session archived"
  else
    echo "✗ Session not archived"
    return 1
  fi
}

test_full_iteration
```

### 8.3 Error Scenario Tests

```bash
# Test agent crash
test_agent_crash() {
  # Mock Claude that exits immediately with error
  cat > /tmp/mock-claude <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
  chmod +x /tmp/mock-claude

  if spawn_agent; then
    sleep 2
    if check_agent_health "$LAST_SESSION_ID"; then
      echo "✗ Should have detected crash"
      return 1
    else
      echo "✓ Detected crash"
    fi
  fi
}

# Test timeout
test_agent_timeout() {
  export SESSION_TIMEOUT=5

  # Mock Claude that runs forever
  cat > /tmp/mock-claude <<'EOF'
#!/usr/bin/env bash
while true; do sleep 1; done
EOF
  chmod +x /tmp/mock-claude

  spawn_agent
  sleep 7

  if check_agent_health "$LAST_SESSION_ID"; then
    echo "✗ Should have detected timeout"
    return 1
  else
    echo "✓ Detected timeout"
  fi
}
```

### 8.4 Testing Checkpoints

**Pre-spawn**:
- [ ] Queue has work items
- [ ] Bootstrap template exists
- [ ] Working directory is valid
- [ ] Claude CLI is available

**Post-spawn**:
- [ ] Session state file created
- [ ] PID file created
- [ ] Process is running
- [ ] Log files created

**During monitoring**:
- [ ] Heartbeat updates every interval
- [ ] Tool usage tracked
- [ ] Interrupts detected
- [ ] Timeouts detected

**Post-completion**:
- [ ] Exit code captured
- [ ] Session state updated
- [ ] Logs archived
- [ ] Work queue updated

---

## 9. Example Session Lifecycle Walkthrough

### 9.1 Complete Session Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    ITERATION 1 START                            │
└─────────────────────────────────────────────────────────────────┘

[15:30:00] Checking work queue...
[15:30:01] SUCCESS: Found 3 work items in queue

┌─────────────────────────────────────────────────────────────────┐
│                        SPAWN AGENT                              │
└─────────────────────────────────────────────────────────────────┘

[15:30:02] Spawning agent for session: ses_a1b2c3d4
[15:30:03] Assigned work: issue-123 (Implement authentication)
[15:30:04] Bootstrap prepared with variables:
           - SESSION_ID: ses_a1b2c3d4
           - ITERATION: 1
           - WORK_ITEM: issue-123
           - RIG: aardwolf_snd

[15:30:05] Session state initialized:
           state/current-session.json
           {
             "session_id": "ses_a1b2c3d4",
             "status": "spawning",
             "work": {"id": "issue-123", ...},
             "started_at": "2026-01-27T15:30:05Z"
           }

[15:30:06] Environment prepared:
           - SESSION_ID=ses_a1b2c3d4
           - HARNESS_SESSION=true
           - BD_ACTOR=harness-agent
           - Working directory: ~/gt

[15:30:07] Claude command built:
           claude -p "You are spawned by the harness..." \
             --session-id ses_a1b2c3d4 \
             --output-format stream-json \
             --append-system-prompt-file /tmp/harness-bootstrap-ses_a1b2c3d4.md \
             --allowedTools "Bash,Read,Edit,Write,Glob,Grep,mcp__serena__*" \
             --max-turns 50 \
             --max-budget-usd 10.00 \
             --verbose

[15:30:08] Process spawned: PID=12345

[15:30:09] Session state updated:
           {
             "status": "running",
             "pid": 12345,
             ...
           }

[15:30:10] SUCCESS: Session running (PID: 12345)

┌─────────────────────────────────────────────────────────────────┐
│                      MONITOR SESSION                            │
└─────────────────────────────────────────────────────────────────┘

[15:30:10] Monitoring session: ses_a1b2c3d4

[15:30:15] Agent: Using tool: Read (gt prime)
[15:30:20] Agent: Using tool: Read (bd hook)
[15:30:25] Heartbeat updated: 3 messages, 5 tool calls
[15:30:40] Agent: Using tool: Edit (implement auth)
[15:31:10] Heartbeat updated: 8 messages, 12 tool calls
[15:31:40] Agent: Using tool: Bash (run tests)
[15:32:10] Heartbeat updated: 12 messages, 18 tool calls

[15:32:45] Agent process exited (PID: 12345)
[15:32:46] Exit code: 0

┌─────────────────────────────────────────────────────────────────┐
│                    DETECT COMPLETION                            │
└─────────────────────────────────────────────────────────────────┘

[15:32:47] SUCCESS: Session completed successfully
[15:32:48] Session state updated:
           {
             "status": "completed",
             "exit_code": 0,
             "ended_at": "2026-01-27T15:32:47Z"
           }

[15:32:49] Usage stats:
           - Input tokens: 45,230
           - Output tokens: 8,450
           - Tool calls: 18
           - Duration: 2m 39s

┌─────────────────────────────────────────────────────────────────┐
│                      NEXT ITERATION                             │
└─────────────────────────────────────────────────────────────────┘

[15:32:50] Archiving session data:
           - state/current-session.json
             → docs/sessions/ses_a1b2c3d4.json
           - Logs preserved:
             • docs/sessions/ses_a1b2c3d4.log (stdout)
             • docs/sessions/ses_a1b2c3d4.err (stderr)
             • ~/.claude/transcripts/ses_a1b2c3d4.jsonl (transcript)

[15:32:51] Cleaning up temp files:
           - /tmp/harness-bootstrap-ses_a1b2c3d4.md (removed)
           - state/ses_a1b2c3d4.pid (removed)
           - state/ses_a1b2c3d4.exit (removed)

[15:32:52] SUCCESS: Ready for next iteration

[15:32:57] Waiting 5s before next iteration...

┌─────────────────────────────────────────────────────────────────┐
│                    ITERATION 2 START                            │
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 Interrupt Scenario

```
[15:45:30] Agent: Using tool: Bash (run tests)
[15:45:35] Agent: Tests failed

[15:45:40] Agent writes: state/interrupt-request.txt
           Content: "Tests failing, need to check main branch health"

[15:45:50] Interrupt check triggered
[15:45:51] INTERRUPT: Tests failing, need to check main branch health

┌─────────────────────────────────────────────────────────────────┐
│                    PRESERVE CONTEXT                             │
└─────────────────────────────────────────────────────────────────┘

[15:45:52] Preserving context for session: ses_a1b2c3d4
[15:45:53] Killing agent gracefully (PID: 12345)
[15:45:54] Agent stopped
[15:45:55] Session state updated: "interrupted"

[15:45:56] Context saved:
           - docs/sessions/ses_a1b2c3d4-context.json
           - docs/sessions/ses_a1b2c3d4-summary.md
           - docs/sessions/ses_a1b2c3d4-beads.json

[15:45:57] Notifying overseer via gt mail...
[15:45:58] Mail sent to overseer: "HARNESS INTERRUPT"

┌─────────────────────────────────────────────────────────────────┐
│                    WAIT FOR RESUME                              │
└─────────────────────────────────────────────────────────────────┘

[15:45:59] Waiting for interrupt to be resolved...
[15:46:00] Remove state/interrupt-request.txt to resume

... (harness paused) ...

[16:10:00] Interrupt file removed by human
[16:10:01] SUCCESS: Interrupt resolved, resuming

┌─────────────────────────────────────────────────────────────────┐
│                    ITERATION 2 START                            │
└─────────────────────────────────────────────────────────────────┘
```

### 9.3 Filesystem State During Session

**state/** directory during active session:

```
state/
├── current-session.json        # Active session metadata
│   └── { session_id, status: "running", pid: 12345, ... }
│
├── queue.json                  # Remaining work items
│   └── [{"id": "issue-124", ...}, {"id": "issue-125", ...}]
│
├── ses_a1b2c3d4.pid           # Process ID: 12345
├── ses_a1b2c3d4-tools.jsonl   # Tool usage log (real-time)
├── iteration.log               # Main harness log
└── metrics.json                # Harness metrics
```

**docs/sessions/** during active session:

```
docs/sessions/
├── ses_a1b2c3d4.log           # stdout (stream-json events, growing)
└── ses_a1b2c3d4.err           # stderr (errors, warnings)
```

**~/.claude/transcripts/** (managed by Claude Code):

```
~/.claude/transcripts/
└── ses_a1b2c3d4.jsonl         # Full conversation (growing)
```

**After session completion**:

```
state/
├── queue.json                  # Updated (issue-123 removed)
└── iteration.log               # Updated with completion

docs/sessions/
├── ses_a1b2c3d4.json          # Archived session state (final)
├── ses_a1b2c3d4.log           # Complete stdout log
├── ses_a1b2c3d4.err           # Complete stderr log
└── ses_a1b2c3d4-summary.md    # Human-readable summary (if interrupted)

~/.claude/transcripts/
└── ses_a1b2c3d4.jsonl         # Complete transcript (persists)
```

---

## 10. Implementation Roadmap

### Phase 1: Core Spawn Function (Days 1-2)

**Tasks**:
- [ ] Implement complete `spawn_agent()` function
- [ ] Add session state management functions
- [ ] Implement bootstrap variable substitution
- [ ] Test basic spawning with mock work

**Deliverables**:
- Working `spawn_agent()` function in `loop.sh`
- Session state JSON schema implemented
- Unit tests passing

### Phase 2: Monitoring & Health Checks (Days 3-4)

**Tasks**:
- [ ] Implement stream-JSON output capture
- [ ] Build heartbeat mechanism
- [ ] Add health check logic
- [ ] Implement timeout detection

**Deliverables**:
- Real-time monitoring of agent sessions
- Health checks integrated into main loop
- Timeout handling working

### Phase 3: Error Handling (Days 5-6)

**Tasks**:
- [ ] Implement all error detection paths
- [ ] Add graceful failure handling
- [ ] Build backoff/retry logic
- [ ] Test error scenarios

**Deliverables**:
- Comprehensive error handling
- Graceful degradation
- Error test suite passing

### Phase 4: Integration & Testing (Days 7-8)

**Tasks**:
- [ ] Integrate with existing scripts
- [ ] End-to-end testing with real Claude Code
- [ ] Performance tuning
- [ ] Documentation updates

**Deliverables**:
- Fully integrated harness
- Integration tests passing
- Updated documentation

---

## Conclusion

This architecture provides a complete, testable design for spawning and managing Claude Code agents within the automation harness. The design emphasizes:

1. **Filesystem-based state** for auditability and debugging
2. **Shell-based implementation** for consistency and simplicity
3. **Minimal context burn** via lazy documentation loading
4. **Graceful error handling** with interrupt mechanism
5. **Real-time monitoring** via stream-JSON parsing
6. **Integration** with existing harness components

The implementation is straightforward and builds directly on the CLI research findings, providing a clear path from design to working code.

---

**Document Status**: Complete
**Ready for Implementation**: Yes
**Next Step**: Begin Phase 1 (spawn_agent() implementation)
