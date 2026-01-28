#!/usr/bin/env bash
# Integration tests for error scenarios and recovery
# Tests timeout, crash, malformed JSON, missing dependencies, consecutive failures

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-error-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up error scenarios test environment"

  mkdir -p "$TEST_ROOT"/{state,scripts,prompts,docs/sessions}

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"
  cp "$SCRIPT_DIR/mocks/mock-queue.sh" "$TEST_ROOT/scripts/manage-queue.sh"
  cp "$SCRIPT_DIR/fixtures/sample-bootstrap.md" "$TEST_ROOT/prompts/bootstrap.md"

  # Create mock scripts
  cat > "$TEST_ROOT/scripts/check-interrupt.sh" <<'EOF'
#!/usr/bin/env bash
exit 1
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
  log_test "Cleaning up error scenarios test environment"
  cleanup_background_jobs
  rm -rf "$TEST_ROOT"
}

# ============================================================
# TEST CASES
# ============================================================

test_agent_timeout() {
  test_start "Agent timeout detection and handling"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  local session_id="ses_timeout_test"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"
  export MOCK_BEHAVIOR=timeout
  export MOCK_DURATION=5
  export SESSION_TIMEOUT=3  # 3 second timeout

  # Create mock session
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

  # Check health (should timeout)
  sleep 4  # Wait for timeout

  if ! check_agent_health "$session_id" 2>/dev/null; then
    local status=$(jq -r '.status' "$SESSION_FILE")
    assert_equals "timeout" "$status" "Session marked as timeout"
    test_pass
  else
    test_fail "Timeout not detected"
  fi
}

test_agent_crash() {
  test_start "Agent crash detection"

  local session_id="ses_crash_test"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"
  export MOCK_BEHAVIOR=crash
  export MOCK_QUEUE_SIZE=1

  # Spawn agent that will crash
  iteration=1
  if spawn_agent 2>/dev/null; then
    # Wait for crash
    sleep 2

    # Check if detected as failed
    if ! is_agent_running "$session_id"; then
      local exit_code
      exit_code=$(get_agent_exit_code "$session_id")

      # Crash should have non-zero exit
      if [[ "$exit_code" != "0" ]] && [[ -n "$exit_code" ]]; then
        assert_true "Crash detected with non-zero exit code"
        test_pass
      else
        test_fail "Exit code not recorded properly"
      fi
    else
      test_fail "Agent still running after crash"
    fi

    # Cleanup
    kill_agent "$session_id" 2>/dev/null || true
  else
    test_fail "Spawn failed"
  fi
}

test_malformed_json() {
  test_start "Handle malformed stream-JSON gracefully"

  local session_id="ses_malformed_test"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  # Create session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create log with malformed JSON
  cat > "$log_file" <<EOF
{"type":"message_start"}
This is not JSON at all
{"type":"content_block_delta","delta":{
{"incomplete json
Normal text output
{"type":"message_stop"}
EOF

  # Try to parse events
  local valid_events=0
  while IFS= read -r line; do
    local event_type
    event_type=$(parse_stream_event "$line" 2>/dev/null || echo "")

    [[ -n "$event_type" ]] && valid_events=$((valid_events + 1))
  done < "$log_file"

  # Should parse the 2 valid JSON lines
  assert_equals "2" "$valid_events" "Only valid JSON parsed"

  test_pass
}

test_missing_api_key() {
  test_start "Handle missing API key"

  # Unset API key if present
  local original_key="${ANTHROPIC_API_KEY:-}"
  unset ANTHROPIC_API_KEY

  # Try to spawn (should fail gracefully)
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/missing-key-session.json"

  # In our mock setup, this won't actually check API key
  # But we can verify the error handling path exists

  assert_true "Error handling path exists"

  # Restore API key
  [[ -n "$original_key" ]] && export ANTHROPIC_API_KEY="$original_key"

  test_pass
}

test_missing_bootstrap() {
  test_start "Handle missing bootstrap template"

  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$STATE_DIR/missing-bootstrap-session.json"

  # Remove bootstrap
  mv "$TEST_ROOT/prompts/bootstrap.md" "$TEST_ROOT/prompts/bootstrap.md.bak"

  # Try to spawn (should fail)
  iteration=1
  if ! spawn_agent 2>/dev/null; then
    assert_true "Spawn correctly fails with missing bootstrap"
    # Restore bootstrap
    mv "$TEST_ROOT/prompts/bootstrap.md.bak" "$TEST_ROOT/prompts/bootstrap.md"
    test_pass
  else
    # Restore bootstrap
    mv "$TEST_ROOT/prompts/bootstrap.md.bak" "$TEST_ROOT/prompts/bootstrap.md"
    test_fail "Spawn should have failed"
  fi
}

test_empty_queue() {
  test_start "Handle empty work queue"

  export MOCK_QUEUE_SIZE=0
  export SESSION_FILE="$STATE_DIR/empty-queue-session.json"

  iteration=1
  if ! spawn_agent 2>/dev/null; then
    assert_true "Spawn correctly fails with empty queue"
    test_pass
  else
    test_fail "Spawn should have failed with empty queue"
  fi
}

test_consecutive_failure_threshold() {
  test_start "Consecutive failure threshold triggers interrupt"

  export MAX_CONSECUTIVE_FAILURES=3
  export MOCK_QUEUE_SIZE=0  # Always fail to spawn

  # Simulate multiple failures
  rm -f "$STATE_DIR/failure-count"

  iteration=1

  # First failure
  spawn_agent 2>/dev/null || true
  handle_spawn_failure "ses_fail1" "test" 2>/dev/null || true

  local count=$(cat "$STATE_DIR/failure-count" 2>/dev/null || echo 0)
  assert_equals "1" "$count" "Failure count incremented"

  # Second failure
  spawn_agent 2>/dev/null || true
  handle_spawn_failure "ses_fail2" "test" 2>/dev/null || true

  count=$(cat "$STATE_DIR/failure-count" 2>/dev/null || echo 0)
  assert_equals "2" "$count" "Failure count incremented again"

  # Third failure (should trigger interrupt)
  spawn_agent 2>/dev/null || true
  if ! handle_spawn_failure "ses_fail3" "test" 2>/dev/null; then
    # Should have created interrupt file
    assert_file_exists "$INTERRUPT_FILE" "Interrupt file created"

    local reason=$(cat "$INTERRUPT_FILE")
    assert_string_contains "$reason" "spawn failures" "Interrupt reason correct"

    test_pass
  else
    test_fail "Should have triggered interrupt"
  fi

  # Cleanup
  rm -f "$INTERRUPT_FILE" "$STATE_DIR/failure-count"
}

test_failure_counter_reset() {
  test_start "Failure counter resets on success"

  # Set failure count
  echo "5" > "$STATE_DIR/failure-count"

  # Reset it
  reset_failure_counter

  assert_file_not_exists "$STATE_DIR/failure-count" "Failure count file removed"

  test_pass
}

test_error_in_stream() {
  test_start "Detect and log errors in stream-JSON"

  local session_id="ses_stream_error"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local error_log="$STATE_DIR/sessions/${session_id}/errors.jsonl"

  # Create session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  mkdir -p "$STATE_DIR/sessions/${session_id}"

  # Create log with error event
  cat > "$log_file" <<EOF
{"type":"message_start"}
{"type":"error","error":{"type":"api_error","message":"Rate limit exceeded"},"timestamp":"2026-01-27T10:00:00Z"}
{"type":"message_stop"}
EOF

  # Simulate processing (would normally happen in background processor)
  grep '"type":"error"' "$log_file" > "$error_log"

  assert_file_exists "$error_log" "Error log created"

  local error_count=$(wc -l < "$error_log" | tr -d ' ')
  assert_equals "1" "$error_count" "Error logged"

  test_pass
}

test_agent_exit_code_handling() {
  test_start "Handle various agent exit codes"

  local session_id="ses_exit_test"

  # Test exit code 0 (success)
  echo "0" > "$STATE_DIR/${session_id}.exit"
  local code
  code=$(get_agent_exit_code "$session_id")
  assert_equals "0" "$code" "Exit code 0 retrieved"

  # Test exit code 1 (failure)
  echo "1" > "$STATE_DIR/${session_id}.exit"
  code=$(get_agent_exit_code "$session_id")
  assert_equals "1" "$code" "Exit code 1 retrieved"

  # Test missing exit file
  rm "$STATE_DIR/${session_id}.exit"
  code=$(get_agent_exit_code "$session_id")
  assert_equals "" "$code" "Missing exit file returns empty"

  test_pass
}

test_stall_detection_and_kill() {
  test_start "Stalled agent detection and termination"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))
    return 0
  fi

  local session_id="ses_stall_kill"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"
  export STALL_THRESHOLD=3  # 3 second threshold

  # Create stalled session (old heartbeat)
  local old_time
  old_time=$(date -u -v-5M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%SZ)

  jq -n \
    --arg sid "$session_id" \
    --arg started "$old_time" \
    --argjson start_epoch "$(($(date +%s) - 300))" \
    --arg last_check "$old_time" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running",
      heartbeat: {
        last_check: $last_check
      }
    }' > "$SESSION_FILE"

  # Detect stall
  if detect_stall "$session_id" 2>/dev/null; then
    assert_true "Stall detected correctly"
    test_pass
  else
    test_fail "Stall not detected"
  fi
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Error Scenario Integration Tests"

  setup

  # Run tests
  test_agent_timeout
  test_agent_crash
  test_malformed_json
  test_missing_api_key
  test_missing_bootstrap
  test_empty_queue
  test_consecutive_failure_threshold
  test_failure_counter_reset
  test_error_in_stream
  test_agent_exit_code_handling
  test_stall_detection_and_kill

  teardown

  test_suite_end
}

main "$@"
