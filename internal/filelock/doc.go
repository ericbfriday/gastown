// Package filelock provides file-based locking for concurrent access protection.
//
// # Overview
//
// This package implements advisory file locks using OS primitives to prevent
// race conditions when multiple processes or goroutines access shared state files.
// It supports both exclusive locks (for writes) and shared locks (for reads).
//
// # Architecture
//
// Lock files are stored in .gastown/locks/ directories alongside the data files
// they protect. Each data file has a corresponding .lock file:
//
//	data.json           -> .gastown/locks/data.json.lock
//	issues.jsonl        -> .gastown/locks/issues.jsonl.lock
//	state.json          -> .gastown/locks/state.json.lock
//
// # OS Implementation
//
// - Unix/Linux/macOS: Uses flock(2) system call
// - Windows: Uses LockFileEx/UnlockFileEx API
//
// Both provide advisory locks - processes must cooperate by using this package.
//
// # Lock Types
//
// Exclusive Lock:
//   - For write operations
//   - Only one process can hold at a time
//   - Blocks both readers and writers
//
// Shared Lock:
//   - For read operations
//   - Multiple processes can hold simultaneously
//   - Blocks writers but not other readers
//
// # Usage Patterns
//
// Simple read operation:
//
//	err := filelock.WithReadLock("/path/to/data.json", func() error {
//	    data, err := os.ReadFile("/path/to/data.json")
//	    if err != nil {
//	        return err
//	    }
//	    // ... process data ...
//	    return nil
//	})
//
// Simple write operation:
//
//	err := filelock.WithWriteLock("/path/to/data.json", func() error {
//	    return os.WriteFile("/path/to/data.json", data, 0644)
//	})
//
// Advanced usage with custom timeout:
//
//	lock := filelock.NewWithOptions("/path/to/data.json", filelock.Options{
//	    Timeout:  5 * time.Second,
//	    LockType: filelock.Exclusive,
//	})
//	if err := lock.Lock(); err != nil {
//	    return err
//	}
//	defer lock.Unlock()
//	// ... perform operations ...
//
// # Error Handling
//
// The package provides specific errors for different failure modes:
//
//   - ErrTimeout: Lock acquisition timed out
//   - ErrAlreadyLocked: Attempted to lock when already holding lock
//   - ErrNotLocked: Attempted to unlock when not holding lock
//
// Timeout behavior:
//
//   - Positive timeout: Wait up to specified duration
//   - Zero timeout: Wait indefinitely
//   - Negative timeout: Fail immediately if lock unavailable
//
// # Retry and Backoff
//
// Lock acquisition automatically retries with exponential backoff:
//
//   - Initial retry: 10ms delay
//   - After 5 attempts: 50ms delay
//   - After 10 attempts: 100ms delay (maximum)
//
// # Stale Lock Cleanup
//
// Lock files from dead processes are automatically detected and cleaned:
//
//	cleaned, err := filelock.CleanStaleLocks("/path/to/.gastown/locks")
//	fmt.Printf("Cleaned %d stale locks\n", cleaned)
//
// Stale detection works by attempting to acquire the lock - if successful,
// the lock was not held by any process.
//
// # Integration Examples
//
// Protecting state file operations:
//
//	// In internal/state/state.go
//	func Save(s *State) error {
//	    return filelock.WithWriteLock(StatePath(), func() error {
//	        data, _ := json.MarshalIndent(s, "", "  ")
//	        tmp := StatePath() + ".tmp"
//	        os.WriteFile(tmp, data, 0600)
//	        return os.Rename(tmp, StatePath())
//	    })
//	}
//
// Protecting JSONL file appends:
//
//	func AppendIssue(issue Issue) error {
//	    path := ".beads/issues.jsonl"
//	    return filelock.WithWriteLock(path, func() error {
//	        f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	        if err != nil {
//	            return err
//	        }
//	        defer f.Close()
//	        return json.NewEncoder(f).Encode(issue)
//	    })
//	}
//
// Protecting database operations:
//
//	func UpdateRegistry(fn func() error) error {
//	    path := ".gastown/registry.json"
//	    lock := filelock.New(path)
//	    if err := lock.Lock(); err != nil {
//	        return err
//	    }
//	    defer lock.Unlock()
//	    return fn()
//	}
//
// # Performance Considerations
//
// Lock Granularity:
//   - Fine-grained: Separate locks for different files (better concurrency)
//   - Coarse-grained: Single lock for related files (simpler, less concurrency)
//
// Read vs Write:
//   - Use Shared locks for reads when possible (allows parallel reads)
//   - Use Exclusive locks only when necessary (serializes all access)
//
// Lock Duration:
//   - Acquire just before critical section
//   - Release immediately after (use defer for safety)
//   - Avoid holding locks during I/O or network operations when possible
//
// # Thread Safety
//
// FileLock instances are safe for concurrent use by multiple goroutines.
// The internal mutex protects state transitions.
//
// # Platform Notes
//
// Advisory vs Mandatory:
//   - All locks are advisory (require cooperation)
//   - Processes not using filelock can still access files
//   - Use for coordinating known cooperative processes
//
// File Descriptor Management:
//   - Lock files are opened with O_CREATE | O_RDWR
//   - File descriptors are closed on unlock
//   - Lock files are removed on clean unlock (best effort)
//
// Signal Safety:
//   - Locks are NOT automatically released on signal
//   - Use defer unlock() or signal handlers as needed
//   - OS will release locks when process exits
//
// # Common Pitfalls
//
// 1. Forgetting to unlock:
//
//	// BAD - no defer
//	lock.Lock()
//	doWork()
//	lock.Unlock() // May not execute if doWork() panics
//
//	// GOOD - defer ensures unlock
//	lock.Lock()
//	defer lock.Unlock()
//	doWork()
//
// 2. Holding locks too long:
//
//	// BAD - holds lock during slow I/O
//	filelock.WithWriteLock(path, func() error {
//	    data := fetchFromNetwork() // Slow!
//	    return os.WriteFile(path, data, 0644)
//	})
//
//	// GOOD - minimize lock duration
//	data := fetchFromNetwork()
//	filelock.WithWriteLock(path, func() error {
//	    return os.WriteFile(path, data, 0644)
//	})
//
// 3. Deadlock with multiple locks:
//
//	// BAD - can deadlock if different order in different places
//	lockA.Lock()
//	lockB.Lock()
//	// ... vs elsewhere ...
//	lockB.Lock()
//	lockA.Lock()
//
//	// GOOD - consistent lock ordering
//	// Always acquire locks in same order (e.g., alphabetical)
//
// 4. Using wrong lock type:
//
//	// BAD - exclusive lock for read
//	filelock.WithWriteLock(path, func() error {
//	    data, _ := os.ReadFile(path) // Just reading!
//	    return process(data)
//	})
//
//	// GOOD - shared lock for read
//	filelock.WithReadLock(path, func() error {
//	    data, _ := os.ReadFile(path)
//	    return process(data)
//	})
//
// # Debugging
//
// Lock files contain the PID of the holding process for debugging:
//
//	$ cat .gastown/locks/state.json.lock
//	12345
//
// Check which processes are holding locks:
//
//	$ lsof .gastown/locks/*.lock
//	COMMAND   PID USER   FD   TYPE DEVICE SIZE NODE NAME
//	gt      12345 user    3u   REG   1,5    6  123 .gastown/locks/state.json.lock
//
// # Migration Guide
//
// To add file locking to existing code:
//
// 1. Identify critical sections (file reads/writes)
// 2. Wrap in filelock.WithReadLock or WithWriteLock
// 3. Test for race conditions (go test -race)
// 4. Monitor for lock contention in production
//
// Example migration:
//
//	// Before
//	func UpdateConfig() error {
//	    data, _ := os.ReadFile("config.json")
//	    config := parse(data)
//	    config.Updated = true
//	    newData, _ := json.Marshal(config)
//	    return os.WriteFile("config.json", newData, 0644)
//	}
//
//	// After
//	func UpdateConfig() error {
//	    return filelock.WithWriteLock("config.json", func() error {
//	        data, _ := os.ReadFile("config.json")
//	        config := parse(data)
//	        config.Updated = true
//	        newData, _ := json.Marshal(config)
//	        return os.WriteFile("config.json", newData, 0644)
//	    })
//	}
package filelock
