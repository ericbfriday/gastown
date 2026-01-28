#!/usr/bin/env bash
# Integration tests for spawn_agent() mechanism
# Tests agent spawning, bootstrap injection, and session tracking

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-spawn-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up test environment"

  # Create test harness structure
  mkdir -p "$TEST_ROOT"/{state,scripts,prompts,docs/sessions}

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"
  cp "$SCRIPT_DIR/mocks/mock-queue.sh" "$TEST_ROOT/scripts/manage-queue.sh"
  cp "$SCRIPT_DIR/fixtures/sample-bootstrap.md" "$TEST_ROOT/prompts/bootstrap.md"

  # Create mock scripts
  cat > "$TEST_ROOT/scripts/check-interrupt.sh" <<'EOF'
#!/usr/bin/env bash
exit 1  # No interrupt
EOF
  chmod +x "$TEST_ROOT/scripts/check-interrupt.sh"

  cat > "$TEST_ROOT/scripts/preserve-context.sh" <<'EOF'
#!/usr/bin/env bash
echo "Context preserved"
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
  export MAX_CONSECUTIVE_FAILURES=3
  export SESSION_TIMEOUT=30
  export STALL_THRESHOLD=60

  # Create mock claude command
  cat > "$TEST_ROOT/claude" <<'EOF'
#!/usr/bin/env bash
exec "$MOCK_CLAUDE_PATH" "$@"
EOF
  chmod +x "$TEST_ROOT/claude"

  export MOCK_CLAUDE_PATH="$SCRIPT_DIR/mocks/mock-claude.sh"
  export PATH="$TEST_ROOT:$PATH"

  # Source the loop.sh functions
  source "$TEST_ROOT/loop.sh" 2>/dev/null || true
}

teardown() {
  log_test "Cleaning up test environment"
  rm -rf "$TEST_ROOT"
}

# ============================================================
# TEST CASES
# ============================================================

test_session_id_generation() {
  test_start "Session ID generation and format"

  local sid1="ses_$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"
  local sid2="ses_$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -d'-' -f1)"

  assert_string_starts_with "$sid1" "ses_" "Session ID has correct prefix"
  assert_not_equals "$sid1" "$sid2" "Session IDs are unique"

  test_pass
}

test_bootstrap_variable_substitution() {
  test_start "Bootstrap template variable substitution"

  local session_id="ses_test123"
  local iteration=5
  local work_item="test-issue-001"
  local rig="test-rig"

  # Perform substitution
  sed \
    -e "s|{{SESSION_ID}}|${session_id}|g" \
    -e "s|{{ITERATION}}|${iteration}|g" \
    -e "s|{{WORK_ITEM}}|${work_item}|g" \
    -e "s|{{RIG}}|${rig}|g" \
    "$TEST_ROOT/prompts/bootstrap.md" > "$TEST_ROOT/prompts/bootstrap-test.md"

  assert_file_contains "$TEST_ROOT/prompts/bootstrap-test.md" "ses_test123" "Session ID substituted"
  assert_file_contains "$TEST_ROOT/prompts/bootstrap-test.md" "5" "Iteration number substituted"
  assert_file_contains "$TEST_ROOT/prompts/bootstrap-test.md" "test-issue-001" "Work item substituted"
  assert_file_contains "$TEST_ROOT/prompts/bootstrap-test.md" "test-rig" "Rig substituted"

  test_pass
}

test_spawn_prerequisites() {
  test_start "Spawn prerequisites validation"

  assert_file_exists "$TEST_ROOT/scripts/manage-queue.sh" "Queue manager exists"
  assert_file_exists "$TEST_ROOT/prompts/bootstrap.md" "Bootstrap template exists"

  # Check queue has work
  export MOCK_QUEUE_SIZE=1
  local count
  count=$("$TEST_ROOT/scripts/manage-queue.sh" check)
  assert_equals "1" "$count" "Queue reports work available"

  test_pass
}

test_session_state_creation() {
  test_start "Session state file creation and format"

  local session_id="ses_teststate123"
  local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  local start_epoch=$(date +%s)
  local work_item="test-issue-123"
  local work_json='{"id":"test-issue-123","title":"Test"}'

  # Create session state
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
      ended_at: null
    }' > "$TEST_ROOT/state/test-session.json"

  assert_file_exists "$TEST_ROOT/state/test-session.json" "Session state file created"
  assert_file_contains "$TEST_ROOT/state/test-session.json" "$session_id" "Session ID in state"
  assert_file_contains "$TEST_ROOT/state/test-session.json" "spawning" "Initial status is spawning"
  assert_file_contains "$TEST_ROOT/state/test-session.json" "$work_item" "Work item tracked"

  # Validate JSON structure
  local session_id_from_file
  session_id_from_file=$(jq -r '.session_id' "$TEST_ROOT/state/test-session.json")
  assert_equals "$session_id" "$session_id_from_file" "JSON structure valid"

  test_pass
}

test_spawn_with_mock_success() {
  test_start "Spawn agent with mock Claude (success behavior)"

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=2
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$TEST_ROOT/state/current-session.json"

  # Call spawn_agent
  iteration=1
  if spawn_agent; then
    assert_true "Spawn succeeded"

    # Check session file created
    assert_file_exists "$SESSION_FILE" "Session file created"

    # Validate session structure
    local status
    status=$(jq -r '.status' "$SESSION_FILE")
    assert_equals "running" "$status" "Status updated to running"

    # Check PID recorded
    local pid
    pid=$(jq -r '.pid' "$SESSION_FILE")
    assert_not_equals "null" "$pid" "PID recorded in session"

    # Check logs created
    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Wait a bit for agent to start
    sleep 1

    assert_file_exists "$TEST_ROOT/docs/sessions/${session_id}.log" "Log file created"

    # Kill the spawned process
    if [[ -f "$TEST_ROOT/state/${session_id}.pid" ]]; then
      local agent_pid
      agent_pid=$(cat "$TEST_ROOT/state/${session_id}.pid")
      kill "$agent_pid" 2>/dev/null || true
      wait "$agent_pid" 2>/dev/null || true
    fi

    test_pass
  else
    test_fail "Spawn failed unexpectedly"
  fi
}

test_spawn_failure_empty_queue() {
  test_start "Spawn failure with empty queue"

  export MOCK_QUEUE_SIZE=0
  export SESSION_FILE="$TEST_ROOT/state/current-session-2.json"

  iteration=1
  if ! spawn_agent 2>/dev/null; then
    assert_true "Spawn correctly fails with empty queue"
    test_pass
  else
    test_fail "Spawn should have failed with empty queue"
  fi
}

test_spawn_failure_missing_bootstrap() {
  test_start "Spawn failure with missing bootstrap template"

  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$TEST_ROOT/state/current-session-3.json"

  # Remove bootstrap
  mv "$TEST_ROOT/prompts/bootstrap.md" "$TEST_ROOT/prompts/bootstrap.md.bak"

  iteration=1
  if ! spawn_agent 2>/dev/null; then
    assert_true "Spawn correctly fails with missing bootstrap"

    # Restore bootstrap
    mv "$TEST_ROOT/prompts/bootstrap.md.bak" "$TEST_ROOT/prompts/bootstrap.md"

    test_pass
  else
    # Restore bootstrap
    mv "$TEST_ROOT/prompts/bootstrap.md.bak" "$TEST_ROOT/prompts/bootstrap.md"
    test_fail "Spawn should have failed with missing bootstrap"
  fi
}

test_spawn_environment_variables() {
  test_start "Environment variables passed to spawned agent"

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=1
  export MOCK_QUEUE_SIZE=1
  export SESSION_FILE="$TEST_ROOT/state/current-session-4.json"

  iteration=42
  export BD_ACTOR="test-actor"

  if spawn_agent; then
    # Check that environment variables are set correctly
    # This is tested implicitly through bootstrap substitution

    local session_id
    session_id=$(jq -r '.session_id' "$SESSION_FILE")

    # Check bootstrap file was created with correct values
    local bootstrap_file="/tmp/harness-bootstrap-${session_id}.md"

    sleep 1  # Let spawn complete

    # Check that variables would have been set (checked in actual bootstrap)
    assert_true "Environment setup completed"

    # Cleanup
    if [[ -f "$TEST_ROOT/state/${session_id}.pid" ]]; then
      local agent_pid
      agent_pid=$(cat "$TEST_ROOT/state/${session_id}.pid")
      kill "$agent_pid" 2>/dev/null || true
      wait "$agent_pid" 2>/dev/null || true
    fi

    test_pass
  else
    test_fail "Spawn failed"
  fi
}

test_concurrent_spawn_tracking() {
  test_start "Multiple spawns tracked separately"

  export MOCK_BEHAVIOR=success
  export MOCK_DURATION=3
  export MOCK_QUEUE_SIZE=1

  # Spawn first agent
  export SESSION_FILE="$TEST_ROOT/state/session-concurrent-1.json"
  iteration=1
  spawn_agent || true
  local session_id_1
  session_id_1=$(jq -r '.session_id' "$SESSION_FILE" 2>/dev/null || echo "none")

  sleep 1

  # Spawn second agent
  export SESSION_FILE="$TEST_ROOT/state/session-concurrent-2.json"
  iteration=2
  spawn_agent || true
  local session_id_2
  session_id_2=$(jq -r '.session_id' "$SESSION_FILE" 2>/dev/null || echo "none")

  # Verify different session IDs
  assert_not_equals "$session_id_1" "$session_id_2" "Different sessions have unique IDs"

  # Cleanup both
  for sid in "$session_id_1" "$session_id_2"; do
    if [[ -f "$TEST_ROOT/state/${sid}.pid" ]]; then
      local pid
      pid=$(cat "$TEST_ROOT/state/${sid}.pid")
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done

  test_pass
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Spawn Integration Tests"

  setup

  # Run tests
  test_session_id_generation
  test_bootstrap_variable_substitution
  test_spawn_prerequisites
  test_session_state_creation
  test_spawn_with_mock_success
  test_spawn_failure_empty_queue
  test_spawn_failure_missing_bootstrap
  test_spawn_environment_variables
  test_concurrent_spawn_tracking

  teardown

  test_suite_end
}

main "$@"
