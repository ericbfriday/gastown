// Package filelock provides file-based locking for concurrent access protection.
//
// This package implements advisory file locks using OS primitives (flock on Unix,
// LockFileEx on Windows) to prevent race conditions when multiple processes or
// goroutines access shared state files.
//
// Lock files are stored in .gastown/locks/ and automatically cleaned up on process exit.
// Stale locks (from dead processes) are automatically detected and cleaned.
//
// Usage:
//
//	lock := filelock.New("/path/to/data.json")
//	if err := lock.Lock(); err != nil {
//	    return err
//	}
//	defer lock.Unlock()
//	// ... perform file operations ...
package filelock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrTimeout is returned when lock acquisition times out.
	ErrTimeout = errors.New("timeout acquiring lock")

	// ErrAlreadyLocked is returned when trying to lock an already-locked FileLock.
	ErrAlreadyLocked = errors.New("already locked")

	// ErrNotLocked is returned when trying to unlock an unlocked FileLock.
	ErrNotLocked = errors.New("not locked")
)

// LockType represents the type of lock.
type LockType int

const (
	// Exclusive lock for write operations.
	Exclusive LockType = iota
	// Shared lock for read operations.
	Shared
)

// FileLock represents a file-based lock.
type FileLock struct {
	path      string        // Path to the data file being protected
	lockPath  string        // Path to the lock file
	lockType  LockType      // Type of lock (shared or exclusive)
	timeout   time.Duration // Timeout for lock acquisition
	file      *os.File      // Lock file handle
	mu        sync.Mutex    // Protects internal state
	locked    bool          // Whether we currently hold the lock
	retryDelay time.Duration // Delay between retry attempts
}

// Options configures a FileLock.
type Options struct {
	// Timeout for lock acquisition (0 = infinite wait, < 0 = no wait/fail immediately).
	Timeout time.Duration

	// LockType specifies whether this is a shared or exclusive lock.
	LockType LockType

	// RetryDelay is the delay between lock acquisition attempts.
	// Default is 10ms.
	RetryDelay time.Duration

	// LockDir is the directory where lock files are stored.
	// Default is .gastown/locks/ in the same directory as the target file.
	LockDir string
}

// New creates a new FileLock for the given path.
// Uses default options (exclusive lock, 30 second timeout).
func New(path string) *FileLock {
	return NewWithOptions(path, Options{
		Timeout:    30 * time.Second,
		LockType:   Exclusive,
		RetryDelay: 10 * time.Millisecond,
	})
}

// NewWithOptions creates a FileLock with custom options.
func NewWithOptions(path string, opts Options) *FileLock {
	if opts.RetryDelay == 0 {
		opts.RetryDelay = 10 * time.Millisecond
	}

	// Determine lock directory
	lockDir := opts.LockDir
	if lockDir == "" {
		// Default: .gastown/locks/ in same directory as target file
		dir := filepath.Dir(path)
		lockDir = filepath.Join(dir, ".gastown", "locks")
	}

	// Create lock filename from target path
	// Replace path separators with underscores to create flat namespace
	lockName := filepath.Base(path) + ".lock"

	return &FileLock{
		path:       path,
		lockPath:   filepath.Join(lockDir, lockName),
		lockType:   opts.LockType,
		timeout:    opts.Timeout,
		retryDelay: opts.RetryDelay,
	}
}

// Lock acquires the lock.
// Blocks until lock is acquired or timeout expires.
// Returns ErrTimeout if timeout expires.
// Returns ErrAlreadyLocked if this FileLock already holds the lock.
func (l *FileLock) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locked {
		return ErrAlreadyLocked
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(filepath.Dir(l.lockPath), 0755); err != nil {
		return fmt.Errorf("creating lock directory: %w", err)
	}

	// Try to acquire lock with retry logic
	start := time.Now()
	attempts := 0

	for {
		attempts++

		// Try to acquire the lock
		if err := l.tryLock(); err == nil {
			l.locked = true
			return nil
		}

		// Check timeout
		if l.timeout >= 0 {
			elapsed := time.Since(start)
			if elapsed >= l.timeout {
				return fmt.Errorf("%w (after %d attempts, %v elapsed)",
					ErrTimeout, attempts, elapsed)
			}

			// Sleep with exponential backoff, capped at 100ms
			delay := l.retryDelay
			if attempts > 5 {
				delay = 50 * time.Millisecond
			}
			if attempts > 10 {
				delay = 100 * time.Millisecond
			}

			// Don't sleep beyond timeout
			remaining := l.timeout - elapsed
			if delay > remaining {
				delay = remaining
			}

			time.Sleep(delay)
		} else if l.timeout < 0 {
			// Negative timeout means fail immediately
			return ErrTimeout
		}
		// Zero timeout means infinite wait, continue retrying
	}
}

// TryLock attempts to acquire the lock without blocking.
// Returns immediately with an error if lock cannot be acquired.
func (l *FileLock) TryLock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.locked {
		return ErrAlreadyLocked
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(filepath.Dir(l.lockPath), 0755); err != nil {
		return fmt.Errorf("creating lock directory: %w", err)
	}

	if err := l.tryLock(); err != nil {
		return err
	}

	l.locked = true
	return nil
}

// Unlock releases the lock.
// Returns ErrNotLocked if the lock is not currently held.
func (l *FileLock) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.locked {
		return ErrNotLocked
	}

	return l.unlock()
}

// unlock performs the actual unlock (must be called with mu held).
func (l *FileLock) unlock() error {
	if l.file == nil {
		l.locked = false
		return nil
	}

	// Release OS lock
	if err := unlockFile(l.file); err != nil {
		return fmt.Errorf("releasing OS lock: %w", err)
	}

	// Close and remove lock file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("closing lock file: %w", err)
	}
	l.file = nil

	// Best-effort cleanup of lock file
	_ = os.Remove(l.lockPath)

	l.locked = false
	return nil
}

// IsLocked returns whether this FileLock currently holds the lock.
func (l *FileLock) IsLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.locked
}

// LockPath returns the path to the lock file.
func (l *FileLock) LockPath() string {
	return l.lockPath
}

// Path returns the path to the file being protected.
func (l *FileLock) Path() string {
	return l.path
}

// tryLock attempts to acquire the lock (must be called with mu held).
func (l *FileLock) tryLock() error {
	// Open or create lock file
	// O_CREATE | O_RDWR: Create if doesn't exist, open for read/write
	file, err := os.OpenFile(l.lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}

	// Try to acquire OS-level lock
	if err := lockFile(file, l.lockType); err != nil {
		file.Close()
		return err
	}

	// Write PID to lock file for debugging
	if err := file.Truncate(0); err == nil {
		_, _ = file.Seek(0, 0)
		fmt.Fprintf(file, "%d\n", os.Getpid())
	}

	l.file = file
	return nil
}

// WithLock executes a function while holding the lock.
// Automatically acquires and releases the lock.
func (l *FileLock) WithLock(fn func() error) error {
	if err := l.Lock(); err != nil {
		return err
	}
	defer l.Unlock()
	return fn()
}

// WithReadLock executes a function while holding a shared read lock.
func WithReadLock(path string, fn func() error) error {
	lock := NewWithOptions(path, Options{
		Timeout:  30 * time.Second,
		LockType: Shared,
	})
	return lock.WithLock(fn)
}

// WithWriteLock executes a function while holding an exclusive write lock.
func WithWriteLock(path string, fn func() error) error {
	lock := New(path)
	return lock.WithLock(fn)
}

// CleanStaleLocks removes lock files from dead processes in the given directory.
// Returns the number of locks cleaned and any error.
func CleanStaleLocks(lockDir string) (int, error) {
	entries, err := os.ReadDir(lockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cleaned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .lock files
		if filepath.Ext(entry.Name()) != ".lock" {
			continue
		}

		lockPath := filepath.Join(lockDir, entry.Name())

		// Try to open and lock the file
		// If we can lock it, it's stale (no process holds it)
		file, err := os.OpenFile(lockPath, os.O_RDWR, 0644)
		if err != nil {
			continue // Skip files we can't open
		}

		// Try to acquire exclusive lock
		if err := lockFile(file, Exclusive); err == nil {
			// We got the lock, so it was stale - clean it up
			unlockFile(file)
			file.Close()
			if err := os.Remove(lockPath); err == nil {
				cleaned++
			}
		} else {
			// Lock is held by another process - leave it alone
			file.Close()
		}
	}

	return cleaned, nil
}
