package svnrepos

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/svncli"
)

// TestIntegrationRealSubversion exercises the real svn/svnadmin binaries. It
// is skipped with -short and whenever the tools are not installed, so the unit
// suite stays hermetic.
func TestIntegrationRealSubversion(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test disabled by -short")
	}

	svnBinary, err := exec.LookPath("svn")
	if err != nil {
		t.Skip("svn is not installed")
	}

	adminBinary, err := exec.LookPath("svnadmin")
	if err != nil {
		t.Skip("svnadmin is not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	parent := t.TempDir()
	trash := filepath.Join(t.TempDir(), "trash")

	store := NewStore(Options{
		Parents:  []string{parent},
		CLI:      svncli.New(svncli.Options{SVN: svnBinary, SVNAdmin: adminBinary}),
		TrashDir: trash,
	})

	repository, err := store.CreateRepository(ctx, model.RepositorySpec{
		Name:      "it-repo",
		Structure: model.StructureSingle,
	})
	if err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	if !IsRepository(repository.Path) {
		t.Fatalf("%s is not recognised as a repository", repository.Path)
	}

	t.Logf("created %s (parent %s, url %s)", repository.Path, repository.Parent, repository.URL)

	tree, err := store.Tree(ctx, "it-repo", "", 0)
	if err != nil {
		t.Fatalf("Tree() error = %v", err)
	}

	if tree.Revision != 1 {
		t.Errorf("HEAD = %d, want 1 (the layout commit)", tree.Revision)
	}

	names := map[string]bool{}
	for _, entry := range tree.Entries {
		names[entry.Name] = true
	}

	for _, want := range []string{"trunk", "branches", "tags"} {
		if !names[want] {
			t.Errorf("tree lacks %q: %+v", want, tree.Entries)
		}
	}

	t.Logf("tree @r%d entries=%d", tree.Revision, len(tree.Entries))

	// A nested path still resolves HEAD from the repository root; the empty
	// trunk must not look like revision 0.
	nested, err := store.Tree(ctx, "it-repo", "trunk", 0)
	if err != nil {
		t.Fatalf("Tree(trunk) error = %v", err)
	}

	if nested.Revision != 1 || len(nested.Entries) != 0 {
		t.Errorf("nested tree = %+v", nested)
	}

	info, err := store.Info(ctx, "it-repo")
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.HeadRevision != 1 || info.LastChangedRev != 1 {
		t.Errorf("info = %+v", info)
	}

	t.Logf("info head=%d lastChanged=%d author=%q", info.HeadRevision, info.LastChangedRev, info.LastChangedAuthor)

	if _, err := store.DeleteRepository(ctx, "it-repo"); err != nil {
		t.Fatalf("DeleteRepository() error = %v", err)
	}

	if _, err := os.Stat(repository.Path); !os.IsNotExist(err) {
		t.Errorf("repository still exists after delete: %v", err)
	}

	entries, err := os.ReadDir(trash)
	if err != nil || len(entries) != 1 {
		t.Fatalf("trash entries = %v (%v)", entries, err)
	}

	if !IsRepository(filepath.Join(trash, entries[0].Name())) {
		t.Errorf("the trashed directory is not a repository: %s", entries[0].Name())
	}

	t.Logf("deleted -> %s", filepath.Join(trash, entries[0].Name()))
}
