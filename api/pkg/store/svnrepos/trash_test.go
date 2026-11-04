package svnrepos

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var purgeNow = time.Date(2026, time.November, 4, 12, 0, 0, 0, time.UTC)

// writeTrashEntry creates a directory (or the given kind) with a controlled
// mtime. It returns the full path.
func writeTrashEntry(t *testing.T, root, name string, mtime time.Time) string {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}

	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", path, err)
	}

	return path
}

func TestTrashPurgerRemovesExpiredEntriesOnly(t *testing.T) {
	root := t.TempDir()
	old := purgeNow.Add(-40 * 24 * time.Hour)
	fresh := purgeNow.Add(-1 * time.Hour)

	expiredByName := writeTrashEntry(t, root, "oldrepo-20200101T000000Z", fresh)
	expiredByMtime := writeTrashEntry(t, root, "touchedrepo-20261104T110000Z", old)
	kept := writeTrashEntry(t, root, "freshrepo-20261104T110000Z", fresh)

	// A valid name that is not a directory is never deleted.
	filePath := filepath.Join(root, "afile-20200101T000000Z")
	if err := os.WriteFile(filePath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// An unrecognised name is never deleted, even when it is ancient.
	unknown := writeTrashEntry(t, root, "operator-notes", old)

	purger := NewTrashPurger(TrashPurgeOptions{
		Dir:       root,
		Retention: 720 * time.Hour,
		Now:       func() time.Time { return purgeNow },
	})

	report, err := purger.Purge(context.Background())
	if err != nil {
		t.Fatalf("Purge() error = %v", err)
	}

	for _, path := range []string{expiredByName, expiredByMtime} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("%s still exists, want it purged", filepath.Base(path))
		}
	}

	for _, path := range []string{kept, filePath, unknown, root} {
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("%s was removed, want it kept: %v", filepath.Base(path), err)
		}
	}

	if len(report.Removed) != 2 {
		t.Errorf("Removed = %v, want 2 entries", report.Removed)
	}
}

func TestTrashPurgerRefusesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink needs SeCreateSymbolicLinkPrivilege on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()
	victim := filepath.Join(outside, "victim")
	if err := os.MkdirAll(victim, 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	link := filepath.Join(root, "linkrepo-20200101T000000Z")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	purger := NewTrashPurger(TrashPurgeOptions{
		Dir:       root,
		Retention: time.Hour,
		Now:       func() time.Time { return purgeNow },
	})

	report, err := purger.Purge(context.Background())
	if err != nil {
		t.Fatalf("Purge() error = %v", err)
	}

	if _, err := os.Lstat(link); err != nil {
		t.Errorf("symlink was removed, want it kept: %v", err)
	}

	if _, err := os.Stat(victim); err != nil {
		t.Errorf("symlink target was removed: %v", err)
	}

	if len(report.Removed) != 0 {
		t.Errorf("Removed = %v, want none", report.Removed)
	}
}

func TestTrashPurgerRefusesNonPositiveRetention(t *testing.T) {
	root := t.TempDir()
	entry := writeTrashEntry(t, root, "oldrepo-20200101T000000Z", purgeNow.Add(-90*24*time.Hour))

	purger := NewTrashPurger(TrashPurgeOptions{
		Dir:       root,
		Retention: 0,
		Now:       func() time.Time { return purgeNow },
	})

	if _, err := purger.Purge(context.Background()); err == nil {
		t.Fatal("Purge() error = nil, want a refusal for a non-positive retention")
	}

	if _, err := os.Lstat(entry); err != nil {
		t.Errorf("entry was removed despite the refusal: %v", err)
	}
}

func TestTrashPurgerMissingDirectoryIsNotAnError(t *testing.T) {
	purger := NewTrashPurger(TrashPurgeOptions{
		Dir:       filepath.Join(t.TempDir(), "nope"),
		Retention: time.Hour,
		Now:       func() time.Time { return purgeNow },
	})

	if _, err := purger.Purge(context.Background()); err != nil {
		t.Fatalf("Purge() error = %v, want nil for a missing trash directory", err)
	}
}

func TestTrashPurgerHonoursCancellation(t *testing.T) {
	root := t.TempDir()
	writeTrashEntry(t, root, "oldrepo-20200101T000000Z", purgeNow.Add(-90*24*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	purger := NewTrashPurger(TrashPurgeOptions{
		Dir:       root,
		Retention: time.Hour,
		Now:       func() time.Time { return purgeNow },
	})

	if _, err := purger.Purge(ctx); err == nil {
		t.Fatal("Purge() error = nil, want the context error")
	}
}
