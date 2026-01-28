# Claude Code CLI Research
**Date**: 2026-01-27
**Purpose**: Comprehensive analysis of Claude Code CLI capabilities for automated spawning and orchestration

## Executive Summary

Claude Code provides robust CLI capabilities suitable for programmatic spawning, session management, and output capture. The `-p` (print/programmatic) mode enables non-interactive execution with structured output formats, making it ideal for automation harnesses.

**Key Findings**:
- Programmatic mode via `claude -p` flag supports non-interactive execution
- Session management with resume/continue capabilities via session IDs
- Structured output formats (JSON, stream-JSON) for parsing
- Comprehensive prompt injection via stdin, flags, and file-based methods
- Session logs stored as JSONL at `~/.claude/transcripts/ses_*.jsonl`
- Environment variables and configuration files enable context control

**Recommended Approach**: Use `claude -p` with `--output-format stream-json` for real-time monitoring, `--session-id` for tracking, and `--allowedTools` for permission automation in sandbox environments.

---

## 1. CLI Structure & Options

### Basic Command Structure

```bash
claude [command] [options] ["prompt"]
```

### Primary Commands

| Command | Description | Use Case |
|---------|-------------|----------|
| `claude` | Start interactive REPL | Manual interaction |
| `claude "query"` | Start REPL with initial prompt | Interactive with bootstrap |
| `claude -p "query"` | Query via SDK, then exit | **Programmatic/automation** |
| `cat file \| claude -p "query"` | Process piped content | Content injection via stdin |
| `claude -c` | Continue most recent conversation | Resume last session |
| `claude -c -p "query"` | Continue via SDK | Programmatic resume |
| `claude -r "<session>" "query"` | Resume session by ID or name | Resume specific session |
| `claude update` | Update to latest version | Maintenance |

### Key Flags for Automation

#### Session Management
- `--print`, `-p`: Enable programmatic/non-interactive mode (required for automation)
- `--continue`, `-c`: Load most recent conversation in current directory
- `--resume`, `-r`: Resume specific session by ID or name
- `--session-id`: Use specific UUID for conversation tracking
- `--fork-session`: Create new session ID when resuming (use with `--resume` or `--continue`)
- `--no-session-persistence`: Disable session saving to disk (print mode only)

#### Output Control
- `--output-format`: Specify output format (`text`, `json`, `stream-json`)
- `--json-schema`: Get validated JSON output matching a schema (print mode only)
- `--include-partial-messages`: Include partial streaming events (requires `--print` and `--output-format=stream-json`)
- `--input-format`: Specify input format (`text`, `stream-json`)
- `--verbose`: Enable verbose logging (helpful for debugging)

#### Permission & Tool Control
- `--allowedTools`: Tools that execute without prompting (e.g., `"Bash,Read,Edit"`)
- `--disallowedTools`: Tools removed from model's context
- `--tools`: Restrict which built-in tools Claude can use (e.g., `"Bash,Edit,Read"`)
- `--dangerously-skip-permissions`: Skip all permission prompts (use with caution)
- `--allow-dangerously-skip-permissions`: Enable permission bypassing as option
- `--permission-mode`: Begin in specified permission mode (`plan`, etc.)
- `--permission-prompt-tool`: Specify MCP tool for permission prompts in non-interactive mode

#### Prompt Customization
- `--system-prompt`: Replace entire default system prompt
- `--system-prompt-file`: Load system prompt from file (print mode only)
- `--append-system-prompt`: Append to default prompt (safest option)
- `--append-system-prompt-file`: Append from file (print mode only)

#### Working Directory & Context
- `--add-dir`: Add additional working directories (e.g., `../apps ../lib`)
- `--setting-sources`: Comma-separated list of setting sources (`user`, `project`, `local`)
- `--settings`: Path to settings JSON file or JSON string

#### Resource Limits
- `--max-budget-usd`: Maximum dollar amount for API calls (print mode only)
- `--max-turns`: Limit number of agentic turns (print mode only)
- `--fallback-model`: Enable automatic fallback to specified model (print mode only)
- `--model`: Set model for current session (alias or full name)

#### Advanced Features
- `--agents`: Define custom subagents dynamically via JSON
- `--agent`: Specify agent for current session
- `--mcp-config`: Load MCP servers from JSON files or strings
- `--strict-mcp-config`: Only use MCP servers from `--mcp-config`
- `--chrome`: Enable Chrome browser integration
- `--debug`: Enable debug mode with optional category filtering
- `--betas`: Beta headers to include in API requests (API key users only)

### Complete Flag Reference
See [Claude Code CLI Reference](https://code.claude.com/docs/en/cli-reference) for full flag documentation.

---

## 2. Prompt/Content Injection

### Methods for Injecting Initial Context

#### 1. Direct Argument (Simplest)
```bash
claude -p "Your prompt here"
```

#### 2. Stdin Piping (File Content)
```bash
cat context.txt | claude -p "Analyze this content"
echo "Task description" | claude -p
```

#### 3. System Prompt Files (Bootstrap Instructions)
```bash
# Replace entire system prompt
claude -p --system-prompt-file ./bootstrap.txt "Execute task"

# Append to default prompt (recommended)
claude -p --append-system-prompt-file ./extra-rules.txt "Execute task"
```

#### 4. Inline System Prompt
```bash
# Replace entire prompt
claude -p --system-prompt "You are a security engineer. Review for vulnerabilities." "Review this PR"

# Append to default (safer)
claude -p --append-system-prompt "Always use TypeScript" "Implement the feature"
```

#### 5. Environment Files (Context Setup)
```bash
# Set via environment variable before spawning
export CLAUDE_ENV_FILE=/path/to/env-setup.sh
claude -p "Execute task with environment"
```

### System Prompt Flag Comparison

| Flag | Behavior | Modes | Use Case |
|------|----------|-------|----------|
| `--system-prompt` | **Replaces** entire default prompt | Interactive + Print | Complete control over behavior |
| `--system-prompt-file` | **Replaces** with file contents | Print only | File-based prompts for reproducibility |
| `--append-system-prompt` | **Appends** to default prompt | Interactive + Print | Add instructions while keeping defaults (recommended) |
| `--append-system-prompt-file` | **Appends** file contents | Print only | File-based additions while keeping defaults |

**Recommendation for Harness**: Use `--append-system-prompt-file` with role-specific bootstrap files to inject agent context while preserving Claude Code's built-in capabilities.

### Example Bootstrap Pattern

```bash
# Create role-specific bootstrap files
cat > /tmp/explorer-bootstrap.txt <<EOF
You are an Explorer agent focused on code analysis.
Your task is to gather comprehensive information about the codebase.
Use Read, Grep, and Glob tools to map out the architecture.
Return findings in structured format.
EOF

# Spawn with bootstrap
claude -p \
  --append-system-prompt-file /tmp/explorer-bootstrap.txt \
  --output-format json \
  --allowedTools "Read,Grep,Glob,Bash" \
  "Analyze the authentication module"
```

---

## 3. Session Management

### Session Identification

Every Claude Code session has a unique session ID (UUID format or custom name).

#### Session ID Formats
- **UUID**: `550e8400-e29b-41d4-a716-446655440000`
- **Short ID**: `ses_3fd6fb207ffelYR9V4VsVNQDw5`
- **Named**: `auth-refactor` (user-defined)

#### Specifying Session ID
```bash
# Use specific UUID
claude -p --session-id "550e8400-e29b-41d4-a716-446655440000" "Task"

# Resume by name
claude -r "auth-refactor" "Continue this PR"
```

### Session Operations

#### 1. Start New Session
```bash
# Default (auto-generated session ID)
claude -p "Start new task"

# With specific session ID for tracking
claude -p --session-id "$(uuidgen)" "Start new task"
```

#### 2. Continue Recent Session
```bash
# Continue most recent in current directory
claude -p --continue "Next step"
claude -c -p "Next step"  # Short form
```

#### 3. Resume Specific Session
```bash
# By ID
claude -p --resume "ses_3fd6fb207ffelYR9V4VsVNQDw5" "Continue"

# By name
claude -r "auth-refactor" "Finish this PR"

# Interactive picker (not useful for automation)
claude --resume
```

#### 4. Fork Session (Branch Conversation)
```bash
# Create new session ID from existing session
claude --resume abc123 --fork-session -p "Try alternative approach"
```

#### 5. Disable Persistence (Ephemeral Sessions)
```bash
# Session won't be saved to disk
claude -p --no-session-persistence "One-off task"
```

### Capturing Session ID for Multi-Turn Workflows

```bash
#!/bin/bash

# Capture session ID from JSON output
session_id=$(claude -p "Start a review" --output-format json | jq -r '.session_id')

# Use session ID in subsequent calls
claude -p "Continue review" --resume "$session_id"
claude -p "Generate summary" --resume "$session_id"

echo "Session ID: $session_id"
```

### Session Lifecycle

1. **Creation**: Session starts with first message
2. **Persistence**: Session saved to `~/.claude/transcripts/ses_*.jsonl`
3. **Resume**: Session can be continued via `--continue` or `--resume`
4. **Cleanup**: Sessions inactive for >30 days deleted at startup (configurable via `cleanupPeriodDays` setting)

### Session Context Sharing

```bash
# Share task list across sessions via environment variable
export CLAUDE_CODE_TASK_LIST_ID="shared-harness-tasks"
claude -p "Task 1"
claude -p "Task 2"  # Same task list
```

---

## 4. Output & Logging

### Output Formats

#### 1. Text Format (Default)
```bash
claude -p "Query"
# Output: plain text response
```

#### 2. JSON Format (Structured)
```bash
claude -p "Query" --output-format json
# Output: {"result": "...", "session_id": "...", "usage": {...}, ...}
```

**JSON Structure**:
```json
{
  "result": "The response text",
  "session_id": "ses_abc123",
  "usage": {
    "input_tokens": 1500,
    "output_tokens": 800
  },
  "model": "claude-sonnet-4-5-20250929",
  "timestamp": "2026-01-27T..."
}
```

#### 3. Stream-JSON Format (Real-Time)
```bash
claude -p "Query" --output-format stream-json
# Output: newline-delimited JSON events
```

**Stream-JSON Events**:
```json
{"type":"message_start","timestamp":"..."}
{"type":"content_block_delta","delta":{"text":"Hello"},"timestamp":"..."}
{"type":"tool_use","name":"Read","input":{...},"timestamp":"..."}
{"type":"message_stop","timestamp":"..."}
```

#### 4. Structured Output with Schema
```bash
claude -p "Extract function names from auth.py" \
  --output-format json \
  --json-schema '{
    "type":"object",
    "properties":{
      "functions":{"type":"array","items":{"type":"string"}}
    },
    "required":["functions"]
  }'
# Output includes: "structured_output": {"functions": ["login", "logout", ...]}
```

### Parsing Output with jq

```bash
# Extract result text
claude -p "Query" --output-format json | jq -r '.result'

# Extract session ID
session_id=$(claude -p "Query" --output-format json | jq -r '.session_id')

# Extract structured output
claude -p "Extract data" --output-format json --json-schema '...' \
  | jq '.structured_output'

# Monitor usage
claude -p "Query" --output-format json | jq '.usage'
```

### Session Log Locations

#### Transcript Files
- **Location**: `~/.claude/transcripts/ses_<SESSION_ID>.jsonl`
- **Format**: JSONL (newline-delimited JSON)
- **Content**: Complete conversation history including user messages, assistant responses, tool calls, and tool results

**Example Transcript Structure**:
```jsonl
{"type":"user","timestamp":"2026-01-28T03:05:37.558Z","content":"Your prompt"}
{"type":"tool_use","timestamp":"2026-01-28T03:05:59.813Z","tool_name":"Read","tool_input":{"filePath":"..."}}
{"type":"tool_result","timestamp":"2026-01-28T03:06:00.125Z","tool_name":"Read","result":"..."}
{"type":"assistant","timestamp":"2026-01-28T03:06:15.432Z","content":"Response text"}
```

#### Configuration & State Files

| File/Directory | Location | Purpose |
|----------------|----------|---------|
| Transcripts | `~/.claude/transcripts/` | Session conversation logs (JSONL) |
| Projects | `~/.claude/projects/<PROJECT>/` | Project-specific session data |
| Debug Logs | `~/.claude/debug/` | Debug output (if `--debug` enabled) |
| Session Environment | `~/.claude/session-env/` | Session-specific environment snapshots |
| Task Lists | `~/.claude/tasks/` | Task tracking data |
| History | `~/.claude/history.jsonl` | Command history |
| Settings | `~/.claude/settings.json` | User-level settings |

#### Reading Transcript Logs

```bash
# View latest session transcript
latest=$(ls -t ~/.claude/transcripts/ses_*.jsonl | head -1)
cat "$latest" | jq '.'

# Extract all user messages
cat "$latest" | jq 'select(.type == "user") | .content'

# Extract tool calls
cat "$latest" | jq 'select(.type == "tool_use") | {tool: .tool_name, input: .tool_input}'

# Count tokens (requires parsing usage from API responses)
cat "$latest" | jq 'select(.usage != null) | .usage'
```

### Exit Codes

Claude Code returns standard exit codes:

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| `0` | Success | Continue workflow |
| `1` | General error | Check stderr, log error |
| `127` | Command not found | Claude not installed/in PATH |
| Other | Specific error | Check documentation |

**Example Error Handling**:
```bash
#!/bin/bash
set -euo pipefail

claude -p "Task" --output-format json > output.json
exit_code=$?

if [[ $exit_code -eq 0 ]]; then
  echo "Success"
  jq '.result' output.json
else
  echo "Failed with exit code: $exit_code"
  cat output.json
  exit $exit_code
fi
```

### Logging Best Practices

```bash
#!/bin/bash

# Create log directory
LOG_DIR="./harness/logs"
mkdir -p "$LOG_DIR"

# Generate log filename
timestamp=$(date +%Y%m%d_%H%M%S)
session_id=$(uuidgen)
log_file="$LOG_DIR/agent_${session_id}_${timestamp}.log"

# Run with logging
{
  echo "=== Session Start: $session_id ==="
  echo "Timestamp: $(date -Iseconds)"
  echo "Command: claude -p \"Task\""
  echo ""

  claude -p "Task" \
    --session-id "$session_id" \
    --output-format json \
    --verbose 2>&1

  exit_code=$?
  echo ""
  echo "Exit Code: $exit_code"
  echo "=== Session End ==="
} | tee "$log_file"

# Copy transcript to logs
cp ~/.claude/transcripts/ses_${session_id}.jsonl "$LOG_DIR/"
```

---

## 5. Environment & Context

### Environment Variables

#### Authentication
- `ANTHROPIC_API_KEY`: API key for Claude (keep unset when using Claude subscription to avoid charges)

#### Configuration
- `CLAUDE_CONFIG_DIR`: Customize configuration/data storage location (default: `~/.claude`)
- `CLAUDE_CODE_TMPDIR`: Override temp directory (default: `/tmp` on Unix/macOS)
- `CLAUDE_ENV_FILE`: Path to shell script sourced before each Bash command

#### Session Control
- `CLAUDE_CODE_TASK_LIST_ID`: Share task list across sessions
- `CLAUDE_CODE_EXIT_AFTER_STOP_DELAY`: Auto-exit delay for automated workflows (milliseconds)
- `CLAUDE_SESSION_ID`: Session UUID (feature request, not yet available)

#### Behavior Flags
- `CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR=1`: Reset to project directory after each bash command
- `CLAUDE_CODE_USE_BEDROCK=1`: Use Amazon Bedrock integration
- `CLAUDE_CODE_ENABLE_TELEMETRY=1`: Enable telemetry
- `HTTPS_PROXY`: Route traffic through corporate proxy

#### Example Environment Setup
```bash
#!/bin/bash

# Set environment for automation
export CLAUDE_CONFIG_DIR="/opt/harness/claude-config"
export CLAUDE_CODE_TMPDIR="/opt/harness/tmp"
export CLAUDE_CODE_TASK_LIST_ID="harness-$(date +%Y%m%d)"
export CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR=1

# Disable telemetry in CI
export CLAUDE_CODE_ENABLE_TELEMETRY=0

# Run agent
claude -p "Task" --output-format json
```

### Configuration Files

#### Settings Hierarchy (Highest to Lowest Priority)

1. **Managed Settings** (System-level, IT-deployed)
   - macOS: `/Library/Application Support/ClaudeCode/managed-settings.json`
   - Linux/WSL: `/etc/claude-code/managed-settings.json`
   - Windows: `C:\Program Files\ClaudeCode\managed-settings.json`

2. **Command Line Arguments** (Temporary overrides)
   - `--settings ./settings.json` or `--settings '{"key":"value"}'`

3. **Local Project Settings** (Developer-specific, gitignored)
   - `.claude/settings.local.json`

4. **Shared Project Settings** (Team-wide, committed to git)
   - `.claude/settings.json`

5. **User Settings** (Personal defaults)
   - `~/.claude/settings.json`

#### Other Configuration Files

| File | Location | Purpose | Shared? |
|------|----------|---------|---------|
| `settings.json` | `~/.claude/` | User-level settings | No |
| `settings.json` | `.claude/` | Project-level settings | Yes (git) |
| `settings.local.json` | `.claude/` | Personal overrides | No (gitignored) |
| `.claude.json` | `~/.claude.json` | Preferences, OAuth, MCP servers, trust settings | No |
| `.mcp.json` | Project root | Project-scoped MCP servers | Yes (git) |
| `CLAUDE.md` | `~/.claude/` or `.claude/` | Memory/context files with instructions | Varies |

#### Example Settings Configuration

```json
{
  "permissions": {
    "allow": [
      "Bash(npm run *)",
      "Read",
      "Edit",
      "Glob",
      "Grep"
    ],
    "deny": [
      "Bash(curl *)",
      "Read(./.env*)",
      "Read(./secrets/**)",
      "WebFetch"
    ]
  },
  "env": {
    "CLAUDE_CODE_ENABLE_TELEMETRY": "0",
    "NODE_ENV": "production"
  },
  "sandbox": {
    "enabled": true,
    "excludedCommands": ["docker", "kubectl"]
  },
  "attribution": {
    "commit": "🤖 Generated with Claude Code",
    "pr": ""
  },
  "cleanupPeriodDays": 30
}
```

### Working Directory Control

#### Set Working Directory
```bash
# Run in specific directory
cd /path/to/project && claude -p "Task"

# Add additional directories
claude -p "Task" --add-dir ../shared-libs ../config
```

#### Maintain Working Directory
```bash
# Reset to project directory after each bash command
export CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR=1
claude -p "Run multiple commands"
```

### Session Isolation

Claude Code sessions are isolated by:
1. **Session ID**: Each session has unique identifier
2. **Working Directory**: Sessions track their starting directory
3. **Project Context**: Session data stored in project-specific directories
4. **Environment**: Environment variables can be session-scoped

**Example Isolation Pattern**:
```bash
#!/bin/bash

# Create isolated environment
export CLAUDE_CONFIG_DIR="/tmp/harness-$(uuidgen)"
export CLAUDE_CODE_TMPDIR="/tmp/harness-tmp-$(uuidgen)"

# Run isolated session
claude -p "Task" \
  --no-session-persistence \
  --session-id "$(uuidgen)" \
  --output-format json

# Cleanup
rm -rf "$CLAUDE_CONFIG_DIR" "$CLAUDE_CODE_TMPDIR"
```

---

## 6. Integration Patterns

### Pattern 1: Basic Programmatic Invocation

```bash
#!/bin/bash
set -euo pipefail

# Simple single-turn execution
result=$(claude -p "Analyze auth.py" --output-format json)
echo "$result" | jq -r '.result'
```

### Pattern 2: Session-Based Multi-Turn Workflow

```bash
#!/bin/bash
set -euo pipefail

# Start session and capture ID
session_output=$(claude -p "Review codebase for security issues" \
  --output-format json \
  --allowedTools "Read,Grep,Glob")

session_id=$(echo "$session_output" | jq -r '.session_id')

# Continue with follow-up prompts
claude -p "Focus on authentication logic" \
  --resume "$session_id" \
  --output-format json

claude -p "Generate summary report" \
  --resume "$session_id" \
  --output-format json \
  > security_report.json
```

### Pattern 3: Stream Monitoring with Real-Time Processing

```bash
#!/bin/bash
set -euo pipefail

# Process streaming output in real-time
claude -p "Complex task" \
  --output-format stream-json \
  --include-partial-messages \
  --verbose |
while IFS= read -r line; do
  event_type=$(echo "$line" | jq -r '.type')

  case $event_type in
    "content_block_delta")
      # Handle content updates
      text=$(echo "$line" | jq -r '.delta.text // empty')
      [[ -n "$text" ]] && echo -n "$text"
      ;;
    "tool_use")
      # Monitor tool usage
      tool=$(echo "$line" | jq -r '.name')
      echo "[TOOL: $tool]" >&2
      ;;
    "message_stop")
      echo "" >&2
      echo "[COMPLETE]" >&2
      ;;
  esac
done
```

### Pattern 4: Worker Loop with Error Handling

```bash
#!/bin/bash
set -euo pipefail

LOG_DIR="./logs"
mkdir -p "$LOG_DIR"

# Cleanup trap for unexpected exits
cleanup() {
  local exit_code=$?
  echo "Worker interrupted with exit code: $exit_code" >&2
  # Mark tasks as failed, cleanup resources
  exit $exit_code
}
trap cleanup EXIT INT TERM

# Worker function
run_worker() {
  local task="$1"
  local timestamp=$(date +%Y%m%d_%H%M%S)
  local log_file="$LOG_DIR/worker_${timestamp}.log"

  echo "=== Starting worker: $task ===" | tee -a "$log_file"

  # Run with comprehensive error handling
  if output=$(claude -p "$task" \
    --output-format json \
    --allowedTools "Bash,Read,Edit" \
    --max-turns 10 \
    --verbose 2>&1 | tee -a "$log_file"); then

    echo "=== Worker completed successfully ===" | tee -a "$log_file"
    echo "$output" | jq '.'
    return 0
  else
    local exit_code=$?
    echo "=== Worker failed: exit code $exit_code ===" | tee -a "$log_file"
    return $exit_code
  fi
}

# Main loop
while IFS= read -r task; do
  if run_worker "$task"; then
    echo "Task completed: $task"
  else
    echo "Task failed: $task"
  fi
done < tasks.txt
```

### Pattern 5: Parallel Agent Spawning

```bash
#!/bin/bash

# Spawn multiple agents in parallel
spawn_agent() {
  local role="$1"
  local task="$2"
  local bootstrap_file="$3"
  local session_id=$(uuidgen)

  claude -p "$task" \
    --session-id "$session_id" \
    --append-system-prompt-file "$bootstrap_file" \
    --output-format json \
    --allowedTools "Read,Grep,Glob" \
    > "output_${role}_${session_id}.json" 2>&1 &

  echo "$session_id:$!"
}

# Launch agents
explorer_pid=$(spawn_agent "explorer" "Analyze codebase structure" "explorer.txt")
librarian_pid=$(spawn_agent "librarian" "Document API usage" "librarian.txt")

# Wait for completion
wait ${explorer_pid##*:}
wait ${librarian_pid##*:}

# Collect results
cat output_explorer_*.json | jq -s '.'
cat output_librarian_*.json | jq -s '.'
```

### Pattern 6: CI/CD Integration

```bash
#!/bin/bash
# .github/workflows/claude-review.sh

set -euo pipefail

# Get PR diff
pr_diff=$(gh pr diff "$PR_NUMBER")

# Run review with Claude
review=$(echo "$pr_diff" | claude -p \
  --append-system-prompt "You are a security engineer. Review for vulnerabilities and best practices." \
  --output-format json \
  --allowedTools "Bash(git *)" \
  --max-budget-usd 5.00 \
  "Review this PR diff for security issues and code quality")

# Extract review comments
echo "$review" | jq -r '.result' > review.md

# Post comment to PR
gh pr comment "$PR_NUMBER" --body-file review.md
```

### Pattern 7: Automated Commit Generation

```bash
#!/bin/bash

# Generate commit from staged changes
claude -p "Look at my staged changes and create an appropriate commit" \
  --allowedTools "Bash(git diff *),Bash(git status *),Bash(git commit *)" \
  --output-format json

# Verify commit was created
if git log -1 --pretty=format:"%s" | grep -q .; then
  echo "Commit created successfully"
else
  echo "Failed to create commit"
  exit 1
fi
```

### Pattern 8: Structured Data Extraction

```bash
#!/bin/bash

# Extract structured data with schema validation
schema='{
  "type": "object",
  "properties": {
    "functions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "description": {"type": "string"},
          "parameters": {"type": "array", "items": {"type": "string"}}
        },
        "required": ["name", "description"]
      }
    }
  },
  "required": ["functions"]
}'

output=$(claude -p "Extract all function definitions from auth.py" \
  --output-format json \
  --json-schema "$schema" \
  --allowedTools "Read")

# Access structured output
echo "$output" | jq '.structured_output.functions'
```

### Pattern 9: Bootstrap with Role-Specific Context

```bash
#!/bin/bash

# Create role-specific bootstrap
cat > /tmp/implementer-bootstrap.txt <<EOF
You are an Implementer agent responsible for code generation.
Follow these guidelines:
- Write clean, well-documented code
- Include comprehensive error handling
- Add unit tests for new functionality
- Follow project style guide in .claude/settings.json
- Use TypeScript with strict mode
- Prefer functional programming patterns
EOF

# Spawn with bootstrap
claude -p "Implement user authentication endpoint" \
  --append-system-prompt-file /tmp/implementer-bootstrap.txt \
  --allowedTools "Read,Edit,Write,Bash(npm test *)" \
  --output-format json
```

### Pattern 10: Monitoring and Heartbeat

```bash
#!/bin/bash

# Start agent with monitoring
session_id=$(uuidgen)
pid_file="/tmp/claude_${session_id}.pid"

# Run in background
claude -p "Long-running task" \
  --session-id "$session_id" \
  --output-format stream-json \
  --verbose &

agent_pid=$!
echo "$agent_pid" > "$pid_file"

# Monitor progress
monitor_agent() {
  local session_file=~/.claude/transcripts/ses_${session_id}.jsonl

  while kill -0 "$agent_pid" 2>/dev/null; do
    if [[ -f "$session_file" ]]; then
      # Count messages
      msg_count=$(wc -l < "$session_file")
      echo "Progress: $msg_count messages" >&2
    fi
    sleep 5
  done

  # Check exit status
  wait "$agent_pid"
  echo "Agent completed with exit code: $?" >&2
}

monitor_agent &
wait "$agent_pid"
```

---

## 7. Best Practices for Harness Implementation

### Recommendations

#### 1. Use Programmatic Mode
- Always use `claude -p` for automation
- Avoid interactive mode in automated contexts

#### 2. Structured Output
- Use `--output-format json` for parsing
- Use `--output-format stream-json` for real-time monitoring
- Use `--json-schema` for validated structured data

#### 3. Session Management
- Generate and track session IDs with `--session-id "$(uuidgen)"`
- Use `--resume` for multi-turn workflows
- Consider `--no-session-persistence` for ephemeral tasks

#### 4. Permission Automation
- Use `--allowedTools` to whitelist tools for sandbox execution
- Use specific patterns: `"Bash(git diff *)"` for prefix matching
- Avoid `--dangerously-skip-permissions` except in fully sandboxed environments

#### 5. Bootstrap Context
- Use `--append-system-prompt-file` for role-specific instructions
- Keep bootstrap files version-controlled
- Preserve Claude Code defaults (avoid `--system-prompt` unless necessary)

#### 6. Error Handling
- Check exit codes
- Log stdout and stderr
- Implement cleanup traps
- Set `set -euo pipefail` in shell scripts

#### 7. Monitoring
- Parse stream-JSON for real-time updates
- Track transcript files for detailed logs
- Monitor tool usage and resource consumption

#### 8. Resource Limits
- Use `--max-budget-usd` to cap costs
- Use `--max-turns` to prevent runaway agents
- Set timeouts at the process level

### Implementation Checklist

- [ ] Use `claude -p` for programmatic execution
- [ ] Implement session ID tracking (`--session-id "$(uuidgen)"`)
- [ ] Use `--output-format stream-json` for monitoring
- [ ] Create role-specific bootstrap files
- [ ] Use `--append-system-prompt-file` for context injection
- [ ] Whitelist tools with `--allowedTools`
- [ ] Implement exit code checking and error logging
- [ ] Parse transcript files for detailed analysis
- [ ] Set resource limits (`--max-turns`, `--max-budget-usd`)
- [ ] Implement cleanup traps for graceful shutdown
- [ ] Use environment variables for configuration
- [ ] Isolate sessions in sandboxed environments

---

## 8. Limitations and Constraints

### Known Limitations

1. **Programmatic Tool Calling (PTC)**: Not yet available in Claude Code CLI (GitHub issue #12836). Advanced PTC features like `allowed_callers` flag only available via direct API.

2. **Interactive Commands**: User-invoked skills (e.g., `/commit`) and built-in commands only work in interactive mode, not in `-p` mode.

3. **Session ID Exposure**: `CLAUDE_SESSION_ID` environment variable is a feature request (GitHub issue #17188), not yet available.

4. **Platform Compatibility**: Some flags and features may vary between macOS, Linux, and Windows.

5. **MCP Server Compatibility**: Some MCP servers may not work properly in programmatic mode.

6. **Exit Codes**: Limited documentation on specific exit code meanings beyond standard codes.

### Workarounds

- **PTC**: Use `--allowedTools` with permission rule syntax for tool whitelisting
- **Interactive Commands**: Describe tasks explicitly in `-p` mode instead of using slash commands
- **Session Tracking**: Capture session ID from JSON output instead of environment variable
- **Platform Issues**: Test thoroughly on target platform, use Docker for consistency

---

## 9. References and Sources

### Official Documentation
- [CLI Reference](https://code.claude.com/docs/en/cli-reference)
- [Programmatic Usage (Headless Mode)](https://code.claude.com/docs/en/headless)
- [Settings and Configuration](https://code.claude.com/docs/en/settings)
- [Agent SDK Documentation](https://platform.claude.com/docs/en/agent-sdk/overview)

### GitHub
- [Claude Code Repository](https://github.com/anthropics/claude-code)
- [Issue #6493: Missing CLI Documentation](https://github.com/anthropics/claude-code/issues/6493)
- [Issue #12836: PTC Feature Request](https://github.com/anthropics/claude-code/issues/12836)
- [Issue #17188: Session Metadata Environment Variables](https://github.com/anthropics/claude-code/issues/17188)

### Guides and Best Practices
- [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [Shipyard Claude Code Cheatsheet](https://shipyard.build/blog/claude-code-cheat-sheet/)
- [Claude Code CLI Reference by eesel.ai](https://www.eesel.ai/blog/claude-code-cli-reference)
- [Automated Claude Code Workers](https://www.blle.co/blog/automated-claude-code-workers)
- [Managing API Keys](https://support.claude.com/en/articles/12304248-managing-api-key-environment-variables-in-claude-code)

### Community Resources
- [Various Ways of Accessing Claude Code](https://kvssetty.medium.com/various-ways-of-accessing-claude-code-in-2026-822aff7c53bd)
- [Claude Code Environment Variables Guide](https://medium.com/@dan.avila7/claude-code-environment-variables-a-complete-reference-guide-41229ef18120)
- [The Complete Guide](https://www.siddharthbharath.com/claude-code-the-complete-guide/)

---

## 10. Recommended Architecture for Harness

### Core Components

1. **Agent Spawner**
   - Use `subprocess.Popen()` or equivalent to spawn `claude -p` processes
   - Generate UUID session IDs for tracking
   - Inject role-specific bootstrap files via `--append-system-prompt-file`
   - Pass `--output-format stream-json` for monitoring

2. **Session Manager**
   - Track session IDs and PIDs
   - Store session metadata (role, task, start_time, status)
   - Monitor transcript files at `~/.claude/transcripts/ses_*.jsonl`
   - Implement resume/continue logic via `--resume`

3. **Output Processor**
   - Parse stream-JSON events in real-time
   - Extract tool calls, content deltas, errors
   - Aggregate results for synthesis
   - Detect completion via `message_stop` events

4. **Resource Manager**
   - Set `--max-turns` and `--max-budget-usd` limits
   - Implement process timeouts
   - Monitor CPU, memory, disk usage
   - Clean up orphaned sessions

5. **Error Handler**
   - Capture exit codes
   - Parse stderr for errors
   - Retry failed tasks with backoff
   - Log errors to centralized system

### Example Spawn Function

```python
import subprocess
import uuid
import json
from typing import Dict, Any, Optional
from pathlib import Path

def spawn_claude_agent(
    role: str,
    task: str,
    bootstrap_file: Path,
    allowed_tools: list[str],
    max_turns: int = 10,
    max_budget_usd: float = 5.0,
    working_dir: Path = Path.cwd(),
    output_format: str = "stream-json"
) -> Dict[str, Any]:
    """
    Spawn a Claude Code agent with specified parameters.

    Returns:
        dict: {
            'session_id': str,
            'pid': int,
            'process': subprocess.Popen,
            'role': str,
            'start_time': float
        }
    """
    import time

    # Generate session ID
    session_id = str(uuid.uuid4())

    # Build command
    cmd = [
        "claude", "-p", task,
        "--session-id", session_id,
        "--output-format", output_format,
        "--append-system-prompt-file", str(bootstrap_file),
        "--allowedTools", ",".join(allowed_tools),
        "--max-turns", str(max_turns),
        "--max-budget-usd", str(max_budget_usd),
        "--verbose"
    ]

    # Spawn process
    process = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd=working_dir,
        bufsize=1,  # Line buffered
        universal_newlines=True
    )

    return {
        'session_id': session_id,
        'pid': process.pid,
        'process': process,
        'role': role,
        'task': task,
        'start_time': time.time(),
        'status': 'running'
    }

# Usage
agent = spawn_claude_agent(
    role="explorer",
    task="Analyze the authentication module",
    bootstrap_file=Path("./harness/prompts/explorer.txt"),
    allowed_tools=["Read", "Grep", "Glob", "Bash"],
    max_turns=10
)

print(f"Spawned agent: {agent['role']} (PID: {agent['pid']}, Session: {agent['session_id']})")

# Monitor output
for line in agent['process'].stdout:
    event = json.loads(line)
    print(f"Event: {event['type']}")

    if event['type'] == 'message_stop':
        break

# Check completion
exit_code = agent['process'].wait()
print(f"Agent completed with exit code: {exit_code}")
```

### Monitoring Example

```python
import json
from pathlib import Path
from typing import Dict, Any

def monitor_agent_transcript(session_id: str) -> list[Dict[str, Any]]:
    """
    Monitor agent transcript file and return parsed events.
    """
    transcript_file = Path.home() / ".claude" / "transcripts" / f"ses_{session_id}.jsonl"

    events = []
    if transcript_file.exists():
        with open(transcript_file, 'r') as f:
            for line in f:
                event = json.loads(line)
                events.append(event)

    return events

def get_agent_tool_usage(session_id: str) -> list[Dict[str, Any]]:
    """
    Extract tool usage from transcript.
    """
    events = monitor_agent_transcript(session_id)
    tool_calls = [e for e in events if e.get('type') == 'tool_use']

    return [
        {
            'tool': tc.get('tool_name'),
            'timestamp': tc.get('timestamp'),
            'input': tc.get('tool_input')
        }
        for tc in tool_calls
    ]

# Usage
session_id = "abc123"
tools = get_agent_tool_usage(session_id)
print(f"Agent used {len(tools)} tools")
for tool in tools:
    print(f"  - {tool['tool']} at {tool['timestamp']}")
```

---

## Conclusion

Claude Code CLI provides robust capabilities for programmatic invocation and automation. The `-p` flag with structured output formats, session management, and comprehensive configuration options make it suitable for building an automation harness.

**Key Takeaways**:
1. Use `claude -p` with `--output-format stream-json` for real-time monitoring
2. Track sessions with `--session-id` and resume with `--resume`
3. Inject context via `--append-system-prompt-file` (preferred over full replacement)
4. Whitelist tools with `--allowedTools` for permission automation
5. Parse transcript files at `~/.claude/transcripts/ses_*.jsonl` for detailed logs
6. Implement proper error handling with exit codes and stderr capture
7. Set resource limits with `--max-turns` and `--max-budget-usd`

**Next Steps for Harness**:
- Implement `spawn_agent()` function using subprocess
- Create role-specific bootstrap files for Explorer, Librarian, Implementer, Oracle
- Build output processor for stream-JSON parsing
- Design session manager with transcript monitoring
- Add resource limits and timeout handling
- Test parallel agent spawning and coordination

---

**Document Version**: 1.0
**Last Updated**: 2026-01-27
**Maintainer**: Harness Development Team
