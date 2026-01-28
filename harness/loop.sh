#!/usr/bin/env bash
# Claude Automation Harness - Main Loop
# Implements Ralph Wiggum pattern for continuous agent spawning

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
SCRIPTS_DIR="$HARNESS_ROOT/scripts"
PROMPTS_DIR="$HARNESS_ROOT/prompts"
DOCS_DIR="$HARNESS_ROOT/docs"

ITERATION_FILE="$STATE_DIR/iteration.log"
SESSION_FILE="$STATE_DIR/current-session.json"
QUEUE_FILE="$STATE_DIR/queue.json"
INTERRUPT_FILE="$STATE_DIR/interrupt-request.txt"

# Configuration (can be overridden via environment)
MAX_ITERATIONS=${MAX_ITERATIONS:-0}  # 0 = infinite
ITERATION_DELAY=${ITERATION_DELAY:-5}  # seconds between iterations
INTERRUPT_CHECK_INTERVAL=${INTERRUPT_CHECK_INTERVAL:-30}  # seconds
AGENT_TYPE=${AGENT_TYPE:-claude-sonnet}
SESSION_TIMEOUT=${SESSION_TIMEOUT:-3600}  # 1 hour
STALL_THRESHOLD=${STALL_THRESHOLD:-300}  # 5 minutes (no heartbeat updates)
MAX_CONSECUTIVE_FAILURES=${MAX_CONSECUTIVE_FAILURES:-5}  # Max spawn failures before interrupt

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
  echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*" | tee -a "$ITERATION_FILE"
}

log_error() {
  echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $*" | tee -a "$ITERATION_FILE" >&2
}

log_success() {
  echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS:${NC} $*" | tee -a "$ITERATION_FILE"
}

log_warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARN:${NC} $*" | tee -a "$ITERATION_FILE"
}

log_iteration() {
  local iteration="$1"
  echo "" >> "$ITERATION_FILE"
  log "====== ITERATION $iteration START ======"
}

# Initialize harness
init_harness() {
  log "Initializing Claude Automation Harness"

  # Create necessary directories
  mkdir -p "$STATE_DIR" "$DOCS_DIR"/{research,sessions,decisions}

  # Initialize queue if doesn't exist
  if [[ ! -f "$QUEUE_FILE" ]]; then
    echo "[]" > "$QUEUE_FILE"
  fi

  # Clear any stale interrupt
  rm -f "$INTERRUPT_FILE"

  # Verify dependencies
  for cmd in gt bd jq; do
    if ! command -v "$cmd" &>/dev/null; then
      log_error "Required command not found: $cmd"
      exit 1
    fi
  done

  log_success "Harness initialized"
}

# Check work queue
check_work_queue() {
  log "Checking work queue..."

  # Refresh queue
  if [[ -x "$SCRIPTS_DIR/manage-queue.sh" ]]; then
    local count
    count=$("$SCRIPTS_DIR/manage-queue.sh" check)

    if [[ "$count" -gt 0 ]]; then
      log_success "Found $count work items in queue"
      return 0
    else
      log "Queue is empty"
      return 1
    fi
  else
    log_warn "Queue manager not found, checking manually"

    # Fallback: check bd ready
    if bd ready 2>/dev/null | grep -q "No ready"; then
      log "No ready work in beads"
      return 1
    else
      log_success "Found work in beads"
      return 0
    fi
  fi
}

# Spawn Claude agent
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
  # STEP 8: Start background output processor
  # ============================================================

  # Start output processor in background
  process_session_output "$session_id" &
  local processor_pid=$!
  echo "$processor_pid" > "$STATE_DIR/${session_id}.processor.pid"

  log "Output processor started: PID=$processor_pid"

  # ============================================================
  # STEP 9: Return session metadata
  # ============================================================

  # Export for caller
  export LAST_SESSION_ID="$session_id"
  export LAST_SESSION_PID="$agent_pid"

  return 0
}

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

# Update session status
update_session_status() {
  local session_id="$1"
  local new_status="$2"
  local reason="${3:-}"

  local session_file="$SESSION_FILE"

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

# Update heartbeat
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
    message_count=$(grep -c '"type":"assistant"' "$transcript" 2>/dev/null || echo 0)
    tool_count=$(grep -c '"type":"tool_use"' "$transcript" 2>/dev/null || echo 0)
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

# Parse stream-JSON event from a line
parse_stream_event() {
  local line="$1"
  
  # Try to parse as JSON
  if ! echo "$line" | jq -e '.' >/dev/null 2>&1; then
    return 1  # Not valid JSON
  fi
  
  # Extract event type
  local event_type
  event_type=$(echo "$line" | jq -r '.type // "unknown"')
  
  echo "$event_type"
}

# Extract metrics from session
extract_session_metrics() {
  local session_id="$1"
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local metrics_file="$STATE_DIR/sessions/${session_id}/metrics.json"
  
  mkdir -p "$STATE_DIR/sessions/${session_id}"
  
  # Initialize metrics
  local total_input=0
  local total_output=0
  local total_tools=0
  local duration=0
  local turns=0
  
  # Parse transcript for usage stats
  if [[ -f "$transcript" ]]; then
    # Count turns (assistant messages)
    turns=$(grep -c '"type":"assistant"' "$transcript" 2>/dev/null || echo 0)
    
    # Count tool calls
    total_tools=$(grep -c '"type":"tool_use"' "$transcript" 2>/dev/null || echo 0)
    
    # Extract token usage (if available in transcript)
    if grep -q '"usage"' "$transcript" 2>/dev/null; then
      local usage_json
      usage_json=$(grep '"usage"' "$transcript" | jq -s 'map(.usage) | {
        total_input: map(.input_tokens // 0) | add,
        total_output: map(.output_tokens // 0) | add
      }')
      total_input=$(echo "$usage_json" | jq -r '.total_input // 0')
      total_output=$(echo "$usage_json" | jq -r '.total_output // 0')
    fi
  fi
  
  # Calculate duration from session state
  if [[ -f "$SESSION_FILE" ]]; then
    local start_epoch
    start_epoch=$(jq -r '.start_epoch // 0' "$SESSION_FILE")
    local now_epoch=$(date +%s)
    duration=$((now_epoch - start_epoch))
  fi
  
  # Extract tool usage breakdown from log
  local tool_breakdown="{}"
  if [[ -f "$log_file" ]]; then
    tool_breakdown=$(grep '"type":"tool_use"' "$log_file" 2>/dev/null | \
      jq -s 'group_by(.name) | 
      map({key: .[0].name, value: length}) | 
      from_entries' || echo '{}')
  fi
  
  # Generate metrics JSON
  jq -n \
    --argjson input_tokens "$total_input" \
    --argjson output_tokens "$total_output" \
    --argjson tool_calls "$total_tools" \
    --argjson duration_seconds "$duration" \
    --argjson turns "$turns" \
    --argjson tool_breakdown "$tool_breakdown" \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{
      session_id: "'"$session_id"'",
      collected_at: $timestamp,
      api_usage: {
        input_tokens: $input_tokens,
        output_tokens: $output_tokens,
        total_tokens: ($input_tokens + $output_tokens)
      },
      tool_usage: {
        total_calls: $tool_calls,
        breakdown: $tool_breakdown
      },
      session_metrics: {
        duration_seconds: $duration_seconds,
        turns: $turns
      }
    }' > "$metrics_file"
  
  echo "$metrics_file"
}

# Update progress indicators
update_progress() {
  local session_id="$1"
  local session_file="$SESSION_FILE"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  
  if [[ ! -f "$session_file" ]]; then
    return 1
  fi
  
  # Extract latest events from log
  if [[ ! -f "$log_file" ]]; then
    return 1
  fi
  
  # Count events by type
  local message_starts=0
  local message_stops=0
  local tool_uses=0
  local errors=0
  
  # Use tail to get recent lines (more efficient than processing entire file)
  if [[ -f "$log_file" ]]; then
    message_starts=$(grep -c '"type":"message_start"' "$log_file" 2>/dev/null || echo 0)
    message_stops=$(grep -c '"type":"message_stop"' "$log_file" 2>/dev/null || echo 0)
    tool_uses=$(grep -c '"type":"tool_use"' "$log_file" 2>/dev/null || echo 0)
    errors=$(grep -c '"type":"error"' "$log_file" 2>/dev/null || echo 0)
  fi
  
  # Update session file with progress
  jq \
    --argjson msg_starts "$message_starts" \
    --argjson msg_stops "$message_stops" \
    --argjson tools "$tool_uses" \
    --argjson errs "$errors" \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '.progress = {
      message_starts: $msg_starts,
      message_stops: $msg_stops,
      tool_calls: $tools,
      errors: $errs,
      last_updated: $timestamp
    }' \
    "$session_file" > "$session_file.tmp"
  
  mv "$session_file.tmp" "$session_file"
}

# Detect stalled session
detect_stall() {
  local session_id="$1"
  local session_file="$SESSION_FILE"
  local stall_threshold=${STALL_THRESHOLD:-300}  # 5 minutes default
  
  if [[ ! -f "$session_file" ]]; then
    return 1
  fi
  
  # Check last heartbeat time
  local last_check
  last_check=$(jq -r '.heartbeat.last_check // ""' "$session_file")
  
  if [[ -z "$last_check" ]]; then
    # No heartbeat yet, check if session is new
    local start_time
    start_time=$(jq -r '.started_at // ""' "$session_file")
    
    if [[ -n "$start_time" ]]; then
      local start_epoch
      start_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$start_time" +%s 2>/dev/null || echo 0)
      local now_epoch=$(date +%s)
      local elapsed=$((now_epoch - start_epoch))
      
      if [[ $elapsed -gt $stall_threshold ]]; then
        log_warn "Session stalled: No heartbeat for ${elapsed}s"
        return 0  # Stalled
      fi
    fi
  else
    # Check time since last heartbeat
    local check_epoch
    check_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_check" +%s 2>/dev/null || echo 0)
    local now_epoch=$(date +%s)
    local elapsed=$((now_epoch - check_epoch))
    
    if [[ $elapsed -gt $stall_threshold ]]; then
      log_warn "Session stalled: No heartbeat for ${elapsed}s (since $last_check)"
      return 0  # Stalled
    fi
  fi
  
  return 1  # Not stalled
}

# Process session output in real-time (background processor)
process_session_output() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local event_log="$STATE_DIR/sessions/${session_id}/events.jsonl"
  
  mkdir -p "$STATE_DIR/sessions/${session_id}"
  
  # Wait for log file to be created
  local wait_count=0
  while [[ ! -f "$log_file" ]] && [[ $wait_count -lt 30 ]]; do
    sleep 1
    wait_count=$((wait_count + 1))
  done
  
  if [[ ! -f "$log_file" ]]; then
    log_error "Log file never created: $log_file"
    return 1
  fi
  
  log "Starting output processor for session: $session_id"
  
  # Tail log file and process events
  tail -f "$log_file" 2>/dev/null | while IFS= read -r line; do
    # Check if session is still running
    if ! is_agent_running "$session_id"; then
      break
    fi
    
    # Try to parse as JSON event
    local event_type
    event_type=$(parse_stream_event "$line")
    
    if [[ "$event_type" == "unknown" ]] || [[ -z "$event_type" ]]; then
      continue  # Skip non-JSON or malformed lines
    fi
    
    # Log event to structured event log
    echo "$line" >> "$event_log"
    
    # Handle specific event types
    case "$event_type" in
      message_start)
        # New message starting
        ;;
        
      message_stop)
        # Message completed, update heartbeat
        update_heartbeat "$session_id"
        ;;
        
      tool_use)
        # Tool being used
        local tool_name
        tool_name=$(echo "$line" | jq -r '.name // "unknown"')
        log "Agent using tool: $tool_name"
        ;;
        
      error)
        # Error occurred
        local error_msg
        error_msg=$(echo "$line" | jq -r '.error.message // "Unknown error"')
        log_error "Agent error: $error_msg"
        
        # Write error to errors log
        echo "$line" >> "$STATE_DIR/sessions/${session_id}/errors.jsonl"
        ;;
        
      message_delta)
        # Update usage stats if present
        if echo "$line" | jq -e '.delta.usage' >/dev/null 2>&1; then
          # Usage stats in delta, could track these
          :
        fi
        ;;
    esac
  done
  
  log "Output processor ended for session: $session_id"
}

# Detect completion from stream-JSON
detect_stream_completion() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  
  if [[ ! -f "$log_file" ]]; then
    return 1
  fi
  
  # Check for message_stop events indicating turns completed
  local stops
  stops=$(grep -c '"type":"message_stop"' "$log_file" 2>/dev/null || echo 0)
  
  if [[ $stops -gt 0 ]]; then
    # Check last event stop_reason
    local last_stop
    last_stop=$(grep '"type":"message_delta"' "$log_file" 2>/dev/null | tail -1)
    
    if [[ -n "$last_stop" ]]; then
      local stop_reason
      stop_reason=$(echo "$last_stop" | jq -r '.delta.stop_reason // ""')
      
      case "$stop_reason" in
        end_turn)
          return 0  # Normal completion
          ;;
        max_tokens)
          log_warn "Session hit max tokens"
          return 0
          ;;
        stop_sequence)
          return 0
          ;;
        *)
          ;;
      esac
    fi
  fi
  
  return 1
}

# Check agent health
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

# Handle spawn failure with backoff
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

# Monitor agent session
monitor_session() {
  # Check if session file exists and is running
  if [[ ! -f "$SESSION_FILE" ]]; then
    log_warn "No active session"
    return 1
  fi

  local session_id
  session_id=$(jq -r '.session_id' "$SESSION_FILE")

  # Check if agent is still running
  if ! is_agent_running "$session_id"; then
    log "Agent no longer running: $session_id"
    return 1
  fi

  local status
  status=$(jq -r '.status' "$SESSION_FILE")

  case "$status" in
    running|spawned)
      return 0  # Continue monitoring
      ;;
    completed|failed|interrupted)
      return 1  # Session ended
      ;;
    *)
      log_warn "Unknown session status: $status"
      return 1
      ;;
  esac
}

# Check for interrupt conditions
check_interrupt() {
  # Check for explicit interrupt request
  if [[ -f "$INTERRUPT_FILE" ]]; then
    local reason
    reason=$(cat "$INTERRUPT_FILE")
    log_warn "Interrupt requested: $reason"
    return 0
  fi

  # Additional interrupt checks via helper script
  if [[ -x "$SCRIPTS_DIR/check-interrupt.sh" ]]; then
    if "$SCRIPTS_DIR/check-interrupt.sh"; then
      return 0
    fi
  fi

  # No interrupt detected
  return 1
}

# Preserve context during interrupt
preserve_context() {
  log "Preserving context for interrupt..."

  if [[ -x "$SCRIPTS_DIR/preserve-context.sh" ]]; then
    "$SCRIPTS_DIR/preserve-context.sh"
    log_success "Context preserved"
  else
    log_warn "Context preservation script not found"
  fi
}

# Wait for interrupt to be resolved
wait_for_resume() {
  log "Waiting for interrupt to be resolved..."
  log "Remove $INTERRUPT_FILE to resume"

  # Notify overseer
  if command -v gt &>/dev/null; then
    local reason
    reason=$(cat "$INTERRUPT_FILE" 2>/dev/null || echo "Unknown")
    gt mail send overseer -s "HARNESS INTERRUPT" -m "Reason: $reason" 2>/dev/null || true
  fi

  # Wait for interrupt file to be removed
  while [[ -f "$INTERRUPT_FILE" ]]; do
    sleep 10
  done

  log_success "Interrupt resolved, resuming"
}

# Prepare for next iteration
next_iteration() {
  log "Preparing next iteration..."

  # Archive session data
  if [[ -f "$SESSION_FILE" ]]; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")
    mv "$SESSION_FILE" "$DOCS_DIR/sessions/${session_id}.json"
    log "Session data archived: ${session_id}.json"
  fi

  # Clean up any temporary files
  rm -f "$STATE_DIR"/*.tmp

  log_success "Ready for next iteration"
}

# Signal handler for graceful shutdown
cleanup() {
  log ""
  log "Received shutdown signal, cleaning up..."

  # Preserve current state
  preserve_context

  # Update session status
  if [[ -f "$SESSION_FILE" ]]; then
    jq '.status = "interrupted" | .ended_at = now | .reason = "harness shutdown"' \
      "$SESSION_FILE" > "$SESSION_FILE.tmp"
    mv "$SESSION_FILE.tmp" "$SESSION_FILE"
  fi

  log_success "Harness stopped gracefully"
  exit 0
}

trap cleanup SIGINT SIGTERM

# Main loop
main() {
  init_harness

  log "Starting main loop"
  log "Configuration:"
  log "  Max iterations: ${MAX_ITERATIONS:-infinite}"
  log "  Iteration delay: ${ITERATION_DELAY}s"
  log "  Interrupt check: ${INTERRUPT_CHECK_INTERVAL}s"
  log "  Agent type: $AGENT_TYPE"
  log "  Session timeout: ${SESSION_TIMEOUT}s"
  log "  Stall threshold: ${STALL_THRESHOLD:-300}s"

  iteration=0

  while true; do
    iteration=$((iteration + 1))
    log_iteration "$iteration"

    # Check for available work
    if ! check_work_queue; then
      log "No work available, waiting ${ITERATION_DELAY}s..."
      sleep "$ITERATION_DELAY"
      continue
    fi

    # Spawn agent for work
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
      
      # Update progress indicators
      update_progress "$session_id"

      # Check health
      if ! check_agent_health "$session_id"; then
        log_error "Agent health check failed"
        break
      fi
      
      # Check for stalled session
      if detect_stall "$session_id"; then
        log_warn "Session stalled, killing agent"
        kill_agent "$session_id"
        update_session_status "$session_id" "failed" "stalled"
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
    
    # Stop output processor if still running
    local processor_pid_file="$STATE_DIR/${session_id}.processor.pid"
    if [[ -f "$processor_pid_file" ]]; then
      local processor_pid
      processor_pid=$(cat "$processor_pid_file")
      if kill -0 "$processor_pid" 2>/dev/null; then
        kill "$processor_pid" 2>/dev/null || true
      fi
      rm -f "$processor_pid_file"
    fi
    
    # Extract final metrics
    local metrics_file
    metrics_file=$(extract_session_metrics "$session_id")
    log "Metrics collected: $metrics_file"

    # Archive session
    next_iteration

    # Check iteration limit
    if [[ $MAX_ITERATIONS -gt 0 && $iteration -ge $MAX_ITERATIONS ]]; then
      log_success "Reached max iterations ($MAX_ITERATIONS), exiting"
      break
    fi

    # Delay before next iteration
    sleep "$ITERATION_DELAY"
  done

  log_success "Main loop completed"
}

# Run main loop only if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
