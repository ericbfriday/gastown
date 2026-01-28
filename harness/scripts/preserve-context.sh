#!/usr/bin/env bash
# Preserve context during interrupts or handoffs

set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="$HARNESS_ROOT/state"
DOCS_DIR="$HARNESS_ROOT/docs"
SESSION_ID="${SESSION_ID:-unknown}"

preserve_context() {
  local context_file="$DOCS_DIR/sessions/${SESSION_ID}-context.json"
  local timestamp
  timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)

  echo "Preserving context for session: $SESSION_ID" >&2

  # Capture current state
  local hook_json="{}"
  local status_json="{}"

  # Try to get gt hook info
  if command -v gt &>/dev/null; then
    hook_json=$(gt hook --format json 2>/dev/null || echo '{}')
  fi

  # Try to get git status
  if command -v git &>/dev/null && [[ -d ~/gt/.git ]]; then
    status_json=$(git -C ~/gt status --porcelain=v2 --branch 2>/dev/null |
      jq -R -s 'split("\n") | map(select(length > 0))' || echo '{}')
  fi

  # Build context document
  jq -n \
    --arg session "$SESSION_ID" \
    --arg timestamp "$timestamp" \
    --argjson hook "$hook_json" \
    --argjson status "$status_json" \
    --arg rig "${BD_RIG:-unknown}" \
    --arg actor "${BD_ACTOR:-unknown}" \
    --arg interrupt_reason "$(cat "$STATE_DIR/interrupt-request.txt" 2>/dev/null || echo "unknown")" \
    '{
      session: $session,
      timestamp: $timestamp,
      hook: $hook,
      git_status: $status,
      rig: $rig,
      actor: $actor,
      interrupt_reason: $interrupt_reason,
      working_directory: env.PWD
    }' > "$context_file"

  echo "Context saved to: $context_file" >&2

  # Capture Serena memories list if available
  if command -v gt &>/dev/null; then
    gt serena list-memories 2>/dev/null > "$DOCS_DIR/sessions/${SESSION_ID}-memories.txt" || true
  fi

  # Capture recent iteration log
  if [[ -f "$STATE_DIR/iteration.log" ]]; then
    tail -n 100 "$STATE_DIR/iteration.log" > "$DOCS_DIR/sessions/${SESSION_ID}-logs.txt"
  fi

  # Capture current beads state
  if command -v bd &>/dev/null; then
    bd list --status in_progress --format json 2>/dev/null > "$DOCS_DIR/sessions/${SESSION_ID}-beads.json" || true
  fi

  # Create a summary file
  cat > "$DOCS_DIR/sessions/${SESSION_ID}-summary.md" <<EOF
# Session Context Summary

**Session ID:** $SESSION_ID
**Timestamp:** $timestamp
**Rig:** ${BD_RIG:-unknown}
**Actor:** ${BD_ACTOR:-unknown}

## Interrupt Reason

$(cat "$STATE_DIR/interrupt-request.txt" 2>/dev/null || echo "Not specified")

## Hook State

$(gt hook 2>/dev/null || echo "Unable to retrieve hook state")

## Git Status

$(git -C ~/gt status 2>/dev/null || echo "Unable to retrieve git status")

## In-Progress Work

$(bd list --status in_progress 2>/dev/null || echo "Unable to retrieve beads status")

## Session Files

- Context: ${SESSION_ID}-context.json
- Memories: ${SESSION_ID}-memories.txt
- Logs: ${SESSION_ID}-logs.txt
- Beads: ${SESSION_ID}-beads.json

EOF

  echo "Summary created: ${SESSION_ID}-summary.md" >&2
  echo "Context preservation complete" >&2
}

# Execute preservation
preserve_context
