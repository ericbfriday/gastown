# Session Cycling UX

## Overview

The Session Cycling UX provides a smooth, visually informative interface for transitioning between polecat work sessions in Gas Town. It replaces abrupt session restarts with a polished TUI experience that shows progress, preserves context, and integrates with the lifecycle hook system.

## Problem Statement

Before this implementation, session transitions were jarring:
- No visual feedback during transitions
- Context lost between sessions
- Unclear when hooks were executing
- Hard to debug transition failures
- No progress indication for long operations

## Solution

A Terminal User Interface (TUI) wrapper built with Bubbletea that provides:

1. **Visual Feedback**: Real-time display of transition phases
2. **Progress Tracking**: Progress bar with elapsed time
3. **Context Preservation**: Display of preserved data between sessions
4. **Hook Integration**: Visual indication of hook execution
5. **Error Handling**: Clear error messages with recovery options

## Architecture

### Components

```
internal/tui/session/
├── model.go       # Core TUI model and state machine
├── view.go        # Rendering and styling
├── keys.go        # Keyboard shortcuts
└── model_test.go  # Unit tests

internal/cmd/
└── session_cycle.go  # CLI integration
```

### Transition Phases

The session cycling process goes through these phases:

1. **PhaseIdle**: No transition in progress
2. **PhasePreShutdown**: Running pre-shutdown hooks
3. **PhaseShuttingDown**: Stopping session and preserving context
4. **PhaseShutdownHook**: Running post-shutdown hooks
5. **PhasePreStart**: Running pre-start checks
6. **PhaseStarting**: Creating new session
7. **PhaseStartupHook**: Running post-startup hooks
8. **PhaseComplete**: Transition successful
9. **PhaseError**: Transition failed

### State Machine

```
Idle
  ↓
PreShutdown → ShuttingDown → ShutdownHook
                                ↓
                           PreStart → Starting → StartupHook → Complete
                                                                    ↓
                                                                 Idle
```

### Context Preservation

During transitions, the system preserves:
- **Previous Issue**: The issue being worked on
- **Next Issue**: The issue to work on next
- **Last Output**: Recent terminal output (50 lines)
- **Custom Data**: Hook-provided context data

## Usage

### Basic Session Cycling

```bash
# Restart a session with TUI
gt session cycle wyvern/Toast

# Restart without TUI (simple output)
gt session cycle wyvern/Toast --no-tui

# Cycle to a specific issue
gt session cycle wyvern/Toast --issue gt-abc123

# Cycle and auto-attach
gt session cycle wyvern/Toast --attach
```

### Starting New Sessions

```bash
# Start new session with TUI
gt session cycle wyvern/NewPolecat

# Start with specific issue
gt session cycle wyvern/NewPolecat --issue gt-xyz789
```

### Interactive Controls

When the TUI is running:

- **`?`**: Toggle help display
- **`q`** or **Ctrl-C**: Quit (if idle or complete)
- **`r`**: Restart session (if idle)
- **`s`**: Stop session (if idle)
- **`f`**: Force stop (emergency)
- **Enter**: Start session (if idle)

## Visual Design

### Color Scheme

- **Primary** (Pink): Titles and highlights
- **Secondary** (Purple): Subtitles
- **Success** (Green): Completed operations
- **Warning** (Orange): Shutdown operations
- **Error** (Red): Failures
- **Info** (Blue): General information
- **Muted** (Gray): Secondary text

### Layout

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

? help   q quit
```

## Integration Points

### Session Manager

The TUI integrates with the existing `SessionManager`:

```go
// Create TUI model
m := session.New(sessionMgr, rig, rigName, polecatName)

// Start cycling
p := tea.NewProgram(m, tea.WithAltScreen())
go func() {
    p.Send(session.StartCycleMsg{})
}()
finalModel, err := p.Run()
```

### Hook System

Lifecycle hooks are executed transparently:

1. **Pre-session-start**: Before creating new session
2. **Post-session-start**: After session ready
3. **Pre-shutdown**: Before stopping session
4. **Post-shutdown**: After session terminated

The TUI shows visual feedback during hook execution.

### Tmux Integration

The TUI uses the existing tmux integration:

- Session creation via `tmux new-session`
- Session termination via `KillSessionWithProcesses`
- Output capture for context preservation
- Session health checks

## Error Handling

### Common Errors

1. **Session Already Running**
   - Action: Show error and suggest force stop
   - Recovery: User can force stop or cancel

2. **Session Not Found**
   - Action: Offer to start new session
   - Recovery: Automatic transition to start flow

3. **Hook Failure**
   - Action: Display hook error details
   - Recovery: User can retry or skip hook

4. **Tmux Connection Lost**
   - Action: Show connection error
   - Recovery: Check tmux server and retry

### Error Display

```
╭─────────────────────────────────────╮
│ Error:                              │
│ Failed to start session: tmux       │
│ server not running                  │
│                                     │
│ Failed to stop session              │
╰─────────────────────────────────────╯
```

## Performance Considerations

### Non-Blocking Operations

- Hook execution happens asynchronously
- Progress updates via message passing
- No blocking UI renders

### Minimal Overhead

- TUI only active during transitions (seconds)
- Cleanup on completion/error
- Efficient terminal rendering

### Fallback Mode

For automated/scripted use:
```bash
gt session cycle wyvern/Toast --no-tui
```

Simple text output without TUI overhead.

## Testing

### Unit Tests

```bash
go test ./internal/tui/session/...
```

Tests cover:
- State transitions
- Progress calculation
- Context preservation
- Message handling
- Error states
- Duration formatting
- String truncation

### Integration Tests

Manual testing workflow:
1. Start TUI: `gt session cycle test-rig/test-polecat`
2. Verify progress display
3. Check context preservation
4. Test keyboard shortcuts
5. Trigger errors (kill tmux mid-transition)
6. Verify recovery

## Future Enhancements

### Phase 1 (Current)
- ✅ Basic TUI with progress display
- ✅ Context preservation
- ✅ Hook integration
- ✅ Error handling

### Phase 2 (Planned)
- [ ] Multi-session cycling (batch operations)
- [ ] Session health dashboard
- [ ] Real-time log streaming
- [ ] Advanced filtering and sorting

### Phase 3 (Future)
- [ ] Remote session cycling (SSH support)
- [ ] Session migration between rigs
- [ ] Automated session healing
- [ ] Analytics and metrics

## Implementation Notes

### Design Decisions

1. **Bubbletea Framework**: Already used in project (convoy, feed TUIs)
2. **Phase-Based State Machine**: Clear progression, easy to debug
3. **Auto-Advance Mode**: Default behavior for smooth UX
4. **Context Preservation**: Last 50 lines sufficient for most cases
5. **Spinner + Progress Bar**: Visual variety for different phases

### Technical Constraints

- Requires tmux for session management
- Terminal must support ANSI colors
- Minimum terminal width: 60 columns
- Alt screen mode for clean exit

### Compatibility

- Works with existing session commands
- Compatible with scripts via `--no-tui`
- Integrates with hook system
- Preserves session manager behavior

## References

- [Session Manager](../internal/polecat/session_manager.go)
- [Hook System](../internal/hooks/types.go)
- [Lifecycle Events](../internal/daemon/lifecycle.go)
- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)

## See Also

- [Session Management](SESSION-MANAGEMENT.md)
- [Hook System](HOOKS.md)
- [TUI Components](TUI-COMPONENTS.md)
