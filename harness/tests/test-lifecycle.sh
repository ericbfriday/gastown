#!/usr/bin/env bash
# Integration tests for session lifecycle and state transitions
# Tests spawning→running→completed/failed/timeout/interrupted transitions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-lifecycle-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up lifecycle test environment"

  mkdir -p "$TEST_ROOT"/{state,docs/sessions}

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"

  # Set up environment
  export HARNESS_ROOT="$TEST_ROOT"
  export STATE_DIR="$TEST_ROOT/state"
  export DOCS_DIR="$TEST_ROOT/docs"

  # Source loop.sh
  source "$TEST_ROOT/loop.sh" 2>/dev/null || true
}

teardown() {
  log_test "Cleaning up lifecycle test environment"
  rm -rf "$TEST_ROOT"
}

# ============================================================
# TEST CASES
# ============================================================

test_state_transition_spawning_to_running() {
  test_start "State transition: spawning → running"

  local session_id="ses_lifecycle_1"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create session in spawning state
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "spawning"
    }' > "$SESSION_FILE"

  # Transition to running
  update_session_status "$session_id" "running" ""

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "running" "$status" "Status updated to running"

  # Verify timestamp updated
  local updated_at
  updated_at=$(jq -r '.status_updated_at' "$SESSION_FILE")
  assert_not_equals "null" "$updated_at" "Status timestamp recorded"

  test_pass
}

test_state_transition_running_to_completed() {
  test_start "State transition: running → completed"

  local session_id="ses_lifecycle_2"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create running session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Transition to completed
  update_session_status "$session_id" "completed" "work finished"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "completed" "$status" "Status updated to completed"

  # Verify ended_at set
  local ended_at
  ended_at=$(jq -r '.ended_at' "$SESSION_FILE")
  assert_not_equals "null" "$ended_at" "End timestamp set"

  # Verify reason recorded
  local reason
  reason=$(jq -r '.status_reason' "$SESSION_FILE")
  assert_equals "work finished" "$reason" "Reason recorded"

  test_pass
}

test_state_transition_running_to_failed() {
  test_start "State transition: running → failed"

  local session_id="ses_lifecycle_3"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create running session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Transition to failed
  update_session_status "$session_id" "failed" "exit code 1"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "failed" "$status" "Status updated to failed"

  # Verify ended_at set
  local ended_at
  ended_at=$(jq -r '.ended_at' "$SESSION_FILE")
  assert_not_equals "null" "$ended_at" "End timestamp set"

  test_pass
}

test_state_transition_running_to_timeout() {
  test_start "State transition: running → timeout"

  local session_id="ses_lifecycle_4"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create running session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Transition to timeout
  update_session_status "$session_id" "timeout" "exceeded time limit"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "timeout" "$status" "Status updated to timeout"

  # Verify ended_at set
  local ended_at
  ended_at=$(jq -r '.ended_at' "$SESSION_FILE")
  assert_not_equals "null" "$ended_at" "End timestamp set"

  test_pass
}

test_state_transition_running_to_interrupted() {
  test_start "State transition: running → interrupted"

  local session_id="ses_lifecycle_5"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create running session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Transition to interrupted
  update_session_status "$session_id" "interrupted" "manual interrupt"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "interrupted" "$status" "Status updated to interrupted"

  test_pass
}

test_state_transition_interrupted_to_running() {
  test_start "State transition: interrupted → running (resume)"

  local session_id="ses_lifecycle_6"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create interrupted session
  create_mock_session "$session_id" "$STATE_DIR" "interrupted"

  # Transition back to running (resume)
  update_session_status "$session_id" "running" "resumed after interrupt"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "running" "$status" "Status updated back to running"

  test_pass
}

test_detect_completion_success() {
  test_start "Detect completion with exit code 0"

  local session_id="ses_completion_ok"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create completed session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Create exit code file (success)
  echo "0" > "$STATE_DIR/${session_id}.exit"

  # Create PID file (but process not running)
  echo "99999" > "$STATE_DIR/${session_id}.pid"

  # Detect completion
  if detect_completion "$session_id" 2>/dev/null; then
    local status
    status=$(jq -r '.status' "$SESSION_FILE")
    assert_equals "completed" "$status" "Completion detected and status updated"
    test_pass
  else
    test_fail "Completion not detected"
  fi
}

test_detect_completion_failure() {
  test_start "Detect completion with non-zero exit code"

  local session_id="ses_completion_fail"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Create exit code file (failure)
  echo "1" > "$STATE_DIR/${session_id}.exit"

  # Create PID file
  echo "99999" > "$STATE_DIR/${session_id}.pid"

  # Detect completion (should return failure)
  if ! detect_completion "$session_id" 2>/dev/null; then
    local status
    status=$(jq -r '.status' "$SESSION_FILE")
    assert_equals "failed" "$status" "Failure detected and status updated"
    test_pass
  else
    test_fail "Should have detected failure"
  fi
}

test_is_agent_running() {
  test_start "Check if agent is still running"

  local session_id="ses_running_check"

  # Create PID file with current shell PID (running)
  echo "$$" > "$STATE_DIR/${session_id}.pid"

  if is_agent_running "$session_id"; then
    assert_true "Running agent detected"
  else
    test_fail "Should detect running agent"
    return
  fi

  # Create PID file with non-existent PID
  echo "99999" > "$STATE_DIR/${session_id}.pid"

  if ! is_agent_running "$session_id" 2>/dev/null; then
    assert_true "Non-running agent detected correctly"
    test_pass
  else
    test_fail "Should not detect non-existent PID"
  fi
}

test_session_archival() {
  test_start "Session archival on iteration end"

  local session_id="ses_archive_test"
  export SESSION_FILE="$STATE_DIR/current-session.json"

  # Create current session
  create_mock_session "$session_id" "$STATE_DIR" "completed"
  cp "$STATE_DIR/${session_id}.json" "$SESSION_FILE"

  # Archive session
  next_iteration

  # Verify moved to docs/sessions
  assert_file_exists "$DOCS_DIR/sessions/${session_id}.json" "Session archived"
  assert_file_not_exists "$SESSION_FILE" "Current session file removed"

  test_pass
}

test_lifecycle_complete_workflow() {
  test_start "Complete lifecycle: spawn → run → complete → archive"

  local session_id="ses_complete_workflow"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # 1. Spawning
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "spawning"
    }' > "$SESSION_FILE"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "spawning" "$status" "Step 1: Spawning"

  # 2. Running
  update_session_status "$session_id" "running" ""
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "running" "$status" "Step 2: Running"

  # 3. Completed
  echo "0" > "$STATE_DIR/${session_id}.exit"
  echo "99999" > "$STATE_DIR/${session_id}.pid"
  detect_completion "$session_id" 2>/dev/null || true
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "completed" "$status" "Step 3: Completed"

  # 4. Archive
  export SESSION_FILE="$STATE_DIR/current-session.json"
  cp "$STATE_DIR/${session_id}.json" "$SESSION_FILE"
  next_iteration
  assert_file_exists "$DOCS_DIR/sessions/${session_id}.json" "Step 4: Archived"

  test_pass
}

test_lifecycle_failure_workflow() {
  test_start "Failure lifecycle: spawn → run → fail → archive"

  local session_id="ses_failure_workflow"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create and run session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Simulate failure
  echo "1" > "$STATE_DIR/${session_id}.exit"
  echo "99999" > "$STATE_DIR/${session_id}.pid"

  detect_completion "$session_id" 2>/dev/null || true

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "failed" "$status" "Failed status set"

  # Archive
  export SESSION_FILE="$STATE_DIR/current-session.json"
  cp "$STATE_DIR/${session_id}.json" "$SESSION_FILE"
  next_iteration

  assert_file_exists "$DOCS_DIR/sessions/${session_id}.json" "Failed session archived"

  test_pass
}

test_lifecycle_interrupt_workflow() {
  test_start "Interrupt lifecycle: spawn → run → interrupt → resume"

  local session_id="ses_interrupt_workflow"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create running session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Interrupt
  update_session_status "$session_id" "interrupted" "manual stop"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "interrupted" "$status" "Interrupted status set"

  # Resume
  update_session_status "$session_id" "running" "resumed"

  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "running" "$status" "Resumed to running"

  test_pass
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Lifecycle Integration Tests"

  setup

  # Run tests
  test_state_transition_spawning_to_running
  test_state_transition_running_to_completed
  test_state_transition_running_to_failed
  test_state_transition_running_to_timeout
  test_state_transition_running_to_interrupted
  test_state_transition_interrupted_to_running
  test_detect_completion_success
  test_detect_completion_failure
  test_is_agent_running
  test_session_archival
  test_lifecycle_complete_workflow
  test_lifecycle_failure_workflow
  test_lifecycle_interrupt_workflow

  teardown

  test_suite_end
}

main "$@"
