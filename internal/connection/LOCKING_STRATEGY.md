# MachineRegistry Locking Strategy

## Overview

The `MachineRegistry` uses a two-level locking strategy to protect against both multi-process and multi-threaded concurrent access:

1. **File-level locks** (via `internal/filelock`) - Protects against concurrent access from multiple processes
2. **In-memory mutex** (`sync.RWMutex`) - Protects against concurrent access from multiple goroutines within the same process

## Lock Hierarchy

**File Lock > Memory Mutex**

- File locks must be acquired first
- Memory mutex operations happen within file-locked sections
- This prevents deadlocks and ensures consistent ordering

## Operations

### Read Operations

**Get(), List()**: Read-only operations that don't touch disk
- Use memory RLock only
- No file lock needed (reads from cached in-memory state)

**load()**: Loads registry from disk
- Acquires file Read lock
- Updates memory under Write lock
- Used only during initialization

### Write Operations

**Add(), Remove()**: Modify registry
- Acquire file Write lock
- Reload from disk (for multi-process safety)
- Modify in-memory state under mutex
- Save back to disk atomically
- This implements a read-modify-write transaction

## Multi-Process Safety

The file-level locking ensures that when multiple processes attempt to modify the registry:

1. Only one process can hold the write lock at a time
2. Each process reloads the latest state before modifying
3. Changes are written atomically via temp file + rename
4. Lost updates are prevented

## Multi-Thread Safety

The in-memory mutex ensures that within a single process:

1. Multiple goroutines can read concurrently (RLock)
2. Only one goroutine can write at a time (Lock)
3. Writes block all reads and other writes
4. Reads from cached state are consistent

## Example Scenarios

### Scenario 1: Concurrent Add from Same Process
```
Goroutine A: filelock.WithWriteLock -> reload -> modify -> save
Goroutine B: filelock.WithWriteLock (blocks) -> reload -> modify -> save
```

### Scenario 2: Concurrent Add from Different Processes
```
Process A: filelock.WithWriteLock -> reload -> modify -> save
Process B: filelock.WithWriteLock (blocks at OS level) -> reload -> modify -> save
```

### Scenario 3: Read During Write
```
Goroutine A: filelock.WithWriteLock -> reload (mu.Lock) -> modify (mu.Lock)
Goroutine B: Get() (mu.RLock) -> blocks until A releases mu
```

## Atomic Writes

All writes use the temp-file-and-rename pattern:
1. Write to `machines.json.tmp`
2. `rename(machines.json.tmp, machines.json)`
3. Atomic at filesystem level - readers never see partial writes

## Lock Files

File locks are stored in `.gastown/locks/` relative to the registry file directory.
Lock files are automatically cleaned up on normal exit.
Stale locks (from crashed processes) can be cleaned with `filelock.CleanStaleLocks()`.

## Performance Considerations

- **Reads are fast**: Use only in-memory RLock, no disk I/O
- **Writes reload from disk**: Necessary for multi-process safety, but adds overhead
- **File locks use flock/LockFileEx**: OS-level primitives, efficient
- **Lock contention**: High write contention may cause timeouts (default 30s)

## Testing

Concurrent access is tested in `registry_concurrent_test.go`:
- TestMachineRegistryConcurrentAdd: Multiple goroutines adding machines
- TestMachineRegistryConcurrentReadWrite: Mixed read/write workload
- TestMachineRegistryConcurrentRemove: Concurrent removals
- TestMachineRegistryMultiProcess: Simulates multi-process access
- TestMachineRegistryAtomicWrite: Verifies atomic write behavior
- TestMachineRegistryLockCleanup: Verifies lock cleanup

Run with race detector: `go test -race ./internal/connection/`

## Future Improvements

Potential optimizations if write contention becomes an issue:

1. **Read-through cache**: Cache file mtime, skip reload if unchanged
2. **Optimistic locking**: Version numbers to detect conflicts
3. **Write coalescing**: Batch multiple writes into single file update
4. **Lock-free reads**: Use atomic.Value for lock-free cached reads
