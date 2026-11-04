package authz

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

const sample = `# hand maintained
[groups]
devs = alice, bob
ops = carol

[repo1:/]
@devs = rw
alice = r
$authenticated = r

[repo1:/trunk]
bob = rw
&everyone = r
; unknown lines and principals must survive

[repo2:/secret]
* =
`

func TestParseStructure(t *testing.T) {
	file := Parse("access", []byte(sample))

	wantPaths := []string{"repo1:/", "repo1:/trunk", "repo2:/secret"}
	gotPaths := file.Paths()
	if len(gotPaths) != len(wantPaths) {
		t.Fatalf("Paths() = %v, want %v", gotPaths, wantPaths)
	}

	for i := range wantPaths {
		if gotPaths[i] != wantPaths[i] {
			t.Errorf("Paths()[%d] = %q, want %q", i, gotPaths[i], wantPaths[i])
		}
	}

	groups := file.Groups()
	if len(groups) != 2 || groups[0] != "devs" || groups[1] != "ops" {
		t.Errorf("Groups() = %v", groups)
	}

	devs := file.UsersOfGroup("devs")
	if len(devs) != 2 || devs[0] != "alice" || devs[1] != "bob" {
		t.Errorf("UsersOfGroup(devs) = %v", devs)
	}

	acl := file.ACL("repo1:/")
	if len(acl) != 3 {
		t.Fatalf("ACL(repo1:/) = %+v", acl)
	}

	if acl[0].Principal != "@devs" || acl[0].Permission != model.PermissionReadWrite {
		t.Errorf("acl[0] = %+v", acl[0])
	}

	if acl[1].Principal != "alice" || acl[1].Permission != model.PermissionRead {
		t.Errorf("acl[1] = %+v", acl[1])
	}

	if acl[2].Principal != "$authenticated" || acl[2].Permission != model.PermissionRead {
		t.Errorf("acl[2] = %+v", acl[2])
	}

	trunk := file.ACL("repo1:/trunk")
	if len(trunk) != 2 || trunk[1].Principal != "&everyone" {
		t.Errorf("trunk ACL = %+v", trunk)
	}

	secret := file.ACL("repo2:/secret")
	if len(secret) != 1 || secret[0].Principal != "*" || secret[0].Permission != model.PermissionNone {
		t.Errorf("secret ACL = %+v", secret)
	}

	if got := file.ACL("missing:/path"); got != nil {
		t.Errorf("ACL(missing) = %+v, want nil", got)
	}
}

func TestParseIsLossless(t *testing.T) {
	data := "[groups]\r\ndevs = alice\r\n\r\n[repo1:/]\r\n  alice = r \r\n"

	file := Parse("access", []byte(data))

	var rebuilt strings.Builder
	for _, line := range file.Lines {
		rebuilt.WriteString(line.Raw)
		rebuilt.WriteString(line.EOL)
	}

	if rebuilt.String() != data {
		t.Errorf("reassembled content differs:\n got %q\nwant %q", rebuilt.String(), data)
	}
}

func TestPermissionOf(t *testing.T) {
	tests := map[string]model.Permission{
		"rw": model.PermissionReadWrite,
		"r":  model.PermissionRead,
		"":   model.PermissionNone,
		"  ": model.PermissionNone,
		"w":  model.PermissionReadWrite,
		"??": model.PermissionNone,
	}

	for value, want := range tests {
		if got := PermissionOf(value); got != want {
			t.Errorf("PermissionOf(%q) = %q, want %q", value, got, want)
		}
	}
}

func writeAccessFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "svn-access-file")
	if err := os.WriteFile(path, []byte(sample), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}

func TestStoreGroups(t *testing.T) {
	store := New(writeAccessFile(t))
	ctx := context.Background()

	page, err := store.ListGroups(ctx, model.PageQuery{Q: "d"})
	if err != nil {
		t.Fatalf("ListGroups() error = %v", err)
	}

	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "devs" {
		t.Fatalf("filtered page = %+v", page)
	}

	group, err := store.GetGroup(ctx, "devs")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}

	if len(group.Members) != 2 || group.Members[1] != "bob" {
		t.Errorf("group = %+v", group)
	}

	if _, err := store.GetGroup(ctx, "nope"); !model.IsNotFound(err) {
		t.Errorf("GetGroup(nope) error = %v, want NotFoundError", err)
	}

	groups, err := store.GroupsOfUser(ctx, "carol")
	if err != nil {
		t.Fatalf("GroupsOfUser() error = %v", err)
	}

	if len(groups) != 1 || groups[0] != "ops" {
		t.Errorf("GroupsOfUser(carol) = %v", groups)
	}
}

func TestStoreAccessPaths(t *testing.T) {
	store := New(writeAccessFile(t))
	ctx := context.Background()

	page, err := store.ListAccessPaths(ctx, model.PageQuery{Size: 2})
	if err != nil {
		t.Fatalf("ListAccessPaths() error = %v", err)
	}

	if page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("page = %+v", page)
	}

	path, err := store.GetAccessPath(ctx, "repo1:/trunk")
	if err != nil {
		t.Fatalf("GetAccessPath() error = %v", err)
	}

	if path.Path != "repo1:/trunk" || len(path.ACL) != 2 {
		t.Errorf("access path = %+v", path)
	}

	if _, err := store.GetAccessPath(ctx, "missing:/x"); !model.IsNotFound(err) {
		t.Errorf("GetAccessPath(missing) error = %v, want NotFoundError", err)
	}

	paths, err := store.AccessPathsOfGroup(ctx, "devs")
	if err != nil {
		t.Fatalf("AccessPathsOfGroup() error = %v", err)
	}

	if len(paths) != 1 || paths[0] != "repo1:/" {
		t.Errorf("AccessPathsOfGroup(devs) = %v", paths)
	}
}

func TestStoreMissingFileIsTypedError(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "missing"))

	_, err := store.ListAccessPaths(context.Background(), model.PageQuery{})
	if err == nil {
		t.Fatal("ListAccessPaths() error = nil, want a read error")
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatalf("error = %T, want *authz.Error", err)
	}
}
