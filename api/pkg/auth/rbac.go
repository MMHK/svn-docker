package auth

import (
	"sort"

	"github.com/mmhk/svn-admin/pkg/model"
)

// Module names an API area.
type Module string

// Actions name what a role may do in a module.
const (
	ModuleUser           Module = "user"
	ModuleGroup          Module = "group"
	ModuleRepository     Module = "repository"
	ModuleAccessPath     Module = "access-path"
	ModuleProjectManager Module = "projectmanager"
	ModuleRole           Module = "role"
	ModuleSettings       Module = "settings"
)

// Action is a verb applied to a module.
type Action string

const (
	ActionView     Action = "view"
	ActionCreate   Action = "create"
	ActionUpdate   Action = "update"
	ActionDelete   Action = "delete"
	ActionAssign   Action = "assign"
	ActionUnassign Action = "unassign"
	// ActionAdd is the P3 spelling used by group and access-path creation. The
	// older ActionCreate remains for the user and repository modules.
	ActionAdd Action = "add"

	// ActionChangePass guards changing one's own password; the "-other"
	// variant guards resetting somebody else's (plan section 4.2).
	ActionChangePass      Action = "changepass"
	ActionChangePassOther Action = "changepass-other"
)

type rule struct {
	module Module
	action Action
	roles  []model.Role
}

// rules is the static module x action matrix from plan section 4.3. It
// replaces the v2 FSAclManager "clone the ACL and inject a role" hack.
var rules = []rule{
	{ModuleUser, ActionView, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleUser, ActionCreate, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleUser, ActionUpdate, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleUser, ActionDelete, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	// Every authenticated account may rotate its own password; resetting
	// somebody else's needs user management rights.
	{ModuleUser, ActionChangePass, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager, model.RoleUser}},
	{ModuleUser, ActionChangePassOther, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},

	{ModuleGroup, ActionView, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleGroup, ActionCreate, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleGroup, ActionAdd, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleGroup, ActionUpdate, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleGroup, ActionDelete, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},
	{ModuleGroup, ActionAssign, []model.Role{model.RoleAdministrator, model.RoleUserGroupManager}},

	{ModuleRepository, ActionView, []model.Role{model.RoleAdministrator, model.RoleRepositoryManager, model.RoleRepositoryCreator, model.RoleProjectManager, model.RoleUser}},
	{ModuleRepository, ActionCreate, []model.Role{model.RoleAdministrator, model.RoleRepositoryCreator}},
	// P4 spells repository creation as repository:add (plan T4.2); the older
	// repository:create spelling stays for compatibility.
	{ModuleRepository, ActionAdd, []model.Role{model.RoleAdministrator, model.RoleRepositoryCreator}},
	{ModuleRepository, ActionUpdate, []model.Role{model.RoleAdministrator, model.RoleRepositoryManager}},
	{ModuleRepository, ActionDelete, []model.Role{model.RoleAdministrator, model.RoleRepositoryManager}},

	{ModuleAccessPath, ActionView, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager, model.RoleProjectManager}},
	{ModuleAccessPath, ActionCreate, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager}},
	{ModuleAccessPath, ActionAdd, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager, model.RoleProjectManager}},
	{ModuleAccessPath, ActionUpdate, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager}},
	{ModuleAccessPath, ActionDelete, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager, model.RoleProjectManager}},
	{ModuleAccessPath, ActionAssign, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager, model.RoleProjectManager}},
	{ModuleAccessPath, ActionUnassign, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager, model.RoleProjectManager}},

	// Project-manager delegation is an access-path management action; a
	// Project-Manager cannot hand out scope to somebody else.
	{ModuleProjectManager, ActionView, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager}},
	{ModuleProjectManager, ActionAssign, []model.Role{model.RoleAdministrator, model.RoleAccessPathManager}},

	{ModuleRole, ActionView, []model.Role{model.RoleAdministrator, model.RoleRoleManager}},
	{ModuleRole, ActionAssign, []model.Role{model.RoleAdministrator, model.RoleRoleManager}},

	{ModuleSettings, ActionView, []model.Role{model.RoleAdministrator}},
}

// Allowed reports whether any held role grants module:action. Administrator
// always wins.
func Allowed(held []model.Role, module Module, action Action) bool {
	for _, role := range held {
		if role == model.RoleAdministrator {
			return true
		}
	}

	for _, entry := range rules {
		if entry.module != module || entry.action != action {
			continue
		}

		for _, granted := range entry.roles {
			for _, role := range held {
				if role == granted {
					return true
				}
			}
		}
	}

	return false
}

// Permissions renders the roles as a sorted `module:action` list for the
// session response.
func Permissions(held []model.Role) []string {
	seen := make(map[string]bool)

	for _, entry := range rules {
		if !Allowed(held, entry.module, entry.action) {
			continue
		}

		seen[string(entry.module)+":"+string(entry.action)] = true
	}

	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}

	sort.Strings(out)

	return out
}
