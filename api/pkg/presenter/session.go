package presenter

import (
	"context"
	"strings"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Session serves the login/logout/refresh use cases.
type Session struct {
	credentials model.CredentialStore
	roles       model.RoleStore
	tokens      *auth.Manager
	limiter     *auth.RateLimiter
}

// NewSession builds the session presenter.
func NewSession(
	credentials model.CredentialStore,
	roles model.RoleStore,
	tokens *auth.Manager,
	limiter *auth.RateLimiter,
) *Session {
	return &Session{credentials: credentials, roles: roles, tokens: tokens, limiter: limiter}
}

// Login verifies credentials and issues an opaque bearer token. clientIP keys
// the rate limiter.
func (p *Session) Login(ctx context.Context, username, password, clientIP string) (view.Session, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return view.Session{}, &model.ValidationError{Field: "credentials", Reason: "username and password are required"}
	}

	if allowed, retry := p.limiter.Allow(clientIP); !allowed {
		return view.Session{}, &model.RateLimitError{RetryAfter: retry}
	}

	if err := p.credentials.VerifyPassword(ctx, username, password); err != nil {
		if model.IsUnauthorized(err) {
			return view.Session{}, &model.UnauthorizedError{Reason: "invalid credentials"}
		}

		return view.Session{}, err
	}

	roles, err := p.rolesOf(ctx, username)
	if err != nil {
		return view.Session{}, err
	}

	token, session, err := p.tokens.Issue(username, roles)
	if err != nil {
		return view.Session{}, err
	}

	return sessionView(session, token), nil
}

// Refresh rotates the token presented by the caller.
func (p *Session) Refresh(_ context.Context, token string) (view.Session, error) {
	newToken, session, err := p.tokens.Refresh(token)
	if err != nil {
		return view.Session{}, err
	}

	return sessionView(session, newToken), nil
}

// Logout revokes the token presented by the caller.
func (p *Session) Logout(_ context.Context, token string) error {
	return p.tokens.Revoke(token)
}

// Describe renders an already validated session without exposing the token.
func (p *Session) Describe(session auth.Session) view.Session {
	return sessionView(session, "")
}

func (p *Session) rolesOf(ctx context.Context, username string) ([]model.Role, error) {
	assignments, err := p.roles.Assignments(ctx)
	if err != nil {
		return nil, err
	}

	roles := assignments.Roles[username]
	if roles == nil {
		roles = []model.Role{}
	}

	return roles, nil
}

func sessionView(session auth.Session, token string) view.Session {
	roles := make([]string, 0, len(session.Roles))
	for _, role := range session.Roles {
		roles = append(roles, string(role))
	}

	return view.Session{
		Token:         token,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
		Me:            view.SessionUser{Name: session.Username, Roles: roles},
		Roles:         roles,
		Permissions:   auth.Permissions(session.Roles),
	}
}
