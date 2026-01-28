# Claude Automation Harness - Implementation Roadmap

**Version:** 1.0
**Created:** 2026-01-27
**Status:** Phase 1 - Foundation Complete

## Implementation Status

### ✅ Phase 1: Foundation (COMPLETE)

**Goal:** Core infrastructure and basic loop mechanism

**Completed:**
- [x] Directory structure created
- [x] Main loop script (`loop.sh`) with logging and iteration control
- [x] Configuration system (`config.yaml`)
- [x] Bootstrap prompt template
- [x] Helper scripts (queue, interrupt, context, status)
- [x] Documentation (README, workflow guide)
- [x] Basic monitoring and reporting

**Deliverables:**
- Functional loop structure ✅
- Work queue management ✅
- Interrupt detection framework ✅
- Context preservation system ✅
- Status reporting ✅

### 🔄 Phase 2: Claude Code Integration (NEXT)

**Goal:** Actually spawn and manage Claude Code agents

**Tasks:**
1. [ ] Research Claude Code CLI invocation
2. [ ] Implement agent spawning in `spawn_agent()`
3. [ ] Session management (start, monitor, terminate)
4. [ ] Bootstrap prompt injection
5. [ ] Session output capture and logging

**Key Challenge:** How to programmatically invoke Claude Code CLI with custom prompts

**Approach:**
```bash
# Option 1: Use claude CLI directly
claude --prompt-file prompts/bootstrap.md

# Option 2: Use /handoff or similar mechanism
# (TBD based on Claude Code capabilities)

# Option 3: Pipe commands to running session
# (Investigate if Claude Code supports this)
```

**Deliverables:**
- Actual agent spawning working
- Session lifecycle management
- Output/log capture
- Agent termination on completion

**Estimated Effort:** 2-3 days

### 🔜 Phase 3: Knowledge & Documentation (PLANNED)

**Goal:** Complete knowledge preservation and discovery

**Tasks:**
1. [ ] Document template system
2. [ ] Automated research preservation
3. [ ] Serena memory integration
4. [ ] Session summary generation
5. [ ] Knowledge indexing and search

**Features:**
- Auto-generate session summaries
- Preserve research automatically
- Index documentation for quick discovery
- Cross-session knowledge graph

**Deliverables:**
- Documentation templates
- Auto-preservation hooks
- Serena integration complete
- Knowledge discovery tools

**Estimated Effort:** 3-4 days

### 🔜 Phase 4: Work Orchestration (PLANNED)

**Goal:** Intelligent work routing and tracking

**Tasks:**
1. [ ] Enhanced work queue with prioritization
2. [ ] Rig-aware dispatching
3. [ ] Convoy integration
4. [ ] Parallel work support (future)
5. [ ] Dependency tracking

**Features:**
- Smart work routing to appropriate rigs
- Priority-based scheduling
- Convoy-based batch tracking
- Support for parallel agents

**Deliverables:**
- Intelligent queue manager
- Rig dispatcher
- Convoy tracking integration
- Parallel agent framework

**Estimated Effort:** 4-5 days

### 🔜 Phase 5: Advanced Monitoring (PLANNED)

**Goal:** Comprehensive observability and control

**Tasks:**
1. [ ] Metrics collection system
2. [ ] Web dashboard (optional)
3. [ ] Real-time status updates
4. [ ] Alert system
5. [ ] Performance analytics

**Features:**
- Real-time metrics (iterations/hour, success rate)
- Web-based status dashboard
- Slack/email notifications
- Performance tracking
- Historical analytics

**Deliverables:**
- Metrics database
- Status dashboard (web or TUI)
- Notification system
- Analytics reports

**Estimated Effort:** 5-7 days

### 🔜 Phase 6: Hardening & Polish (PLANNED)

**Goal:** Production readiness

**Tasks:**
1. [ ] Error recovery improvements
2. [ ] Edge case handling
3. [ ] Performance optimization
4. [ ] Security review
5. [ ] Complete documentation
6. [ ] End-to-end testing

**Features:**
- Robust error handling
- Automatic recovery from crashes
- Optimized performance
- Security hardening
- Comprehensive docs

**Deliverables:**
- Production-ready system
- Complete test suite
- Full documentation
- Runbooks and playbooks

**Estimated Effort:** 3-5 days

## Current Focus: Phase 2

### Immediate Next Steps

1. **Research Claude Code CLI**
   - How to invoke with custom prompts
   - How to inject bootstrap context
   - How to monitor session progress
   - How to capture output/logs

2. **Implement Spawning**
   - Update `spawn_agent()` function
   - Test basic agent invocation
   - Verify bootstrap prompt delivery
   - Confirm session lifecycle works

3. **Test End-to-End**
   - Run single iteration
   - Verify agent receives work
   - Confirm output is captured
   - Validate state preservation

### Critical Questions

**Q1: How do we invoke Claude Code with a custom prompt file?**
- Need to investigate Claude Code CLI options
- Possibly use --prompt-file or similar
- May need to use stdin redirection
- Could use chat continuation if available

**Q2: How do we monitor session progress?**
- Check if Claude Code exposes session status
- May need to monitor process state
- Could parse output logs
- Might use file-based signaling

**Q3: How do we inject dynamic context (SESSION_ID, etc.)?**
- Template expansion in bootstrap.md
- Environment variables
- Stdin injection
- Pre-processed prompt file

**Q4: How do we know when session completes?**
- Process exit status
- Marker file creation
- Output parsing
- Timeout detection

## Dependencies & Blockers

### Current Blockers

None - Phase 1 complete

### Phase 2 Dependencies

- Claude Code CLI documentation/examples
- Understanding of Claude Code prompt injection
- Session lifecycle management approach

### Technical Debt

- `spawn_agent()` is currently placeholder
- No actual session monitoring
- No output capture
- Limited error recovery

## Testing Strategy

### Phase 1 Testing ✅

- [x] Loop structure executes
- [x] Queue management works
- [x] Interrupt detection functions
- [x] Context preservation saves files
- [x] Status reporting displays correctly

### Phase 2 Testing Plan

- [ ] Agent spawning executes Claude Code
- [ ] Bootstrap prompt is delivered
- [ ] Session runs and completes
- [ ] Output is captured
- [ ] State is preserved correctly

### Integration Testing Plan

- [ ] End-to-end workflow (queue → spawn → work → complete)
- [ ] Interrupt and resume cycle
- [ ] Knowledge preservation across sessions
- [ ] Multi-iteration continuity

## Metrics & Success Criteria

### Phase 1 Success ✅

- Loop runs without crashing
- Queue can be managed
- Interrupts are detected
- Context is preserved
- Status is visible

### Phase 2 Success Criteria

- Agents spawn successfully
- Work is executed by agents
- Sessions complete and cleanup
- Logs are captured
- State persists correctly

### Overall Success Criteria

- Runs continuously for 24+ hours without intervention
- Agents complete work successfully
- Knowledge accumulates over time
- Interrupts are timely and appropriate
- Human intervention is minimal
- Work progresses across rigs

## Risk Assessment

### High Risk

1. **Claude Code Integration** - Unknown API/CLI capabilities
   - Mitigation: Research thoroughly, start simple, iterate

2. **Session Management** - Complex lifecycle, state tracking
   - Mitigation: Robust error handling, state preservation

### Medium Risk

1. **Performance** - Resource usage with continuous operation
   - Mitigation: Monitoring, limits, optimization

2. **Error Recovery** - Graceful handling of failures
   - Mitigation: Comprehensive error cases, testing

### Low Risk

1. **Configuration** - YAML parsing, validation
   - Mitigation: Schema validation, defaults

2. **Documentation** - Keeping docs in sync
   - Mitigation: Documentation as code, reviews

## Timeline Estimate

**Total Implementation:** 4-5 weeks

- Phase 1: ✅ Complete (1 week)
- Phase 2: 2-3 days
- Phase 3: 3-4 days
- Phase 4: 4-5 days
- Phase 5: 5-7 days (optional)
- Phase 6: 3-5 days

**Minimum Viable Product:** End of Phase 4 (~2 weeks from now)
**Full Featured:** End of Phase 6 (~4-5 weeks from now)

## Resources & References

### Documentation

- [Claude Harness Workflow](../docs/CLAUDE-HARNESS-WORKFLOW.md) - Full implementation guide
- [Gastown Guide](../GASTOWN-CLAUDE.md) - System overview
- [Agent Instructions](../AGENTS.md) - Basic workflow

### Tools & Dependencies

- Gastown (`gt`) - v0.4.0
- Beads (`bd`) - v0.47.1
- Claude Code CLI - TBD version
- jq - 1.6+
- git - 2.x

### External Resources

- Claude Code documentation
- Ralph Wiggum loop pattern
- Multi-agent orchestration patterns

## Notes & Learnings

### Design Decisions

1. **Shell-based loop** - Simple, debuggable, integrates with existing tools
2. **File-based state** - JSON files for persistence, easy to inspect
3. **Interrupt mechanism** - File-based signaling, simple and reliable
4. **Minimal bootstrap** - Agents build context as needed, efficient

### Lessons Learned

1. **Keep it simple** - Complex systems fail, simple systems work
2. **File-based signaling** - Robust and debuggable
3. **Document early** - Helps future agents and humans
4. **Test incrementally** - Don't build too much before testing

### Open Questions

1. How to best integrate with Claude Code sessions?
2. Should we support multiple parallel agents?
3. How to handle session crashes gracefully?
4. What's the right interval for interrupt checks?
5. How to measure agent "success" quantitatively?

---

**Next Review:** After Phase 2 completion
**Last Updated:** 2026-01-27
**Maintainer:** Eric Friday (via Claude)
