package svnrepos

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mmhk/svn-admin/pkg/model"
)

func makeRepository(t *testing.T, parent, name string) string {
	t.Helper()

	path := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(path, "db"), 0o755); err != nil {
		t.Fatalf("MkdirAll(db) error = %v", err)
	}

	if err := os.MkdirAll(filepath.Join(path, "conf"), 0o755); err != nil {
		t.Fatalf("MkdirAll(conf) error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(path, "format"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(format) error = %v", err)
	}

	return path
}

func TestListRepositoriesDetection(t *testing.T) {
	parent := t.TempDir()

	makeRepository(t, parent, "repo1")
	makeRepository(t, parent, "repo2")

	// A plain directory must not be reported as a repository (v2 bug F6).
	if err := os.Mkdir(filepath.Join(parent, "notarepo"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	// A directory with only `format` is also not a repository.
	partial := filepath.Join(parent, "partial")
	if err := os.Mkdir(partial, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(partial, "format"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := New(parent)

	page, err := store.ListRepositories(context.Background(), model.PageQuery{})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("page = %+v", page)
	}

	for _, repository := range page.Items {
		if !repository.IsRepository {
			t.Errorf("%s is not marked as a repository", repository.Name)
		}

		if repository.Parent != parent {
			t.Errorf("%s parent = %q", repository.Name, repository.Parent)
		}

		if repository.Path == "" {
			t.Errorf("%s has no path", repository.Name)
		}
	}

	if page.Items[0].Name != "repo1" || page.Items[1].Name != "repo2" {
		t.Errorf("order = %q, %q", page.Items[0].Name, page.Items[1].Name)
	}
}

func TestListRepositoriesFilterAndPaging(t *testing.T) {
	parent := t.TempDir()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		makeRepository(t, parent, name)
	}

	store := New(parent)

	filtered, err := store.ListRepositories(context.Background(), model.PageQuery{Q: "a"})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if filtered.Total != 3 {
		t.Errorf("filter q=a total = %d, want 3 (alpha, beta, gamma all contain a)", filtered.Total)
	}

	needle, err := store.ListRepositories(context.Background(), model.PageQuery{Q: "al"})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if needle.Total != 1 || needle.Items[0].Name != "alpha" {
		t.Errorf("filter q=al = %+v", needle)
	}

	paged, err := store.ListRepositories(context.Background(), model.PageQuery{Page: 2, Size: 2})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if paged.Total != 3 || len(paged.Items) != 1 || paged.Items[0].Name != "gamma" {
		t.Errorf("page 2 = %+v", paged)
	}
}

func TestGetRepository(t *testing.T) {
	parent := t.TempDir()
	makeRepository(t, parent, "repo1")

	store := New(parent)

	repository, err := store.GetRepository(context.Background(), "repo1")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}

	if repository.Name != "repo1" || repository.Parent != parent {
		t.Errorf("repository = %+v", repository)
	}

	if _, err := store.GetRepository(context.Background(), "nope"); !model.IsNotFound(err) {
		t.Errorf("GetRepository(nope) error = %v, want NotFoundError", err)
	}
}

func TestMultipleParents(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()

	makeRepository(t, first, "one")
	makeRepository(t, second, "two")

	store := New(first, second)

	page, err := store.ListRepositories(context.Background(), model.PageQuery{})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}

	if page.Items[0].Parent == page.Items[1].Parent {
		t.Errorf("parents = %q/%q, want different", page.Items[0].Parent, page.Items[1].Parent)
	}
}

func TestMissingParentIsTypedError(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "missing"))

	_, err := store.ListRepositories(context.Background(), model.PageQuery{})
	if err == nil {
		t.Fatal("ListRepositories() error = nil, want an error")
	}

	if _, ok := err.(*Error); !ok {
		t.Fatalf("error = %T, want *svnrepos.Error", err)
	}
}
