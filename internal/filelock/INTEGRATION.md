# File Locking Integration Guide

This guide explains how to integrate file locking into existing Gas Town components to prevent concurrent access issues.

## Quick Reference

```go
// Protect a read operation
err := filelock.WithReadLock(path, func() error {
    data, _ := os.ReadFile(path)
    return process(data)
})

// Protect a write operation
err := filelock.WithWriteLock(path, func() error {
    return os.WriteFile(path, data, 0644)
})
```

## Critical Files Requiring Locking

### 1. Beads Database Files

Location: `.beads/`

Files to protect:
- `issues.jsonl` - Issue database (append-only log)
- `sync_base.jsonl` - Sync state
- `interactions.jsonl` - Interaction history
- `routes.jsonl` - Route definitions
- `beads.db*` - SQLite database files (if used)

### 2. State Files

Location: `~/.local/state/gastown/`

Files to protect:
- `state.json` - Global state (already protected)
- `registry.json` - Agent registry
- `daemon.lock` - Daemon lock file

### 3. Queue Files

Location: `.gastown/queues/`

Files to protect:
- `tasks.jsonl` - Task queue
- `events.jsonl` - Event log

### 4. Configuration Files

Location: `~/.config/gastown/`

Files to protect:
- `config.json` - User configuration
- `plugins.json` - Plugin registry

## Integration Patterns

### Pattern 1: JSONL Append (Issues, Events, Logs)

**Problem**: Multiple processes appending to the same JSONL file can interleave writes or corrupt data.

**Solution**: Use write lock for appends, read lock for reads.

```go
// In internal/beads/beads.go or similar

import "github.com/steveyegge/gastown/internal/filelock"

// AppendIssue adds a new issue to issues.jsonl
func AppendIssue(issue Issue) error {
    path := filepath.Join(beadsDir, "issues.jsonl")

    return filelock.WithWriteLock(path, func() error {
        f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return err
        }
        defer f.Close()
        return json.NewEncoder(f).Encode(issue)
    })
}

// ReadIssues reads all issues from issues.jsonl
func ReadIssues() ([]Issue, error) {
    path := filepath.Join(beadsDir, "issues.jsonl")
    var issues []Issue

    err := filelock.WithReadLock(path, func() error {
        f, err := os.Open(path)
        if err != nil {
            if os.IsNotExist(err) {
                return nil // Empty list
            }
            return err
        }
        defer f.Close()

        scanner := bufio.NewScanner(f)
        for scanner.Scan() {
            var issue Issue
            if err := json.Unmarshal(scanner.Bytes(), &issue); err != nil {
                return err
            }
            issues = append(issues, issue)
        }
        return scanner.Err()
    })

    return issues, err
}
```

### Pattern 2: JSON State Files (Registry, Config)

**Problem**: Read-modify-write cycles can race, losing updates.

**Solution**: Use write lock for entire read-modify-write cycle.

```go
// In internal/daemon/registry.go or similar

// UpdateAgentStatus updates an agent's status in the registry
func UpdateAgentStatus(agentID string, status string) error {
    path := filepath.Join(stateDir, "registry.json")

    return filelock.WithWriteLock(path, func() error {
        // Read current registry
        var registry map[string]AgentInfo
        data, err := os.ReadFile(path)
        if err != nil && !os.IsNotExist(err) {
            return err
        }
        if len(data) > 0 {
            if err := json.Unmarshal(data, &registry); err != nil {
                return err
            }
        } else {
            registry = make(map[string]AgentInfo)
        }

        // Modify
        info := registry[agentID]
        info.Status = status
        info.UpdatedAt = time.Now()
        registry[agentID] = info

        // Write back atomically
        newData, _ := json.MarshalIndent(registry, "", "  ")
        tmp := path + ".tmp"
        if err := os.WriteFile(tmp, newData, 0644); err != nil {
            return err
        }
        return os.Rename(tmp, path)
    })
}

// GetAgentStatus reads an agent's status (read-only)
func GetAgentStatus(agentID string) (string, error) {
    path := filepath.Join(stateDir, "registry.json")
    var status string

    err := filelock.WithReadLock(path, func() error {
        data, err := os.ReadFile(path)
        if err != nil {
            if os.IsNotExist(err) {
                return nil
            }
            return err
        }

        var registry map[string]AgentInfo
        if err := json.Unmarshal(data, &registry); err != nil {
            return err
        }

        if info, ok := registry[agentID]; ok {
            status = info.Status
        }
        return nil
    })

    return status, err
}
```

### Pattern 3: Queue Operations (Task Queue, Event Queue)

**Problem**: Concurrent enqueue/dequeue can corrupt queue state.

**Solution**: Use write locks for both operations.

```go
// In internal/mq/queue.go or similar

type TaskQueue struct {
    path string
}

func NewTaskQueue(path string) *TaskQueue {
    return &TaskQueue{path: path}
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task Task) error {
    return filelock.WithWriteLock(q.path, func() error {
        f, err := os.OpenFile(q.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return err
        }
        defer f.Close()
        return json.NewEncoder(f).Encode(task)
    })
}

// Dequeue removes and returns the first task
func (q *TaskQueue) Dequeue() (*Task, error) {
    var task *Task

    err := filelock.WithWriteLock(q.path, func() error {
        // Read all tasks
        data, err := os.ReadFile(q.path)
        if err != nil {
            if os.IsNotExist(err) {
                return nil // Empty queue
            }
            return err
        }

        // Parse tasks
        var tasks []Task
        scanner := bufio.NewScanner(bytes.NewReader(data))
        for scanner.Scan() {
            var t Task
            if err := json.Unmarshal(scanner.Bytes(), &t); err != nil {
                return err
            }
            tasks = append(tasks, t)
        }

        if len(tasks) == 0 {
            return nil // Empty queue
        }

        // Take first task
        task = &tasks[0]

        // Write remaining tasks back
        f, err := os.Create(q.path)
        if err != nil {
            return err
        }
        defer f.Close()

        for i := 1; i < len(tasks); i++ {
            if err := json.NewEncoder(f).Encode(tasks[i]); err != nil {
                return err
            }
        }
        return nil
    })

    return task, err
}

// Peek returns the first task without removing it
func (q *TaskQueue) Peek() (*Task, error) {
    var task *Task

    err := filelock.WithReadLock(q.path, func() error {
        f, err := os.Open(q.path)
        if err != nil {
            if os.IsNotExist(err) {
                return nil
            }
            return err
        }
        defer f.Close()

        scanner := bufio.NewScanner(f)
        if scanner.Scan() {
            var t Task
            if err := json.Unmarshal(scanner.Bytes(), &t); err != nil {
                return err
            }
            task = &t
        }
        return scanner.Err()
    })

    return task, err
}
```

### Pattern 4: Batch Operations

**Problem**: Need to perform multiple operations atomically.

**Solution**: Use manual lock management.

```go
// In internal/beads/batch.go or similar

// BatchUpdateIssues updates multiple issues atomically
func BatchUpdateIssues(updates []IssueUpdate) error {
    issuesPath := filepath.Join(beadsDir, "issues.jsonl")

    // Create lock with custom timeout
    lock := filelock.NewWithOptions(issuesPath, filelock.Options{
        Timeout:  30 * time.Second,
        LockType: filelock.Exclusive,
    })

    // Acquire lock
    if err := lock.Lock(); err != nil {
        return fmt.Errorf("acquiring lock: %w", err)
    }
    defer lock.Unlock()

    // Read all issues
    issues, err := readIssuesUnsafe(issuesPath) // No internal locking
    if err != nil {
        return err
    }

    // Apply all updates
    for _, update := range updates {
        for i := range issues {
            if issues[i].ID == update.IssueID {
                issues[i].Status = update.Status
                issues[i].UpdatedAt = time.Now()
            }
        }
    }

    // Write back all issues
    return writeIssuesUnsafe(issuesPath, issues) // No internal locking
}
```

### Pattern 5: Database Access (SQLite, Beads DB)

**Problem**: SQLite has built-in locking, but file-level locks can prevent corruption during backup/restore.

**Solution**: Use read locks for queries, write locks for transactions.

```go
// In internal/beads/db.go or similar

// QueryIssues executes a read-only query
func QueryIssues(filter string) ([]Issue, error) {
    dbPath := filepath.Join(beadsDir, "beads.db")
    var issues []Issue

    err := filelock.WithReadLock(dbPath, func() error {
        db, err := sql.Open("sqlite3", dbPath)
        if err != nil {
            return err
        }
        defer db.Close()

        rows, err := db.Query("SELECT * FROM issues WHERE " + filter)
        if err != nil {
            return err
        }
        defer rows.Close()

        for rows.Next() {
            var issue Issue
            if err := rows.Scan(&issue.ID, &issue.Title, &issue.Status); err != nil {
                return err
            }
            issues = append(issues, issue)
        }
        return rows.Err()
    })

    return issues, err
}

// UpdateIssue executes a write transaction
func UpdateIssue(issueID string, status string) error {
    dbPath := filepath.Join(beadsDir, "beads.db")

    return filelock.WithWriteLock(dbPath, func() error {
        db, err := sql.Open("sqlite3", dbPath)
        if err != nil {
            return err
        }
        defer db.Close()

        _, err = db.Exec("UPDATE issues SET status = ? WHERE id = ?", status, issueID)
        return err
    })
}
```

## Migration Checklist

For each file that needs protection:

- [ ] Identify all read operations
- [ ] Identify all write operations
- [ ] Identify read-modify-write cycles
- [ ] Wrap reads in `WithReadLock` or `WithWriteLock` (if part of transaction)
- [ ] Wrap writes in `WithWriteLock`
- [ ] Wrap read-modify-write in single `WithWriteLock`
- [ ] Test for race conditions: `go test -race`
- [ ] Test concurrent access scenarios
- [ ] Monitor lock contention in production
- [ ] Document locking strategy in code comments

## Testing for Race Conditions

```bash
# Run tests with race detector
go test -race ./internal/beads/
go test -race ./internal/state/
go test -race ./internal/mq/

# Run specific race condition tests
go test -race -run TestConcurrent ./internal/beads/
```

## Performance Monitoring

Add logging to detect lock contention:

```go
import "time"

func monitoredUpdate(path string, fn func() error) error {
    start := time.Now()
    lock := filelock.New(path)

    if err := lock.Lock(); err != nil {
        return err
    }

    waitTime := time.Since(start)
    if waitTime > 100*time.Millisecond {
        log.Printf("Lock contention on %s: waited %v", path, waitTime)
    }

    defer lock.Unlock()
    return fn()
}
```

## Common Mistakes

### 1. Holding Locks Too Long

```go
// BAD - holds lock during slow I/O
filelock.WithWriteLock(path, func() error {
    data := fetchFromNetwork() // Slow!
    return os.WriteFile(path, data, 0644)
})

// GOOD - minimize lock duration
data := fetchFromNetwork()
filelock.WithWriteLock(path, func() error {
    return os.WriteFile(path, data, 0644)
})
```

### 2. Forgetting to Unlock

```go
// BAD - no defer, may leak lock on panic
lock.Lock()
doWork()
lock.Unlock()

// GOOD - defer ensures unlock
lock.Lock()
defer lock.Unlock()
doWork()
```

### 3. Nested Locking (Deadlock Risk)

```go
// BAD - can deadlock with different ordering
lockA.Lock()
lockB.Lock()
// ... vs ...
lockB.Lock()
lockA.Lock()

// GOOD - consistent ordering (alphabetical)
locks := []string{pathA, pathB}
sort.Strings(locks)
for _, path := range locks {
    // Lock in consistent order
}
```

### 4. Using Wrong Lock Type

```go
// BAD - exclusive lock for read
filelock.WithWriteLock(path, func() error {
    data, _ := os.ReadFile(path) // Just reading!
    return process(data)
})

// GOOD - shared lock for read
filelock.WithReadLock(path, func() error {
    data, _ := os.ReadFile(path)
    return process(data)
})
```

## Troubleshooting

### Lock Timeout

If you see timeout errors:

1. Check for processes holding locks: `lsof .gastown/locks/*.lock`
2. Increase timeout: `Options{Timeout: 60 * time.Second}`
3. Check for deadlocks: Review lock ordering
4. Consider lock-free alternatives for high-contention operations

### Stale Locks

If locks aren't being released:

1. Ensure defer is used: `defer lock.Unlock()`
2. Check for panic/crashes that skip cleanup
3. Run stale lock cleanup: `filelock.CleanStaleLocks(lockDir)`
4. Add signal handlers to release locks on shutdown

### Performance Issues

If locking slows down operations:

1. Reduce lock duration: Do slow work outside locks
2. Use read locks when possible: Multiple readers don't conflict
3. Increase lock granularity: More fine-grained locks
4. Consider lock-free data structures for hot paths

## See Also

- [README.md](README.md) - Package overview and quick start
- [doc.go](doc.go) - Comprehensive API documentation
- [examples_test.go](examples_test.go) - Working code examples
- [filelock_test.go](filelock_test.go) - Test suite with race condition tests
