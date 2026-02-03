# Mail Orchestrator Daemon

The Mail Orchestrator is a background daemon that orchestrates mail delivery and processing for asynchronous agent communication in Gas Town.

## Overview

The mail orchestrator runs as part of the main Gas Town daemon and provides:

1. **Background mail delivery** - Monitors mail queues and delivers messages automatically
2. **Priority-based processing** - Handles urgent and high-priority messages with priority
3. **Retry mechanism** - Automatically retries failed deliveries with exponential backoff
4. **Dead letter queue** - Captures messages that fail after maximum retries
5. **Interrupt delivery** - Injects urgent messages directly into agent sessions

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Mail Orchestrator                        │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐     ┌──────────────┐     ┌──────────────┐ │
│  │   Inbound   │────▶│   Outbound   │────▶│ Dead Letter  │ │
│  │    Queue    │     │    Queue     │     │    Queue     │ │
│  └─────────────┘     └──────────────┘     └──────────────┘ │
│        │                    │                     │         │
│        │                    │                     │         │
│        ▼                    ▼                     ▼         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         Message Delivery & Routing Engine           │   │
│  │  • Priority sorting                                 │   │
│  │  • Session lookup                                   │   │
│  │  • Interrupt injection                              │   │
│  │  • Retry logic                                      │   │
│  └─────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

## Queue Management

### Inbound Queue
Messages enter the inbound queue when:
- Delivery is set to `interrupt`
- Priority is `urgent` or `high`
- Message requires orchestrated delivery

### Outbound Queue
Messages move to outbound queue when delivery fails and:
- Retry count is below maximum
- Retry delay hasn't elapsed yet

### Dead Letter Queue
Messages move to dead letter queue when:
- Maximum retry count exceeded
- Message marked with `dead-letter` label in beads

## Message Priority

Priority determines processing order:

1. **Urgent** - Immediate interrupt delivery
2. **High** - Priority processing, interrupt if session active
3. **Normal** - Standard queue delivery (no orchestration)
4. **Low** - Best-effort delivery (no orchestration)

## Configuration

Configuration is stored in `mayor/daemon.json`:

```json
{
  "type": "daemon_config",
  "version": 1,
  "patrols": {
    "mail_orchestrator": {
      "enabled": true
    }
  }
}
```

Default configuration:
- Poll interval: 30 seconds
- Max retries: 3
- Retry delay: 5 minutes
- Dead letter threshold: 5 failures
- Priority processing: enabled

## CLI Commands

### Start Mail Orchestrator
```bash
gt mail daemon start
```

Enables the mail orchestrator in daemon configuration. Requires daemon restart to apply.

### Stop Mail Orchestrator
```bash
gt mail daemon stop
```

Disables the mail orchestrator. Requires daemon restart to apply.

### Check Status
```bash
gt mail daemon status
```

Shows whether the mail orchestrator is enabled and running.

### View Logs
```bash
gt mail daemon logs
```

Displays mail orchestrator logs from the daemon log file.

### Queue Status
```bash
gt mail daemon queue
```

Shows current queue sizes:
```
Mail Queue Status:
  Inbound:     2 messages
  Outbound:    0 messages
  Dead Letter: 0 messages
```

## Integration

The mail orchestrator integrates with:

1. **Beads** - Queries beads database for pending messages
2. **Tmux** - Injects interrupt messages into agent sessions
3. **Hooks** - Fires `mail-received` lifecycle hook on delivery
4. **Mail Router** - Uses routing rules for address resolution

## Message Delivery Flow

1. **Scan** - Poll beads for high-priority/interrupt messages
2. **Queue** - Add to inbound queue, sort by priority
3. **Deliver** - Attempt delivery based on message type:
   - Interrupt delivery → inject into tmux session
   - Queue delivery → notify agent via nudge
4. **Retry** - On failure, move to outbound queue with retry tracking
5. **Dead Letter** - After max retries, move to dead letter queue

## Failure Handling

### Delivery Failures
When delivery fails:
1. Increment attempt counter
2. Record error message
3. Move to outbound queue
4. Wait for retry delay
5. Retry up to max attempts
6. Move to dead letter if exhausted

### Dead Letter Recovery
Messages in dead letter queue require manual intervention:
```bash
# View dead letter queue
gt mail daemon queue

# Inspect dead letter messages in beads
bd list --labels dead-letter

# Manually retry or close
bd close <message-id>
```

## Performance

- Queue processing: O(n log n) for priority sorting
- Message delivery: O(1) per message
- Retry processing: O(n) for queue iteration
- Persistence: Atomic writes with JSON serialization

## Best Practices

1. **Use appropriate priorities** - Reserve `urgent` for critical messages
2. **Monitor dead letter queue** - Check regularly for stuck messages
3. **Configure retry limits** - Adjust based on delivery success rate
4. **Enable logging** - Use `gt mail daemon logs` for troubleshooting
5. **Graceful shutdown** - Queues persist across daemon restarts

## Troubleshooting

### Messages not delivering
1. Check daemon status: `gt daemon status`
2. Verify orchestrator enabled: `gt mail daemon status`
3. Check queue status: `gt mail daemon queue`
4. Review logs: `gt mail daemon logs`

### High dead letter count
1. Inspect dead letter messages: `bd list --labels dead-letter`
2. Check for invalid recipients
3. Verify agent sessions are running
4. Increase retry limits if transient failures

### Performance issues
1. Check queue sizes: `gt mail daemon queue`
2. Reduce poll interval in config
3. Enable priority processing
4. Monitor daemon resource usage

## Implementation Details

### Queue Persistence
Queues are persisted to disk at:
- `daemon/mail-queues/inbound.json`
- `daemon/mail-queues/outbound.json`
- `daemon/mail-queues/dead-letter.json`

Files are written on daemon shutdown and loaded on startup.

### Goroutine Architecture
Three goroutines run concurrently:
1. `processInboundQueue()` - Polls beads and queues messages
2. `processOutboundQueue()` - Delivers queued messages
3. `processRetryQueue()` - Processes retry logic

### Thread Safety
Mutex locks protect queue access:
- `inboundMu` - Inbound queue operations
- `outboundMu` - Outbound queue operations
- `deadLetterMu` - Dead letter queue operations

## Future Enhancements

Potential improvements:
- [ ] Configurable poll intervals per priority
- [ ] Message TTL (time to live)
- [ ] Queue size limits with overflow handling
- [ ] Metrics and monitoring integration
- [ ] Priority queue with heap data structure
- [ ] Batch delivery optimization
- [ ] Circuit breaker for failing recipients
