package presenter

import (
	"context"
	"strings"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Group serves authz group read and write use cases.
type Group struct {
	groups      model.GroupStore
	writer      model.GroupWriter
	accessPaths model.AccessPathStore
}

// NewGroup builds the group presenter.
func NewGroup(groups model.GroupStore, writer model.GroupWriter, accessPaths model.AccessPathStore) *Group {
	return &Group{groups: groups, writer: writer, accessPaths: accessPaths}
}

// List returns a page of groups.
func (p *Group) List(ctx context.Context, query model.PageQuery) (view.GroupList, error) {
	page, err := p.groups.ListGroups(ctx, query.Normalize())
	if err != nil {
		return view.GroupList{}, err
	}

	items := make([]view.Group, 0, len(page.Items))
	for _, group := range page.Items {
		items = append(items, groupView(group))
	}

	return view.GroupList{
		Items: items,
		ListMeta: view.ListMeta{
			Page:  page.Page,
			Size:  page.Size,
			Total: page.Total,
		},
	}, nil
}

// Get returns one group plus the access paths that reference it.
func (p *Group) Get(ctx context.Context, name string) (view.GroupDetail, error) {
	if err := (model.Group{Name: name}).Validate(); err != nil {
		return view.GroupDetail{}, err
	}

	group, err := p.groups.GetGroup(ctx, name)
	if err != nil {
		return view.GroupDetail{}, err
	}

	paths, err := p.accessPaths.AccessPathsOfGroup(ctx, name)
	if err != nil {
		return view.GroupDetail{}, err
	}

	if paths == nil {
		paths = []string{}
	}

	return view.GroupDetail{Group: groupView(group), AccessPaths: paths}, nil
}

// Create adds an empty group. A duplicate name is a conflict.
func (p *Group) Create(ctx context.Context, name string, opts model.WriteOptions) (view.Group, error) {
	name = strings.TrimSpace(name)

	group := model.Group{Name: name, Members: []string{}}
	if err := group.Validate(); err != nil {
		return view.Group{}, err
	}

	if err := p.writer.CreateGroup(ctx, group, opts); err != nil {
		return view.Group{}, err
	}

	return groupView(group), nil
}

// Delete removes the group definition and every `@group` ACL entry.
func (p *Group) Delete(ctx context.Context, name string, opts model.WriteOptions) error {
	name = strings.TrimSpace(name)

	if err := model.ValidateGroupName(name); err != nil {
		return err
	}

	return p.writer.DeleteGroup(ctx, name, opts)
}

// Members returns the member list of a group.
func (p *Group) Members(ctx context.Context, name string) (view.GroupMembers, error) {
	name = strings.TrimSpace(name)

	if err := model.ValidateGroupName(name); err != nil {
		return view.GroupMembers{}, err
	}

	group, err := p.groups.GetGroup(ctx, name)
	if err != nil {
		return view.GroupMembers{}, err
	}

	return groupMembersView(group), nil
}

// SetMembers replaces the whole member list of a group.
func (p *Group) SetMembers(ctx context.Context, name string, users []string, opts model.WriteOptions) (view.GroupMembers, error) {
	name = strings.TrimSpace(name)

	if err := model.ValidateGroupName(name); err != nil {
		return view.GroupMembers{}, err
	}

	if err := p.writer.SetGroupMembers(ctx, name, users, opts); err != nil {
		return view.GroupMembers{}, err
	}

	group, err := p.groups.GetGroup(ctx, name)
	if err != nil {
		return view.GroupMembers{}, err
	}

	return groupMembersView(group), nil
}

// AccessPaths lists the access paths whose ACL references the group.
func (p *Group) AccessPaths(ctx context.Context, name string) (view.GroupAccessPaths, error) {
	name = strings.TrimSpace(name)

	if err := model.ValidateGroupName(name); err != nil {
		return view.GroupAccessPaths{}, err
	}

	if _, err := p.groups.GetGroup(ctx, name); err != nil {
		return view.GroupAccessPaths{}, err
	}

	paths, err := p.accessPaths.AccessPathsOfGroup(ctx, name)
	if err != nil {
		return view.GroupAccessPaths{}, err
	}

	if paths == nil {
		paths = []string{}
	}

	return view.GroupAccessPaths{Name: name, AccessPaths: paths}, nil
}

func groupView(group model.Group) view.Group {
	members := make([]string, 0, len(group.Members))
	members = append(members, group.Members...)

	return view.Group{
		Name:        group.Name,
		Description: group.Description,
		Members:     members,
	}
}

func groupMembersView(group model.Group) view.GroupMembers {
	users := make([]string, 0, len(group.Members))
	users = append(users, group.Members...)

	return view.GroupMembers{Name: group.Name, Users: users}
}
