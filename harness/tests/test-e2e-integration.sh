#!/usr/bin/env bash
# End-to-end integration tests
# Tests complete harness loop with mock Claude Code

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-e2e-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up E2E test environment"

  mkdir -p "$TEST_ROOT"/{state,scripts,prompts,docs/sessions}

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"
  cp "$SCRIPT_DIR/mocks/mock-queue.sh" "$TEST_ROOT/scripts/manage-queue.sh"
  cp "$SCRIPT_DIR/fixtures/sample-bootstrap.md" "$TEST_ROOT/prompts/bootstrap.md"

  # Create mock scripts
  cat > "$TEST_ROOT/scripts/check-interrupt.sh" <<'EOF'
#!/usr/bin/env bash
[[ -f "$INTERRUPT_FILE" ]]
EOF
  chmod +x "$TEST_ROOT/scripts/check-interrupt.sh"

  cat > "$TEST_ROOT/scripts/preserve-context.sh" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "$TEST_ROOT/scripts/preserve-context.sh"

  # Set up environment
  export HARNESS_ROOT="$TEST_ROOT"
  export STATE_DIR="$TEST_ROOT/state"
  export SCRIPTS_DIR="$TEST_ROOT/scripts"
  export PROMPTS_DIR="$TEST_ROOT/prompts"
  export DOCS_DIR="$TEST_ROOT/docs"
  export GT_ROOT="$TEST_ROOT"
  export INTERRUPT_FILE="$STATE_DIR/interrupt-request.txt"
  export SESSION_TIMEOUT=30
  export STALL_THRESHOLD=60

  # Create mock claude
  cat > "$TEST_ROOT/claude" <<'EOF'
#!/usr/bin/env bash
exec "$MOCK_CLAUDE_PATH" "$@"
EOF
  chmod +x "$TEST_ROOT/claude"

  export MOCK_CLAUDE_PATH="$SCRIPT_DIR/mocks/mock-claude.sh"
  export PATH="$TEST_ROOT:$PATH"

  # Source loop.sh
  source "$TEST_ROOT/loop.sh" 2>/dev/null || true
}

teardown() {
  log_test "Cleaning up E2E test environment"
  cleanup_background_jobs
  rm -rf "$TEST_ROOT"
}

# ============================================================
# TEST CASES
# ============================================================

test_e2e_single_iteration_success() {
  test_start "E2E: Single iteration with successful completion"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=2
  export MOCK_TOOL_CALLS=3
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/current-session.json"

  # Spawn agent
  iteration=1
  if spawn_agent; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    assert_file_exists "$SESSION_FILE" "Session created"

    # Wait for completion
    sleep 3

    # Check session completed
    if ! is_agent_running "$session_id"; then
      local exit_code
      exit_code=$(get_agent_exit_code "$session_id")

      assert_equals "0" "$exit_code" "Agent exited successfully"

      # Verify logs exist
      assert_file_exists "$DOCS_DIR/sessions/${session_id}.log" "Logs created"

      # Verify events captured
      local log_content
      log_content=$(cat "$DOCS_DIR/sessions/${session_id}.log")

      if [[ "$log_content" == *"message_start"* ]]; then
        assert_true "Stream-JSON events captured"
        test_pass
      else
        test_fail "No stream-JSON events captured"
      fi
    else
      kill_agent "$session_id" 2>/dev/null || true
      test_fail "Agent still running"
    fi
  else
    test_fail "Spawn failed"
  fi
}

test_e2e_spawn_monitor_complete() {
  test_start "E2E: Spawn → Monitor → Complete workflow"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=2
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/e2e-workflow-session.json"

  # 1. Spawn
  iteration=1
  if ! spawn_agent; then
    test_fail "Spawn failed"
    return
  fi

  local session_id
  session_id=$(jq -r '.session_id' "$SESSION_FILE")

  # 2. Monitor (brief)
  for i in {1..3}; do
    update_heartbeat "$session_id" 2>/dev/null || true
    update_progress "$session_id" 2>/dev/null || true
    check_agent_health "$session_id" 2>/dev/null || true
    sleep 1
  done

  # 3. Wait for completion
  local timeout=5
  local elapsed=0
  while is_agent_running "$session_id" && [[ $elapsed -lt $timeout ]]; do
    sleep 1
    elapsed=$((elapsed + 1))
  done

  # 4. Detect completion
  detect_completion "$session_id" 2>/dev/null || true

  # 5. Verify status
  local status
  status=$(jq -r '.status' "$SESSION_FILE")

  if [[ "$status" == "completed" ]]; then
    assert_true "Complete workflow succeeded"

    # Extract metrics
    local metrics_file
    metrics_file=$(extract_session_metrics "$session_id")

    assert_file_exists "$metrics_file" "Metrics extracted"

    test_pass
  else
    kill_agent "$session_id" 2>/dev/null || true
    test_fail "Workflow did not complete (status: $status)"
  fi
}

test_e2e_multiple_iterations() {
  test_start "E2E: Multiple iterations sequentially"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=1
  export MOCK_QUEUE_SIZE=1

  local iterations=2
  local session_ids=()

  for i in $(seq 1 $iterations); do
    export SESSION_FILE="$STATE_DIR/multi-iter-${i}.json"

    iteration=$i
    if spawn_agent; then
      local session_id
      session_id=$(jq -r '.session_id' "$SESSION_FILE")
      session_ids+=("$session_id")

      # Wait for completion
      sleep 2

      # Cleanup
      if is_agent_running "$session_id"; then
        kill_agent "$session_id" 2>/dev/null || true
      fi
    else
      test_fail "Iteration $i failed to spawn"
      return
    fi
  done

  # Verify all iterations completed
  assert_equals "$iterations" "${#session_ids[@]}" "All iterations spawned"

  test_pass
}

test_e2e_error_recovery() {
  test_start "E2E: Error detection and recovery"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=error
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/error-recovery-session.json"

  # Spawn agent that will error
  iteration=1
  if spawn_agent; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Wait for error
    sleep 2

    # Check if detected as failed
    if ! is_agent_running "$session_id"; then
      local exit_code
      exit_code=$(get_agent_exit_code "$session_id")

      # Error should have non-zero exit
      if [[ "$exit_code" != "0" ]]; then
        assert_true "Error detected with non-zero exit"

        # Verify failure handling
        handle_spawn_failure "$session_id" "test error" 2>/dev/null || true

        local failure_count
        failure_count=$(cat "$STATE_DIR/failure-count" 2>/dev/null || echo 0)

        assert_greater_than "$failure_count" "0" "Failure counter incremented"

        # Reset for next test
        reset_failure_counter

        test_pass
      else
        test_fail "No error exit code recorded"
      fi
    else
      kill_agent "$session_id" 2>/dev/null || true
      test_fail "Agent still running after error"
    fi
  else
    test_fail "Spawn failed"
  fi
}

test_e2e_interrupt_and_resume() {
  test_start "E2E: Interrupt handling and resume"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=5
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/interrupt-resume-session.json"

  # Spawn agent
  iteration=1
  if spawn_agent; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Wait a bit, then interrupt
    sleep 1
    echo "E2E TEST: Interrupt request" > "$INTERRUPT_FILE"

    # Check interrupt detected
    if check_interrupt; then
      assert_true "Interrupt detected"

      # Kill agent
      kill_agent "$session_id" 2>/dev/null || true

      # Update status
      update_session_status "$session_id" "interrupted" "test interrupt"

      local status
      status=$(jq -r '.status' "$SESSION_FILE")
      assert_equals "interrupted" "$status" "Status updated to interrupted"

      # Clear interrupt (resume)
      rm -f "$INTERRUPT_FILE"

      # Verify can continue
      if ! check_interrupt; then
        assert_true "Ready to resume"
        test_pass
      else
        test_fail "Interrupt not cleared"
      fi
    else
      kill_agent "$session_id" 2>/dev/null || true
      test_fail "Interrupt not detected"
    fi
  else
    test_fail "Spawn failed"
  fi
}

test_e2e_work_queue_integration() {
  test_start "E2E: Work queue integration"

  export MOCK_QUEUE_SIZE=1
  export MOCK_WORK_ITEM='{"id":"e2e-test-001","title":"E2E test work item","priority":10}'

  # Check queue
  if check_work_queue; then
    assert_true "Queue has work"

    # Get next work item
    local work_json
    work_json=$("$SCRIPTS_DIR/manage-queue.sh" next)

    local work_id
    work_id=$(echo "$work_json" | jq -r '.id')

    assert_equals "e2e-test-001" "$work_id" "Work item retrieved correctly"

    test_pass
  else
    test_fail "Queue check failed"
  fi
}

test_e2e_session_persistence() {
  test_start "E2E: Session state persistence"

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=1
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/persistence-session.json"

  # Spawn agent
  iteration=1
  if spawn_agent; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Verify state persisted
    assert_file_exists "$SESSION_FILE" "Session state persisted"

    # Wait for completion
    sleep 2

    # Archive
    mv "$SESSION_FILE" "$DOCS_DIR/sessions/${session_id}.json"

    assert_file_exists "$DOCS_DIR/sessions/${session_id}.json" "Session archived"

    # Verify can read archived state
    local archived_status
    archived_status=$(jq -r '.status' "$DOCS_DIR/sessions/${session_id}.json")

    assert_not_equals "null" "$archived_status" "Archived state readable"

    test_pass
  else
    test_fail "Spawn failed"
  fi
}

test_e2e_metrics_collection() {
  test_start "E2E: End-to-end metrics collection"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=2
  export MOCK_TOOL_CALLS=5
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/metrics-e2e-session.json"

  # Spawn agent
  iteration=1
  if spawn_agent; then
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Wait for completion
    sleep 3

    # Extract metrics
    local metrics_file
    metrics_file=$(extract_session_metrics "$session_id")

    if [[ -f "$metrics_file" ]]; then
      assert_file_exists "$metrics_file" "Metrics collected"

      # Verify metrics structure
      local has_api_usage=$(jq 'has("api_usage")' "$metrics_file")
      local has_tool_usage=$(jq 'has("tool_usage")' "$metrics_file")

      assert_equals "true" "$has_api_usage" "API usage tracked"
      assert_equals "true" "$has_tool_usage" "Tool usage tracked"

      test_pass
    else
      test_fail "Metrics not collected"
    fi

    # Cleanup
    kill_agent "$session_id" 2>/dev/null || true
  else
    test_fail "Spawn failed"
  fi
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "End-to-End Integration Tests"

  setup

  # Run tests
  test_e2e_single_iteration_success
  test_e2e_spawn_monitor_complete
  test_e2e_multiple_iterations
  test_e2e_error_recovery
  test_e2e_interrupt_and_resume
  test_e2e_work_queue_integration
  test_e2e_session_persistence
  test_e2e_metrics_collection

  teardown

  test_suite_end
}

main "$@"
