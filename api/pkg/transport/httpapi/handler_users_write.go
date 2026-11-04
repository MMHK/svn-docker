package httpapi

import (
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// Write-action names for the stderr operation log (plan T2.5). Each entry
// records actor, target and result; passwords, tokens and hashes never appear.
const (
	actionUserCreate   = "user.create"
	actionUserDelete   = "user.delete"
	actionUserPassword = "user.password"
	actionUserGroups   = "user.groups"
	actionSessionLogin = "session.login"
)

// auditWrite emits one structured line per write attempt. Failures get an
// additional warn line with the error text so an operator can diagnose the
// request without the client ever seeing internals.
func (a *API) auditWrite(r *http.Request, action, target string, err error) {
	actor := ""

	if session, ok := sessionFromContext(r.Context()); ok {
		actor = session.Username
	}

	result := "success"
	if err != nil {
		result = "failure"
	}

	attributes := []slog.Attr{
		slog.String("action", action),
		slog.String("actor", actor),
		slog.String("target", target),
		slog.String("result", result),
		slog.String("request_id", requestIDFromContext(r.Context())),
	}

	if err != nil {
		attributes = append(attributes, slog.String("error", err.Error()))
	}

	a.logger.LogAttrs(r.Context(), slog.LevelInfo, "write operation", attributes...)
}

// auditLogin records a rejected login attempt. It logs the attempted account
// name but never the password, the token or the hash.
func (a *API) auditLogin(r *http.Request, username string, err error) {
	reason := "unauthorized"

	if model.IsRateLimited(err) {
		reason = "rate_limited"
	}

	a.logger.Warn(
		"login failed",
		slog.String("action", actionSessionLogin),
		slog.String("actor", username),
		slog.String("result", "failure"),
		slog.String("reason", reason),
		slog.String("ip", a.clientIP(r)),
		slog.String("request_id", requestIDFromContext(r.Context())),
	)
}

// ifMatchOptions reads the optional If-Match precondition. A malformed value is
// a 422 instead of being ignored silently.
func ifMatchOptions(r *http.Request) (model.WriteOptions, error) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" {
		return model.WriteOptions{}, nil
	}

	raw = strings.TrimPrefix(raw, "W/")
	raw = strings.Trim(raw, `"`)

	if len(raw) != 64 {
		return model.WriteOptions{}, &model.ValidationError{
			Field:  "If-Match",
			Reason: "must be a sha256 hex digest",
		}
	}

	if _, err := hex.DecodeString(raw); err != nil {
		return model.WriteOptions{}, &model.ValidationError{
			Field:  "If-Match",
			Reason: "must be a sha256 hex digest",
		}
	}

	return model.WriteOptions{ExpectedHash: strings.ToLower(raw)}, nil
}

// createUser godoc
//
//	@Summary		Create an SVN account
//	@Description	Adds a `user:$apr1$...` line to the htpasswd file without disturbing any other line. The password is hashed inside the store and never returned or logged.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			request	body		view.CreateUserRequest	true	"new account"
//	@Success		201		{object}	view.User
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		409		{object}	view.ErrorEnvelope	"the account already exists"
//	@Failure		422		{object}	view.ErrorEnvelope	"invalid name or password"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users [post]
func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var request view.CreateUserRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionUserCreate, request.Username, err)
		a.writeError(w, r, err)

		return
	}

	user, err := a.user.Create(r.Context(), request.Username, request.Password)
	if err != nil {
		a.auditWrite(r, actionUserCreate, request.Username, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionUserCreate, user.Name, nil)

	w.Header().Set("Location", "/svnadmin/api/v1/users/"+user.Name)
	writeJSON(w, http.StatusCreated, user)
}

// deleteUser godoc
//
//	@Summary		Delete an SVN account
//	@Description	Removes the account from every authz group, every access-path ACL and finally the htpasswd file. The steps run in that order; a failure returns 500 with the steps that already completed. `*` is rejected.
//	@Tags			users
//	@Security		bearerAuth
//	@Param			name	path	string	true	"account name"
//	@Success		204
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope	"invalid name such as *"
//	@Failure		500		{object}	view.ErrorEnvelope	"multi-step delete failed"
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users/{name} [delete]
func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("name")

	if err := a.user.Delete(r.Context(), target); err != nil {
		a.auditWrite(r, actionUserDelete, target, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionUserDelete, target, nil)

	w.WriteHeader(http.StatusNoContent)
}

// changeUserPassword godoc
//
//	@Summary		Change an account password
//	@Description	Rewrites only the target htpasswd line with a fresh APR1 hash and revokes every token of that account, including the caller's own token when they rotate their own password. Every authenticated operator may change any account password.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string						true	"account name"
//	@Param			request	body		view.ChangePasswordRequest	true	"new password"
//	@Param			If-Match	header	string						false	"sha256 of the credentials file the edit is based on"
//	@Success		204
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		409		{object}	view.ErrorEnvelope	"the file changed since it was read"
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users/{name}/password [put]
func (a *API) changeUserPassword(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.PathValue("name"))

	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionUserPassword, target, err)
		a.writeError(w, r, err)

		return
	}

	var request view.ChangePasswordRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionUserPassword, target, err)
		a.writeError(w, r, err)

		return
	}

	if err := a.user.ChangePassword(r.Context(), target, request.Password, opts); err != nil {
		a.auditWrite(r, actionUserPassword, target, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionUserPassword, target, nil)

	w.WriteHeader(http.StatusNoContent)
}

// getUserGroups godoc
//
//	@Summary		Read the authz groups of an account
//	@Tags			users
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"account name"
//	@Success		200		{object}	view.UserGroups
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/users/{name}/groups [get]
func (a *API) getUserGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.user.Groups(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, groups)
}

// setUserGroups godoc
//
//	@Summary		Replace the group membership of an account
//	@Description	Edits only the necessary `[groups]` lines of the authz file. Every submitted group must already exist; unknown groups are rejected with 422.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string					true	"account name"
//	@Param			request	body		view.UserGroupsRequest	true	"complete group list"
//	@Param			If-Match	header	string					false	"sha256 of the authz file the edit is based on"
//	@Success		200		{object}	view.UserGroups
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		409		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/users/{name}/groups [put]
func (a *API) setUserGroups(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("name")

	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionUserGroups, target, err)
		a.writeError(w, r, err)

		return
	}

	var request view.UserGroupsRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionUserGroups, target, err)
		a.writeError(w, r, err)

		return
	}

	updated, err := a.user.SetGroups(r.Context(), target, request.Groups, opts)
	if err != nil {
		a.auditWrite(r, actionUserGroups, target, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionUserGroups, target, nil)
	writeJSON(w, http.StatusOK, updated)
}
