//go:build !unix && !windows

package fsx

import (
	"os"
	"sync"
)

// Fallback for platforms without an advisory file lock (planar, wasm, ...).
// It still serialises writers inside one process; cross-process exclusion is
// not available there. The management service is only built for Linux while
// the developer machines run Windows, so this path is never used in
// production - it exists so `go build` keeps working everywhere.

var (
	fallbackMu    sync.Mutex
	fallbackLocks = make(map[string]*sync.Mutex)
)

func fallbackLock(path string) *sync.Mutex {
	fallbackMu.Lock()
	defer fallbackMu.Unlock()

	lock, ok := fallbackLocks[path]
	if !ok {
		lock = new(sync.Mutex)
		fallbackLocks[path] = lock
	}

	return lock
}

func lockFile(f *os.File) error {
	fallbackLock(f.Name()).Lock()

	return nil
}

func unlockFile(f *os.File) error {
	lock := fallbackLock(f.Name())

	// Unlock panics when the mutex is not held; every lockFile call holds it,
	// so this is safe.
	lock.Unlock()

	return nil
}

func lockIsBusy(error) bool {
	return false
}
