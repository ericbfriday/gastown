` Plan-Oracle Plugin

Intelligent work planning and analysis for the gastown ecosystem.

## Overview

Plan-oracle analyzes work items and planning documents to provide:

- **Work Decomposition**: Break down epics/features into actionable tasks
- **Dependency Analysis**: Identify and visualize dependencies
- **Effort Estimation**: Estimate complexity and time based on patterns
- **Execution Ordering**: Recommend optimal task ordering
- **Risk Identification**: Flag high-risk or blocking work items
- **Resource Planning**: Suggest parallelization opportunities

## Commands

### `gt plan-oracle decompose <issue-id>`

Break down a large work item into smaller tasks.

```bash
# Show decomposition preview
gt plan-oracle decompose gt-35x

# Auto-create tasks in beads
gt plan-oracle decompose gt-35x --auto-create

# Only show estimates
gt plan-oracle decompose gt-35x --estimate-only
```

**Decomposition Strategies:**
1. Markdown task lists (`- [ ] items`)
2. Numbered steps or phases
3. Type-based templates (epic, feature, convoy)
4. Historical patterns from similar work

### `gt plan-oracle analyze <issue-id>`

Comprehensive analysis including dependencies, complexity, and risks.

```bash
# Full analysis
gt plan-oracle analyze gt-35x

# Only dependency information
gt plan-oracle analyze gt-35x --deps-only

# Only risk assessment
gt plan-oracle analyze gt-35x --risks-only

# JSON output for scripting
gt plan-oracle analyze gt-35x --json
```

**Analysis Includes:**
- Dependency graph (upstream/downstream)
- Complexity assessment (size, technical risk, cross-rig coordination)
- Test coverage needs
- Documentation requirements
- Risk factors with mitigations

### `gt plan-oracle order [<epic-id>]`

Recommend optimal execution order for work items.

```bash
# Order all open work
gt plan-oracle order

# Order work for specific epic
gt plan-oracle order gt-35x

# Assume N parallel workers
gt plan-oracle order gt-35x --parallel 3

# Output as DOT graph
gt plan-oracle order gt-35x --format dot
```

**Considers:**
- Dependencies (must execute in order)
- Critical path (longest chain)
- Parallelization opportunities
- Rig affinity (reduce context switching)
- Risk factors (fail fast on high-risk items)

### `gt plan-oracle estimate <issue-id>`

Estimate effort and complexity for a work item.

```bash
# Basic estimate
gt plan-oracle estimate gt-35x

# Detailed breakdown
gt plan-oracle estimate gt-35x --verbose

# Show historical comparisons
gt plan-oracle estimate gt-35x --historical
```

**Estimation Factors:**
- Historical data (similar work)
- Code complexity metrics
- Dependency count
- Cross-rig coordination requirements
- Description completeness

### `gt plan-oracle visualize <epic-id>`

Generate dependency graph visualization.

```bash
# Text-based ASCII art
gt plan-oracle visualize gt-35x

# GraphViz DOT format
gt plan-oracle visualize gt-35x --format dot > graph.dot
dot -Tpng graph.dot -o graph.png

# Mermaid diagram
gt plan-oracle visualize gt-35x --format mermaid > graph.md
```

## Architecture

```
planoracle/
├── analyzer/           # Analysis engines
│   ├── decomposer.go   # Work breakdown logic
│   ├── dependency.go   # Dependency graph analysis
│   ├── estimator.go    # Effort estimation
│   ├── ordering.go     # Execution order optimization
│   └── risk.go         # Risk assessment
├── models/             # Data models
│   ├── workitem.go     # Enhanced work item model
│   ├── plan.go         # Execution plan representation
│   └── metrics.go      # Historical metrics
├── sources/            # Data source adapters
│   ├── beads.go        # Beads database reader
│   └── metrics.go      # Metrics collector
├── visualizer/         # Graph generation
│   ├── dot.go          # DOT format
│   └── text.go         # Text-based graphs
└── cmd/                # CLI commands
    ├── decompose.go
    ├── analyze.go
    ├── order.go
    ├── estimate.go
    └── visualize.go
```

## Data Models

### WorkItem

Enhanced work item with planning metadata:

```go
type WorkItem struct {
    // Core fields from beads.Issue
    ID          string
    Title       string
    Type        string
    Status      string

    // Planning metadata
    Complexity    ComplexityScore
    EstimatedDays float64
    ActualDays    float64
    RigAffinity   []string

    // Dependencies
    Dependencies []Dependency
    BlockedBy    []string

    // Risk assessment
    RiskLevel   RiskLevel
    RiskFactors []RiskFactor
}
```

### ExecutionPlan

Recommended execution order:

```go
type ExecutionPlan struct {
    Phases         []Phase
    CriticalPath   []string
    Parallelizable []ParallelSet
    TotalEstimate  float64
    RiskProfile    RiskProfile
}
```

## Data Sources

### Beads Database

Primary source: `.beads/issues.jsonl`

**Extracts:**
- Issue metadata (type, status, priority)
- Dependency relationships
- Parent-child hierarchies
- Historical completion times
- Assignee information (rig affinity)

### Historical Metrics

Collected from completed work items.

**Metrics:**
- Type averages (task, feature, epic)
- Rig velocity (average days per rig)
- Complexity patterns
- Estimate accuracy

**Collection:**
```bash
# Manually trigger metrics collection
gt plan-oracle collect-metrics
```

Metrics are stored in `.beads/metrics/plan-oracle-metrics.json`

## Algorithms

### Work Decomposition

1. Extract structure from description (task lists, steps)
2. Apply type-specific templates if no explicit structure
3. Analyze similar historical work for patterns
4. Estimate each subtask using heuristics
5. Generate task breakdown with dependencies

### Dependency Analysis

1. Load dependency graph from beads
2. Analyze transitive dependencies (BFS)
3. Detect cycles (error if found)
4. Classify dependencies (critical, important, nice-to-have)
5. Calculate risk metrics (cascade potential)

### Effort Estimation

1. Gather estimation signals:
   - Historical data (similar work)
   - Structural complexity (description, dependencies)
   - Code complexity (files touched, cyclomatic complexity)
2. Calculate base estimate (weighted average)
3. Apply adjustment factors (new domain, cross-rig, testing burden)
4. Generate confidence level and range

### Execution Ordering

1. Build dependency graph
2. Validate (detect cycles)
3. Topological sort for valid ordering
4. Identify critical path (longest chain)
5. Find parallelizable sets (no dependencies)
6. Optimize within phases (high-risk first, rig affinity grouping)

## Integration

### With Beads

Plan-oracle reads from beads database and can create tasks:

```bash
# Decompose and create tasks
gt plan-oracle decompose gt-35x --auto-create

# Internally calls:
bd create --title "..." --parent gt-35x --type task
```

### With plan-to-epic (gt-z4g)

Combined workflow:

```bash
# Convert planning doc to epic
gt plan-to-epic docs/plan.md --epic-id gt-abc

# Decompose the epic
gt plan-oracle decompose gt-abc --auto-create

# Generate execution plan
gt plan-oracle order gt-abc
```

### With Formulas

Use in formula workflow:

```toml
[[steps]]
id = "plan-work"
title = "Plan epic execution"
description = """
Use plan-oracle to analyze and order work.

gt plan-oracle decompose {{epic_id}} --auto-create
gt plan-oracle order {{epic_id}} > execution-plan.md
gt plan-oracle analyze {{epic_id}} --risks-only > risks.md
"""
```

## Implementation Phases

### Phase 1: Foundation (Complete)
- ✅ Data models defined
- ✅ Beads database reader
- ✅ Basic CLI structure
- ✅ Metrics collector

### Phase 2: Decomposition (In Progress)
- ✅ Description parser
- ✅ Template-based decomposition
- ⏳ Historical pattern matching
- ⏳ Full CLI implementation

### Phase 3: Dependency Analysis (Planned)
- ⏳ Graph builder
- ⏳ Cycle detection
- ⏳ Risk assessment

### Phase 4: Effort Estimation (Planned)
- ⏳ Historical metrics integration
- ⏳ Estimation algorithm
- ⏳ Confidence calculation

### Phase 5: Execution Ordering (Planned)
- ⏳ Topological sort
- ⏳ Critical path identification
- ⏳ Phase generation

### Phase 6: Visualization (Planned)
- ⏳ Text graph rendering
- ⏳ DOT format generation
- ⏳ Mermaid support

### Phase 7: Polish (Planned)
- ⏳ Comprehensive testing
- ⏳ Error handling
- ⏳ Documentation
- ⏳ Example workflows

## Testing

### Unit Tests

```bash
# Run all plan-oracle tests
go test ./internal/planoracle/...

# Run specific analyzer tests
go test ./internal/planoracle/analyzer/...
```

### Integration Tests

```bash
# Test with real beads database
cd testdata/sample-repo
gt plan-oracle decompose test-epic-001
gt plan-oracle analyze test-feature-002
```

## Examples

### Example 1: Decompose Plugin Implementation

```bash
# Epic: gt-35x - Implement plan-oracle plugin
gt plan-oracle decompose gt-35x --auto-create
```

**Output:**
```
Epic: gt-35x - Plan-oracle plugin
├─ Task: Design data models and architecture (2d, P2)
├─ Task: Implement decomposer engine (3d, P2)
├─ Task: Implement dependency analyzer (2d, P2)
├─ Task: Implement effort estimator (2d, P2)
├─ Task: Implement execution ordering (2d, P2)
├─ Task: Implement visualization (1d, P3)
├─ Task: Write CLI commands (2d, P2)
├─ Task: Write tests and documentation (1d, P3)
└─ Total: 15 days, 8 tasks

Create all tasks? [y/N]: y
Creating tasks...
✓ Created gt-35x-1: Design data models
✓ Created gt-35x-2: Implement decomposer
✓ Created gt-35x-3: Implement dependency analyzer
...
```

### Example 2: Analyze Complexity

```bash
gt plan-oracle analyze gt-35x
```

**Output:**
```
Analysis: gt-35x - Plan-oracle plugin

Complexity Assessment:
  Size: L (Large - 15d estimated)
  Technical Risk: Medium (graph algorithms, pattern matching)
  Cross-Rig Risk: Low (gastown only)
  Testing Burden: Medium (unit + integration tests)
  Overall: 3.2/5.0

Risk Factors:
  ⚠ MEDIUM: Complex graph algorithms may have edge cases
      Mitigation: Comprehensive test coverage
  ⚠ MEDIUM: Historical metrics may be sparse initially
      Mitigation: Start with basic heuristics

Recommendations:
  - Start with decomposer (least dependencies)
  - Build incrementally, test each component
  - Consider parallel work on visualizer
```

### Example 3: Execution Plan

```bash
gt plan-oracle order gt-35x --parallel 2
```

**Output:**
```
Execution Plan for Epic: gt-35x

Phase 1: Foundation (5d)
  ✓ gt-35x-1: Design data models (2d) [Critical Path]
  ∥ gt-35x-6: Visualizer (1d) [Parallel]

Phase 2: Core Engines (7d)
  ✓ gt-35x-2: Decomposer (3d) [Critical Path]
  ∥ gt-35x-3: Dependency analyzer (2d)
  ∥ gt-35x-4: Estimator (2d)

Phase 3: Integration (4d)
  ✓ gt-35x-5: Ordering (2d) [Critical Path]
  ✓ gt-35x-7: CLI commands (2d) [Critical Path]

Phase 4: Polish (1d)
  ✓ gt-35x-8: Tests and docs (1d)

Critical Path: 1 → 2 → 5 → 7 → 8 (14d)
With 2 workers: 15d → 10d (parallelization savings: 5d)
```

## Future Enhancements

### Machine Learning
- Learn from historical data
- Improve estimation accuracy over time
- Detect successful decomposition patterns

### Interactive Mode
- Wizard-style decomposition
- User confirmation/editing of suggestions
- Auto-create with review

### Continuous Monitoring
- Watch mode for tracking execution progress
- Real-time critical path updates
- Estimate vs actual tracking

### External Integrations
- Export to project management tools
- Import estimates from external sources
- Bidirectional sync

## Contributing

Plan-oracle is part of the gastown ecosystem. See main repository for contribution guidelines.

## License

Copyright © 2026 Gastown Project
