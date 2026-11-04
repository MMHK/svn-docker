package fsx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func TestReadFileAndHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "hello\n")

	snapshot, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !snapshot.Exists || string(snapshot.Data) != "hello\n" {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	if snapshot.Hash != Hash([]byte("hello\n")) || len(snapshot.Hash) != 64 {
		t.Errorf("hash = %q", snapshot.Hash)
	}

	if snapshot.ModTime.IsZero() {
		t.Error("ModTime is zero")
	}

	missing, err := ReadFile(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("ReadFile(missing) error = %v, want nil", err)
	}

	if missing.Exists || missing.Hash != "" {
		t.Errorf("missing snapshot = %+v", missing)
	}
}

func TestWriteAtomicRejectsStaleHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "original\n")

	err := WriteAtomic(path, []byte("replacement\n"), Hash([]byte("something else")))
	if !IsConflict(err) {
		t.Fatalf("WriteAtomic() error = %v, want a conflict", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "original\n" {
		t.Errorf("file was modified despite the conflict: %q", content)
	}
}

func TestWriteAtomicAcceptsMatchingHashAndPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "original\n")

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	if err := WriteAtomic(path, []byte("next\n"), Hash([]byte("original\n"))); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(content) != "next\n" {
		t.Errorf("content = %q", content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	// Windows only models the read-only bit, so the mode check is POSIX only.
	if info.Mode().Perm() != 0o600 && os.PathSeparator == '/' {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("temporary file left behind: %s", entry.Name())
		}
	}
}

func TestWriteAtomicCreatesMissingFileWithDefaultMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new")

	if err := WriteAtomic(path, []byte("first\n"), ""); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if os.PathSeparator == '/' && info.Mode().Perm() != DefaultPerm {
		t.Errorf("mode = %o, want %o", info.Mode().Perm(), DefaultPerm)
	}
}

func TestConcurrentWritersDoNotLoseUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "start\n")

	const writers = 20

	var (
		wait   sync.WaitGroup
		mu     sync.Mutex
		failed []error
	)

	for i := 0; i < writers; i++ {
		wait.Add(1)

		go func(index int) {
			defer wait.Done()

			line := fmt.Sprintf("line-%02d\n", index)

			lock, err := Lock(path)
			if err != nil {
				mu.Lock()
				failed = append(failed, err)
				mu.Unlock()

				return
			}
			defer lock.Unlock()

			snapshot, err := ReadFile(path)
			if err != nil {
				mu.Lock()
				failed = append(failed, err)
				mu.Unlock()

				return
			}

			updated := append(append([]byte(nil), snapshot.Data...), line...)

			if err := WriteAtomic(path, updated, snapshot.Hash); err != nil {
				mu.Lock()
				failed = append(failed, err)
				mu.Unlock()
			}
		}(i)
	}

	wait.Wait()

	if len(failed) > 0 {
		t.Fatalf("concurrent writers failed: %v", failed)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) != writers+1 || lines[0] != "start" {
		t.Fatalf("lines = %d (%q)", len(lines), content)
	}

	seen := make(map[string]bool, writers)
	for _, line := range lines[1:] {
		seen[line] = true
	}

	for i := 0; i < writers; i++ {
		expected := fmt.Sprintf("line-%02d", i)
		if !seen[expected] {
			t.Errorf("lost update: %s", expected)
		}
	}
}

func TestLockExcludesCompetingLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	writeFile(t, path, "x")

	first, err := Lock(path)
	if err != nil {
		t.Fatalf("Lock() error = %v", err)
	}

	// A second lock attempt must time out while the first is held.
	_, err = LockWithTimeout(path, 50*time.Millisecond)
	if err == nil {
		t.Fatal("second Lock() succeeded while the first was held")
	}

	var timeout *TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("error = %T %v, want *TimeoutError", err, err)
	}

	if err := first.Unlock(); err != nil {
		t.Fatalf("Unlock() error = %v", err)
	}

	again, err := LockWithTimeout(path, 2*time.Second)
	if err != nil {
		t.Fatalf("Lock() after unlock error = %v", err)
	}

	_ = again.Unlock()

	// Unlock is idempotent.
	if err := first.Unlock(); err != nil {
		t.Errorf("second Unlock() error = %v", err)
	}
}

func TestBackupRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svn-auth-file")
	backups := filepath.Join(dir, "backup")

	writeFile(t, path, "v0\n")

	base := time.Date(2026, 11, 4, 5, 12, 39, 0, time.UTC)
	created := make([]string, 0, 5)

	for i := 1; i <= 5; i++ {
		backup, err := backupAt(path, backups, 3, base.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("Backup() error = %v", err)
		}

		created = append(created, backup)

		writeFile(t, path, fmt.Sprintf("v%d\n", i))
	}

	entries, err := os.ReadDir(backups)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("backups = %d, want 3 (%v)", len(entries), entries)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())

		if !strings.HasPrefix(entry.Name(), "svn-auth-file.") {
			t.Errorf("unexpected backup name %q", entry.Name())
		}
	}

	sort.Strings(names)

	// The newest three survive: v2, v3 and v4 (v0/v1 were rotated out).
	if names[0] != filepath.Base(created[2]) {
		t.Errorf("oldest surviving backup = %q, want %q", names[0], filepath.Base(created[2]))
	}

	// A missing source is not an error.
	if backup, err := Backup(filepath.Join(dir, "nope"), backups, 3); err != nil || backup != "" {
		t.Errorf("Backup(missing) = %q, %v", backup, err)
	}
}

func TestBackupKeepsByteContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	content := "# comment\r\nalice:$apr1$x\r\n"
	writeFile(t, path, content)

	backup, err := Backup(path, filepath.Join(dir, "backup"), 5)
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	copied, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("ReadFile(backup) error = %v", err)
	}

	if string(copied) != content {
		t.Errorf("backup content = %q, want %q", copied, content)
	}
}

func TestSplitLinesIsLossless(t *testing.T) {
	for _, content := range []string{
		"",
		"\n",
		"a",
		"a\n",
		"a\r\nb\r\n",
		"# c\r\n\r\nalice:x\r\n",
		"trailing\r\n\r\n",
	} {
		lines := SplitLines([]byte(content))
		if got := string(JoinLines(lines)); got != content {
			t.Errorf("round trip %q = %q", content, got)
		}
	}
}
