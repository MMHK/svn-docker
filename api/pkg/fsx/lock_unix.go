//go:build unix

package fsx

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// lockFile takes a non-blocking exclusive flock(2) on f. Lock ownership is
// tied to the open file description, so two opens inside the same process
// conflict exactly like two processes do.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}

func lockIsBusy(err error) bool {
	return errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) ||
		errors.Is(err, unix.EACCES)
}
