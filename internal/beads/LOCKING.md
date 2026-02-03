# Beads Database Locking Strategy

This document describes the file locking strategy implemented in the beads package to prevent concurrent access corruption.

## Overview

The beads package integrates with the `internal/filelock` package to protect all file operations against concurrent access from multiple processes or goroutines. This prevents data corruption, lost updates, and race conditions when multiple agents or tools access beads databases simultaneously.

## Lock Types

We use two types of locks:

1. **Shared (Read) Locks**: Multiple readers can hold shared locks simultaneously. Used for read-only operations.
2. **Exclusive (Write) Locks**: Only one writer can hold an exclusive lock. Used for write operations and read-modify-write cycles.

## Protected Files

### 1. Redirect Files (`.beads/redirect`)

**Location**: `beads_redirect.go`

**Operations**:
- `ResolveBeadsDir()` - READ lock when reading redirect
- `resolveBeadsDirWithDepth()` - READ lock when reading redirect
- `SetupRedirect()` - WRITE lock when creating redirect
- `cleanBeadsRuntimeFiles()` - WRITE lock using `.cleanup.lock` marker

**Rationale**: Redirect files control routing between worktrees and shared databases. Concurrent reads are safe, but writes must be exclusive to prevent partial writes or corruption.

### 2. Routes File (`.beads/routes.jsonl`)

**Location**: `routes.go`

**Operations**:
- `LoadRoutes()` - READ lock
- `AppendRouteToDir()` - WRITE lock for entire read-modify-write cycle
- `RemoveRoute()` - WRITE lock for entire read-modify-write cycle
- `WriteRoutes()` - WRITE lock with atomic tmp+rename pattern

**Rationale**: Routes control issue prefix routing to beads directories. Read-modify-write operations (append, remove) must hold exclusive locks to prevent lost updates. The helper functions `loadRoutesUnsafe()` and `writeRoutesUnsafe()` are used within locked contexts to avoid nested locking.

**Atomic Writes**: Uses temp file + rename pattern to ensure atomic updates:
```go
tmpPath := routesPath + ".tmp"
// write to tmpPath
os.Rename(tmpPath, routesPath)
```

### 3. Molecule Catalog (`.beads/molecules.jsonl`)

**Location**: `catalog.go`

**Operations**:
- `LoadFromFile()` - READ lock
- `SaveToFile()` - WRITE lock with atomic tmp+rename

**Rationale**: Molecule catalogs are read frequently by polecats and modified rarely. Shared read locks allow concurrent reads. Writes use atomic tmp+rename to prevent partial writes.

### 4. Audit Log (`.beads/audit.log`)

**Location**: `audit.go`

**Operations**:
- `LogDetachAudit()` - WRITE lock for append

**Rationale**: Audit logs use append-only JSONL format. Each append operation must be atomic to prevent interleaved writes from corrupting the log structure.

### 5. Template Files (`.beads/templates/*.template.toml`)

**Location**: `template.go`

**Operations**:
- `LoadTemplate()` - READ lock
- `ListTemplates()` - READ lock using `.list.lock` marker

**Rationale**: Templates are read-only during normal operations. Shared read locks allow concurrent template loads. Directory listings use a marker file for lock coordination since we can't lock directories directly.

### 6. PRIME.md (`.beads/PRIME.md`)

**Location**: `beads.go`

**Operations**:
- `ProvisionPrimeMD()` - WRITE lock with double-check inside lock

**Rationale**: PRIME.md is provisioned once per worktree. The write lock ensures atomic creation and prevents duplicate writes from racing provisioning operations.

## Lock Patterns

### Pattern 1: Simple Read

```go
err := filelock.WithReadLock(path, func() error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    // process data
    return nil
})
```

### Pattern 2: Simple Write (Atomic)

```go
err := filelock.WithWriteLock(path, func() error {
    tmpPath := path + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmpPath, path)
})
```

### Pattern 3: Read-Modify-Write

```go
err := filelock.WithWriteLock(path, func() error {
    // Read current data (no nested lock)
    data := loadUnsafe(path)

    // Modify data
    data = modify(data)

    // Write back atomically (no nested lock)
    return writeUnsafe(path, data)
})
```

### Pattern 4: Double-Check Lock

```go
err := filelock.WithWriteLock(path, func() error {
    // Check again inside lock to avoid race
    if _, err := os.Stat(path); err == nil {
        return nil // Already exists
    }

    return os.WriteFile(path, data, 0644)
})
```

### Pattern 5: Directory-Level Operations

For operations on entire directories (cleanup, listing), we use marker files:

```go
lockPath := filepath.Join(dir, ".operation.lock")
err := filelock.WithWriteLock(lockPath, func() error {
    // Perform directory operations
    return nil
})
```

## Lock Granularity

We use **file-level locking** rather than database-level or directory-level locking because:

1. **Reduced Contention**: Different files can be accessed concurrently
2. **Simpler Reasoning**: Locks map directly to file operations
3. **Better Performance**: No global lock bottleneck
4. **Natural Boundaries**: File operations are natural transaction boundaries

## Lock Duration

Locks are held **only during I/O operations**. We minimize lock duration by:

1. Performing expensive computation outside locks
2. Using `defer unlock()` to ensure prompt release
3. Avoiding network calls or slow operations inside locks
4. Batching multiple operations when possible

Example of minimizing lock duration:

```go
// BAD - holds lock during expensive processing
filelock.WithWriteLock(path, func() error {
    data := processExpensive(input) // Slow!
    return os.WriteFile(path, data, 0644)
})

// GOOD - only lock during I/O
data := processExpensive(input)
filelock.WithWriteLock(path, func() error {
    return os.WriteFile(path, data, 0644)
})
```

## Deadlock Prevention

To prevent deadlocks, we follow these rules:

1. **No Nested Locking**: Never acquire a lock while holding another lock
2. **No Lock Inversion**: If multiple files must be locked, use consistent ordering
3. **Timeout**: All locks have a 30-second timeout to prevent indefinite waits
4. **Helper Functions**: Use `*Unsafe()` helpers inside locked contexts

## Stale Lock Cleanup

The filelock package automatically handles stale locks (from dead processes):

- Lock files include the process PID
- Stale locks are detected and cleaned automatically
- Manual cleanup: `filelock.CleanStaleLocks(lockDir)`

## Testing

### Race Detector

All tests must pass with the race detector:

```bash
go test -race ./internal/beads/
```

### Concurrent Access Tests

See `beads_concurrent_test.go` for tests that verify:

- Multiple readers can access files concurrently
- Writers have exclusive access
- Read-modify-write cycles are atomic
- No data corruption under concurrent load

## Performance Monitoring

The filelock package logs contention warnings when lock acquisition takes >100ms. Monitor logs for:

```
Lock contention on <path>: waited <duration>
```

If you see frequent contention:

1. Review lock duration - are locks held too long?
2. Consider increasing lock granularity
3. Check for unnecessary write locks (could be read locks)
4. Profile to find hot paths

## CLI Operations

**Important**: The `bd` CLI command is NOT protected by filelock at this layer. The CLI has its own internal locking mechanisms. Filelock only protects direct file operations in Go code (redirects, routes, catalogs, etc.).

## Migration Notes

When adding new file operations to beads:

1. Identify if operation is read, write, or read-modify-write
2. Wrap with appropriate lock type
3. Use atomic write pattern (tmp+rename) for writes
4. Add test coverage for concurrent access
5. Document locking strategy in this file
6. Test with `-race` flag

## See Also

- [internal/filelock/INTEGRATION.md](/Users/ericfriday/gt/internal/filelock/INTEGRATION.md) - Integration guide
- [internal/filelock/README.md](/Users/ericfriday/gt/internal/filelock/README.md) - Package documentation
- [internal/filelock/filelock.go](/Users/ericfriday/gt/internal/filelock/filelock.go) - Implementation
