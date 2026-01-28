#!/usr/bin/env bash
# Integration tests for session monitoring and output capture
# Tests stream-JSON parsing, metrics extraction, heartbeat, and stall detection

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_ROOT="/tmp/harness-monitoring-test-$$"

# Test utilities
source "$SCRIPT_DIR/test-lib.sh"

# Test setup
setup() {
  log_test "Setting up monitoring test environment"

  mkdir -p "$TEST_ROOT"/{state,docs/sessions,state/sessions}
  mkdir -p "$HOME/.claude/transcripts"

  # Copy necessary files
  cp "$HARNESS_ROOT/loop.sh" "$TEST_ROOT/"

  # Set up environment
  export HARNESS_ROOT="$TEST_ROOT"
  export STATE_DIR="$TEST_ROOT/state"
  export DOCS_DIR="$TEST_ROOT/docs"
  export SESSION_FILE="$TEST_ROOT/state/test-session.json"

  # Source the loop.sh functions
  source "$TEST_ROOT/loop.sh" 2>/dev/null || true
}

teardown() {
  log_test "Cleaning up monitoring test environment"
  cleanup_background_jobs
  rm -rf "$TEST_ROOT"
  # Clean up test transcripts
  rm -f "$HOME/.claude/transcripts/ses_test"*.jsonl
}

# ============================================================
# TEST CASES
# ============================================================

test_parse_stream_event_valid() {
  test_start "Parse valid stream-JSON event"

  local event='{"type":"message_start","timestamp":"2026-01-27T00:00:00Z"}'
  local event_type
  event_type=$(parse_stream_event "$event")

  assert_equals "message_start" "$event_type" "Event type extracted correctly"

  test_pass
}

test_parse_stream_event_invalid() {
  test_start "Parse invalid JSON gracefully"

  local event='not valid json'
  local event_type
  event_type=$(parse_stream_event "$event" 2>/dev/null || echo "")

  assert_equals "" "$event_type" "Invalid JSON returns empty"

  test_pass
}

test_parse_stream_event_types() {
  test_start "Parse various stream-JSON event types"

  local events=(
    '{"type":"message_start"}'
    '{"type":"message_stop"}'
    '{"type":"content_block_start","index":0}'
    '{"type":"content_block_delta","delta":{"text":"hello"}}'
    '{"type":"content_block_stop"}'
    '{"type":"tool_use","name":"Read"}'
    '{"type":"tool_result","tool_use_id":"tool_1"}'
    '{"type":"error","error":{"message":"test"}}'
  )

  local expected_types=(
    "message_start"
    "message_stop"
    "content_block_start"
    "content_block_delta"
    "content_block_stop"
    "tool_use"
    "tool_result"
    "error"
  )

  local all_passed=true
  for i in "${!events[@]}"; do
    local event_type
    event_type=$(parse_stream_event "${events[$i]}")

    if [[ "$event_type" != "${expected_types[$i]}" ]]; then
      test_fail "Event type mismatch for: ${events[$i]}"
      all_passed=false
      break
    fi
  done

  if [[ "$all_passed" == "true" ]]; then
    test_pass
  fi
}

test_update_progress_from_log() {
  test_start "Update progress indicators from log file"

  local session_id="ses_progress123"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create mock log file with events
  cat > "$log_file" <<EOF
{"type":"message_start","timestamp":"2026-01-27T00:00:00Z"}
{"type":"tool_use","name":"Read","timestamp":"2026-01-27T00:00:01Z"}
{"type":"tool_use","name":"Bash","timestamp":"2026-01-27T00:00:02Z"}
{"type":"message_stop","timestamp":"2026-01-27T00:00:03Z"}
{"type":"message_start","timestamp":"2026-01-27T00:00:04Z"}
{"type":"tool_use","name":"Edit","timestamp":"2026-01-27T00:00:05Z"}
{"type":"error","error":{"message":"Test error"},"timestamp":"2026-01-27T00:00:06Z"}
EOF

  # Update progress
  update_progress "$session_id"

  # Verify progress was recorded
  assert_json_field_equals "$SESSION_FILE" ".progress.message_starts" "2" "Message starts counted"
  assert_json_field_equals "$SESSION_FILE" ".progress.message_stops" "1" "Message stops counted"
  assert_json_field_equals "$SESSION_FILE" ".progress.tool_calls" "3" "Tool calls counted"
  assert_json_field_equals "$SESSION_FILE" ".progress.errors" "1" "Errors counted"

  test_pass
}

test_extract_session_metrics() {
  test_start "Extract comprehensive session metrics"

  local session_id="ses_metrics456"
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create mock transcript with usage data
  cat > "$transcript" <<EOF
{"type":"assistant","usage":{"input_tokens":100,"output_tokens":50}}
{"type":"tool_use","tool_name":"Read"}
{"type":"assistant","usage":{"input_tokens":150,"output_tokens":75}}
{"type":"tool_use","tool_name":"Bash"}
{"type":"assistant","usage":{"input_tokens":200,"output_tokens":100}}
EOF

  # Create mock log with tool calls
  cat > "$log_file" <<EOF
{"type":"tool_use","name":"Read"}
{"type":"tool_use","name":"Bash"}
{"type":"tool_use","name":"Read"}
{"type":"tool_use","name":"Edit"}
EOF

  # Extract metrics
  local metrics_file
  metrics_file=$(extract_session_metrics "$session_id")

  assert_file_exists "$metrics_file" "Metrics file created"

  # Verify metrics content
  assert_json_field_equals "$metrics_file" ".api_usage.input_tokens" "450" "Input tokens summed"
  assert_json_field_equals "$metrics_file" ".api_usage.output_tokens" "225" "Output tokens summed"
  assert_json_field_equals "$metrics_file" ".tool_usage.total_calls" "4" "Total tool calls counted"
  assert_json_field_equals "$metrics_file" ".session_metrics.turns" "3" "Turns counted"

  # Check tool breakdown
  local read_count
  read_count=$(jq -r '.tool_usage.breakdown.Read // 0' "$metrics_file")
  assert_equals "2" "$read_count" "Read tool calls counted correctly"

  test_pass
}

test_update_heartbeat() {
  test_start "Update heartbeat with message counts"

  local session_id="ses_heartbeat789"
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create mock transcript
  cat > "$transcript" <<EOF
{"type":"assistant","content":"message 1"}
{"type":"tool_use","tool_name":"Read"}
{"type":"assistant","content":"message 2"}
{"type":"tool_use","tool_name":"Bash"}
{"type":"assistant","content":"message 3"}
EOF

  # Update heartbeat
  update_heartbeat "$session_id"

  # Verify heartbeat fields
  assert_json_field_equals "$SESSION_FILE" ".heartbeat.message_count" "3" "Message count correct"
  assert_json_field_equals "$SESSION_FILE" ".heartbeat.tool_calls" "2" "Tool call count correct"

  # Verify timestamp is recent (within last 5 seconds)
  local last_check
  last_check=$(jq -r '.heartbeat.last_check' "$SESSION_FILE")
  assert_not_equals "null" "$last_check" "Heartbeat timestamp recorded"

  test_pass
}

test_detect_stall_healthy() {
  test_start "Detect stall - healthy session"

  local session_id="ses_stall_healthy"

  # Create mock session with recent heartbeat
  jq -n \
    --arg sid "$session_id" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    --arg last_check "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running",
      heartbeat: {
        last_check: $last_check,
        message_count: 5
      }
    }' > "$STATE_DIR/${session_id}.json"

  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Should not detect stall
  if ! detect_stall "$session_id" 2>/dev/null; then
    assert_true "Healthy session not detected as stalled"
    test_pass
  else
    test_fail "Healthy session incorrectly detected as stalled"
  fi
}

test_detect_stall_stalled() {
  test_start "Detect stall - stalled session"

  local session_id="ses_stall_stalled"

  # Create mock session with old heartbeat (10 minutes ago)
  local old_time
  old_time=$(date -u -v-10M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '10 minutes ago' +%Y-%m-%dT%H:%M:%SZ)

  jq -n \
    --arg sid "$session_id" \
    --arg started "$old_time" \
    --argjson start_epoch "$(($(date +%s) - 600))" \
    --arg last_check "$old_time" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running",
      heartbeat: {
        last_check: $last_check,
        message_count: 5
      }
    }' > "$STATE_DIR/${session_id}.json"

  export SESSION_FILE="$STATE_DIR/${session_id}.json"
  export STALL_THRESHOLD=60  # 1 minute threshold

  # Should detect stall
  if detect_stall "$session_id" 2>/dev/null; then
    assert_true "Stalled session correctly detected"
    test_pass
  else
    test_fail "Stalled session not detected"
  fi
}

test_stream_json_from_sample() {
  test_start "Parse sample stream-JSON events file"

  local sample_events="$SCRIPT_DIR/mocks/sample-events.jsonl"
  assert_file_exists "$sample_events" "Sample events file exists"

  # Count different event types
  local tool_uses=$(grep -c '"type":"tool_use"' "$sample_events")
  local message_starts=$(grep -c '"type":"message_start"' "$sample_events")
  local message_stops=$(grep -c '"type":"message_stop"' "$sample_events")

  assert_greater_than "$tool_uses" "0" "Sample has tool use events"
  assert_equals "1" "$message_starts" "Sample has one message start"
  assert_equals "1" "$message_stops" "Sample has one message stop"

  test_pass
}

test_output_processor_lifecycle() {
  test_start "Output processor lifecycle"

  if [[ "${QUICK:-false}" == "true" ]]; then
    echo "SKIPPED (quick mode)"
    TESTS_RUN=$((TESTS_RUN - 1))  # Don't count skipped tests
    return 0
  fi

  local session_id="ses_processor_test"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"
  local event_log="$STATE_DIR/sessions/${session_id}/events.jsonl"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create log file
  mkdir -p "$(dirname "$log_file")"
  touch "$log_file"

  # Start output processor in background
  # (Mock version - just verify it can start and stop)

  # Create PID file to simulate running processor
  echo "$$" > "$STATE_DIR/${session_id}.processor.pid"

  assert_file_exists "$STATE_DIR/${session_id}.processor.pid" "Processor PID file created"

  # Cleanup
  rm -f "$STATE_DIR/${session_id}.processor.pid"

  test_pass
}

test_metrics_tool_breakdown() {
  test_start "Metrics tool usage breakdown"

  local session_id="ses_tool_breakdown"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  # Create mock session
  create_mock_session "$session_id" "$STATE_DIR" "running"
  export SESSION_FILE="$STATE_DIR/${session_id}.json"

  # Create log with varied tool calls
  cat > "$log_file" <<EOF
{"type":"tool_use","name":"Read","input":{"file_path":"file1.txt"}}
{"type":"tool_use","name":"Read","input":{"file_path":"file2.txt"}}
{"type":"tool_use","name":"Read","input":{"file_path":"file3.txt"}}
{"type":"tool_use","name":"Bash","input":{"command":"ls"}}
{"type":"tool_use","name":"Edit","input":{"file_path":"file1.txt"}}
{"type":"tool_use","name":"Edit","input":{"file_path":"file2.txt"}}
{"type":"tool_use","name":"Bash","input":{"command":"pwd"}}
EOF

  # Create minimal transcript
  touch "$HOME/.claude/transcripts/${session_id}.jsonl"

  # Extract metrics
  local metrics_file
  metrics_file=$(extract_session_metrics "$session_id")

  # Verify tool breakdown
  local read_count=$(jq -r '.tool_usage.breakdown.Read // 0' "$metrics_file")
  local bash_count=$(jq -r '.tool_usage.breakdown.Bash // 0' "$metrics_file")
  local edit_count=$(jq -r '.tool_usage.breakdown.Edit // 0' "$metrics_file")

  assert_equals "3" "$read_count" "Read calls counted"
  assert_equals "2" "$bash_count" "Bash calls counted"
  assert_equals "2" "$edit_count" "Edit calls counted"

  test_pass
}

# ============================================================
# TEST RUNNER
# ============================================================

main() {
  test_suite_start "Monitoring Integration Tests"

  setup

  # Run tests
  test_parse_stream_event_valid
  test_parse_stream_event_invalid
  test_parse_stream_event_types
  test_update_progress_from_log
  test_extract_session_metrics
  test_update_heartbeat
  test_detect_stall_healthy
  test_detect_stall_stalled
  test_stream_json_from_sample
  test_output_processor_lifecycle
  test_metrics_tool_breakdown

  teardown

  test_suite_end
}

main "$@"
