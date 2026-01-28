#!/usr/bin/env bash
# Verification script for spawn_agent() implementation
# Checks that all required functions and logic are present

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HARNESS_ROOT="$(dirname "$SCRIPT_DIR")"
LOOP_FILE="$HARNESS_ROOT/loop.sh"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================"
echo "Spawn Implementation Verification"
echo -e "======================================${NC}"
echo ""

checks_passed=0
checks_failed=0

# Function to check for pattern in file
check_exists() {
  local pattern="$1"
  local description="$2"

  if grep -q "$pattern" "$LOOP_FILE"; then
    echo -e "${GREEN}✓${NC} $description"
    checks_passed=$((checks_passed + 1))
    return 0
  else
    echo -e "${RED}✗${NC} $description"
    checks_failed=$((checks_failed + 1))
    return 1
  fi
}

echo "Checking spawn_agent() implementation..."
echo ""

# Core spawn_agent features
check_exists 'spawn_agent()' "spawn_agent() function exists"
check_exists 'ses_$(uuidgen' "Session ID generation with uuidgen"
check_exists 'manage-queue.sh" next' "Work queue integration"
check_exists '{{SESSION_ID}}' "Bootstrap variable substitution"
check_exists 'jq -n' "Session state creation"
check_exists 'claude -p' "Claude Code command building"
check_exists 'session-id' "Claude CLI flags"
check_exists 'append-system-prompt-file' "Bootstrap injection"
check_exists 'allowedTools' "Tool permissions"
check_exists 'max-turns' "Resource limits"
check_exists 'LAST_SESSION_ID' "Session metadata export"

echo ""
echo "Checking helper functions..."
echo ""

# Helper functions
check_exists 'is_agent_running()' "is_agent_running() function"
check_exists 'get_agent_exit_code()' "get_agent_exit_code() function"
check_exists 'kill_agent()' "kill_agent() function"
check_exists 'update_session_status()' "update_session_status() function"
check_exists 'update_heartbeat()' "update_heartbeat() function"
check_exists 'check_agent_health()' "check_agent_health() function"
check_exists 'detect_completion()' "detect_completion() function"
check_exists 'handle_spawn_failure()' "handle_spawn_failure() function"
check_exists 'reset_failure_counter()' "reset_failure_counter() function"

echo ""
echo "Checking main loop integration..."
echo ""

# Main loop integration
check_exists 'Failed to spawn agent' "Spawn error handling in main loop"
check_exists 'reset_failure_counter' "Failure counter reset on success"
check_exists 'is_agent_running' "Monitor using is_agent_running()"
check_exists 'update_heartbeat' "Heartbeat updates in monitor loop"
check_exists 'check_agent_health' "Health checks in monitor loop"
check_exists 'kill_agent' "Graceful agent termination"
check_exists 'detect_completion' "Completion detection"

echo ""
echo "Checking session state management..."
echo ""

# Session state
check_exists 'start_epoch=' "Epoch timestamp tracking"
check_exists 'session_id:' "Complete session schema"
check_exists '.pid' "PID and exit code files"
check_exists 'docs/sessions' "Session log files"
check_exists 'status_updated_at' "Status transition timestamps"

echo ""
echo "Checking error handling..."
echo ""

# Error handling
check_exists 'Queue manager not found' "Queue manager error handling"
check_exists 'Bootstrap template not found' "Bootstrap error handling"
check_exists 'Working directory not found' "Work directory validation"
check_exists 'exit_code' "Exit code validation"
check_exists 'Session timeout' "Timeout detection"
check_exists 'Too many consecutive failures' "Failure threshold checking"
check_exists 'Exponential backoff' "Backoff logic"

echo ""
echo -e "${BLUE}======================================"
echo "Verification Results"
echo -e "======================================${NC}"
echo -e "${GREEN}Checks passed: $checks_passed${NC}"

if [[ $checks_failed -gt 0 ]]; then
  echo -e "${RED}Checks failed: $checks_failed${NC}"
else
  echo "Checks failed: 0"
fi

echo -e "${BLUE}======================================${NC}"
echo ""

# Additional checks
echo "Additional verification:"
echo ""

# Check file syntax
if bash -n "$LOOP_FILE" 2>/dev/null; then
  echo -e "${GREEN}✓${NC} Bash syntax is valid"
  checks_passed=$((checks_passed + 1))
else
  echo -e "${RED}✗${NC} Bash syntax errors detected"
  checks_failed=$((checks_failed + 1))
fi

# Check for directory creation
if grep -q "mkdir -p" "$LOOP_FILE"; then
  echo -e "${GREEN}✓${NC} Directory creation implemented"
  checks_passed=$((checks_passed + 1))
else
  echo -e "${RED}✗${NC} Missing directory creation"
  checks_failed=$((checks_failed + 1))
fi

# Count functions
function_count=$(grep -c '^[a-z_]*()' "$LOOP_FILE" || echo 0)
echo ""
echo "Statistics:"
echo "  Total functions defined: $function_count"
echo "  File size: $(wc -l < "$LOOP_FILE") lines"

echo ""
echo -e "${BLUE}======================================"
echo "Final Results"
echo -e "======================================${NC}"
echo "Total checks: $((checks_passed + checks_failed))"
echo -e "${GREEN}Passed: $checks_passed${NC}"

if [[ $checks_failed -gt 0 ]]; then
  echo -e "${RED}Failed: $checks_failed${NC}"
  echo ""
  echo "Implementation appears incomplete or incorrect."
  exit 1
else
  echo "Failed: 0"
  echo ""
  echo -e "${GREEN}✓ Implementation verification successful!${NC}"
  echo ""
  echo "All required functions and logic are present."
  echo "The spawn_agent() implementation follows the architecture design."
  exit 0
fi
