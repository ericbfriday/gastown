# Mail Orchestrator Implementation (gt-3fm)

This document describes the implementation of the mail orchestrator daemon for asynchronous agent communication in Gas Town.

## Implementation Summary

The mail orchestrator daemon has been implemented to provide background mail delivery and queue management for async agent communication.

### Files Created

1. **internal/daemon/mail_orchestrator.go** - Core mail orchestrator daemon
   - Background mail queue monitoring
   - Priority-based message processing
   - Retry logic with exponential backoff
   - Dead letter queue management
   - Queue persistence across restarts

2. **internal/daemon/mail_orchestrator_test.go** - Test suite
   - Queue management tests
   - Priority sorting tests
   - Orchestration logic tests
   - Persistence tests
   - Dead letter queue tests
   - Retry logic tests

3. **internal/cmd/mail_daemon.go** - CLI commands
   - `gt mail daemon start` - Enable orchestrator
   - `gt mail daemon stop` - Disable orchestrator
   - `gt mail daemon status` - Show status
   - `gt mail daemon logs` - View logs
   - `gt mail daemon queue` - Show queue status

4. **docs/mail-orchestrator.md** - User documentation
   - Architecture overview
   - Queue management guide
   - Configuration reference
   - CLI command reference
   - Troubleshooting guide

5. **docs/mail-orchestrator-implementation.md** - Implementation docs (this file)

### Files Modified

1. **internal/daemon/daemon.go**
   - Added `mailOrchestrator` field to Daemon struct
   - Integrated orchestrator startup in `Run()`
   - Integrated orchestrator shutdown in `shutdown()`
   - Added patrol config check for enable/disable

2. **internal/daemon/types.go**
   - Added `MailOrchestrator` to `PatrolsConfig`
   - Added `"mail-orchestrator"` case to `IsPatrolEnabled()`

3. **internal/mail/bd.go**
   - Exported `RunBdCommand()` for use by daemon package

## Architecture

### Component Diagram

```
┌────────────────────────────────────────────────────────────────┐
│                       Gas Town Daemon                          │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              Mail Orchestrator                            │ │
│  │                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │ │
│  │  │   Inbound    │  │   Outbound   │  │ Dead Letter  │   │ │
│  │  │   Processor  │  │   Processor  │  │   Processor  │   │ │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │ │
│  │         │                  │                  │           │ │
│  │         └──────────┬───────┴──────────────────┘           │ │
│  │                    │                                      │ │
│  │         ┌──────────▼──────────┐                          │ │
│  │         │  Delivery Engine    │                          │ │
│  │         │  • Priority sort    │                          │ │
│  │         │  • Routing rules    │                          │ │
│  │         │  • Interrupt inject │                          │ │
│  │         │  • Retry logic      │                          │ │
│  │         └──────────┬──────────┘                          │ │
│  │                    │                                      │ │
│  └────────────────────┼──────────────────────────────────────┘ │
│                       │                                        │
│         ┌─────────────┼─────────────┐                         │
│         │             │             │                         │
│         ▼             ▼             ▼                         │
│    ┌────────┐   ┌─────────┐   ┌────────┐                    │
│    │ Beads  │   │  Tmux   │   │ Hooks  │                    │
│    │  DB    │   │Sessions │   │ System │                    │
│    └────────┘   └─────────┘   └────────┘                    │
└────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Message Creation**
   ```
   Agent → gt mail send → Router.Send() → Beads DB
   ```

2. **Orchestrator Scan**
   ```
   Orchestrator → Poll Beads → Query priority messages
                             → Add to inbound queue
                             → Sort by priority
   ```

3. **Message Delivery**
   ```
   Inbound Queue → Delivery Engine → Route by type
                                   → Interrupt: Tmux inject
                                   → Queue: Notification
   ```

4. **Retry Flow**
   ```
   Failed Delivery → Outbound Queue → Wait retry delay
                                    → Move to inbound
                                    → Retry delivery
   ```

5. **Dead Letter**
   ```
   Max Retries Exceeded → Dead Letter Queue → Mark in beads
                                           → Log for manual review
   ```

## Key Design Decisions

### 1. Integration with Main Daemon

**Decision**: Integrate mail orchestrator into main daemon rather than separate process.

**Rationale**:
- Shares daemon lifecycle (start/stop/restart)
- Reduces process management complexity
- Enables unified logging
- Consistent with existing daemon architecture (KRC pruner, convoy watcher)

**Implementation**:
- Added as field in `Daemon` struct
- Started/stopped in daemon `Run()`/`shutdown()`
- Controlled via patrol config like other patrols

### 2. Queue-Based Architecture

**Decision**: Use three separate queues (inbound, outbound, dead letter).

**Rationale**:
- Clear separation of concerns
- Independent processing loops
- Easy to reason about message state
- Supports priority-based reordering

**Implementation**:
- Inbound: New messages from beads scan
- Outbound: Failed deliveries awaiting retry
- Dead letter: Permanently failed messages

### 3. Priority Processing

**Decision**: Enable priority-based message processing by default.

**Rationale**:
- Urgent messages need immediate delivery
- High-priority messages should process first
- Aligns with real-world mail expectations
- Configurable via `EnablePriorityProcessing`

**Implementation**:
- Priority values: Urgent(3) > High(2) > Normal(1) > Low(0)
- Sort using `sortByPriority()` with stable sort
- Older messages processed first within same priority

### 4. Persistence

**Decision**: Persist queues to disk on shutdown, load on startup.

**Rationale**:
- Messages survive daemon restarts
- No data loss during planned maintenance
- Simple JSON serialization sufficient
- Atomic write prevents corruption

**Implementation**:
- JSON files in `daemon/mail-queues/`
- Saved on graceful shutdown
- Loaded on startup
- Each queue in separate file

### 5. Delivery Methods

**Decision**: Support two delivery methods (interrupt, queue).

**Rationale**:
- Interrupt for urgent messages needing immediate attention
- Queue for normal messages checked periodically
- Aligns with existing mail system design
- Different use cases have different urgency

**Implementation**:
- Interrupt: `tmux.NudgeSession()` injection
- Queue: Notification via nudge
- Automatic selection based on priority/delivery field

### 6. Retry Strategy

**Decision**: Fixed retry with exponential backoff.

**Rationale**:
- Simple to implement and understand
- Prevents thundering herd
- Configurable limits
- Dead letter after max retries

**Implementation**:
- Max retries: 3 (configurable)
- Retry delay: 5 minutes (configurable)
- Track attempts per message
- Move to dead letter after threshold

### 7. Configuration

**Decision**: Use existing patrol config system (`mayor/daemon.json`).

**Rationale**:
- Consistent with other daemon patrols
- Centralized configuration
- Easy enable/disable
- Version controlled

**Implementation**:
- Added `MailOrchestrator` to `PatrolsConfig`
- Check via `IsPatrolEnabled()`
- Default: enabled
- CLI commands to toggle

## Implementation Details

### Queue Structures

```go
type QueuedMessage struct {
    Message     *mail.Message // The mail message
    Attempts    int           // Delivery attempts
    LastAttempt time.Time     // Last delivery attempt
    QueuedAt    time.Time     // When queued
    Error       string        // Last error message
}
```

### Goroutine Architecture

Three concurrent goroutines:

1. **Inbound Processor** (`processInboundQueue`)
   - Polls beads every 30s
   - Queries priority messages
   - Queues for delivery
   - Sorts by priority

2. **Outbound Processor** (`processOutboundQueue`)
   - Delivers queued messages
   - Handles routing
   - Tracks failures
   - Moves to retry queue

3. **Retry Processor** (`processRetryQueue`)
   - Checks retry delays
   - Moves eligible messages back to inbound
   - Respects max retry count
   - Moves to dead letter if exhausted

### Thread Safety

Mutex locks protect queue access:
```go
inboundMu      sync.Mutex  // Protects inboundQueue
outboundMu     sync.Mutex  // Protects outboundQueue
deadLetterMu   sync.Mutex  // Protects deadLetterQueue
```

### Graceful Shutdown

Context cancellation for clean shutdown:
```go
ctx, cancel := context.WithCancel(context.Background())
wg.Add(3)  // Wait for all goroutines
// ... goroutines select on ctx.Done()
cancel()   // Signal shutdown
wg.Wait()  // Wait for completion
saveQueues() // Persist state
```

## Testing Strategy

### Test Coverage

1. **Queue Management** - Add/remove messages, check sizes
2. **Priority Sorting** - Verify correct sort order
3. **Orchestration Logic** - Which messages need orchestration
4. **Persistence** - Save/load queues across restarts
5. **Dead Letter** - Max retries trigger dead letter
6. **Retry Logic** - Failed messages retry after delay

### Test Approach

- Unit tests for core logic
- Mock beads/tmux for integration tests
- Temporary directories for file I/O
- Time-based tests use shorter intervals
- No external dependencies

## Performance Considerations

### Complexity Analysis

- Queue scan: O(n) where n = message count
- Priority sort: O(n log n) with stable sort
- Delivery: O(1) per message
- Retry check: O(n) queue iteration
- Persistence: O(n) JSON serialization

### Optimization Opportunities

1. **Heap-based priority queue** - O(log n) insertion/removal
2. **Batch delivery** - Group messages by recipient
3. **Index by priority** - Separate queues per priority
4. **Background persistence** - Periodic saves vs shutdown-only
5. **Connection pooling** - Reuse tmux/beads connections

### Resource Usage

- Memory: O(n) for queued messages
- Disk: JSON files ~1KB per message
- CPU: Minimal (poll-based, not event-driven)
- Network: None (local tmux/beads)

## Future Enhancements

### Short Term
- [ ] Add metrics for monitoring
- [ ] Support configurable poll intervals per priority
- [ ] Add message TTL (time to live)
- [ ] Queue size limits with overflow handling

### Medium Term
- [ ] Circuit breaker for failing recipients
- [ ] Batch delivery optimization
- [ ] Event-driven delivery (vs polling)
- [ ] Delivery acknowledgments

### Long Term
- [ ] Distributed queue support
- [ ] Message routing rules engine
- [ ] Dead letter queue replay
- [ ] A/B testing for delivery strategies

## Integration Points

### Beads Database
- Query: `bd list --type=message --status=open`
- Label: `bd label add <id> dead-letter`
- Custom types ensured via `beads.EnsureCustomTypes()`

### Tmux Sessions
- Check: `tmux.HasSession(sessionID)`
- Nudge: `tmux.NudgeSession(sessionID, message)`
- Session IDs from `addressToSessionIDs()`

### Hooks System
- Event: `hooks.EventMailReceived`
- Context: Includes from/to/subject metadata
- Best-effort execution (failures logged)

### Mail Router
- Address resolution via `mail.Router`
- Routing rules for delivery
- Integration with existing mail system

## Deployment

### Prerequisites
- Beads database initialized (`bd init`)
- Main daemon running (`gt daemon start`)
- Mayor/daemon.json configuration

### Installation
1. Code already integrated into daemon
2. Enable in config: `gt mail daemon start`
3. Restart daemon: `gt daemon stop && gt daemon start`

### Verification
```bash
# Check status
gt mail daemon status

# View logs
gt mail daemon logs

# Monitor queues
gt mail daemon queue
```

### Monitoring
- Check daemon logs for "Mail orchestrator" entries
- Monitor queue sizes with `gt mail daemon queue`
- Watch for dead letter messages
- Track delivery success/failure rates

## Troubleshooting

See [docs/mail-orchestrator.md](mail-orchestrator.md#troubleshooting) for detailed troubleshooting guide.

## References

- [Mail System Documentation](mail-system.md)
- [Daemon Architecture](daemon-architecture.md)
- [Beads Integration](beads-integration.md)
- [Hooks System](hooks-system.md)
