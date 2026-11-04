// Package roles parses data/userroleassignments.ini. The format is the v2
// iF.SVNAdmin one: `[user]` with role keys and `[user:AccessPaths]` with
// index keys. P1 only reads, plus one append-only write used by the bootstrap
// administrator flow; the lossless editor arrives in P2.
package roles

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mmhk/svn-admin/pkg/fsx"
	"github.com/mmhk/svn-admin/pkg/model"
)

// AccessPathsSuffix marks the project-manager scope section of a user.
const AccessPathsSuffix = ":AccessPaths"

// Error is the typed failure returned by the store.
type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("roles: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

type snapshot struct {
	roles       map[string][]model.Role
	accessPaths map[string][]string
	order       []string
	raw         []byte
}

// Store is a cached, read-mostly role assignment store. Reads use an mtime
// cache; writes are locked, lossless and atomic.
type Store struct {
	path      string
	backupDir string
	keep      int

	mu      sync.Mutex
	cache   *snapshot
	modTime time.Time
	size    int64
	loaded  bool

	writeMu sync.Mutex
}

// New builds a store for path.
func New(path string, opts ...Option) *Store {
	store := &Store{path: path, keep: fsx.DefaultBackupKeep}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// Path returns the configured file path.
func (s *Store) Path() string {
	return s.path
}

// Parse interprets role assignment content. Unknown lines are ignored but the
// raw bytes are retained for the future lossless writer.
func Parse(data []byte) *snapshot {
	snap := &snapshot{
		roles:       make(map[string][]model.Role),
		accessPaths: make(map[string][]string),
		raw:         data,
	}

	user := ""
	inAccessPaths := false

	ensure := func(name string) {
		if _, ok := snap.roles[name]; !ok {
			snap.roles[name] = nil
		}

		if _, ok := snap.accessPaths[name]; !ok {
			snap.accessPaths[name] = nil
		}

		for _, known := range snap.order {
			if known == name {
				return
			}
		}

		snap.order = append(snap.order, name)
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			inAccessPaths = len(name) > len(AccessPathsSuffix) &&
				strings.EqualFold(name[len(name)-len(AccessPathsSuffix):], AccessPathsSuffix)

			if inAccessPaths {
				name = strings.TrimSpace(name[:len(name)-len(AccessPathsSuffix)])
			}

			user = name

			if user != "" {
				ensure(user)
			}

			continue
		}

		if user == "" {
			continue
		}

		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if inAccessPaths {
			if value != "" {
				snap.accessPaths[user] = append(snap.accessPaths[user], value)
			}

			continue
		}

		if strings.EqualFold(key, "Role") {
			if role, ok := parseRole(value); ok {
				snap.roles[user] = appendRole(snap.roles[user], role)
			}

			continue
		}

		if role, ok := parseRole(key); ok {
			// The v2 files use `Role=` with an empty value; accept a value too.
			if value != "" {
				if declared, valid := parseRole(value); valid && declared != role {
					continue
				}
			}

			snap.roles[user] = appendRole(snap.roles[user], role)
		}
	}

	return snap
}

func parseRole(value string) (model.Role, bool) {
	if value == "" {
		return "", false
	}

	for _, role := range model.AllRoles() {
		if strings.EqualFold(string(role), value) {
			return role, true
		}
	}

	return "", false
}

func appendRole(roles []model.Role, candidate model.Role) []model.Role {
	for _, existing := range roles {
		if existing == candidate {
			return roles
		}
	}

	return append(roles, candidate)
}

// load returns the current snapshot, re-reading the file when its mtime or
// size changed. A missing file is an empty, valid snapshot so the bootstrap
// flow can create it.
func (s *Store) load() (*snapshot, error) {
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &snapshot{
				roles:       make(map[string][]model.Role),
				accessPaths: make(map[string][]string),
			}, nil
		}

		return nil, &Error{Op: "stat", Path: s.path, Err: err}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.loaded && info.ModTime().Equal(s.modTime) && info.Size() == s.size {
		return s.cache, nil
	}

	content, err := os.ReadFile(s.path)
	if err != nil {
		return nil, &Error{Op: "read", Path: s.path, Err: err}
	}

	snap := Parse(content)
	s.cache = snap
	s.modTime = info.ModTime()
	s.size = info.Size()
	s.loaded = true

	return snap, nil
}

var _ model.RoleStore = (*Store)(nil)

// ListRoles implements model.RoleStore and returns the fixed catalogue.
func (s *Store) ListRoles(context.Context) ([]model.Role, error) {
	return model.AllRoles(), nil
}

// Assignments implements model.RoleStore. The returned maps are copies.
func (s *Store) Assignments(context.Context) (model.RoleAssignments, error) {
	snap, err := s.load()
	if err != nil {
		return model.RoleAssignments{}, err
	}

	out := model.RoleAssignments{
		Roles:       make(map[string][]model.Role, len(snap.roles)),
		AccessPaths: make(map[string][]string, len(snap.accessPaths)),
	}

	for user, roles := range snap.roles {
		out.Roles[user] = append([]model.Role(nil), roles...)
	}

	for user, paths := range snap.accessPaths {
		out.AccessPaths[user] = append([]string(nil), paths...)
	}

	return out, nil
}

// HasAdministrator reports whether any user holds the Administrator role.
func (s *Store) HasAdministrator(ctx context.Context) (bool, error) {
	assignments, err := s.Assignments(ctx)
	if err != nil {
		return false, err
	}

	for _, roles := range assignments.Roles {
		for _, role := range roles {
			if role == model.RoleAdministrator {
				return true, nil
			}
		}
	}

	return false, nil
}
