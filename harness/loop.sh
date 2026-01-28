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
  local session_id="session-$(date +%Y%m%d-%H%M%S)-$$"
  local work_item=""

  log "Spawning agent for session: $session_id"

  # Get next work item
  if [[ -x "$SCRIPTS_DIR/manage-queue.sh" ]]; then
    work_item=$("$SCRIPTS_DIR/manage-queue.sh" next)
    log "Assigned work: $work_item"
  fi

  # Record session start
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg work "$work_item" \
    '{
      session_id: $sid,
      started_at: $started,
      status: "running",
      work_item: $work,
      pid: null
    }' > "$SESSION_FILE"

  # Build agent prompt
  local prompt_file="$PROMPTS_DIR/bootstrap.md"
  if [[ ! -f "$prompt_file" ]]; then
    log_error "Bootstrap prompt not found: $prompt_file"
    return 1
  fi

  # Prepare environment for agent
  export SESSION_ID="$session_id"
  export HARNESS_SESSION="true"
  export INTERRUPT_FILE="$INTERRUPT_FILE"

  # TODO: Actual Claude Code spawning
  # For now, this is a placeholder
  log_warn "Agent spawning not yet fully implemented"
  log "Would spawn Claude Code with:"
  log "  - Session ID: $session_id"
  log "  - Bootstrap: $prompt_file"
  log "  - Work: $work_item"

  # Mark session as started
  jq '.status = "spawned"' "$SESSION_FILE" > "$SESSION_FILE.tmp"
  mv "$SESSION_FILE.tmp" "$SESSION_FILE"

  return 0
}

# Monitor agent session
monitor_session() {
  # Check if session file exists and is running
  if [[ ! -f "$SESSION_FILE" ]]; then
    log_warn "No active session"
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
      log_error "Failed to spawn agent, retrying next iteration"
      sleep "$ITERATION_DELAY"
      continue
    fi

    # Monitor agent session
    while monitor_session; do
      # Check for interrupt conditions
      if check_interrupt; then
        log_warn "Interrupt detected, pausing harness"
        preserve_context
        wait_for_resume
        log "Resuming after interrupt"
      fi

      sleep "$INTERRUPT_CHECK_INTERVAL"
    done

    # Session complete, prepare next iteration
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

# Run main loop
main "$@"
