# Ralph Loop Pattern Guide

**Created:** 2026-02-03  
**Based On:** Successful 4-iteration autonomous session  
**Status:** Production-proven pattern  

## What is the Ralph Loop?

The Ralph Loop is a self-referential iteration pattern where an AI agent continuously works on the same prompt, seeing its own previous work in files and git history, creating a feedback loop for autonomous improvement.

**Name Origin:** From the Simpsons character Ralph Wiggum, inspired by the self-referential nature of continuously feeding the same prompt while seeing previous iterations.

## Core Mechanism

```
Iteration 1: Prompt → Work → Files changed → Commit
Iteration 2: SAME Prompt → See own files → More work → Commit  
Iteration 3: SAME Prompt → See own commits → Continue → Commit
...repeat until completion promise fulfilled
```

**Key Property:** Each iteration has access to all previous work through:
- Modified files in the working directory
- Git history showing own commits
- Memory files documenting own discoveries
- Task lists showing completed work

## When to Use Ralph Loop

### Ideal Use Cases ✅
1. **Systematic Cleanup:** Fixing multiple related build/test errors
2. **Iterative Refinement:** Improving code quality through multiple passes
3. **Comprehensive Testing:** Test, fix, re-test cycles
4. **Documentation Work:** Multi-pass documentation generation
5. **Refactoring:** Step-by-step code transformation

### Poor Use Cases ❌
1. **Single-step tasks:** No benefit from iteration
2. **Exploratory work:** Needs different approach each iteration
3. **Creative design:** Requires human judgment at each step
4. **Ambiguous goals:** Needs clear completion criteria

## Setup Requirements

### 1. Completion Promise (Critical)
```bash
ralph-loop "Fix all compilation errors"
```

The completion promise defines when to stop. Must be:
- **Specific:** Clear what "done" means
- **Verifiable:** Can check if fulfilled
- **Scoped:** Not infinite (e.g., "make code perfect")

### 2. Ralph Loop Configuration
```bash
# Basic (infinite iterations)
ralph-loop "Fix all build errors"

# With iteration limit
ralph-loop --max-iterations 10 "Improve test coverage"

# With specific completion promise
ralph-loop --completion-promise "All tests pass" "Fix failing tests"
```

### 3. Stop Hook
The ralph loop uses a stop hook that:
- Intercepts normal exit/completion
- Checks completion promise
- Feeds same prompt back if not complete
- Allows exit if promise fulfilled

## Execution Pattern

### Phase 1: Initial Iteration
```
User: "Fix all build errors"
Agent: 
  1. Check build status
  2. Identify errors
  3. Fix systematically  
  4. Commit changes
  5. Document progress
```

### Phase 2: Subsequent Iterations
```
User: [Same prompt via stop hook]
Agent:
  1. See previous fixes in git log
  2. Read own documentation
  3. Check if more errors exist
  4. Fix remaining issues
  5. Commit and document
```

### Phase 3: Completion
```
Agent:
  1. Verify completion promise fulfilled
  2. Create final summary
  3. Output completion message
  4. Loop stops automatically
```

## Best Practices

### Do's ✅

**1. Create Memories After Each Iteration**
```python
# After fixing issues
write_memory("iteration_N_complete", 
  "What was done, what remains, key learnings")
```

**2. Make Clean, Descriptive Commits**
```bash
# Good
git commit -m "fix(build): resolve namespace conflicts in cmd package"

# Bad  
git commit -m "fixes"
```

**3. Test After Each Fix**
```python
# After each change
run_tests()
verify_fix()
document_result()
```

**4. Document Discoveries**
```python
# Capture patterns
write_memory("pattern_discovered",
  "Why it happened, how to prevent, similar issues")
```

**5. Use Task Lists**
```python
# Track progress across iterations
TaskCreate("Fix namespace conflicts")
TaskUpdate(id, "completed")
```

### Don'ts ❌

**1. Don't Mix Unrelated Changes**
```python
# Bad: Mixing build fixes with features
fix_build_error()
add_new_feature()  # Wrong iteration scope
```

**2. Don't Make Large Commits**
```python
# Bad: Everything in one commit
fix_all_21_errors_in_one_commit()

# Good: Logical groupings
fix_namespace_conflicts()  # Commit 1
fix_missing_apis()         # Commit 2
```

**3. Don't Skip Verification**
```python
# Bad: Assuming fix works
fix_error()
move_to_next()  # Didn't verify!

# Good: Verify each fix  
fix_error()
test_fix()
verify_working()
```

**4. Don't Ignore Context**
```python
# Bad: Not checking previous work
start_from_scratch()

# Good: Build on previous iterations
read_previous_memories()
check_git_history()
continue_from_where_left_off()
```

**5. Don't Over-document**
```python
# Bad: Excessive detail
write_memory("every_single_keystroke", ...)

# Good: Key discoveries only
write_memory("root_cause_analysis", ...)
```

## Iteration Structure

### Recommended Flow
```
1. CHECK STATUS
   - Read git log (see previous iterations)
   - Read memories (understand context)
   - Read task list (track progress)

2. ANALYZE REMAINING WORK
   - Identify what's left
   - Prioritize next fixes
   - Plan approach

3. FIX SYSTEMATICALLY
   - Fix one category at a time
   - Test after each fix
   - Document as you go

4. COMMIT CLEANLY
   - Logical groupings
   - Descriptive messages
   - Co-authored attribution

5. DOCUMENT PROGRESS
   - Update memories
   - Update task list
   - Note key discoveries

6. CHECK COMPLETION
   - Verify promise fulfilled?
   - Document final state
   - Prepare summary
```

## Memory Management

### Iteration Memories
Create one memory per iteration:
```
ralph_iteration_5_complete.md
ralph_iteration_6_complete.md  
ralph_iteration_7_complete.md
```

### Summary Memory
Create overall summary:
```
ralph_loop_autonomous_summary.md
```

### Session Memory
Create session record:
```
session_YYYY-MM-DD_ralph_loop_description.md
```

### Pattern Memory
Capture reusable patterns:
```
ralph_loop_pattern_learnings.md
```

## Success Metrics

### Quantitative
- **Completion Rate:** Fulfilled promise? (Yes/No)
- **Error Reduction:** Errors before vs after
- **Test Pass Rate:** Tests passing before vs after
- **Commit Quality:** Number of reverts needed
- **Iteration Efficiency:** Issues fixed per iteration

### Qualitative  
- **Code Quality:** Maintainability improved?
- **Documentation:** Clear for future work?
- **Pattern Recognition:** Learnings captured?
- **Git History:** Professional quality?

## Common Pitfalls

### 1. Infinite Loop
**Problem:** Completion promise never fulfilled  
**Solution:** 
- Use --max-iterations as safety
- Make promise specific and verifiable
- Include "stop if no progress" logic

### 2. Scope Creep
**Problem:** Keeps finding new things to fix  
**Solution:**
- Strict completion promise
- Separate "nice to have" from "must have"
- Document future work separately

### 3. Quality Degradation
**Problem:** Later iterations lower quality  
**Solution:**
- Maintain coding standards throughout
- Re-read style guide each iteration
- Code review own commits

### 4. Context Loss
**Problem:** Forgetting earlier decisions  
**Solution:**
- Read all memories at start
- Document architectural decisions
- Explain "why" not just "what"

### 5. Test Fatigue
**Problem:** Skipping tests in later iterations  
**Solution:**
- Test after EVERY change
- Automate verification
- Make testing non-negotiable

## Example Session

See `session_2026-02-03_ralph_loop_build_fixes` for complete example:
- 4 iterations (5-8)
- 29+ issues fixed
- 4 clean commits
- 100% build success
- Comprehensive documentation

## Tool Integration

### Essential Tools
- `git log` - See previous iterations
- `write_memory` - Document progress
- `read_memory` - Maintain context
- `TaskCreate/TaskUpdate` - Track progress
- `think_about_collected_information` - Self-reflection

### Recommended Tools
- `Bash` for testing
- `Edit` for focused changes
- `Grep/Glob` for searching
- `Read` for understanding

## Completion Detection

### Check Completion Promise
```python
def check_completion():
    if completion_promise == "Fix all build errors":
        result = run("go build ./...")
        return result.success
    
    elif completion_promise == "All tests pass":
        result = run("go test ./...")
        return "FAIL" not in result.output
    
    return False
```

### Output Completion
```python
if check_completion():
    write_memory("final_summary", ...)
    print("✅ Completion promise fulfilled!")
    # Loop stops
else:
    print("Continuing to next iteration...")
    # Loop continues
```

## Advanced Patterns

### Parallel Work
```python
# Fix multiple independent issues
TaskCreate("Fix namespace conflicts")
TaskCreate("Fix API functions")  
TaskCreate("Fix type mismatches")

# Work in parallel where possible
```

### Dependency-Aware
```python
# Fix in dependency order
fix_api_functions()  # Must come first
fix_api_usage()      # Depends on API
fix_tests()          # Depends on both
```

### Progressive Refinement
```python
# Iteration 1: Fix compilation
# Iteration 2: Fix tests
# Iteration 3: Fix warnings
# Iteration 4: Improve quality
```

## Conclusion

The Ralph Loop pattern is highly effective for systematic, autonomous work that benefits from iteration. Key to success:
1. **Clear completion promise**
2. **Systematic approach**  
3. **Clean commits**
4. **Comprehensive documentation**
5. **Self-reflection between iterations**

When used correctly, it enables autonomous work quality matching or exceeding human-guided work, with better documentation and cleaner git history.

---

**Pattern Status:** ✅ Production-proven  
**Success Rate:** 100% (1/1 sessions)  
**Recommended:** Yes, for appropriate use cases  
**Documentation:** Comprehensive
