package presenter

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// UserDeps are the ports the user presenter needs. Writes are split into
// separate interfaces (accounts, authz file) because the delete cascade in
// plan T2.3 touches both and must report progress per step.
type UserDeps struct {
	Users       model.UserStore
	Accounts    model.AccountWriter
	Groups      model.GroupStore
	Authz       model.AuthzWriter
	AccessPaths model.AccessPathStore
	Tokens      *auth.Manager
}

// User serves the user read and write use cases.
type User struct {
	users       model.UserStore
	accounts    model.AccountWriter
	groups      model.GroupStore
	authz       model.AuthzWriter
	accessPaths model.AccessPathStore
	tokens      *auth.Manager
}

// NewUser builds the user presenter.
func NewUser(deps UserDeps) *User {
	return &User{
		users:       deps.Users,
		accounts:    deps.Accounts,
		groups:      deps.Groups,
		authz:       deps.Authz,
		accessPaths: deps.AccessPaths,
		tokens:      deps.Tokens,
	}
}

// List returns a page of users enriched with groups. Hashes never leave the
// store.
func (p *User) List(ctx context.Context, query model.PageQuery) (view.UserList, error) {
	query = query.Normalize()

	page, err := p.users.ListUsers(ctx, query)
	if err != nil {
		return view.UserList{}, err
	}

	groupsByUser, err := p.groupsByUser(ctx)
	if err != nil {
		return view.UserList{}, err
	}

	items := make([]view.User, 0, len(page.Items))
	for _, user := range page.Items {
		items = append(items, userView(user, groupsByUser[user.Name]))
	}

	return view.UserList{
		Items: items,
		ListMeta: view.ListMeta{
			Page:  page.Page,
			Size:  page.Size,
			Total: page.Total,
		},
	}, nil
}

// Get returns a single user.
func (p *User) Get(ctx context.Context, name string) (view.User, error) {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return view.User{}, err
	}

	user, err := p.users.GetUser(ctx, name)
	if err != nil {
		return view.User{}, err
	}

	groups, err := p.groups.GroupsOfUser(ctx, name)
	if err != nil {
		return view.User{}, err
	}

	return userView(user, groups), nil
}

// Create adds an htpasswd account. The password is hashed inside the store and
// never stored in the domain model.
func (p *User) Create(ctx context.Context, name, password string) (view.User, error) {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return view.User{}, err
	}

	if err := model.ValidatePassword(password); err != nil {
		return view.User{}, err
	}

	if err := p.accounts.CreateAccount(ctx, name, password, model.WriteOptions{}); err != nil {
		return view.User{}, err
	}

	return view.User{Name: name, Groups: []string{}}, nil
}

// Delete removes the account and every trace of it. The steps run in the order
// the plan fixes (authz groups + ACLs, htpasswd) and the first failure is
// reported together with the steps that already succeeded. P2 is sequential
// per-file, not a cross-file two-phase commit.
//
// The If-Match precondition is intentionally not applied here: the delete spans
// three files, so a single expected hash would be ambiguous. Each store still
// re-checks the hash of the file it writes between its read and its rename.
func (p *User) Delete(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return err
	}

	if _, err := p.users.GetUser(ctx, name); err != nil {
		return err
	}

	steps := []struct {
		label string
		run   func() error
	}{
		{"authz", func() error { return p.authz.RemoveUser(ctx, name, model.WriteOptions{}) }},
		{"htpasswd", func() error { return p.users.DeleteUser(ctx, name, model.WriteOptions{}) }},
	}

	completed := make([]string, 0, len(steps))

	for _, step := range steps {
		if err := step.run(); err != nil {
			return &model.OperationError{
				Operation: "user.delete",
				Step:      step.label,
				Completed: completed,
				Err:       err,
			}
		}

		completed = append(completed, step.label)
	}

	p.revoke(name)

	return nil
}

// ChangePassword sets a new password and revokes every token of that account,
// including the caller's own token when an operator rotates their own
// password (plan section 5.7).
func (p *User) ChangePassword(ctx context.Context, name, password string, opts model.WriteOptions) error {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return err
	}

	if err := model.ValidatePassword(password); err != nil {
		return err
	}

	if _, err := p.users.GetUser(ctx, name); err != nil {
		return err
	}

	if err := p.accounts.SetPassword(ctx, name, password, opts); err != nil {
		return err
	}

	p.revoke(name)

	return nil
}

// Groups returns the authz group memberships of name.
func (p *User) Groups(ctx context.Context, name string) (view.UserGroups, error) {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return view.UserGroups{}, err
	}

	if _, err := p.users.GetUser(ctx, name); err != nil {
		return view.UserGroups{}, err
	}

	groups, err := p.groups.GroupsOfUser(ctx, name)
	if err != nil {
		return view.UserGroups{}, err
	}

	return userGroupsView(name, groups), nil
}

// SetGroups replaces the group membership of name. Every group must exist.
func (p *User) SetGroups(ctx context.Context, name string, groups []string, opts model.WriteOptions) (view.UserGroups, error) {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return view.UserGroups{}, err
	}

	if _, err := p.users.GetUser(ctx, name); err != nil {
		return view.UserGroups{}, err
	}

	for _, group := range groups {
		if err := model.ValidateGroupName(group); err != nil {
			return view.UserGroups{}, err
		}

		if _, err := p.groups.GetGroup(ctx, group); err != nil {
			if model.IsNotFound(err) {
				return view.UserGroups{}, &model.ValidationError{Field: "groups", Reason: "unknown group " + group}
			}

			return view.UserGroups{}, err
		}
	}

	if err := p.authz.SetGroupsOfUser(ctx, name, groups, opts); err != nil {
		return view.UserGroups{}, err
	}

	return p.Groups(ctx, name)
}

// AccessPaths returns every effective access source of the account: direct
// entries, entries inherited through its groups and the global `*` entry.
func (p *User) AccessPaths(ctx context.Context, name string) (view.UserAccessPaths, error) {
	name = strings.TrimSpace(name)

	if err := (model.User{Name: name}).Validate(); err != nil {
		return view.UserAccessPaths{}, err
	}

	if _, err := p.users.GetUser(ctx, name); err != nil {
		return view.UserAccessPaths{}, err
	}

	groups, err := p.groups.GroupsOfUser(ctx, name)
	if err != nil {
		return view.UserAccessPaths{}, err
	}

	inGroup := make(map[string]bool, len(groups))
	for _, group := range groups {
		inGroup[group] = true
	}

	paths, err := p.accessPaths.AllAccessPaths(ctx)
	if err != nil {
		return view.UserAccessPaths{}, err
	}

	items := make([]view.UserAccessPath, 0)

	for _, accessPath := range paths {
		for _, entry := range accessPath.ACL {
			switch {
			case entry.Principal == name:
				items = append(items, view.UserAccessPath{
					Path:       accessPath.Path,
					Permission: string(entry.Permission),
					Inherited:  false,
					Source:     "user",
				})
			case entry.Principal == "*":
				items = append(items, view.UserAccessPath{
					Path:       accessPath.Path,
					Permission: string(entry.Permission),
					Inherited:  true,
					Source:     "global",
				})
			case strings.HasPrefix(entry.Principal, "@"):
				group := strings.TrimPrefix(entry.Principal, "@")
				if !inGroup[group] {
					continue
				}

				items = append(items, view.UserAccessPath{
					Path:       accessPath.Path,
					Permission: string(entry.Permission),
					Inherited:  true,
					Source:     "group",
					Group:      group,
				})
			}
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}

		return sourceRank(items[i].Source) < sourceRank(items[j].Source)
	})

	return view.UserAccessPaths{Name: name, Items: items}, nil
}

func sourceRank(source string) int {
	switch source {
	case "user":
		return 0
	case "group":
		return 1
	case "global":
		return 2
	default:
		return 3
	}
}

func (p *User) revoke(name string) {
	if p.tokens != nil {
		p.tokens.RevokeUser(name)
	}
}

func (p *User) groupsByUser(ctx context.Context) (map[string][]string, error) {
	page, err := p.groups.ListGroups(ctx, model.PageQuery{Size: model.MaxPageSize})
	if err != nil {
		return nil, err
	}

	out := make(map[string][]string)

	for _, group := range page.Items {
		for _, member := range group.Members {
			out[member] = append(out[member], group.Name)
		}
	}

	return out, nil
}

func userView(user model.User, groups []string) view.User {
	groupNames := make([]string, 0, len(groups))
	groupNames = append(groupNames, groups...)

	return view.User{
		Name:      user.Name,
		Comment:   user.Comment,
		Groups:    groupNames,
		CreatedAt: timePtr(user.CreatedAt),
		UpdatedAt: timePtr(user.UpdatedAt),
	}
}

func userGroupsView(name string, groups []string) view.UserGroups {
	out := make([]string, 0, len(groups))
	out = append(out, groups...)

	return view.UserGroups{Name: name, Groups: out}
}

// timePtr renders a timestamp as null instead of the zero value.
func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	out := value

	return &out
}
