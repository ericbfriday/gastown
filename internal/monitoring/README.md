# Agent Monitoring

Agent monitoring system with status inference from activity patterns.

## Overview

The monitoring package provides real-time agent status tracking with three detection methods:
1. **Boss Override**: Witness/Mayor manually sets status (highest priority)
2. **Self-Reported**: Agent reports own status
3. **Inferred**: Detected from pane output patterns (lowest priority)

## Core Components

### Status Tracker

Central tracking for all agents:

```go
tracker := monitoring.NewStatusTracker()

// Update from output
report := tracker.UpdateFromOutput("duneagent/polecats/rust", "Reading file...")

// Manual status set
tracker.SetStatus("mayor", monitoring.StatusWorking, monitoring.SourceBossOverride, "Coordinating work")

// Get current status
status, source, exists := tracker.GetStatus("duneagent/polecats/rust")
```

### Activity Detector

Pattern-based status detection:

```go
detector := monitoring.NewDetector()

// Detect from single line
report := detector.Detect("agent-id", "Thinking...")

// Detect from multiple lines
lines := []string{"Reading files...", "ERROR: File not found"}
report := detector.DetectMultiline("agent-id", lines)
```

### Idle Detector

Automatic idle detection:

```go
config := monitoring.DefaultIdleConfig()
config.Timeout = 5 * time.Minute
config.CheckInterval = 30 * time.Second

detector := monitoring.NewIdleDetector(tracker, config)

ctx := context.Background()
detector.Start(ctx)
defer detector.Stop()
```

## Agent Statuses

| Status | Description | Priority |
|--------|-------------|----------|
| `available` | Ready for work | Low |
| `working` | Actively working | Medium |
| `thinking` | Processing/analyzing | Medium |
| `blocked` | Waiting for dependency | High |
| `waiting` | Waiting for user input | Medium |
| `reviewing` | Reviewing changes | Medium |
| `idle` | No recent activity | Low |
| `paused` | Manually paused | Medium |
| `error` | Error state | High |
| `offline` | Not running | Highest |

## Pattern Matching

### Default Patterns

Built-in patterns detect common status indicators:

**High Priority (100):**
- `BLOCKED:` → blocked
- `ERROR:`, `Error:` → error

**Medium Priority (70-80):**
- `Thinking...` → thinking
- `Reading`, `Writing`, `Editing` → working
- `Searching`, `Running`, `Executing` → working
- `Using tool:`, `<function_calls>` → working

**Review (65-70):**
- `Reviewing`, `Checking` → reviewing

**Waiting (60):**
- `waiting for`, `Would you like`, `Should I` → waiting

**Completion (50):**
- `completed`, `finished`, `done` → available

### Custom Patterns

Add domain-specific patterns:

```go
detector := monitoring.NewDetector()

detector.AddPattern(monitoring.ActivityPattern{
    Pattern:     "Compiling",
    Status:      monitoring.StatusWorking,
    Priority:    75,
    Description: "Build operation",
})

detector.AddPattern(monitoring.ActivityPattern{
    Pattern:     `(?i)test.*fail`,
    Status:      monitoring.StatusError,
    Priority:    90,
    IsRegex:     true,
    Description: "Test failures",
})
```

## Status Priority

When multiple statuses apply, highest priority source wins:

1. **Boss Override** (priority 3) - Manual override from supervisor
2. **Self-Reported** (priority 2) - Agent reports own status
3. **Inferred** (priority 1) - Detected from output

Within inferred status, pattern priority determines which status wins.

## Integration Example

### Monitoring Pane Output

```go
import (
    "github.com/steveyegge/gastown/internal/monitoring"
    "github.com/steveyegge/gastown/internal/tmux"
)

func monitorAgent(sessionID string) {
    tracker := monitoring.NewStatusTracker()
    tmuxClient := tmux.New()

    // Start idle detection
    idleDetector := monitoring.NewIdleDetector(
        tracker,
        monitoring.DefaultIdleConfig(),
    )
    ctx := context.Background()
    idleDetector.Start(ctx)
    defer idleDetector.Stop()

    // Monitor pane output
    for {
        output, err := tmuxClient.CapturePane(sessionID, "-p", "-S", "-100")
        if err != nil {
            continue
        }

        lines := strings.Split(output, "\n")
        report := tracker.UpdateFromOutput(sessionID, lines[len(lines)-1])

        fmt.Printf("Agent %s: %s (%s)\n",
            sessionID, report.Status, report.Source)

        time.Sleep(5 * time.Second)
    }
}
```

### Status Display

```go
func displayAgentStatus(tracker *monitoring.StatusTracker) {
    statuses := tracker.GetAllStatuses()

    for agentID, report := range statuses {
        statusIcon := "○"
        switch report.Status {
        case monitoring.StatusWorking:
            statusIcon = "●"
        case monitoring.StatusError:
            statusIcon = "✗"
        case monitoring.StatusIdle:
            statusIcon = "○"
        }

        fmt.Printf("%s %s [%s] %s\n",
            statusIcon,
            agentID,
            report.Status,
            report.Source,
        )
    }
}
```

## Configuration

### Idle Detection

```go
config := monitoring.IdleConfig{
    Timeout:       5 * time.Minute,  // Idle after 5min inactivity
    CheckInterval: 30 * time.Second, // Check every 30s
    Enabled:       true,
}

tracker := monitoring.NewStatusTrackerWithConfig(config)
```

### Custom Detection

```go
tracker := monitoring.NewStatusTracker()

// Add custom pattern
tracker.Detector().AddPattern(monitoring.ActivityPattern{
    Pattern:     "Deploying",
    Status:      monitoring.StatusWorking,
    Priority:    80,
    Description: "Deployment in progress",
})
```

## CLI Integration

See implementation in `internal/cmd/status.go`:

```go
// Add status to agent display
type AgentRuntime struct {
    // ... existing fields ...
    Status       monitoring.AgentStatus  `json:"status,omitempty"`
    StatusSource monitoring.StatusSource `json:"status_source,omitempty"`
}

// Update during status collection
if tracker != nil {
    status, source, _ := tracker.GetStatus(agentID)
    agent.Status = status
    StatusSource = source
}

// Display in compact view
statusIcon := getStatusIcon(agent.Status)
fmt.Printf("%s %s %s\n", statusIcon, agent.Name, agent.Status)
```

## Performance Considerations

- Pattern matching is O(n) where n = number of patterns
- Regex patterns are slower than string matching
- Tracker uses sync.RWMutex for concurrent access
- History limited to last 50 status changes per agent
- Idle detection runs in background goroutine

## Testing

Example test structure:

```go
func TestStatusInference(t *testing.T) {
    detector := monitoring.NewDetector()

    tests := []struct {
        output string
        want   monitoring.AgentStatus
    }{
        {"Thinking...", monitoring.StatusThinking},
        {"ERROR: failed", monitoring.StatusError},
        {"Reading file.txt", monitoring.StatusWorking},
    }

    for _, tt := range tests {
        report := detector.Detect("agent", tt.output)
        assert.Equal(t, tt.want, report.Status)
    }
}
```

## Future Enhancements

Potential additions:
- Resource monitoring (CPU/memory via os.Process)
- Status change notifications/events
- Persistent status history
- Agent performance metrics
- Status transitions graph
- Custom status types per role
