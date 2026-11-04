package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/view"
)

// setupAdmin godoc
//
//	@Summary		Create the first administrator
//	@Description	Requires the one-time bootstrap token from SVN_ADMIN_BOOTSTRAP_TOKEN (or the token printed to stderr at startup). It grants the Administrator role to an existing SVN account and fails once an administrator already exists. Other API routes answer 503 setup_required until this succeeds.
//	@Tags			setup
//	@Accept			json
//	@Produce		json
//	@Param			request	body		view.SetupRequest	true	"account name and bootstrap token"
//	@Success		201		{object}	view.SetupResult
//	@Failure		403		{object}	view.ErrorEnvelope	"invalid bootstrap token"
//	@Failure		409		{object}	view.ErrorEnvelope	"an administrator already exists"
//	@Failure		422		{object}	view.ErrorEnvelope
//	@Router			/svnadmin/api/v1/setup/admin [post]
func (a *API) setupAdmin(w http.ResponseWriter, r *http.Request) {
	var request view.SetupRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)

		return
	}

	result, err := a.setup.Admin(r.Context(), request.Username, request.Token)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusCreated, result)
}
