package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/view"
)

// listRepositories godoc
//
//	@Summary		List repositories
//	@Description	Lists the directories under SVNParentPath that contain format, db/ and conf/, so stray directories are not reported as repositories.
//	@Tags			repositories
//	@Produce		json
//	@Security		bearerAuth
//	@Param			q		query		string	false	"case-insensitive name filter"
//	@Param			page	query		int		false	"page number (default 1)"
//	@Param			size	query		int		false	"page size (default 50, max 500)"
//	@Success		200		{object}	view.RepositoryList
//	@Failure		401		{object}	view.ErrorEnvelope
//	@Failure		403		{object}	view.ErrorEnvelope
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Failure		503		{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/repos [get]
func (a *API) listRepositories(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRepository, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	query, err := parsePageQuery(r)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	list, err := a.repo.List(r.Context(), query)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.RepositoryList = list
	writeJSON(w, http.StatusOK, payload)
}
