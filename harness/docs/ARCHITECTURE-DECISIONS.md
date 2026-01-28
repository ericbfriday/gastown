# Architecture Decision Records (ADR)

## ADR-001: Shell-Based Implementation

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need automation harness for continuous Claude agent spawning

### Decision
Implement harness using bash shell scripts rather than compiled language (Go, Rust) or interpreted language (Python, Node.js).

### Rationale
- Direct integration with existing Gastown tools (gt, bd, git)
- Simple and debuggable
- No additional runtime dependencies
- Easy to inspect and modify
- Matches Gastown's philosophy of composable tools
- Lower barrier to contribution

### Consequences
**Positive:**
- Simple implementation and maintenance
- Easy debugging (just read the script)
- No dependency management needed
- Fast iteration during development

**Negative:**
- Less performant than compiled languages
- Bash-specific (portability considerations)
- Limited data structure support
- Error handling can be verbose

**Mitigations:**
- Use `set -euo pipefail` for safer scripts
- Modular functions for maintainability
- Comprehensive error checking
- Clear variable naming conventions

---

## ADR-002: File-Based State Management

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need persistent state across harness iterations and crashes

### Decision
Use JSON files for state storage rather than database or in-memory state.

### Rationale
- Human-readable and inspectable
- Easy to debug and recover from failures
- No database dependencies (SQLite, PostgreSQL)
- Works well with git for versioning
- Simple to implement and maintain
- Atomic writes via temp file + move

### Consequences
**Positive:**
- Easy inspection and debugging
- Simple recovery from crashes
- No external dependencies
- Version control friendly

**Negative:**
- Slower than in-memory or binary formats
- No built-in concurrency control
- File size grows over time

**Mitigations:**
- JSON is fast enough for current scale
- Single-writer pattern (no concurrency yet)
- Archival of old session files

**Files:**
- `state/queue.json` - Work queue
- `state/current-session.json` - Active session
- `docs/sessions/*.json` - Archived sessions

---

## ADR-003: Interrupt Mechanism

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need way for agents to signal when human attention required

### Decision
Use file-based interrupt signaling via presence of `state/interrupt-request.txt`.

### Rationale
- Simple and robust
- Accessible from any process (agents can create easily)
- Easy to inspect and debug
- Works across process boundaries
- Graceful handling (not abrupt signal)
- Human-readable reason in file content

### Consequences
**Positive:**
- Simple implementation
- Easy for agents to use
- Human-readable reason
- Debuggable

**Negative:**
- Requires filesystem access
- Polling-based detection (30s interval)
- Not instant notification

**Alternatives Considered:**
- Signal-based (SIGUSR1) - Too complex, harder to debug
- Socket/pipe - Overkill for use case
- Database flag - No database in stack

**Implementation:**
```bash
# Agent signals
echo "Need clarification on auth" > state/interrupt-request.txt

# Harness detects
if [[ -f "$INTERRUPT_FILE" ]]; then
  preserve_context
  wait_for_resume
fi

# Human resolves
rm state/interrupt-request.txt
```

---

## ADR-004: Minimal Context Bootstrap

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Agents need starting prompt but want to avoid context overload

### Decision
Provide minimal bootstrap prompt with pointers to documentation rather than full context upfront.

### Rationale
- Efficient token usage
- Teaches agents to discover and build context
- Simulates real developer onboarding
- Prevents context overload
- Encourages documentation-driven development
- Scalable approach

### Consequences
**Positive:**
- Efficient token usage
- Agents learn discovery patterns
- Scalable approach
- Documentation becomes critical (good pressure)

**Negative:**
- Longer ramp-up time per agent
- Assumes documentation is comprehensive
- Agents may miss important context

**Alternatives Considered:**
- Full context upfront - Token inefficient, doesn't scale
- No context - Too minimal, agents would be lost
- Progressive loading - Future enhancement

**Bootstrap Contents:**
- Session ID and iteration
- Pointers to documentation
- Workflow instructions
- Knowledge preservation guide
- Interrupt mechanism usage

---

## ADR-005: Multiple Knowledge Preservation

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need to preserve research and findings across sessions

### Decision
Use multiple preservation mechanisms rather than single source.

### Mechanisms
1. **Serena Memories** - Machine-readable, cross-session
2. **Session Documentation** - Session-specific context
3. **Research Notes** - Human-readable findings
4. **Decision Logs** - Architecture decisions (like this file)

### Rationale
- Redundancy ensures knowledge not lost
- Different formats for different use cases
- Human and machine readable options
- Fits with existing Gastown patterns
- Flexibility in storage format

### Consequences
**Positive:**
- Comprehensive knowledge capture
- Multiple access patterns
- Redundancy prevents loss
- Format flexibility

**Negative:**
- Requires agent discipline to preserve
- No automatic deduplication
- Multiple places to check

**Mitigations:**
- Clear guidelines in bootstrap prompt
- Examples of each mechanism
- Agent responsibility to preserve

**Structure:**
```
docs/
├── research/          # Ad-hoc research findings
├── sessions/          # Session context
└── decisions/         # Architecture decisions

Serena:
~/.serena/memories/    # Cross-session knowledge
```

---

## ADR-006: YAML Configuration

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need configurable behavior without code changes

### Decision
Use YAML-based configuration with environment variable overrides.

### Rationale
- Human-readable format
- Common pattern (familiar to users)
- Supports complex nested structure
- Environment variable overrides for runtime
- Comments for documentation

### Consequences
**Positive:**
- Easy to read and modify
- Runtime overrides available
- Supports complex config
- Self-documenting with comments

**Negative:**
- Requires YAML parser (yq or fallback to jq)
- More verbose than env vars alone

**Alternatives Considered:**
- JSON - Less human-friendly for config
- Environment variables only - Too many variables
- Command-line flags - Not persistent

**Key Sections:**
- Loop control
- Work routing
- Interrupt conditions
- Quality gates
- Notifications
- Rig-specific settings

---

## ADR-007: Quality Gates

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need to enforce quality before work completion

### Decision
Implement configurable pre/post-work quality gates.

### Rationale
- Catch issues early
- Enforce standards
- Configurable per-rig
- Clear pass/fail criteria
- Prevents broken work from being marked complete

### Gates
**Pre-work (Advisory):**
- Check main branch health
- File issues if failing

**Post-work (Required):**
- Tests must pass
- Build must succeed
- Git working tree must be clean
- Changes must be pushed

### Actions on Failure
- `interrupt` - Pause for human resolution
- `file_issue` - Create beads issue
- `warn` - Log but continue

### Consequences
**Positive:**
- Quality enforcement
- Configurable per-rig
- Clear criteria
- Prevents broken commits

**Negative:**
- May slow work completion
- Tests must be maintained
- False positives possible

**Configuration:**
```yaml
quality_gates:
  post_work:
    - name: "Run tests"
      command: "go test ./..."
      required: true
      on_failure: "interrupt"
```

---

## ADR-008: Placeholder Agent Spawning

**Status:** Accepted (Temporary)
**Date:** 2026-01-27
**Context:** Phase 1 focus is infrastructure, not agent execution

### Decision
Implement `spawn_agent()` as placeholder in Phase 1, defer actual implementation to Phase 2.

### Rationale
- Phase 1 validates infrastructure (loop, queue, interrupts)
- Agent spawning requires Claude Code CLI research
- Better to get infrastructure right first
- Placeholder allows testing of loop mechanics

### Consequences
**Positive:**
- Focus on infrastructure correctness
- Loop can be tested without agent complexity
- Clean separation of concerns

**Negative:**
- Cannot test end-to-end workflow yet
- Phase 1 is not fully functional

**Phase 2 Requirements:**
- Research Claude Code CLI invocation
- Implement actual spawning
- Session lifecycle management
- Output capture
- Completion detection

**Placeholder Implementation:**
```bash
spawn_agent() {
  log "Would spawn Claude Code with:"
  log "  - Session ID: $session_id"
  log "  - Bootstrap: $prompt_file"
  log "  - Work: $work_item"

  jq '.status = "spawned"' "$SESSION_FILE" > "$SESSION_FILE.tmp"
  mv "$SESSION_FILE.tmp" "$SESSION_FILE"
}
```

---

## ADR-009: Status Reporting

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need visibility into harness operation

### Decision
Implement file-based logging with dedicated status reporting script.

### Rationale
- Simple to implement
- Easy to inspect logs
- Status script for quick overview
- Foundation for future dashboard
- No runtime dependencies

### Components
- `state/iteration.log` - Main loop log
- `docs/sessions/*.log` - Session logs
- `scripts/report-status.sh` - Status reporter

### Consequences
**Positive:**
- Good visibility into operation
- Easy to debug issues
- Simple implementation
- Human-readable

**Negative:**
- No real-time dashboard (yet)
- Manual refresh needed
- Log rotation required eventually

**Future Enhancements:**
- Web dashboard
- Metrics export (Prometheus)
- Alert system
- Real-time updates

---

## ADR-010: Work Queue Strategy

**Status:** Accepted
**Date:** 2026-01-27
**Context:** Need to pull work from multiple sources

### Decision
Unified queue with priority-based ordering, pulling from beads and gt.

### Rationale
- Single source of truth for available work
- Can prioritize across sources
- Easy to inspect and manage
- Supports manual work injection

### Implementation
```bash
# Pull from beads
bd ready --json

# Pull from gt
gt ready --json

# Merge and prioritize
jq -s '.[0] + .[1] | sort_by(.priority) | reverse' > queue.json
```

### Consequences
**Positive:**
- Unified work view
- Clear prioritization
- Inspectable
- Manual override possible

**Negative:**
- Requires syncing from sources
- Stale data if sources change
- Priority logic in multiple places

**Alternatives Considered:**
- Poll sources directly - No single view
- Database queue - No database in stack
- Redis queue - Unnecessary dependency

---

## Decision Principles

### Simplicity First
Prefer simple, debuggable solutions over complex ones.

### File-Based Operations
Use filesystem for state, signaling, and persistence.

### Minimal Dependencies
Rely on existing tools (gt, bd, git), avoid new dependencies.

### Human-Readable Formats
JSON, YAML, Markdown over binary formats.

### Graceful Degradation
Handle errors gracefully, preserve state, continue when possible.

### Observable Operation
Comprehensive logging, status reporting, inspectable state.

---

**Document Version:** 1.0
**Last Updated:** 2026-01-27
**Status:** Active - Phase 1 Decisions
**Next Review:** After Phase 2 completion
