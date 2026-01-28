#!/usr/bin/env bash
# Generate status report for harness

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"

# Check if detailed mode
DETAILED=${1:-false}

generate_report() {
  local now
  now=$(date)

  cat <<EOF
╔════════════════════════════════════════════════════════════════╗
║          Claude Automation Harness - Status Report             ║
╚════════════════════════════════════════════════════════════════╝

Generated: $now

┌─ Current State ─────────────────────────────────────────────────┐
EOF

  if [[ -f "$STATE_DIR/current-session.json" ]]; then
    jq -r '
      "│ Session:    " + .session_id + "\n" +
      "│ Started:    " + .started_at + "\n" +
      "│ Status:     " + .status + "\n" +
      "│ Work Item:  " + (.work_item // "none")
    ' "$STATE_DIR/current-session.json"
  else
    echo "│ No active session"
  fi

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Work Queue ────────────────────────────────────────────────────┐
EOF

  if [[ -x "$HARNESS_ROOT/scripts/manage-queue.sh" ]]; then
    local count
    count=$("$HARNESS_ROOT/scripts/manage-queue.sh" check 2>/dev/null || echo "0")
    echo "│ Ready items: $count"

    if [[ "$DETAILED" == "--detailed" && "$count" -gt 0 ]]; then
      echo "│"
      jq -r '.[] | "│   - " + .id + ": " + .title' "$STATE_DIR/queue.json" 2>/dev/null || true
    fi
  else
    echo "│ Queue manager not available"
  fi

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Recent Activity ───────────────────────────────────────────────┐
EOF

  if [[ -f "$STATE_DIR/iteration.log" ]]; then
    tail -n 5 "$STATE_DIR/iteration.log" | sed 's/^/│ /'
  else
    echo "│ No activity log"
  fi

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Gastown Environment ───────────────────────────────────────────┐
EOF

  if command -v gt &>/dev/null; then
    echo "│ Active Rigs:"
    gt rig list 2>/dev/null | sed 's/^/│   /' || echo "│   Unable to list rigs"
  else
    echo "│ gt command not available"
  fi

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Interrupts & Blocks ───────────────────────────────────────────┐
EOF

  if [[ -f "$STATE_DIR/interrupt-request.txt" ]]; then
    echo "│ ⚠️  INTERRUPT ACTIVE:"
    cat "$STATE_DIR/interrupt-request.txt" | sed 's/^/│   /'
  else
    echo "│ No active interrupts"
  fi

  # Recent interrupts (last 24h)
  local interrupt_count
  interrupt_count=$(find "$DOCS_DIR/sessions" -name "*-context.json" -mtime -1 2>/dev/null | wc -l | tr -d ' ')
  echo "│ Interrupts (24h): $interrupt_count"

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Session Statistics ────────────────────────────────────────────┐
EOF

  local total_sessions
  local completed_sessions
  total_sessions=$(find "$DOCS_DIR/sessions" -name "session-*.json" 2>/dev/null | wc -l | tr -d ' ')
  completed_sessions=$(grep -l '"status": "completed"' "$DOCS_DIR/sessions"/session-*.json 2>/dev/null | wc -l | tr -d ' ')

  echo "│ Total sessions:     $total_sessions"
  echo "│ Completed sessions: $completed_sessions"

  cat <<EOF
└─────────────────────────────────────────────────────────────────┘

EOF

  if [[ "$DETAILED" == "--detailed" ]]; then
    cat <<EOF
┌─ Convoys in Progress ───────────────────────────────────────────┐
EOF

    if command -v gt &>/dev/null; then
      gt convoy list --active 2>/dev/null | sed 's/^/│ /' || echo "│ No active convoys"
    else
      echo "│ gt command not available"
    fi

    cat <<EOF
└─────────────────────────────────────────────────────────────────┘

┌─ Recent Sessions ───────────────────────────────────────────────┐
EOF

    find "$DOCS_DIR/sessions" -name "session-*.json" -mtime -1 2>/dev/null |
      sort -r |
      head -n 5 |
      while read -r session_file; do
        jq -r '"│ " + .session_id + " [" + .status + "] " + .started_at' "$session_file" 2>/dev/null || true
      done

    if [[ ! -s /dev/stdin ]]; then
      echo "│ No recent sessions"
    fi

    cat <<EOF
└─────────────────────────────────────────────────────────────────┘
EOF
  fi
}

generate_report
