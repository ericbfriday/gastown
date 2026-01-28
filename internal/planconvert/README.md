# Plan-to-Epic Converter

Converts markdown planning documents into beads epic structures with subtasks.

## Overview

This package bridges high-level planning documents (like design docs with phases and task lists) to executable work items in the beads system. It parses structured markdown documents and generates beads-compatible epic hierarchies.

## Architecture

### Core Components

- **`parser.go`** - Markdown document parser
  - Extracts document structure (headers, sections, lists)
  - Identifies phase headers and task lists
  - Parses metadata from frontmatter
  - Handles nested sections

- **`epic.go`** - Epic generation logic
  - Converts parsed structure to beads format
  - Generates unique IDs for epics and tasks
  - Maintains phase hierarchy
  - Links tasks to epics via dependencies

- **`output.go`** - Multiple output formats
  - JSONL (beads-compatible)
  - JSON (pretty-printed)
  - Pretty (human-readable summary)
  - Shell (bd CLI commands)

- **`types.go`** - Data structures
  - PlanDocument, Section, Task
  - Epic, Bead, Dependency
  - ConversionOptions

## Supported Document Formats

### Phase-Based Plans

Documents with explicit phase structure:

```markdown
# Project Name

**Document Version:** 1.0
**Status:** Draft

## Phase 1: Foundation

### Phase 1.1: Setup (Week 1)

**Tasks:**
1. Set up environment
2. Configure CI/CD
3. Initialize repo

**Deliverables:**
- ✅ Environment ready
- ✅ CI running

**Success Criteria:**
- All tests pass
- Deployment works
```

### Pattern Recognition

The parser recognizes:
- **Phase headers**: `## Phase 1.1: Title`
- **Task sections**: `**Tasks:**` followed by numbered lists
- **Deliverables**: Checkboxes (✅, ☐, [ ], [x])
- **Success criteria**: Acceptance conditions
- **Metadata**: Key-value pairs in frontmatter

## Usage

### Standalone Demo Binary

```bash
# Build
go build -o bin/plan-to-epic-demo ./cmd/plan-to-epic-demo/

# Pretty print summary
./bin/plan-to-epic-demo docs/design.md --format pretty

# Generate JSONL for beads
./bin/plan-to-epic-demo docs/design.md --format jsonl --output epic.jsonl

# Generate shell script with bd commands
./bin/plan-to-epic-demo docs/design.md --format shell --output create-epic.sh
bash create-epic.sh
```

### Programmatic Usage

```go
import "github.com/steveyegge/gastown/internal/planconvert"

// Parse document
doc, err := planconvert.ParsePlanDocument("design.md")
if err != nil {
    log.Fatal(err)
}

// Convert to epic
opts := planconvert.ConversionOptions{
    Prefix:   "project",
    Priority: 2,
}
epic, err := planconvert.ConvertToEpic(doc, opts)
if err != nil {
    log.Fatal(err)
}

// Output as JSONL
err = planconvert.WriteEpic(epic, os.Stdout, planconvert.FormatJSONL)
```

### CLI Command (Future)

Once build issues are resolved:

```bash
# Preview conversion
gt plan-to-epic docs/design.md --dry-run

# Generate output
gt plan-to-epic docs/design.md --output epic.jsonl

# Create beads directly
gt plan-to-epic docs/design.md --create --rig myproject
```

## Output Formats

### JSONL (Beads Format)

One JSON object per line (epic + tasks):

```jsonl
{"id":"proj-abc","title":"Epic","issue_type":"epic",...}
{"id":"proj-xyz-1","title":"Task 1","issue_type":"task","dependencies":[...],...}
{"id":"proj-xyz-2","title":"Task 2","issue_type":"task","dependencies":[...],...}
```

### JSON (Pretty)

Formatted JSON with full epic structure:

```json
{
  "id": "proj-abc",
  "title": "Epic Title",
  "subtasks": [
    {"id": "proj-xyz-1", "title": "Task 1", ...},
    {"id": "proj-xyz-2", "title": "Task 2", ...}
  ]
}
```

### Pretty (Human-Readable)

```
Epic: Example Multi-Phase Project
ID: demo-00ozi
Priority: 2
Source: testdata/plans/example-phases.md

Tasks (24):
  1. [demo-sfu0j-1] Set up development environment
     Phase: Phase 1.1: Infrastructure (Week 1)
  2. [demo-4rw0j-2] Configure CI/CD pipeline
     Phase: Phase 1.1: Infrastructure (Week 1)
  ...
```

### Shell (BD Commands)

Executable bash script:

```bash
#!/usr/bin/env bash
set -e

# Create epic
EPIC_ID=$(bd create --title "Epic" --type epic --priority 2)

# Task 1
bd create --title "Task 1" --type task --depends-on "$EPIC_ID"

# Task 2
bd create --title "Task 2" --type task --depends-on "$EPIC_ID"
```

## Testing

Run tests:

```bash
go test ./internal/planconvert/...
```

Example results:
- `example-phases.md`: 3 phases, 24 tasks
- `parallel-coordination-design.md`: 7 phases, 35 tasks

## Examples

### Example 1: Simple Phase Plan

See `testdata/plans/example-phases.md` for a complete example.

### Example 2: Real Planning Document

The converter was tested with `harness/docs/research/parallel-coordination-design.md`, successfully extracting:
- 7 implementation phases
- 35 individual tasks
- Deliverables and success criteria
- Phase-to-task hierarchy

## Limitations & Future Work

### Current Limitations

1. **Metadata parsing** - Frontmatter parsing needs enhancement
2. **Dependency detection** - Text-based dependency extraction not yet implemented
3. **BD integration** - Direct bead creation via CLI pending
4. **ID generation** - Uses simple timestamp-based IDs vs beads standard

### Planned Enhancements

1. **Smarter parsing**
   - Detect dependencies from "Depends on:", "Blocks:" patterns
   - Extract priority hints from text
   - Parse effort estimates

2. **BD CLI integration**
   - Direct bead creation with `--create` flag
   - Validation against existing beads
   - Merge detection

3. **Advanced features**
   - Template support for different plan formats
   - Incremental updates (diff existing epic)
   - Milestone tracking

4. **CLI improvements**
   - Interactive mode for ambiguous sections
   - Preview with diff view
   - Validation warnings

## Related

- **Task**: gt-z4g (this implementation)
- **Use case**: Converting `harness/docs/research/` plans to epics
- **Beads format**: `.beads/issues.jsonl`
- **Dependencies**: stdlib only (no external markdown parser yet)

## Contributing

When adding support for new planning formats:
1. Add pattern recognition to `parser.go`
2. Create test document in `testdata/plans/`
3. Add test case to `parser_test.go`
4. Update this README with format example
