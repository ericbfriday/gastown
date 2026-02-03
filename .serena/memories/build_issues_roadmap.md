# Build Issues Roadmap - Post Iteration 4

**Date:** 2026-02-03
**Status:** 11 compilation errors blocking main binary build
**Progress:** 7/14 errors fixed (50%) by refinery agent in Iteration 4
**Estimated Fix Time:** 4-6 hours total

## Current Build Status

```bash
$ go build ./cmd/gt
# FAILS with 11 errors
```

### Error Breakdown

**Category 1: Namespace Conflicts (4 errors)**
- Severity: CRITICAL
- Estimate: 1-2 hours
- Impact: Blocks main binary compilation

**Category 2: Missing Rig API (3 errors)**
- Severity: CRITICAL
- Estimate: 2-3 hours
- Impact: Blocks merge-oracle and names commands

**Category 3: Type Mismatches (3 errors)**
- Severity: HIGH
- Estimate: 1 hour
- Impact: Blocks merge-oracle command

**Category 4: Minor Test Issues (2 errors)**
- Severity: LOW
- Estimate: 30 minutes
- Impact: Test compilation only (not main binary)

**Total:** 11 errors, 4-6 hours estimated fix time

## Detailed Error Reference

### 1. Namespace Conflicts (4 errors)

**Problem:** Multiple files in `internal/cmd` package define the same global variables/functions

#### Error 1a: outputJSON redeclared
```
internal/cmd/mq_list.go:273:6: outputJSON redeclared in this block
    internal/cmd/merge_oracle.go:372:6: other declaration of outputJSON
```

**Files Affected:**
- `internal/cmd/merge_oracle.go` (line 372)
- `internal/cmd/mq_list.go` (line 273)

**Root Cause:** Both files define package-level `var outputJSON bool`

**Fix Options:**
1. **Option A: Unique Names** (Quick - 15 min)
   ```go
   // merge_oracle.go
   var mergeOracleOutputJSON bool
   
   // mq_list.go
   var mqListOutputJSON bool
   ```

2. **Option B: Command-Local** (Better - 30 min)
   ```go
   // In each command's Run function
   var outputJSON bool
   cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
   ```

3. **Option C: Shared Utility** (Best - 1 hour)
   ```go
   // internal/cmd/util.go
   type OutputFormat struct {
       JSON bool
   }
   func NewOutputFormat(cmd *cobra.Command) *OutputFormat { ... }
   ```

**Recommendation:** Start with Option B (command-local), refactor to Option C later

#### Error 1b: formatDuration redeclared
```
internal/cmd/session.go:592:6: formatDuration redeclared in this block
    internal/cmd/merge_oracle.go:565:6: other declaration of formatDuration
```

**Files Affected:**
- `internal/cmd/merge_oracle.go` (line 565)
- `internal/cmd/session.go` (line 592)
- `internal/cmd/boot.go` (also has formatDuration)

**Root Cause:** Same utility function defined in multiple files

**Fix:**
```go
// internal/cmd/util.go (NEW FILE or add to existing)
package cmd

import "time"

// formatDuration formats a duration in human-readable form
func formatDuration(d time.Duration) string {
    if d < time.Second {
        return fmt.Sprintf("%dms", d.Milliseconds())
    }
    if d < time.Minute {
        return fmt.Sprintf("%.1fs", d.Seconds())
    }
    return fmt.Sprintf("%.1fm", d.Minutes())
}
```

Then replace all package-level definitions with calls to this function.

**Estimate:** 20 minutes

#### Error 1c & 1d: statusJSON and statusVerbose redeclared
```
internal/cmd/workspace_status.go:12:2: statusJSON redeclared in this block
    internal/cmd/status.go:28:5: other declaration of statusJSON
    
internal/cmd/workspace_status.go:13:2: statusVerbose redeclared in this block
    internal/cmd/status.go:32:5: other declaration of statusVerbose
```

**Files Affected:**
- `internal/cmd/status.go` (lines 28, 32)
- `internal/cmd/workspace_status.go` (lines 12, 13)

**Root Cause:** Both commands define same flag variables

**Fix:** Make command-local or rename
```go
// workspace_status.go
var workspaceStatusJSON bool
var workspaceStatusVerbose bool

// OR command-local in Run function
var statusJSON, statusVerbose bool
cmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
cmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Verbose output")
```

**Estimate:** 15 minutes

**Total for Category 1:** 1-2 hours (with testing)

### 2. Missing Rig API Functions (3 errors)

**Problem:** Code calls non-existent rig package functions

#### Error 2a: rig.Load() undefined
```
internal/cmd/merge_oracle.go:332:16: undefined: rig.Load
```

**Context:**
```go
// Current code at line 332
r, err := rig.Load(rigEntry.Path)  // rig.Load doesn't exist
```

**Available API:**
```go
// What actually exists in internal/rig/
rig.LoadRigConfig(rigPath string) (*RigConfig, error)
rig.Manager.GetRig(name string) (*Rig, error)
rig.Manager.DiscoverRigs() ([]*Rig, error)
```

**Fix Option A: Implement Missing Function** (Preferred)
```go
// internal/rig/rig.go
// Load loads a rig from the given path
func Load(path string) (*Rig, error) {
    config, err := LoadRigConfig(path)
    if err != nil {
        return nil, err
    }
    return &Rig{
        Name:   filepath.Base(path),
        Path:   path,
        Config: config,
    }, nil
}
```

**Fix Option B: Refactor Caller**
```go
// internal/cmd/merge_oracle.go
mgr := rig.NewManager()
rigs, err := mgr.DiscoverRigs()
if err != nil {
    return err
}
var r *rig.Rig
for _, rig := range rigs {
    if rig.Path == rigEntry.LocalRepo {
        r = rig
        break
    }
}
```

**Estimate:** 30-45 minutes (implement + test)

#### Error 2b: rig.FindFromCwd() undefined
```
internal/cmd/merge_oracle.go:339:16: undefined: rig.FindFromCwd
```

**Context:**
```go
// Current code at line 339
r, err := rig.FindFromCwd()  // rig.FindFromCwd doesn't exist
```

**Fix Option A: Implement Missing Function**
```go
// internal/rig/rig.go
// FindFromCwd finds the rig containing the current working directory
func FindFromCwd() (*Rig, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return nil, err
    }
    return FindRigFromPath(cwd)
}
```

**Fix Option B: Refactor Caller**
```go
// internal/cmd/merge_oracle.go
cwd, err := os.Getwd()
if err != nil {
    return err
}
mgr := rig.NewManager()
rigs, err := mgr.DiscoverRigs()
// Find rig containing cwd...
```

**Estimate:** 30-45 minutes

#### Error 2c: rig.FindRigFromPath() undefined
```
internal/cmd/names.go:405:22: undefined: rig.FindRigFromPath
```

**Context:**
```go
// Current code at line 405
r, err := rig.FindRigFromPath(path)  // Doesn't exist
```

**Fix Option A: Implement Missing Function**
```go
// internal/rig/rig.go
// FindRigFromPath finds the rig containing the given path
func FindRigFromPath(path string) (*Rig, error) {
    mgr := NewManager()
    rigs, err := mgr.DiscoverRigs()
    if err != nil {
        return nil, err
    }
    
    absPath, err := filepath.Abs(path)
    if err != nil {
        return nil, err
    }
    
    for _, r := range rigs {
        if strings.HasPrefix(absPath, r.Path) {
            return r, nil
        }
    }
    
    return nil, fmt.Errorf("no rig found containing path: %s", path)
}
```

**Estimate:** 45 minutes (most complex, needs path traversal logic)

**Total for Category 2:** 2-3 hours (implement all 3 functions + tests)

**Recommended Approach:** Implement all 3 functions in `internal/rig/rig.go` as convenience wrappers around Manager API

### 3. Type Mismatches (3 errors)

#### Error 3a: beads.New() signature mismatch
```
internal/cmd/merge_oracle.go:346:12: assignment mismatch: 2 variables but beads.New returns 1 value
```

**Context:**
```go
// Current code at line 346
b, err := beads.New(r.Path)  // Wrong: beads.New returns 1 value

// Actual signature
func New(workDir string) *Beads  // Returns pointer only, no error
```

**Fix:**
```go
// internal/cmd/merge_oracle.go:346
b := beads.New(r.Path)
// Handle any initialization checks separately if needed
```

**Estimate:** 5 minutes

#### Error 3b: config.RigEntry.Path undefined
```
internal/cmd/merge_oracle.go:332:30: rigEntry.Path undefined (type config.RigEntry has no field or method Path)
```

**Context:**
```go
// Current code
r, err := rig.Load(rigEntry.Path)  // rigEntry has no Path field
```

**Actual RigEntry Structure:**
```go
type RigEntry struct {
    GitURL      string       `json:"git_url"`
    LocalRepo   string       `json:"local_repo,omitempty"`  // Use this!
    AddedAt     time.Time    `json:"added_at"`
    BeadsConfig *BeadsConfig `json:"beads,omitempty"`
}
```

**Fix:**
```go
// internal/cmd/merge_oracle.go:332
r, err := rig.Load(rigEntry.LocalRepo)  // Use LocalRepo instead of Path
```

**Estimate:** 5 minutes

#### Error 3c: GroupAnalysis undefined
```
internal/cmd/merge_oracle.go:34:11: undefined: GroupAnalysis
```

**Context:**
```go
// merge_oracle.go
var mergeOracleCmd = &cobra.Command{
    Use:     "merge-oracle",
    GroupID: GroupAnalysis,  // Undefined constant
    // ...
}
```

**Fix:**
```go
// internal/cmd/root.go (add command group constants)
const (
    GroupCore     = "core"
    GroupAnalysis = "analysis"  // Add this
    GroupWorkflow = "workflow"
    GroupStatus   = "status"
)

// Then in rootCmd.AddCommand, define the groups
rootCmd.AddGroup(&cobra.Group{ID: GroupAnalysis, Title: "Analysis Commands:"})
```

**Estimate:** 10 minutes

**Total for Category 3:** 1 hour (including verification)

### 4. Minor Test Issues (2 errors)

#### Error 4a: Type assertion in errors test
```
internal/errors/examples_test.go:108:2: invalid operation: gitErr (variable of type *GitError) is not an interface
```

**Context:** Type assertion issue in example test

**Fix:**
```go
// internal/errors/examples_test.go:108
// Before (broken)
gitErr.(*GitError)

// After (fixed)
var gitErr *GitError
if errors.As(err, &gitErr) {
    // Use gitErr...
}
```

**Estimate:** 10 minutes

#### Error 4b: Missing import in daemon test
```
internal/daemon/mail_orchestrator_test.go:654:13: undefined: json
```

**Fix:**
```go
// internal/daemon/mail_orchestrator_test.go
import (
    // ... existing imports
    "encoding/json"  // Add this
)
```

**Estimate:** 2 minutes

**Total for Category 4:** 30 minutes

## Implementation Plan

### Phase 1: Quick Wins (1 hour)
**Priority: Start here for fastest progress**

1. Fix beads.New() signature (5 min)
2. Fix config.RigEntry.Path → LocalRepo (5 min)
3. Add GroupAnalysis constant (10 min)
4. Fix missing json import (2 min)
5. Fix type assertion in errors test (10 min)
6. Fix namespace conflicts - make command-local (30 min)

**Result:** 6 errors fixed, 5 remaining

### Phase 2: Rig API Implementation (2-3 hours)
**Priority: Necessary for merge-oracle and names commands**

1. Implement rig.Load() (45 min)
2. Implement rig.FindFromCwd() (45 min)
3. Implement rig.FindRigFromPath() (60 min)
4. Add tests for new functions (30 min)

**Result:** 3 more errors fixed, 2 remaining (namespace conflicts)

### Phase 3: Refactoring (Optional - 1-2 hours)
**Priority: Clean up but not blocking**

1. Extract formatDuration to util.go (20 min)
2. Create OutputFormat utility (30 min)
3. Refactor status flags pattern (30 min)
4. Add integration tests (1 hour)

**Result:** All errors fixed, code quality improved

## Testing Strategy

After each phase:

```bash
# Build verification
go build ./cmd/gt

# Package tests
go test ./internal/cmd/... -v
go test ./internal/rig/... -v
go test ./internal/beads/... -v

# Race detector (after all fixes)
go test -race ./internal/...
```

## Success Criteria

- ✅ Main binary compiles: `go build ./cmd/gt`
- ✅ All package tests pass: `go test ./internal/...`
- ✅ No race conditions: `go test -race ./internal/...`
- ✅ Commands work: `./gt --help`, `./gt version`

## Risk Assessment

**Low Risk Fixes:**
- beads.New() call site - straightforward
- Missing imports - trivial
- GroupAnalysis constant - simple addition

**Medium Risk Fixes:**
- Namespace conflicts - requires careful flag handling
- config.RigEntry.Path access - need to verify LocalRepo is correct field

**Higher Risk Fixes:**
- Rig API functions - need to implement correctly with proper error handling
- May need to understand rig discovery logic thoroughly

## Estimated Timeline

**Conservative (with testing):** 6-8 hours
- Phase 1: 1 hour
- Phase 2: 3 hours
- Testing & verification: 2 hours
- Documentation: 1 hour

**Optimistic (experienced developer):** 4-5 hours
- Phase 1: 45 min
- Phase 2: 2.5 hours
- Testing: 1 hour
- Documentation: 30 min

**Recommended:** 6 hours with proper testing and verification

## Next Session Recommendation

**Option A: Spawn Fix Agents (Ralph Loop Style)**
```
Spawn 3 parallel agents:
1. Quick Wins Agent - Phase 1 fixes
2. Rig API Agent - Phase 2 implementation
3. Test Verification Agent - Comprehensive testing
```

**Option B: Sequential Fix Session**
```
Work through phases 1-2 sequentially
Run tests after each phase
Document fixes as you go
```

**Option C: Hybrid Approach**
```
Phase 1 manually (1 hour, straightforward)
Spawn agents for Phase 2 (rig API implementation)
Verify and test
```

**Recommendation:** Option A (spawn agents) for consistency with Ralph Loop pattern and parallel efficiency

---

**Status:** Roadmap complete, ready for implementation
**Estimated Completion:** 4-6 hours with testing
**Confidence:** 0.95 (straightforward fixes, well-documented)
