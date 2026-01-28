# Workers Command Documentation

## Overview

The `gt workers` command provides CLI tools for monitoring and reporting on worker (crew/polecat) status across all rigs. This gives visibility into what agents are doing, resource utilization, and system health.

## Commands

### `gt workers list [rig]`

List all workers with basic status information.

**Arguments:**
- `[rig]` - Optional rig name to filter workers (default: all rigs)

**Flags:**
- `--json` - Output as JSON

**Output includes:**
- Worker name and type (crew/polecat)
- Current state (idle/working/stalled/crashed)
- Current assignment (issue ID)
- Last activity timestamp
- Session status (running/stopped)

**Examples:**
```bash
gt workers list                  # List all workers across all rigs
gt workers list duneagent        # List workers in duneagent rig only
gt workers list --json           # JSON output for automation
```

**Sample Output:**
```
Workers

  ● duneagent/ericfriday [crew]  working
    Last activity: 2 minutes ago
  ● duneagent/rust [polecat]  working
    gt-3yj: Implement agent monitoring system
    Last activity: 1 minute ago
  ○ aardwolf_snd/ericfriday [crew]  idle

Total: 3 workers
```

### `gt workers status <rig>/<name>`

Show detailed status for a specific worker.

**Arguments:**
- `<rig>/<name>` - Worker identifier (e.g., `duneagent/ericfriday`)

**Flags:**
- `--json` - Output as JSON

**Output includes:**
- Worker type, rig, and name
- Current state and activity status
- Current assignment (issue ID and description)
- Session information (running/stopped, session ID)
- Git status (branch, uncommitted changes, commits ahead)
- Last activity timestamp and duration

**Examples:**
```bash
gt workers status duneagent/ericfriday
gt workers status duneagent/rust
gt workers status aardwolf_snd/ericfriday --json
```

**Sample Output:**
```
Worker: duneagent/ericfriday

  Type:          crew
  State:         working
  Branch:        main

Session
  Status:        running
  Session ID:    gt-duneagent-crew-ericfriday
  Last Activity: 14:23:15 (2 minutes ago)

Git Status
  Working Tree:  dirty
  Uncommitted:   3 files
  Commits Ahead: 2
```

### `gt workers active`

Show only workers that are currently active (have running sessions).

**Flags:**
- `--rig <name>` - Filter by specific rig
- `--json` - Output as JSON

**Examples:**
```bash
gt workers active                   # Show all active workers
gt workers active --rig duneagent   # Active workers in duneagent rig
gt workers active --json            # JSON output
```

**Sample Output:**
```
Active Workers

  ● duneagent/ericfriday [crew]  working
    Last activity: 1 minute ago
  ● duneagent/rust [polecat]  working
    gt-3yj: Implement agent monitoring system
    Last activity: 30 seconds ago

Total: 2 workers
```

### `gt workers health`

Overall system health check aggregating status across all workers.

**Flags:**
- `--json` - Output as JSON

**Output includes:**
- Total workers, active count, utilization percentage
- Stalled/crashed workers needing attention
- Resource bottlenecks (if available)
- Recommendations for action

**Examples:**
```bash
gt workers health         # Health check
gt workers health --json  # JSON output for automation
```

**Sample Output:**
```
System Health

  Total Workers:    5
  Active:           3
  Idle:             2
  Utilization:      60.0%

Recommendations
  • Consider assigning work to idle workers

  Overall:          HEALTHY
```

**Sample Output (with problems):**
```
System Health

  Total Workers:    5
  Active:           2
  Idle:             2
  Stalled:          1
  Utilization:      40.0%

Problems
  ⚠ duneagent/rust is stalled

Recommendations
  • Investigate 1 stalled worker(s)
  • Low utilization - consider assigning more work

  Overall:          ATTENTION NEEDED
```

## Status Indicators

### Session Status
- `●` (green) - Session running
- `○` (gray) - Session stopped

### Worker State
- `working` - Actively working on assigned task
- `idle` - Ready for work, no active task
- `stalled` - Session exists but not making progress
- `crashed` - Session terminated unexpectedly

### Git Status
- `clean` - No uncommitted changes
- `dirty` - Uncommitted changes present

## JSON Output Format

All commands support `--json` flag for machine-readable output.

### WorkerInfo Structure
```json
{
  "rig": "duneagent",
  "name": "ericfriday",
  "type": "crew",
  "state": "working",
  "issue": "gt-3yj: Implement agent monitoring",
  "branch": "main",
  "session_running": true,
  "session_id": "gt-duneagent-crew-ericfriday",
  "last_activity": "2026-01-28T14:23:15Z",
  "git_status": {
    "branch": "main",
    "uncommitted_files": 2,
    "commits_ahead": 1,
    "is_dirty": true
  }
}
```

### WorkerHealth Structure
```json
{
  "total_workers": 5,
  "active_workers": 3,
  "idle_workers": 2,
  "stalled_workers": 0,
  "error_workers": 0,
  "utilization_percent": 60.0,
  "problems": [],
  "recommendations": [
    "Consider assigning work to idle workers"
  ]
}
```

## Integration

### With Monitoring System

The workers command integrates with the `internal/monitoring/` package for status tracking. In the future, this could be enhanced to:

- Display real-time agent status from monitoring.StatusTracker
- Show inferred activity status from pane output analysis
- Track idle time and automatically mark idle agents
- Provide resource usage metrics (CPU, memory)

### With Tmux Sessions

Worker status is derived from tmux session state:
- Session existence indicates active worker
- Session activity timestamp shows last interaction
- Pane output can be monitored for status inference

### With Git State

Git status provides insight into worker progress:
- Uncommitted files indicate work in progress
- Commits ahead show unpushed changes
- Branch name shows current work context

## Implementation Notes

### Worker Discovery

Workers are discovered by scanning rig directories:
- **Crew workers**: `<rig>/crew/<name>/` directories
- **Polecats**: Via `polecat.Manager.List()` API

### Session Tracking

Sessions are identified by name pattern:
- Crew: `gt-<rig>-crew-<name>`
- Polecat: `gt-<rig>-<name>`

### State Determination

Worker state is determined by:
1. Tmux session existence (running vs stopped)
2. Polecat state field (for polecats)
3. Activity timestamp (active vs idle)

## Future Enhancements

Potential additions tracked in separate issues:

1. **Real-time monitoring** - Live status updates from tmux pane output
2. **Resource metrics** - CPU/memory usage per worker
3. **Historical tracking** - Worker activity over time
4. **Alert integration** - Notify on stalled/crashed workers
5. **Dashboard view** - Interactive TUI for monitoring
6. **Performance metrics** - Track issues completed per worker
7. **Load balancing** - Recommendations for work distribution

## Related Commands

- `gt crew list` - List only crew workers
- `gt crew status <name>` - Detailed crew worker status
- `gt polecat list` - List only polecats
- `gt polecat status <rig>/<name>` - Detailed polecat status
- `gt status` - Overall system status

The `gt workers` command provides a unified view across both worker types.

## Examples

### Monitor active work
```bash
# Quick check of what's happening
gt workers active

# Detailed status of specific worker
gt workers status duneagent/rust
```

### Health check before assigning work
```bash
# Check if workers available
gt workers health

# Find idle workers
gt workers list | grep idle
```

### Automation with JSON
```bash
# Get active workers as JSON for scripting
gt workers active --json | jq '.[] | select(.type=="polecat")'

# Check health and alert if problems
gt workers health --json | jq -e '.error_workers == 0 and .stalled_workers == 0'
```

### Debug stalled worker
```bash
# List all workers to find stalled
gt workers list

# Get detailed status
gt workers status duneagent/rust

# Check git state
cd ~/gt/duneagent/polecats/rust
git status
```
