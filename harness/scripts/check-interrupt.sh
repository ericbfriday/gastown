#!/usr/bin/env bash
# Check for interrupt conditions

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
INTERRUPT_FILE="$STATE_DIR/interrupt-request.txt"

# Check for explicit interrupt request
if [[ -f "$INTERRUPT_FILE" ]]; then
  reason=$(cat "$INTERRUPT_FILE")
  echo "INTERRUPT: $reason" >&2
  exit 0
fi

# Check for quality gate failures
if [[ -f "$STATE_DIR/quality-gate-failed" ]]; then
  echo "INTERRUPT: Quality gate failed" >&2
  exit 0
fi

# Check for blocked work via gt hook
if command -v gt &>/dev/null; then
  if gt hook 2>/dev/null | grep -qi "blocked"; then
    echo "INTERRUPT: Work is blocked" >&2
    exit 0
  fi
fi

# Check for approval gate marker
if [[ -f "$STATE_DIR/approval-required" ]]; then
  echo "INTERRUPT: Manual approval required" >&2
  exit 0
fi

# Check if session has been running too long
if [[ -f "$STATE_DIR/current-session.json" ]]; then
  session_timeout=${SESSION_TIMEOUT:-3600}
  started=$(jq -r '.started_at' "$STATE_DIR/current-session.json")
  started_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started" +%s 2>/dev/null || echo 0)
  now_epoch=$(date +%s)
  duration=$((now_epoch - started_epoch))

  if [[ $duration -gt $session_timeout ]]; then
    echo "INTERRUPT: Session timeout (${duration}s > ${session_timeout}s)" >&2
    exit 0
  fi
fi

# Check for error markers
if [[ -f "$STATE_DIR/error" ]]; then
  echo "INTERRUPT: Error occurred (see $STATE_DIR/error)" >&2
  exit 0
fi

# No interrupt detected
exit 1
