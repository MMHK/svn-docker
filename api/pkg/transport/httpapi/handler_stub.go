package httpapi

import "net/http"

// notImplemented is the honest P0 answer for every management endpoint that
// has not been built yet. P1 replaces these paths with real handlers.
func (a *API) notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeEnvelope(
		w,
		http.StatusNotImplemented,
		"not_implemented",
		"endpoint is not implemented in P0; see the rewrite plan tasks P1-P4",
	)
}

// notFound is used for the docs subtree when SVN_ADMIN_ENABLE_DOCS=false.
func (a *API) notFound(w http.ResponseWriter, _ *http.Request) {
	writeEnvelope(w, http.StatusNotFound, "not_found", "not found")
}
