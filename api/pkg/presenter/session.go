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
	admins      model.AdminStore
	tokens      *auth.Manager
	limiter     *auth.RateLimiter
}

// NewSession builds the session presenter.
func NewSession(
	credentials model.CredentialStore,
	admins model.AdminStore,
	tokens *auth.Manager,
	limiter *auth.RateLimiter,
) *Session {
	return &Session{credentials: credentials, admins: admins, tokens: tokens, limiter: limiter}
}

// Login verifies credentials and issues an opaque bearer token. clientIP keys
// the rate limiter. Only the single administrator recorded in the admin file
// may sign in; other valid accounts get 403 (plan P9, D30).
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

	admin, err := p.currentAdmin(ctx)
	if err != nil {
		return view.Session{}, err
	}

	if admin == "" {
		return view.Session{}, &model.SetupRequiredError{}
	}

	if username != admin {
		return view.Session{}, &model.NotAdministratorError{Reason: "the account is not the svnadmin administrator"}
	}

	token, session, err := p.tokens.Issue(username)
	if err != nil {
		return view.Session{}, err
	}

	return sessionView(session, token), nil
}

// Refresh rotates the token presented by the caller. The account must still be
// the administrator, otherwise the freshly issued token is revoked immediately.
func (p *Session) Refresh(ctx context.Context, token string) (view.Session, error) {
	newToken, session, err := p.tokens.Refresh(token)
	if err != nil {
		return view.Session{}, err
	}

	admin, err := p.currentAdmin(ctx)
	if err != nil {
		_ = p.tokens.Revoke(newToken)

		return view.Session{}, err
	}

	if admin == "" || session.Username != admin {
		_ = p.tokens.Revoke(newToken)

		return view.Session{}, &model.NotAdministratorError{Reason: "the account is not the svnadmin administrator"}
	}

	return sessionView(session, newToken), nil
}

// IsAdministrator reports whether username is the account recorded in the
// admin file. It is used by the bearer middleware so a token stops working as
// soon as the file is edited (plan D30, D31).
func (p *Session) IsAdministrator(ctx context.Context, username string) (bool, error) {
	admin, err := p.currentAdmin(ctx)
	if err != nil {
		return false, err
	}

	return admin != "" && username == admin, nil
}

func (p *Session) currentAdmin(ctx context.Context) (string, error) {
	if p.admins == nil {
		return "", nil
	}

	return p.admins.Admin(ctx)
}

// Logout revokes the token presented by the caller.
func (p *Session) Logout(_ context.Context, token string) error {
	return p.tokens.Revoke(token)
}

// Describe renders an already validated session without exposing the token.
func (p *Session) Describe(session auth.Session) view.Session {
	return sessionView(session, "")
}

func sessionView(session auth.Session, token string) view.Session {
	return view.Session{
		Token:         token,
		ExpiresAt:     session.ExpiresAt,
		IdleExpiresAt: session.IdleExpiresAt,
		Me:            view.SessionUser{Name: session.Username},
	}
}
