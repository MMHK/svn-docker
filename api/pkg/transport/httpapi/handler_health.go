package httpapi

import (
	"net/http"

	"github.com/mmhk/svn-admin/pkg/view"
)

// healthz godoc
//
//	@Summary	Liveness probe
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	view.Health
//	@Router		/healthz [get]
func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, view.Health{Status: "ok"})
}

// readyz godoc
//
//	@Summary	Readiness probe
//	@Description	Checks that the SVN parent directory is usable and the auth/access files exist.
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	view.Readiness
//	@Failure	503	{object}	view.Readiness	"one or more checks failed"
//	@Router		/readyz [get]
func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	status := a.system.Ready(r.Context())

	code := http.StatusOK
	if status.Status != "ok" {
		code = http.StatusServiceUnavailable
	}

	writeJSON(w, code, status)
}
