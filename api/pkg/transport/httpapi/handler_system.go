package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/view"
)

// systemInfo godoc
//
//	@Summary		System information
//	@Description	Service version, Go runtime and the state of the configured SVN paths. File contents are never returned. Requires a valid bearer token.
//	@Tags			system
//	@Produce		json
//	@Security		bearerAuth
//	@Success		200	{object}	view.SystemInfo
//	@Failure		401	{object}	view.ErrorEnvelope
//	@Failure		403	{object}	view.ErrorEnvelope	"source address is not allowed"
//	@Failure		500	{object}	view.ErrorEnvelope
//	@Failure		503	{object}	view.ErrorEnvelope	"setup_required"
//	@Router			/svnadmin/api/v1/system/info [get]
func (a *API) systemInfo(w http.ResponseWriter, r *http.Request) {
	// The explicit view type keeps swag's type resolver anchored to the view
	// package when it parses the {object} annotation above.
	var info view.SystemInfo = a.system.Info(r.Context())
	writeJSON(w, http.StatusOK, info)
}
