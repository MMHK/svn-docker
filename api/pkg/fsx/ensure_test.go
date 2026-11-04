package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureFileCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "created")

	created, err := EnsureFile(path, 0o640)
	if err != nil {
		t.Fatalf("EnsureFile() error = %v", err)
	}

	if !created {
		t.Fatal("EnsureFile() created = false, want true")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat created file: %v", err)
	}

	if info.Size() != 0 {
		t.Errorf("size = %d, want 0", info.Size())
	}

	if mode := info.Mode().Perm(); os.PathSeparator == '/' && mode != 0o640 {
		t.Errorf("mode = %o, want 640", mode)
	}
}

func TestEnsureFileKeepsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing")
	writeFile(t, path, "keep me\n")

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	created, err := EnsureFile(path, 0o640)
	if err != nil {
		t.Fatalf("EnsureFile() error = %v", err)
	}

	if created {
		t.Error("EnsureFile() created = true for an existing file")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(data) != "keep me\n" {
		t.Errorf("content = %q, want it untouched", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if mode := info.Mode().Perm(); os.PathSeparator == '/' && mode != 0o600 {
		t.Errorf("mode = %o, want the original 600", mode)
	}
}

func TestEnsureFileReportsMissingParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "file")

	created, err := EnsureFile(path, 0o640)
	if err == nil {
		t.Fatal("EnsureFile() error = nil, want a missing-directory error")
	}

	if created {
		t.Error("created = true after a failure")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want it to wrap os.ErrNotExist", err)
	}
}

func TestEnsureFileRefusesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "link")

	if err := os.Symlink(filepath.Join(dir, "missing-target"), path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if _, err := EnsureFile(path, 0o640); err == nil {
		t.Fatal("EnsureFile() followed a dangling symlink")
	}

	if _, err := os.Stat(filepath.Join(dir, "missing-target")); !os.IsNotExist(err) {
		t.Error("EnsureFile() created the symlink target")
	}
}
