//go:build unix || darwin || linux

package filelock

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// lockFile acquires an OS-level lock on the file.
// Uses flock(2) on Unix systems.
func lockFile(file *os.File, lockType LockType) error {
	how := unix.LOCK_EX // Exclusive lock by default
	if lockType == Shared {
		how = unix.LOCK_SH
	}
	how |= unix.LOCK_NB // Non-blocking

	if err := unix.Flock(int(file.Fd()), how); err != nil {
		if err == unix.EWOULDBLOCK {
			return fmt.Errorf("file is locked by another process")
		}
		return fmt.Errorf("flock: %w", err)
	}
	return nil
}

// unlockFile releases an OS-level lock on the file.
func unlockFile(file *os.File) error {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_UN); err != nil {
		return fmt.Errorf("unlock flock: %w", err)
	}
	return nil
}
