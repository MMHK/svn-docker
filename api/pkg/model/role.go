package model

import "strings"

// Role is one of the fixed roles understood by the management API. The names
// match the v2 userroleassignments.ini values so a rollback stays readable.
type Role string

const (
	RoleUser              Role = "User"
	RoleUserGroupManager  Role = "User-Group-Manager"
	RoleAccessPathManager Role = "Access-Path-Manager"
	RoleRepositoryCreator Role = "Repository-Creator"
	RoleRepositoryManager Role = "Repository-Manager"
	RoleRoleManager       Role = "Role-Manager"
	RoleAdministrator     Role = "Administrator"
	RoleProjectManager    Role = "Project-Manager"
)

// allRoles is the canonical order used by the API and the UI.
var allRoles = []Role{
	RoleUser,
	RoleUserGroupManager,
	RoleAccessPathManager,
	RoleRepositoryCreator,
	RoleRepositoryManager,
	RoleRoleManager,
	RoleAdministrator,
	RoleProjectManager,
}

// AllRoles returns a copy of the fixed role list.
func AllRoles() []Role {
	out := make([]Role, len(allRoles))
	copy(out, allRoles)

	return out
}

// Valid reports whether r is a known role.
func (r Role) Valid() bool {
	for _, known := range allRoles {
		if r == known {
			return true
		}
	}

	return false
}

// ParseRole resolves a submitted role name. Matching is case-insensitive but
// the canonical spelling is always returned, so the file never gains a
// duplicate role under a different case.
func ParseRole(value string) (Role, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", &ValidationError{Field: "roles", Reason: "must not be empty"}
	}

	for _, known := range allRoles {
		if strings.EqualFold(string(known), trimmed) {
			return known, nil
		}
	}

	return "", &ValidationError{Field: "roles", Reason: "unknown role " + trimmed}
}

// Description returns the human readable purpose of the role.
func (r Role) Description() string {
	switch r {
	case RoleUser:
		return "Can sign in and browse permitted repositories."
	case RoleUserGroupManager:
		return "Can create and manage users and groups."
	case RoleAccessPathManager:
		return "Can create access paths and edit ACL entries."
	case RoleRepositoryCreator:
		return "Can create repositories."
	case RoleRepositoryManager:
		return "Can inspect and delete repositories."
	case RoleRoleManager:
		return "Can assign roles to users."
	case RoleAdministrator:
		return "Full control over the management API."
	case RoleProjectManager:
		return "Can manage access paths delegated to them."
	default:
		return ""
	}
}
