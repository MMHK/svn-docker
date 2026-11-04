package httpapi

import (
	"encoding/base64"
	"net/http"

	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// listAccessPaths godoc
//
//	@Summary		List access paths
//	@Description	Every non-groups section of the authz file, with its ACL. Principals are returned verbatim so $authenticated, $anonymous and &aliases stay visible.
//	@Tags			access-paths
//	@Produce		json
//	@Security		bearerAuth
//	@Param			q		query		string	false	"case-insensitive path filter"
//	@Param			page	query		int		false	"page number (default 1)"
//	@Param			size	query		int		false	"page size (default 50, max 500)"
//	@Success		200		{object}	view.AccessPathList
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths [get]
func (a *API) listAccessPaths(w http.ResponseWriter, r *http.Request) {
	query, err := parsePageQuery(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	list, err := a.accessPath.List(r.Context(), query)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.AccessPathList = list
	writeJSON(w, http.StatusOK, payload)
}

// getAccessPathACL godoc
//
//	@Summary		Get one access path ACL
//	@Description	The path segment is the URL-safe base64 encoding of values such as project:/trunk; ?path= is accepted instead.
//	@Tags			access-paths
//	@Produce		json
//	@Security		bearerAuth
//	@Param			path	path		string	true	"URL-safe base64 of <repo>:/<path>"
//	@Success		200		{object}	view.AccessPath
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths/{path}/acl [get]
func (a *API) getAccessPathACL(w http.ResponseWriter, r *http.Request) {
	path, err := accessPathFromRequest(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	accessPath, err := a.accessPath.Get(r.Context(), path)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, accessPath)
}

// createAccessPath godoc
//
//	@Summary		Create an access path
//	@Description	Appends a `<repo>:/<path>` section. The path must match ^[A-Za-z0-9_-]{1,64}:/.*$ and must not contain '..', NUL, a control character or a backslash.
//	@Tags			access-paths
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			request		body		view.CreateAccessPathRequest	true	"new access path"
//	@Param			If-Match	header		string						false	"sha256 of the authz file the edit is based on"
//	@Success		201			{object}	view.AccessPath
//	@Failure		401			{object}	view.ErrorEnvelope
//	@Failure		403			{object}	view.ErrorEnvelope
//	@Failure		409			{object}	view.ErrorEnvelope	"the path already exists or the file changed"
//	@Failure		422			{object}	view.ErrorEnvelope
//	@Failure		503			{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths [post]
func (a *API) createAccessPath(w http.ResponseWriter, r *http.Request) {
	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionAccessPathCreate, "", err)
		a.writeError(w, r, err)

		return
	}

	var request view.CreateAccessPathRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionAccessPathCreate, request.Path, err)
		a.writeError(w, r, err)

		return
	}

	created, err := a.accessPath.Create(r.Context(), request.Path, opts)
	if err != nil {
		a.auditWrite(r, actionAccessPathCreate, request.Path, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionAccessPathCreate, created.Path, nil)

	encoded := base64.RawURLEncoding.EncodeToString([]byte(created.Path))
	w.Header().Set("Location", "/svnadmin/api/v1/access-paths/"+encoded+"/acl")
	writeJSON(w, http.StatusCreated, created)
}

// deleteAccessPath godoc
//
//	@Summary		Delete an access path
//	@Description	Removes the section and every ACL entry in it.
//	@Tags			access-paths
//	@Security		bearerAuth
//	@Param			path	path	string	true	"URL-safe base64 of <repo>:/<path>"
//	@Success		204
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		500		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths/{path} [delete]
func (a *API) deleteAccessPath(w http.ResponseWriter, r *http.Request) {
	path, err := accessPathFromRequest(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	if err := a.accessPath.Delete(r.Context(), path, model.WriteOptions{}); err != nil {
		a.auditWrite(r, actionAccessPathDelete, path, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionAccessPathDelete, path, nil)

	w.WriteHeader(http.StatusNoContent)
}

// setAccessPathACL godoc
//
//	@Summary		Set one ACL entry
//	@Description	Writes `principal = permission` for a user, an @group or `*`. The permission is none, r or rw; none removes the line. $authenticated, $anonymous and &aliases are read-only and rejected with 422.
//	@Tags			access-paths
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			path		path		string							true	"URL-safe base64 of <repo>:/<path>"
//	@Param			principal	path		string							true	"user name, @group or *"
//	@Param			request		body		view.SetACLPermissionRequest	true	"permission"
//	@Param			If-Match	header		string							false	"sha256 of the authz file the edit is based on"
//	@Success		200			{object}	view.AccessPath
//	@Failure		401			{object}	view.ErrorEnvelope
//	@Failure		403			{object}	view.ErrorEnvelope
//	@Failure		404			{object}	view.ErrorEnvelope
//	@Failure		409			{object}	view.ErrorEnvelope
//	@Failure		422			{object}	view.ErrorEnvelope
//	@Failure		503			{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths/{path}/acl/{principal} [put]
func (a *API) setAccessPathACL(w http.ResponseWriter, r *http.Request) {
	path, err := accessPathFromRequest(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	principal := r.PathValue("principal")
	if principal == "" {
		err := &model.ValidationError{Field: "principal", Reason: "is required"}
		a.writeError(w, r, err)

		return
	}

	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionAccessPathACL, path, err)
		a.writeError(w, r, err)

		return
	}

	var request view.SetACLPermissionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionAccessPathACL, path, err)
		a.writeError(w, r, err)

		return
	}

	permission, err := model.ParsePermission(request.Permission)
	if err != nil {
		a.auditWrite(r, actionAccessPathACL, path, err)
		a.writeError(w, r, err)

		return
	}

	updated, err := a.accessPath.SetACL(r.Context(), path, principal, permission, opts)
	if err != nil {
		a.auditWrite(r, actionAccessPathACL, path, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionAccessPathACL, path, nil)
	writeJSON(w, http.StatusOK, updated)
}

// deleteAccessPathACL godoc
//
//	@Summary		Remove one ACL entry
//	@Description	Deletes every line for the principal. Removing an entry that is already absent is a no-op. Read-only principals are rejected with 422.
//	@Tags			access-paths
//	@Security		bearerAuth
//	@Param			path		path	string	true	"URL-safe base64 of <repo>:/<path>"
//	@Param			principal	path	string	true	"user name, @group or *"
//	@Success		204
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/access-paths/{path}/acl/{principal} [delete]
func (a *API) deleteAccessPathACL(w http.ResponseWriter, r *http.Request) {
	path, err := accessPathFromRequest(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	principal := r.PathValue("principal")
	if principal == "" {
		err := &model.ValidationError{Field: "principal", Reason: "is required"}
		a.writeError(w, r, err)

		return
	}

	if err := a.accessPath.DeleteACL(r.Context(), path, principal, model.WriteOptions{}); err != nil {
		a.auditWrite(r, actionAccessPathACL, path, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionAccessPathACL, path, nil)

	w.WriteHeader(http.StatusNoContent)
}
