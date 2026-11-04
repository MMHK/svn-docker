package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/model"
	"github.com/mmhk/svn-admin/pkg/view"
)

// listGroups godoc
//
//	@Summary		List authz groups
//	@Tags			groups
//	@Produce		json
//	@Security		bearerAuth
//	@Param			q		query		string	false	"case-insensitive name filter"
//	@Param			page	query		int		false	"page number (default 1)"
//	@Param			size	query		int		false	"page size (default 50, max 500)"
//	@Success		200		{object}	view.GroupList
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups [get]
func (a *API) listGroups(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	query, err := parsePageQuery(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	list, err := a.group.List(r.Context(), query)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.GroupList = list
	writeJSON(w, http.StatusOK, payload)
}

// getGroup godoc
//
//	@Summary		Get one authz group
//	@Description	Returns the group members and the access paths whose ACL references the group.
//	@Tags			groups
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"group name"
//	@Success		200		{object}	view.GroupDetail
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups/{name} [get]
func (a *API) getGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	group, err := a.group.Get(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.GroupDetail = group
	writeJSON(w, http.StatusOK, payload)
}

// createGroup godoc
//
//	@Summary		Create an authz group
//	@Description	Appends a `name =` line to the [groups] block without rewriting any other line. The name must match ^[A-Za-z0-9_-]{1,64}$.
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			request		body		view.CreateGroupRequest	true	"new group"
//	@Param			If-Match	header		string					false	"sha256 of the authz file the edit is based on"
//	@Success		201			{object}	view.Group
//	@Failure		401			{object}	view.ErrorEnvelope
//	@Failure		403			{object}	view.ErrorEnvelope
//	@Failure		409			{object}	view.ErrorEnvelope	"the group already exists or the file changed"
//	@Failure		422			{object}	view.ErrorEnvelope	"invalid group name"
//	@Failure		503			{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups [post]
func (a *API) createGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionAdd); err != nil {
		a.writeError(w, r, err)

		return
	}

	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionGroupCreate, "", err)
		a.writeError(w, r, err)

		return
	}

	var request view.CreateGroupRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionGroupCreate, request.Name, err)
		a.writeError(w, r, err)

		return
	}

	group, err := a.group.Create(r.Context(), request.Name, opts)
	if err != nil {
		a.auditWrite(r, actionGroupCreate, request.Name, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionGroupCreate, group.Name, nil)

	w.Header().Set("Location", "/svnadmin/api/v1/groups/"+group.Name)
	writeJSON(w, http.StatusCreated, group)
}

// deleteGroup godoc
//
//	@Summary		Delete an authz group
//	@Description	Removes every `name =` line from [groups] and every `@name` entry from access-path sections. Users with the same name are never touched.
//	@Tags			groups
//	@Security		bearerAuth
//	@Param			name	path	string	true	"group name"
//	@Success		204
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups/{name} [delete]
func (a *API) deleteGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionDelete); err != nil {
		a.writeError(w, r, err)

		return
	}

	target := r.PathValue("name")

	if err := a.group.Delete(r.Context(), target, model.WriteOptions{}); err != nil {
		a.auditWrite(r, actionGroupDelete, target, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionGroupDelete, target, nil)

	w.WriteHeader(http.StatusNoContent)
}

// getGroupMembers godoc
//
//	@Summary		Read the members of a group
//	@Tags			groups
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"group name"
//	@Success		200		{object}	view.GroupMembers
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups/{name}/members [get]
func (a *API) getGroupMembers(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	members, err := a.group.Members(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, members)
}

// setGroupMembers godoc
//
//	@Summary		Replace the members of a group
//	@Description	Rewrites only the matching `[groups]` line. The submitted users replace the previous member list; every name is validated.
//	@Tags			groups
//	@Accept			json
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name		path		string					true	"group name"
//	@Param			request		body		view.GroupMembersRequest	true	"complete member list"
//	@Param			If-Match	header		string					false	"sha256 of the authz file the edit is based on"
//	@Success		200			{object}	view.GroupMembers
//	@Failure		401			{object}	view.ErrorEnvelope
//	@Failure		403			{object}	view.ErrorEnvelope
//	@Failure		404			{object}	view.ErrorEnvelope
//	@Failure		409			{object}	view.ErrorEnvelope
//	@Failure		422			{object}	view.ErrorEnvelope
//	@Failure		503			{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups/{name}/members [put]
func (a *API) setGroupMembers(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionAssign); err != nil {
		a.writeError(w, r, err)

		return
	}

	target := r.PathValue("name")

	opts, err := ifMatchOptions(r)
	if err != nil {
		a.auditWrite(r, actionGroupMembers, target, err)
		a.writeError(w, r, err)

		return
	}

	var request view.GroupMembersRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.auditWrite(r, actionGroupMembers, target, err)
		a.writeError(w, r, err)

		return
	}

	members, err := a.group.SetMembers(r.Context(), target, request.Users, opts)
	if err != nil {
		a.auditWrite(r, actionGroupMembers, target, err)
		a.writeError(w, r, err)

		return
	}

	a.auditWrite(r, actionGroupMembers, target, nil)
	writeJSON(w, http.StatusOK, members)
}

// getGroupAccessPaths godoc
//
//	@Summary		List the access paths that reference a group
//	@Tags			groups
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"group name"
//	@Success		200		{object}	view.GroupAccessPaths
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/groups/{name}/access-paths [get]
func (a *API) getGroupAccessPaths(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleGroup, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	paths, err := a.group.AccessPaths(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, paths)
}
