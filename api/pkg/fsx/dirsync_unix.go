//go:build unix

package fsx

import "os"

// syncDir fsyncs a directory so a rename inside it survives a crash.
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}

	syncErr := dir.Sync()
	closeErr := dir.Close()

	if syncErr != nil {
		return syncErr
	}

	return closeErr
}
