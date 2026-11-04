package authz

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const writeFixture = `# svn authz, maintained by hand
[groups]
devs = testuser, viewer
ops = ops1
solo = testuser
empty =

[/]
* = r

[repo1:/]
@devs = rw
testuser = r

[repo2:/trunk]
viewer = r
testuser = rw
`

func newWriteStore(t *testing.T, content string) (string, *Store, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "svn-access-file")

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	backupDir := filepath.Join(dir, "backup")

	return path, New(path, WithBackupDir(backupDir)), backupDir
}

func readAuthz(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	return string(data)
}

func assertPreserved(t *testing.T, before, after string, changed func(string) bool) {
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

func TestSetGroupsOfUser(t *testing.T) {
	path, store, _ := newWriteStore(t, writeFixture)
	ctx := context.Background()

	if err := store.SetGroupsOfUser(ctx, "testuser", []string{"devs", "ops"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser() error = %v", err)
	}

	content := readAuthz(t, path)

	if !strings.Contains(content, "devs = testuser, viewer") {
		t.Errorf("the unchanged group line was rewritten:\n%s", content)
	}

	if !strings.Contains(content, "ops = ops1, testuser") {
		t.Errorf("the membership was not appended:\n%s", content)
	}

	assertPreserved(t, writeFixture, content, func(line string) bool {
		return strings.HasPrefix(line, "ops =") || strings.HasPrefix(line, "solo =")
	})

	groups, err := store.GroupsOfUser(ctx, "testuser")
	if err != nil {
		t.Fatalf("GroupsOfUser() error = %v", err)
	}

	if len(groups) != 2 || groups[0] != "devs" || groups[1] != "ops" {
		t.Errorf("groups = %v", groups)
	}
}

func TestSetGroupsOfUserRemovesOthers(t *testing.T) {
	path, store, _ := newWriteStore(t, writeFixture)
	ctx := context.Background()

	if err := store.SetGroupsOfUser(ctx, "testuser", []string{"ops"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser() error = %v", err)
	}

	content := readAuthz(t, path)

	// viewer keeps the membership; only testuser is removed.
	if !strings.Contains(content, "devs = viewer") {
		t.Errorf("devs = %q, want only viewer left:\n%s", lineWithPrefix(content, "devs ="), content)
	}

	groups, err := store.GroupsOfUser(ctx, "testuser")
	if err != nil {
		t.Fatalf("GroupsOfUser() error = %v", err)
	}

	if len(groups) != 1 || groups[0] != "ops" {
		t.Errorf("groups = %v", groups)
	}

	// Removing the last member drops the line but no other group definition.
	if err := store.SetGroupsOfUser(ctx, "testuser", nil, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser(empty) error = %v", err)
	}

	content = readAuthz(t, path)

	if strings.Contains(content, "solo =") {
		t.Errorf("the emptied group line survived:\n%s", content)
	}

	if !strings.Contains(content, "devs = viewer") || !strings.Contains(content, "empty =") {
		t.Errorf("other groups were touched:\n%s", content)
	}

	if !strings.Contains(content, "ops = ops1") {
		t.Errorf("ops lost its remaining member:\n%s", content)
	}
}

func TestSetGroupsOfUserAddsMissingGroup(t *testing.T) {
	path, store, _ := newWriteStore(t, writeFixture)

	if err := store.SetGroupsOfUser(context.Background(), "newbie", []string{"devs", "brandnew"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser() error = %v", err)
	}

	content := readAuthz(t, path)

	if !strings.Contains(content, "devs = testuser, viewer, newbie") {
		t.Errorf("membership not appended to devs:\n%s", content)
	}

	if !strings.Contains(content, "brandnew = newbie") {
		t.Errorf("missing group definition:\n%s", content)
	}

	assertPreserved(t, writeFixture, content, func(line string) bool {
		return strings.HasPrefix(line, "devs =") || strings.HasPrefix(line, "brandnew =")
	})
}

func TestSetGroupsOfUserRejectsBadNames(t *testing.T) {
	_, store, _ := newWriteStore(t, writeFixture)
	ctx := context.Background()

	if err := store.SetGroupsOfUser(ctx, "bad name", []string{"devs"}, model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("bad user error = %v, want ValidationError", err)
	}

	for _, group := range []string{"", "a b", "a=b", "a,b", "@devs"} {
		if err := store.SetGroupsOfUser(ctx, "testuser", []string{group}, model.WriteOptions{}); !model.IsValidation(err) {
			t.Errorf("bad group %q error = %v, want ValidationError", group, err)
		}
	}
}

func TestRemoveUserStripsGroupsAndACL(t *testing.T) {
	path, store, _ := newWriteStore(t, writeFixture)
	ctx := context.Background()

	if err := store.RemoveUser(ctx, "testuser", model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveUser() error = %v", err)
	}

	content := readAuthz(t, path)

	if strings.Contains(content, "testuser") {
		t.Errorf("the user survived the delete:\n%s", content)
	}

	if !strings.Contains(content, "devs = viewer") {
		t.Errorf("the remaining group members were lost:\n%s", content)
	}

	// Nothing else changed: the * rule, the sections and the comments stay.
	assertPreserved(t, writeFixture, content, func(line string) bool {
		return strings.Contains(line, "testuser")
	})

	file, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, section := range file.Sections {
		for _, entry := range section.Entries {
			if entry.Principal == "testuser" {
				t.Errorf("ACL entry survived in %s", section.Name)
			}
		}
	}

	// A missing user is a no-op.
	if err := store.RemoveUser(ctx, "nobody", model.WriteOptions{}); err != nil {
		t.Errorf("RemoveUser(unknown) error = %v", err)
	}
}

func TestRemoveUserHandlesOnlyACL(t *testing.T) {
	const aclOnly = "[repo:/]\nalice = rw\nbob = r\n"

	path, store, _ := newWriteStore(t, aclOnly)

	if err := store.RemoveUser(context.Background(), "alice", model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveUser() error = %v", err)
	}

	if content := readAuthz(t, path); content != "[repo:/]\nbob = r\n" {
		t.Errorf("content = %q", content)
	}

	// The file has no [groups] section at all.
	if err := store.RemoveUser(context.Background(), "bob", model.WriteOptions{}); err != nil {
		t.Fatalf("RemoveUser(second) error = %v", err)
	}

	if content := readAuthz(t, path); content != "[repo:/]\n" {
		t.Errorf("content = %q", content)
	}
}

func TestRemoveUserCreatesNoGroupsSection(t *testing.T) {
	const noGroups = "[repo:/]\nviewer = r\n"

	path, store, _ := newWriteStore(t, noGroups)

	if err := store.SetGroupsOfUser(context.Background(), "viewer", []string{"devs"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser() error = %v", err)
	}

	content := readAuthz(t, path)
	if !strings.Contains(content, "[groups]") || !strings.Contains(content, "devs = viewer") {
		t.Fatalf("a missing [groups] section was not created:\n%s", content)
	}

	assertPreserved(t, noGroups, content, nil)
}

func TestAuthzWriteHonoursExpectedHash(t *testing.T) {
	path, store, _ := newWriteStore(t, writeFixture)

	stale := model.WriteOptions{ExpectedHash: strings.Repeat("0", 64)}

	if err := store.RemoveUser(context.Background(), "testuser", stale); !model.IsConflict(err) {
		t.Fatalf("RemoveUser(stale) error = %v, want ConflictError", err)
	}

	if readAuthz(t, path) != writeFixture {
		t.Error("a stale write modified the file")
	}
}

func TestAuthzWriteKeepsCRLF(t *testing.T) {
	fixture := strings.ReplaceAll(writeFixture, "\n", "\r\n")

	path, store, _ := newWriteStore(t, fixture)

	if err := store.SetGroupsOfUser(context.Background(), "testuser", []string{"ops"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupsOfUser() error = %v", err)
	}

	content := readAuthz(t, path)

	if strings.Count(content, "\n") != strings.Count(content, "\r\n") {
		t.Errorf("some lines lost their CRLF ending:\n%q", content)
	}
}

func lineWithPrefix(content, prefix string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSuffix(line, "\r")
		}
	}

	return ""
}
