// Package svnrepos discovers Subversion repositories under one or more
// SVNParentPath directories. A directory only counts as a repository when it
// contains the `format` file plus the `db/` and `conf/` directories, which
// fixes the v2 habit of treating every directory as a repository (F6).
//
// From P4 the same store also creates and retires repositories (T4.3, T4.5)
// and browses them through pkg/svncli (T4.2, T4.6). The CLI is optional so the
// read-only directory scan keeps working without a Subversion installation.
package svnrepos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/svncli"
)

// Error is the typed failure returned by the store.
type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("svnrepos: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Options configures a Store.
type Options struct {
	// Parents are the configured SVNParentPath directories, in priority
	// order. Creation uses the first one that exists and is writable.
	Parents []string
	// CLI runs svn/svnadmin. Nil keeps the store read-only (P1 behaviour).
	CLI *svncli.Client
	// TrashDir receives deleted repositories. Required for deletion.
	TrashDir string
	// Now is injectable so the trash suffix is deterministic in tests.
	Now func() time.Time
	// Rename is injectable so the cross-filesystem branch is testable.
	Rename func(oldPath, newPath string) error
}

// Store scans the configured parent directories on demand.
type Store struct {
	parents []string
	cli     *svncli.Client
	trash   string
	now     func() time.Time
	rename  func(oldPath, newPath string) error
}

// New builds a read-only store for the given parent directories.
func New(parents ...string) *Store {
	return NewStore(Options{Parents: parents})
}

// NewStore builds a store with the given options.
func NewStore(opts Options) *Store {
	filtered := make([]string, 0, len(opts.Parents))

	for _, parent := range opts.Parents {
		if trimmed := strings.TrimSpace(parent); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	store := &Store{
		parents: filtered,
		cli:     opts.CLI,
		trash:   strings.TrimSpace(opts.TrashDir),
		now:     opts.Now,
		rename:  opts.Rename,
	}

	if store.now == nil {
		store.now = time.Now
	}

	if store.rename == nil {
		store.rename = os.Rename
	}

	return store
}

// Parents returns the configured parent directories.
func (s *Store) Parents() []string {
	out := make([]string, len(s.parents))
	copy(out, s.parents)

	return out
}

var _ model.RepositoryStore = (*Store)(nil)

// ListRepositories implements model.RepositoryStore.
func (s *Store) ListRepositories(_ context.Context, query model.PageQuery) (model.Page[model.Repository], error) {
	all, scanned := s.scan()
	if !scanned {
		return model.Page[model.Repository]{}, &Error{
			Op:   "list",
			Path: strings.Join(s.parents, ","),
			Err:  os.ErrNotExist,
		}
	}

	normalized := query.Normalize()
	needle := strings.ToLower(normalized.Q)

	filtered := make([]model.Repository, 0, len(all))

	for _, repository := range all {
		if needle == "" || strings.Contains(strings.ToLower(repository.Name), needle) {
			filtered = append(filtered, repository)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Name == filtered[j].Name {
			return filtered[i].Parent < filtered[j].Parent
		}

		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})

	total := len(filtered)
	start := (normalized.Page - 1) * normalized.Size
	if start > total {
		start = total
	}

	end := start + normalized.Size
	if end > total {
		end = total
	}

	return model.Page[model.Repository]{
		Items: filtered[start:end],
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetRepository implements model.RepositoryStore.
func (s *Store) GetRepository(_ context.Context, name string) (model.Repository, error) {
	all, _ := s.scan()

	for _, repository := range all {
		if repository.Name == name {
			return repository, nil
		}
	}

	return model.Repository{}, &model.NotFoundError{Kind: model.KindRepository, Name: name}
}

// scan walks every parent. scanned reports whether at least one parent was
// readable; an unreadable parent is skipped so one bad mount does not hide the
// repositories on the others.
func (s *Store) scan() (repositories []model.Repository, scanned bool) {
	seen := make(map[string]bool)

	for _, parent := range s.parents {
		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}

		scanned = true

		for _, entry := range entries {
			name := entry.Name()
			if seen[name] || !entry.IsDir() {
				continue
			}

			path := filepath.Join(parent, name)
			if !IsRepository(path) {
				continue
			}

			seen[name] = true
			repositories = append(repositories, model.Repository{
				Name:         name,
				Path:         path,
				Parent:       parent,
				IsRepository: true,
			})
		}
	}

	return repositories, scanned
}

// IsRepository applies the plan section 5.4 detection rule.
func IsRepository(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}

	format, err := os.Lstat(filepath.Join(path, "format"))
	if err != nil || !format.Mode().IsRegular() {
		return false
	}

	for _, directory := range []string{"db", "conf"} {
		info, err := os.Lstat(filepath.Join(path, directory))
		if err != nil || !info.IsDir() {
			return false
		}
	}

	return true
}
