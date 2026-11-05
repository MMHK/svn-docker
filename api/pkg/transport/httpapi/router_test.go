package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/auth"
	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/presenter"
	"github.com/mmhk/svn-admin/pkg/store/adminfile"
	"github.com/mmhk/svn-admin/pkg/store/authz"
	"github.com/mmhk/svn-admin/pkg/store/htpasswd"
	"github.com/mmhk/svn-admin/pkg/store/svnrepos"
	"github.com/mmhk/svn-admin/pkg/svncli"
	"github.com/mmhk/svn-admin/pkg/view"
	"github.com/mmhk/svn-admin/pkg/webui"
)

// testPassword is the plaintext behind testHash (APR1 vector).
const (
	testPassword = "testpass"
	testHash     = "$apr1$abcdefgh$JDh3DOtFBWdMeBAh2S//z."
	bootstrap    = "test-bootstrap-token"
)

const defaultAccessFile = `# fixture
[groups]
devs = testuser, viewer

[repo1:/]
@devs = rw
testuser = r

[repo2:/trunk]
viewer = r
`

type testEnv struct {
	handler    http.Handler
	cfg        config.Config
	dir        string
	tokens     *auth.Manager
	bootstrap  string
	accessFile string
	runner     *fakeSvnRunner
	trashDir   string
}

func newTestEnv(t *testing.T, mutate func(*config.Config)) testEnv {
	t.Helper()

	return newTestEnvWithLogs(t, mutate, io.Discard)
}

// newTestEnvWithLogs is newTestEnv with a caller supplied log sink.
func newTestEnvWithLogs(t *testing.T, mutate func(*config.Config), logs io.Writer) testEnv {
	t.Helper()

	dir := t.TempDir()
	parent := filepath.Join(dir, "svn")

	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}

	makeRepository(t, parent, "repo1")
	makeRepository(t, parent, "repo2")

	if err := os.Mkdir(filepath.Join(parent, "notarepo"), 0o755); err != nil {
		t.Fatalf("mkdir notarepo: %v", err)
	}

	authFile := filepath.Join(dir, "svn-auth-file")
	authContent := "testuser:" + testHash + "\n" +
		"viewer:" + testHash + "\n" +
		"norole:" + testHash + "\n"

	if err := os.WriteFile(authFile, []byte(authContent), 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}

	accessFile := filepath.Join(dir, "svn-access-file")
	if err := os.WriteFile(accessFile, []byte(defaultAccessFile), 0o600); err != nil {
		t.Fatalf("write access file: %v", err)
	}

	cfg := config.Config{
		Addr:            ":0",
		SVNParentPath:   parent,
		SVNAuthFile:     authFile,
		SVNAccessFile:   accessFile,
		BackupDir:       filepath.Join(dir, "backup"),
		TrashDir:        filepath.Join(dir, "trash"),
		AdminFile:       filepath.Join(dir, "svnadmin-admin"),
		SVNBinary:       "svn",
		SVNAdminBinary:  "svnadmin",
		AllowCIDRs:      []string{"10.0.0.0/8"},
		TrustedProxies:  []string{"10.0.0.2"},
		UID:             1000,
		GID:             1000,
		TokenTTL:        12 * time.Hour,
		TokenIdle:       2 * time.Hour,
		LoginRatePerMin: 5,
		EnableDocs:      true,
		RepoDelete:      true,
		DefaultLocale:   config.LocaleZhCN,
		ShutdownTimeout: 10 * time.Second,
	}

	if mutate != nil {
		mutate(&cfg)
	}

	// Every deployment under test has a single administrator. A test that
	// wants the marker absent points AdminFile at a path it controls during
	// mutate and leaves the file out.
	if _, err := os.Stat(cfg.AdminFile); os.IsNotExist(err) {
		if err := os.WriteFile(cfg.AdminFile, []byte("testuser\n"), 0o600); err != nil {
			t.Fatalf("write admin file: %v", err)
		}
	}

	allowlist, err := NewAllowlist(cfg.AllowCIDRs, cfg.TrustedProxies)
	if err != nil {
		t.Fatalf("NewAllowlist() error = %v", err)
	}

	web, err := webui.Handler()
	if err != nil {
		t.Fatalf("webui.Handler() error = %v", err)
	}

	users := htpasswd.New(cfg.SVNAuthFile)
	groups := authz.New(cfg.SVNAccessFile)
	admins := adminfile.New(cfg.AdminFile)
	runner := newFakeSvnRunner()
	repoStore := svnrepos.NewStore(svnrepos.Options{
		Parents:  []string{cfg.SVNParentPath},
		CLI:      svncli.New(svncli.Options{Runner: runner, SVN: "svn", SVNAdmin: "svnadmin"}),
		TrashDir: cfg.TrashDir,
		Now:      func() time.Time { return fakeSvnNow },
	})

	tokens := auth.NewManager(auth.Options{TTL: cfg.TokenTTL, Idle: cfg.TokenIdle})
	limiter := auth.NewRateLimiter(cfg.LoginRatePerMin, time.Minute)

	handler, err := New(Options{
		Config:    cfg,
		Logger:    slog.New(slog.NewJSONHandler(logs, nil)),
		Allowlist: allowlist,
		Tokens:    tokens,
		System:    presenter.NewSystem("3.0.0-test", cfg),
		Session:   presenter.NewSession(users, admins, tokens, limiter),
		Setup: presenter.NewSetup(presenter.SetupDeps{
			Users:          users,
			Accounts:       users,
			Admins:         admins,
			AuthFile:       cfg.SVNAuthFile,
			AccessFile:     cfg.SVNAccessFile,
			AdminFile:      cfg.AdminFile,
			BootstrapToken: bootstrap,
		}),
		User: presenter.NewUser(presenter.UserDeps{
			Users:       users,
			Accounts:    users,
			Groups:      groups,
			Authz:       groups,
			AccessPaths: groups,
			Tokens:      tokens,
		}),
		Group: presenter.NewGroup(groups, groups, groups),
		Repo: presenter.NewRepository(presenter.RepositoryDeps{
			Repos:         repoStore,
			Writer:        repoStore,
			Browser:       repoStore,
			AccessPaths:   groups,
			DeleteEnabled: cfg.RepoDelete,
		}),
		AccessPath: presenter.NewAccessPath(
			groups,
			groups,
		),
		Web: web,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return testEnv{
		handler:    handler,
		cfg:        cfg,
		dir:        dir,
		tokens:     tokens,
		bootstrap:  bootstrap,
		accessFile: cfg.SVNAccessFile,
		runner:     runner,
		trashDir:   cfg.TrashDir,
	}
}

func makeRepository(t *testing.T, parent, name string) {
	t.Helper()

	path := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(path, "db"), 0o755); err != nil {
		t.Fatalf("mkdir db: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(path, "conf"), 0o755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}

	if err := os.WriteFile(filepath.Join(path, "format"), []byte("5\n"), 0o644); err != nil {
		t.Fatalf("write format: %v", err)
	}
}

func (env testEnv) request(
	t *testing.T,
	method, path, remoteAddr string,
	headers map[string]string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.RemoteAddr = remoteAddr

	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	env.handler.ServeHTTP(recorder, request)

	return recorder
}

func (env testEnv) login(t *testing.T, username, password string) *httptest.ResponseRecorder {
	t.Helper()

	body := `{"username":"` + username + `","password":"` + password + `"}`

	return env.request(t, http.MethodPost, "/svnadmin/api/v1/session", "127.0.0.1:5555", nil, body)
}

func (env testEnv) mustToken(t *testing.T, username, password string) string {
	t.Helper()

	recorder := env.login(t, username, password)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login(%s) status = %d, body = %s", username, recorder.Code, recorder.Body.String())
	}

	return decodeBody[view.Session](t, recorder).Token
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func decodeBody[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}

	return payload
}

func assertNoSetCookie(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if cookie := recorder.Header().Get("Set-Cookie"); cookie != "" {
		t.Errorf("response contains Set-Cookie: %q", cookie)
	}
}

func TestLoginIssuesTokenWithoutCookie(t *testing.T) {
	env := newTestEnv(t, nil)

	recorder := env.login(t, "testuser", testPassword)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	assertNoSetCookie(t, recorder)

	session := decodeBody[view.Session](t, recorder)

	if !strings.HasPrefix(session.Token, auth.TokenPrefix) {
		t.Errorf("token = %q, want the svna_ prefix", session.Token)
	}

	if session.Me.Name != "testuser" {
		t.Errorf("me = %+v", session.Me)
	}

	if session.ExpiresAt.IsZero() || session.IdleExpiresAt.IsZero() {
		t.Errorf("expiry fields = %+v", session)
	}

	if !strings.Contains(recorder.Header().Get("Cache-Control"), "no-store") {
		t.Errorf("Cache-Control = %q", recorder.Header().Get("Cache-Control"))
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	env := newTestEnv(t, nil)

	recorder := env.login(t, "testuser", "wrong")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "unauthorized" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}

	if strings.Contains(recorder.Body.String(), testHash) || strings.Contains(recorder.Body.String(), "apr1") {
		t.Error("error response leaks the password hash")
	}

	assertNoSetCookie(t, recorder)
}

func TestLoginRateLimit(t *testing.T) {
	env := newTestEnv(t, nil)

	for i := 0; i < 5; i++ {
		if recorder := env.login(t, "testuser", "wrong"); recorder.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, recorder.Code)
		}
	}

	recorder := env.login(t, "testuser", "wrong")
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 (body %s)", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "rate_limited" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}

	if recorder.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestUsersRequiresToken(t *testing.T) {
	env := newTestEnv(t, nil)

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", nil, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body %s)", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "unauthorized" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}
}

func TestUsersWithToken(t *testing.T) {
	env := newTestEnv(t, nil)
	token := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(token), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	list := decodeBody[view.UserList](t, recorder)
	if list.Total != 3 || len(list.Items) != 3 {
		t.Fatalf("list = %+v", list)
	}

	if strings.Contains(recorder.Body.String(), "apr1") || strings.Contains(recorder.Body.String(), testHash) {
		t.Error("user list leaks a password hash")
	}

}

func TestUsersPagingAndFilter(t *testing.T) {
	env := newTestEnv(t, nil)
	token := env.mustToken(t, "testuser", testPassword)

	paged := env.request(t, http.MethodGet, "/svnadmin/api/v1/users?size=1&page=2", "127.0.0.1:5555", bearer(token), "")
	if paged.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", paged.Code, paged.Body.String())
	}

	list := decodeBody[view.UserList](t, paged)
	if list.Total != 3 || len(list.Items) != 1 || list.Size != 1 || list.Page != 2 {
		t.Errorf("paged list = %+v", list)
	}

	filtered := env.request(t, http.MethodGet, "/svnadmin/api/v1/users?q=view", "127.0.0.1:5555", bearer(token), "")
	if filtered.Code != http.StatusOK {
		t.Fatalf("status = %d", filtered.Code)
	}

	filteredList := decodeBody[view.UserList](t, filtered)
	if filteredList.Total != 1 || filteredList.Items[0].Name != "viewer" {
		t.Errorf("filtered list = %+v", filteredList)
	}

	invalid := env.request(t, http.MethodGet, "/svnadmin/api/v1/users?page=nope", "127.0.0.1:5555", bearer(token), "")
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid page status = %d, want 422", invalid.Code)
	}
}

func TestSessionEndpoints(t *testing.T) {
	env := newTestEnv(t, nil)
	token := env.mustToken(t, "testuser", testPassword)

	current := env.request(t, http.MethodGet, "/svnadmin/api/v1/session", "127.0.0.1:5555", bearer(token), "")
	if current.Code != http.StatusOK {
		t.Fatalf("GET /session status = %d, body = %s", current.Code, current.Body.String())
	}

	session := decodeBody[view.Session](t, current)
	if session.Token != "" {
		t.Errorf("GET /session returned the token: %q", session.Token)
	}

	if session.Me.Name != "testuser" {
		t.Errorf("me = %+v", session.Me)
	}

	refresh := env.request(t, http.MethodPost, "/svnadmin/api/v1/session/refresh", "127.0.0.1:5555", bearer(token), "")
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", refresh.Code, refresh.Body.String())
	}

	rotated := decodeBody[view.Session](t, refresh)
	if rotated.Token == "" || rotated.Token == token {
		t.Fatalf("refresh token = %q", rotated.Token)
	}

	old := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(token), "")
	if old.Code != http.StatusUnauthorized {
		t.Errorf("old token status = %d, want 401", old.Code)
	}

	fresh := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(rotated.Token), "")
	if fresh.Code != http.StatusOK {
		t.Errorf("new token status = %d, want 200", fresh.Code)
	}

	logout := env.request(t, http.MethodDelete, "/svnadmin/api/v1/session", "127.0.0.1:5555", bearer(rotated.Token), "")
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logout.Code, logout.Body.String())
	}

	revoked := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(rotated.Token), "")
	if revoked.Code != http.StatusUnauthorized {
		t.Errorf("revoked token status = %d, want 401", revoked.Code)
	}
}

func TestSetupRequiredAndBootstrap(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		// Point to an empty auth file so the setup gate stays closed.
		cfg.SVNAuthFile = filepath.Join(cfg.SVNParentPath, "empty-auth")
	})

	blocked := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", nil, "")
	if blocked.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", blocked.Code, blocked.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, blocked)
	if envelope.Error.Code != "setup_required" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}

	// Bootstrap creates the first account.
	ok := env.request(t, http.MethodPost, "/svnadmin/api/v1/setup/admin", "127.0.0.1:5555", nil,
		`{"username":"admin","password":"Passw0rd!","token":"`+env.bootstrap+`"}`)
	if ok.Code != http.StatusCreated {
		t.Fatalf("setup status = %d, body = %s", ok.Code, ok.Body.String())
	}

	// Now login and verify the setup gate is open.
	token := env.mustToken(t, "admin", "Passw0rd!")

	if recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(token), ""); recorder.Code != http.StatusOK {
		t.Fatalf("after setup status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	again := env.request(t, http.MethodPost, "/svnadmin/api/v1/setup/admin", "127.0.0.1:5555", nil,
		`{"username":"testuser","token":"`+env.bootstrap+`"}`)
	if again.Code != http.StatusConflict {
		t.Errorf("second setup status = %d, want 409", again.Code)
	}
}

func TestReadEndpoints(t *testing.T) {
	env := newTestEnv(t, nil)
	token := env.mustToken(t, "testuser", testPassword)
	headers := bearer(token)

	t.Run("system info", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/system/info", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		info := decodeBody[view.SystemInfo](t, recorder)
		if info.Version != "3.0.0-test" {
			t.Errorf("version = %q", info.Version)
		}
	})

	t.Run("user detail", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/viewer", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		user := decodeBody[view.User](t, recorder)
		if user.Name != "viewer" || len(user.Groups) != 1 || user.Groups[0] != "devs" {
			t.Errorf("user = %+v", user)
		}

		missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/nobody", "127.0.0.1:5555", headers, "")
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing user status = %d, want 404", missing.Code)
		}
	})

	t.Run("groups", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d", recorder.Code)
		}

		list := decodeBody[view.GroupList](t, recorder)
		if list.Total != 1 || list.Items[0].Name != "devs" || len(list.Items[0].Members) != 2 {
			t.Errorf("groups = %+v", list)
		}

		detail := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups/devs", "127.0.0.1:5555", headers, "")
		if detail.Code != http.StatusOK {
			t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
		}

		group := decodeBody[view.GroupDetail](t, detail)
		if len(group.AccessPaths) != 1 || group.AccessPaths[0] != "repo1:/" {
			t.Errorf("group access paths = %v", group.AccessPaths)
		}

		missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups/nope", "127.0.0.1:5555", headers, "")
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing group status = %d, want 404", missing.Code)
		}
	})

	t.Run("repositories", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		list := decodeBody[view.RepositoryList](t, recorder)
		if list.Total != 2 {
			t.Fatalf("repos = %+v (notarepo must not be listed)", list)
		}

		for _, repository := range list.Items {
			if repository.Name == "notarepo" {
				t.Error("notarepo was listed as a repository")
			}

			if !repository.IsRepository || repository.Parent == "" {
				t.Errorf("repository = %+v", repository)
			}
		}
	})

	t.Run("access paths", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/access-paths", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		list := decodeBody[view.AccessPathList](t, recorder)
		if list.Total != 2 {
			t.Fatalf("access paths = %+v", list)
		}

		encoded := base64.RawURLEncoding.EncodeToString([]byte("repo2:/trunk"))
		acl := env.request(t, http.MethodGet, "/svnadmin/api/v1/access-paths/"+encoded+"/acl", "127.0.0.1:5555", headers, "")
		if acl.Code != http.StatusOK {
			t.Fatalf("acl status = %d, body = %s", acl.Code, acl.Body.String())
		}

		accessPath := decodeBody[view.AccessPath](t, acl)
		if accessPath.Path != "repo2:/trunk" || len(accessPath.ACL) != 1 || accessPath.ACL[0].Principal != "viewer" {
			t.Errorf("access path = %+v", accessPath)
		}

		query := env.request(t, http.MethodGet, "/svnadmin/api/v1/access-paths/x/acl?path=repo1:/", "127.0.0.1:5555", headers, "")
		if query.Code != http.StatusOK {
			t.Errorf("?path= status = %d, body = %s", query.Code, query.Body.String())
		}

		missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/access-paths/"+base64.RawURLEncoding.EncodeToString([]byte("missing:/x"))+"/acl", "127.0.0.1:5555", headers, "")
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing access path status = %d, want 404", missing.Code)
		}
	})

	t.Run("roles removed", func(t *testing.T) {
		recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/roles", "127.0.0.1:5555", headers, "")
		if recorder.Code != http.StatusNotImplemented {
			t.Errorf("GET /roles = %d, want 501", recorder.Code)
		}
	})
}

func TestUnknownAPIPathIsNotImplemented(t *testing.T) {
	env := newTestEnv(t, nil)
	token := env.mustToken(t, "testuser", testPassword)

	// Unknown /svnadmin/api paths must stay an honest 501 rather than falling
	// through to the SPA.
	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/nope", "127.0.0.1:5555", bearer(token), "{}")
	if recorder.Code != http.StatusNotImplemented {
		t.Errorf("POST /svnadmin/api/v1/nope status = %d, want 501", recorder.Code)
	}
}

func TestDocsServedWithoutToken(t *testing.T) {
	env := newTestEnv(t, nil)

	index := env.request(t, http.MethodGet, "/svnadmin/api/docs/index.html", "127.0.0.1:5555", nil, "")
	if index.Code != http.StatusOK {
		t.Fatalf("index status = %d, body = %s", index.Code, index.Body.String())
	}

	if !strings.Contains(index.Body.String(), "SwaggerUIBundle") {
		t.Error("index.html does not look like Swagger UI")
	}

	spec := env.request(t, http.MethodGet, "/svnadmin/api/docs/openapi.json", "127.0.0.1:5555", nil, "")
	if spec.Code != http.StatusOK {
		t.Fatalf("openapi status = %d", spec.Code)
	}

	var document map[string]any
	if err := json.Unmarshal(spec.Body.Bytes(), &document); err != nil {
		t.Fatalf("openapi.json is not JSON: %v", err)
	}

	definitions, _ := document["securityDefinitions"].(map[string]any)
	if _, ok := definitions["bearerAuth"]; !ok {
		t.Error("openapi.json is missing the bearerAuth security definition")
	}

	paths, _ := document["paths"].(map[string]any)

	for _, path := range []string{
		"/svnadmin/api/v1/session",
		"/svnadmin/api/v1/session/refresh",
		"/svnadmin/api/v1/setup/admin",
		"/svnadmin/api/v1/setup/status",
		"/svnadmin/api/v1/system/info",
		"/svnadmin/api/v1/users",
		"/svnadmin/api/v1/users/{name}",
		"/svnadmin/api/v1/users/{name}/password",
		"/svnadmin/api/v1/users/{name}/groups",
		"/svnadmin/api/v1/users/{name}/access-paths",
		"/svnadmin/api/v1/groups",
		"/svnadmin/api/v1/groups/{name}",
		"/svnadmin/api/v1/groups/{name}/members",
		"/svnadmin/api/v1/groups/{name}/access-paths",
		"/svnadmin/api/v1/repos",
		"/svnadmin/api/v1/repos/{repo}",
		"/svnadmin/api/v1/repos/{repo}/tree",
		"/svnadmin/api/v1/repos/{repo}/info",
		"/svnadmin/api/v1/access-paths",
		"/svnadmin/api/v1/access-paths/{path}",
		"/svnadmin/api/v1/access-paths/{path}/acl",
		"/svnadmin/api/v1/access-paths/{path}/acl/{principal}",
	} {
		if _, ok := paths[path]; !ok {
			t.Errorf("openapi.json is missing %s", path)
		}
	}
}

func TestDocsDisabled(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		cfg.EnableDocs = false
	})

	for _, path := range []string{
		"/svnadmin/api/docs",
		"/svnadmin/api/docs/",
		"/svnadmin/api/docs/index.html",
		"/svnadmin/api/docs/openapi.json",
	} {
		recorder := env.request(t, http.MethodGet, path, "127.0.0.1:5555", nil, "")
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, recorder.Code)
		}
	}
}

func TestHealthzAndReadyzStayPublic(t *testing.T) {
	env := newTestEnv(t, nil)

	health := env.request(t, http.MethodGet, "/healthz", "127.0.0.1:5555", nil, "")
	if health.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", health.Code)
	}

	if got := decodeBody[view.Health](t, health); got.Status != "ok" {
		t.Errorf("health payload = %q", got.Status)
	}

	ready := env.request(t, http.MethodGet, "/readyz", "127.0.0.1:5555", nil, "")
	if ready.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, body = %s", ready.Code, ready.Body.String())
	}

	header := health.Header()
	if header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff header")
	}

	if header.Get("Referrer-Policy") != "no-referrer" {
		t.Error("missing Referrer-Policy header")
	}

	if header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", header.Get("Cache-Control"))
	}
}

func TestAllowlistRejectsBeforeAuth(t *testing.T) {
	env := newTestEnv(t, nil)

	// The peer is outside the allowlist and is not a trusted proxy, so the
	// forged X-Forwarded-For header must be ignored.
	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "203.0.113.9:1234", map[string]string{
		"X-Forwarded-For": "10.1.2.3",
	}, "")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "forbidden" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}
}

func TestAllowlistTrustedProxy(t *testing.T) {
	env := newTestEnv(t, nil)

	allowed := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "10.0.0.2:1234", map[string]string{
		"X-Forwarded-For": "10.5.5.5",
	}, "")
	if allowed.Code != http.StatusUnauthorized {
		t.Fatalf("trusted XFF status = %d, want 401 (auth still required)", allowed.Code)
	}

	denied := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "10.0.0.2:1234", map[string]string{
		"X-Forwarded-For": "192.168.1.50",
	}, "")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("untrusted XFF status = %d, want 403", denied.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	env := newTestEnv(t, nil)

	for _, path := range []string{"/", "/repos", "/users/alice"} {
		recorder := env.request(t, http.MethodGet, path, "127.0.0.1:5555", nil, "")
		if recorder.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", path, recorder.Code)
		}

		if !strings.Contains(recorder.Body.String(), `id="app"`) {
			t.Errorf("%s did not return the SPA shell", path)
		}
	}
}

var generatedRequestID = regexp.MustCompile(`^[0-9a-f]{16}$`)

func TestRequestID(t *testing.T) {
	env := newTestEnv(t, nil)

	reused := env.request(t, http.MethodGet, "/healthz", "127.0.0.1:5555", map[string]string{
		"X-Request-Id": "abcdefgh",
	}, "")
	if got := reused.Header().Get("X-Request-Id"); got != "abcdefgh" {
		t.Errorf("request id = %q, want abcdefgh", got)
	}

	generated := env.request(t, http.MethodGet, "/healthz", "127.0.0.1:5555", map[string]string{
		"X-Request-Id": "short",
	}, "")
	if got := generated.Header().Get("X-Request-Id"); !generatedRequestID.MatchString(got) {
		t.Errorf("generated request id = %q, want 16 hex chars", got)
	}
}

func TestAccessLogShapeAndNoSecrets(t *testing.T) {
	var logs bytes.Buffer

	dir := t.TempDir()
	parent := filepath.Join(dir, "svn")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}

	authFile := filepath.Join(dir, "svn-auth-file")
	if err := os.WriteFile(authFile, []byte("testuser:"+testHash+"\n"), 0o600); err != nil {
		t.Fatalf("write auth file: %v", err)
	}

	accessFile := filepath.Join(dir, "svn-access-file")
	if err := os.WriteFile(accessFile, []byte("[repo1:/]\n"), 0o600); err != nil {
		t.Fatalf("write access file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "svnadmin-admin"), []byte("testuser\n"), 0o600); err != nil {
		t.Fatalf("write admin file: %v", err)
	}

	cfg := config.Config{
		SVNParentPath:   parent,
		SVNAuthFile:     authFile,
		SVNAccessFile:   accessFile,
		TrashDir:        filepath.Join(dir, "trash"),
		AdminFile:       filepath.Join(dir, "svnadmin-admin"),
		SVNBinary:       "svn",
		SVNAdminBinary:  "svnadmin",
		AllowCIDRs:      []string{"10.0.0.0/8"},
		TrustedProxies:  []string{"10.0.0.2"},
		TokenTTL:        12 * time.Hour,
		TokenIdle:       2 * time.Hour,
		LoginRatePerMin: 5,
		EnableDocs:      true,
		RepoDelete:      true,
		DefaultLocale:   config.LocaleZhCN,
	}

	allowlist, err := NewAllowlist(cfg.AllowCIDRs, cfg.TrustedProxies)
	if err != nil {
		t.Fatalf("NewAllowlist() error = %v", err)
	}

	users := htpasswd.New(cfg.SVNAuthFile)
	groups := authz.New(cfg.SVNAccessFile)
	admins := adminfile.New(cfg.AdminFile)
	repoStore := svnrepos.NewStore(svnrepos.Options{
		Parents:  []string{cfg.SVNParentPath},
		CLI:      svncli.New(svncli.Options{Runner: newFakeSvnRunner()}),
		TrashDir: cfg.TrashDir,
	})

	tokens := auth.NewManager(auth.Options{TTL: cfg.TokenTTL, Idle: cfg.TokenIdle})

	handler, err := New(Options{
		Config:    cfg,
		Logger:    slog.New(slog.NewJSONHandler(&logs, nil)),
		Allowlist: allowlist,
		Tokens:    tokens,
		System:    presenter.NewSystem("3.0.0-test", cfg),
		Session:   presenter.NewSession(users, admins, tokens, auth.NewRateLimiter(5, time.Minute)),
		Setup: presenter.NewSetup(presenter.SetupDeps{
			Users:          users,
			Accounts:       users,
			Admins:         admins,
			AuthFile:       cfg.SVNAuthFile,
			AccessFile:     cfg.SVNAccessFile,
			AdminFile:      cfg.AdminFile,
			BootstrapToken: bootstrap,
		}),
		User: presenter.NewUser(presenter.UserDeps{
			Users:       users,
			Accounts:    users,
			Groups:      groups,
			Authz:       groups,
			AccessPaths: groups,
			Tokens:      tokens,
		}),
		Group: presenter.NewGroup(groups, groups, groups),
		Repo: presenter.NewRepository(presenter.RepositoryDeps{
			Repos:         repoStore,
			Writer:        repoStore,
			Browser:       repoStore,
			AccessPaths:   groups,
			DeleteEnabled: cfg.RepoDelete,
		}),
		AccessPath: presenter.NewAccessPath(
			groups,
			groups,
		),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/svnadmin/api/v1/users", nil)
	request.RemoteAddr = "127.0.0.1:5555"
	request.Header.Set("Authorization", "Bearer super-secret-token")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	output := logs.String()
	if strings.Contains(output, "super-secret-token") {
		t.Errorf("access log leaked the Authorization header: %s", output)
	}

	if strings.Contains(output, "Secret") || strings.Contains(output, "password") {
		t.Errorf("access log contains sensitive words: %s", output)
	}

	for _, field := range []string{`"method":"GET"`, `"path":"/svnadmin/api/v1/users"`, `"status":401`, `"duration":`, `"ip":"127.0.0.1"`, `"request_id":`} {
		if !strings.Contains(output, field) {
			t.Errorf("access log is missing %s: %s", field, output)
		}
	}
}
