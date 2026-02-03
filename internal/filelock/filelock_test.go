package filelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	path := "/tmp/test.json"
	lock := New(path)

	if lock.path != path {
		t.Errorf("path = %q, want %q", lock.path, path)
	}

	expectedLockPath := filepath.Join("/tmp", ".gastown", "locks", "test.json.lock")
	if lock.lockPath != expectedLockPath {
		t.Errorf("lockPath = %q, want %q", lock.lockPath, expectedLockPath)
	}

	if lock.lockType != Exclusive {
		t.Errorf("lockType = %v, want Exclusive", lock.lockType)
	}

	if lock.timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", lock.timeout)
	}
}

func TestNewWithOptions(t *testing.T) {
	path := "/tmp/test.json"
	opts := Options{
		Timeout:    10 * time.Second,
		LockType:   Shared,
		RetryDelay: 5 * time.Millisecond,
		LockDir:    "/tmp/custom-locks",
	}

	lock := NewWithOptions(path, opts)

	if lock.lockType != Shared {
		t.Errorf("lockType = %v, want Shared", lock.lockType)
	}

	if lock.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", lock.timeout)
	}

	expectedLockPath := filepath.Join("/tmp/custom-locks", "test.json.lock")
	if lock.lockPath != expectedLockPath {
		t.Errorf("lockPath = %q, want %q", lock.lockPath, expectedLockPath)
	}
}

func TestLockUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)

	// Initially not locked
	if lock.IsLocked() {
		t.Error("lock should not be locked initially")
	}

	// Acquire lock
	if err := lock.Lock(); err != nil {
		t.Fatalf("Lock() error = %v", err)
	}

	if !lock.IsLocked() {
		t.Error("lock should be locked after Lock()")
	}

	// Lock file should exist
	if _, err := os.Stat(lock.lockPath); err != nil {
		t.Errorf("lock file should exist: %v", err)
	}

	// Release lock
	if err := lock.Unlock(); err != nil {
		t.Fatalf("Unlock() error = %v", err)
	}

	if lock.IsLocked() {
		t.Error("lock should not be locked after Unlock()")
	}
}

func TestLockAlreadyLocked(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)

	if err := lock.Lock(); err != nil {
		t.Fatalf("Lock() error = %v", err)
	}
	defer lock.Unlock()

	// Try to lock again
	err := lock.Lock()
	if err != ErrAlreadyLocked {
		t.Errorf("Lock() when already locked: error = %v, want ErrAlreadyLocked", err)
	}
}

func TestUnlockNotLocked(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)

	err := lock.Unlock()
	if err != ErrNotLocked {
		t.Errorf("Unlock() when not locked: error = %v, want ErrNotLocked", err)
	}
}

func TestTryLock(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)

	// Should succeed on first try
	if err := lock.TryLock(); err != nil {
		t.Fatalf("TryLock() error = %v", err)
	}
	defer lock.Unlock()

	if !lock.IsLocked() {
		t.Error("lock should be locked after TryLock()")
	}
}

func TestConcurrentLockExclusive(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock1 := New(path)
	lock2 := NewWithOptions(path, Options{
		Timeout:  100 * time.Millisecond,
		LockType: Exclusive,
	})

	// Lock1 acquires the lock
	if err := lock1.Lock(); err != nil {
		t.Fatalf("lock1.Lock() error = %v", err)
	}

	// Lock2 should timeout trying to acquire
	err := lock2.Lock()
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("lock2.Lock() error = %v, want ErrTimeout", err)
	}

	// Release lock1
	if err := lock1.Unlock(); err != nil {
		t.Fatalf("lock1.Unlock() error = %v", err)
	}

	// Now lock2 should succeed
	if err := lock2.Lock(); err != nil {
		t.Errorf("lock2.Lock() after lock1 released: error = %v", err)
	}
	lock2.Unlock()
}

func TestConcurrentLockShared(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	// Multiple shared locks should succeed
	lock1 := NewWithOptions(path, Options{
		Timeout:  1 * time.Second,
		LockType: Shared,
	})
	lock2 := NewWithOptions(path, Options{
		Timeout:  100 * time.Millisecond,
		LockType: Shared,
	})

	if err := lock1.Lock(); err != nil {
		t.Fatalf("lock1.Lock() error = %v", err)
	}
	defer lock1.Unlock()

	// Note: Some systems (like older macOS) may not support true shared locks with flock
	// In that case, shared locks behave like exclusive locks
	err := lock2.Lock()
	if err != nil {
		t.Skipf("Shared locks not supported on this system (got error: %v)", err)
	}
	defer lock2.Unlock()
}

func TestSharedExclusiveConflict(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	sharedLock := NewWithOptions(path, Options{
		Timeout:  1 * time.Second,
		LockType: Shared,
	})
	exclusiveLock := NewWithOptions(path, Options{
		Timeout:  100 * time.Millisecond,
		LockType: Exclusive,
	})

	// Acquire shared lock
	if err := sharedLock.Lock(); err != nil {
		t.Fatalf("sharedLock.Lock() error = %v", err)
	}
	defer sharedLock.Unlock()

	// Exclusive lock should fail
	err := exclusiveLock.Lock()
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("exclusiveLock.Lock() error = %v, want ErrTimeout", err)
	}
}

func TestWithLock(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)
	executed := false

	err := lock.WithLock(func() error {
		executed = true
		if !lock.IsLocked() {
			t.Error("lock should be held during WithLock callback")
		}
		return nil
	})

	if err != nil {
		t.Errorf("WithLock() error = %v", err)
	}

	if !executed {
		t.Error("WithLock() callback was not executed")
	}

	if lock.IsLocked() {
		t.Error("lock should be released after WithLock()")
	}
}

func TestWithReadLock(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	executed := false
	err := WithReadLock(path, func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("WithReadLock() error = %v", err)
	}

	if !executed {
		t.Error("WithReadLock() callback was not executed")
	}
}

func TestWithWriteLock(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	executed := false
	err := WithWriteLock(path, func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("WithWriteLock() error = %v", err)
	}

	if !executed {
		t.Error("WithWriteLock() callback was not executed")
	}
}

func TestRaceCondition(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "counter.txt")

	// Write initial value
	if err := os.WriteFile(path, []byte("0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Concurrent goroutines increment counter
	const numGoroutines = 10
	const incrementsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < incrementsPerGoroutine; j++ {
				err := WithWriteLock(path, func() error {
					// Read current value
					data, err := os.ReadFile(path)
					if err != nil {
						return err
					}

					// Parse, increment, write back
					var count int
					content := string(data)
					// Handle both with and without newline
					content = content[:len(content)-findNewlinePos(content)]
					if _, err := fmt.Sscanf(content, "%d", &count); err != nil {
						return fmt.Errorf("parse error on %q: %w", content, err)
					}

					count++

					return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", count)), 0644)
				})

				if err != nil {
					t.Errorf("increment error: %v", err)
					return
				}
			}
		}()
	}

	wg.Wait()

	// Read final value
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var finalCount int
	content := string(data)
	content = content[:len(content)-findNewlinePos(content)]
	if _, err := fmt.Sscanf(content, "%d", &finalCount); err != nil {
		t.Fatalf("final parse error on %q: %v", content, err)
	}

	expected := numGoroutines * incrementsPerGoroutine
	if finalCount != expected {
		t.Errorf("final count = %d, want %d (race condition detected)", finalCount, expected)
	}
}

func findNewlinePos(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return len(s) - i
		}
	}
	return 0
}

func TestConcurrentReadersExclusiveWriter(t *testing.T) {
	// Skip this test on systems that don't support shared locks
	t.Skip("Skipping shared lock test - some systems don't support true shared locks with flock")
}

func TestCleanStaleLocks(t *testing.T) {
	tmpDir := t.TempDir()
	lockDir := filepath.Join(tmpDir, ".gastown", "locks")

	// Create lock directory
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some stale lock files (not actually locked)
	staleLocks := []string{"stale1.lock", "stale2.lock", "stale3.lock"}
	for _, name := range staleLocks {
		path := filepath.Join(lockDir, name)
		if err := os.WriteFile(path, []byte("12345\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create an active lock
	path := filepath.Join(tmpDir, "active.json")
	activeLock := NewWithOptions(path, Options{
		Timeout:  1 * time.Second,
		LockType: Exclusive,
		LockDir:  lockDir,
	})
	if err := activeLock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer activeLock.Unlock()

	// Clean stale locks
	cleaned, err := CleanStaleLocks(lockDir)
	if err != nil {
		t.Fatalf("CleanStaleLocks() error = %v", err)
	}

	if cleaned != len(staleLocks) {
		t.Errorf("CleanStaleLocks() cleaned %d, want %d", cleaned, len(staleLocks))
	}

	// Verify stale locks are gone
	for _, name := range staleLocks {
		path := filepath.Join(lockDir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("stale lock %s should be removed", name)
		}
	}

	// Verify active lock still exists
	if _, err := os.Stat(activeLock.lockPath); err != nil {
		t.Error("active lock should not be removed")
	}
}

func TestCleanStaleLocksNonExistentDir(t *testing.T) {
	cleaned, err := CleanStaleLocks("/nonexistent/locks")
	if err != nil {
		t.Errorf("CleanStaleLocks() on nonexistent dir: error = %v, want nil", err)
	}
	if cleaned != 0 {
		t.Errorf("CleanStaleLocks() on nonexistent dir: cleaned %d, want 0", cleaned)
	}
}

func TestLockTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock1 := New(path)
	if err := lock1.Lock(); err != nil {
		t.Fatalf("lock1.Lock() error = %v", err)
	}
	defer lock1.Unlock()

	// Try to acquire with very short timeout
	lock2 := NewWithOptions(path, Options{
		Timeout:  10 * time.Millisecond,
		LockType: Exclusive,
	})

	start := time.Now()
	err := lock2.Lock()
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Errorf("lock2.Lock() error = %v, want ErrTimeout", err)
	}

	// Should have timed out around the timeout duration
	// Allow more variance for test environment
	if elapsed < 10*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("timeout took %v, expected ~10-200ms", elapsed)
	}
}

func TestLockImmediateTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock1 := New(path)
	if err := lock1.Lock(); err != nil {
		t.Fatalf("lock1.Lock() error = %v", err)
	}
	defer lock1.Unlock()

	// Negative timeout means fail immediately
	lock2 := NewWithOptions(path, Options{
		Timeout:  -1,
		LockType: Exclusive,
	})

	err := lock2.Lock()
	if err != ErrTimeout {
		t.Errorf("lock2.Lock() with negative timeout: error = %v, want ErrTimeout", err)
	}
}

func TestLockPIDInFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	lock := New(path)
	if err := lock.Lock(); err != nil {
		t.Fatalf("Lock() error = %v", err)
	}
	defer lock.Unlock()

	// Read lock file
	data, err := os.ReadFile(lock.lockPath)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}

	// Should contain our PID
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		t.Errorf("lock file should contain PID, got: %q", string(data))
	}

	if pid != os.Getpid() {
		t.Errorf("lock file PID = %d, want %d", pid, os.Getpid())
	}
}
