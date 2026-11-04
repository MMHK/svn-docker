package presenter

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Setup serves the first-administrator bootstrap flow (plan D6, task T1.6).
// It replaces the v2 admin/admin backdoor: a one-time token, printed to stderr
// at startup when SVN_ADMIN_BOOTSTRAP_TOKEN is not set, is required.
type Setup struct {
	roles     model.RoleStore
	users     model.UserStore
	bootstrap string
}

// NewSetup builds the setup presenter.
func NewSetup(roles model.RoleStore, users model.UserStore, bootstrapToken string) *Setup {
	return &Setup{roles: roles, users: users, bootstrap: bootstrapToken}
}

// Required reports whether no user holds the Administrator role yet.
func (p *Setup) Required(ctx context.Context) (bool, error) {
	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return false, err
	}

	for _, roles := range assignments.Roles {
		for _, role := range roles {
			if role == model.RoleAdministrator {
				return false, nil
			}
		}
	}

	return true, nil
}

// Admin grants the Administrator role to an existing account, provided the
// bootstrap token matches and no administrator exists yet.
func (p *Setup) Admin(ctx context.Context, username, token string) (view.SetupResult, error) {
	username = strings.TrimSpace(username)

	if p.bootstrap == "" || token == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(p.bootstrap)) != 1 {
		return view.SetupResult{}, &model.ForbiddenError{Reason: "invalid bootstrap token"}
	}

	if err := (model.User{Name: username}).Validate(); err != nil {
		return view.SetupResult{}, err
	}

	required, err := p.Required(ctx)
	if err != nil {
		return view.SetupResult{}, err
	}

	if !required {
		return view.SetupResult{}, &model.ConflictError{
			Kind:   model.KindUser,
			Name:   username,
			Reason: "an administrator already exists",
		}
	}

	if _, err := p.users.GetUser(ctx, username); err != nil {
		if model.IsNotFound(err) {
			return view.SetupResult{}, &model.ValidationError{Field: "username", Reason: "unknown account; create it before granting the role"}
		}

		return view.SetupResult{}, err
	}

	if err := p.roles.AssignRole(ctx, username, model.RoleAdministrator); err != nil {
		return view.SetupResult{}, err
	}

	return view.SetupResult{Username: username, Role: string(model.RoleAdministrator)}, nil
}
