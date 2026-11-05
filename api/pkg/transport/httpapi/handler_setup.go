package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/view"
)

// setupStatus godoc
//
//	@Summary		Report the first-run setup state
//	@Description	Public diagnostic used by the login page. It reports whether the single administrator still has to be created, why (missing_auth_file, missing_access_file, no_users, no_admin), and whether the auth/access/admin files are usable. It never exposes absolute paths, password hashes, file contents or the administrator account name. The users list is only populated while setup is pending.
//	@Tags			setup
//	@Produce		json
//	@Success		200	{object}	view.SetupStatus
//	@Router			/svnadmin/api/v1/setup/status [get]
func (a *API) setupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.setup.Status(r.Context()))
}

// setupAdmin godoc
//
//	@Summary		Create the first administrator
//	@Description	Requires the one-time bootstrap token from SVN_ADMIN_BOOTSTRAP_TOKEN (or the token printed to stderr at startup). The chosen account is written to the single-line administrator file and becomes the only account allowed to sign in to the management API. When the account does not exist yet a password is required and the account is created; when it exists the password is ignored. On a fresh deployment the marker is overwritten; once the marker exists, moving the administrator is a manual file edit. Missing auth/access/admin files are created. Other API routes answer 503 setup_required until this succeeds.
//	@Tags			setup
//	@Accept			json
//	@Produce		json
//	@Param			request	body		view.SetupRequest	true	"account name, optional password and bootstrap token"
//	@Success		201		{object}	view.SetupResult
//	@Failure		403		{object}	view.ErrorEnvelope	"invalid bootstrap token"
//	@Failure		409		{object}	view.ErrorEnvelope	"an administrator already exists"
//	@Failure		422		{object}	view.ErrorEnvelope	"invalid account name or missing/invalid password"
//	@Failure		500		{object}	view.ErrorEnvelope	"the write failed; a freshly created account is rolled back"
//	@Failure		503		{object}	view.ErrorEnvelope	"storage_unavailable: the auth/access/admin directory is missing or not writable"
//	@Router			/svnadmin/api/v1/setup/admin [post]
func (a *API) setupAdmin(w http.ResponseWriter, r *http.Request) {
	var request view.SetupRequest
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)

		return
	}

	result, err := a.setup.Admin(r.Context(), request.Username, request.Password, request.Token)
	if err != nil {
		a.writeError(w, r, err)

		return
	}

	writeJSON(w, http.StatusCreated, result)
}
