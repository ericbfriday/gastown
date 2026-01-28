#!/usr/bin/env bash
# Work Queue Manager for Claude Harness

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
QUEUE_FILE="$HARNESS_ROOT/state/queue.json"

# Ensure queue file exists
ensure_queue() {
  if [[ ! -f "$QUEUE_FILE" ]]; then
    echo "[]" > "$QUEUE_FILE"
  fi
}

# Check and refresh queue from beads and gt
check_queue() {
  ensure_queue

  # Get ready work from beads
  local ready_work
  ready_work=$(bd ready --format json 2>/dev/null || echo "[]")

  # Get work from gt across rigs
  local gt_ready
  gt_ready=$(gt ready --format json 2>/dev/null || echo "[]")

  # Merge and prioritize work
  # Priority: higher number = higher priority
  jq -s '.[0] + .[1] |
    map(select(.status == "pending" or .status == "ready")) |
    sort_by(.priority // 0) |
    reverse' \
    <(echo "$ready_work") \
    <(echo "$gt_ready") \
    > "$QUEUE_FILE"

  # Return count of work items
  jq 'length' "$QUEUE_FILE"
}

# Get next highest priority work item
get_next_work() {
  ensure_queue

  # Get first item (highest priority)
  jq -r '.[0] // empty | @json' "$QUEUE_FILE"
}

# Mark work as claimed (remove from queue)
mark_claimed() {
  local issue_id="$1"
  ensure_queue

  # Find and remove the item with matching ID
  jq --arg id "$issue_id" \
    'map(select(.id != $id))' \
    "$QUEUE_FILE" > "$QUEUE_FILE.tmp"
  mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"

  echo "Claimed: $issue_id"
}

# Add work item to queue
add_work() {
  local work_json="$1"
  ensure_queue

  # Add to queue and re-sort
  jq --argjson work "$work_json" \
    '. + [$work] | sort_by(.priority // 0) | reverse' \
    "$QUEUE_FILE" > "$QUEUE_FILE.tmp"
  mv "$QUEUE_FILE.tmp" "$QUEUE_FILE"

  echo "Added work item"
}

# Show queue contents
show_queue() {
  ensure_queue
  jq '.' "$QUEUE_FILE"
}

# Clear queue
clear_queue() {
  echo "[]" > "$QUEUE_FILE"
  echo "Queue cleared"
}

# Main command dispatch
case "${1:-}" in
  check)
    check_queue
    ;;
  next)
    get_next_work
    ;;
  claim)
    if [[ -z "${2:-}" ]]; then
      echo "Usage: $0 claim <issue-id>" >&2
      exit 1
    fi
    mark_claimed "$2"
    ;;
  add)
    if [[ -z "${2:-}" ]]; then
      echo "Usage: $0 add <work-json>" >&2
      exit 1
    fi
    add_work "$2"
    ;;
  show)
    show_queue
    ;;
  clear)
    clear_queue
    ;;
  *)
    cat >&2 <<EOF
Usage: $0 {check|next|claim|add|show|clear}

Commands:
  check       - Refresh queue from beads/gt and return count
  next        - Get next highest priority work item
  claim <id>  - Mark work item as claimed (remove from queue)
  add <json>  - Add work item to queue
  show        - Display queue contents
  clear       - Clear all items from queue
EOF
    exit 1
    ;;
esac
