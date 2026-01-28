# Merge Oracle Package

Intelligent analysis of merge queue safety, conflict prediction, and merge timing recommendations.

## Overview

The merge oracle analyzes pending merge requests in the refinery queue to provide:
- **Risk scoring**: Comprehensive risk assessment (0-100 scale)
- **Conflict prediction**: Detect potential conflicts before merging
- **Test coverage analysis**: Ensure adequate testing
- **Dependency tracking**: Identify blocking relationships
- **Timing recommendations**: Optimal merge windows

This is an **advisory tool only** - it never blocks merges, only provides insights.

## Architecture

### Components

```
mergeoracle/
├── types.go      - Data structures and types
├── analyzer.go   - Core analysis engine
└── README.md     - This file
```

### Integration Points

- **Git**: Analyzes diffs, history, and merge bases
- **Refinery**: Queries merge queue state from beads
- **Beads**: Accesses issue metadata and dependencies
- **CLI**: Provides `gt merge-oracle` commands

## Risk Scoring

### Overall Risk Score (0-100)

Lower scores = safer to merge

```
Risk Score = Base(20)
           + ConflictRisk(0-30)
           + TestRisk(0-20)
           + SizeRisk(0-15)
           + DependencyRisk(0-10)
           + HistoryRisk(0-5)
```

### Risk Levels

| Score  | Level    | Icon | Meaning |
|--------|----------|------|---------|
| 0-30   | Low      | 🟢   | Safe to merge anytime |
| 31-50  | Medium   | 🟡   | Merge with caution |
| 51-70  | High     | 🟠   | Address concerns first |
| 71-100 | Critical | 🔴   | Do not merge |

### Risk Factors

#### Conflict Risk (0-30 points)
- File overlap with pending MRs: +10 per overlapping MR
- Recent conflicts in same files: +5 per conflict
- Target branch divergence: +5 if >10 commits ahead
- Merge base age: +5 if >1 week old

#### Test Risk (0-20 points)
- No test changes: +10 if code changes without tests
- Failing tests: +20 if tests currently failing
- Low coverage delta: +5 if coverage decreases
- Flaky test history: +5 if files have flaky tests

#### Size Risk (0-15 points)
- Lines changed: +5 (>500), +10 (>1000), +15 (>2000)
- Files changed: +5 (>20), +10 (>50)
- Scope spread: +5 if >3 subsystems

#### Dependency Risk (0-10 points)
- Blocks other MRs: +5 if others depend on this
- Blocked by failing MRs: +10 if dependency failing
- Convoy fragmentation: +5 if convoy partially merged

#### History Risk (0-5 points)
- Author merge failure rate: +5 if >30% fail
- File change velocity: +5 if files frequently changed

## Usage

### CLI Commands

```bash
# Show merge queue with risk scores
gt merge-oracle queue

# Analyze specific branch
gt merge-oracle analyze feature/new-api

# Predict conflicts
gt merge-oracle conflicts

# Recommend merge order
gt merge-oracle recommend

# JSON output
gt merge-oracle queue --json

# Specific rig
gt merge-oracle queue --rig duneagent

# Verbose output
gt merge-oracle analyze mybranch -v

# Skip certain analyses
gt merge-oracle analyze mybranch --no-conflict --no-test
```

### Programmatic Usage

```go
import "github.com/steveyegge/gastown/internal/mergeoracle"

// Create analyzer
analyzer, err := mergeoracle.NewAnalyzer(repoPath, nil)
if err != nil {
    return err
}

// Analyze single MR
analysis, err := analyzer.AnalyzeMR(mr, queue)
if err != nil {
    return err
}

fmt.Printf("Risk: %d (%s)\n", analysis.RiskScore, analysis.RiskLevel)

// Analyze entire queue
queueAnalysis, err := analyzer.AnalyzeQueue(queue)
if err != nil {
    return err
}

fmt.Printf("Queue health: %s\n", queueAnalysis.QueueHealth.Status)
```

### Configuration

```go
config := &mergeoracle.AnalysisConfig{
    IncludeHistory:    true,   // Enable historical analysis
    IncludeTesting:    true,   // Enable test coverage analysis
    IncludeConflicts:  true,   // Enable conflict prediction
    MaxHistoryDays:    30,     // Limit history to 30 days
    ConflictThreshold: 0.5,    // Minimum confidence for warnings
}

analyzer, err := mergeoracle.NewAnalyzer(repoPath, config)
```

## Data Structures

### MRAnalysis

Complete analysis of a merge request:

```go
type MRAnalysis struct {
    MR              *refinery.MRInfo
    RiskScore       int
    RiskLevel       RiskLevel
    ConflictRisk    ConflictRisk
    TestRisk        TestRisk
    SizeRisk        SizeRisk
    DependencyRisk  DependencyRisk
    HistoryRisk     HistoryRisk
    Conflicts       []ConflictPrediction
    Recommendations []Recommendation
    OptimalWindow   *MergeWindow
    AnalyzedAt      time.Time
}
```

### ConflictPrediction

Predicted conflict with another MR:

```go
type ConflictPrediction struct {
    WithMR      string           // Conflicting MR ID
    WithBranch  string           // Conflicting branch
    Files       []string         // Conflicting files
    Severity    ConflictSeverity // low/medium/high
    Confidence  float64          // 0.0-1.0
    Type        ConflictType     // direct/semantic/build/test
    Suggestions []string         // How to avoid
}
```

### QueueAnalysis

Analysis of entire merge queue:

```go
type QueueAnalysis struct {
    MRs              []*MRAnalysis
    TotalRisk        int
    ConflictPairs    []ConflictPair
    RecommendedOrder []string
    QueueHealth      QueueHealth
    AnalyzedAt       time.Time
}
```

## Conflict Detection

### Methods

1. **Static Analysis**
   - File overlap detection (same files modified)
   - Hunk proximity (nearby lines in same file)
   - Dependency conflicts (package.json, go.mod)

2. **Predictive Analysis**
   - Historical conflict patterns
   - Import/dependency graph changes
   - Merge base divergence

### Conflict Types

- **Direct**: Same files modified (highest risk)
- **Semantic**: Related code modified (imports, signatures)
- **Build**: Package/dependency conflicts
- **Test**: Test-only conflicts (lower risk)

### Severity Levels

- **Low**: <20% file overlap, easy to resolve
- **Medium**: 20-50% overlap, requires review
- **High**: >50% overlap, significant merge work

## Performance

### Optimization Strategies

1. **Caching**: Memoize expensive git operations
2. **Parallel Analysis**: Analyze MRs concurrently
3. **Incremental Updates**: Only re-analyze changed MRs
4. **Bounded History**: Limit to last 30 days

### Expected Performance

- Single MR analysis: <2 seconds
- Queue analysis (10 MRs): <5 seconds
- Large queue (50+ MRs): <15 seconds

## Testing

### Unit Tests

```bash
go test ./internal/mergeoracle
```

Test coverage includes:
- Risk scoring accuracy
- Conflict detection precision
- Edge cases (empty queue, single MR)

### Integration Tests

```bash
go test ./internal/mergeoracle -tags=integration
```

Tests against real git repositories.

## Future Enhancements

### Phase 1 (Current)
- [x] Core risk scoring
- [x] Basic conflict detection
- [x] CLI commands
- [ ] Full git integration
- [ ] Refinery queue query

### Phase 2
- [ ] Test coverage parsing (go test -cover, jest)
- [ ] Historical pattern learning
- [ ] CI integration
- [ ] Real-time monitoring

### Phase 3
- [ ] Machine learning for conflict prediction
- [ ] Web dashboard
- [ ] Slack notifications
- [ ] Auto-reordering suggestions

## Design Principles

1. **Advisory Only**: Never blocks merges, only provides insights
2. **Fast**: Analysis completes in seconds, not minutes
3. **Accurate**: Minimize false positives while catching real issues
4. **Actionable**: Provide specific recommendations, not just scores
5. **ZFC-Compliant**: Derive state from git/beads, no persistent state

## Implementation Status

### Completed
- ✅ Type definitions (types.go)
- ✅ Core analyzer structure (analyzer.go)
- ✅ Risk scoring algorithm
- ✅ Basic conflict detection (file overlap)
- ✅ CLI command structure (merge_oracle.go)
- ✅ Documentation

### In Progress
- ⏳ Git integration (needs git package methods)
- ⏳ Refinery queue query (needs beads integration)

### Planned
- ⏳ Test coverage analysis
- ⏳ Historical pattern analysis
- ⏳ Advanced conflict prediction
- ⏳ Performance optimization

## Example Output

### Queue View

```
● Merge Queue Analysis

Queue Health: 🟡 warning
Total MRs: 3
High Risk: 1
Conflicts: 2
Average Age: 3h

Pos  MR          Risk      Branch              Files  Age
---  ----------  ----      ------------------  -----  ---
1    gt-abc123   🟢 25     feature/auth        8      2h
2    gt-def456   🟠 55     refactor/handlers   32     5h  ⚠
3    gt-ghi789   🟢 20     fix/logging         3      1h

Recommendations:
  • Monitor conflicts and high-risk merges
```

### Analysis View

```
● Merge Analysis: feature/new-api

Risk Score: 🟡 45/100 (Medium Risk)

Risk Breakdown:
  🟡 Conflict Risk: 15/30 pts (2 overlapping MRs, 5 commits divergence)
  🟢 Test Risk: 0/20 pts (Good test coverage)
  🟡 Size Risk: 10/15 pts (450 lines, 15 files)
  🟢 Dependency Risk: 0/10 pts (No blocking dependencies)
  🟢 History Risk: 0/5 pts

Potential Conflicts:
  ⚠ gt-abc123: 2 files (severity: medium, confidence: 80%)

Recommendations:
  1. [conflict] Merge after gt-abc123 (conflicts in 2 files)
  2. [size] Consider splitting large refactor into smaller MRs

Optimal Merge Window:
  Wait for gt-abc123 to merge first
  After: gt-abc123
  Estimated wait: 1h
```

## Contributing

When adding new risk factors or analysis methods:

1. Update risk scoring in `analyzer.go`
2. Add corresponding fields to types in `types.go`
3. Update documentation in this README
4. Add tests for new functionality
5. Update CLI output formatting

## References

- [Design Document](/Users/ericfriday/gt/docs/MERGE-ORACLE-DESIGN.md)
- [Refinery Package](/Users/ericfriday/gt/internal/refinery)
- [Git Package](/Users/ericfriday/gt/internal/git)
- [Beads Package](/Users/ericfriday/gt/internal/beads)
