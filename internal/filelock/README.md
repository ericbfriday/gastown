# File Locking Package

Package `filelock` provides file-based locking for concurrent access protection in Gas Town.

## Overview

This package implements advisory file locks using OS primitives (flock on Unix, LockFileEx on Windows) to prevent race conditions when multiple processes or goroutines access shared state files.

## Features

- **Cross-platform**: Works on Unix, Linux, macOS, and Windows
- **Two lock types**: Exclusive (write) and Shared (read)
- **Automatic retry**: Exponential backoff for lock acquisition
- **Timeout support**: Configurable timeout with multiple modes
- **Stale lock detection**: Automatic cleanup of locks from dead processes
- **Thread-safe**: Safe for concurrent use by multiple goroutines
- **Zero dependencies**: Uses only standard library and golang.org/x/sys

## Quick Start

### Simple Read Lock

```go
import "github.com/steveyegge/gastown/internal/filelock"

// Read a file with protection
err := filelock.WithReadLock("/path/to/data.json", func() error {
    data, err := os.ReadFile("/path/to/data.json")
    if err != nil {
        return err
    }
    // Process data...
    return nil
})
```

### Simple Write Lock

```go
// Write a file with protection
err := filelock.WithWriteLock("/path/to/data.json", func() error {
    return os.WriteFile("/path/to/data.json", data, 0644)
})
```

### Advanced Usage

```go
// Custom timeout and options
lock := filelock.NewWithOptions("/path/to/data.json", filelock.Options{
    Timeout:  5 * time.Second,
    LockType: filelock.Exclusive,
})

if err := lock.Lock(); err != nil {
    return err
}
defer lock.Unlock()

// Perform operations...
```

## Lock Types

### Exclusive Lock (Write)

- Only one process can hold at a time
- Blocks both readers and writers
- Use for write operations

```go
lock := filelock.New(path) // Default is Exclusive
lock.Lock()
defer lock.Unlock()
```

### Shared Lock (Read)

- Multiple processes can hold simultaneously
- Blocks writers but not other readers
- Use for read operations

```go
lock := filelock.NewWithOptions(path, filelock.Options{
    LockType: filelock.Shared,
})
lock.Lock()
defer lock.Unlock()
```

## Timeout Behavior

```go
opts := filelock.Options{
    Timeout: duration,
}
```

- **Positive timeout** (e.g., `5 * time.Second`): Wait up to specified duration
- **Zero timeout**: Wait indefinitely until lock acquired
- **Negative timeout** (e.g., `-1`): Fail immediately if unavailable (try-lock)

## Error Handling

```go
err := lock.Lock()
if err != nil {
    switch {
    case errors.Is(err, filelock.ErrTimeout):
        // Lock acquisition timed out
    case errors.Is(err, filelock.ErrAlreadyLocked):
        // Already holding the lock
    default:
        // Other error
    }
}
```

## Stale Lock Cleanup

```go
// Clean locks from dead processes
cleaned, err := filelock.CleanStaleLocks("/path/to/.gastown/locks")
if err != nil {
    return err
}
fmt.Printf("Cleaned %d stale locks\n", cleaned)
```

## Integration Examples

### Protecting State Files

```go
// internal/state/state.go
func Save(s *State) error {
    return filelock.WithWriteLock(StatePath(), func() error {
        data, _ := json.MarshalIndent(s, "", "  ")
        tmp := StatePath() + ".tmp"
        os.WriteFile(tmp, data, 0600)
        return os.Rename(tmp, StatePath())
    })
}

func Load() (*State, error) {
    var state State
    err := filelock.WithReadLock(StatePath(), func() error {
        data, _ := os.ReadFile(StatePath())
        return json.Unmarshal(data, &state)
    })
    return &state, err
}
```

### Protecting JSONL Files

```go
// Append to issues.jsonl
func AppendIssue(issue Issue) error {
    path := ".beads/issues.jsonl"
    return filelock.WithWriteLock(path, func() error {
        f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            return err
        }
        defer f.Close()
        return json.NewEncoder(f).Encode(issue)
    })
}

// Read all issues
func ReadIssues() ([]Issue, error) {
    path := ".beads/issues.jsonl"
    var issues []Issue

    err := filelock.WithReadLock(path, func() error {
        f, err := os.Open(path)
        if err != nil {
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

### Protecting Registry Operations

```go
// Update registry atomically
func UpdateRegistry(fn func(*Registry) error) error {
    path := ".gastown/registry.json"

    return filelock.WithWriteLock(path, func() error {
        // Read current state
        data, err := os.ReadFile(path)
        if err != nil && !os.IsNotExist(err) {
            return err
        }

        var reg Registry
        if len(data) > 0 {
            if err := json.Unmarshal(data, &reg); err != nil {
                return err
            }
        }

        // Apply changes
        if err := fn(&reg); err != nil {
            return err
        }

        // Write back atomically
        newData, _ := json.MarshalIndent(reg, "", "  ")
        tmp := path + ".tmp"
        if err := os.WriteFile(tmp, newData, 0644); err != nil {
            return err
        }
        return os.Rename(tmp, path)
    })
}
```

## Best Practices

### 1. Always Use Defer

```go
// GOOD
lock.Lock()
defer lock.Unlock()
doWork()

// BAD - may not unlock on panic
lock.Lock()
doWork()
lock.Unlock()
```

### 2. Minimize Lock Duration

```go
// GOOD - fetch data outside lock
data := fetchFromNetwork()
filelock.WithWriteLock(path, func() error {
    return os.WriteFile(path, data, 0644)
})

// BAD - holds lock during slow I/O
filelock.WithWriteLock(path, func() error {
    data := fetchFromNetwork() // Slow!
    return os.WriteFile(path, data, 0644)
})
```

### 3. Use Correct Lock Type

```go
// GOOD - shared lock for reads
filelock.WithReadLock(path, func() error {
    data, _ := os.ReadFile(path)
    return process(data)
})

// BAD - exclusive lock for read-only operation
filelock.WithWriteLock(path, func() error {
    data, _ := os.ReadFile(path)
    return process(data)
})
```

### 4. Consistent Lock Ordering

```go
// GOOD - always acquire in same order
lockA.Lock()
defer lockA.Unlock()
lockB.Lock()
defer lockB.Unlock()

// BAD - different order can deadlock
// Thread 1: A then B
// Thread 2: B then A
```

## Performance Considerations

### Lock Granularity

- **Fine-grained**: Separate locks for different files (better concurrency)
- **Coarse-grained**: Single lock for related files (simpler, less concurrency)

Choose based on your access patterns.

### Retry Backoff

Lock acquisition uses exponential backoff:

- Initial retry: 10ms delay
- After 5 attempts: 50ms delay
- After 10 attempts: 100ms delay (maximum)

Tune with `Options.RetryDelay` if needed.

### Read vs Write Locks

Multiple readers can hold shared locks simultaneously. Use shared locks when possible to maximize parallelism.

## Debugging

### Check Lock Files

Lock files contain the PID of the holding process:

```bash
$ cat .gastown/locks/state.json.lock
12345
```

### Find Lock Holders

```bash
# On Unix/Linux/macOS
$ lsof .gastown/locks/*.lock

# On macOS
$ lsof | grep gastown/locks
```

### Monitor Lock Contention

Add logging to critical sections:

```go
start := time.Now()
err := lock.Lock()
duration := time.Since(start)
if duration > 100*time.Millisecond {
    log.Printf("Lock contention: waited %v for %s", duration, path)
}
defer lock.Unlock()
```

## Testing

The package includes comprehensive tests:

```bash
# Run tests
go test ./internal/filelock/

# Run with race detector
go test -race ./internal/filelock/

# Run specific test
go test -run TestConcurrentLock ./internal/filelock/
```

## Common Patterns

### Transaction-like Updates

```go
func UpdateConfigField(key, value string) error {
    return filelock.WithWriteLock("config.json", func() error {
        // Read
        data, _ := os.ReadFile("config.json")
        var config map[string]string
        json.Unmarshal(data, &config)

        // Modify
        config[key] = value

        // Write
        newData, _ := json.MarshalIndent(config, "", "  ")
        tmp := "config.json.tmp"
        os.WriteFile(tmp, newData, 0644)
        return os.Rename(tmp, "config.json")
    })
}
```

### Batch Operations

```go
func BatchUpdate(updates []Update) error {
    lock := filelock.New("data.json")
    if err := lock.Lock(); err != nil {
        return err
    }
    defer lock.Unlock()

    // Perform multiple updates under single lock
    for _, update := range updates {
        if err := applyUpdate(update); err != nil {
            return err
        }
    }
    return nil
}
```

### Conditional Locking

```go
func MaybeUpdate(path string, shouldUpdate bool) error {
    if !shouldUpdate {
        return nil
    }

    return filelock.WithWriteLock(path, func() error {
        // Only lock if we actually need to update
        return performUpdate(path)
    })
}
```

## Architecture

### Lock File Location

```
project/
├── data.json
└── .gastown/
    └── locks/
        └── data.json.lock
```

Lock files are created in `.gastown/locks/` relative to the data file.

### Custom Lock Directory

```go
lock := filelock.NewWithOptions(path, filelock.Options{
    LockDir: "/var/run/gastown/locks",
})
```

## Limitations

1. **Advisory Locks Only**: Processes must cooperate by using this package. Non-cooperating processes can still access files.

2. **Not Distributed**: Works only on single machine. For distributed locking, use a distributed lock service.

3. **Signal Handling**: Locks are not automatically released on signals. Use signal handlers or defer.

4. **OS Dependencies**: Lock behavior may vary slightly across platforms.

## Migration Guide

To add file locking to existing code:

1. **Identify critical sections** where files are read/written
2. **Wrap in WithReadLock or WithWriteLock**
3. **Test with race detector**: `go test -race`
4. **Monitor for lock contention** in production

Example:

```go
// Before
func UpdateConfig() error {
    data, _ := os.ReadFile("config.json")
    config := parse(data)
    config.Updated = true
    newData, _ := json.Marshal(config)
    return os.WriteFile("config.json", newData, 0644)
}

// After
func UpdateConfig() error {
    return filelock.WithWriteLock("config.json", func() error {
        data, _ := os.ReadFile("config.json")
        config := parse(data)
        config.Updated = true
        newData, _ := json.Marshal(config)
        return os.WriteFile("config.json", newData, 0644)
    })
}
```

## Support

For issues or questions, see the main Gas Town documentation or file an issue.
