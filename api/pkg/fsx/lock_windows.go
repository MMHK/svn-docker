//go:build windows

package fsx

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockRange is the single byte range locked on the lock file. A fixed range
// (offset 0, length 1) is enough: every writer locks the same byte.
const lockRange = 1

// lockFile takes a non-blocking exclusive LockFileEx lock on f. Byte-range
// locks are per handle, so two handles in the same process conflict.
func lockFile(f *os.File) error {
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockRange,
		0,
		new(windows.Overlapped),
	)
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		lockRange,
		0,
		new(windows.Overlapped),
	)
}

func lockIsBusy(err error) bool {
	return errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
		errors.Is(err, windows.ERROR_IO_PENDING)
}
