package fsx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Handle is an exclusive advisory lock on "<path>.lock". It excludes other
// processes (flock on Unix, LockFileEx on Windows) and other goroutines, so a
// read-modify-write cycle is serialised without depending on the in-process
// cache.
type Handle struct {
	path string
	file *os.File

	mu       sync.Mutex
	released bool
}

// Lock acquires the exclusive lock for path, waiting up to DefaultLockTimeout.
func Lock(path string) (*Handle, error) {
	return LockWithTimeout(path, DefaultLockTimeout)
}

// LockWithTimeout acquires the exclusive lock for path. The lock file is
// created next to the target (mode 0640) and is never removed: unlinking a
// lock file while another process waits on it would break mutual exclusion.
func LockWithTimeout(path string, timeout time.Duration) (*Handle, error) {
	lockPath := path + ".lock"

	if dir := filepath.Dir(lockPath); dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("fsx: create lock directory %s: %w", dir, err)
		}
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, DefaultPerm)
	if err != nil {
		return nil, fmt.Errorf("fsx: open lock file %s: %w", lockPath, err)
	}

	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}

	deadline := time.Now().Add(timeout)

	for {
		err := lockFile(file)
		if err == nil {
			return &Handle{path: lockPath, file: file}, nil
		}

		if !lockIsBusy(err) {
			_ = file.Close()

			return nil, fmt.Errorf("fsx: lock %s: %w", lockPath, err)
		}

		if time.Now().After(deadline) {
			_ = file.Close()

			return nil, &TimeoutError{Path: lockPath, After: timeout}
		}

		// 2ms keeps the wait cheap for the common "another request is writing
		// right now" case without busy spinning.
		time.Sleep(2 * time.Millisecond)
	}
}

// Path returns the lock file path (useful in logs).
func (l *Handle) Path() string {
	return l.path
}

// Unlock releases the lock. It is safe to call more than once.
func (l *Handle) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.released {
		return nil
	}

	l.released = true

	err := unlockFile(l.file)
	closeErr := l.file.Close()

	if err != nil {
		return fmt.Errorf("fsx: unlock %s: %w", l.path, err)
	}

	if closeErr != nil {
		return fmt.Errorf("fsx: close lock %s: %w", l.path, closeErr)
	}

	return nil
}
