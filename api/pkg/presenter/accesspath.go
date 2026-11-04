package presenter

import (
	"context"
	"strings"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// AccessPath serves authz access-path read and write use cases.
type AccessPath struct {
	paths      model.AccessPathStore
	writer     model.AccessPathWriter
	roles      model.RoleStore
	roleWriter model.RoleWriter
}

// NewAccessPath builds the access-path presenter.
func NewAccessPath(
	paths model.AccessPathStore,
	writer model.AccessPathWriter,
	roles model.RoleStore,
	roleWriter model.RoleWriter,
) *AccessPath {
	return &AccessPath{paths: paths, writer: writer, roles: roles, roleWriter: roleWriter}
}

// List returns a page of access paths with ACLs and project managers.
func (p *AccessPath) List(ctx context.Context, query model.PageQuery) (view.AccessPathList, error) {
	page, err := p.paths.ListAccessPaths(ctx, query.Normalize())
	if err != nil {
		return view.AccessPathList{}, err
	}

	managers, err := p.managersByPath(ctx)
	if err != nil {
		return view.AccessPathList{}, err
	}

	items := make([]view.AccessPath, 0, len(page.Items))
	for _, path := range page.Items {
		items = append(items, accessPathView(path, managers[path.Path]))
	}

	return view.AccessPathList{
		Items: items,
		ListMeta: view.ListMeta{
			Page:  page.Page,
			Size:  page.Size,
			Total: page.Total,
		},
	}, nil
}

// Get returns one access path and its ACL.
func (p *AccessPath) Get(ctx context.Context, path string) (view.AccessPath, error) {
	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return view.AccessPath{}, err
	}

	accessPath, err := p.paths.GetAccessPath(ctx, path)
	if err != nil {
		return view.AccessPath{}, err
	}

	managers, err := p.managersByPath(ctx)
	if err != nil {
		return view.AccessPath{}, err
	}

	return accessPathView(accessPath, managers[accessPath.Path]), nil
}

// Create adds an empty access-path section.
func (p *AccessPath) Create(ctx context.Context, path string, opts model.WriteOptions) (view.AccessPath, error) {
	path = strings.TrimSpace(path)

	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return view.AccessPath{}, err
	}

	if err := p.writer.CreateAccessPath(ctx, path, opts); err != nil {
		return view.AccessPath{}, err
	}

	return view.AccessPath{Path: path, ACL: []view.ACLEntry{}, Managers: []string{}}, nil
}

// Delete removes the section and every project-manager assignment that points
// at it (plan T3.6).
func (p *AccessPath) Delete(ctx context.Context, path string, opts model.WriteOptions) error {
	path = strings.TrimSpace(path)

	if err := (model.AccessPath{Path: path}).Validate(); err != nil {
		return err
	}

	if err := p.writer.DeleteAccessPath(ctx, path, opts); err != nil {
		return err
	}

	if err := p.clearManagers(ctx, path); err != nil {
		return &model.OperationError{
			Operation: "access-path.delete",
			Step:      "project-managers",
			Completed: []string{"authz"},
			Err:       err,
		}
	}

	return nil
}

// SetACL creates or replaces one principal/permission pair.
func (p *AccessPath) SetACL(
	ctx context.Context,
	path, principal string,
	permission model.Permission,
	opts model.WriteOptions,
) (view.AccessPath, error) {
	path = strings.TrimSpace(path)
	principal = strings.TrimSpace(principal)

	if err := p.writer.SetACL(ctx, path, principal, permission, opts); err != nil {
		return view.AccessPath{}, err
	}

	return p.Get(ctx, path)
}

// DeleteACL removes one principal/permission pair.
func (p *AccessPath) DeleteACL(ctx context.Context, path, principal string, opts model.WriteOptions) error {
	path = strings.TrimSpace(path)
	principal = strings.TrimSpace(principal)

	return p.writer.DeleteACL(ctx, path, principal, opts)
}

// Managers returns the project managers of one path.
func (p *AccessPath) Managers(ctx context.Context, path string) (view.AccessPathManagers, error) {
	path = strings.TrimSpace(path)

	if err := p.validateManagerPath(path); err != nil {
		return view.AccessPathManagers{}, err
	}

	if _, err := p.paths.GetAccessPath(ctx, path); err != nil {
		return view.AccessPathManagers{}, err
	}

	managers, err := p.managersByPath(ctx)
	if err != nil {
		return view.AccessPathManagers{}, err
	}

	users := managers[path]
	if users == nil {
		users = []string{}
	}

	return view.AccessPathManagers{Path: path, Users: users}, nil
}

// SetManagers replaces the project-manager list of one path. Additions and
// removals are written to `[user:AccessPaths]` with the v2 index keys.
func (p *AccessPath) SetManagers(
	ctx context.Context,
	path string,
	users []string,
	opts model.WriteOptions,
) (view.AccessPathManagers, error) {
	path = strings.TrimSpace(path)

	if err := p.validateManagerPath(path); err != nil {
		return view.AccessPathManagers{}, err
	}

	if _, err := p.paths.GetAccessPath(ctx, path); err != nil {
		return view.AccessPathManagers{}, err
	}

	desired := make([]string, 0, len(users))
	desiredSet := make(map[string]bool, len(users))

	for _, user := range users {
		user = strings.TrimSpace(user)

		if err := model.ValidateUserName(user); err != nil {
			return view.AccessPathManagers{}, err
		}

		if desiredSet[user] {
			continue
		}

		desiredSet[user] = true
		desired = append(desired, user)
	}

	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return view.AccessPathManagers{}, err
	}

	current := make(map[string]bool)

	for user, paths := range assignments.AccessPaths {
		for _, managed := range paths {
			if managed == path {
				current[user] = true

				break
			}
		}
	}

	for user := range current {
		if desiredSet[user] {
			continue
		}

		if err := p.roleWriter.RemoveAccessPaths(ctx, user, []string{path}, opts); err != nil {
			return view.AccessPathManagers{}, err
		}
	}

	for _, user := range desired {
		if current[user] {
			continue
		}

		if err := p.roleWriter.AddAccessPaths(ctx, user, []string{path}, opts); err != nil {
			return view.AccessPathManagers{}, err
		}
	}

	return view.AccessPathManagers{Path: path, Users: desired}, nil
}

// AuthorizePath enforces the Project-Manager scope from plan section 4.3.
// Administrator and Access-Path-Manager bypass the check; everybody else must
// manage the exact path or one of its parents.
func (p *AccessPath) AuthorizePath(ctx context.Context, username string, roles []model.Role, path string) error {
	for _, role := range roles {
		if role == model.RoleAdministrator || role == model.RoleAccessPathManager {
			return nil
		}
	}

	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return err
	}

	for _, managed := range assignments.AccessPaths[username] {
		if managed == "" {
			continue
		}

		if path == managed || strings.HasPrefix(path, managed) {
			return nil
		}
	}

	return &model.ForbiddenError{Reason: "the project-manager scope does not cover " + path}
}

// managersByPath inverts the [user:AccessPaths] project-manager scope.
func (p *AccessPath) managersByPath(ctx context.Context) (map[string][]string, error) {
	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string][]string)

	for user, paths := range assignments.AccessPaths {
		for _, path := range paths {
			out[path] = append(out[path], user)
		}
	}

	return out, nil
}

// clearManagers drops the path from every `[user:AccessPaths]` block.
func (p *AccessPath) clearManagers(ctx context.Context, path string) error {
	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return err
	}

	for user, paths := range assignments.AccessPaths {
		for _, managed := range paths {
			if managed != path {
				continue
			}

			if err := p.roleWriter.RemoveAccessPaths(ctx, user, []string{path}, model.WriteOptions{}); err != nil {
				return err
			}

			break
		}
	}

	return nil
}

// validateManagerPath rejects the global root in addition to the normal
// access-path contract (plan T3.5: `/` cannot be delegated).
func (p *AccessPath) validateManagerPath(path string) error {
	if path == "/" {
		return &model.ValidationError{
			Field:  "path",
			Reason: "the global root cannot be assigned as a project-manager scope",
		}
	}

	return (model.AccessPath{Path: path}).Validate()
}

func accessPathView(path model.AccessPath, managers []string) view.AccessPath {
	acl := make([]view.ACLEntry, 0, len(path.ACL))
	for _, entry := range path.ACL {
		acl = append(acl, view.ACLEntry{Principal: entry.Principal, Permission: string(entry.Permission)})
	}

	if managers == nil {
		managers = []string{}
	}

	return view.AccessPath{Path: path.Path, ACL: acl, Managers: managers}
}
