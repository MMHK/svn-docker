package svnrepos

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/svncli"
)

var (
	_ model.RepositoryWriter  = (*Store)(nil)
	_ model.RepositoryBrowser = (*Store)(nil)
)

// CreateRepository runs `svnadmin create --fs-type fsfs` in the first writable
// parent, then optionally commits the standard layout (plan T4.3).
func (s *Store) CreateRepository(ctx context.Context, spec model.RepositorySpec) (model.Repository, error) {
	if err := spec.Validate(); err != nil {
		return model.Repository{}, err
	}

	if s.cli == nil {
		return model.Repository{}, model.ErrReadOnly
	}

	// A name collision in *any* configured parent is a conflict: listing is
	// name-based, so two repositories with the same name would be ambiguous.
	for _, parent := range s.parents {
		if _, err := os.Lstat(filepath.Join(parent, spec.Name)); err == nil {
			return model.Repository{}, &model.ConflictError{
				Kind:   model.KindRepository,
				Name:   spec.Name,
				Reason: "already exists",
			}
		}
	}

	parent, err := s.creationParent(spec.Parent)
	if err != nil {
		return model.Repository{}, err
	}

	path, err := containedPath(parent, spec.Name)
	if err != nil {
		return model.Repository{}, err
	}

	if err := s.cli.Create(ctx, path); err != nil {
		return model.Repository{}, &model.OperationError{
			Operation: "repository.create",
			Step:      "svnadmin-create",
			Err:       err,
		}
	}

	completed := []string{"svnadmin-create"}

	if dirs := spec.Structure.LayoutDirs(spec.ProjectName); len(dirs) > 0 {
		if err := s.cli.MkDirs(ctx, svncli.FileURL(path), dirs); err != nil {
			return model.Repository{}, &model.OperationError{
				Operation: "repository.create",
				Step:      "layout",
				Completed: completed,
				Err:       err,
			}
		}
	}

	return model.Repository{
		Name:         spec.Name,
		Path:         path,
		Parent:       parent,
		URL:          svncli.FileURL(path),
		IsRepository: true,
	}, nil
}

// DeleteRepository renames the repository into the trash directory. It never
// removes data recursively: an operator can restore or delete it later.
func (s *Store) DeleteRepository(ctx context.Context, name string) (model.RepositoryDeletion, error) {
	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return model.RepositoryDeletion{}, err
	}

	repository, err := s.GetRepository(ctx, name)
	if err != nil {
		return model.RepositoryDeletion{}, err
	}

	if s.trash == "" {
		return model.RepositoryDeletion{}, errors.New("svnrepos: the trash directory is not configured")
	}

	if err := os.MkdirAll(s.trash, 0o750); err != nil {
		return model.RepositoryDeletion{}, &Error{Op: "delete", Path: s.trash, Err: err}
	}

	target, err := s.trashTarget(name)
	if err != nil {
		return model.RepositoryDeletion{}, err
	}

	if err := s.rename(repository.Path, target); err != nil {
		switch {
		case isCrossDevice(err):
			// No recursive fallback: a copy+delete across filesystems could
			// destroy a 77G repository halfway through.
			return model.RepositoryDeletion{}, &model.ConflictError{
				Kind:   model.KindRepository,
				Name:   name,
				Reason: "cannot be moved to the trash directory on another filesystem; move it manually",
			}
		case errors.Is(err, fs.ErrPermission):
			return model.RepositoryDeletion{}, &model.ForbiddenError{
				Reason: "the repository cannot be moved to the trash directory",
			}
		default:
			return model.RepositoryDeletion{}, &Error{Op: "delete", Path: name, Err: err}
		}
	}

	return model.RepositoryDeletion{Name: name, TrashPath: target}, nil
}

// Tree lists one repository directory through `svn list --xml`.
func (s *Store) Tree(ctx context.Context, name, path string, revision int64) (model.RepositoryTree, error) {
	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return model.RepositoryTree{}, err
	}

	if err := model.ValidateRepositoryPath(path); err != nil {
		return model.RepositoryTree{}, err
	}

	if s.cli == nil {
		return model.RepositoryTree{}, model.ErrReadOnly
	}

	repository, err := s.GetRepository(ctx, name)
	if err != nil {
		return model.RepositoryTree{}, err
	}

	target := svncli.FileURL(repository.Path)
	if path != "" {
		target = svncli.JoinURL(target, strings.Split(path, "/")...)
	}

	effective := revision
	if effective <= 0 {
		// HEAD is a repository-wide property; asking `svn log -r HEAD` about
		// a subdirectory returns nothing when that directory did not change
		// in the newest revision, so always resolve it at the root URL.
		head, err := s.cli.HeadRevision(ctx, svncli.FileURL(repository.Path))
		if err != nil {
			return model.RepositoryTree{}, mapSVNError(err, name, path)
		}

		effective = head
	}

	listing, err := s.cli.List(ctx, target, effective)
	if err != nil {
		return model.RepositoryTree{}, mapSVNError(err, name, path)
	}

	entries := make([]model.RepositoryEntry, 0, len(listing.Entries))
	for _, entry := range listing.Entries {
		entries = append(entries, model.RepositoryEntry{
			Name:     entry.Name,
			Kind:     entry.Kind,
			Size:     entry.Size,
			Revision: entry.Revision,
			Author:   entry.Author,
			Date:     entry.Date,
		})
	}

	return model.RepositoryTree{Path: path, Revision: effective, Entries: entries}, nil
}

// Info reads HEAD and the last-changed metadata through svn info/log.
func (s *Store) Info(ctx context.Context, name string) (model.Repository, error) {
	if err := (model.Repository{Name: name}).Validate(); err != nil {
		return model.Repository{}, err
	}

	if s.cli == nil {
		return model.Repository{}, model.ErrReadOnly
	}

	repository, err := s.GetRepository(ctx, name)
	if err != nil {
		return model.Repository{}, err
	}

	target := svncli.FileURL(repository.Path)

	info, err := s.cli.Info(ctx, target)
	if err != nil {
		return model.Repository{}, mapSVNError(err, name, "")
	}

	head, err := s.cli.HeadRevision(ctx, target)
	if err != nil {
		// The entry revision is the last change to the root, not HEAD, but it
		// is still a usable approximation when the log is unavailable.
		head = info.Revision
	}

	repository.URL = target
	repository.HeadRevision = head
	repository.LastChangedRev = info.LastChangedRev
	repository.LastChangedAuthor = info.LastChangedAuthor
	repository.LastChangedAt = info.LastChangedAt

	return repository, nil
}

// creationParent picks the parent directory for a new repository: the
// explicitly requested one when given, otherwise the first configured parent
// that exists and is writable.
func (s *Store) creationParent(requested string) (string, error) {
	candidates := s.parents
	if requested != "" {
		candidates = []string{requested}
	}

	seenDirectory := false

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}

		seenDirectory = true

		if !writableDir(candidate) {
			continue
		}

		return candidate, nil
	}

	if seenDirectory {
		return "", &model.ForbiddenError{Reason: "the repository parent is not writable"}
	}

	return "", &model.ValidationError{Field: "parent", Reason: "the repository parent does not exist"}
}

// containedPath joins name onto parent and refuses anything that escapes it.
func containedPath(parent, name string) (string, error) {
	cleanParent := filepath.Clean(parent)
	path := filepath.Join(cleanParent, name)

	if filepath.Dir(path) != cleanParent || path == cleanParent {
		return "", &model.ValidationError{Field: "name", Reason: "must not escape the repository parent"}
	}

	return path, nil
}

// trashTarget builds `<repo>-<UTC20060102T150405Z>` and appends -2, -3, ...
// when a same-second deletion already used the name.
func (s *Store) trashTarget(name string) (string, error) {
	base := name + "-" + s.now().UTC().Format("20060102T150405Z")
	candidate := filepath.Join(s.trash, base)

	for suffix := 2; suffix < 1000; suffix++ {
		_, err := os.Lstat(candidate)
		if errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		}

		if err != nil {
			return "", &Error{Op: "delete", Path: s.trash, Err: err}
		}

		candidate = filepath.Join(s.trash, fmt.Sprintf("%s-%d", base, suffix))
	}

	return "", &model.ConflictError{
		Kind:   model.KindRepository,
		Name:   name,
		Reason: "the trash directory already holds too many copies",
	}
}

// writableDir probes a directory by creating and removing a temporary file.
func writableDir(path string) bool {
	probe, err := os.CreateTemp(path, ".svnadmin-write-probe-*")
	if err != nil {
		return false
	}

	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)

	return true
}

// isCrossDevice reports EXDEV on Unix and ERROR_NOT_SAME_DEVICE (17) on
// Windows, where syscall.EXDEV is not the same constant.
func isCrossDevice(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}

	if errno == syscall.EXDEV {
		return true
	}

	// Windows reports ERROR_NOT_SAME_DEVICE (17), which is a different
	// constant from the Unix EXDEV value.
	return runtime.GOOS == "windows" && errno == syscall.Errno(17)
}

// mapSVNError translates the one CLI failure the API can describe precisely:
// a missing path inside the repository becomes a 404 instead of a 500. Every
// other failure stays an internal error and its stderr never leaves the log.
func mapSVNError(err error, name, path string) error {
	if svncli.IsNotFound(err) {
		target := name
		if path != "" {
			target = name + "/" + path
		}

		return &model.NotFoundError{Kind: model.KindRepository, Name: target}
	}

	return err
}
