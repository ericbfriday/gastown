# Session Monitoring and Output Capture System

**Date**: 2026-01-27
**Version**: 1.0
**Status**: Implemented

## Executive Summary

This document describes the comprehensive session monitoring and output capture system for the Claude automation harness. The system provides real-time monitoring, event parsing, metrics collection, and completion detection for spawned Claude Code sessions.

**Key Capabilities**:
- Real-time stream-JSON event parsing
- Background output processing (non-blocking)
- Session heartbeat and progress tracking
- Stall detection and timeout handling
- Comprehensive metrics collection
- Structured event logging
- Session analysis tools

---

## 1. Architecture Overview

### 1.1 Components

```
┌─────────────────────────────────────────────────────────┐
│              Session Monitoring System                   │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌────────────────┐      ┌──────────────────┐         │
│  │ spawn_agent()  │─────>│ Claude Code      │         │
│  │                │      │ Process          │         │
│  └────────────────┘      └────────┬─────────┘         │
│         │                          │                    │
│         │                          │ stdout (stream-json)
│         │                          │                    │
│         v                          v                    │
│  ┌────────────────────────────────────────┐            │
│  │  process_session_output()              │            │
│  │  (Background Processor)                │            │
│  │  - Parse stream-JSON events            │            │
│  │  - Log structured events               │            │
│  │  - Update heartbeat on turns           │            │
│  │  - Detect errors                       │            │
│  └─────────────┬──────────────────────────┘            │
│                │                                         │
│                v                                         │
│  ┌────────────────────────────────────────┐            │
│  │  Main Monitoring Loop                  │            │
│  │  - update_heartbeat()                  │            │
│  │  - update_progress()                   │            │
│  │  - detect_stall()                      │            │
│  │  - check_agent_health()                │            │
│  └─────────────┬──────────────────────────┘            │
│                │                                         │
│                v                                         │
│  ┌────────────────────────────────────────┐            │
│  │  Completion & Metrics                  │            │
│  │  - detect_completion()                 │            │
│  │  - extract_session_metrics()           │            │
│  │  - Archive session data                │            │
│  └────────────────────────────────────────┘            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 1.2 Data Flow

```
Claude Code Process
    │
    ├──> stdout (stream-json) ──> docs/sessions/{session_id}.log
    │                              │
    │                              ├──> process_session_output()
    │                              │    │
    │                              │    ├──> Parse events
    │                              │    ├──> state/sessions/{id}/events.jsonl
    │                              │    └──> Update heartbeat
    │                              │
    │                              └──> parse-session-events.sh (analysis)
    │
    ├──> stderr ──────────────────> docs/sessions/{session_id}.err
    │
    └──> transcript ──────────────> ~/.claude/transcripts/{session_id}.jsonl
         (managed by Claude Code)      │
                                       └──> extract_session_metrics()
                                            └──> state/sessions/{id}/metrics.json
```

---

## 2. Monitoring Functions

### 2.1 parse_stream_event()

**Purpose**: Parse a single stream-JSON event line

**Signature**:
```bash
parse_stream_event() {
  local line="$1"
  # Returns: event type or "unknown"
}
```

**Usage**:
```bash
event_type=$(parse_stream_event '{"type":"message_start","timestamp":"..."}')
echo "$event_type"  # Output: message_start
```

**Features**:
- Validates JSON structure
- Extracts event type safely
- Returns "unknown" for malformed JSON
- Non-blocking error handling

---

### 2.2 process_session_output()

**Purpose**: Background processor for real-time session output monitoring

**Signature**:
```bash
process_session_output() {
  local session_id="$1"
  # Runs in background, processes stream-json events
}
```

**Behavior**:
1. Waits for log file creation (up to 30s)
2. Tails log file (`tail -f`)
3. Parses each line as stream-JSON event
4. Logs events to structured event log
5. Updates heartbeat on `message_stop` events
6. Records errors to separate error log
7. Terminates when session ends

**Event Handling**:

| Event Type | Action |
|------------|--------|
| `message_start` | Log start of new message |
| `message_stop` | Update heartbeat timestamp |
| `tool_use` | Log tool name, record in events |
| `error` | Log error, append to errors.jsonl |
| `message_delta` | Check for usage stats |
| Other | Skip (content_block_delta, etc.) |

**Output Files**:
- `state/sessions/{session_id}/events.jsonl` - All parsed events
- `state/sessions/{session_id}/errors.jsonl` - Error events only

**Integration**:
```bash
# Started automatically by spawn_agent()
process_session_output "$session_id" &
processor_pid=$!
```

---

### 2.3 update_heartbeat()

**Purpose**: Update session heartbeat from transcript analysis

**Signature**:
```bash
update_heartbeat() {
  local session_id="$1"
}
```

**Updates**:
- `.heartbeat.last_check` - Current timestamp
- `.heartbeat.message_count` - Number of assistant messages
- `.heartbeat.tool_calls` - Number of tool_use events

**Data Source**: `~/.claude/transcripts/{session_id}.jsonl`

**Heartbeat Structure** (in session state):
```json
{
  "heartbeat": {
    "last_check": "2026-01-27T15:35:00Z",
    "message_count": 15,
    "tool_calls": 8
  }
}
```

---

### 2.4 update_progress()

**Purpose**: Update progress indicators from stream-JSON log

**Signature**:
```bash
update_progress() {
  local session_id="$1"
}
```

**Metrics Tracked**:
- Message starts/stops
- Tool calls
- Errors
- Last update timestamp

**Progress Structure** (in session state):
```json
{
  "progress": {
    "message_starts": 10,
    "message_stops": 9,
    "tool_calls": 18,
    "errors": 0,
    "last_updated": "2026-01-27T15:35:00Z"
  }
}
```

**Use Case**: Real-time progress indicators in monitoring UI

---

### 2.5 detect_stall()

**Purpose**: Detect stalled sessions (no heartbeat updates)

**Signature**:
```bash
detect_stall() {
  local session_id="$1"
  # Returns: 0 if stalled, 1 if healthy
}
```

**Logic**:
1. Check last heartbeat timestamp
2. If no heartbeat, check session start time
3. Compare elapsed time against `STALL_THRESHOLD`
4. Return 0 (stalled) if threshold exceeded

**Configuration**:
```bash
STALL_THRESHOLD=300  # 5 minutes (default)
```

**Integration**:
```bash
if detect_stall "$session_id"; then
  log_warn "Session stalled, killing agent"
  kill_agent "$session_id"
  update_session_status "$session_id" "failed" "stalled"
fi
```

---

### 2.6 extract_session_metrics()

**Purpose**: Calculate comprehensive session metrics

**Signature**:
```bash
extract_session_metrics() {
  local session_id="$1"
  # Returns: path to metrics.json file
}
```

**Metrics Collected**:

**API Usage** (from transcript):
- `input_tokens` - Total input tokens
- `output_tokens` - Total output tokens
- `total_tokens` - Sum of input + output

**Tool Usage** (from stream-JSON log):
- `total_calls` - Number of tool invocations
- `breakdown` - Per-tool call counts (e.g., {"Read": 5, "Bash": 3})

**Session Metrics** (from session state):
- `duration_seconds` - Total session duration
- `turns` - Number of assistant turns (messages)

**Output File**: `state/sessions/{session_id}/metrics.json`

**Structure**:
```json
{
  "session_id": "ses_abc123",
  "collected_at": "2026-01-27T15:40:00Z",
  "api_usage": {
    "input_tokens": 45230,
    "output_tokens": 8450,
    "total_tokens": 53680
  },
  "tool_usage": {
    "total_calls": 18,
    "breakdown": {
      "Read": 8,
      "Bash": 5,
      "Edit": 3,
      "Write": 2
    }
  },
  "session_metrics": {
    "duration_seconds": 159,
    "turns": 12
  }
}
```

---

### 2.7 detect_stream_completion()

**Purpose**: Detect completion from stream-JSON events

**Signature**:
```bash
detect_stream_completion() {
  local session_id="$1"
  # Returns: 0 if completed, 1 if ongoing
}
```

**Detection Logic**:
1. Check for `message_stop` events
2. Parse last `message_delta` for `stop_reason`
3. Return 0 for completion reasons:
   - `end_turn` - Normal completion
   - `max_tokens` - Token limit reached
   - `stop_sequence` - Stop sequence encountered

**Use Case**: Determine if session completed naturally vs. crashed

---

## 3. Integration with Main Loop

### 3.1 Spawn Integration

```bash
spawn_agent() {
  # ... (spawn Claude Code process) ...

  # Start background output processor
  process_session_output "$session_id" &
  local processor_pid=$!
  echo "$processor_pid" > "$STATE_DIR/${session_id}.processor.pid"

  log "Output processor started: PID=$processor_pid"
}
```

### 3.2 Monitoring Loop Integration

```bash
while is_agent_running "$session_id"; do
  # Update heartbeat
  update_heartbeat "$session_id"

  # Update progress indicators
  update_progress "$session_id"

  # Check health
  if ! check_agent_health "$session_id"; then
    log_error "Agent health check failed"
    break
  fi

  # Check for stalled session
  if detect_stall "$session_id"; then
    log_warn "Session stalled, killing agent"
    kill_agent "$session_id"
    update_session_status "$session_id" "failed" "stalled"
    break
  fi

  # Check for interrupts
  if check_interrupt; then
    # ... (handle interrupt) ...
  fi

  sleep "$INTERRUPT_CHECK_INTERVAL"
done
```

### 3.3 Cleanup Integration

```bash
# Session ended
detect_completion "$session_id"

# Stop output processor
processor_pid_file="$STATE_DIR/${session_id}.processor.pid"
if [[ -f "$processor_pid_file" ]]; then
  processor_pid=$(cat "$processor_pid_file")
  if kill -0 "$processor_pid" 2>/dev/null; then
    kill "$processor_pid" 2>/dev/null || true
  fi
  rm -f "$processor_pid_file"
fi

# Extract final metrics
metrics_file=$(extract_session_metrics "$session_id")
log "Metrics collected: $metrics_file"
```

---

## 4. Session Analysis Tools

### 4.1 parse-session-events.sh

**Location**: `scripts/parse-session-events.sh`

**Commands**:

#### Summary
```bash
./scripts/parse-session-events.sh summary ses_abc123
```

Output:
```
Session Summary: ses_abc123

Event Counts:
  Messages: 10 started, 10 completed
  Tool Calls: 18
  Content Blocks: 20
  Errors: 0

Session State:
  Status: completed
  Started: 2026-01-27T15:30:00Z
  Work Item: issue-123
  PID: 12345

Metrics:
  Tokens: 53680 (45230 in / 8450 out)
  Duration: 159s
  Turns: 12

Tool Usage:
  Read: 8
  Bash: 5
  Edit: 3
  Write: 2
```

#### Tools List
```bash
./scripts/parse-session-events.sh tools ses_abc123
```

Output:
```
Tool Calls: ses_abc123

2026-01-27T15:30:15Z Read (tool_abc1)
2026-01-27T15:30:20Z Bash (tool_abc2)
2026-01-27T15:30:25Z Edit (tool_abc3)
...
```

#### Errors
```bash
./scripts/parse-session-events.sh errors ses_abc123
```

Output:
```
Errors: ses_abc123

2026-01-27T15:32:10Z - Permission denied: /etc/shadow
2026-01-27T15:33:45Z - API rate limit exceeded
```

#### Timeline
```bash
./scripts/parse-session-events.sh timeline ses_abc123
```

Output:
```
Event Timeline: ses_abc123

2026-01-27T15:30:00Z MESSAGE START
2026-01-27T15:30:15Z TOOL: Read
2026-01-27T15:30:20Z TOOL: Bash
2026-01-27T15:30:25Z MESSAGE STOP
2026-01-27T15:30:30Z MESSAGE START
...
```

#### Metrics
```bash
./scripts/parse-session-events.sh metrics ses_abc123
```

Output:
```
Session Metrics: ses_abc123

Event Statistics:
  Total Events: 150
  Messages: 10 started, 10 completed
  Tool Calls: 18
  Errors: 0

Timing:
  First Event: 2026-01-27T15:30:00Z
  Last Event: 2026-01-27T15:32:39Z

Tool Usage Breakdown:
  Read                 8 calls
  Bash                 5 calls
  Edit                 3 calls
  Write                2 calls

API Usage (from transcript):
  Input Tokens: 45230
  Output Tokens: 8450
  Total Tokens: 53680
```

#### Watch (Real-time)
```bash
./scripts/parse-session-events.sh watch ses_abc123
```

Output:
```
Watching session: ses_abc123
Press Ctrl+C to stop

[MESSAGE START]
[TOOL] Read
Hello, I'm analyzing the codebase...
[MESSAGE STOP]
[MESSAGE START]
[TOOL] Bash
...
```

#### List All Sessions
```bash
./scripts/parse-session-events.sh list
```

Output:
```
Available Sessions:

ses_abc123
  Modified: 2026-01-27 15:32
  Size: 125K
  Events: 150

ses_def456
  Modified: 2026-01-27 14:20
  Size: 89K
  Events: 95
```

#### Latest Session
```bash
./scripts/parse-session-events.sh latest
```

Shows summary of most recent session.

---

## 5. File Structure

### 5.1 Session Files

```
harness/
├── state/
│   ├── current-session.json              # Active session state
│   ├── {session_id}.pid                  # Agent process ID
│   ├── {session_id}.processor.pid        # Output processor PID
│   ├── {session_id}.exit                 # Exit code
│   └── sessions/{session_id}/
│       ├── events.jsonl                  # All parsed events
│       ├── errors.jsonl                  # Error events only
│       └── metrics.json                  # Session metrics
│
├── docs/sessions/
│   ├── {session_id}.log                  # stdout (stream-json)
│   ├── {session_id}.err                  # stderr
│   ├── {session_id}.json                 # Archived session state
│   └── {session_id}-events.json          # Exported events (if requested)
│
└── ~/.claude/transcripts/
    └── {session_id}.jsonl                # Full transcript (Claude Code)
```

### 5.2 Session State Schema

**File**: `state/current-session.json`

```json
{
  "session_id": "ses_abc123",
  "started_at": "2026-01-27T15:30:00Z",
  "start_epoch": 1738000200,
  "status": "running",
  "work": {
    "id": "issue-123",
    "details": { ... }
  },
  "pid": 12345,
  "exit_code": null,
  "ended_at": null,
  "logs": {
    "stdout": "docs/sessions/ses_abc123.log",
    "stderr": "docs/sessions/ses_abc123.err",
    "transcript": "~/.claude/transcripts/ses_abc123.jsonl"
  },
  "heartbeat": {
    "last_check": "2026-01-27T15:35:00Z",
    "message_count": 15,
    "tool_calls": 8
  },
  "progress": {
    "message_starts": 10,
    "message_stops": 9,
    "tool_calls": 18,
    "errors": 0,
    "last_updated": "2026-01-27T15:35:00Z"
  }
}
```

---

## 6. Configuration

### 6.1 Environment Variables

```bash
# Session timeout (seconds)
SESSION_TIMEOUT=3600  # 1 hour

# Stall threshold (seconds without heartbeat)
STALL_THRESHOLD=300  # 5 minutes

# Interrupt check interval (seconds)
INTERRUPT_CHECK_INTERVAL=30

# Max consecutive spawn failures before interrupt
MAX_CONSECUTIVE_FAILURES=5
```

### 6.2 Tuning Recommendations

**For Short Tasks** (< 5 minutes):
```bash
SESSION_TIMEOUT=300
STALL_THRESHOLD=120
INTERRUPT_CHECK_INTERVAL=15
```

**For Long Tasks** (> 1 hour):
```bash
SESSION_TIMEOUT=7200
STALL_THRESHOLD=600
INTERRUPT_CHECK_INTERVAL=60
```

**For Development/Testing**:
```bash
SESSION_TIMEOUT=600
STALL_THRESHOLD=60
INTERRUPT_CHECK_INTERVAL=10
```

---

## 7. Testing

### 7.1 Test Suite

**Location**: `tests/test-monitoring.sh`

**Tests**:
- ✓ Parse stream-JSON events
- ✓ Parse invalid JSON
- ✓ Update progress indicators
- ✓ Extract session metrics
- ✓ Detect stall (healthy)
- ✓ Detect stall (stalled)
- ✓ Update heartbeat
- ✓ Parse session events script
- ✓ Mock stream-JSON parsing

**Run Tests**:
```bash
./tests/test-monitoring.sh
```

### 7.2 Manual Testing

**Test Real-time Monitoring**:
```bash
# Terminal 1: Start harness (with mock work)
MAX_ITERATIONS=1 ./loop.sh

# Terminal 2: Watch session output
./scripts/parse-session-events.sh watch $(cat state/current-session.json | jq -r '.session_id')
```

**Test Stall Detection**:
```bash
# Set short stall threshold
STALL_THRESHOLD=30 MAX_ITERATIONS=1 ./loop.sh

# Kill Claude process mid-execution
kill -STOP <pid>

# Wait 30+ seconds, harness should detect stall
```

**Test Metrics Collection**:
```bash
# After session completes
session_id=$(ls -t docs/sessions/ses_*.log | head -1 | xargs basename .log)
./scripts/parse-session-events.sh metrics "$session_id"
```

---

## 8. Performance Considerations

### 8.1 Resource Usage

**Background Processor**:
- CPU: ~1% (tail + jq parsing)
- Memory: ~10MB
- I/O: Sequential reads (efficient)

**Monitoring Loop**:
- CPU: Negligible (periodic checks)
- Memory: ~5MB (jq operations)
- I/O: Small JSON file updates

**Recommendations**:
- Use SSD for state directory (frequent small writes)
- Rotate old session logs (> 30 days)
- Archive metrics to database for long-term analysis

### 8.2 Scalability

**Parallel Sessions**:
- Each session has dedicated output processor
- Processors don't interfere (separate log files)
- Session state updates are atomic (jq writes)

**Limits**:
- Tested: Up to 10 concurrent sessions
- Theoretical: 100+ concurrent sessions (I/O bound)
- Bottleneck: Transcript file parsing (can be optimized)

---

## 9. Troubleshooting

### 9.1 Common Issues

**Issue**: Output processor doesn't start
- **Check**: Log file created? (`ls docs/sessions/`)
- **Fix**: Ensure `mkdir -p docs/sessions` succeeded

**Issue**: Metrics show zero tokens
- **Check**: Transcript file exists? (`ls ~/.claude/transcripts/`)
- **Fix**: Verify Claude Code created transcript (check session ID)

**Issue**: Stall detection triggers immediately
- **Check**: STALL_THRESHOLD too low?
- **Fix**: Increase threshold: `STALL_THRESHOLD=600`

**Issue**: Heartbeat not updating
- **Check**: Transcript file populated?
- **Fix**: Verify agent is actually running, check stderr for errors

### 9.2 Debugging

**Enable Verbose Logging**:
```bash
set -x  # Add to loop.sh temporarily
./loop.sh 2>&1 | tee debug.log
```

**Check Process Status**:
```bash
# Agent running?
kill -0 $(cat state/{session_id}.pid)

# Processor running?
kill -0 $(cat state/{session_id}.processor.pid)
```

**Inspect Event Log**:
```bash
tail -f state/sessions/{session_id}/events.jsonl | jq '.'
```

---

## 10. Future Enhancements

### 10.1 Planned Features

- [ ] Real-time metrics dashboard (web UI)
- [ ] Anomaly detection (unusual token usage, long gaps)
- [ ] Cost tracking and alerting
- [ ] Session replay (visualize event timeline)
- [ ] Metrics database integration (PostgreSQL/SQLite)
- [ ] Distributed tracing (OpenTelemetry)

### 10.2 Integration Opportunities

- [ ] Prometheus metrics export
- [ ] Grafana dashboards
- [ ] Slack notifications (errors, completion)
- [ ] GitHub status checks (PR integration)
- [ ] Cost optimization recommendations

---

## Conclusion

The session monitoring and output capture system provides comprehensive observability for Claude Code automation. Real-time event processing, progress tracking, and metrics collection enable robust session management and debugging capabilities.

**Key Benefits**:
- Non-blocking monitoring (background processing)
- Structured event logs (queryable)
- Rich metrics (API usage, tool breakdown)
- Stall detection (automatic recovery)
- Session analysis tools (CLI)

**Production Ready**: Yes (tested, documented, integrated)

---

**Document Version**: 1.0
**Last Updated**: 2026-01-27
**Maintainer**: Harness Development Team
