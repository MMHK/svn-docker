// Package webui serves the embedded single page application.
//
// The dist directory is produced by the rspack build (P0 frontend task) and is
// embedded into the Go binary so the runtime image needs neither Node nor a
// static file server. A placeholder index.html is committed so the Go build
// works before the frontend pipeline exists.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// mountPath is the URL prefix the SPA is served under. The edge proxy forwards
// only /svnadmin/*, and the frontend publicPath is /svnadmin/ (plan D10).
const mountPath = "/svnadmin"

// Handler returns the SPA handler. Unknown paths fall back to index.html so
// Vue Router history mode works (plan D10).
func Handler() (http.Handler, error) {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, err
	}

	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}

	return &spaHandler{files: dist, index: index}, nil
}

type spaHandler struct {
	files fs.FS
	index []byte
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	name, ok := distName(r.URL.Path)
	if !ok || isLegacyPHP(name) {
		http.NotFound(w, r)

		return
	}

	if name != "" {
		if stat, err := fs.Stat(h.files, name); err == nil && !stat.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}

			http.ServeFileFS(w, r, h.files, name)

			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(h.index)
}

// distName maps a request path to a name inside the embedded dist tree.
//
// The mount prefix is stripped first, so both /svnadmin/assets/x.js (the
// publicPath the frontend emits) and /assets/x.js resolve to the same file.
// /svnadmin itself maps to the SPA root. ok is false when the path tries to
// climb out of dist; the caller must not serve anything but a 404.
func distName(urlPath string) (name string, ok bool) {
	if hasDotDot(urlPath) {
		return "", false
	}

	cleaned := path.Clean("/" + urlPath)

	switch {
	case cleaned == mountPath:
		cleaned = "/"
	case strings.HasPrefix(cleaned, mountPath+"/"):
		cleaned = strings.TrimPrefix(cleaned, mountPath)
	}

	name = strings.TrimPrefix(cleaned, "/")
	if name == "" {
		return "", true
	}

	// Defence in depth: fs.ValidPath already rejects "..", absolute paths and
	// backslash weirdness, but state it explicitly so nothing escapes dist.
	if !fs.ValidPath(name) {
		return "", false
	}

	return name, true
}

// hasDotDot reports whether the path carries a ".." segment. path.Clean would
// otherwise silently fold /svnadmin/../x into /x, which is not a request the
// SPA should answer.
func hasDotDot(urlPath string) bool {
	for _, segment := range strings.Split(urlPath, "/") {
		if segment == ".." {
			return true
		}
	}

	return false
}

// isLegacyPHP reports whether a dist-relative name would have been served by
// the v2 PHP application. Plan D14 removes those routes without a redirect, so
// the history fallback must answer 404 instead of the SPA shell; otherwise
// /svnadmin/userlist.php would look like a working page again.
func isLegacyPHP(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".php")
}
