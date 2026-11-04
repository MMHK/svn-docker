package model

import (
	"context"
	"strings"
)

// Paging defaults from the rewrite plan (section 4.1).
const (
	DefaultPageSize = 50
	MaxPageSize     = 500
)

// PageQuery is the shared list filter. Handlers bind it; stores honour it.
type PageQuery struct {
	Q    string
	Page int
	Size int
}

// Normalize applies the default page size and the hard upper bound.
func (q PageQuery) Normalize() PageQuery {
	out := q
	if out.Page < 1 {
		out.Page = 1
	}

	if out.Size <= 0 {
		out.Size = DefaultPageSize
	}

	if out.Size > MaxPageSize {
		out.Size = MaxPageSize
	}

	out.Q = strings.TrimSpace(out.Q)

	return out
}

// Page is a paged result set.
type Page[T any] struct {
	Items []T
	Total int
	Page  int
	Size  int
}

// UserStore is the persistence port for SVN accounts.
type UserStore interface {
	ListUsers(ctx context.Context, q PageQuery) (Page[User], error)
	GetUser(ctx context.Context, name string) (User, error)
	DeleteUser(ctx context.Context, name string, opts WriteOptions) error
}

// CredentialStore verifies a password against the account database. It is
// deliberately separate from UserStore so the auth flow can be tested with a
// fake and so no hash ever leaves the store.
type CredentialStore interface {
	VerifyPassword(ctx context.Context, username, password string) error
}

// WriteOptions carries the per-write preconditions of plan sections 4.1 and
// 5.1. ExpectedHash is the sha256 the caller based its edit on; when it is set
// and the file changed, the store returns a ConflictError (HTTP 409). An empty
// hash means "whatever is on disk now", and the store still re-checks the
// hash between its read and its rename.
type WriteOptions struct {
	ExpectedHash string
}

// AccountWriter is the write port for htpasswd-backed accounts. A password is
// passed in exactly once, hashed inside the store and never returned.
type AccountWriter interface {
	CreateAccount(ctx context.Context, name, password string, opts WriteOptions) error
	SetPassword(ctx context.Context, name, password string, opts WriteOptions) error
}

// RoleWriter is the lossless writer for data/userroleassignments.ini.
type RoleWriter interface {
	SetRoles(ctx context.Context, username string, roles []Role, opts WriteOptions) error
	AddAccessPaths(ctx context.Context, username string, paths []string, opts WriteOptions) error
	RemoveAccessPaths(ctx context.Context, username string, paths []string, opts WriteOptions) error
	RemoveUser(ctx context.Context, username string, opts WriteOptions) error
}

// AuthzWriter is the lossless writer for the authz file: group membership and
// per-user ACL removal (plan T2.3, T2.4; the full editor is P3).
type AuthzWriter interface {
	SetGroupsOfUser(ctx context.Context, username string, groups []string, opts WriteOptions) error
	RemoveUser(ctx context.Context, username string, opts WriteOptions) error
}

// GroupStore is the read persistence port for authz groups.
type GroupStore interface {
	ListGroups(ctx context.Context, q PageQuery) (Page[Group], error)
	GetGroup(ctx context.Context, name string) (Group, error)
	GroupsOfUser(ctx context.Context, username string) ([]string, error)
}

// GroupWriter is the lossless writer for `[groups]` and the access-path
// entries that reference a group (plan T3.1, T3.2).
type GroupWriter interface {
	CreateGroup(ctx context.Context, group Group, opts WriteOptions) error
	DeleteGroup(ctx context.Context, name string, opts WriteOptions) error
	SetGroupMembers(ctx context.Context, name string, members []string, opts WriteOptions) error
}

// RepositoryStore is the persistence port for repositories.
type RepositoryStore interface {
	ListRepositories(ctx context.Context, q PageQuery) (Page[Repository], error)
	GetRepository(ctx context.Context, name string) (Repository, error)
}

// RepositoryWriter creates and retires repositories (plan T4.3, T4.5).
type RepositoryWriter interface {
	// CreateRepository validates nothing; the presenter does. The store
	// resolves the parent, runs `svnadmin create`, writes the optional
	// layout and returns the new repository.
	CreateRepository(ctx context.Context, spec RepositorySpec) (Repository, error)
	// DeleteRepository moves the repository into the trash directory. It
	// never removes data recursively (plan D7).
	DeleteRepository(ctx context.Context, name string) (RepositoryDeletion, error)
}

// RepositoryBrowser is the read side that needs the svn CLI (plan T4.2,
// T4.6). It is separate from RepositoryStore so P1's directory scan stays
// usable without a Subversion installation.
type RepositoryBrowser interface {
	Tree(ctx context.Context, name, path string, revision int64) (RepositoryTree, error)
	Info(ctx context.Context, name string) (Repository, error)
}

// AccessPathStore is the read persistence port for authz sections.
type AccessPathStore interface {
	ListAccessPaths(ctx context.Context, q PageQuery) (Page[AccessPath], error)
	GetAccessPath(ctx context.Context, path string) (AccessPath, error)
	AccessPathsOfGroup(ctx context.Context, group string) ([]string, error)
	// AllAccessPaths returns every section without paging so derived views
	// (for example the three access sources of a user) never truncate.
	AllAccessPaths(ctx context.Context) ([]AccessPath, error)
}

// AccessPathWriter is the lossless writer for authz access-path sections and
// their ACL entries (plan T3.1, T3.3, T3.4).
type AccessPathWriter interface {
	CreateAccessPath(ctx context.Context, path string, opts WriteOptions) error
	DeleteAccessPath(ctx context.Context, path string, opts WriteOptions) error
	SetACL(ctx context.Context, path, principal string, permission Permission, opts WriteOptions) error
	DeleteACL(ctx context.Context, path, principal string, opts WriteOptions) error
}

// RoleAssignments is the parsed content of data/userroleassignments.ini.
//
// Roles maps a user name to the roles assigned in the [user] section;
// AccessPaths maps a user name to the project-manager scope recorded under
// [user:AccessPaths]. Both maps preserve file order.
type RoleAssignments struct {
	Roles       map[string][]Role
	AccessPaths map[string][]string
}

// RoleStore is the persistence port for role assignments.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]Role, error)
	Assignments(ctx context.Context) (RoleAssignments, error)
	AssignRole(ctx context.Context, username string, role Role) error
}
