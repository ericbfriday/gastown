# Claude Automation Harness - Commands Reference

## Quick Reference for Harness Operations

### Running the Harness

```bash
# Navigate to harness directory
cd ~/gt/harness

# Start harness (continuous loop)
./loop.sh

# Start with iteration limit (for testing)
MAX_ITERATIONS=10 ./loop.sh

# Run in background
nohup ./loop.sh > loop.out 2>&1 &

# Stop harness
pkill -f "loop.sh"
# Or Ctrl+C if running in foreground
```

### Configuration

```bash
# Edit main configuration
vi ~/gt/harness/config.yaml

# Key settings:
# - max_iterations: 0 (infinite) or N
# - iteration_delay: 5 (seconds between iterations)
# - agent_type: claude-sonnet
# - session_timeout: 3600 (1 hour)
# - parallel_agents: 1 (Phase 3)
```

### Status & Monitoring

```bash
# Check harness status
./scripts/report-status.sh

# Detailed status
./scripts/report-status.sh --detailed

# Watch status (updates every 5 seconds)
watch -n 5 ./scripts/report-status.sh

# Check iteration log
tail -f state/iteration.log

# View recent sessions
ls -lt docs/sessions/
```

### Session Analysis

```bash
# Parse session events CLI
./scripts/parse-session-events.sh

# Watch active session in real-time
./scripts/parse-session-events.sh watch <session-id>

# Get session summary
./scripts/parse-session-events.sh summary <session-id>

# Show all tool calls
./scripts/parse-session-events.sh tools <session-id>

# Show errors only
./scripts/parse-session-events.sh errors <session-id>

# Show timeline
./scripts/parse-session-events.sh timeline <session-id>

# Extract metrics
./scripts/parse-session-events.sh metrics <session-id>

# Export to JSON
./scripts/parse-session-events.sh export <session-id> output.json

# List all sessions
./scripts/parse-session-events.sh list

# Get latest session
./scripts/parse-session-events.sh latest
```

### Work Queue Management

```bash
# Check queue
./scripts/manage-queue.sh check

# Get next work item
./scripts/manage-queue.sh next

# Show queue contents
./scripts/manage-queue.sh show

# Claim work item
./scripts/manage-queue.sh claim <issue-id>

# Add item to queue
./scripts/manage-queue.sh add '{"id":"test-123","type":"feature","rig":"aardwolf_snd"}'

# Clear queue
./scripts/manage-queue.sh clear
```

### Interrupt Management

```bash
# Request interrupt manually
echo "Manual interrupt for code review" > state/interrupt-request.txt

# Check for interrupt
./scripts/check-interrupt.sh
echo $?  # 0 = interrupt detected, 1 = no interrupt

# Resume from interrupt
rm state/interrupt-request.txt
# Harness will automatically resume
```

### Context Preservation

```bash
# Preserve current session context
./scripts/preserve-context.sh

# Output files created:
# - docs/sessions/<session-id>-context.json
# - docs/sessions/<session-id>-summary.md
# - docs/sessions/<session-id>-memories.txt
# - docs/sessions/<session-id>-logs.txt
# - docs/sessions/<session-id>-beads.json
```

### Testing

```bash
cd ~/gt/harness/tests

# Run all tests
./integration-suite.sh

# Run with verbose output
./integration-suite.sh --verbose

# Run specific suite
./integration-suite.sh --spawn
./integration-suite.sh --monitoring
./integration-suite.sh --errors
./integration-suite.sh --interrupts
./integration-suite.sh --lifecycle
./integration-suite.sh --audit
./integration-suite.sh --e2e

# Run in parallel (faster)
./integration-suite.sh --parallel

# Quick mode (skip slow tests)
./integration-suite.sh --quick

# Run specific test file
./test-spawn-integration.sh
```

### Session Files & Logs

```bash
# View current session state
cat state/current-session.json | jq .

# View session log (stdout)
cat docs/sessions/<session-id>.log

# View session errors (stderr)
cat docs/sessions/<session-id>.err

# View session events
cat state/sessions/<session-id>/events.jsonl | jq .

# View session metrics
cat state/sessions/<session-id>/metrics.json | jq .

# View Claude transcript
cat ~/.claude/transcripts/<session-id>.jsonl | jq .

# Tail session log in real-time
tail -f docs/sessions/<session-id>.log
```

### Cleanup & Maintenance

```bash
# Archive old session logs (older than 30 days)
find docs/sessions -name "*.log" -mtime +30 -exec gzip {} \;

# Clean up test artifacts
rm -rf /tmp/harness-test-*

# Clean up old bootstrap files
find /tmp -name "harness-bootstrap-ses_*" -mtime +7 -delete

# Verify harness directory structure
ls -la state/ prompts/ scripts/ docs/
```

### Troubleshooting

```bash
# Check required commands
which gt bd jq git claude

# Check versions
gt --version    # Should be v0.4.0
bd --version    # Should be v0.47.1

# Check for errors in iteration log
tail -100 state/iteration.log | grep ERROR

# View recent session failures
grep "status.*failed" state/iteration.log | tail -10

# Check disk space
df -h ~/gt/harness

# Check memory usage
ps aux | grep -E "(loop.sh|claude)" | awk '{print $2, $3, $4, $11}'

# Verify queue file format
cat state/queue.json | jq .

# Check bootstrap template
cat prompts/bootstrap.md

# Validate configuration
cat config.yaml | grep -v "^#" | grep -v "^$"
```

### Integration with Gastown

```bash
# These commands work from harness directory or anywhere in ~/gt/

# Check your hook (assigned work)
gt hook

# Check mail
gt mail inbox

# Prime context
gt prime && bd prime

# Find work
bd ready
gt ready

# Dispatch work to harness
bd create --title "Task for harness" --type feature
# Then harness will pick it up automatically

# Send mail to overseer
gt mail send overseer -s "Harness Status" -m "Phase 2 complete"
```

### Common Workflows

#### Starting a Fresh Harness Run
```bash
cd ~/gt/harness
rm state/interrupt-request.txt  # Clear any interrupts
./scripts/manage-queue.sh check  # Verify work available
MAX_ITERATIONS=5 ./loop.sh       # Test run with limit
# If successful:
./loop.sh                         # Production run
```

#### Analyzing a Failed Session
```bash
session_id="ses_abc123"  # Replace with actual session ID

# View summary
./scripts/parse-session-events.sh summary $session_id

# Check errors
./scripts/parse-session-events.sh errors $session_id

# View full log
cat docs/sessions/${session_id}.log

# Check stderr
cat docs/sessions/${session_id}.err

# View exit code
cat state/sessions/${session_id}/exit_code
```

#### Monitoring Active Session
```bash
# Terminal 1: Run harness
cd ~/gt/harness
./loop.sh

# Terminal 2: Watch status
watch -n 5 './scripts/report-status.sh'

# Terminal 3: Watch active session
current_session=$(cat state/current-session.json | jq -r '.session_id')
./scripts/parse-session-events.sh watch $current_session
```

#### Emergency Stop and Recover
```bash
# Stop harness
pkill -f "loop.sh"

# Preserve current state
./scripts/preserve-context.sh

# Check what was in progress
cat state/current-session.json | jq .

# Clean up if needed
rm state/current-session.json

# Resume when ready
./loop.sh
```

### Environment Variables

```bash
# Override defaults with environment variables
MAX_ITERATIONS=10 \
ITERATION_DELAY=10 \
INTERRUPT_CHECK_INTERVAL=60 \
AGENT_TYPE=claude-opus \
SESSION_TIMEOUT=7200 \
./loop.sh
```

### File Locations Reference

**Configuration:**
- Main config: `~/gt/harness/config.yaml`
- Bootstrap prompt: `~/gt/harness/prompts/bootstrap.md`

**State:**
- Work queue: `~/gt/harness/state/queue.json`
- Current session: `~/gt/harness/state/current-session.json`
- Iteration log: `~/gt/harness/state/iteration.log`
- Interrupt request: `~/gt/harness/state/interrupt-request.txt`

**Session Data:**
- Logs: `~/gt/harness/docs/sessions/<session-id>.log`
- Errors: `~/gt/harness/docs/sessions/<session-id>.err`
- Events: `~/gt/harness/state/sessions/<session-id>/events.jsonl`
- Metrics: `~/gt/harness/state/sessions/<session-id>/metrics.json`
- Transcripts: `~/.claude/transcripts/<session-id>.jsonl`

**Documentation:**
- Main README: `~/gt/harness/README.md`
- Roadmap: `~/gt/harness/ROADMAP.md`
- Phase 2 Summary: `~/gt/harness/docs/PHASE-2-SUMMARY.md`
- Production Rollout: `~/gt/harness/docs/PRODUCTION-ROLLOUT.md`
- Monitoring System: `~/gt/harness/docs/monitoring-system.md`

**Scripts:**
- Main loop: `~/gt/harness/loop.sh`
- Queue manager: `~/gt/harness/scripts/manage-queue.sh`
- Interrupt checker: `~/gt/harness/scripts/check-interrupt.sh`
- Context preservation: `~/gt/harness/scripts/preserve-context.sh`
- Status reporter: `~/gt/harness/scripts/report-status.sh`
- Session parser: `~/gt/harness/scripts/parse-session-events.sh`

**Tests:**
- Test runner: `~/gt/harness/tests/integration-suite.sh`
- Test library: `~/gt/harness/tests/test-lib.sh`
- Mock Claude: `~/gt/harness/tests/mocks/mock-claude.sh`
