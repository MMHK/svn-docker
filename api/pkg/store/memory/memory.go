// Package memory is an in-memory implementation of the model store ports.
// It exists so P0 can exercise the presenter layer end to end; the real
// htpasswd/authz stores arrive in P1-P3.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
)

// Store is a concurrency-safe fake store.
type Store struct {
	mu           sync.RWMutex
	users        map[string]model.User
	passwords    map[string]string
	groups       map[string]model.Group
	repositories map[string]model.Repository
	accessPaths  map[string]model.AccessPath
}

var (
	_ model.UserStore        = (*Store)(nil)
	_ model.CredentialStore  = (*Store)(nil)
	_ model.AccountWriter    = (*Store)(nil)
	_ model.GroupStore       = (*Store)(nil)
	_ model.GroupWriter      = (*Store)(nil)
	_ model.RepositoryStore  = (*Store)(nil)
	_ model.AccessPathStore  = (*Store)(nil)
	_ model.AccessPathWriter = (*Store)(nil)
	_ model.AuthzWriter      = (*Store)(nil)
)

// New returns a store seeded with a small, deterministic data set.
func New() *Store {
	now := time.Now().UTC()

	store := &Store{
		users:        make(map[string]model.User),
		passwords:    make(map[string]string),
		groups:       make(map[string]model.Group),
		repositories: make(map[string]model.Repository),
		accessPaths:  make(map[string]model.AccessPath),
	}

	store.users["admin"] = model.User{
		Name:      "admin",
		Comment:   "Bootstrap administrator (no password in P0)",
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.users["alice"] = model.User{
		Name:      "alice",
		Comment:   "Example developer",
		Groups:    []string{"devs"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	store.groups["devs"] = model.Group{
		Name:        "devs",
		Description: "Development team",
		Members:     []string{"alice"},
	}

	store.repositories["project"] = model.Repository{
		Name:           "project",
		HeadRevision:   42,
		LastChangedRev: 42,
		LastChangedAt:  now,
	}

	store.accessPaths["project:/"] = model.AccessPath{
		Path: "project:/",
		ACL: []model.ACLEntry{
			{Principal: "@devs", Permission: model.PermissionReadWrite},
		},
	}
	store.accessPaths["project:/trunk"] = model.AccessPath{
		Path: "project:/trunk",
		ACL: []model.ACLEntry{
			{Principal: "@devs", Permission: model.PermissionReadWrite},
		},
	}

	return store
}

// ListUsers implements model.UserStore.
func (s *Store) ListUsers(_ context.Context, query model.PageQuery) (model.Page[model.User], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]model.User, 0, len(s.users))
	for _, user := range s.users {
		all = append(all, user)
	}

	items, total := filterAndPage(all, query, func(user model.User) string { return user.Name })

	return model.Page[model.User]{
		Items: items,
		Total: total,
		Page:  query.Normalize().Page,
		Size:  query.Normalize().Size,
	}, nil
}

// GetUser implements model.UserStore.
func (s *Store) GetUser(_ context.Context, name string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[name]
	if !ok {
		return model.User{}, &model.NotFoundError{Kind: model.KindUser, Name: name}
	}

	return user, nil
}

// CreateUser implements the in-memory fake's convenience path used by tests.
func (s *Store) CreateUser(_ context.Context, user model.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Name]; exists {
		return &model.ConflictError{Kind: model.KindUser, Name: user.Name, Reason: "already exists"}
	}

	s.users[user.Name] = user

	return nil
}

// DeleteUser implements model.UserStore.
func (s *Store) DeleteUser(_ context.Context, name string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; !exists {
		return &model.NotFoundError{Kind: model.KindUser, Name: name}
	}

	delete(s.users, name)
	delete(s.passwords, name)

	return nil
}

// CreateAccount implements model.AccountWriter.
func (s *Store) CreateAccount(_ context.Context, name, password string, _ model.WriteOptions) error {
	if err := model.ValidateUserName(name); err != nil {
		return err
	}

	if err := model.ValidatePassword(password); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; exists {
		return &model.ConflictError{Kind: model.KindUser, Name: name, Reason: "already exists"}
	}

	s.users[name] = model.User{Name: name}
	s.passwords[name] = password

	return nil
}

// SetPassword implements model.AccountWriter.
func (s *Store) SetPassword(_ context.Context, name, password string, _ model.WriteOptions) error {
	if err := model.ValidatePassword(password); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[name]; !exists {
		return &model.NotFoundError{Kind: model.KindUser, Name: name}
	}

	s.passwords[name] = password

	return nil
}

// ListGroups implements model.GroupStore.
func (s *Store) ListGroups(_ context.Context, query model.PageQuery) (model.Page[model.Group], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]model.Group, 0, len(s.groups))
	for _, group := range s.groups {
		all = append(all, group)
	}

	items, total := filterAndPage(all, query, func(group model.Group) string { return group.Name })
	normalized := query.Normalize()

	return model.Page[model.Group]{
		Items: items,
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetGroup implements model.GroupStore.
func (s *Store) GetGroup(_ context.Context, name string) (model.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	group, ok := s.groups[name]
	if !ok {
		return model.Group{}, &model.NotFoundError{Kind: model.KindGroup, Name: name}
	}

	return group, nil
}

// GroupsOfUser implements model.GroupStore.
func (s *Store) GroupsOfUser(_ context.Context, username string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []string

	for name, group := range s.groups {
		for _, member := range group.Members {
			if member == username {
				out = append(out, name)

				break
			}
		}
	}

	sort.Strings(out)

	return out, nil
}

// CreateGroup implements model.GroupWriter.
func (s *Store) CreateGroup(_ context.Context, group model.Group, _ model.WriteOptions) error {
	if err := group.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[group.Name]; exists {
		return &model.ConflictError{Kind: model.KindGroup, Name: group.Name, Reason: "already exists"}
	}

	s.groups[group.Name] = group

	return nil
}

// DeleteGroup implements model.GroupWriter. It also drops every `@name` ACL
// entry, matching the real authz store.
func (s *Store) DeleteGroup(_ context.Context, name string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[name]; !exists {
		return &model.NotFoundError{Kind: model.KindGroup, Name: name}
	}

	delete(s.groups, name)

	principal := "@" + name

	for path, accessPath := range s.accessPaths {
		kept := make([]model.ACLEntry, 0, len(accessPath.ACL))

		for _, entry := range accessPath.ACL {
			if entry.Principal != principal {
				kept = append(kept, entry)
			}
		}

		accessPath.ACL = kept
		s.accessPaths[path] = accessPath
	}

	return nil
}

// SetGroupMembers implements model.GroupWriter.
func (s *Store) SetGroupMembers(_ context.Context, name string, members []string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	group, exists := s.groups[name]
	if !exists {
		return &model.NotFoundError{Kind: model.KindGroup, Name: name}
	}

	sanitised := make([]string, 0, len(members))

	for _, member := range members {
		if err := model.ValidateUserName(member); err != nil {
			return err
		}

		if !contains(sanitised, member) {
			sanitised = append(sanitised, member)
		}
	}

	group.Members = sanitised
	s.groups[name] = group

	return nil
}

// ListRepositories implements model.RepositoryStore.
func (s *Store) ListRepositories(_ context.Context, query model.PageQuery) (model.Page[model.Repository], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]model.Repository, 0, len(s.repositories))
	for _, repository := range s.repositories {
		all = append(all, repository)
	}

	items, total := filterAndPage(all, query, func(repository model.Repository) string { return repository.Name })
	normalized := query.Normalize()

	return model.Page[model.Repository]{
		Items: items,
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetRepository implements model.RepositoryStore.
func (s *Store) GetRepository(_ context.Context, name string) (model.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	repository, ok := s.repositories[name]
	if !ok {
		return model.Repository{}, &model.NotFoundError{Kind: model.KindRepository, Name: name}
	}

	return repository, nil
}

// ListAccessPaths implements model.AccessPathStore.
func (s *Store) ListAccessPaths(_ context.Context, query model.PageQuery) (model.Page[model.AccessPath], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]model.AccessPath, 0, len(s.accessPaths))
	for _, accessPath := range s.accessPaths {
		all = append(all, accessPath)
	}

	items, total := filterAndPage(all, query, func(accessPath model.AccessPath) string { return accessPath.Path })
	normalized := query.Normalize()

	return model.Page[model.AccessPath]{
		Items: items,
		Total: total,
		Page:  normalized.Page,
		Size:  normalized.Size,
	}, nil
}

// GetAccessPath implements model.AccessPathStore.
func (s *Store) GetAccessPath(_ context.Context, path string) (model.AccessPath, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accessPath, ok := s.accessPaths[path]
	if !ok {
		return model.AccessPath{}, &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
	}

	return accessPath, nil
}

// AllAccessPaths implements model.AccessPathStore.
func (s *Store) AllAccessPaths(_ context.Context) ([]model.AccessPath, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.AccessPath, 0, len(s.accessPaths))
	for _, accessPath := range s.accessPaths {
		out = append(out, accessPath)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path)
	})

	return out, nil
}

// CreateAccessPath implements model.AccessPathWriter.
func (s *Store) CreateAccessPath(_ context.Context, path string, _ model.WriteOptions) error {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accessPaths[path]; exists {
		return &model.ConflictError{Kind: model.KindAccessPath, Name: path, Reason: "already exists"}
	}

	s.accessPaths[path] = model.AccessPath{Path: path}

	return nil
}

// DeleteAccessPath implements model.AccessPathWriter.
func (s *Store) DeleteAccessPath(_ context.Context, path string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accessPaths[path]; !exists {
		return &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
	}

	delete(s.accessPaths, path)

	return nil
}

// SetACL implements model.AccessPathWriter.
func (s *Store) SetACL(_ context.Context, path, principal string, permission model.Permission, _ model.WriteOptions) error {
	if !model.ValidPrincipal(principal) {
		return &model.ValidationError{Field: "principal", Reason: "invalid principal " + principal}
	}

	if !permission.Valid() {
		return &model.ValidationError{Field: "permission", Reason: "must be one of none, r, rw"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accessPath, exists := s.accessPaths[path]
	if !exists {
		return &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
	}

	if permission == model.PermissionNone {
		accessPath.ACL = removePrincipal(accessPath.ACL, principal)
		s.accessPaths[path] = accessPath

		return nil
	}

	for index, entry := range accessPath.ACL {
		if entry.Principal == principal {
			accessPath.ACL[index].Permission = permission
			s.accessPaths[path] = accessPath

			return nil
		}
	}

	accessPath.ACL = append(accessPath.ACL, model.ACLEntry{Principal: principal, Permission: permission})
	s.accessPaths[path] = accessPath

	return nil
}

// DeleteACL implements model.AccessPathWriter.
func (s *Store) DeleteACL(_ context.Context, path, principal string, _ model.WriteOptions) error {
	if !model.ValidPrincipal(principal) {
		return &model.ValidationError{Field: "principal", Reason: "invalid principal " + principal}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	accessPath, exists := s.accessPaths[path]
	if !exists {
		return &model.NotFoundError{Kind: model.KindAccessPath, Name: path}
	}

	accessPath.ACL = removePrincipal(accessPath.ACL, principal)
	s.accessPaths[path] = accessPath

	return nil
}

func removePrincipal(entries []model.ACLEntry, principal string) []model.ACLEntry {
	kept := make([]model.ACLEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.Principal != principal {
			kept = append(kept, entry)
		}
	}

	return kept
}

// AccessPathsOfGroup implements model.AccessPathStore.
func (s *Store) AccessPathsOfGroup(_ context.Context, group string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	principal := "@" + group
	var out []string

	for path, accessPath := range s.accessPaths {
		for _, entry := range accessPath.ACL {
			if entry.Principal == principal {
				out = append(out, path)

				break
			}
		}
	}

	sort.Strings(out)

	return out, nil
}

// VerifyPassword implements model.CredentialStore. The fake accepts a single
// fixed password so auth flows can be tested without a hash file.
func (s *Store) VerifyPassword(_ context.Context, username, password string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.users[username]; !ok {
		return model.ErrInvalidCredentials
	}

	// Accounts created through the write port carry their own password; the
	// seeded fixtures keep accepting the historical test password.
	if stored, ok := s.passwords[username]; ok {
		if stored != password {
			return model.ErrInvalidCredentials
		}

		return nil
	}

	if password != "testpass" {
		return model.ErrInvalidCredentials
	}

	return nil
}

// RemoveUser implements model.AuthzWriter for the fake: it drops the group
// memberships.
func (s *Store) RemoveUser(_ context.Context, username string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, group := range s.groups {
		group.Members = removeValue(group.Members, username)
		s.groups[name] = group
	}

	return nil
}

// SetGroupsOfUser implements model.AuthzWriter.
func (s *Store) SetGroupsOfUser(_ context.Context, username string, groups []string, _ model.WriteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[username]; !ok {
		return &model.NotFoundError{Kind: model.KindUser, Name: username}
	}

	wanted := make(map[string]bool, len(groups))
	for _, name := range groups {
		if err := model.ValidateGroupName(name); err != nil {
			return err
		}

		group, ok := s.groups[name]
		if !ok {
			return &model.ValidationError{Field: "groups", Reason: "unknown group " + name}
		}

		wanted[name] = true

		if !contains(group.Members, username) {
			group.Members = append(group.Members, username)
			s.groups[name] = group
		}
	}

	for name, group := range s.groups {
		if wanted[name] {
			continue
		}

		group.Members = removeValue(group.Members, username)
		s.groups[name] = group
	}

	return nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}

	return false
}

func removeValue(values []string, target string) []string {
	kept := make([]string, 0, len(values))

	for _, value := range values {
		if value != target {
			kept = append(kept, value)
		}
	}

	return kept
}

func appendUnique(values []string, candidate string) []string {
	if contains(values, candidate) {
		return values
	}

	return append(values, candidate)
}

// filterAndPage applies the q filter, sorts by key and slices one page.
func filterAndPage[T any](items []T, query model.PageQuery, key func(T) string) ([]T, int) {
	normalized := query.Normalize()
	needle := strings.ToLower(normalized.Q)

	filtered := make([]T, 0, len(items))

	for _, item := range items {
		if needle == "" || strings.Contains(strings.ToLower(key(item)), needle) {
			filtered = append(filtered, item)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(key(filtered[i])) < strings.ToLower(key(filtered[j]))
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

	return filtered[start:end], total
}
