# Session Cycling TUI

A smooth Terminal User Interface for cycling polecat sessions in Gas Town.

## Overview

The Session Cycling TUI provides visual feedback and progress tracking during session transitions, replacing abrupt restarts with a polished interface that shows:

- **Real-time progress** with animated spinner and progress bar
- **Transition phases** (pre-shutdown, shutdown, startup, etc.)
- **Context preservation** displaying data carried between sessions
- **Hook execution** status with visual feedback
- **Error handling** with clear messages and recovery guidance

## Quick Start

```bash
# Restart a session with TUI
gt session cycle wyvern/Toast

# Restart to specific issue
gt session cycle wyvern/Toast --issue gt-abc123

# Restart and auto-attach
gt session cycle wyvern/Toast --attach

# Use simple output (no TUI)
gt session cycle wyvern/Toast --no-tui
```

## Architecture

### Components

- **`model.go`**: Core Bubbletea model, state machine, and transition logic
- **`view.go`**: Rendering functions and styling with Lipgloss
- **`keys.go`**: Keyboard shortcuts and help system
- **`model_test.go`**: Unit tests for all components

### State Machine

```
Idle ──→ PreShutdown ──→ ShuttingDown ──→ ShutdownHook
                                              ↓
                        Complete ←── StartupHook ←── Starting ←── PreStart
```

### Transition Phases

| Phase | Description | Visual |
|-------|-------------|--------|
| `PhaseIdle` | No transition active | ⏸ Muted |
| `PhasePreShutdown` | Running pre-shutdown hooks | Spinner + Warning |
| `PhaseShuttingDown` | Stopping session, preserving context | Spinner + Warning |
| `PhaseShutdownHook` | Running post-shutdown hooks | Spinner + Info |
| `PhasePreStart` | Running pre-start checks | Spinner + Info |
| `PhaseStarting` | Creating new session | Spinner + Info |
| `PhaseStartupHook` | Running post-startup hooks | Spinner + Info |
| `PhaseComplete` | Transition successful | ✓ Success |
| `PhaseError` | Transition failed | ✗ Error |

## Features

### Progress Tracking

- **Progress bar**: Visual indicator of completion (0-100%)
- **Step counter**: Shows current step and total steps
- **Elapsed time**: Running timer since transition started
- **Estimated completion**: Based on typical transition times

### Context Preservation

The TUI preserves and displays context between sessions:

```go
PreservedData: {
    "previous_issue": "gt-abc123",
    "next_issue": "gt-xyz789",
    "last_output": "✓ Tests passed...",
    "custom_data": {...}
}
```

**What's preserved**:
- Previous hooked issue
- Next issue to work on
- Last 50 lines of terminal output
- Custom data from hooks

### Hook Integration

Lifecycle hooks are executed with visual feedback:

1. **Pre-shutdown**: `pre-shutdown` event
2. **Post-shutdown**: `post-shutdown` event
3. **Pre-start**: `pre-session-start` event
4. **Post-start**: `post-session-start` event

Hook failures block the transition and show detailed errors.

### Error Handling

When errors occur, the TUI shows:

```
╭─────────────────────────────────────╮
│ Error:                              │
│ Failed to start session: polecat    │
│ not found                           │
│                                     │
│ Pre-session-start hook blocked      │
╰─────────────────────────────────────╯
```

Common errors and recovery:
- **Session not found**: Automatically switches to start flow
- **Hook failure**: Shows hook error details with exit code
- **Tmux error**: Connection issues with troubleshooting steps
- **Timeout**: Long-running operations with cancel option

## Usage

### Basic Cycling

```bash
# Interactive TUI (default)
gt session cycle wyvern/Toast

# Watch the progress:
# 1. Pre-shutdown checks
# 2. Stop current session
# 3. Post-shutdown hooks
# 4. Pre-start checks
# 5. Start new session
# 6. Post-startup hooks
# 7. Complete
```

### Advanced Options

```bash
# Cycle to specific issue
gt session cycle wyvern/Toast --issue gt-abc123

# Cycle and attach immediately
gt session cycle wyvern/Toast --attach

# Force cycle (skip graceful shutdown)
gt session cycle wyvern/Toast --force

# Non-interactive mode
gt session cycle wyvern/Toast --no-tui
```

### Interactive Controls

While TUI is running:

- **`?`**: Toggle help display
- **`q` / `Ctrl-C`**: Quit (when idle/complete)
- **`r`**: Restart session (when idle)
- **`s`**: Stop session (when idle)
- **`Enter`**: Start session (when idle)
- **`f`**: Force stop (emergency)

## Integration

### Session Manager

```go
import (
    "github.com/steveyegge/gastown/internal/polecat"
    "github.com/steveyegge/gastown/internal/tui/session"
    tea "github.com/charmbracelet/bubbletea"
)

// Create TUI model
sessionMgr := polecat.NewSessionManager(tmux, rig)
m := session.New(sessionMgr, rig, "wyvern", "Toast")

// Set next issue (optional)
m.SetNextIssue("gt-abc123")

// Run TUI
p := tea.NewProgram(m, tea.WithAltScreen())
go func() {
    p.Send(session.StartCycleMsg{})
}()
finalModel, err := p.Run()

// Check result
if m, ok := finalModel.(session.Model); ok {
    if m.HasError() {
        fmt.Println("Error:", m.GetError())
    }
}
```

### Custom Hooks

Hooks receive transition context:

```bash
#!/bin/bash
# .claude/hooks/pre-start.sh

# Access preserved context
PREV_ISSUE=$(echo "$HOOK_METADATA" | jq -r '.previous_issue')
NEXT_ISSUE=$(echo "$HOOK_METADATA" | jq -r '.next_issue')

echo "Transitioning: $PREV_ISSUE → $NEXT_ISSUE"

# Block startup if needed
if [ "$PREV_ISSUE" == "$NEXT_ISSUE" ]; then
    echo "ERROR: Cannot cycle to same issue"
    exit 1
fi
```

## Testing

### Run Tests

```bash
go test ./internal/tui/session/... -v
```

Tests cover:
- ✅ State transitions
- ✅ Progress calculation
- ✅ Context preservation
- ✅ Message handling
- ✅ Error states
- ✅ Duration formatting
- ✅ String truncation
- ✅ Window resize handling

### Manual Testing

```bash
# Start test session
gt session start duneagent/test-polecat

# Cycle with TUI
gt session cycle duneagent/test-polecat

# Test error handling (kill tmux mid-transition)
tmux kill-session -t gt-duneagent-test-polecat

# Verify recovery
gt session cycle duneagent/test-polecat
```

## Visual Design

### Color Scheme

- **Primary** (Pink `#205`): Titles and highlights
- **Secondary** (Purple `#135`): Subtitles
- **Success** (Green `#42`): Completed operations
- **Warning** (Orange `#214`): Shutdown operations
- **Error** (Red `#196`): Failures
- **Info** (Blue `#39`): General information
- **Muted** (Gray `#241`): Secondary text

### Layout Example

```
╭─────────────────────────────────────╮
│ Session Cycling                     │
│ wyvern/Toast                        │
╰─────────────────────────────────────╯

╭─────────────────────────────────────╮
│ ● starting                          │
│ Starting new session...             │
│ Step 5/7                            │
╰─────────────────────────────────────╯

[████████████░░░░░░░░░░] 71%
Elapsed: 2.3s

╭─────────────────────────────────────╮
│ Preserved Context:                  │
│   Previous issue: gt-abc123         │
│   Next issue: gt-xyz789             │
│   Last output: Successfully comp... │
╰─────────────────────────────────────╯

│ Session started successfully

? toggle help   q quit
```

## Performance

### Overhead

- **TUI active time**: 2-5 seconds (typical transition)
- **Memory usage**: ~5MB (Bubbletea framework)
- **CPU usage**: Minimal (event-driven updates)
- **Terminal**: Requires ANSI color support

### Optimization

- Non-blocking hook execution
- Efficient terminal rendering (alt screen)
- Cleanup on exit
- Fallback to simple output (`--no-tui`)

## Troubleshooting

### TUI won't start

```bash
# Check terminal
echo $TERM

# Ensure tmux is running
tmux ls

# Try without TUI
gt session cycle <rig>/<polecat> --no-tui
```

### Transition hangs

1. Press `Ctrl-C` to cancel
2. Check hook scripts for infinite loops
3. Verify tmux session responsive: `tmux ls`
4. Force restart: `gt session cycle <rig>/<polecat> --force`

### Context not preserved

1. Verify capture works: `gt session capture <rig>/<polecat>`
2. Check hook metadata access
3. Ensure proper permissions

## Future Enhancements

- [ ] Multi-session cycling (batch operations)
- [ ] Real-time log streaming during transition
- [ ] Session migration between rigs
- [ ] Health metrics and analytics
- [ ] Automated session healing
- [ ] Remote session cycling (SSH support)

## See Also

- [Session Management](../../../docs/SESSION-MANAGEMENT.md)
- [Hook System](../../../docs/HOOKS.md)
- [Lifecycle Events](../../../docs/LIFECYCLE.md)
- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
