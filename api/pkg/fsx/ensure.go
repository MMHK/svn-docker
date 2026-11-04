package fsx

import (
	"fmt"
	"os"
)

// EnsureFile creates an empty file at path when it does not exist yet.
//
// An existing file is left untouched: its content, mode and mtime are not
// modified. A missing parent directory is reported as an error that wraps
// os.ErrNotExist, so callers can map it to their own "storage unavailable"
// response. Creation uses O_EXCL, which also refuses to follow a symlink:
// a dangling symlink fails instead of silently writing through it.
func EnsureFile(path string, mode os.FileMode) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("fsx: inspect %s: %w", path, err)
	}

	handle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return false, fmt.Errorf("fsx: create %s: %w", path, err)
	}

	if err := handle.Close(); err != nil {
		return true, fmt.Errorf("fsx: close %s: %w", path, err)
	}

	return true, nil
}
