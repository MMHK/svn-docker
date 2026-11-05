package adminfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

func TestAdminReturnsEmptyForMissingOrEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svnadmin-admin")
	store := New(path)
	ctx := context.Background()

	name, err := store.Admin(ctx)
	if err != nil {
		t.Fatalf("Admin() error = %v", err)
	}

	if name != "" {
		t.Errorf("Admin() = %q, want an empty name for a missing file", name)
	}

	if err := os.WriteFile(path, []byte("\r\n   \r\n"), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	name, err = store.Admin(ctx)
	if err != nil {
		t.Fatalf("Admin() error = %v", err)
	}

	if name != "" {
		t.Errorf("Admin() = %q, want an empty name for a blank file", name)
	}
}

func TestAdminTakesFirstNonEmptyLineWithCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svnadmin-admin")

	if err := os.WriteFile(path, []byte(" \r\n alice \r\nbob\r\n"), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	name, err := New(path).Admin(context.Background())
	if err != nil {
		t.Fatalf("Admin() error = %v", err)
	}

	if name != "alice" {
		t.Errorf("Admin() = %q, want alice", name)
	}
}

func TestSetAdminWritesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "svnadmin-admin")
	store := New(path)
	ctx := context.Background()

	if err := store.SetAdmin(ctx, "alice"); err != nil {
		t.Fatalf("SetAdmin() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "alice\n" {
		t.Errorf("file = %q, want %q", data, "alice\n")
	}

	if err := store.SetAdmin(ctx, " bob "); err != nil {
		t.Fatalf("SetAdmin() error = %v", err)
	}

	name, err := store.Admin(ctx)
	if err != nil {
		t.Fatalf("Admin() error = %v", err)
	}

	if name != "bob" {
		t.Errorf("Admin() = %q, want bob", name)
	}
}

func TestSetAdminRejectsEmptyName(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "svnadmin-admin"))

	err := store.SetAdmin(context.Background(), "   ")
	if err == nil {
		t.Fatal("SetAdmin() error = nil, want a validation error")
	}

	var validation *model.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want *model.ValidationError", err)
	}

	if validation.Field != "username" {
		t.Errorf("Field = %q, want username", validation.Field)
	}
}
