package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	handler, err := Handler()
	if err != nil {
		t.Fatalf("Handler() error = %v", err)
	}

	return handler
}

func distFS(t *testing.T) fs.FS {
	t.Helper()

	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		t.Fatalf("fs.Sub(dist) error = %v", err)
	}

	return dist
}

// firstMatch resolves a real asset name from dist so the test never hardcodes
// a content hash.
func firstMatch(t *testing.T, pattern string) string {
	t.Helper()

	matches, err := fs.Glob(distFS(t), pattern)
	if err != nil {
		t.Fatalf("fs.Glob(%q) error = %v", pattern, err)
	}

	if len(matches) == 0 {
		t.Fatalf("fs.Glob(%q) found no files in dist", pattern)
	}

	return matches[0]
}

func serve(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

	return recorder
}

func TestSPAIndexUnderMountPath(t *testing.T) {
	handler := newTestHandler(t)

	for _, target := range []string{"/svnadmin", "/svnadmin/"} {
		recorder := serve(t, handler, target)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", target, recorder.Code)
		}

		if ct := recorder.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s Content-Type = %q, want text/html", target, ct)
		}

		if !strings.Contains(recorder.Body.String(), `id="app"`) {
			t.Errorf("%s did not return the SPA shell", target)
		}
	}
}

func TestSPAAssetsUnderMountPath(t *testing.T) {
	handler := newTestHandler(t)

	cases := []struct {
		pattern string
		want    string
	}{
		{pattern: "assets/*.js", want: "javascript"},
		{pattern: "assets/*.css", want: "text/css"},
	}

	for _, tc := range cases {
		asset := firstMatch(t, tc.pattern)

		recorder := serve(t, handler, "/svnadmin/"+asset)
		if recorder.Code != http.StatusOK {
			t.Errorf("/svnadmin/%s status = %d, want 200", asset, recorder.Code)
		}

		ct := recorder.Header().Get("Content-Type")
		if !strings.Contains(ct, tc.want) {
			t.Errorf("/svnadmin/%s Content-Type = %q, want it to contain %q", asset, ct, tc.want)
		}

		if got := recorder.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
			t.Errorf("/svnadmin/%s Cache-Control = %q, want immutable", asset, got)
		}
	}
}

func TestSPAAssetsWithoutMountPath(t *testing.T) {
	handler := newTestHandler(t)
	asset := firstMatch(t, "assets/*.js")

	recorder := serve(t, handler, "/"+asset)
	if recorder.Code != http.StatusOK {
		t.Errorf("/%s status = %d, want 200 (bare /assets compatibility)", asset, recorder.Code)
	}

	if ct := recorder.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("/%s Content-Type = %q, want javascript", asset, ct)
	}
}

func TestSPAHistoryFallbackUnderMountPath(t *testing.T) {
	handler := newTestHandler(t)

	for _, target := range []string{"/svnadmin/repositories", "/svnadmin/users/alice"} {
		recorder := serve(t, handler, target)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", target, recorder.Code)
		}

		if ct := recorder.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s Content-Type = %q, want text/html", target, ct)
		}

		if !strings.Contains(recorder.Body.String(), `id="app"`) {
			t.Errorf("%s did not return the SPA shell", target)
		}
	}
}

func TestSPARejectsPathTraversal(t *testing.T) {
	handler := newTestHandler(t)

	for _, target := range []string{
		"/svnadmin/../api/pkg/webui/embed.go",
		"/svnadmin/%2e%2e/api/pkg/webui/embed.go",
		"/svnadmin/../../../../etc/passwd",
	} {
		recorder := serve(t, handler, target)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", target, recorder.Code)
		}

		body := recorder.Body.String()
		if strings.Contains(body, "package webui") || strings.Contains(body, "func Handler") {
			t.Errorf("%s leaked Go source: %q", target, body)
		}
	}
}

// TestSPARejectsLegacyPHP covers plan D14 / T6.4: the v2 PHP routes are removed
// without a redirect, so the history fallback must not answer them with the SPA
// shell (which would look like a working page).
func TestSPARejectsLegacyPHP(t *testing.T) {
	handler := newTestHandler(t)

	for _, target := range []string{
		"/svnadmin/userlist.php",
		"/svnadmin/index.php",
		"/svnadmin/accesspathview.php?accesspath=test1%3A%2F",
		"/svnadmin/USERLIST.PHP",
	} {
		recorder := serve(t, handler, target)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", target, recorder.Code)
		}
	}
}
