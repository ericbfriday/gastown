# Phase 3 Testing Strategy

**Version:** 1.0
**Status:** Planning
**Phase:** 3 (Parallel Agent Support)
**Date:** 2026-01-28

## Overview

This document defines the comprehensive testing strategy for Phase 3 parallel agent support. Testing is organized into multiple layers: unit tests (component isolation), integration tests (component interaction), system tests (end-to-end workflows), and stress tests (edge cases and limits).

**Testing Philosophy:**
- Test early, test often
- Automate everything
- Fast feedback loops
- Comprehensive coverage of critical paths
- Explicit testing of failure scenarios

**Quality Targets:**
- Unit test coverage: >90%
- Integration test coverage: >80%
- All critical paths tested
- Zero known race conditions
- All failure scenarios covered

---

## Test Pyramid

```
                    ┌─────────────────┐
                    │  Stress Tests   │  ← 24hr stability, 10 agents, high contention
                    │  (~5 tests)     │
                    └─────────────────┘
                 ┌──────────────────────┐
                 │  System Tests        │  ← End-to-end workflows, real agents
                 │  (~10 tests)         │
                 └──────────────────────┘
            ┌───────────────────────────────┐
            │  Integration Tests            │  ← Component pairs, mock agents
            │  (~20 tests)                  │
            └───────────────────────────────┘
       ┌──────────────────────────────────────────┐
       │  Unit Tests                              │  ← Individual functions, isolated
       │  (~25 tests)                             │
       └──────────────────────────────────────────┘
```

**Total Tests:** ~60 tests
**Estimated Test Code:** ~3000 lines

---

## Layer 1: Unit Tests

**Goal:** Test individual functions in isolation
**Execution Time:** <1 minute total
**Mock Level:** High (mock external dependencies)

### Test Suite 1.1: Atomic Claims (`tests/test-atomic-claims.sh`)

**Functions Under Test:**
- `claim_work_item(item_id, agent_id)`
- `release_work_item(item_id)`
- `get_claim_owner(item_id)`
- `list_claimed_work()`

**Test Cases:**

1. **test_single_claim_succeeds**
   - Setup: 1 work item, 1 agent
   - Action: Claim work item
   - Assert: Claim succeeds, lock file created, owner recorded

2. **test_duplicate_claim_fails**
   - Setup: 1 work item, 2 agents
   - Action: Agent 1 claims, then agent 2 tries
   - Assert: Agent 1 succeeds, agent 2 fails

3. **test_parallel_claims_atomic**
   - Setup: 1 work item, 10 agents
   - Action: All 10 attempt to claim simultaneously
   - Assert: Exactly 1 succeeds, 9 fail

4. **test_claim_release_reclaim**
   - Setup: 1 work item, 2 agents
   - Action: Agent 1 claims, releases, agent 2 claims
   - Assert: Both claims succeed (sequentially)

5. **test_stale_claim_detection**
   - Setup: 1 work item claimed 3 hours ago
   - Action: Check for stale claims
   - Assert: Claim detected as stale

6. **test_claim_ownership_tracked**
   - Setup: 3 work items, 3 agents
   - Action: Each agent claims one item
   - Assert: Ownership correctly recorded for all

**Mock Requirements:**
- Mock work queue JSON file
- Mock agent state directories

**Validation:**
```bash
./tests/test-atomic-claims.sh
# Expected: 6/6 tests pass in <10 seconds
```

---

### Test Suite 1.2: Worktree Management (`tests/test-worktree-isolation.sh`)

**Functions Under Test:**
- `setup_agent_worktree(agent_id)`
- `cleanup_agent_worktree(agent_id)`
- `verify_worktree_isolation(agent_id)`

**Test Cases:**

1. **test_worktree_creation**
   - Setup: Clean git repo
   - Action: Create worktree for agent-1
   - Assert: Worktree directory exists, git configured

2. **test_multiple_worktrees_simultaneously**
   - Setup: Clean git repo
   - Action: Create 3 worktrees in parallel
   - Assert: All 3 created successfully, no conflicts

3. **test_commit_isolation**
   - Setup: 2 worktrees
   - Action: Commit in worktree-1, check worktree-2
   - Assert: Commit not visible in worktree-2

4. **test_branch_independence**
   - Setup: 2 worktrees
   - Action: Different branches in each
   - Assert: Branches don't interfere

5. **test_worktree_cleanup**
   - Setup: 1 worktree
   - Action: Cleanup worktree
   - Assert: Directory removed, no orphaned branches

6. **test_force_cleanup_locked_worktree**
   - Setup: 1 worktree with lock
   - Action: Force cleanup
   - Assert: Cleanup succeeds despite lock

**Mock Requirements:**
- Test git repository
- Mock harness directory structure

**Validation:**
```bash
./tests/test-worktree-isolation.sh
# Expected: 6/6 tests pass in <15 seconds
```

---

### Test Suite 1.3: Status Aggregation (`tests/test-status-aggregation.sh`)

**Functions Under Test:**
- `aggregate_agent_status()`
- `aggregate_queue_stats()`
- `calculate_metrics()`

**Test Cases:**

1. **test_aggregate_zero_agents**
   - Setup: No agents running
   - Action: Aggregate status
   - Assert: Empty agents array, coordinator info present

2. **test_aggregate_single_agent**
   - Setup: 1 agent (mocked state)
   - Action: Aggregate status
   - Assert: 1 agent in status, correct fields

3. **test_aggregate_multiple_agents**
   - Setup: 3 agents (mocked state)
   - Action: Aggregate status
   - Assert: 3 agents in status, all with correct data

4. **test_aggregate_mixed_states**
   - Setup: Agents in different states (idle, working, dead)
   - Action: Aggregate status
   - Assert: States correctly represented

5. **test_queue_statistics**
   - Setup: Queue with available, claimed, completed work
   - Action: Aggregate queue stats
   - Assert: Counts accurate

6. **test_metrics_calculation**
   - Setup: Historical data for metrics
   - Action: Calculate metrics
   - Assert: Throughput, success rate, etc. correct

**Mock Requirements:**
- Mock agent state directories
- Mock work queue file
- Mock historical data

**Validation:**
```bash
./tests/test-status-aggregation.sh
# Expected: 6/6 tests pass in <10 seconds
```

---

### Test Suite 1.4: Health Monitoring (`tests/test-health-checks.sh`)

**Functions Under Test:**
- `check_agent_process(agent_id)`
- `check_agent_alive(agent_id)`
- `check_stale_work()`

**Test Cases:**

1. **test_alive_agent_detection**
   - Setup: Agent with recent heartbeat
   - Action: Check if alive
   - Assert: Detected as alive

2. **test_dead_agent_no_heartbeat**
   - Setup: Agent with old heartbeat (>2 min)
   - Action: Check if alive
   - Assert: Detected as dead

3. **test_crashed_agent_no_process**
   - Setup: PID file but no process
   - Action: Check if process alive
   - Assert: Detected as crashed

4. **test_stale_work_detection**
   - Setup: Work claimed >2 hours ago
   - Action: Check for stale work
   - Assert: Detected as stale

5. **test_healthy_agent_all_checks**
   - Setup: Healthy agent (process + heartbeat)
   - Action: Run all health checks
   - Assert: All checks pass

**Mock Requirements:**
- Mock agent state files
- Mock process table (for PID checks)

**Validation:**
```bash
./tests/test-health-checks.sh
# Expected: 5/5 tests pass in <10 seconds
```

---

## Layer 2: Integration Tests

**Goal:** Test component interactions with realistic data
**Execution Time:** 1-5 minutes per suite
**Mock Level:** Medium (mock Claude CLI, real coordination)

### Test Suite 2.1: Agent Worker Lifecycle (`tests/test-agent-worker.sh`)

**Components Under Test:**
- Agent worker script
- Work claim mechanism
- Heartbeat updates
- Work release

**Test Cases:**

1. **test_worker_spawns_successfully**
   - Setup: Mock work queue
   - Action: Spawn worker
   - Assert: PID recorded, state initialized

2. **test_worker_claims_work**
   - Setup: Available work in queue
   - Action: Worker attempts claim
   - Assert: Work claimed, status updated

3. **test_worker_heartbeat_updates**
   - Setup: Running worker
   - Action: Wait 35 seconds
   - Assert: Heartbeat file updated

4. **test_worker_executes_work**
   - Setup: Worker with claimed work
   - Action: Execute work (mock agent)
   - Assert: Work executed, logs created

5. **test_worker_releases_work_on_completion**
   - Setup: Worker completing work
   - Action: Work finishes
   - Assert: Work released, claim removed

6. **test_worker_handles_multiple_cycles**
   - Setup: Queue with 3 work items
   - Action: Worker processes all 3
   - Assert: All completed, worker still running

7. **test_worker_graceful_shutdown**
   - Setup: Running worker
   - Action: Send SIGTERM
   - Assert: Releases work, cleans up, exits

**Mock Requirements:**
- Mock Claude CLI (`tests/mocks/mock-claude.sh`)
- Mock work queue

**Validation:**
```bash
./tests/test-agent-worker.sh
# Expected: 7/7 tests pass in <2 minutes
```

---

### Test Suite 2.2: Coordinator and Agent Pool (`tests/test-coordinator.sh`)

**Components Under Test:**
- Parallel coordinator
- Agent pool management
- Agent spawning/termination

**Test Cases:**

1. **test_coordinator_startup**
   - Setup: Clean state
   - Action: Start coordinator
   - Assert: PID recorded, state initialized

2. **test_spawn_target_agent_count**
   - Setup: Target = 3 agents
   - Action: Coordinator spawns agents
   - Assert: Exactly 3 agents spawned

3. **test_agents_staggered**
   - Setup: Stagger = 5 seconds
   - Action: Spawn 3 agents
   - Assert: ~5 seconds between each spawn

4. **test_maintain_agent_pool**
   - Setup: Target = 3, running = 2
   - Action: Pool management runs
   - Assert: 1 more agent spawned

5. **test_scale_down_excess_agents**
   - Setup: Target = 2, running = 3
   - Action: Pool management runs
   - Assert: 1 agent terminated

6. **test_coordinator_graceful_shutdown**
   - Setup: Coordinator with 3 agents
   - Action: Send SIGTERM to coordinator
   - Assert: All agents terminated, cleanup complete

**Mock Requirements:**
- Mock agent workers
- Mock work queue

**Validation:**
```bash
./tests/test-coordinator.sh
# Expected: 6/6 tests pass in <3 minutes
```

---

### Test Suite 2.3: Failure Recovery (`tests/test-failure-recovery.sh`)

**Components Under Test:**
- Agent crash detection
- Work release on failure
- Context preservation
- Respawn logic

**Test Cases:**

1. **test_detect_crashed_agent**
   - Setup: Agent crashes (kill -9)
   - Action: Health monitor runs
   - Assert: Crash detected within 2 minutes

2. **test_release_work_on_crash**
   - Setup: Agent with claimed work crashes
   - Action: Recovery runs
   - Assert: Work released back to queue

3. **test_preserve_crash_context**
   - Setup: Agent crashes
   - Action: Recovery runs
   - Assert: Logs archived, state preserved

4. **test_respawn_after_crash**
   - Setup: Agent crashes (first time)
   - Action: Recovery runs
   - Assert: Replacement agent spawned

5. **test_failure_threshold_stops_respawn**
   - Setup: Agent crashes 5 times
   - Action: Recovery runs
   - Assert: No respawn, overseer notified

6. **test_multiple_simultaneous_crashes**
   - Setup: 2 agents crash at same time
   - Action: Recovery runs
   - Assert: Both recovered independently

**Mock Requirements:**
- Mock agents that can crash on command
- Mock notification system

**Validation:**
```bash
./tests/test-failure-recovery.sh
# Expected: 6/6 tests pass in <5 minutes
```

---

### Test Suite 2.4: Coordinator Crash Recovery (`tests/test-coordinator-recovery.sh`)

**Components Under Test:**
- Coordinator crash detection
- State recovery
- Claim cleanup

**Test Cases:**

1. **test_detect_stale_coordinator_pid**
   - Setup: Stale coordinator.pid file
   - Action: Start coordinator
   - Assert: Stale PID detected

2. **test_cleanup_stale_claims**
   - Setup: Stale claims from previous run
   - Action: Coordinator recovers
   - Assert: Claims cleaned up

3. **test_reset_agent_states**
   - Setup: Agent states from crashed coordinator
   - Action: Coordinator recovers
   - Assert: States reset to unknown

4. **test_verify_system_integrity**
   - Setup: Inconsistent state after crash
   - Action: Coordinator recovers
   - Assert: Integrity verified, inconsistencies logged

**Mock Requirements:**
- Mock stale state files
- Mock work queue

**Validation:**
```bash
./tests/test-coordinator-recovery.sh
# Expected: 4/4 tests pass in <2 minutes
```

---

## Layer 3: System Tests

**Goal:** Test complete end-to-end workflows with real components
**Execution Time:** 5-30 minutes per suite
**Mock Level:** Low (mock only Claude CLI, use real coordination)

### Test Suite 3.1: End-to-End Parallel Workflow (`tests/test-parallel-e2e.sh`)

**System Under Test:** Complete parallel harness

**Test Cases:**

1. **test_single_agent_baseline**
   - Setup: 10 work items, 1 agent
   - Action: Process all work
   - Assert: All completed, measure time

2. **test_three_agent_parallel**
   - Setup: 10 work items, 3 agents
   - Action: Process all work
   - Assert: All completed, time < baseline/2

3. **test_ten_agent_stress**
   - Setup: 100 work items, 10 agents
   - Action: Process all work
   - Assert: All completed, no conflicts

4. **test_mixed_work_priorities**
   - Setup: Work with high/medium/low priority
   - Action: Agents process work
   - Assert: High priority processed first

5. **test_rig_affinity**
   - Setup: Work for different rigs
   - Action: Agents process with affinity enabled
   - Assert: Agents prefer same rig

**Validation:**
```bash
./tests/test-parallel-e2e.sh
# Expected: 5/5 tests pass in <15 minutes
```

---

### Test Suite 3.2: Race Condition Testing (`tests/test-race-conditions.sh`)

**System Under Test:** Work claiming under contention

**Test Cases:**

1. **test_high_contention_claims**
   - Setup: 1 work item, 10 agents
   - Action: All claim simultaneously (1000 iterations)
   - Assert: Exactly 1 success per iteration, no duplicates

2. **test_rapid_claim_release_reclaim**
   - Setup: 1 work item, 2 agents
   - Action: Rapid claim/release cycles
   - Assert: No claim corruption, clean state

3. **test_concurrent_git_operations**
   - Setup: 3 agents, 3 worktrees
   - Action: Simultaneous commits/pushes
   - Assert: No git conflicts, all operations succeed

4. **test_simultaneous_status_updates**
   - Setup: 5 agents updating status
   - Action: All update simultaneously
   - Assert: All updates persisted, no corruption

**Validation:**
```bash
./tests/test-race-conditions.sh
# Expected: 4/4 tests pass in <10 minutes
# Zero race conditions detected
```

---

### Test Suite 3.3: Failure Scenario Testing (`tests/test-failure-scenarios.sh`)

**System Under Test:** System behavior under various failures

**Test Cases:**

1. **test_agent_crash_during_work**
   - Setup: Agent processing work
   - Action: Kill agent mid-work
   - Assert: Work released, replacement spawned, work reclaimed

2. **test_network_failure**
   - Setup: Agent trying to push to remote
   - Action: Simulate network failure
   - Assert: Error handled, work marked failed

3. **test_disk_full**
   - Setup: Agent running
   - Action: Simulate disk full
   - Assert: Agent stops gracefully, notifies overseer

4. **test_out_of_memory**
   - Setup: Agent with high memory usage
   - Action: Trigger OOM
   - Assert: Process killed by ulimit, recovery runs

5. **test_git_conflict**
   - Setup: 2 agents modifying same file
   - Action: Both try to push
   - Assert: One succeeds, one retries

**Validation:**
```bash
./tests/test-failure-scenarios.sh
# Expected: 5/5 tests pass in <20 minutes
```

---

## Layer 4: Stress Tests

**Goal:** Validate system behavior under extreme conditions
**Execution Time:** Hours to days
**Mock Level:** Minimal (real system, controlled load)

### Test Suite 4.1: Load Testing (`tests/test-load.sh`)

**System Under Test:** System under heavy load

**Test Cases:**

1. **test_10_agents_continuous_load**
   - Setup: 1000 work items, 10 agents
   - Action: Process all work
   - Assert: All completed, no crashes, performance acceptable

2. **test_rapid_agent_churn**
   - Setup: 5 agents, kill/respawn randomly
   - Action: Run for 1 hour
   - Assert: System remains stable, work progresses

3. **test_queue_overflow**
   - Setup: Add 500 work items rapidly
   - Action: Agents process work
   - Assert: All work processed, queue stable

**Validation:**
```bash
./tests/test-load.sh
# Expected: 3/3 tests pass in <2 hours
```

---

### Test Suite 4.2: Stability Testing (`tests/test-24hour-stability.sh`)

**System Under Test:** Long-running system stability

**Test Cases:**

1. **test_24_hour_continuous_operation**
   - Setup: 3 agents, continuous work stream
   - Action: Run for 24 hours
   - Assert: No crashes, memory stable, throughput consistent

2. **test_resource_leak_detection**
   - Setup: 3 agents, 24 hours
   - Action: Monitor memory/disk/file descriptors
   - Assert: No leaks detected, resources stable

**Validation:**
```bash
./tests/test-24hour-stability.sh
# Expected: 2/2 tests pass in 24+ hours
# Monitor: memory, disk, file descriptors, throughput
```

---

## Test Infrastructure

### Mock Claude CLI (`tests/mocks/mock-claude.sh`)

**Purpose:** Simulate Claude Code behavior for fast, deterministic testing

**Capabilities:**
- Simulate successful work execution
- Simulate failures (timeout, error, crash)
- Generate stream-JSON events
- Configurable execution time
- Controllable exit codes

**Usage:**
```bash
# Simulate successful work (5 second execution)
./tests/mocks/mock-claude.sh --session-id test-123 --duration 5 --exit-code 0

# Simulate failure
./tests/mocks/mock-claude.sh --session-id test-456 --duration 2 --exit-code 1

# Simulate crash
./tests/mocks/mock-claude.sh --session-id test-789 --crash-after 3
```

---

### Test Library (`tests/test-lib.sh`)

**Purpose:** Shared utilities for all tests

**Functions:**

```bash
# Test lifecycle
setup_test()        # Initialize test environment
teardown_test()     # Cleanup after test
run_test()          # Execute test with logging

# Assertions
assert_equal()      # Assert values equal
assert_file_exists()  # Assert file exists
assert_process_running()  # Assert PID alive
assert_no_race_condition()  # Detect race conditions

# Mocking
mock_work_queue()   # Create mock queue
mock_agent_state()  # Create mock agent state
mock_claude_cli()   # Use mock Claude

# Utilities
wait_for_condition()  # Busy-wait for condition
get_test_timestamp()  # Unique timestamp for test
cleanup_test_state()  # Remove test artifacts
```

**Usage:**
```bash
#!/usr/bin/env bash
source "$(dirname "${BASH_SOURCE[0]}")/test-lib.sh"

test_example() {
  setup_test

  # Test logic
  result=$(some_function)
  assert_equal "$result" "expected_value"

  teardown_test
}

run_test test_example
```

---

## Test Execution

### Running Tests Locally

**Individual Test Suite:**
```bash
# Run specific test suite
./tests/test-atomic-claims.sh

# With verbose output
VERBOSE=true ./tests/test-atomic-claims.sh

# Stop on first failure
FAIL_FAST=true ./tests/test-atomic-claims.sh
```

**All Unit Tests:**
```bash
# Run all unit tests
for test in tests/test-*.sh; do
  echo "Running $test..."
  if ! "$test"; then
    echo "FAILED: $test"
    exit 1
  fi
done
```

**All Tests (Full Suite):**
```bash
# Run complete test suite
./tests/run-all-tests.sh

# Output:
# ═══════════════════════════════════════
#   Phase 3 Test Suite
# ═══════════════════════════════════════
#
# Unit Tests:
#   ✅ test-atomic-claims.sh (6/6)
#   ✅ test-worktree-isolation.sh (6/6)
#   ✅ test-status-aggregation.sh (6/6)
#   ✅ test-health-checks.sh (5/5)
#
# Integration Tests:
#   ✅ test-agent-worker.sh (7/7)
#   ✅ test-coordinator.sh (6/6)
#   ✅ test-failure-recovery.sh (6/6)
#   ✅ test-coordinator-recovery.sh (4/4)
#
# System Tests:
#   ✅ test-parallel-e2e.sh (5/5)
#   ✅ test-race-conditions.sh (4/4)
#   ✅ test-failure-scenarios.sh (5/5)
#
# ═══════════════════════════════════════
# Total: 60/60 tests passed
# Duration: 45 minutes
# Status: ALL TESTS PASSED ✅
```

---

### Continuous Testing

**During Development:**
- Run unit tests after each change (~1 minute)
- Run integration tests before commit (~5 minutes)
- Run system tests before push (~30 minutes)

**Pre-Merge:**
- Full test suite must pass (60/60)
- No shellcheck warnings
- Code review approved

**Pre-Deployment:**
- Full test suite on staging environment
- 24-hour stability test
- Performance benchmarks met

---

## Performance Benchmarks

### Baseline Metrics (1 Agent)

**Throughput:**
- Items/hour: 4
- Average duration: 15 minutes/item
- Success rate: >95%

**Resources:**
- Memory: ~2GB
- CPU: ~50% of 1 core
- Disk: ~500MB

### Target Metrics (3 Agents)

**Throughput:**
- Items/hour: 10 (2.5x baseline)
- Average duration: <15 minutes/item
- Success rate: >90%

**Resources:**
- Memory: ~6GB (2GB × 3)
- CPU: ~150% (1.5 cores)
- Disk: ~2GB (3 worktrees)

**Efficiency:**
- Overhead: <20%
- Claim contention: <10%
- Coordinator CPU: <5%

### Stress Test Metrics (10 Agents)

**Throughput:**
- Items/hour: 25-30 (6-7.5x baseline)
- Success rate: >80%

**Resources:**
- Memory: ~20GB (2GB × 10)
- CPU: ~500% (5 cores)
- Disk: ~6GB (10 worktrees)

**Stability:**
- Uptime: 24+ hours
- Crashes: 0
- Memory leaks: 0

---

## Test Coverage Tracking

### Coverage Metrics

**Functions:**
- Total functions: ~50
- Covered: >45 (90%)
- Critical path coverage: 100%

**Failure Scenarios:**
- Identified: 10
- Tested: 10 (100%)

**Race Conditions:**
- Potential sites: 5 (work claim, status update, worktree ops, git push, coordinator state)
- Tested: 5 (100%)

**Integration Points:**
- Total: 8
- Tested: 8 (100%)

### Coverage Report

```
Component                 Coverage
────────────────────────  ────────
Atomic Claims             100%
Worktree Management        95%
Agent Worker              90%
Coordinator               90%
Health Monitoring         95%
Failure Recovery          100%
Status Aggregation        85%
```

---

## Test Maintenance

### Adding New Tests

1. **Choose Appropriate Layer**
   - Unit test for isolated functions
   - Integration test for component pairs
   - System test for end-to-end workflows

2. **Use Test Library**
   - Source `test-lib.sh`
   - Use standard assertions
   - Follow naming conventions

3. **Document Test**
   - Add to this document
   - Describe what is tested
   - Explain expected behavior

4. **Run Test Locally**
   - Verify passes
   - Check execution time
   - Ensure cleanup works

5. **Add to Test Suite**
   - Include in `run-all-tests.sh`
   - Update coverage metrics

### Test Naming Conventions

**Test Files:**
- `test-<component>.sh` for unit tests
- `test-<feature>-integration.sh` for integration tests
- `test-<scenario>.sh` for system tests

**Test Functions:**
- `test_<what_is_tested>`
- Use underscores, not dashes
- Be descriptive

**Example:**
```bash
# tests/test-atomic-claims.sh
test_parallel_claims_atomic()
test_duplicate_claim_fails()
test_claim_release_reclaim()
```

---

## Success Criteria

### Test Suite Acceptance

**Before Week 1 Completion:**
- ✅ All unit tests passing (100%)
- ✅ Test execution time <5 minutes
- ✅ Mock infrastructure complete

**Before Week 2 Completion:**
- ✅ All integration tests passing (100%)
- ✅ Test execution time <15 minutes
- ✅ Real components integrated

**Before Week 3 Completion:**
- ✅ All system tests passing (100%)
- ✅ Zero race conditions detected
- ✅ All failure scenarios tested

**Before Production Deployment:**
- ✅ All tests passing (60/60)
- ✅ 24-hour stability test passed
- ✅ Performance benchmarks met
- ✅ No critical bugs

---

## Appendix: Test Scenarios

### Complete Test Matrix

| Scenario | Unit | Integration | System | Stress |
|----------|------|-------------|--------|--------|
| Single claim | ✅ | ✅ | ✅ | - |
| Parallel claims | ✅ | ✅ | ✅ | ✅ |
| Claim release | ✅ | ✅ | ✅ | - |
| Stale work | ✅ | ✅ | ✅ | - |
| Worktree creation | ✅ | ✅ | ✅ | - |
| Commit isolation | ✅ | ✅ | ✅ | ✅ |
| Agent spawning | - | ✅ | ✅ | ✅ |
| Heartbeat updates | ✅ | ✅ | ✅ | ✅ |
| Agent crash | - | ✅ | ✅ | ✅ |
| Work release on crash | ✅ | ✅ | ✅ | - |
| Coordinator crash | - | ✅ | ✅ | - |
| High contention | - | - | ✅ | ✅ |
| 24-hour stability | - | - | - | ✅ |

**Total Scenarios:** 13
**Total Test Cases:** 60+

---

**Document Status:** Final
**Next Steps:** Implement test suites during Weeks 1-3
**Owner:** System Architect
**Created:** 2026-01-28
**Version:** 1.0
