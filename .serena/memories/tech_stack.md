# Technology Stack

## System Environment

**Platform**: macOS Darwin 25.2.0 (Apple Silicon)
**Shell**: bash (primary), with zsh compatibility

## Programming Languages

### Primary Languages
1. **Bash** - System scripts, harness, automation
2. **JavaScript/TypeScript** - Mudlet packages, web dashboards (in rigs)
3. **Go** - Gastown (`gt`) and related tools (in rigs)

### Available Runtimes

**Python**:
- System: Python 3.14.2 (Homebrew)
- Package Manager: `uv 0.9.11`
- Location: `/opt/homebrew/bin/python3`
- Managed Pythons: `/Users/ericfriday/.local/share/uv/python/`

**Node.js** (via Volta):
- Default: v20.19.6
- Available: v22.21.1, v24.4.1, v24.12.0
- Manager: Volta.sh
- npm: 10.8.2, Yarn: 4.11.0

**Go**:
- Version: 1.25.6
- GOPATH: `/Users/ericfriday/go`
- GOROOT: `/opt/homebrew/Cellar/go/1.25.6/libexec`

## Key Tools & Binaries

**Gastown Suite**:
- `gt` - Gastown orchestration tool (`/Users/ericfriday/go/bin/gt`, v0.4.0)
- `bd` - Beads issue tracker (`/Users/ericfriday/.local/bin/bd`, v0.47.1)

**Version Control**:
- Git (configured for Eric Friday <ericfriday@gmail.com>)

**Build & Package Tools**:
- npm (Node.js packages)
- Yarn (alternative package manager)
- uv (Python package manager)
- go build/test

**Utilities**:
- jq - JSON processing
- GNU tools via Homebrew (findutils, coreutils, gnu-sed)

## File Formats

- **YAML** - Configuration files (harness/config.yaml)
- **TOML** - Workflow formulas (.beads/formulas/)
- **JSON** - State files, work queues, beads data
- **JSONL** - Beads interactions (.beads/interactions.jsonl)
- **Markdown** - Documentation, notes, summaries

## PATH Priority

1. `/opt/homebrew/bin` - Homebrew packages
2. `/Users/ericfriday/.volta/bin` - Node.js via Volta
3. `/Users/ericfriday/.local/bin` - Python/uv, bd
4. `/Users/ericfriday/go/bin` - Go binaries, gt
5. System paths
6. GNU tools
