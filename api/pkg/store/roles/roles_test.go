package roles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const sample = `; managed by iF.SVNAdmin
[mixmedia]
Administrator=

[sam:AccessPaths]
0=laneige-coupon:/
1=other:/trunk

[alice]
Role=User-Group-Manager
Repository-Creator=

[bob]
User=
`

func writeRoleFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "userroleassignments.ini")
	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}

func TestAssignments(t *testing.T) {
	store := New(writeRoleFile(t, sample))

	assignments, err := store.Assignments(context.Background())
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := assignments.Roles["mixmedia"]; len(got) != 1 || got[0] != model.RoleAdministrator {
		t.Errorf("mixmedia roles = %v", got)
	}

	if got := assignments.Roles["alice"]; len(got) != 2 ||
		got[0] != model.RoleUserGroupManager || got[1] != model.RoleRepositoryCreator {
		t.Errorf("alice roles = %v", got)
	}

	if got := assignments.Roles["bob"]; len(got) != 1 || got[0] != model.RoleUser {
		t.Errorf("bob roles = %v", got)
	}

	if got := assignments.AccessPaths["sam"]; len(got) != 2 ||
		got[0] != "laneige-coupon:/" || got[1] != "other:/trunk" {
		t.Errorf("sam access paths = %v", got)
	}

	if _, ok := assignments.Roles["sam"]; !ok {
		t.Error("sam is missing from the roles map; [user:AccessPaths] must register the user")
	}
}

func TestHasAdministrator(t *testing.T) {
	store := New(writeRoleFile(t, sample))

	has, err := store.HasAdministrator(context.Background())
	if err != nil {
		t.Fatalf("HasAdministrator() error = %v", err)
	}

	if !has {
		t.Error("HasAdministrator() = false, want true")
	}

	empty := New(writeRoleFile(t, "[alice]\nUser=\n"))

	has, err = empty.HasAdministrator(context.Background())
	if err != nil {
		t.Fatalf("HasAdministrator() error = %v", err)
	}

	if has {
		t.Error("HasAdministrator() = true for a file without Administrator")
	}
}

func TestMissingFileIsEmptyNotError(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "missing"))

	assignments, err := store.Assignments(context.Background())
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if len(assignments.Roles) != 0 || len(assignments.AccessPaths) != 0 {
		t.Errorf("assignments = %+v, want empty", assignments)
	}
}

func TestAssignRoleAppendsAndIsIdempotent(t *testing.T) {
	path := writeRoleFile(t, sample)
	store := New(path)
	ctx := context.Background()

	if err := store.AssignRole(ctx, "alice", model.RoleAdministrator); err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "[alice]") {
		t.Errorf("appended content missing the section:\n%s", content)
	}

	assignments, err := store.Assignments(ctx)
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := assignments.Roles["alice"]; len(got) != 3 {
		t.Errorf("alice roles after append = %v", got)
	}

	before := string(content)

	if err := store.AssignRole(ctx, "alice", model.RoleAdministrator); err != nil {
		t.Fatalf("AssignRole() second call error = %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(after) != before {
		t.Error("AssignRole() appended a duplicate role")
	}

	if err := store.AssignRole(ctx, "alice", model.Role("Nope")); !model.IsValidation(err) {
		t.Errorf("AssignRole(bad role) error = %v, want ValidationError", err)
	}
}

func TestAssignRoleCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "userroleassignments.ini")
	store := New(path)

	if err := store.AssignRole(context.Background(), "alice", model.RoleAdministrator); err != nil {
		t.Fatalf("AssignRole() error = %v", err)
	}

	assignments, err := store.Assignments(context.Background())
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := assignments.Roles["alice"]; len(got) != 1 || got[0] != model.RoleAdministrator {
		t.Errorf("alice roles = %v", got)
	}
}

func TestStoreSeesExternalRewrite(t *testing.T) {
	path := writeRoleFile(t, "[alice]\nUser=\n")
	store := New(path)

	first, err := store.Assignments(context.Background())
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if len(first.Roles["alice"]) != 1 {
		t.Fatalf("initial roles = %v", first.Roles["alice"])
	}

	if err := os.WriteFile(path, []byte("[alice]\nAdministrator=\n"), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	second, err := store.Assignments(context.Background())
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := second.Roles["alice"]; len(got) != 1 || got[0] != model.RoleAdministrator {
		t.Errorf("roles after rewrite = %v", got)
	}
}

func TestListRoles(t *testing.T) {
	store := New(writeRoleFile(t, sample))

	roles, err := store.ListRoles(context.Background())
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}

	if len(roles) != len(model.AllRoles()) {
		t.Errorf("ListRoles() = %v", roles)
	}
}

func TestTypedReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	store := New(path)

	_, err := store.Assignments(context.Background())
	if err == nil {
		t.Fatal("Assignments() error = nil, want a read error")
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *roles.Error", err)
	}
}
