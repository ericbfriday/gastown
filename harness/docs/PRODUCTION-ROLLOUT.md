# Production Rollout Guide: Claude Automation Harness Phase 2

**Version:** 1.0
**Date:** 2026-01-27
**Phase:** 2 (Claude Code Integration)
**Status:** Ready for Rollout

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Pre-Deployment Checklist](#pre-deployment-checklist)
4. [Installation Steps](#installation-steps)
5. [Configuration](#configuration)
6. [Testing in Staging](#testing-in-staging)
7. [Production Deployment](#production-deployment)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Troubleshooting](#troubleshooting)
10. [Rollback Procedures](#rollback-procedures)
11. [Operational Runbook](#operational-runbook)

---

## Overview

This guide provides step-by-step instructions for rolling out Phase 2 of the Claude Automation Harness to production. Phase 2 introduces actual Claude Code integration with complete lifecycle management.

### What's Being Deployed

**Core Capabilities:**
- Production-ready agent spawning
- Real-time session monitoring
- Comprehensive state tracking
- Robust error handling
- Metrics collection
- Event analysis tools

**Changed Components:**
- `harness/loop.sh` - Complete spawn implementation
- `harness/scripts/parse-session-events.sh` - New analysis tool
- Session state schema - Enhanced tracking
- Monitoring infrastructure - Real-time visibility

**System Requirements:**
- Claude Code CLI installed and authenticated
- Gastown (gt) and Beads (bd) CLIs available
- Git repository initialized
- Sufficient disk space (~2GB for logs)
- macOS or Linux environment

---

## Prerequisites

### Required Software

**✅ Verify before proceeding:**

```bash
# Check Claude Code CLI
claude --version
# Expected: Claude Code CLI v1.x.x

# Verify authentication
claude auth status
# Expected: Authenticated (subscription or API key)

# Check Gastown
gt --version
# Expected: gt v0.4.0 or higher

# Check Beads
bd --version
# Expected: bd v0.47.1 or higher

# Check jq
jq --version
# Expected: jq-1.6 or higher

# Check git
git --version
# Expected: git version 2.x
```

### System Access

- [ ] SSH access to deployment host
- [ ] Write permissions to `~/gt/harness/`
- [ ] Access to git repository
- [ ] Claude Code subscription or API key configured
- [ ] Beads actor credentials set up

### Network Requirements

- [ ] Internet access for Claude API
- [ ] Git remote access (push/pull)
- [ ] No corporate proxy blocking Claude API

### Resource Requirements

**Minimum:**
- 8GB RAM
- 20GB disk space
- 2 CPU cores
- Stable internet connection

**Recommended:**
- 16GB RAM
- 50GB disk space
- 4 CPU cores
- High-bandwidth internet (for API calls)

---

## Pre-Deployment Checklist

### Code Preparation

- [ ] Pull latest code from `feature/claude-automation-harness` branch
- [ ] Review PHASE-2-SUMMARY.md for changes
- [ ] Verify all files present (use git status)
- [ ] Check permissions on scripts (should be executable)

**Verification:**
```bash
cd ~/gt/harness

# Check branch
git branch --show-current
# Expected: feature/claude-automation-harness

# Verify commit
git log --oneline -1
# Expected: Latest Phase 2 commit

# Check script permissions
ls -l loop.sh scripts/*.sh
# Expected: -rwxr-xr-x (executable)
```

### Environment Configuration

- [ ] Review `config.yaml` settings
- [ ] Set required environment variables
- [ ] Verify working directory exists
- [ ] Check bootstrap template present

**Environment Setup:**
```bash
# Set Beads actor
export BD_ACTOR="harness-agent"

# Set rig (optional)
export BD_RIG="aardwolf_snd"

# Set work directory
export GT_ROOT="$HOME/gt"

# Verify bootstrap template
ls -la prompts/bootstrap.md
# Expected: File exists
```

### Backup

- [ ] Backup current harness state
- [ ] Backup configuration files
- [ ] Tag current commit for rollback
- [ ] Document current system state

**Backup Commands:**
```bash
# Create backup directory
mkdir -p ~/gt-backup/harness-$(date +%Y%m%d)

# Backup state
cp -r state ~/gt-backup/harness-$(date +%Y%m%d)/

# Backup config
cp config.yaml ~/gt-backup/harness-$(date +%Y%m%d)/

# Tag current state
git tag pre-phase2-rollout-$(date +%Y%m%d)
git push origin pre-phase2-rollout-$(date +%Y%m%d)
```

### Testing Prerequisites

- [ ] Create test work queue
- [ ] Prepare test scenarios
- [ ] Set up monitoring terminals
- [ ] Document baseline metrics

**Test Setup:**
```bash
# Create test queue
cat > state/queue.json <<'EOF'
[
  {
    "id": "test-001",
    "type": "test",
    "title": "Phase 2 validation test",
    "priority": "high",
    "rig": "aardwolf_snd"
  }
]
EOF

# Verify queue
./scripts/manage-queue.sh show
```

---

## Installation Steps

### Step 1: Update Code

```bash
# Navigate to harness directory
cd ~/gt/harness

# Stash any local changes
git stash

# Fetch latest code
git fetch origin feature/claude-automation-harness

# Checkout and pull
git checkout feature/claude-automation-harness
git pull origin feature/claude-automation-harness

# Verify correct commit
git log --oneline -1
```

### Step 2: Verify File Integrity

```bash
# Check all required files present
./scripts/verify-installation.sh

# Or manually verify
ls -l loop.sh
ls -l scripts/parse-session-events.sh
ls -l prompts/bootstrap.md
ls -l config.yaml
```

### Step 3: Set Permissions

```bash
# Make scripts executable
chmod +x loop.sh
chmod +x scripts/*.sh

# Verify
ls -l *.sh scripts/*.sh | grep -E "^-rwx"
```

### Step 4: Initialize Directories

```bash
# Create state directories
mkdir -p state/sessions
mkdir -p docs/sessions
mkdir -p docs/research

# Verify
ls -ld state docs state/sessions docs/sessions docs/research
```

### Step 5: Validate Configuration

```bash
# Check config syntax
yq eval '.' config.yaml > /dev/null
echo "Config valid: $?"

# Review settings
cat config.yaml
```

### Step 6: Test Installation

```bash
# Source loop.sh without running
bash -n loop.sh
# Expected: No output (syntax OK)

# Test help
./scripts/parse-session-events.sh --help
# Expected: Usage information

# Verify dependencies
command -v gt bd jq git claude
# Expected: Paths to all commands
```

---

## Configuration

### Core Configuration

**File:** `harness/config.yaml`

**Review and adjust:**
```yaml
harness:
  # Loop control
  max_iterations: 0                    # 0 = infinite (production)
  iteration_delay: 5                   # seconds between iterations
  interrupt_check_interval: 30         # seconds between checks

  # Agent spawning
  agent_type: claude-sonnet            # Model to use
  session_timeout: 3600                # 1 hour max per session

  # Monitoring
  stall_threshold: 300                 # 5 minutes no progress = stalled
  max_consecutive_failures: 5          # Failures before interrupt

  # Interrupts
  interrupts:
    quality_gate_failure: true         # Interrupt on test failures
    blocked_work: true                 # Interrupt when blocked
    approval_required: true            # Interrupt needing approval
    session_timeout: true              # Interrupt on timeout
```

### Environment Variables

**Set before starting:**
```bash
# Required
export BD_ACTOR="harness-agent"
export GT_ROOT="$HOME/gt"

# Optional
export BD_RIG="aardwolf_snd"
export SESSION_TIMEOUT=3600
export STALL_THRESHOLD=300
export MAX_CONSECUTIVE_FAILURES=5
export INTERRUPT_CHECK_INTERVAL=30
```

### Bootstrap Template

**File:** `prompts/bootstrap.md`

**Variables substituted:**
- `{{SESSION_ID}}` - Unique session identifier
- `{{ITERATION}}` - Current iteration number
- `{{WORK_ITEM}}` - Assigned work item ID
- `{{RIG}}` - Current rig name

**Customize for your workflow:**
```markdown
# Claude Harness Agent Bootstrap

Session: {{SESSION_ID}}
Iteration: {{ITERATION}}
Work Item: {{WORK_ITEM}}
Rig: {{RIG}}

## Your Role

You are an autonomous agent in the Claude Automation Harness...

[Customize agent instructions here]
```

---

## Testing in Staging

### Stage 1: Smoke Test (5 minutes)

**Objective:** Verify basic functionality

```bash
# Start harness with single iteration
MAX_ITERATIONS=1 ./loop.sh

# Expected output:
# - Session spawned
# - Agent runs
# - Session completes
# - Logs created
```

**Validation:**
```bash
# Check session files created
ls -l state/current-session.json
ls -l docs/sessions/ses_*.log

# Verify no errors
./scripts/parse-session-events.sh latest
```

### Stage 2: Monitoring Test (10 minutes)

**Objective:** Verify real-time monitoring

```bash
# Terminal 1: Start harness
./loop.sh

# Terminal 2: Watch status
watch -n 5 ./scripts/report-status.sh

# Terminal 3: Monitor events
SESSION_ID=$(jq -r '.session_id' state/current-session.json)
./scripts/parse-session-events.sh watch $SESSION_ID
```

**Validation:**
- [ ] Events appear in real-time
- [ ] Status updates correctly
- [ ] Tool calls are logged
- [ ] Progress indicators update

### Stage 3: Error Handling Test (15 minutes)

**Objective:** Verify error scenarios

**Test 1: Spawn Failure**
```bash
# Temporarily make claude unavailable
mv $(which claude) $(which claude).bak

# Start harness
MAX_ITERATIONS=1 ./loop.sh
# Expected: Graceful failure, error logged

# Restore claude
mv $(which claude).bak $(which claude)
```

**Test 2: Session Timeout**
```bash
# Set very short timeout
SESSION_TIMEOUT=10 MAX_ITERATIONS=1 ./loop.sh
# Expected: Timeout detected, agent killed, state updated
```

**Test 3: Work Queue Empty**
```bash
# Empty queue
echo '[]' > state/queue.json

# Start harness
MAX_ITERATIONS=1 ./loop.sh
# Expected: No work message, graceful retry
```

**Validation:**
- [ ] Errors logged properly
- [ ] Recovery mechanisms work
- [ ] State preserved correctly
- [ ] No process leaks

### Stage 4: Full Iteration Test (30 minutes)

**Objective:** Complete workflow validation

```bash
# Set up real work item
./scripts/manage-queue.sh add '{
  "id": "stage-test-001",
  "title": "Complete validation test",
  "type": "feature",
  "priority": "high"
}'

# Start harness with 3 iterations
MAX_ITERATIONS=3 ./loop.sh

# Monitor progress
while [[ -f state/current-session.json ]]; do
  ./scripts/report-status.sh
  sleep 30
done
```

**Validation:**
- [ ] All 3 iterations complete
- [ ] Work items processed
- [ ] Logs captured completely
- [ ] Metrics accurate
- [ ] No errors or crashes

### Stage 5: Interrupt Test (15 minutes)

**Objective:** Verify interrupt mechanism

```bash
# Start harness
./loop.sh &
HARNESS_PID=$!

# Wait for session to start
sleep 30

# Trigger interrupt
echo "Testing interrupt mechanism" > state/interrupt-request.txt

# Verify harness pauses
# Expected: Harness detects, preserves context, waits

# Check preserved context
ls docs/sessions/*-context.json

# Resume
rm state/interrupt-request.txt

# Verify harness continues
wait $HARNESS_PID
```

**Validation:**
- [ ] Interrupt detected promptly
- [ ] Context preserved
- [ ] Harness pauses
- [ ] Resume works correctly

### Stage 6: Metrics Test (10 minutes)

**Objective:** Verify metrics collection

```bash
# Run one iteration
MAX_ITERATIONS=1 ./loop.sh

# Get session ID
SESSION_ID=$(ls -t docs/sessions/ses_*.json | head -1 | xargs basename .json)

# Check metrics
./scripts/parse-session-events.sh metrics $SESSION_ID

# Verify metrics file
cat state/sessions/$SESSION_ID/metrics.json | jq
```

**Validation:**
- [ ] Metrics collected
- [ ] Token counts accurate
- [ ] Tool usage tracked
- [ ] Duration calculated
- [ ] JSON format valid

### Staging Success Criteria

**✅ All stages must pass before production:**

- [ ] Smoke test passes
- [ ] Monitoring works in real-time
- [ ] Error handling recovers gracefully
- [ ] Full iterations complete successfully
- [ ] Interrupt mechanism functions
- [ ] Metrics collection accurate
- [ ] No memory leaks
- [ ] No process zombies
- [ ] Logs are complete
- [ ] State tracking works

---

## Production Deployment

### Pre-Deployment

**Final Checks:**
```bash
# Verify staging results
cat staging-test-results.txt

# Confirm all tests passed
[[ $? -eq 0 ]] || { echo "Staging failed"; exit 1; }

# Review configuration
cat config.yaml

# Check resource availability
df -h ~/gt/harness
free -h
```

### Deployment Steps

**Step 1: Schedule Maintenance Window**
- Recommended: Off-peak hours
- Duration: 1-2 hours
- Notify stakeholders

**Step 2: Stop Existing Harness (if running)**
```bash
# Find harness process
ps aux | grep loop.sh

# Stop gracefully
pkill -TERM -f "loop.sh"

# Wait for shutdown (up to 2 minutes)
sleep 120

# Verify stopped
ps aux | grep loop.sh | grep -v grep
```

**Step 3: Deploy New Code**
```bash
# Pull latest production-ready code
git fetch origin feature/claude-automation-harness
git checkout feature/claude-automation-harness
git pull

# Verify correct version
git log --oneline -1 | grep "feat: implement Claude automation harness"

# Check file integrity
git status
# Expected: On branch, up to date
```

**Step 4: Configuration Validation**
```bash
# Review config
cat config.yaml

# Validate syntax
yq eval '.' config.yaml

# Set environment
export BD_ACTOR="harness-agent"
export GT_ROOT="$HOME/gt"
```

**Step 5: Initialize Production State**
```bash
# Clean old state (if desired)
mv state state.old-$(date +%Y%m%d)
mkdir -p state/sessions

# Initialize queue
./scripts/manage-queue.sh check

# Verify
./scripts/manage-queue.sh show
```

**Step 6: Start Harness**
```bash
# Start in background with logging
nohup ./loop.sh > harness.out 2>&1 &

# Capture PID
echo $! > harness.pid

# Verify started
ps -p $(cat harness.pid)
```

**Step 7: Verify Operation**
```bash
# Wait for first session
sleep 60

# Check status
./scripts/report-status.sh

# Verify session running
cat state/current-session.json | jq

# Monitor logs
tail -f state/iteration.log
```

### Post-Deployment Validation

**Immediate (First 15 minutes):**
- [ ] Harness process running
- [ ] First session spawned successfully
- [ ] Monitoring shows activity
- [ ] No errors in logs
- [ ] State files being created

**Short-term (First hour):**
- [ ] Multiple iterations complete
- [ ] Work items being processed
- [ ] Metrics being collected
- [ ] No memory growth
- [ ] No performance degradation

**Medium-term (First 24 hours):**
- [ ] Continuous operation without crashes
- [ ] No unexpected interrupts
- [ ] Resource usage stable
- [ ] All work categories handled
- [ ] Error recovery working

---

## Monitoring and Observability

### Real-Time Monitoring

**Terminal 1: Status Dashboard**
```bash
watch -n 10 ./scripts/report-status.sh
```

**Terminal 2: Event Stream**
```bash
# Get current session
SESSION_ID=$(jq -r '.session_id' state/current-session.json)

# Watch events
./scripts/parse-session-events.sh watch $SESSION_ID
```

**Terminal 3: System Resources**
```bash
# Monitor CPU/memory
top -pid $(cat harness.pid)

# Or htop
htop -p $(cat harness.pid)
```

**Terminal 4: Logs**
```bash
tail -f state/iteration.log
```

### Key Metrics to Track

**Session Metrics:**
- Sessions per hour
- Average session duration
- Success rate
- Failure rate
- Interrupt rate

**Resource Metrics:**
- CPU usage
- Memory usage
- Disk space
- Network bandwidth (API calls)

**API Metrics:**
- Tokens per session
- Cost per session
- API calls per hour
- Rate limit proximity

### Health Checks

**Manual Health Check:**
```bash
#!/bin/bash
# health-check.sh

echo "=== Harness Health Check ==="

# Check process
if ps -p $(cat harness.pid 2>/dev/null) > /dev/null 2>&1; then
  echo "✓ Process running"
else
  echo "✗ Process not running"
  exit 1
fi

# Check recent activity
LAST_UPDATE=$(stat -f %m state/current-session.json 2>/dev/null)
NOW=$(date +%s)
AGE=$((NOW - LAST_UPDATE))

if [[ $AGE -lt 300 ]]; then
  echo "✓ Recent activity (${AGE}s ago)"
else
  echo "✗ No recent activity (${AGE}s ago)"
  exit 1
fi

# Check for errors
ERROR_COUNT=$(tail -100 state/iteration.log | grep -c ERROR)
if [[ $ERROR_COUNT -lt 5 ]]; then
  echo "✓ Error rate acceptable ($ERROR_COUNT recent errors)"
else
  echo "⚠ High error rate ($ERROR_COUNT recent errors)"
fi

echo "✓ Health check passed"
```

**Automated Monitoring (cron):**
```bash
# Add to crontab
crontab -e

# Run health check every 5 minutes
*/5 * * * * cd ~/gt/harness && ./health-check.sh || echo "HARNESS HEALTH CHECK FAILED" | mail -s "Harness Alert" you@example.com
```

### Alert Conditions

**Critical (Immediate attention):**
- Harness process died
- Multiple consecutive spawn failures
- Disk space below 10%
- Memory usage above 90%

**Warning (Monitor closely):**
- Session timeout occurred
- Error rate elevated
- Interrupt requested
- Queue growing unbounded

**Info (Normal operations):**
- Session completed
- Work item processed
- Metrics collected

---

## Troubleshooting

### Common Issues

#### Issue 1: Harness Won't Start

**Symptoms:**
- Process exits immediately
- Error in logs
- No session created

**Diagnosis:**
```bash
# Check for errors
tail state/iteration.log

# Verify dependencies
command -v claude gt bd jq

# Check permissions
ls -l loop.sh

# Test syntax
bash -n loop.sh
```

**Solutions:**
- Install missing dependencies
- Fix file permissions (`chmod +x loop.sh`)
- Review error message in logs
- Check configuration syntax

#### Issue 2: Agent Won't Spawn

**Symptoms:**
- Session status stuck on "spawning"
- No Claude process found
- Spawn failures in log

**Diagnosis:**
```bash
# Check Claude CLI
claude --version
claude auth status

# Check work queue
./scripts/manage-queue.sh show

# Check bootstrap
cat prompts/bootstrap.md

# Review logs
tail docs/sessions/ses_*.err
```

**Solutions:**
- Authenticate Claude CLI (`claude auth login`)
- Add work to queue
- Fix bootstrap template
- Check API rate limits

#### Issue 3: Session Stalls

**Symptoms:**
- No progress for >5 minutes
- Heartbeat not updating
- Process running but idle

**Diagnosis:**
```bash
# Check session state
cat state/current-session.json | jq '.heartbeat'

# Check process
ps -p $(jq -r '.pid' state/current-session.json)

# Check stall detection
grep "stalled" state/iteration.log
```

**Solutions:**
- Wait for stall detection to kill agent
- Manually kill if needed: `kill $(jq -r '.pid' state/current-session.json)`
- Adjust stall threshold if false positives

#### Issue 4: High Error Rate

**Symptoms:**
- Multiple errors in logs
- Spawn failures
- Agent crashes

**Diagnosis:**
```bash
# Count errors
grep -c ERROR state/iteration.log

# Analyze errors
grep ERROR state/iteration.log | tail -20

# Check system resources
free -h
df -h
```

**Solutions:**
- Check system resources (RAM, disk)
- Review error messages
- Adjust configuration (timeouts, limits)
- Check API status/rate limits

#### Issue 5: Interrupt Not Resolving

**Symptoms:**
- Harness paused
- Interrupt file exists
- No one knows why

**Diagnosis:**
```bash
# Check interrupt reason
cat state/interrupt-request.txt

# Review context
cat docs/sessions/*-context.json | tail -1 | jq

# Check recent activity
tail -50 state/iteration.log
```

**Solutions:**
- Review interrupt reason
- Check preserved context
- Resolve underlying issue
- Remove interrupt file: `rm state/interrupt-request.txt`

### Debug Mode

**Enable verbose logging:**
```bash
# Stop harness
pkill -TERM -f "loop.sh"

# Start with debugging
DEBUG=true VERBOSE=true ./loop.sh
```

**Enable Claude verbose:**
```bash
# Modify spawn_agent() to add --debug flag
# Or set in environment
export CLAUDE_DEBUG=true
```

### Emergency Procedures

**Force Stop:**
```bash
# Kill harness
pkill -KILL -f "loop.sh"

# Kill any orphan Claude processes
pkill -KILL claude

# Clean up PIDs
rm state/ses_*.pid
```

**Reset State:**
```bash
# Backup current state
mv state state.emergency-$(date +%Y%m%d_%H%M%S)

# Reinitialize
mkdir -p state/sessions
echo '[]' > state/queue.json
```

**Recover Session:**
```bash
# Find latest session
SESSION=$(ls -t docs/sessions/ses_*.json | head -1)

# Review state
cat $SESSION | jq

# Manually mark as complete if needed
jq '.status = "completed"' $SESSION > state/current-session.json
```

---

## Rollback Procedures

### When to Rollback

**Rollback if:**
- Multiple critical failures in first hour
- Data loss or corruption
- Performance degradation >50%
- Unable to resolve issues quickly
- Safety concerns

**Don't rollback if:**
- Minor errors (recoverable)
- Single failure (retry succeeds)
- Configuration issues (fixable)
- Expected behavior (just different)

### Rollback Steps

**Step 1: Stop Current Harness**
```bash
# Graceful stop
pkill -TERM -f "loop.sh"

# Wait for shutdown
sleep 60

# Force if needed
pkill -KILL -f "loop.sh"

# Verify stopped
ps aux | grep loop.sh | grep -v grep
```

**Step 2: Preserve State for Analysis**
```bash
# Create forensics directory
mkdir -p ~/rollback-forensics/$(date +%Y%m%d_%H%M%S)

# Copy state
cp -r state ~/rollback-forensics/$(date +%Y%m%d_%H%M%S)/

# Copy logs
cp -r docs/sessions ~/rollback-forensics/$(date +%Y%m%d_%H%M%S)/

# Copy config
cp config.yaml ~/rollback-forensics/$(date +%Y%m%d_%H%M%S)/
```

**Step 3: Restore Previous Version**
```bash
# Find rollback tag
git tag | grep pre-phase2-rollout

# Checkout previous version
git checkout pre-phase2-rollout-YYYYMMDD

# Verify
git log --oneline -1
```

**Step 4: Restore State (if applicable)**
```bash
# Restore backed up state
cp -r ~/gt-backup/harness-YYYYMMDD/state .
cp ~/gt-backup/harness-YYYYMMDD/config.yaml .
```

**Step 5: Restart Previous Version**
```bash
# Start old harness
nohup ./loop.sh > harness.out 2>&1 &

# Verify
ps aux | grep loop.sh
```

**Step 6: Verify Operation**
```bash
# Check status
./scripts/report-status.sh

# Monitor logs
tail -f state/iteration.log

# Watch for issues
```

**Step 7: Notify Stakeholders**
```
Subject: Harness Phase 2 Rollback

The Phase 2 deployment has been rolled back to the previous version due to [REASON].

Status: [STABLE/INVESTIGATING]
Next Steps: [ACTION PLAN]
ETA for Resolution: [TIMEFRAME]
```

### Post-Rollback

**Analysis:**
- Review forensic logs
- Identify root cause
- Document failure
- Update rollout plan
- Test fix in staging

**Re-Deployment:**
- Fix identified issues
- Re-test thoroughly
- Schedule new deployment
- Monitor closely

---

## Operational Runbook

### Daily Operations

**Morning Checklist:**
```bash
# Check harness health
./health-check.sh

# Review overnight activity
./scripts/report-status.sh

# Check error rate
grep ERROR state/iteration.log | tail -50

# Check disk space
df -h ~/gt/harness

# Review completed work
grep "SUCCESS" state/iteration.log | grep -c "Session completed"
```

**Afternoon Checklist:**
```bash
# Check session count
ls docs/sessions/*.json | wc -l

# Review metrics
./scripts/parse-session-events.sh latest

# Check for interrupts
ls state/interrupt-request.txt 2>/dev/null && echo "Interrupt active"

# Verify work queue
./scripts/manage-queue.sh show
```

**Evening Checklist:**
```bash
# Calculate daily throughput
grep "completed successfully" state/iteration.log | grep $(date +%Y-%m-%d) | wc -l

# Archive old logs (optional)
find docs/sessions -name "*.log" -mtime +7 -exec gzip {} \;

# Check resource trends
du -sh state docs

# Prepare for overnight
echo "Overnight monitoring: $(date)"
```

### Weekly Operations

**Maintenance Tasks:**
```bash
# Analyze trends
./scripts/analyze-weekly-metrics.sh

# Clean up old sessions (>30 days)
find docs/sessions -name "*.json" -mtime +30 -delete

# Review configuration
cat config.yaml

# Update documentation if needed

# Test rollback procedure
git tag | grep pre-phase2
```

### Monthly Operations

**Health Review:**
- Analyze success/failure rates
- Review interrupt frequency
- Assess resource utilization
- Optimize configuration
- Update runbooks
- Train operators

**Performance Tuning:**
- Adjust timeouts based on data
- Optimize bootstrap prompt
- Review tool allowlist
- Update max turns/budget
- Fine-tune stall threshold

### Emergency Contacts

**Escalation Path:**
1. Operator (immediate response)
2. Tech Lead (if operator can't resolve)
3. System Architect (for critical issues)
4. Eric Friday (for system-wide problems)

**Contact Methods:**
- Slack: #harness-ops
- Email: harness-team@example.com
- On-call: [Phone number]

---

## Success Criteria

### Production Rollout Success

**✅ Deployment successful if:**

- [ ] Harness runs for 24 hours without crashes
- [ ] >90% of sessions complete successfully
- [ ] <5% interrupt rate
- [ ] No data loss or corruption
- [ ] Monitoring provides visibility
- [ ] Error recovery works as designed
- [ ] Performance meets baseline
- [ ] Resource usage acceptable
- [ ] Operator can troubleshoot effectively

### Ongoing Success Metrics

**Track over time:**
- Sessions per day
- Success rate (%)
- Average session duration
- API cost per session
- Interrupts per day
- Error rate
- Operator interventions

**Target Values:**
- Success rate: >80%
- Interrupts: <5 per day
- Operator interventions: <1 per day
- Uptime: >99%

---

## Appendix

### Useful Commands Reference

**Status:**
```bash
./scripts/report-status.sh                    # Quick status
./scripts/parse-session-events.sh latest     # Latest session
./scripts/parse-session-events.sh list       # All sessions
```

**Monitoring:**
```bash
watch -n 10 ./scripts/report-status.sh       # Watch status
tail -f state/iteration.log                   # Watch logs
./scripts/parse-session-events.sh watch $SID # Watch session
```

**Analysis:**
```bash
./scripts/parse-session-events.sh metrics $SID    # Session metrics
./scripts/parse-session-events.sh tools $SID      # Tool usage
./scripts/parse-session-events.sh errors $SID     # Errors
./scripts/parse-session-events.sh timeline $SID   # Timeline
```

**Management:**
```bash
pkill -TERM -f "loop.sh"                     # Stop harness
nohup ./loop.sh > harness.out 2>&1 &        # Start harness
ps aux | grep loop.sh                        # Check process
rm state/interrupt-request.txt              # Clear interrupt
```

### Log Locations

```
state/iteration.log                   # Main harness log
docs/sessions/ses_<id>.log           # Session output (stream-JSON)
docs/sessions/ses_<id>.err           # Session errors
docs/sessions/ses_<id>.json          # Session state
~/.claude/transcripts/ses_<id>.jsonl # Full transcript
state/sessions/<id>/metrics.json     # Session metrics
```

### Configuration Files

```
config.yaml                          # Main configuration
prompts/bootstrap.md                 # Bootstrap template
state/queue.json                     # Work queue
state/current-session.json           # Active session
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-27
**Maintainer:** System Operations Team
**Review Schedule:** Monthly
