# Getting Started with Claude Automation Harness

**Quick Start Guide** - Get the harness running in 5 minutes

## What is This?

The Claude Automation Harness is a continuous loop system that spawns Claude Code agents to work on tasks automatically. Think of it as a "Ralph Wiggum loop" - agents continuously work on your Gastown projects, building context from docs, doing work, and preserving knowledge for the next iteration.

## Prerequisites

Make sure you have:

```bash
# Check required commands
which gt    # Gastown
which bd    # Beads
which jq    # JSON processor
which git   # Git
```

All should return a path. If any are missing, install them first.

## 5-Minute Quick Start

### 1. Navigate to Harness

```bash
cd ~/gt/harness
```

### 2. Check Configuration (Optional)

```bash
# View config
cat config.yaml

# Key settings:
# - max_iterations: 0 (infinite loop)
# - iteration_delay: 5 (seconds between iterations)
# - agent_type: claude-sonnet
```

You can run with defaults or edit `config.yaml` to customize.

### 3. Test Status (Verify Setup)

```bash
./scripts/report-status.sh
```

Should show:
- No active session (expected - not running yet)
- Work queue status
- Active rigs
- No interrupts

### 4. Start the Harness (Test Mode)

For your first run, limit iterations:

```bash
# Run 3 iterations then stop
MAX_ITERATIONS=3 ./loop.sh
```

Watch the output - you'll see:
- Initialization
- Iteration starts
- Work queue checks
- Status updates

**Note:** Agent spawning is currently a placeholder. The loop will cycle but won't actually spawn Claude Code sessions yet. This is Phase 1 (infrastructure). Phase 2 will add actual agent spawning.

### 5. Check Status in Another Terminal

While the harness runs, open a new terminal:

```bash
cd ~/gt/harness

# Watch status
watch -n 5 ./scripts/report-status.sh

# Or tail the log
tail -f state/iteration.log
```

### 6. Stop the Harness

In the harness terminal:
- Press `Ctrl+C`
- Harness shuts down gracefully
- Context is preserved

## What Just Happened?

You ran the harness loop through 3 iterations:

1. **Initialization** - Created state directory, checked dependencies
2. **Iteration Loop** - Checked for work, attempted to spawn agent
3. **Graceful Shutdown** - Preserved state, cleaned up

Check what was created:

```bash
# View iteration log
cat state/iteration.log

# Check state files
ls -la state/
```

## Next Steps

### Understand the System

Read the documentation:

```bash
# Quick overview
cat README.md

# Full workflow
cat ../docs/CLAUDE-HARNESS-WORKFLOW.md

# Implementation roadmap
cat ROADMAP.md
```

### Run Continuously (When Ready)

Once Phase 2 is complete and agent spawning works:

```bash
# Run indefinitely
./loop.sh

# Or run in background
nohup ./loop.sh > loop.out 2>&1 &

# Check background process
ps aux | grep loop.sh
```

### Monitor

```bash
# Quick status
./scripts/report-status.sh

# Detailed status
./scripts/report-status.sh --detailed

# Watch continuously
watch -n 10 ./scripts/report-status.sh
```

### Manage Work Queue

```bash
# Check queue
./scripts/manage-queue.sh check

# Show queue contents
./scripts/manage-queue.sh show

# Manually add work (JSON format)
echo '{"id":"gt-123","title":"Test task","priority":5}' | \
  ./scripts/manage-queue.sh add "$(cat)"
```

### Test Interrupt Mechanism

```bash
# Trigger an interrupt
echo "Testing interrupt system" > state/interrupt-request.txt

# Start harness
./loop.sh

# Harness will detect interrupt, preserve context, and wait

# In another terminal, check preserved context
ls -la docs/sessions/

# Resume by removing interrupt
rm state/interrupt-request.txt

# Harness continues automatically
```

## Common Workflows

### Test a Single Iteration

```bash
MAX_ITERATIONS=1 ./loop.sh
```

Good for testing changes without running continuously.

### Run With Faster Cycle

```bash
ITERATION_DELAY=2 ./loop.sh
```

Speeds up iteration for testing.

### Debug Mode

```bash
# Run with verbose logging
set -x
./loop.sh
```

Or edit `loop.sh` and add `set -x` near the top.

### Check for Problems

```bash
# View errors in log
grep ERROR state/iteration.log

# Check recent sessions
ls -lt docs/sessions/ | head -10

# Verify queue is working
./scripts/manage-queue.sh check
```

## Customization

### Edit Configuration

```bash
# Edit config
vi config.yaml

# Key sections:
# - harness.max_iterations
# - harness.iteration_delay
# - harness.interrupts (what triggers interrupts)
# - quality_gates (tests to run)
```

### Customize Bootstrap Prompt

```bash
# Edit agent bootstrap prompt
vi prompts/bootstrap.md

# This is what agents see when they start
# Customize for your workflow
```

### Add Quality Gates

In `config.yaml`:

```yaml
quality_gates:
  post_work:
    - name: "Custom check"
      command: "./my-custom-check.sh"
      required: true
      on_failure: "interrupt"
```

### Change Notification Recipients

In `config.yaml`:

```yaml
notifications:
  notify_on_interrupt: your-email@example.com
  notify_on_error: your-email@example.com
```

## Troubleshooting

### "Command not found: gt"

```bash
# Add to PATH
export PATH="$HOME/go/bin:$PATH"

# Or install Gastown
# (see Gastown documentation)
```

### "Command not found: bd"

```bash
# Add to PATH
export PATH="$HOME/.local/bin:$PATH"

# Or install Beads
# (see Beads documentation)
```

### "Permission denied" on scripts

```bash
# Make scripts executable
chmod +x loop.sh scripts/*.sh
```

### Loop exits immediately

```bash
# Check for errors
cat state/iteration.log

# Common issues:
# - No work available (expected if queue empty)
# - Missing dependencies
# - Configuration errors
```

### Can't stop the harness

```bash
# Find process
ps aux | grep loop.sh

# Kill it
pkill -f loop.sh

# Or with PID
kill <pid>
```

## FAQ

**Q: Why doesn't the harness spawn agents yet?**
A: Phase 1 (infrastructure) is complete. Phase 2 (Claude Code integration) is next. See ROADMAP.md.

**Q: Can I run this in production?**
A: Not yet. Wait for Phase 2 completion and testing.

**Q: How do I add work to the queue?**
A: Work is automatically pulled from `bd ready` and `gt ready`. Or manually use `./scripts/manage-queue.sh add`.

**Q: What happens if the harness crashes?**
A: Restart it. State is preserved in `state/` directory. Context is saved in `docs/sessions/`.

**Q: How do I customize what agents do?**
A: Edit `prompts/bootstrap.md` for agent instructions and `config.yaml` for behavior.

**Q: Where are the logs?**
A: `state/iteration.log` for loop activity, `docs/sessions/` for individual session logs.

## Getting Help

- **Documentation:** See README.md and docs/
- **Roadmap:** See ROADMAP.md for implementation status
- **Issues:** File via beads or contact overseer
- **Contact:** Eric Friday <ericfriday@gmail.com>

## What's Next?

1. **Read Documentation** - Understand the system fully
2. **Review Roadmap** - See what's coming in Phase 2
3. **Test Locally** - Run iterations and observe
4. **Customize** - Adjust for your workflow
5. **Wait for Phase 2** - Actual agent spawning

**Welcome to the Claude Automation Harness!** 🚂

---

**Version:** 1.0 (Phase 1)
**Updated:** 2026-01-27
**Status:** Foundation Complete, Agent Spawning Pending
