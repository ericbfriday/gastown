# Merge Oracle Plugin Design (gt-pio)

## Overview

The merge-oracle plugin provides intelligent analysis of the merge queue (refinery) to help decision-making about merge safety, conflict prediction, test coverage, and merge timing.

## Architecture

### Components

1. **CLI Commands** (`internal/cmd/merge_oracle.go`)
   - `gt merge-oracle analyze <branch>` - Analyze merge safety for a branch
   - `gt merge-oracle queue` - Show merge queue with risk scores
   - `gt merge-oracle conflicts` - Predict potential conflicts
   - `gt merge-oracle recommend` - Recommend merge order

2. **Analysis Engine** (`internal/mergeoracle/`)
   - `analyzer.go` - Core analysis logic
   - `conflict.go` - Conflict detection and prediction
   - `coverage.go` - Test coverage analysis
   - `risk.go` - Risk scoring algorithms
   - `types.go` - Data structures

3. **Data Sources**
   - Git history and diff analysis
   - Refinery merge queue state (from beads issues)
   - Test results (from CI/test output)
   - File change patterns

### Design Principles

1. **Non-blocking**: Advisory only, never prevents merges
2. **Fast**: Analysis should complete in <5 seconds
3. **Accurate**: Minimize false positives while catching real issues
4. **Actionable**: Provide specific recommendations, not just scores
5. **ZFC-compliant**: Derive state from git/beads, no persistent state files

## Risk Scoring Algorithm

### Risk Score (0-100)

**Lower scores = safer to merge**

```
Risk Score = Base(20)
           + ConflictRisk(0-30)
           + TestRisk(0-20)
           + SizeRisk(0-15)
           + DependencyRisk(0-10)
           + HistoryRisk(0-5)
```

### Factors

#### 1. Conflict Risk (0-30 points)
- **File overlap with pending merges**: +10 per overlapping MR
- **Recent conflicts in same files**: +5 per recent conflict
- **Target branch divergence**: +5 if target >10 commits ahead
- **Merge base age**: +5 if >1 week old

#### 2. Test Risk (0-20 points)
- **No test changes**: +10 if code changes without tests
- **Failing tests**: +20 if tests currently failing
- **Low coverage delta**: +5 if coverage decreases
- **Flaky test history**: +5 if touched files have flaky tests

#### 3. Size Risk (0-15 points)
- **Lines changed**: +5 if >500 lines, +10 if >1000, +15 if >2000
- **Files changed**: +5 if >20 files, +10 if >50
- **Scope spread**: +5 if changes span >3 subsystems

#### 4. Dependency Risk (0-10 points)
- **Blocks other MRs**: +5 if other MRs depend on this
- **Depends on failing MRs**: +10 if blocked by failing MR
- **Convoy fragmentation**: +5 if convoy partially merged

#### 5. History Risk (0-5 points)
- **Author merge failure rate**: +5 if >30% recent merges failed
- **File change velocity**: +5 if files changed frequently (>5x/day)

### Risk Categories

- **0-30**: 🟢 Low risk - Safe to merge anytime
- **31-50**: 🟡 Medium risk - Merge with caution, review recommendations
- **51-70**: 🟠 High risk - Address concerns before merging
- **71-100**: 🔴 Critical risk - Do not merge, significant issues

## Conflict Detection

### Static Analysis
1. **File overlap detection**: Compare changed files across pending MRs
2. **Hunk proximity**: Detect changes to nearby lines in same file
3. **Dependency conflicts**: Detect package.json/go.mod conflicts

### Predictive Analysis
1. **Pattern matching**: Historical conflict patterns in file pairs
2. **Import/dependency graph**: Changes to shared dependencies
3. **Merge base divergence**: Time since common ancestor

### Conflict Types
- **Direct**: Same files modified
- **Semantic**: Related code modified (imports, function signatures)
- **Build**: Package/dependency conflicts
- **Test**: Test-only conflicts (lower risk)

## Test Coverage Analysis

### Metrics
1. **Coverage delta**: Change in line/branch coverage
2. **Untested code**: New code without corresponding tests
3. **Test quality**: Test assertions vs code complexity
4. **Test file ratio**: Test files per source file

### Heuristics
- Look for `*_test.go`, `*.test.ts`, `test_*.py` files
- Parse test output for coverage data (if available)
- Estimate based on file patterns if no coverage data

## Dependency Analysis

### MR Dependencies
1. **Explicit blocking**: Check beads issue `blocked_by` field
2. **Convoy ordering**: Respect convoy task order
3. **File dependencies**: MR depends on files changed in earlier MR

### Detection Methods
- Query beads for merge-request issues
- Parse issue metadata for dependencies
- Analyze git diff for file dependencies

## Timing Recommendations

### Optimal Merge Windows
1. **Queue position**: Based on priority score (from refinery)
2. **Conflict avoidance**: Merge before/after conflicting MRs
3. **Author availability**: Avoid merging when author offline
4. **CI load**: Avoid merging during high CI activity

### Heuristics
- **Morning merges**: Better for visibility, author available
- **After convoy completion**: Wait for related work
- **Before long branches**: Merge frequently to reduce drift

## Output Format

### Risk Report Structure
```
Branch: feature/new-api
Risk Score: 35/100 (🟡 Medium Risk)

Concerns:
  🟠 Conflict Risk (15pts): 2 pending MRs modify overlapping files
  🟡 Size Risk (10pts): 450 lines changed across 15 files
  🟢 Test Risk (0pts): Good test coverage
  🟢 Dependency Risk (0pts): No blocking dependencies

Recommendations:
  1. Merge after gt-abc123 (conflicts with internal/api/handler.go)
  2. Consider splitting large refactor into smaller MRs
  3. Monitor test results before merging

Optimal Merge Window: After gt-abc123 merges (est. 2 hours)
```

### Queue Display Format
```
Merge Queue (3 pending)

Pos  MR          Risk  Branch              Files  Age
---  ----------  ----  ------------------  -----  ---
1    gt-abc123   25    feature/auth        8      2h
2    gt-def456   45    refactor/handlers   32     5h  ⚠ Conflicts
3    gt-ghi789   20    fix/logging         3      1h
```

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Create `internal/mergeoracle/` package structure
- [ ] Implement risk scoring algorithm
- [ ] Basic conflict detection (file overlap)
- [ ] CLI command structure

### Phase 2: Git Integration
- [ ] Git diff analysis
- [ ] Merge base detection
- [ ] File change history
- [ ] Author statistics

### Phase 3: Refinery Integration
- [ ] Query merge queue from beads
- [ ] Parse MR metadata
- [ ] Dependency detection
- [ ] Queue priority analysis

### Phase 4: Advanced Analysis
- [ ] Test coverage estimation
- [ ] Predictive conflict detection
- [ ] Timing recommendations
- [ ] Historical pattern matching

### Phase 5: Polish
- [ ] Rich CLI output formatting
- [ ] JSON output mode
- [ ] Performance optimization
- [ ] Documentation

## Data Structures

### MRAnalysis
```go
type MRAnalysis struct {
    MR             *refinery.MRInfo
    RiskScore      int
    RiskLevel      RiskLevel
    ConflictRisk   ConflictRisk
    TestRisk       TestRisk
    SizeRisk       SizeRisk
    DependencyRisk DependencyRisk
    HistoryRisk    HistoryRisk
    Recommendations []Recommendation
    OptimalWindow   *MergeWindow
}
```

### ConflictPrediction
```go
type ConflictPrediction struct {
    WithMR      string
    Files       []string
    Severity    ConflictSeverity
    Confidence  float64
    Suggestions []string
}
```

## Testing Strategy

### Unit Tests
- Risk scoring algorithm accuracy
- Conflict detection precision/recall
- Edge cases (empty queue, single MR)

### Integration Tests
- Real git repository analysis
- Beads query integration
- Queue ordering verification

### Manual Testing
- Run on actual merge queues
- Validate predictions against real conflicts
- Tune scoring weights based on outcomes

## Performance Considerations

1. **Cache git operations**: Memoize expensive git commands
2. **Parallel analysis**: Analyze multiple MRs concurrently
3. **Incremental updates**: Only re-analyze changed MRs
4. **Bounded history**: Limit historical analysis to last 30 days

## Future Enhancements

1. **Machine learning**: Train on historical conflict/merge data
2. **CI integration**: Parse actual test results
3. **Real-time monitoring**: Watch refinery and auto-alert
4. **Web dashboard**: Visual merge queue status
5. **Slack notifications**: Alert on high-risk merges
6. **Auto-reordering**: Suggest queue reordering for optimal throughput

## Success Metrics

1. **Prediction accuracy**: >80% conflict prediction accuracy
2. **False positive rate**: <10% false conflict warnings
3. **Analysis speed**: <5s for typical merge queue
4. **User adoption**: Used in 50%+ of manual merge decisions
5. **Merge failure reduction**: 30% fewer failed merges after implementation
