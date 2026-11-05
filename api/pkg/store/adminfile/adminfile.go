// Package adminfile stores the single svnadmin administrator account name.
//
// The file contains one account name per line; only the first non-empty line is
// significant (plan P9, D28). A missing or empty file means "no administrator
// yet". Reads are stateless so an operator can move the administrator role by
// editing the file without restarting the service (plan D31).
package adminfile

import (
	"context"
	"strings"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// Store is the filesystem-backed model.AdminStore.
type Store struct {
	path string
}

// New builds a store for path. The file is read on demand.
func New(path string) *Store {
	return &Store{path: path}
}

// Path returns the configured file path.
func (s *Store) Path() string {
	return s.path
}

// Admin implements model.AdminStore. A missing or empty file yields an empty
// name and no error so callers can treat "not configured" as setup pending.
func (s *Store) Admin(_ context.Context) (string, error) {
	snapshot, err := fsx.ReadFile(s.path)
	if err != nil {
		return "", err
	}

	if !snapshot.Exists {
		return "", nil
	}

	for _, line := range fsx.SplitLines(snapshot.Data) {
		if name := line.Trimmed(); name != "" {
			return name, nil
		}
	}

	return "", nil
}

// SetAdmin implements model.AdminStore. It serialises writers with the shared
// fsx lock and replaces the file atomically.
func (s *Store) SetAdmin(_ context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &model.ValidationError{
			Field:  "username",
			Reason: "administrator account must not be empty",
		}
	}

	if strings.TrimSpace(s.path) == "" {
		return &model.ValidationError{
			Field:  "adminFile",
			Reason: "path is not configured",
		}
	}

	lock, err := fsx.Lock(s.path)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Unlock() }()

	return fsx.WriteAtomic(s.path, []byte(name+"\n"), "")
}
