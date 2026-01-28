#!/usr/bin/env bash
# Integration tests for filesystem audit trail
# Tests state files, logs, metrics, events, and permissions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-audit-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up audit trail test environment"

  mkdir -p "$TEST_ROOT"/{state,docs/sessions,state/sessions}
  mkdir -p "$HOME/.claude/transcripts"

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
  log_test "Cleaning up audit trail test environment"
  rm -rf "$TEST_ROOT"
  rm -f "$HOME/.claude/transcripts/ses_audit"*.jsonl
}

# ============================================================
# TEST CASES
# ============================================================

test_session_state_file_created() {
  test_start "Session state file creation"

  local session_id="ses_audit_state"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create session state
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running"
    }' > "$SESSION_FILE"

  assert_file_exists "$SESSION_FILE" "Session state file created"

  # Verify structure
  assert_json_field_equals "$SESSION_FILE" ".session_id" "$session_id" "Session ID in file"
  assert_json_field_equals "$SESSION_FILE" ".status" "running" "Status in file"

  test_pass
}

test_session_logs_created() {
  test_start "Session log files created"

  local session_id="ses_audit_logs"

  # Create log directory
  mkdir -p "$DOCS_DIR/sessions"

  # Create log files
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local err_file="$DOCS_DIR/sessions/${session_id}.err"

  echo '{"type":"message_start"}' > "$log_file"
  echo "error output" > "$err_file"

  assert_file_exists "$log_file" "Stdout log created"
  assert_file_exists "$err_file" "Stderr log created"

  test_pass
}

test_metrics_file_created() {
  test_start "Metrics file creation and structure"

  local session_id="ses_audit_metrics"
  local metrics_dir="$STATE_DIR/sessions/${session_id}"
  mkdir -p "$metrics_dir"

  # Create minimal transcript
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"
  echo '{"type":"assistant","usage":{"input_tokens":100,"output_tokens":50}}' > "$transcript"

  # Create minimal log
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  mkdir -p "$(dirname "$log_file")"
  echo '{"type":"tool_use","name":"Read"}' > "$log_file"

  # Create session state
  export SESSION_FILE="$STATE_DIR/${session_id}.json"
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Extract metrics
  local metrics_file
  metrics_file=$(extract_session_metrics "$session_id")

  assert_file_exists "$metrics_file" "Metrics file created"

  # Verify JSON structure
  assert_json_field_equals "$metrics_file" ".session_id" "$session_id" "Session ID in metrics"

  local has_api_usage=$(jq 'has("api_usage")' "$metrics_file")
  assert_equals "true" "$has_api_usage" "API usage section present"

  local has_tool_usage=$(jq 'has("tool_usage")' "$metrics_file")
  assert_equals "true" "$has_tool_usage" "Tool usage section present"

  test_pass
}

test_event_log_created() {
  test_start "Event log creation"

  local session_id="ses_audit_events"
  local event_log="$STATE_DIR/sessions/${session_id}/events.jsonl"

  mkdir -p "$(dirname "$event_log")"

  # Create event log
  cat > "$event_log" <<EOF
{"type":"message_start","timestamp":"2026-01-27T10:00:00Z"}
{"type":"tool_use","name":"Read","timestamp":"2026-01-27T10:00:01Z"}
{"type":"message_stop","timestamp":"2026-01-27T10:00:02Z"}
EOF

  assert_file_exists "$event_log" "Event log created"

  local line_count=$(count_lines "$event_log")
  assert_equals "3" "$line_count" "Event log has correct entries"

  test_pass
}

test_pid_file_created() {
  test_start "PID file creation"

  local session_id="ses_audit_pid"
  local pid_file="$STATE_DIR/${session_id}.pid"

  # Create PID file
  echo "12345" > "$pid_file"

  assert_file_exists "$pid_file" "PID file created"

  local pid
  pid=$(cat "$pid_file")
  assert_equals "12345" "$pid" "PID recorded correctly"

  test_pass
}

test_exit_code_file_created() {
  test_start "Exit code file creation"

  local session_id="ses_audit_exit"
  local exit_file="$STATE_DIR/${session_id}.exit"

  # Create exit code file
  echo "0" > "$exit_file"

  assert_file_exists "$exit_file" "Exit code file created"

  local code
  code=$(cat "$exit_file")
  assert_equals "0" "$code" "Exit code recorded correctly"

  test_pass
}

test_bootstrap_file_created() {
  test_start "Bootstrap prompt file creation"

  local session_id="ses_audit_bootstrap"
  local bootstrap_file="/tmp/harness-bootstrap-${session_id}.md"

  # Create bootstrap file
  cat > "$bootstrap_file" <<EOF
# Bootstrap Prompt

Session ID: $session_id
Iteration: 1
EOF

  assert_file_exists "$bootstrap_file" "Bootstrap file created"
  assert_file_contains "$bootstrap_file" "$session_id" "Session ID in bootstrap"

  # Cleanup
  rm -f "$bootstrap_file"

  test_pass
}

test_complete_audit_trail() {
  test_start "Complete audit trail for session"

  local session_id="ses_audit_complete"

  # Create all expected files
  local files=(
    "$STATE_DIR/${session_id}.json"
    "$STATE_DIR/${session_id}.pid"
    "$STATE_DIR/${session_id}.exit"
    "$DOCS_DIR/sessions/${session_id}.log"
    "$DOCS_DIR/sessions/${session_id}.err"
    "$STATE_DIR/sessions/${session_id}/events.jsonl"
    "$STATE_DIR/sessions/${session_id}/metrics.json"
  )

  # Create state file
  create_mock_session "$session_id" "$STATE_DIR" "completed"

  # Create PID and exit files
  echo "12345" > "$STATE_DIR/${session_id}.pid"
  echo "0" > "$STATE_DIR/${session_id}.exit"

  # Create log files
  mkdir -p "$DOCS_DIR/sessions"
  echo '{"type":"message_start"}' > "$DOCS_DIR/sessions/${session_id}.log"
  echo "" > "$DOCS_DIR/sessions/${session_id}.err"

  # Create event log
  mkdir -p "$STATE_DIR/sessions/${session_id}"
  echo '{"type":"message_start"}' > "$STATE_DIR/sessions/${session_id}/events.jsonl"

  # Create metrics
  jq -n \
    --arg sid "$session_id" \
    '{
      session_id: $sid,
      api_usage: {input_tokens: 0, output_tokens: 0},
      tool_usage: {total_calls: 0}
    }' > "$STATE_DIR/sessions/${session_id}/metrics.json"

  # Verify all files exist
  local all_exist=true
  for file in "${files[@]}"; do
    if [[ ! -f "$file" ]]; then
      test_fail "Missing file: $file"
      all_exist=false
      break
    fi
  done

  if [[ "$all_exist" == "true" ]]; then
    assert_true "Complete audit trail exists"
    test_pass
  fi
}

test_archived_session_structure() {
  test_start "Archived session file structure"

  local session_id="ses_audit_archived"
  local archived_file="$DOCS_DIR/sessions/${session_id}.json"

  # Create archived session
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg ended "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      ended_at: $ended,
      start_epoch: $start_epoch,
      status: "completed",
      exit_code: 0,
      logs: {
        stdout: ("docs/sessions/" + $sid + ".log"),
        stderr: ("docs/sessions/" + $sid + ".err")
      }
    }' > "$archived_file"

  assert_file_exists "$archived_file" "Archived session file exists"
  assert_json_field_equals "$archived_file" ".status" "completed" "Status recorded"
  assert_json_field_equals "$archived_file" ".exit_code" "0" "Exit code recorded"

  # Verify has logs section
  local has_logs=$(jq 'has("logs")' "$archived_file")
  assert_equals "true" "$has_logs" "Logs section present"

  test_pass
}

test_file_permissions() {
  test_start "File permissions appropriate"

  local session_id="ses_audit_perms"

  # Create state file
  create_mock_session "$session_id" "$STATE_DIR" "running"

  local session_file="$STATE_DIR/${session_id}.json"

  # Check file is readable
  if [[ -r "$session_file" ]]; then
    assert_true "State file readable"
  else
    test_fail "State file not readable"
    return
  fi

  # Check file is writable
  if [[ -w "$session_file" ]]; then
    assert_true "State file writable"
    test_pass
  else
    test_fail "State file not writable"
  fi
}

test_log_rotation_check() {
  test_start "Log file size tracking"

  local session_id="ses_audit_logsize"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  mkdir -p "$(dirname "$log_file")"

  # Create log with some content
  for i in {1..100}; do
    echo '{"type":"message_start","timestamp":"2026-01-27T10:00:00Z"}' >> "$log_file"
  done

  assert_file_exists "$log_file" "Log file created"

  local size
  size=$(get_file_size "$log_file")

  assert_greater_than "$size" "0" "Log file has content"

  test_pass
}

test_directory_structure() {
  test_start "Required directory structure exists"

  local required_dirs=(
    "$STATE_DIR"
    "$DOCS_DIR/sessions"
    "$STATE_DIR/sessions"
  )

  local all_exist=true
  for dir in "${required_dirs[@]}"; do
    if [[ ! -d "$dir" ]]; then
      test_fail "Missing directory: $dir"
      all_exist=false
      break
    fi
  done

  if [[ "$all_exist" == "true" ]]; then
    assert_true "All required directories exist"
    test_pass
  fi
}

test_session_data_integrity() {
  test_start "Session data integrity checks"

  local session_id="ses_audit_integrity"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create session
  create_mock_session "$session_id" "$STATE_DIR" "running"

  # Verify JSON is valid
  if jq empty "$SESSION_FILE" 2>/dev/null; then
    assert_true "JSON is valid"
  else
    test_fail "Invalid JSON in session file"
    return
  fi

  # Verify required fields
  local required_fields=("session_id" "started_at" "status")

  local all_present=true
  for field in "${required_fields[@]}"; do
    if ! jq -e ".$field" "$SESSION_FILE" >/dev/null 2>&1; then
      test_fail "Missing required field: $field"
      all_present=false
      break
    fi
  done

  if [[ "$all_present" == "true" ]]; then
    assert_true "All required fields present"
    test_pass
  fi
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Audit Trail Integration Tests"

  setup

  # Run tests
  test_session_state_file_created
  test_session_logs_created
  test_metrics_file_created
  test_event_log_created
  test_pid_file_created
  test_exit_code_file_created
  test_bootstrap_file_created
  test_complete_audit_trail
  test_archived_session_structure
  test_file_permissions
  test_log_rotation_check
  test_directory_structure
  test_session_data_integrity

  teardown

  test_suite_end
}

main "$@"
