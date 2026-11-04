package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/view"
)

// listUsers godoc
//
//	@Summary		List SVN accounts
//	@Description	Paged list of htpasswd accounts with their groups. Password hashes are never returned.
//	@Tags			users
//	@Produce		json
//	@Security		bearerAuth
//	@Param			q		query		string	false	"case-insensitive name filter"
//	@Param			page	query		int		false	"page number (default 1)"
//	@Param			size	query		int		false	"page size (default 50, max 500)"
//	@Success		200		{object}	view.UserList
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users [get]
func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	query, err := parsePageQuery(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	list, err := a.user.List(r.Context(), query)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.UserList = list
	writeJSON(w, http.StatusOK, payload)
}

// getUserAccessPaths godoc
//
//	@Summary		List the effective access paths of an account
//	@Description	Returns every source of access: direct user entries, entries inherited through the account's groups and the global `*` entry. inherited is true for the last two; source is user, group or global.
//	@Tags			users
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"account name"
//	@Success		200		{object}	view.UserAccessPaths
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users/{name}/access-paths [get]
func (a *API) getUserAccessPaths(w http.ResponseWriter, r *http.Request) {
	paths, err := a.user.AccessPaths(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, paths)
}

// getUser godoc
//
//	@Summary		Get one SVN account
//	@Tags			users
//	@Produce		json
//	@Security		bearerAuth
//	@Param			name	path		string	true	"account name"
//	@Success		200		{object}	view.User
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		404		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/users/{name} [get]
func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	user, err := a.user.Get(r.Context(), r.PathValue("name"))
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.User = user
	writeJSON(w, http.StatusOK, payload)
}
