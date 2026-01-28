#!/usr/bin/env bash
# Parse session events from stream-JSON logs
# Provides detailed analysis of session activity

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"

# Parse command line arguments
COMMAND="${1:-summary}"
SESSION_ID="${2:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
  cat <<EOF
Usage: $0 <command> [session_id]

Commands:
  summary <session_id>     - Show session summary
  tools <session_id>       - List all tool calls
  errors <session_id>      - Show all errors
  timeline <session_id>    - Show event timeline
  metrics <session_id>     - Calculate session metrics
  export <session_id>      - Export events to JSON
  watch <session_id>       - Watch session in real-time
  list                     - List all sessions
  latest                   - Show latest session summary

Examples:
  $0 summary ses_a1b2c3d4
  $0 tools ses_a1b2c3d4
  $0 watch ses_a1b2c3d4
  $0 latest
EOF
}

# Find log file for session
find_log_file() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  if [[ -f "$log_file" ]]; then
    echo "$log_file"
    return 0
  else
    echo "Session log not found: $log_file" >&2
    return 1
  fi
}

# Get latest session ID
get_latest_session() {
  if [[ -f "$STATE_DIR/current-session.json" ]]; then
    jq -r '.session_id' "$STATE_DIR/current-session.json"
  else
    # Find latest log file
    local latest_log
    latest_log=$(ls -t "$DOCS_DIR/sessions"/ses_*.log 2>/dev/null | head -1)
    if [[ -n "$latest_log" ]]; then
      basename "$latest_log" .log
    else
      echo ""
    fi
  fi
}

# Show session summary
cmd_summary() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  echo -e "${BLUE}Session Summary: ${session_id}${NC}"
  echo ""

  # Count events
  local message_starts=$(grep -c '"type":"message_start"' "$log_file" 2>/dev/null || echo 0)
  local message_stops=$(grep -c '"type":"message_stop"' "$log_file" 2>/dev/null || echo 0)
  local tool_uses=$(grep -c '"type":"tool_use"' "$log_file" 2>/dev/null || echo 0)
  local errors=$(grep -c '"type":"error"' "$log_file" 2>/dev/null || echo 0)
  local content_blocks=$(grep -c '"type":"content_block_start"' "$log_file" 2>/dev/null || echo 0)

  echo "Event Counts:"
  echo "  Messages: ${message_starts} started, ${message_stops} completed"
  echo "  Tool Calls: ${tool_uses}"
  echo "  Content Blocks: ${content_blocks}"
  echo "  Errors: ${errors}"
  echo ""

  # Show session state if available
  local session_file="$STATE_DIR/current-session.json"
  if [[ -f "$session_file" ]] && [[ "$(jq -r '.session_id' "$session_file")" == "$session_id" ]]; then
    echo "Session State:"
    jq -r '
      "  Status: \(.status)",
      "  Started: \(.started_at)",
      "  Work Item: \(.work.id)",
      "  PID: \(.pid // "N/A")"
    ' "$session_file"
    echo ""
  fi

  # Show metrics if available
  local metrics_file="$STATE_DIR/sessions/${session_id}/metrics.json"
  if [[ -f "$metrics_file" ]]; then
    echo "Metrics:"
    jq -r '
      "  Tokens: \(.api_usage.total_tokens) (\(.api_usage.input_tokens) in / \(.api_usage.output_tokens) out)",
      "  Duration: \(.session_metrics.duration_seconds)s",
      "  Turns: \(.session_metrics.turns)"
    ' "$metrics_file"
    echo ""
  fi

  # Show tool breakdown
  if [[ $tool_uses -gt 0 ]]; then
    echo "Tool Usage:"
    grep '"type":"tool_use"' "$log_file" 2>/dev/null | \
      jq -r '.name' | \
      sort | uniq -c | sort -rn | \
      awk '{printf "  %s: %d\n", $2, $1}'
    echo ""
  fi
}

# List all tool calls
cmd_tools() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  echo -e "${BLUE}Tool Calls: ${session_id}${NC}"
  echo ""

  grep '"type":"tool_use"' "$log_file" 2>/dev/null | \
    jq -r '[.timestamp, .name, .id] | @tsv' | \
    while IFS=$'\t' read -r timestamp name tool_id; do
      echo -e "${GREEN}${timestamp}${NC} ${name} (${tool_id})"
    done
}

# Show all errors
cmd_errors() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  echo -e "${RED}Errors: ${session_id}${NC}"
  echo ""

  local errors
  errors=$(grep '"type":"error"' "$log_file" 2>/dev/null)

  if [[ -z "$errors" ]]; then
    echo -e "${GREEN}No errors found${NC}"
    return 0
  fi

  echo "$errors" | jq -r '
    if .error.message then
      .timestamp + " - " + .error.message
    else
      .timestamp + " - " + (.error | tostring)
    end
  '
}

# Show event timeline
cmd_timeline() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  echo -e "${BLUE}Event Timeline: ${session_id}${NC}"
  echo ""

  # Parse major events
  cat "$log_file" | jq -r '
    select(.type == "message_start" or
           .type == "message_stop" or
           .type == "tool_use" or
           .type == "error") |
    [.timestamp, .type, (.name // .error.message // "")] | @tsv
  ' | while IFS=$'\t' read -r timestamp event_type detail; do
    case "$event_type" in
      message_start)
        echo -e "${BLUE}${timestamp}${NC} MESSAGE START"
        ;;
      message_stop)
        echo -e "${GREEN}${timestamp}${NC} MESSAGE STOP"
        ;;
      tool_use)
        echo -e "${YELLOW}${timestamp}${NC} TOOL: ${detail}"
        ;;
      error)
        echo -e "${RED}${timestamp}${NC} ERROR: ${detail}"
        ;;
    esac
  done
}

# Calculate and show metrics
cmd_metrics() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  echo -e "${BLUE}Session Metrics: ${session_id}${NC}"
  echo ""

  # Count events
  local total_events=$(wc -l < "$log_file")
  local message_starts=$(grep -c '"type":"message_start"' "$log_file" 2>/dev/null || echo 0)
  local message_stops=$(grep -c '"type":"message_stop"' "$log_file" 2>/dev/null || echo 0)
  local tool_uses=$(grep -c '"type":"tool_use"' "$log_file" 2>/dev/null || echo 0)
  local errors=$(grep -c '"type":"error"' "$log_file" 2>/dev/null || echo 0)

  echo "Event Statistics:"
  echo "  Total Events: ${total_events}"
  echo "  Messages: ${message_starts} started, ${message_stops} completed"
  echo "  Tool Calls: ${tool_uses}"
  echo "  Errors: ${errors}"
  echo ""

  # Calculate timing if possible
  local first_event=$(head -1 "$log_file" | jq -r '.timestamp // ""')
  local last_event=$(tail -1 "$log_file" | jq -r '.timestamp // ""')

  if [[ -n "$first_event" ]] && [[ -n "$last_event" ]]; then
    echo "Timing:"
    echo "  First Event: ${first_event}"
    echo "  Last Event: ${last_event}"
    echo ""
  fi

  # Tool usage breakdown
  if [[ $tool_uses -gt 0 ]]; then
    echo "Tool Usage Breakdown:"
    grep '"type":"tool_use"' "$log_file" 2>/dev/null | \
      jq -r '.name' | \
      sort | uniq -c | sort -rn | \
      awk '{printf "  %-20s %d calls\n", $2, $1}'
    echo ""
  fi

  # Check for usage stats in transcript
  local transcript="$HOME/.claude/transcripts/${session_id}.jsonl"
  if [[ -f "$transcript" ]]; then
    echo "API Usage (from transcript):"
    local usage
    usage=$(grep '"usage"' "$transcript" 2>/dev/null | jq -s '
      map(.usage) | {
        total_input: map(.input_tokens // 0) | add,
        total_output: map(.output_tokens // 0) | add
      }
    ')

    if [[ -n "$usage" ]]; then
      echo "$usage" | jq -r '
        "  Input Tokens: \(.total_input)",
        "  Output Tokens: \(.total_output)",
        "  Total Tokens: \(.total_input + .total_output)"
      '
    else
      echo "  No usage data available"
    fi
  fi
}

# Export events to JSON
cmd_export() {
  local session_id="$1"
  local log_file

  if ! log_file=$(find_log_file "$session_id"); then
    return 1
  fi

  local output_file="$DOCS_DIR/sessions/${session_id}-events.json"

  # Convert JSONL to JSON array
  jq -s '.' "$log_file" > "$output_file"

  echo "Exported events to: $output_file"
  echo "Total events: $(jq 'length' "$output_file")"
}

# Watch session in real-time
cmd_watch() {
  local session_id="$1"
  local log_file="$DOCS_DIR/sessions/${session_id}.log"

  echo -e "${BLUE}Watching session: ${session_id}${NC}"
  echo "Press Ctrl+C to stop"
  echo ""

  # Wait for log file if it doesn't exist yet
  if [[ ! -f "$log_file" ]]; then
    echo "Waiting for session to start..."
    while [[ ! -f "$log_file" ]]; do
      sleep 1
    done
  fi

  # Tail log file and format output
  tail -f "$log_file" 2>/dev/null | while IFS= read -r line; do
    local event_type
    event_type=$(echo "$line" | jq -r '.type // "unknown"' 2>/dev/null)

    case "$event_type" in
      message_start)
        echo -e "${BLUE}[MESSAGE START]${NC}"
        ;;
      message_stop)
        echo -e "${GREEN}[MESSAGE STOP]${NC}"
        ;;
      tool_use)
        local tool_name
        tool_name=$(echo "$line" | jq -r '.name // "unknown"')
        echo -e "${YELLOW}[TOOL]${NC} $tool_name"
        ;;
      content_block_delta)
        local text
        text=$(echo "$line" | jq -r '.delta.text // ""' 2>/dev/null)
        if [[ -n "$text" ]]; then
          echo -n "$text"
        fi
        ;;
      error)
        local error_msg
        error_msg=$(echo "$line" | jq -r '.error.message // "Unknown error"' 2>/dev/null)
        echo -e "${RED}[ERROR]${NC} $error_msg"
        ;;
    esac
  done
}

# List all sessions
cmd_list() {
  echo -e "${BLUE}Available Sessions:${NC}"
  echo ""

  # List session log files
  if ls "$DOCS_DIR/sessions"/ses_*.log >/dev/null 2>&1; then
    ls -t "$DOCS_DIR/sessions"/ses_*.log | while read -r log_file; do
      local session_id
      session_id=$(basename "$log_file" .log)

      local size=$(du -h "$log_file" | cut -f1)
      local modified=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$log_file")

      # Get event count
      local events=$(wc -l < "$log_file" | tr -d ' ')

      echo -e "${GREEN}${session_id}${NC}"
      echo "  Modified: $modified"
      echo "  Size: $size"
      echo "  Events: $events"
      echo ""
    done
  else
    echo "No sessions found"
  fi
}

# Show latest session summary
cmd_latest() {
  local latest_session
  latest_session=$(get_latest_session)

  if [[ -z "$latest_session" ]]; then
    echo "No sessions found"
    return 1
  fi

  cmd_summary "$latest_session"
}

# Main command dispatcher
main() {
  case "$COMMAND" in
    summary)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_summary "$SESSION_ID"
      ;;
    tools)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_tools "$SESSION_ID"
      ;;
    errors)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_errors "$SESSION_ID"
      ;;
    timeline)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_timeline "$SESSION_ID"
      ;;
    metrics)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_metrics "$SESSION_ID"
      ;;
    export)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_export "$SESSION_ID"
      ;;
    watch)
      if [[ -z "$SESSION_ID" ]]; then
        echo "Error: session_id required" >&2
        usage
        exit 1
      fi
      cmd_watch "$SESSION_ID"
      ;;
    list)
      cmd_list
      ;;
    latest)
      cmd_latest
      ;;
    help|--help|-h)
      usage
      ;;
    *)
      echo "Error: Unknown command: $COMMAND" >&2
      echo ""
      usage
      exit 1
      ;;
  esac
}

main "$@"
