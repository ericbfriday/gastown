//go:build windows

package filelock

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// LockFileEx flags
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

// lockFile acquires an OS-level lock on the file.
// Uses LockFileEx on Windows.
func lockFile(file *os.File, lockType LockType) error {
	var flags uint32 = lockfileFailImmediately

	if lockType == Exclusive {
		flags |= lockfileExclusiveLock
	}

	// Lock the entire file
	var overlapped syscall.Overlapped

	r1, _, err := procLockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(flags),
		uintptr(0),           // reserved
		uintptr(1),           // nNumberOfBytesToLockLow (lock 1 byte minimum)
		uintptr(0),           // nNumberOfBytesToLockHigh
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		return fmt.Errorf("LockFileEx: %w", err)
	}

	return nil
}

// unlockFile releases an OS-level lock on the file.
func unlockFile(file *os.File) error {
	var overlapped syscall.Overlapped

	r1, _, err := procUnlockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(0), // reserved
		uintptr(1), // nNumberOfBytesToUnlockLow
		uintptr(0), // nNumberOfBytesToUnlockHigh
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if r1 == 0 {
		return fmt.Errorf("UnlockFileEx: %w", err)
	}

	return nil
}
