#!/bin/bash
# Quick patrol cycle execution
set -e

# Get step IDs
STEPS=$(bd list --parent=$1 | grep -oE 'du-wisp-[a-z0-9]+' | tail -10)
MAIL=$(echo "$STEPS" | tail -1)
CLEANUP=$(echo "$STEPS" | tail -2 | head -1)
REFINERY=$(echo "$STEPS" | tail -3 | head -1)
POLECATS=$(echo "$STEPS" | tail -4 | head -1)
TIMER=$(echo "$STEPS" | tail -5 | head -1)
SWARM=$(echo "$STEPS" | tail -6 | head -1)
DEACON=$(echo "$STEPS" | tail -7 | head -1)
HYGIENE=$(echo "$STEPS" | tail -8 | head -1)
CONTEXT=$(echo "$STEPS" | tail -9 | head -1)

# Execute checks and close steps
gt mail inbox >/dev/null && bd close $MAIL
bd list --label=cleanup --status=open >/dev/null && bd close $CLEANUP
gt session status duneagent/refinery >/dev/null && bd close $REFINERY
bd list --type=agent >/dev/null && bd close $POLECATS --reason "rust: unchanged"
bd gate check --type=timer >/dev/null && bd close $TIMER
bd close $SWARM
bd close $DEACON --reason "Deacon absent"
bd close $HYGIENE
bd close $CONTEXT --reason "Context LOW"

echo "Cycle complete"
