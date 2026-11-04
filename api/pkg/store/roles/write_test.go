package roles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const writeSample = `; managed by iF.SVNAdmin
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

func newWriteStore(t *testing.T, content string) (string, *Store, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "userroleassignments.ini")

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := filepath.Join(dir, "backup")

	return path, New(path, WithBackupDir(backupDir)), backupDir
}

func readRoles(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	return string(data)
}

// preserved re-reads both versions and asserts every line of before survives
// in after, except the lines the caller says may change.
func preserved(t *testing.T, before, after string, changed func(string) bool) {
	t.Helper()

	counts := make(map[string]int)

	for _, line := range strings.Split(after, "\n") {
		counts[line]++
	}

	for _, line := range strings.Split(strings.TrimSuffix(before, "\n"), "\n") {
		if changed != nil && changed(line) {
			continue
		}

		if counts[line] == 0 {
			t.Fatalf("line was not preserved: %q\n--- after ---\n%s", line, after)
		}

		counts[line]--
	}
}

func TestSetRolesIsLossless(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)
	ctx := context.Background()

	roles := []model.Role{model.RoleUser, model.RoleRepositoryManager}
	if err := store.SetRoles(ctx, "bob", roles, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() error = %v", err)
	}

	content := readRoles(t, path)

	preserved(t, writeSample, content, func(line string) bool {
		return line == "User=" || line == "[bob]"
	})

	assignments, err := store.Assignments(ctx)
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	got := assignments.Roles["bob"]
	if len(got) != 2 || got[0] != model.RoleUser || got[1] != model.RoleRepositoryManager {
		t.Errorf("bob roles = %v", got)
	}

	// Other users and the :AccessPaths block must be untouched.
	if got := assignments.Roles["mixmedia"]; len(got) != 1 || got[0] != model.RoleAdministrator {
		t.Errorf("mixmedia roles = %v", got)
	}

	if got := assignments.AccessPaths["sam"]; len(got) != 2 {
		t.Errorf("sam access paths = %v", got)
	}
}

func TestSetRolesKeepsRoleValueStyle(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)

	if err := store.SetRoles(context.Background(), "alice", []model.Role{
		model.RoleUser,
		model.RoleRoleManager,
	}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() error = %v", err)
	}

	content := readRoles(t, path)

	if !strings.Contains(content, "Role=User\n") || !strings.Contains(content, "Role=Role-Manager\n") {
		t.Errorf("the Role=<name> style was not preserved:\n%s", content)
	}

	if strings.Contains(content, "Repository-Creator=") {
		t.Errorf("the removed role line survived:\n%s", content)
	}
}

func TestSetRolesEmptyRemovesOnlyRoleLines(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)

	if err := store.SetRoles(context.Background(), "bob", nil, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() error = %v", err)
	}

	content := readRoles(t, path)

	if strings.Contains(content, "User=") {
		t.Errorf("bob still holds a role:\n%s", content)
	}

	if !strings.Contains(content, "[bob]") {
		t.Errorf("the section header was removed:\n%s", content)
	}

	preserved(t, writeSample, content, func(line string) bool {
		return line == "User="
	})
}

func TestSetRolesCreatesMissingSectionAndIsIdempotent(t *testing.T) {
	path, store, backupDir := newWriteStore(t, writeSample)
	ctx := context.Background()

	roles := []model.Role{model.RoleAdministrator, model.RoleUser}
	if err := store.SetRoles(ctx, "dave", roles, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() error = %v", err)
	}

	content := readRoles(t, path)
	if !strings.Contains(content, "[dave]") {
		t.Fatalf("missing section:\n%s", content)
	}

	// v2 layout for a new section.
	if !strings.Contains(content, "Administrator=") || !strings.Contains(content, "User=") {
		t.Errorf("new section does not use the v2 layout:\n%s", content)
	}

	assignments, err := store.Assignments(ctx)
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := assignments.Roles["dave"]; len(got) != 2 {
		t.Errorf("dave roles = %v", got)
	}

	backups, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	before := len(backups)

	if err := store.SetRoles(ctx, "dave", []model.Role{model.RoleUser, model.RoleAdministrator}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() second call error = %v", err)
	}

	after, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(after) != before {
		t.Errorf("an identical SetRoles() wrote the file (%d backups -> %d)", before, len(after))
	}
}

func TestRemoveUserDropsBothBlocks(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)
	ctx := context.Background()

	if err := store.RemoveUser(ctx, "sam", model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveUser() error = %v", err)
	}

	content := readRoles(t, path)
	if strings.Contains(content, "[sam") || strings.Contains(content, "laneige-coupon:/") {
		t.Errorf("sam survived the delete:\n%s", content)
	}

	if !strings.Contains(content, "[mixmedia]") || !strings.Contains(content, "[bob]") {
		t.Errorf("other sections were removed:\n%s", content)
	}

	if err := store.RemoveUser(ctx, "alice", model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveUser(alice) error = %v", err)
	}

	content = readRoles(t, path)
	if strings.Contains(content, "[alice]") || strings.Contains(content, "Role=User-Group-Manager") {
		t.Errorf("alice survived the delete:\n%s", content)
	}

	assignments, err := store.Assignments(ctx)
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if len(assignments.Roles["alice"]) != 0 {
		t.Errorf("alice still has roles: %v", assignments.Roles["alice"])
	}

	// Removing an unknown user is a no-op, not an error.
	if err := store.RemoveUser(ctx, "nobody", model.WriteOptions{}); err != nil {
		t.Errorf("RemoveUser(unknown) error = %v", err)
	}
}

func TestAccessPathEditing(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)
	ctx := context.Background()

	if err := store.AddAccessPaths(ctx, "sam", []string{"third:/trunk"}, model.WriteOptions{}); err != nil {
		t.Fatalf("AddAccessPaths() error = %v", err)
	}

	content := readRoles(t, path)
	if !strings.Contains(content, "2=third:/trunk") {
		t.Errorf("the next index was not used:\n%s", content)
	}

	preserved(t, writeSample, content, func(line string) bool {
		return strings.HasPrefix(line, "2=")
	})

	// Adding an existing path is a no-op.
	if err := store.AddAccessPaths(ctx, "sam", []string{"third:/trunk"}, model.WriteOptions{}); err != nil {
		t.Fatalf("AddAccessPaths() duplicate error = %v", err)
	}

	if got := strings.Count(readRoles(t, path), "third:/trunk"); got != 1 {
		t.Errorf("duplicate access path written %d times", got)
	}

	if err := store.RemoveAccessPaths(ctx, "sam", []string{"other:/trunk"}, model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveAccessPaths() error = %v", err)
	}

	content = readRoles(t, path)
	if strings.Contains(content, "other:/trunk") {
		t.Errorf("the removed access path survived:\n%s", content)
	}

	// Existing indexes are not renumbered.
	if !strings.Contains(content, "0=laneige-coupon:/") {
		t.Errorf("the remaining index was renumbered:\n%s", content)
	}

	assignments, err := store.Assignments(ctx)
	if err != nil {
		t.Fatalf("Assignments() error = %v", err)
	}

	if got := assignments.AccessPaths["sam"]; len(got) != 2 || got[0] != "laneige-coupon:/" || got[1] != "third:/trunk" {
		t.Errorf("sam access paths = %v", got)
	}
}

func TestSetAccessPathsReplacesTheList(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)
	ctx := context.Background()

	if err := store.SetAccessPaths(ctx, "sam", []string{"fresh:/"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetAccessPaths() error = %v", err)
	}

	content := readRoles(t, path)
	if !strings.Contains(content, "0=fresh:/") {
		t.Errorf("the new list is missing:\n%s", content)
	}

	if strings.Contains(content, "laneige-coupon:/") || strings.Contains(content, "other:/trunk") {
		t.Errorf("old entries survived:\n%s", content)
	}
}

func TestSetRolesRejectsUnknownRoleAndStaleHash(t *testing.T) {
	path, store, _ := newWriteStore(t, writeSample)
	ctx := context.Background()

	if err := store.SetRoles(ctx, "bob", []model.Role{"Nope"}, model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("SetRoles(unknown role) error = %v, want ValidationError", err)
	}

	if err := store.SetRoles(ctx, "../bob", nil, model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("SetRoles(bad name) error = %v, want ValidationError", err)
	}

	stale := model.WriteOptions{ExpectedHash: strings.Repeat("0", 64)}
	if err := store.SetRoles(ctx, "bob", []model.Role{model.RoleUser}, stale); !model.IsConflict(err) {
		t.Errorf("SetRoles(stale hash) error = %v, want ConflictError", err)
	}

	if readRoles(t, path) != writeSample {
		t.Error("a stale write modified the file")
	}
}

func TestSetRolesOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "userroleassignments.ini")
	store := New(path)

	if err := store.SetRoles(context.Background(), "alice", []model.Role{model.RoleUser}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles() error = %v", err)
	}

	content := readRoles(t, path)
	if content != "[alice]\nUser=\n" {
		t.Errorf("content = %q", content)
	}

	if err := store.SetRoles(context.Background(), "alice", nil, model.WriteOptions{}); err != nil {
		t.Fatalf("SetRoles(empty) error = %v", err)
	}

	if content := readRoles(t, path); content != "[alice]\n" {
		t.Errorf("content after removing all roles = %q", content)
	}
}
