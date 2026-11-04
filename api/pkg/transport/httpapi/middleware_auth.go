package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
)

type sessionKey struct{}

func sessionFromContext(ctx context.Context) (auth.Session, bool) {
	session, ok := ctx.Value(sessionKey{}).(auth.Session)

	return session, ok
}

// isAPIPath reports whether the route belongs to the management API. The SPA
// shell and its assets are intentionally public: a browser cannot attach an
// Authorization header to a document or script request, and the login page
// itself has to load before a token exists.
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/svnadmin/api/") || strings.HasPrefix(path, "/api/")
}

// authExempt lists the only API routes reachable without a bearer token.
func authExempt(r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/svnadmin/api/docs") {
		return true
	}

	if r.Method == http.MethodPost {
		switch r.URL.Path {
		case "/svnadmin/api/v1/session", "/svnadmin/api/v1/setup/admin":
			return true
		}
	}

	if r.Method == http.MethodGet && r.URL.Path == "/svnadmin/api/v1/setup/status" {
		return true
	}

	return false
}

// authMiddleware enforces the bearer token and the first-run setup gate for
// every management API route. It never sets a cookie (plan D9).
func (a *API) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAPIPath(r.URL.Path) || authExempt(r) {
			next.ServeHTTP(w, r)

			return
		}

		required, err := a.setup.Required(r.Context())
		if err != nil {
			a.writeError(w, r, err)

			return
		}

		if required {
			a.writeError(w, r, &model.SetupRequiredError{})

			return
		}

		token, err := bearerToken(r)
		if err != nil {
			a.logger.Warn(
				"request rejected: missing bearer token",
				slog.String("path", r.URL.Path),
				slog.String("request_id", requestIDFromContext(r.Context())),
			)
			a.writeError(w, r, &model.UnauthorizedError{Reason: "missing bearer token"})

			return
		}

		session, err := a.tokens.Validate(token)
		if err != nil {
			a.logger.Warn(
				"request rejected: invalid token",
				slog.String("path", r.URL.Path),
				slog.String("request_id", requestIDFromContext(r.Context())),
			)
			a.writeError(w, r, err)

			return
		}

		ctx := context.WithValue(r.Context(), sessionKey{}, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", auth.ErrEmptyToken
	}

	scheme, value, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", errors.New("httpapi: malformed Authorization header")
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", auth.ErrEmptyToken
	}

	return value, nil
}

// authorize applies the static RBAC matrix to the validated session.
func (a *API) authorize(r *http.Request, module auth.Module, action auth.Action) error {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		return &model.UnauthorizedError{Reason: "authentication required"}
	}

	if !auth.Allowed(session.Roles, module, action) {
		return &model.ForbiddenError{Reason: "role does not permit this operation"}
	}

	return nil
}

// sessionToken returns the raw bearer token for refresh/logout.
func sessionToken(r *http.Request) (string, error) {
	return bearerToken(r)
}
