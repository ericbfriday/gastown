#!/usr/bin/env bash
# Mock queue manager for testing

set -euo pipefail

# Mock queue state (can be configured via environment)
MOCK_QUEUE_SIZE=${MOCK_QUEUE_SIZE:-1}
MOCK_WORK_ITEM=${MOCK_WORK_ITEM:-'{"id":"test-issue-123","title":"Test work item","priority":10,"description":"Mock work for testing"}'}

case "$1" in
  check)
    # Return queue size
    echo "$MOCK_QUEUE_SIZE"
    exit 0
    ;;

  next)
    # Return next work item
    if [[ $MOCK_QUEUE_SIZE -gt 0 ]]; then
      echo "$MOCK_WORK_ITEM"
      exit 0
    else
      echo "{}"
      exit 1
    fi
    ;;

  claim)
    # Claim work item (mark as in progress)
    local work_id="${2:-unknown}"
    # In mock, just succeed
    exit 0
    ;;

  complete)
    # Mark work item as completed
    local work_id="${2:-unknown}"
    exit 0
    ;;

  fail)
    # Mark work item as failed
    local work_id="${2:-unknown}"
    exit 0
    ;;

  refresh)
    # Refresh queue from beads
    exit 0
    ;;

  *)
    echo "Unknown command: $1" >&2
    exit 1
    ;;
esac
