package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/view"
)

// listRoles godoc
//
//	@Summary		List the fixed roles
//	@Description	Returns the role catalogue used by userroleassignments.ini together with a description of each role.
//	@Tags			roles
//	@Produce		json
//	@Security		bearerAuth
//	@Success		200	{object}	view.RoleList
//	@Failure		401	{object}	view.ErrorEnvelope
//	@Failure		403	{object}	view.ErrorEnvelope
//	@Failure		503	{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/roles [get]
func (a *API) listRoles(w http.ResponseWriter, r *http.Request) {
	if err := a.authorize(r, auth.ModuleRole, auth.ActionView); err != nil {
		a.writeError(w, r, err)

		return
	}

	list, err := a.role.List(r.Context())
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	var payload view.RoleList = list
	writeJSON(w, http.StatusOK, payload)
}
