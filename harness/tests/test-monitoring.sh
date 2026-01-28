#!/usr/bin/env bash
# Test suite for session monitoring and output capture

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$HARNESS_ROOT/tests"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"

# Source loop.sh functions
source "$HARNESS_ROOT/loop.sh"

# Test utilities
TESTS_PASSED=0
TESTS_FAILED=0

test_passed() {
  local test_name="$1"
  echo -e "${GREEN}✓${NC} $test_name"
  TESTS_PASSED=$((TESTS_PASSED + 1))
}

test_failed() {
  local test_name="$1"
  local reason="${2:-}"
  echo -e "${RED}✗${NC} $test_name"
  if [[ -n "$reason" ]]; then
    echo "  Reason: $reason"
  fi
  TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Test setup
setup_test_environment() {
  export HARNESS_TEST=1
  mkdir -p "$STATE_DIR/test" "$DOCS_DIR/sessions"

  # Create mock session file
  TEST_SESSION_ID="ses_test123"
  TEST_SESSION_FILE="$STATE_DIR/test-session.json"

  jq -n \
    --arg sid "$TEST_SESSION_ID" \
    --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson start_epoch "$(date +%s)" \
    '{
      session_id: $sid,
      started_at: $started,
      start_epoch: $start_epoch,
      status: "running",
      pid: 99999
    }' > "$TEST_SESSION_FILE"

  export SESSION_FILE="$TEST_SESSION_FILE"
}

cleanup_test_environment() {
  rm -rf "$STATE_DIR/test"
  rm -f "$TEST_SESSION_FILE"
  rm -f "$DOCS_DIR/sessions/$TEST_SESSION_ID".*
  rm -rf "$STATE_DIR/sessions/$TEST_SESSION_ID"
}

# Test: Parse stream-JSON event
test_parse_stream_event() {
  local test_name="parse_stream_event"

  # Valid JSON
  local event_type
  event_type=$(parse_stream_event '{"type":"message_start","timestamp":"2026-01-27T00:00:00Z"}')

  if [[ "$event_type" == "message_start" ]]; then
    test_passed "$test_name"
  else
    test_failed "$test_name" "Expected 'message_start', got '$event_type'"
  fi
}

# Test: Parse invalid JSON
test_parse_invalid_json() {
  local test_name="parse_stream_event (invalid JSON)"

  local event_type
  event_type=$(parse_stream_event 'not json' || echo "")

  if [[ -z "$event_type" ]]; then
    test_passed "$test_name"
  else
    test_failed "$test_name" "Should return empty for invalid JSON"
  fi
}

# Test: Update progress
test_update_progress() {
  local test_name="update_progress"

  # Create mock log file
  local log_file="$DOCS_DIR/sessions/${TEST_SESSION_ID}.log"
  cat > "$log_file" <<EOF
{"type":"message_start","timestamp":"2026-01-27T00:00:00Z"}
{"type":"tool_use","name":"Read","timestamp":"2026-01-27T00:00:01Z"}
{"type":"message_stop","timestamp":"2026-01-27T00:00:02Z"}
{"type":"error","error":{"message":"Test error"},"timestamp":"2026-01-27T00:00:03Z"}
EOF

  update_progress "$TEST_SESSION_ID"

  # Check if progress was updated
  if jq -e '.progress' "$TEST_SESSION_FILE" >/dev/null 2>&1; then
    local msg_starts
    msg_starts=$(jq -r '.progress.message_starts' "$TEST_SESSION_FILE")

    if [[ "$msg_starts" -eq 1 ]]; then
      test_passed "$test_name"
    else
      test_failed "$test_name" "Expected 1 message_start, got $msg_starts"
    fi
  else
    test_failed "$test_name" "Progress not updated in session file"
  fi
}

# Test: Extract metrics
test_extract_metrics() {
  local test_name="extract_session_metrics"

  # Create mock log file
  local log_file="$DOCS_DIR/sessions/${TEST_SESSION_ID}.log"
  cat > "$log_file" <<EOF
{"type":"message_start"}
{"type":"tool_use","name":"Read"}
{"type":"tool_use","name":"Bash"}
{"type":"tool_use","name":"Read"}
{"type":"message_stop"}
EOF

  # Create mock transcript
  local transcript="$HOME/.claude/transcripts/${TEST_SESSION_ID}.jsonl"
  mkdir -p "$(dirname "$transcript")"
  cat > "$transcript" <<EOF
{"type":"assistant","usage":{"input_tokens":100,"output_tokens":50}}
{"type":"tool_use","tool_name":"Read"}
EOF

  local metrics_file
  metrics_file=$(extract_session_metrics "$TEST_SESSION_ID")

  if [[ -f "$metrics_file" ]]; then
    local tool_calls
    tool_calls=$(jq -r '.tool_usage.total_calls' "$metrics_file")

    if [[ "$tool_calls" -eq 3 ]]; then
      test_passed "$test_name"
    else
      test_failed "$test_name" "Expected 3 tool calls, got $tool_calls"
    fi
  else
    test_failed "$test_name" "Metrics file not created"
  fi

  # Cleanup
  rm -f "$transcript"
}

# Test: Detect stall (not stalled)
test_detect_stall_healthy() {
  local test_name="detect_stall (healthy)"

  # Update heartbeat to current time
  jq \
    --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '.heartbeat.last_check = $timestamp' \
    "$TEST_SESSION_FILE" > "$TEST_SESSION_FILE.tmp"
  mv "$TEST_SESSION_FILE.tmp" "$TEST_SESSION_FILE"

  if detect_stall "$TEST_SESSION_ID"; then
    test_failed "$test_name" "Should not detect stall for recent heartbeat"
  else
    test_passed "$test_name"
  fi
}

# Test: Detect stall (stalled)
test_detect_stall_stalled() {
  local test_name="detect_stall (stalled)"

  # Set old heartbeat (10 minutes ago)
  local old_time
  old_time=$(date -u -v-10M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '10 minutes ago' +%Y-%m-%dT%H:%M:%SZ)

  jq \
    --arg timestamp "$old_time" \
    '.heartbeat.last_check = $timestamp' \
    "$TEST_SESSION_FILE" > "$TEST_SESSION_FILE.tmp"
  mv "$TEST_SESSION_FILE.tmp" "$TEST_SESSION_FILE"

  if detect_stall "$TEST_SESSION_ID"; then
    test_passed "$test_name"
  else
    test_failed "$test_name" "Should detect stall for old heartbeat"
  fi
}

# Test: Update heartbeat
test_update_heartbeat() {
  local test_name="update_heartbeat"

  # Create mock transcript
  local transcript="$HOME/.claude/transcripts/${TEST_SESSION_ID}.jsonl"
  mkdir -p "$(dirname "$transcript")"
  cat > "$transcript" <<EOF
{"type":"assistant"}
{"type":"tool_use"}
{"type":"assistant"}
EOF

  update_heartbeat "$TEST_SESSION_ID"

  if jq -e '.heartbeat' "$TEST_SESSION_FILE" >/dev/null 2>&1; then
    local msg_count
    msg_count=$(jq -r '.heartbeat.message_count' "$TEST_SESSION_FILE")

    if [[ "$msg_count" -eq 2 ]]; then
      test_passed "$test_name"
    else
      test_failed "$test_name" "Expected 2 messages, got $msg_count"
    fi
  else
    test_failed "$test_name" "Heartbeat not updated"
  fi

  # Cleanup
  rm -f "$transcript"
}

# Test: Parse session events script
test_parse_session_events_script() {
  local test_name="parse-session-events.sh (list)"

  # Create a test log file
  local log_file="$DOCS_DIR/sessions/${TEST_SESSION_ID}.log"
  echo '{"type":"message_start"}' > "$log_file"

  if "$HARNESS_ROOT/scripts/parse-session-events.sh" list >/dev/null 2>&1; then
    test_passed "$test_name"
  else
    test_failed "$test_name" "Script failed to execute"
  fi
}

# Test: Mock stream-JSON parsing
test_mock_stream_json() {
  local test_name="Mock stream-JSON events"

  local log_file="$DOCS_DIR/sessions/${TEST_SESSION_ID}.log"
  cat > "$log_file" <<EOF
{"type":"message_start","message":{"id":"msg_1","type":"message"},"timestamp":"2026-01-27T00:00:00Z"}
{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""},"timestamp":"2026-01-27T00:00:01Z"}
{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"},"timestamp":"2026-01-27T00:00:02Z"}
{"type":"tool_use","id":"tool_1","name":"Read","input":{"file_path":"test.txt"},"timestamp":"2026-01-27T00:00:03Z"}
{"type":"content_block_stop","index":0,"timestamp":"2026-01-27T00:00:04Z"}
{"type":"message_stop","timestamp":"2026-01-27T00:00:05Z"}
EOF

  # Count event types
  local tool_uses
  tool_uses=$(grep -c '"type":"tool_use"' "$log_file")

  if [[ "$tool_uses" -eq 1 ]]; then
    test_passed "$test_name"
  else
    test_failed "$test_name" "Expected 1 tool_use event, got $tool_uses"
  fi
}

# Main test runner
main() {
  echo "Running session monitoring tests..."
  echo ""

  setup_test_environment

  # Run tests
  test_parse_stream_event
  test_parse_invalid_json
  test_update_progress
  test_extract_metrics
  test_detect_stall_healthy
  test_detect_stall_stalled
  test_update_heartbeat
  test_parse_session_events_script
  test_mock_stream_json

  cleanup_test_environment

  # Summary
  echo ""
  echo "Test Results:"
  echo "  Passed: $TESTS_PASSED"
  echo "  Failed: $TESTS_FAILED"

  if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
  else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
  fi
}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

main "$@"
