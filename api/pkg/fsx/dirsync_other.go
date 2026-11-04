//go:build !unix

package fsx

// syncDir is a no-op: Windows has no directory fsync, and opening a directory
// for Sync fails there. The file itself was already fsynced before the rename.
func syncDir(string) error {
	return nil
}
