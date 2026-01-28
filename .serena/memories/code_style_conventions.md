# Code Style & Conventions

## Bash Scripts

**Style Guidelines**:
- Use `#!/usr/bin/env bash` shebang
- Enable strict mode: `set -euo pipefail`
- Functions before main logic
- Use meaningful variable names (UPPER_CASE for constants)
- Comment complex logic
- Log with timestamps

**Example** (from harness/loop.sh):
```bash
#!/usr/bin/env bash
set -euo pipefail

HARNESS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() {
  echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*" | tee -a "$ITERATION_FILE"
}
```

**Executability**:
- All scripts should be executable (`chmod +x`)
- Scripts location matters for context

## Documentation

**Markdown Style**:
- Use proper heading hierarchy (# H1, ## H2, etc.)
- Code blocks with language specifiers
- Examples for all workflows
- Table of contents for long documents
- Reference links for external resources

**Documentation Types**:
- `README.md` - Complete project/component guide
- `GETTING-STARTED.md` - Quick start for new users
- `ROADMAP.md` - Implementation plans
- `SUMMARY.md` - Implementation summaries
- Session docs in `docs/sessions/`
- Research notes in `docs/research/`
- Decisions in `docs/decisions/`

## Configuration Files

**YAML** (harness/config.yaml):
- Comments for complex options
- Grouped by functional area
- Sensible defaults
- Document units (seconds, MB, etc.)

**JSON**:
- Pretty-printed for readability
- Use `.json` extension
- JSONL for append-only logs (`.jsonl`)

## Git Workflow

**Commit Messages**:
- Use conventional commit format: `type: description`
- Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`
- Include issue ID when relevant: `feat: add queue manager (gt-123)`
- Co-author attribution: `Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>`

**Branch Naming**:
- `feature/<name>` - New features
- `polecat/<name>` - Polecat work branches
- `fix/<name>` - Bug fixes

## File Organization

**Directory Structure**:
- Keep related files together
- Use subdirectories for organization
- `scripts/` for helper scripts
- `docs/` for documentation
- `state/` for runtime state
- `prompts/` for agent prompts

**Naming Conventions**:
- kebab-case for files: `manage-queue.sh`
- Descriptive names over abbreviations
- Extension indicates purpose: `.sh`, `.md`, `.yaml`

## Agent-Specific

**Identity Format**: `<town>/<role>/<name>`
- Example: `gastown/crew/ericfriday`

**Agent Attribution**:
- All work attributed to performing agent
- Git commits use agent identity
- Beads issues track creator
- Events log actor

## Testing & Quality

**Required for Completion**:
- All tests must pass
- Build must succeed
- Git working tree must be clean
- Changes must be pushed to remote

**Pre-work Checks** (advisory):
- Verify main branch health
- Run tests on main
- File issues if failures pre-exist

## Automation Patterns

**Ralph Wiggum Loop**:
- Start with minimal context
- Build understanding from docs
- Preserve research and findings
- Signal for help early (interrupt)
- Complete or hand off clearly

**Knowledge Preservation**:
- Document decisions as they're made
- Save research findings
- Use Serena memories for cross-session data
- Create session summaries
