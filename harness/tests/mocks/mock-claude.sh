#!/usr/bin/env bash
# Mock Claude Code CLI for testing
# Simulates agent behavior with stream-JSON output

set -euo pipefail

# Mock behavior configuration (can be set via environment)
MOCK_BEHAVIOR=${MOCK_BEHAVIOR:-success}
MOCK_DURATION=${MOCK_DURATION:-5}
MOCK_TOOL_CALLS=${MOCK_TOOL_CALLS:-3}
MOCK_ERROR_MESSAGE=${MOCK_ERROR_MESSAGE:-"Mock error"}

# Parse arguments
session_id=""
output_format="text"
prompt=""
bootstrap_file=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -p)
      prompt="$2"
      shift 2
      ;;
    --session-id)
      session_id="$2"
      shift 2
      ;;
    --output-format)
      output_format="$2"
      shift 2
      ;;
    --append-system-prompt-file)
      bootstrap_file="$2"
      shift 2
      ;;
    --allowedTools|--max-turns|--max-budget-usd|--verbose|--settings)
      # Ignore these arguments
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

# Generate timestamp
timestamp() {
  date -u +%Y-%m-%dT%H:%M:%SZ
}

# Simulate different behaviors based on MOCK_BEHAVIOR
case "$MOCK_BEHAVIOR" in
  success)
    # Simulate successful agent execution
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5"},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""},"timestamp":"'$(timestamp)'"}'
      sleep 0.2

      echo '{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I am starting work on the assigned task."},"timestamp":"'$(timestamp)'"}'
      sleep 0.3

      # Simulate tool calls
      for i in $(seq 1 $MOCK_TOOL_CALLS); do
        local tool_name="Read"
        [[ $((i % 2)) -eq 0 ]] && tool_name="Bash"

        echo '{"type":"tool_use","id":"tool_'$i'","name":"'$tool_name'","input":{"file_path":"test.txt"},"timestamp":"'$(timestamp)'"}'
        sleep 0.3

        echo '{"type":"tool_result","tool_use_id":"tool_'$i'","content":"Mock result","timestamp":"'$(timestamp)'"}'
        sleep 0.2
      done

      echo '{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" Task completed successfully."},"timestamp":"'$(timestamp)'"}'
      sleep 0.2

      echo '{"type":"content_block_stop","index":0,"timestamp":"'$(timestamp)'"}'
      sleep 0.1

      echo '{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"input_tokens":100,"output_tokens":50},"timestamp":"'$(timestamp)'"}'
      sleep 0.1

      echo '{"type":"message_stop","timestamp":"'$(timestamp)'"}'
    else
      echo "Task completed successfully (text mode)"
    fi
    exit 0
    ;;

  timeout)
    # Simulate agent that runs too long
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1","type":"message"},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Starting long task..."},"timestamp":"'$(timestamp)'"}'

      # Sleep for configured duration
      sleep "$MOCK_DURATION"

      echo '{"type":"message_stop","timestamp":"'$(timestamp)'"}'
    else
      sleep "$MOCK_DURATION"
    fi
    exit 0
    ;;

  error)
    # Simulate agent that encounters an error
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1","type":"message"},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"error","error":{"type":"api_error","message":"'$MOCK_ERROR_MESSAGE'"},"timestamp":"'$(timestamp)'"}'
      sleep 0.2

      echo '{"type":"message_stop","timestamp":"'$(timestamp)'"}'
    else
      echo "Error: $MOCK_ERROR_MESSAGE" >&2
    fi
    exit 1
    ;;

  crash)
    # Simulate immediate crash
    exit 137  # SIGKILL exit code
    ;;

  stall)
    # Simulate agent that starts but then stalls (no heartbeat)
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1","type":"message"},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Working..."},"timestamp":"'$(timestamp)'"}'

      # Stall indefinitely (or until killed)
      sleep 3600
    else
      sleep 3600
    fi
    exit 0
    ;;

  malformed)
    # Simulate malformed stream-JSON output
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1"}}'
      sleep 0.5

      echo 'This is not JSON'
      sleep 0.3

      echo '{"type":"content_block_delta","index":0,"delta":{'  # Incomplete JSON
      sleep 0.3

      echo '{"type":"message_stop"}'
    else
      echo "Normal output"
    fi
    exit 0
    ;;

  max_tokens)
    # Simulate hitting max tokens
    if [[ "$output_format" == "stream-json" ]]; then
      echo '{"type":"message_start","message":{"id":"msg_1","type":"message"},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Working on task..."},"timestamp":"'$(timestamp)'"}'
      sleep 0.5

      echo '{"type":"message_delta","delta":{"stop_reason":"max_tokens"},"usage":{"input_tokens":1000,"output_tokens":4096},"timestamp":"'$(timestamp)'"}'
      sleep 0.2

      echo '{"type":"message_stop","timestamp":"'$(timestamp)'"}'
    fi
    exit 0
    ;;

  *)
    echo "Unknown mock behavior: $MOCK_BEHAVIOR" >&2
    exit 1
    ;;
esac
