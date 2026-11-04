// Package fsx is the file safety layer shared by the credential, authz and
// role stores (plan sections 5.1-5.3, task T2.1): cross-process advisory
// locks, hash-checked atomic writes and rotating backups.
//
// The package deliberately knows nothing about the domain. Callers translate
// a *ConflictError into the typed model.ConflictError the transport maps to
// HTTP 409.
package fsx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultPerm is used when the target file does not exist yet. It matches
	// the mode the v2 tree uses for svn-auth-file / svn-access-file.
	DefaultPerm os.FileMode = 0o640
	// DefaultBackupKeep is the rotation depth from plan section 5.1.
	DefaultBackupKeep = 30
	// DefaultLockTimeout bounds how long Lock waits for a competing writer.
	DefaultLockTimeout = 10 * time.Second

	// maxSameSecondBackups bounds the "-N" collision suffix so a broken clock
	// or a permission problem cannot spin forever.
	maxSameSecondBackups = 1000
)

// ConflictError reports that the file changed between the read the caller
// based its edit on and the write. It never carries file content.
type ConflictError struct {
	Path   string
	Reason string
}

func (e *ConflictError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("fsx: %s changed since it was read", e.Path)
	}

	return fmt.Sprintf("fsx: %s: %s", e.Path, e.Reason)
}

// IsConflict reports whether err is a stale-write conflict.
func IsConflict(err error) bool {
	var target *ConflictError

	return errors.As(err, &target)
}

// TimeoutError reports that the lock could not be acquired in time.
type TimeoutError struct {
	Path  string
	After time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("fsx: timed out after %s waiting for the lock on %s", e.After, e.Path)
}

// Hash returns the lowercase hex SHA-256 of data.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// Snapshot is one consistent read of a file.
type Snapshot struct {
	Path    string
	Data    []byte
	Hash    string
	ModTime time.Time
	Mode    os.FileMode
	Exists  bool
}

// ReadFile reads path together with its hash, mtime and mode. A missing file
// is not an error: it yields Exists=false so callers can create it.
func ReadFile(path string) (Snapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{Path: path}, nil
		}

		return Snapshot{}, fmt.Errorf("fsx: stat %s: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fsx: read %s: %w", path, err)
	}

	return Snapshot{
		Path:    path,
		Data:    data,
		Hash:    Hash(data),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
		Exists:  true,
	}, nil
}

// WriteAtomic replaces path with data. The write goes to a temporary file in
// the same directory, is fsynced, gets the original mode (or DefaultPerm), and
// is renamed over the target; the directory is then fsynced so the rename is
// durable. Ownership is left untouched.
//
// When expectedHash is not empty the current content must match it, otherwise
// a *ConflictError is returned before anything is written.
func WriteAtomic(path string, data []byte, expectedHash string) error {
	current, err := ReadFile(path)
	if err != nil {
		return err
	}

	if expectedHash != "" {
		if !current.Exists {
			return &ConflictError{Path: path, Reason: "the file was removed by another writer"}
		}

		if !strings.EqualFold(expectedHash, current.Hash) {
			return &ConflictError{Path: path, Reason: "the file was modified by another writer"}
		}
	}

	mode := DefaultPerm
	if current.Exists {
		mode = current.Mode.Perm()
	}

	dir := filepath.Dir(path)

	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("fsx: create temporary file next to %s: %w", path, err)
	}

	tempName := temp.Name()

	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	if _, err := temp.Write(data); err != nil {
		cleanup()

		return fmt.Errorf("fsx: write temporary file for %s: %w", path, err)
	}

	if err := temp.Sync(); err != nil {
		cleanup()

		return fmt.Errorf("fsx: fsync temporary file for %s: %w", path, err)
	}

	if err := temp.Chmod(mode); err != nil {
		cleanup()

		return fmt.Errorf("fsx: chmod temporary file for %s: %w", path, err)
	}

	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)

		return fmt.Errorf("fsx: close temporary file for %s: %w", path, err)
	}

	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)

		return fmt.Errorf("fsx: rename temporary file over %s: %w", path, err)
	}

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("fsx: fsync directory %s: %w", dir, err)
	}

	return nil
}

// Backup copies path to backupDir as "<base>.<UTC20060102T150405Z>" and keeps
// only the newest keep files. A missing source file is not an error and yields
// an empty backup path, so "create" flows can call it unconditionally.
func Backup(path, backupDir string, keep int) (string, error) {
	return backupAt(path, backupDir, keep, time.Now().UTC())
}

// backupAt is Backup with an injectable clock, so rotation tests are
// deterministic instead of depending on how many backups fit into one second.
func backupAt(path, backupDir string, keep int, now time.Time) (string, error) {
	if strings.TrimSpace(backupDir) == "" {
		return "", nil
	}

	if keep <= 0 {
		keep = DefaultBackupKeep
	}

	source, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", fmt.Errorf("fsx: open %s for backup: %w", path, err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return "", fmt.Errorf("fsx: stat %s for backup: %w", path, err)
	}

	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return "", fmt.Errorf("fsx: create backup directory %s: %w", backupDir, err)
	}

	base := filepath.Base(path)
	stamp := now.Format("20060102T150405Z")

	destination := filepath.Join(backupDir, base+"."+stamp)
	for suffix := 2; ; suffix++ {
		_, statErr := os.Lstat(destination)
		if errors.Is(statErr, os.ErrNotExist) {
			break
		}

		if statErr != nil {
			return "", fmt.Errorf("fsx: inspect backup target %s: %w", destination, statErr)
		}

		if suffix > maxSameSecondBackups {
			return "", fmt.Errorf(
				"fsx: more than %d backups of %s in the same second",
				maxSameSecondBackups, base,
			)
		}

		destination = fmt.Sprintf("%s.%s-%d", filepath.Join(backupDir, base), stamp, suffix)
	}

	target, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, DefaultPerm)
	if err != nil {
		return "", fmt.Errorf("fsx: create backup %s: %w", destination, err)
	}

	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		_ = os.Remove(destination)

		return "", fmt.Errorf("fsx: copy backup %s: %w", destination, err)
	}

	if err := target.Sync(); err != nil {
		_ = target.Close()
		_ = os.Remove(destination)

		return "", fmt.Errorf("fsx: fsync backup %s: %w", destination, err)
	}

	if err := target.Chmod(info.Mode().Perm()); err != nil {
		_ = target.Close()
		_ = os.Remove(destination)

		return "", fmt.Errorf("fsx: chmod backup %s: %w", destination, err)
	}

	if err := target.Close(); err != nil {
		_ = os.Remove(destination)

		return "", fmt.Errorf("fsx: close backup %s: %w", destination, err)
	}

	if err := pruneBackups(backupDir, base, keep); err != nil {
		return destination, err
	}

	return destination, nil
}

// pruneBackups deletes all but the newest keep backups of base.
func pruneBackups(backupDir, base string, keep int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("fsx: list backup directory %s: %w", backupDir, err)
	}

	prefix := base + "."
	names := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}

		names = append(names, entry.Name())
	}

	// The timestamp is fixed width and UTC, so lexical descending order is
	// chronological descending order. A "-2" collision suffix still sorts
	// next to its stamp.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	for index, name := range names {
		if index < keep {
			continue
		}

		if err := os.Remove(filepath.Join(backupDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("fsx: remove old backup %s: %w", name, err)
		}
	}

	return nil
}
