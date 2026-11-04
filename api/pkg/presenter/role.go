package presenter

import (
	"context"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Role serves the fixed role catalogue.
type Role struct {
	roles model.RoleStore
}

// NewRole builds the role presenter.
func NewRole(roles model.RoleStore) *Role {
	return &Role{roles: roles}
}

// List returns every fixed role with its description.
func (p *Role) List(ctx context.Context) (view.RoleList, error) {
	roles, err := p.roles.ListRoles(ctx)
	if err != nil {
		return view.RoleList{}, err
	}

	items := make([]view.Role, 0, len(roles))
	for _, role := range roles {
		items = append(items, view.Role{Name: string(role), Description: role.Description()})
	}

	return view.RoleList{Items: items}, nil
}
