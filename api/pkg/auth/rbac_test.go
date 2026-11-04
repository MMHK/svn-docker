package auth

import (
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/model"
)

func TestAllowedMatrix(t *testing.T) {
	tests := []struct {
		name   string
		roles  []model.Role
		module Module
		action Action
		want   bool
	}{
		{"no role cannot view users", nil, ModuleUser, ActionView, false},
		{"plain user cannot view users", []model.Role{model.RoleUser}, ModuleUser, ActionView, false},
		{"administrator can do anything", []model.Role{model.RoleAdministrator}, ModuleRole, ActionAssign, true},
		{"administrator reads users", []model.Role{model.RoleAdministrator, model.RoleUser}, ModuleUser, ActionView, true},
		{"user manager reads users", []model.Role{model.RoleUserGroupManager}, ModuleUser, ActionView, true},
		{"user manager creates groups", []model.Role{model.RoleUserGroupManager}, ModuleGroup, ActionCreate, true},
		{"user manager adds groups", []model.Role{model.RoleUserGroupManager}, ModuleGroup, ActionAdd, true},
		{"plain user cannot add groups", []model.Role{model.RoleUser}, ModuleGroup, ActionAdd, false},
		{"user manager cannot manage repositories", []model.Role{model.RoleUserGroupManager}, ModuleRepository, ActionCreate, false},
		{"repository creator creates repositories", []model.Role{model.RoleRepositoryCreator}, ModuleRepository, ActionCreate, true},
		{"repository creator cannot delete repositories", []model.Role{model.RoleRepositoryCreator}, ModuleRepository, ActionDelete, false},
		{"repository manager deletes repositories", []model.Role{model.RoleRepositoryManager}, ModuleRepository, ActionDelete, true},
		{"access path manager edits ACLs", []model.Role{model.RoleAccessPathManager}, ModuleAccessPath, ActionUpdate, true},
		{"access path manager assigns ACLs", []model.Role{model.RoleAccessPathManager}, ModuleAccessPath, ActionAssign, true},
		{"project manager only reads access paths", []model.Role{model.RoleProjectManager}, ModuleAccessPath, ActionView, true},
		{"project manager assigns ACLs in scope", []model.Role{model.RoleProjectManager}, ModuleAccessPath, ActionAssign, true},
		{"project manager cannot edit access paths", []model.Role{model.RoleProjectManager}, ModuleAccessPath, ActionUpdate, false},
		{"project manager cannot delegate managers", []model.Role{model.RoleProjectManager}, ModuleProjectManager, ActionAssign, false},
		{"access path manager delegates managers", []model.Role{model.RoleAccessPathManager}, ModuleProjectManager, ActionAssign, true},
		{"role manager assigns roles", []model.Role{model.RoleRoleManager}, ModuleRole, ActionAssign, true},
		{"role manager cannot read users", []model.Role{model.RoleRoleManager}, ModuleUser, ActionView, false},
		{"plain user lists repositories", []model.Role{model.RoleUser}, ModuleRepository, ActionView, true},
		{"plain user cannot manage settings", []model.Role{model.RoleUser}, ModuleSettings, ActionView, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Allowed(test.roles, test.module, test.action); got != test.want {
				t.Errorf("Allowed(%v, %s, %s) = %v, want %v", test.roles, test.module, test.action, got, test.want)
			}
		})
	}
}

func TestPermissionsForAdministrator(t *testing.T) {
	permissions := Permissions([]model.Role{model.RoleAdministrator})
	if len(permissions) < 10 {
		t.Fatalf("admin permissions = %v, want the full matrix", permissions)
	}

	want := map[string]bool{
		"user:view":             true,
		"repository:create":     true,
		"access-path:view":      true,
		"access-path:add":       true,
		"access-path:assign":    true,
		"projectmanager:assign": true,
		"group:add":             true,
		"role:assign":           true,
		"settings:view":         true,
	}

	for _, permission := range permissions {
		delete(want, permission)
	}

	if len(want) != 0 {
		t.Errorf("admin permissions are missing %v", want)
	}
}

func TestPermissionsForPlainUser(t *testing.T) {
	permissions := Permissions([]model.Role{model.RoleUser})

	// A plain user may browse repositories and rotate their own password
	// (user:changepass from plan T2.3), and nothing else.
	want := []string{"repository:view", "user:changepass"}
	if len(permissions) != len(want) {
		t.Fatalf("user permissions = %v, want %v", permissions, want)
	}

	for index, permission := range want {
		if permissions[index] != permission {
			t.Errorf("user permission %d = %q, want %q", index, permissions[index], permission)
		}
	}

	if Allowed([]model.Role{model.RoleUser}, ModuleUser, ActionChangePassOther) {
		t.Error("a plain user may reset somebody else's password")
	}

	if len(Permissions(nil)) != 0 {
		t.Errorf("permissions without roles = %v, want empty", Permissions(nil))
	}
}

func TestRateLimiter(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)}
	limiter := NewRateLimiter(5, time.Minute)
	limiter.now = clock.Now

	for i := 0; i < 5; i++ {
		allowed, _ := limiter.Allow("10.0.0.7")
		if !allowed {
			t.Fatalf("attempt %d was rejected", i+1)
		}
	}

	allowed, retry := limiter.Allow("10.0.0.7")
	if allowed {
		t.Fatal("sixth attempt was allowed")
	}

	if retry <= 0 || retry > time.Minute {
		t.Errorf("retry after = %v, want between 0 and 1m", retry)
	}

	// Another IP is unaffected.
	if allowed, _ := limiter.Allow("10.0.0.8"); !allowed {
		t.Error("other client was rejected")
	}

	// The window rolls.
	clock.Advance(61 * time.Second)

	if allowed, _ := limiter.Allow("10.0.0.7"); !allowed {
		t.Error("attempt after the window was rejected")
	}
}
