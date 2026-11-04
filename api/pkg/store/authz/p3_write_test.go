package authz

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const p3Fixture = `# svn authz, hand maintained
[groups]
devs = alice, bob
devs = carol
ops = ops1
; a comment inside the groups block

[repo1:/]
; section comment
@devs = rw
alice = r
$authenticated = r
&everyone = r

[weird]
unknown = kept

[repo2:/trunk]
devs = r
* =
`

// TestCreateGroupIsLossless verifies that adding a group inserts exactly one
// line and leaves every existing byte untouched.
func TestCreateGroupIsLossless(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)

	if err := store.CreateGroup(context.Background(), model.Group{Name: "newteam"}, model.WriteOptions{}); err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	content := readAuthz(t, path)
	if !strings.Contains(content, "newteam =") {
		t.Fatalf("group was not created:\n%s", content)
	}

	// Removing the one added line must give back the original file exactly.
	got := strings.Replace(content, "newteam =\n", "", 1)
	if got != p3Fixture {
		t.Errorf("unrelated content changed:\n--- got ---\n%s\n--- want ---\n%s", got, p3Fixture)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !file.HasGroup("newteam") || len(file.UsersOfGroup("newteam")) != 0 {
		t.Errorf("newteam = %+v", file.UsersOfGroup("newteam"))
	}

	if err := store.CreateGroup(context.Background(), model.Group{Name: "newteam"}, model.WriteOptions{}); !model.IsConflict(err) {
		t.Errorf("duplicate CreateGroup() error = %v, want ConflictError", err)
	}
}

func TestCreateGroupCreatesMissingSection(t *testing.T) {
	const noGroups = "# top\n[repo:/]\nalice = r\n"
	path, store, _ := newWriteStore(t, noGroups)

	if err := store.CreateGroup(context.Background(), model.Group{Name: "devs"}, model.WriteOptions{}); err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	content := readAuthz(t, path)
	if !strings.Contains(content, "[groups]\ndevs =") {
		t.Fatalf("the section was not appended:\n%s", content)
	}

	if !strings.HasPrefix(content, noGroups) {
		t.Errorf("the original content was rewritten:\n%s", content)
	}
}

// TestSetGroupMembersConsolidatesDuplicates checks the duplicate-key rule: the
// first line is rewritten, later copies are dropped, other lines stay put.
func TestSetGroupMembersConsolidatesDuplicates(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)

	if err := store.SetGroupMembers(context.Background(), "devs", []string{"dave", "alice"}, model.WriteOptions{}); err != nil {
		t.Fatalf("SetGroupMembers() error = %v", err)
	}

	content := readAuthz(t, path)

	if strings.Contains(content, "devs = carol") || strings.Contains(content, "devs = alice, bob") {
		t.Errorf("duplicate devs lines were not consolidated:\n%s", content)
	}

	if strings.Count(content, "devs = dave, alice") != 1 {
		t.Errorf("members were not replaced:\n%s", content)
	}

	// The same-named user line in repo2 is not a group definition.
	if !strings.Contains(content, "[repo2:/trunk]\ndevs = r\n") {
		t.Errorf("a user entry was mistaken for a group line:\n%s", content)
	}

	assertPreserved(t, p3Fixture, content, func(line string) bool {
		return strings.HasPrefix(line, "devs =")
	})

	if err := store.SetGroupMembers(context.Background(), "missing", []string{"a"}, model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("missing group error = %v, want NotFoundError", err)
	}
}

func TestDeleteGroupRemovesACLButNotUserEntries(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)

	if err := store.DeleteGroup(context.Background(), "devs", model.WriteOptions{}); err != nil {
		t.Fatalf("DeleteGroup() error = %v", err)
	}

	content := readAuthz(t, path)
	if strings.Contains(content, "@devs") {
		t.Errorf("@devs survived the group delete:\n%s", content)
	}

	if strings.Contains(content, "devs = alice") || strings.Contains(content, "devs = carol") {
		t.Errorf("group definitions survived:\n%s", content)
	}

	if !strings.Contains(content, "[repo2:/trunk]\ndevs = r\n") {
		t.Errorf("the same-named user entry was deleted:\n%s", content)
	}

	assertPreserved(t, p3Fixture, content, func(line string) bool {
		return strings.HasPrefix(line, "@devs") ||
			strings.HasPrefix(line, "devs = alice") ||
			strings.HasPrefix(line, "devs = carol")
	})

	if err := store.DeleteGroup(context.Background(), "missing", model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("missing group error = %v, want NotFoundError", err)
	}
}

func TestCreateAndDeleteAccessPath(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)
	ctx := context.Background()

	if err := store.CreateAccessPath(ctx, "project:/nested/trunk", model.WriteOptions{}); err != nil {
		t.Fatalf("CreateAccessPath() error = %v", err)
	}

	content := readAuthz(t, path)
	if !strings.HasSuffix(content, "[project:/nested/trunk]\n") {
		t.Errorf("section was not appended:\n%s", content)
	}

	if err := store.CreateAccessPath(ctx, "project:/nested/trunk", model.WriteOptions{}); !model.IsConflict(err) {
		t.Errorf("duplicate CreateAccessPath() error = %v, want ConflictError", err)
	}

	if err := store.DeleteAccessPath(ctx, "project:/nested/trunk", model.WriteOptions{}); err != nil {
		t.Fatalf("DeleteAccessPath() error = %v", err)
	}

	if got := readAuthz(t, path); got != p3Fixture {
		t.Errorf("deleting the new section did not restore the fixture:\n--- got ---\n%s", got)
	}

	if err := store.DeleteAccessPath(ctx, "missing:/x", model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("missing path error = %v, want NotFoundError", err)
	}
}

func TestDeleteAccessPathRemovesSectionAndEntries(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)

	if err := store.DeleteAccessPath(context.Background(), "repo1:/", model.WriteOptions{}); err != nil {
		t.Fatalf("DeleteAccessPath() error = %v", err)
	}

	content := readAuthz(t, path)
	for _, needle := range []string{"[repo1:/]", "@devs = rw", "alice = r\n", "$authenticated", "&everyone"} {
		if strings.Contains(content, needle) {
			t.Errorf("%q survived the section delete:\n%s", needle, content)
		}
	}

	if !strings.Contains(content, "[weird]\nunknown = kept") || !strings.Contains(content, "[repo2:/trunk]") {
		t.Errorf("other sections were touched:\n%s", content)
	}
}

func TestSetACLAaddsUpdatesAndRemoves(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)
	ctx := context.Background()

	// Update an existing entry: only that line changes.
	if err := store.SetACL(ctx, "repo1:/", "alice", model.PermissionReadWrite, model.WriteOptions{}); err != nil {
		t.Fatalf("SetACL(update) error = %v", err)
	}

	content := readAuthz(t, path)
	want := strings.Replace(p3Fixture, "alice = r\n", "alice = rw\n", 1)
	if content != want {
		t.Errorf("a single entry update rewrote other lines:\n--- got ---\n%s", content)
	}

	// Add a new entry to an existing section.
	if err := store.SetACL(ctx, "repo1:/", "carol", model.PermissionRead, model.WriteOptions{}); err != nil {
		t.Fatalf("SetACL(add) error = %v", err)
	}

	if !strings.Contains(readAuthz(t, path), "carol = r\n") {
		t.Error("the new entry was not written")
	}

	// perm=none removes the line.
	if err := store.SetACL(ctx, "repo1:/", "carol", model.PermissionNone, model.WriteOptions{}); err != nil {
		t.Fatalf("SetACL(none) error = %v", err)
	}

	if strings.Contains(readAuthz(t, path), "carol =") {
		t.Error("perm=none left the line behind")
	}

	// Adding to an empty section starts right below its header.
	if err := store.SetACL(ctx, "repo2:/trunk", "newbie", model.PermissionReadWrite, model.WriteOptions{}); err != nil {
		t.Fatalf("SetACL(empty section) error = %v", err)
	}

	if !strings.Contains(readAuthz(t, path), "* =\nnewbie = rw\n") {
		t.Errorf("entry was not appended to the existing section")
	}

	if err := store.SetACL(ctx, "missing:/x", "alice", model.PermissionRead, model.WriteOptions{}); !model.IsNotFound(err) {
		t.Errorf("missing path error = %v, want NotFoundError", err)
	}
}

func TestSetACLRejectsReadOnlyPrincipals(t *testing.T) {
	_, store, _ := newWriteStore(t, p3Fixture)
	ctx := context.Background()

	for _, principal := range []string{"$authenticated", "$anonymous", "&everyone", "bad name", ""} {
		if err := store.SetACL(ctx, "repo1:/", principal, model.PermissionRead, model.WriteOptions{}); !model.IsValidation(err) {
			t.Errorf("SetACL(%q) error = %v, want ValidationError", principal, err)
		}

		if err := store.DeleteACL(ctx, "repo1:/", principal, model.WriteOptions{}); !model.IsValidation(err) {
			t.Errorf("DeleteACL(%q) error = %v, want ValidationError", principal, err)
		}
	}

	if err := store.SetACL(ctx, "repo1:/", "alice", "write", model.WriteOptions{}); !model.IsValidation(err) {
		t.Errorf("bad permission error = %v, want ValidationError", err)
	}
}

func TestDeleteACLIsIdempotent(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)
	ctx := context.Background()

	if err := store.DeleteACL(ctx, "repo1:/", "alice", model.WriteOptions{}); err != nil {
		t.Fatalf("DeleteACL() error = %v", err)
	}

	if strings.Contains(readAuthz(t, path), "alice =") {
		t.Error("the entry survived")
	}

	before := readAuthz(t, path)

	if err := store.DeleteACL(ctx, "repo1:/", "alice", model.WriteOptions{}); err != nil {
		t.Fatalf("second DeleteACL() error = %v", err)
	}

	if got := readAuthz(t, path); got != before {
		t.Error("removing an absent entry changed the file")
	}
}

func TestAuthzWriteHonoursExpectedHashForP3(t *testing.T) {
	path, store, _ := newWriteStore(t, p3Fixture)
	ctx := context.Background()
	stale := model.WriteOptions{ExpectedHash: strings.Repeat("0", 64)}

	for name, run := range map[string]func() error{
		"group": func() error {
			return store.CreateGroup(ctx, model.Group{Name: "x"}, stale)
		},
		"path": func() error {
			return store.CreateAccessPath(ctx, "repo3:/", stale)
		},
		"acl": func() error {
			return store.SetACL(ctx, "repo1:/", "alice", model.PermissionRead, stale)
		},
	} {
		if err := run(); !model.IsConflict(err) {
			t.Errorf("%s stale write error = %v, want ConflictError", name, err)
		}
	}

	if got := readAuthz(t, path); got != p3Fixture {
		t.Error("a stale write modified the file")
	}
}

// TestConcurrentACLWritesLoseNothing is the plan section 7.2 check 9 analogue at
// the store layer: 20 writers editing different principals must all land.
func TestConcurrentACLWritesLoseNothing(t *testing.T) {
	path, store, _ := newWriteStore(t, "[repo:/]\n")
	ctx := context.Background()

	const writers = 20

	var wait sync.WaitGroup

	errs := make(chan error, writers)

	for index := 0; index < writers; index++ {
		wait.Add(1)

		go func(number int) {
			defer wait.Done()

			principal := fmt.Sprintf("user%02d", number)
			if err := store.SetACL(ctx, "repo:/", principal, model.PermissionRead, model.WriteOptions{}); err != nil {
				errs <- err
			}
		}(index)
	}

	wait.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent SetACL() error = %v", err)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	acl := file.ACL("repo:/")
	if len(acl) != writers {
		t.Fatalf("ACL has %d entries, want %d:\n%s", len(acl), writers, readAuthz(t, path))
	}

	seen := make(map[string]bool, writers)
	for _, entry := range acl {
		seen[entry.Principal] = true
	}

	for index := 0; index < writers; index++ {
		principal := fmt.Sprintf("user%02d", index)
		if !seen[principal] {
			t.Errorf("%s was lost", principal)
		}
	}
}

func TestAuthzP3KeepsCRLFAndUnknownContent(t *testing.T) {
	fixture := strings.ReplaceAll(p3Fixture, "\n", "\r\n")

	path, store, _ := newWriteStore(t, fixture)
	ctx := context.Background()

	if err := store.SetACL(ctx, "repo1:/", "alice", model.PermissionReadWrite, model.WriteOptions{}); err != nil {
		t.Fatalf("SetACL() error = %v", err)
	}

	if err := store.CreateGroup(ctx, model.Group{Name: "crlfteam"}, model.WriteOptions{}); err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}

	if err := store.CreateAccessPath(ctx, "repo3:/crlf", model.WriteOptions{}); err != nil {
		t.Fatalf("CreateAccessPath() error = %v", err)
	}

	content := readAuthz(t, path)

	if strings.Count(content, "\n") != strings.Count(content, "\r\n") {
		t.Errorf("some lines lost their CRLF ending:\n%q", content)
	}

	for _, needle := range []string{
		"# svn authz, hand maintained",
		"; a comment inside the groups block",
		"; section comment",
		"$authenticated = r",
		"&everyone = r",
		"[weird]",
		"unknown = kept",
		"devs = r",
		"* =",
	} {
		if !strings.Contains(content, strings.ReplaceAll(needle, "\n", "\r\n")) {
			t.Errorf("unknown or read-only content was lost: %q\n%s", needle, content)
		}
	}
}
