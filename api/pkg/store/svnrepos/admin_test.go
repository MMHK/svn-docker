package svnrepos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/svncli"
)

var fakeNow = time.Date(2026, time.November, 4, 12, 0, 0, 0, time.UTC)

const adminListXML = `<?xml version="1.0" encoding="UTF-8"?>
<lists>
<list path="file:///svn/repo1/trunk">
<entry kind="dir">
<name>src</name>
<commit revision="6">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
<entry kind="file">
<name>README &amp; notes.txt</name>
<size>2048</size>
<commit revision="5">
<author>viewer</author>
<date>2026-11-03T09:30:00.000000Z</date>
</commit>
</entry>
</list>
</lists>`

const adminInfoXML = `<?xml version="1.0" encoding="UTF-8"?>
<info>
<entry kind="dir" path="." revision="4">
<url>file:///svn/repo1</url>
<commit revision="4">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
</info>`

const adminLogXML = `<?xml version="1.0" encoding="UTF-8"?>
<log><logentry revision="9"><author>mixmedia</author><date>2026-11-04T11:00:00.000000Z</date></logentry></log>`

type adminCall struct {
	name string
	args []string
}

// adminRunner records argv, materialises `svnadmin create` and answers the
// read-only commands with fixtures.
type adminRunner struct {
	mu      sync.Mutex
	calls   []adminCall
	listXML string
	infoXML string
	logXML  string
	fail    map[string]error
}

func newAdminRunner() *adminRunner {
	return &adminRunner{
		listXML: adminListXML,
		infoXML: adminInfoXML,
		logXML:  adminLogXML,
		fail:    map[string]error{},
	}
}

func (r *adminRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.mu.Lock()
	r.calls = append(r.calls, adminCall{name: name, args: append([]string(nil), args...)})
	op := ""
	if len(args) > 0 {
		op = args[0]
	}

	failErr := r.fail[op]
	listXML, infoXML, logXML := r.listXML, r.infoXML, r.logXML
	r.mu.Unlock()

	if failErr != nil {
		return nil, []byte("svn: E160013: File not found\n"), failErr
	}

	switch {
	case name == "svnadmin" && op == "create":
		path := args[len(args)-1]
		if err := os.MkdirAll(filepath.Join(path, "db"), 0o755); err != nil {
			return nil, nil, err
		}

		if err := os.MkdirAll(filepath.Join(path, "conf"), 0o755); err != nil {
			return nil, nil, err
		}

		if err := os.WriteFile(filepath.Join(path, "format"), []byte("5\n"), 0o644); err != nil {
			return nil, nil, err
		}
	case name == "svn" && op == "list":
		return []byte(listXML), nil, nil
	case name == "svn" && op == "info":
		return []byte(infoXML), nil, nil
	case name == "svn" && op == "log":
		return []byte(logXML), nil, nil
	case name == "svn" && op == "mkdir":
	}

	return nil, nil, nil
}

func (r *adminRunner) snapshot() []adminCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]adminCall, len(r.calls))
	copy(out, r.calls)

	return out
}

func newAdminStore(t *testing.T) (*Store, *adminRunner, string, string) {
	t.Helper()

	parent := t.TempDir()
	trash := filepath.Join(t.TempDir(), "trash")
	runner := newAdminRunner()

	store := NewStore(Options{
		Parents:  []string{parent},
		CLI:      svncli.New(svncli.Options{Runner: runner}),
		TrashDir: trash,
		Now:      func() time.Time { return fakeNow },
	})

	return store, runner, parent, trash
}

func TestCreateRepositoryRunsArgvAndLayout(t *testing.T) {
	store, runner, parent, _ := newAdminStore(t)

	repository, err := store.CreateRepository(context.Background(), model.RepositorySpec{
		Name:      "newrepo",
		Structure: model.StructureSingle,
	})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	wantPath := filepath.Join(parent, "newrepo")
	if repository.Path != wantPath || repository.Parent != parent || !repository.IsRepository {
		t.Errorf("repository = %+v", repository)
	}

	if repository.URL != svncli.FileURL(wantPath) {
		t.Errorf("url = %q", repository.URL)
	}

	if !IsRepository(wantPath) {
		t.Error("the created directory does not look like a repository")
	}

	calls := runner.snapshot()
	if len(calls) != 2 {
		t.Fatalf("calls = %+v", calls)
	}

	wantCreate := "create --fs-type fsfs " + wantPath
	if calls[0].name != "svnadmin" || strings.Join(calls[0].args, " ") != wantCreate {
		t.Errorf("create argv = %s %v", calls[0].name, calls[0].args)
	}

	joined := strings.Join(calls[1].args, " ")
	for _, dir := range []string{"/trunk", "/branches", "/tags"} {
		if !strings.Contains(joined, dir) {
			t.Errorf("mkdir argv lacks %q: %v", dir, calls[1].args)
		}
	}

	if !strings.Contains(joined, "--parents") || !strings.Contains(joined, "-m") {
		t.Errorf("mkdir argv = %v", calls[1].args)
	}
}

func TestCreateRepositoryEscapesParentWithSpaces(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "svn data")

	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	runner := newAdminRunner()
	store := NewStore(Options{
		Parents: []string{parent},
		CLI:     svncli.New(svncli.Options{Runner: runner}),
	})

	repository, err := store.CreateRepository(context.Background(), model.RepositorySpec{
		Name:      "spaced",
		Structure: model.StructureSingle,
	})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	if !strings.Contains(repository.URL, "svn%20data") {
		t.Errorf("url = %q, want the escaped parent", repository.URL)
	}

	calls := runner.snapshot()
	joined := strings.Join(calls[1].args, " ")
	if !strings.Contains(joined, "svn%20data/spaced/trunk") {
		t.Errorf("mkdir argv = %v, want an escaped URL", calls[1].args)
	}
}

func TestCreateRepositoryMultiLayout(t *testing.T) {
	store, runner, _, _ := newAdminStore(t)

	if _, err := store.CreateRepository(context.Background(), model.RepositorySpec{
		Name:        "multi",
		Structure:   model.StructureMulti,
		ProjectName: "proj",
	}); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	calls := runner.snapshot()
	joined := strings.Join(calls[1].args, " ")

	for _, dir := range []string{"/proj/trunk", "/proj/branches", "/proj/tags"} {
		if !strings.Contains(joined, dir) {
			t.Errorf("mkdir argv lacks %q: %v", dir, calls[1].args)
		}
	}
}

func TestCreateRepositoryConflictAndValidation(t *testing.T) {
	store, runner, parent, _ := newAdminStore(t)
	makeRepository(t, parent, "taken")

	_, err := store.CreateRepository(context.Background(), model.RepositorySpec{Name: "taken"})
	if !model.IsConflict(err) {
		t.Errorf("duplicate error = %v, want ConflictError", err)
	}

	if len(runner.snapshot()) != 0 {
		t.Error("svnadmin ran for a duplicate name")
	}

	_, err = store.CreateRepository(context.Background(), model.RepositorySpec{Name: "../escape"})
	if !model.IsValidation(err) {
		t.Errorf("invalid name error = %v, want ValidationError", err)
	}

	missing := NewStore(Options{
		Parents: []string{filepath.Join(t.TempDir(), "missing")},
		CLI:     svncli.New(svncli.Options{Runner: newAdminRunner()}),
	})

	if _, err := missing.CreateRepository(context.Background(), model.RepositorySpec{Name: "x"}); !model.IsValidation(err) {
		t.Errorf("missing parent error = %v, want ValidationError", err)
	}

	readOnly := New(parent)
	if _, err := readOnly.CreateRepository(context.Background(), model.RepositorySpec{Name: "x"}); !errors.Is(err, model.ErrReadOnly) {
		t.Errorf("read-only error = %v, want ErrReadOnly", err)
	}
}

func TestDeleteRepositoryMovesToTrash(t *testing.T) {
	parent := t.TempDir()
	trash := filepath.Join(t.TempDir(), "trash")
	path := makeRepository(t, parent, "gone")

	store := NewStore(Options{Parents: []string{parent}, TrashDir: trash, Now: func() time.Time { return fakeNow }})

	deletion, err := store.DeleteRepository(context.Background(), "gone")
	if err != nil {
		t.Fatalf("DeleteRepository() error = %v", err)
	}

	want := filepath.Join(trash, "gone-20261104T120000Z")
	if deletion.TrashPath != want {
		t.Errorf("trash path = %q, want %q", deletion.TrashPath, want)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the repository is still in the parent: %v", err)
	}

	// The data is preserved, not deleted.
	if _, err := os.Stat(filepath.Join(want, "format")); err != nil {
		t.Errorf("format file missing in the trash: %v", err)
	}

	if _, err := store.GetRepository(context.Background(), "gone"); !model.IsNotFound(err) {
		t.Errorf("GetRepository() after delete = %v, want NotFoundError", err)
	}
}

func TestDeleteRepositorySameSecondSuffix(t *testing.T) {
	parent := t.TempDir()
	trash := t.TempDir()
	makeRepository(t, parent, "gone")

	existing := filepath.Join(trash, "gone-20261104T120000Z")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	store := NewStore(Options{Parents: []string{parent}, TrashDir: trash, Now: func() time.Time { return fakeNow }})

	deletion, err := store.DeleteRepository(context.Background(), "gone")
	if err != nil {
		t.Fatalf("DeleteRepository() error = %v", err)
	}

	if want := existing + "-2"; deletion.TrashPath != want {
		t.Errorf("trash path = %q, want %q", deletion.TrashPath, want)
	}
}

func TestDeleteRepositoryCrossDeviceNeedsManualMove(t *testing.T) {
	parent := t.TempDir()
	trash := t.TempDir()
	path := makeRepository(t, parent, "gone")

	store := NewStore(Options{
		Parents:  []string{parent},
		TrashDir: trash,
		Now:      func() time.Time { return fakeNow },
		Rename: func(_, _ string) error {
			return &os.LinkError{Op: "rename", Old: path, New: trash, Err: syscall.EXDEV}
		},
	})

	_, err := store.DeleteRepository(context.Background(), "gone")
	if !model.IsConflict(err) {
		t.Fatalf("error = %v (%T), want ConflictError", err, err)
	}

	if !strings.Contains(err.Error(), "manually") {
		t.Errorf("error %q does not ask for a manual move", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("the repository was removed despite the failure: %v", statErr)
	}
}

func TestDeleteRepositoryRequiresTrashAndExists(t *testing.T) {
	parent := t.TempDir()
	makeRepository(t, parent, "gone")

	store := New(parent)
	if _, err := store.DeleteRepository(context.Background(), "gone"); err == nil {
		t.Error("DeleteRepository() without a trash directory = nil, want an error")
	}

	withTrash := NewStore(Options{Parents: []string{parent}, TrashDir: t.TempDir()})
	if _, err := withTrash.DeleteRepository(context.Background(), "missing"); !model.IsNotFound(err) {
		t.Errorf("missing repository error = %v, want NotFoundError", err)
	}
}

func TestTreeUsesHeadAndParsesEntries(t *testing.T) {
	store, runner, parent, _ := newAdminStore(t)
	makeRepository(t, parent, "repo1")

	tree, err := store.Tree(context.Background(), "repo1", "trunk", 0)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}

	if tree.Revision != 9 || tree.Path != "trunk" {
		t.Errorf("tree = %+v", tree)
	}

	if len(tree.Entries) != 2 {
		t.Fatalf("entries = %+v", tree.Entries)
	}

	if tree.Entries[1].Name != "README & notes.txt" || tree.Entries[1].Size != 2048 || tree.Entries[1].Kind != "file" {
		t.Errorf("file entry = %+v", tree.Entries[1])
	}

	if tree.Entries[0].Name != "src" || tree.Entries[0].Kind != "dir" {
		t.Errorf("dir entry = %+v", tree.Entries[0])
	}

	calls := runner.snapshot()
	list := calls[len(calls)-1]
	rootURL := svncli.FileURL(filepath.Join(parent, "repo1"))
	wantTarget := rootURL + "/trunk"
	if !strings.Contains(strings.Join(list.args, " "), wantTarget) {
		t.Errorf("list argv = %v, want target %q", list.args, wantTarget)
	}

	// HEAD is resolved at the repository root, never at the subdirectory.
	log := calls[len(calls)-2]
	if log.args[0] != "log" || log.args[len(log.args)-1] != rootURL {
		t.Errorf("log argv = %v, want the root URL %q", log.args, rootURL)
	}

	// A revision skips the HEAD lookup entirely.
	before := len(runner.snapshot())

	if _, err := store.Tree(context.Background(), "repo1", "", 3); err != nil {
		t.Fatalf("Tree(rev=3) error = %v", err)
	}

	after := runner.snapshot()
	if len(after)-before != 1 {
		t.Errorf("rev=3 ran %d commands, want 1", len(after)-before)
	}

	if !strings.Contains(strings.Join(after[len(after)-1].args, " "), "-r 3") {
		t.Errorf("rev=3 argv = %v", after[len(after)-1].args)
	}
}

func TestTreeEmptyDirectory(t *testing.T) {
	store, runner, parent, _ := newAdminStore(t)
	makeRepository(t, parent, "repo1")

	runner.listXML = `<?xml version="1.0"?><lists><list path="file:///svn/repo1/trunk"/></lists>`

	tree, err := store.Tree(context.Background(), "repo1", "trunk", 9)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}

	if len(tree.Entries) != 0 {
		t.Errorf("entries = %+v", tree.Entries)
	}
}

func TestTreeMapsMissingPathToNotFound(t *testing.T) {
	store, runner, parent, _ := newAdminStore(t)
	makeRepository(t, parent, "repo1")

	runner.fail["list"] = &svncli.Error{Op: "list", Command: "svn", ExitCode: 1, Code: "E160013"}

	if _, err := store.Tree(context.Background(), "repo1", "nope", 9); !model.IsNotFound(err) {
		t.Errorf("Tree() error = %v, want NotFoundError", err)
	}
}

func TestInfoReadsHeadAndLastChange(t *testing.T) {
	store, _, parent, _ := newAdminStore(t)
	makeRepository(t, parent, "repo1")

	info, err := store.Info(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.HeadRevision != 9 {
		t.Errorf("head = %d, want 9 (svn log), not the entry revision 4", info.HeadRevision)
	}

	if info.LastChangedRev != 4 || info.LastChangedAuthor != "mixmedia" || info.LastChangedAt.IsZero() {
		t.Errorf("last changed = %+v", info)
	}

	if info.URL != svncli.FileURL(filepath.Join(parent, "repo1")) {
		t.Errorf("url = %q", info.URL)
	}
}

func TestReadOnlyStoreIsReadOnly(t *testing.T) {
	store := New(t.TempDir())

	if _, err := store.Tree(context.Background(), "repo1", "", 0); !errors.Is(err, model.ErrReadOnly) {
		t.Errorf("Tree() error = %v, want ErrReadOnly", err)
	}

	if _, err := store.Info(context.Background(), "repo1"); !errors.Is(err, model.ErrReadOnly) {
		t.Errorf("Info() error = %v, want ErrReadOnly", err)
	}
}
