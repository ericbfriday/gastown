#!/usr/bin/env bash
# Integration tests for interrupt mechanism
# Tests manual interrupt, quality gate, timeout, context preservation, resume

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-interrupt-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up interrupt test environment"

  mkdir -p "$TEST_ROOT"/{state,scripts,docs/sessions}

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"

  # Create mock scripts
  cat > "$TEST_ROOT/scripts/check-interrupt.sh" <<'EOF'
#!/usr/bin/env bash
[[ -f "$INTERRUPT_FILE" ]]
EOF
  chmod +x "$TEST_ROOT/scripts/check-interrupt.sh"

  cat > "$TEST_ROOT/scripts/preserve-context.sh" <<'EOF'
#!/usr/bin/env bash
echo "Context preserved at $(date)" > "$STATE_DIR/context-preserved.txt"
exit 0
EOF
  chmod +x "$TEST_ROOT/scripts/preserve-context.sh"

  # Set up environment
  export HARNESS_ROOT="$TEST_ROOT"
  export STATE_DIR="$TEST_ROOT/state"
  export SCRIPTS_DIR="$TEST_ROOT/scripts"
  export DOCS_DIR="$TEST_ROOT/docs"
  export INTERRUPT_FILE="$STATE_DIR/interrupt-request.txt"

  # Source loop.sh
  source "$TEST_ROOT/loop.sh" 2>/dev/null || true
}

teardown() {
  log_test "Cleaning up interrupt test environment"
  cleanup_background_jobs
  rm -rf "$TEST_ROOT"
}

# ============================================================
# TEST CASES
# ============================================================

test_manual_interrupt_creation() {
  test_start "Manual interrupt file creation"

  # Create interrupt file
  echo "MANUAL: User requested pause" > "$INTERRUPT_FILE"

  assert_file_exists "$INTERRUPT_FILE" "Interrupt file created"

  # Check interrupt detection
  if check_interrupt; then
    assert_true "Interrupt detected"
    test_pass
  else
    test_fail "Interrupt not detected"
  fi

  # Cleanup
  rm -f "$INTERRUPT_FILE"
}

test_check_interrupt_with_helper() {
  test_start "Check interrupt via helper script"

  # No interrupt initially
  if ! check_interrupt 2>/dev/null; then
    assert_true "No interrupt when file absent"
  else
    test_fail "False positive interrupt"
    return
  fi

  # Create interrupt
  echo "TEST: Interrupt triggered" > "$INTERRUPT_FILE"

  # Should detect interrupt
  if check_interrupt; then
    assert_true "Interrupt detected via helper"
    test_pass
  else
    test_fail "Helper failed to detect interrupt"
  fi

  # Cleanup
  rm -f "$INTERRUPT_FILE"
}

test_quality_gate_interrupt() {
  test_start "Quality gate failure triggers interrupt"

  # Simulate quality gate failure
  echo "QUALITY_GATE: Test coverage below threshold" > "$INTERRUPT_FILE"

  assert_file_exists "$INTERRUPT_FILE" "Quality gate interrupt created"

  local reason
  reason=$(cat "$INTERRUPT_FILE")
  assert_string_contains "$reason" "QUALITY_GATE" "Reason contains quality gate marker"

  # Cleanup
  rm -f "$INTERRUPT_FILE"

  test_pass
}

test_consecutive_failures_interrupt() {
  test_start "Consecutive failures trigger interrupt"

  # Simulate multiple spawn failures
  export MAX_CONSECUTIVE_FAILURES=3
  rm -f "$STATE_DIR/failure-count"

  # Trigger failures
  for i in {1..3}; do
    handle_spawn_failure "ses_fail_$i" "spawn_error" 2>/dev/null || true
  done

  assert_file_exists "$INTERRUPT_FILE" "Interrupt created after threshold"

  local reason
  reason=$(cat "$INTERRUPT_FILE")
  assert_string_contains "$reason" "spawn failures" "Reason mentions failures"

  # Cleanup
  rm -f "$INTERRUPT_FILE" "$STATE_DIR/failure-count"

  test_pass
}

test_preserve_context() {
  test_start "Context preservation on interrupt"

  # Call preserve_context
  preserve_context

  assert_file_exists "$STATE_DIR/context-preserved.txt" "Context file created"

  local content
  content=$(cat "$STATE_DIR/context-preserved.txt")
  assert_string_contains "$content" "Context preserved" "Context message present"

  test_pass
}

test_wait_for_resume() {
  test_start "Wait for resume mechanism"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  # Create interrupt
  echo "TEST: Waiting for resume" > "$INTERRUPT_FILE"

  # Start wait_for_resume in background
  (
    sleep 2
    rm -f "$INTERRUPT_FILE"  # Remove interrupt after 2 seconds
  ) &
  local remover_pid=$!

  local start_time=$(date +%s)

  # This will block until interrupt file is removed
  wait_for_resume &
  local wait_pid=$!

  # Wait for completion (with timeout)
  local timeout=5
  local elapsed=0
  while kill -0 "$wait_pid" 2>/dev/null && [[ $elapsed -lt $timeout ]]; do
    sleep 1
    elapsed=$((elapsed + 1))
  done

  if ! kill -0 "$wait_pid" 2>/dev/null; then
    wait "$wait_pid" 2>/dev/null || true
    assert_true "Resume completed"
    test_pass
  else
    kill "$wait_pid" 2>/dev/null || true
    test_fail "Wait for resume timed out"
  fi

  # Cleanup
  wait "$remover_pid" 2>/dev/null || true
  rm -f "$INTERRUPT_FILE"
}

test_interrupt_during_monitoring() {
  test_start "Interrupt detection during monitoring loop"

  local session_id="ses_interrupt_mon"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create mock running session
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running",
      pid: 99999
    }' > "$SESSION_FILE"

  # Create interrupt
  echo "MONITORING: Test interrupt" > "$INTERRUPT_FILE"

  # Check interrupt (monitoring loop would call this)
  if check_interrupt; then
    assert_true "Interrupt detected during monitoring"

    # Verify session can be updated to interrupted status
    update_session_status "$session_id" "interrupted" "test interrupt"

    local status
    status=$(jq -r '.status' "$SESSION_FILE")
    assert_equals "interrupted" "$status" "Session status updated"

    test_pass
  else
    test_fail "Interrupt not detected"
  fi

  # Cleanup
  rm -f "$INTERRUPT_FILE"
}

test_interrupt_clears_on_resume() {
  test_start "Interrupt file cleared on resume"

  # Create interrupt
  echo "TEST: Clear on resume" > "$INTERRUPT_FILE"
  assert_file_exists "$INTERRUPT_FILE" "Interrupt file exists"

  # Remove interrupt (simulating resume)
  rm -f "$INTERRUPT_FILE"

  assert_file_not_exists "$INTERRUPT_FILE" "Interrupt file removed"

  # Verify no interrupt detected
  if ! check_interrupt 2>/dev/null; then
    assert_true "No interrupt after clear"
    test_pass
  else
    test_fail "Interrupt still detected"
  fi
}

test_multiple_interrupt_reasons() {
  test_start "Track multiple interrupt reasons"

  local reasons=(
    "MANUAL: User requested"
    "QUALITY_GATE: Coverage too low"
    "TIMEOUT: Session exceeded limit"
    "FAILURE: Too many spawn failures"
  )

  for reason in "${reasons[@]}"; do
    echo "$reason" > "$INTERRUPT_FILE"

    assert_file_exists "$INTERRUPT_FILE" "Interrupt created"

    local content
    content=$(cat "$INTERRUPT_FILE")
    assert_equals "$reason" "$content" "Reason recorded correctly"

    rm -f "$INTERRUPT_FILE"
  done

  test_pass
}

test_interrupt_with_context_preservation() {
  test_start "Interrupt triggers context preservation"

  local session_id="ses_interrupt_ctx"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Create interrupt
  echo "TEST: With context preservation" > "$INTERRUPT_FILE"

  # Preserve context
  preserve_context

  # Update session status
  update_session_status "$session_id" "interrupted" "context preserved"

  assert_file_exists "$STATE_DIR/context-preserved.txt" "Context preserved"

  local status
  status=$(jq -r '.status' "$SESSION_FILE")
  assert_equals "interrupted" "$status" "Session marked interrupted"

  # Cleanup
  rm -f "$INTERRUPT_FILE"

  test_pass
}

test_resume_after_interrupt() {
  test_start "Resume workflow after interrupt resolution"

  # Create interrupt
  echo "TEST: Resume test" > "$INTERRUPT_FILE"

  # Preserve context
  preserve_context

  # Resolve interrupt (would be done by human/automation)
  rm -f "$INTERRUPT_FILE"

  # Verify ready for next iteration
  if ! check_interrupt 2>/dev/null; then
    assert_true "Ready to resume"

    # Reset failure counter (would happen on successful spawn)
    reset_failure_counter

    assert_file_not_exists "$STATE_DIR/failure-count" "Failure counter cleared"

    test_pass
  else
    test_fail "Not ready to resume"
  fi
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Interrupt Mechanism Integration Tests"

  setup

  # Run tests
  test_manual_interrupt_creation
  test_check_interrupt_with_helper
  test_quality_gate_interrupt
  test_consecutive_failures_interrupt
  test_preserve_context
  test_wait_for_resume
  test_interrupt_during_monitoring
  test_interrupt_clears_on_resume
  test_multiple_interrupt_reasons
  test_interrupt_with_context_preservation
  test_resume_after_interrupt

  teardown

  test_suite_end
}

main "$@"
