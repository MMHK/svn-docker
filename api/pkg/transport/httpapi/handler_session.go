package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// login godoc
//
//	@Summary		Create a session
//	@Description	Verifies the htpasswd credentials and returns an opaque bearer token. The token is only returned here and by POST /session/refresh; it is never set as a cookie. Five attempts per minute per client IP are allowed.
//	@Tags			session
//	@Accept			json
//	@Produce		json
//	@Param			request	body		view.LoginRequest	true	"username and password"
//	@Success		200		{object}	view.Session
//	@Failure		401		{object}	view.ErrorEnvelope	"invalid credentials"
//	@Failure		403		{object}	view.ErrorEnvelope	"source address is not allowed"
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		429		{object}	view.ErrorEnvelope	"too many login attempts"
//	@Router			/svnadmin/api/v1/session [post]
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var request view.LoginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)

		return
	}

	session, err := a.session.Login(r.Context(), request.Username, request.Password, a.clientIP(r))
	if err != nil {
		a.auditLogin(r, request.Username, err)
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, session)
}

// currentSession godoc
//
//	@Summary		Describe the current session
//	@Description	Returns the authenticated user, roles and permissions. The token itself is never echoed back.
//	@Tags			session
//	@Produce		json
//	@Security		bearerAuth
//	@Success		200	{object}	view.Session
//	@Failure		401	{object}	view.ErrorEnvelope
//	@Failure		403	{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/session [get]
func (a *API) currentSession(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		a.writeError(w, r, &model.UnauthorizedError{Reason: "authentication required"})

		return
	}

	writeJSON(w, http.StatusOK, a.session.Describe(session))
}

// refreshSession godoc
//
//	@Summary		Rotate the session token
//	@Description	Issues a new token and revokes the presented one immediately, so the old token can never be used again.
//	@Tags			session
//	@Produce		json
//	@Security		bearerAuth
//	@Success		200	{object}	view.Session
//	@Failure		401	{object}	view.ErrorEnvelope
//	@Failure		403	{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/session/refresh [post]
func (a *API) refreshSession(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		a.writeError(w, r, &model.UnauthorizedError{Reason: "missing bearer token"})

		return
	}

	session, err := a.session.Refresh(r.Context(), token)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, session)
}

// logout godoc
//
//	@Summary		Revoke the current session
//	@Description	Revokes the bearer token. The call is idempotent and returns 204 with no body.
//	@Tags			session
//	@Security		bearerAuth
//	@Success		204
//	@Failure		401	{object}	view.ErrorEnvelope
//	@Failure		403	{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/session [delete]
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		a.writeError(w, r, &model.UnauthorizedError{Reason: "missing bearer token"})

		return
	}

	if err := a.session.Logout(r.Context(), token); err != nil {
		a.writeError(w, r, err)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
