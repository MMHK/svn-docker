package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/svncli"
	"github.com/mmhk/svn-admin/pkg/view"
)

// fakeSvnNow is the fixed clock behind the trash suffix so the assertions do
// not depend on the wall clock.
var fakeSvnNow = time.Date(2026, time.November, 4, 12, 0, 0, 0, time.UTC)

const fakeListXML = `<?xml version="1.0" encoding="UTF-8"?>
<lists>
<list path="file:///svn/repo1">
<entry kind="dir">
<name>trunk</name>
<commit revision="4">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
<entry kind="file">
<name>README &amp; notes.txt</name>
<size>1024</size>
<commit revision="3">
<author>viewer</author>
<date>2026-11-03T09:30:00.000000Z</date>
</commit>
</entry>
</list>
</lists>`

const fakeInfoXML = `<?xml version="1.0" encoding="UTF-8"?>
<info>
<entry kind="dir" path="." revision="4">
<url>file:///svn/repo1</url>
<repository>
<root>file:///svn/repo1</root>
<uuid>0f2d6c3a-0000-0000-0000-000000000000</uuid>
</repository>
<commit revision="4">
<author>mixmedia</author>
<date>2026-11-04T10:00:00.000000Z</date>
</commit>
</entry>
</info>`

const fakeLogXML = `<?xml version="1.0" encoding="UTF-8"?>
<log>
<logentry revision="7">
<author>mixmedia</author>
<date>2026-11-04T11:00:00.000000Z</date>
</logentry>
</log>`

// fakeCall is one recorded invocation.
type fakeCall struct {
	Name string
	Args []string
}

// fakeSvnRunner records argv and answers list/info/log with fixtures. A
// `svnadmin create` also materialises the directory so the store's post-conditions
// look like a real run; the returned argv is what the tests assert on.
type fakeSvnRunner struct {
	mu      sync.Mutex
	calls   []fakeCall
	listXML string
	infoXML string
	logXML  string
	fail    map[string]error
	stderr  string
}

func newFakeSvnRunner() *fakeSvnRunner {
	return &fakeSvnRunner{
		listXML: fakeListXML,
		infoXML: fakeInfoXML,
		logXML:  fakeLogXML,
		fail:    map[string]error{},
	}
}

func (f *fakeSvnRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.mu.Lock()
	f.calls = append(f.calls, fakeCall{Name: name, Args: append([]string(nil), args...)})
	failErr := f.fail[operation(args)]
	stderr := f.stderr
	listXML, infoXML, logXML := f.listXML, f.infoXML, f.logXML
	f.mu.Unlock()

	if failErr != nil {
		return nil, []byte(stderr), failErr
	}

	switch name {
	case "svnadmin":
		path := args[len(args)-1]
		for _, directory := range []string{"db", "conf"} {
			if err := os.MkdirAll(filepath.Join(path, directory), 0o755); err != nil {
				return nil, nil, err
			}
		}

		if err := os.WriteFile(filepath.Join(path, "format"), []byte("5\n"), 0o644); err != nil {
			return nil, nil, err
		}

		return nil, nil, nil
	case "svn":
		switch operation(args) {
		case "list":
			return []byte(listXML), nil, nil
		case "info":
			return []byte(infoXML), nil, nil
		case "log":
			return []byte(logXML), nil, nil
		case "mkdir":
			return nil, nil, nil
		}
	}

	return nil, nil, fmt.Errorf("fake runner: unexpected command %q %v", name, args)
}

func operation(args []string) string {
	if len(args) == 0 {
		return ""
	}

	return args[0]
}

func (f *fakeSvnRunner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func (f *fakeSvnRunner) snapshot() []fakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]fakeCall, len(f.calls))
	copy(out, f.calls)

	return out
}

func (f *fakeSvnRunner) set(fn func(*fakeSvnRunner)) {
	f.mu.Lock()
	defer f.mu.Unlock()

	fn(f)
}

func TestRepositoryNameInjectionIsRejected(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	names := []string{
		"../etc",
		"a b",
		"$(id)",
		"line\nbreak",
		".hidden",
		"repo/child",
		"repo\\child",
		"repo\x00name",
		"",
		strings.Repeat("a", 65),
	}

	for _, name := range names {
		body, err := json.Marshal(map[string]string{"name": name})
		if err != nil {
			t.Fatalf("marshal %q: %v", name, err)
		}

		before := env.runner.count()

		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
			bearer(admin), string(body))
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Fatalf("name %q status = %d, want 422 (body %s)", name, recorder.Code, recorder.Body.String())
		}

		if after := env.runner.count(); after != before {
			t.Errorf("name %q invoked svn/svnadmin %d times; validation must run first", name, after-before)
		}

		t.Logf("POST /repos {\"name\":%q} -> %d %s", name, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
}

func TestRepositoryPathInjectionIsRejected(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	paths := []string{
		"../etc",
		"..",
		"trunk/../../etc",
		"/etc/passwd",
		"C:/Windows",
		"back\\slash",
		"nul\x00byte",
		"line\nbreak",
		"trunk//nested",
	}

	for _, path := range paths {
		before := env.runner.count()

		recorder := env.request(
			t,
			http.MethodGet,
			"/svnadmin/api/v1/repos/repo1/tree?path="+url.QueryEscape(path),
			"127.0.0.1:5555",
			bearer(admin),
			"",
		)
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Fatalf("path %q status = %d, want 422 (body %s)", path, recorder.Code, recorder.Body.String())
		}

		if after := env.runner.count(); after != before {
			t.Errorf("path %q invoked svn %d times; validation must run first", path, after-before)
		}

		t.Logf("GET /repos/repo1/tree?path=%q -> %d %s", path, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
}

func TestCreateRepository(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
		bearer(admin), `{"name":"newrepo","structure":"single","accessPath":true}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	created := decodeBody[view.RepositoryCreated](t, recorder)
	if created.Name != "newrepo" || created.Parent != env.cfg.SVNParentPath || created.CreatedStructure != "single" {
		t.Errorf("created = %+v", created)
	}

	if !strings.HasPrefix(created.URL, "file://") {
		t.Errorf("url = %q, want a file:// URL", created.URL)
	}

	location := recorder.Header().Get("Location")
	if location != "/svnadmin/api/v1/repos/newrepo/info" {
		t.Errorf("Location = %q", location)
	}

	// The repository exists on disk and is recognised by the P1 detection rule.
	if _, err := os.Stat(filepath.Join(env.cfg.SVNParentPath, "newrepo", "format")); err != nil {
		t.Errorf("format file missing: %v", err)
	}

	if !strings.Contains(readAccess(t, env), "newrepo:/") {
		t.Error("the optional access-path section was not created")
	}

	// The CLI was invoked with argv arrays, never a shell string.
	calls := env.runner.snapshot()
	if len(calls) != 2 {
		t.Fatalf("calls = %+v, want svnadmin create + svn mkdir", calls)
	}

	for _, call := range calls {
		t.Logf("argv: %s %s", call.Name, strings.Join(call.Args, " "))
	}

	if calls[0].Name != "svnadmin" {
		t.Errorf("first call = %q, want svnadmin", calls[0].Name)
	}

	wantCreate := []string{"create", "--fs-type", "fsfs", filepath.Join(env.cfg.SVNParentPath, "newrepo")}
	if strings.Join(calls[0].Args, "\x00") != strings.Join(wantCreate, "\x00") {
		t.Errorf("svnadmin args = %v, want %v", calls[0].Args, wantCreate)
	}

	if calls[1].Name != "svn" || operation(calls[1].Args) != "mkdir" {
		t.Fatalf("second call = %+v, want svn mkdir", calls[1])
	}

	base := created.URL
	wantDirs := []string{base + "/trunk", base + "/branches", base + "/tags"}
	for i, want := range wantDirs {
		got := calls[1].Args[len(calls[1].Args)-len(wantDirs)+i]
		if got != want {
			t.Errorf("mkdir arg %d = %q, want %q", i, got, want)
		}
	}

	if !strings.Contains(strings.Join(calls[1].Args, " "), "--parents") ||
		!strings.Contains(strings.Join(calls[1].Args, " "), "-m") {
		t.Errorf("mkdir args lack --parents/-m: %v", calls[1].Args)
	}

	// The new repository is listed by GET /repos.
	list := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos", "127.0.0.1:5555", bearer(admin), "")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}

	found := false

	for _, item := range decodeBody[view.RepositoryList](t, list).Items {
		if item.Name == "newrepo" {
			found = true
		}
	}

	if !found {
		t.Error("newrepo is not listed")
	}

	// A second creation is a conflict, not an overwrite.
	duplicate := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
		bearer(admin), `{"name":"newrepo"}`)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate status = %d, want 409 (body %s)", duplicate.Code, duplicate.Body.String())
	}
}

func TestCreateRepositoryStructureRules(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	cases := []struct {
		body   string
		status int
	}{
		{`{"name":"n1","structure":"multi"}`, http.StatusUnprocessableEntity},
		{`{"name":"n2","structure":"multi","projectName":"bad name"}`, http.StatusUnprocessableEntity},
		{`{"name":"n3","structure":"single","projectName":"proj"}`, http.StatusUnprocessableEntity},
		{`{"name":"n4","structure":"weird"}`, http.StatusUnprocessableEntity},
	}

	for _, test := range cases {
		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
			bearer(admin), test.body)
		if recorder.Code != test.status {
			t.Errorf("body %s status = %d, want %d (body %s)", test.body, recorder.Code, test.status, recorder.Body.String())
		}
	}

	multi := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
		bearer(admin), `{"name":"multi","structure":"multi","projectName":"proj"}`)
	if multi.Code != http.StatusCreated {
		t.Fatalf("multi status = %d, body = %s", multi.Code, multi.Body.String())
	}

	calls := env.runner.snapshot()
	mkdir := calls[len(calls)-1]
	if operation(mkdir.Args) != "mkdir" {
		t.Fatalf("last call = %+v", mkdir)
	}

	joined := strings.Join(mkdir.Args, " ")
	for _, directory := range []string{"proj/trunk", "proj/branches", "proj/tags"} {
		if !strings.Contains(joined, directory) {
			t.Errorf("mkdir args lack %q: %v", directory, mkdir.Args)
		}
	}
}

func TestDeleteRepository(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	makeRepository(t, env.cfg.SVNParentPath, "gone")

	missingConfirm := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/gone", "127.0.0.1:5555", bearer(admin), "")
	if missingConfirm.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing confirm status = %d, want 422", missingConfirm.Code)
	}

	wrongConfirm := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/gone?confirm=other", "127.0.0.1:5555", bearer(admin), "")
	if wrongConfirm.Code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong confirm status = %d, want 422", wrongConfirm.Code)
	}

	deleted := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/gone?confirm=gone", "127.0.0.1:5555", bearer(admin), "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", deleted.Code, deleted.Body.String())
	}

	if _, err := os.Stat(filepath.Join(env.cfg.SVNParentPath, "gone")); !os.IsNotExist(err) {
		t.Errorf("repository is still in the parent: %v", err)
	}

	trashed := filepath.Join(env.trashDir, "gone-20261104T120000Z")
	if info, err := os.Stat(trashed); err != nil || !info.IsDir() {
		t.Errorf("trash entry %q missing: %v", trashed, err)
	}

	list := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos", "127.0.0.1:5555", bearer(admin), "")
	for _, item := range decodeBody[view.RepositoryList](t, list).Items {
		if item.Name == "gone" {
			t.Error("deleted repository is still listed")
		}
	}

	again := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/gone?confirm=gone", "127.0.0.1:5555", bearer(admin), "")
	if again.Code != http.StatusNotFound {
		t.Errorf("second delete status = %d, want 404", again.Code)
	}
}

func TestDeleteRepositoryDisabled(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, func(cfg *config.Config) {
		cfg.RepoDelete = false
	})
	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/repo1?confirm=repo1", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", recorder.Code, recorder.Body.String())
	}

	if _, err := os.Stat(filepath.Join(env.cfg.SVNParentPath, "repo1")); err != nil {
		t.Errorf("repository was moved although deletion is disabled: %v", err)
	}
}

func TestRepositoryTree(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	tree := decodeBody[view.RepositoryTree](t, recorder)
	if tree.Revision != 7 {
		t.Errorf("revision = %d, want HEAD 7", tree.Revision)
	}

	if tree.Total != 2 || len(tree.Entries) != 2 {
		t.Fatalf("tree = %+v", tree)
	}

	// Entries are sorted by name, so the escaped file comes before trunk and
	// the XML entity was decoded.
	if tree.Entries[0].Name != "README & notes.txt" || tree.Entries[0].Kind != "file" || tree.Entries[0].Size != 1024 {
		t.Errorf("first entry = %+v", tree.Entries[0])
	}

	if tree.Entries[1].Name != "trunk" || tree.Entries[1].Kind != "dir" {
		t.Errorf("second entry = %+v", tree.Entries[1])
	}

	if tree.Entries[0].Date == nil || tree.Entries[0].Author != "viewer" {
		t.Errorf("commit metadata = %+v", tree.Entries[0])
	}

	// Paging slices the sorted list.
	paged := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?size=1&page=2", "127.0.0.1:5555", bearer(admin), "")
	if paged.Code != http.StatusOK {
		t.Fatalf("paged status = %d", paged.Code)
	}

	page := decodeBody[view.RepositoryTree](t, paged)
	if page.Total != 2 || page.Size != 1 || page.Page != 2 || len(page.Entries) != 1 || page.Entries[0].Name != "trunk" {
		t.Errorf("page 2 = %+v", page)
	}

	// size above the cap is clamped to 1000, and the default is 100.
	capped := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?size=5000", "127.0.0.1:5555", bearer(admin), "")
	if capped.Code != http.StatusOK || decodeBody[view.RepositoryTree](t, capped).Size != 1000 {
		t.Errorf("size cap = %d (%s)", capped.Code, capped.Body.String())
	}

	defaulted := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree", "127.0.0.1:5555", bearer(admin), "")
	if decodeBody[view.RepositoryTree](t, defaulted).Size != 100 {
		t.Errorf("default size = %d, want 100", decodeBody[view.RepositoryTree](t, defaulted).Size)
	}

	// A revision is passed through as an argv value, never concatenated.
	byRev := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?rev=3", "127.0.0.1:5555", bearer(admin), "")
	if byRev.Code != http.StatusOK {
		t.Fatalf("rev=3 status = %d", byRev.Code)
	}

	lastList := lastCall(env.runner.snapshot(), "list")
	if lastList == nil || !strings.Contains(strings.Join(lastList.Args, " "), "-r 3") {
		t.Errorf("list args = %+v, want -r 3", lastList)
	}

	for _, rev := range []string{"0", "-1", "abc", "1.5"} {
		bad := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?rev="+rev, "127.0.0.1:5555", bearer(admin), "")
		if bad.Code != http.StatusUnprocessableEntity {
			t.Errorf("rev=%s status = %d, want 422", rev, bad.Code)
		}
	}
}

func TestRepositoryTreeEmptyDirectory(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	env.runner.set(func(r *fakeSvnRunner) {
		r.listXML = `<?xml version="1.0" encoding="UTF-8"?><lists><list path="file:///svn/repo1/trunk"/></lists>`
	})

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?path=trunk", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	tree := decodeBody[view.RepositoryTree](t, recorder)
	if tree.Path != "trunk" || tree.Total != 0 || len(tree.Entries) != 0 {
		t.Errorf("empty tree = %+v", tree)
	}
}

func TestRepositoryTreeNotFound(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	env.runner.set(func(r *fakeSvnRunner) {
		r.stderr = "svn: E160013: File not found: revision 7, path '/nope'\n"
		r.fail["list"] = &svncli.Error{Op: "list", Command: "svn", ExitCode: 1, Code: "E160013"}
	})

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree?path=nope", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %s)", recorder.Code, recorder.Body.String())
	}

	body := recorder.Body.String()
	if strings.Contains(body, "E160013") || strings.Contains(body, "File not found") {
		t.Errorf("the response leaked svn stderr: %s", body)
	}

	if strings.Contains(body, env.cfg.SVNParentPath) {
		t.Errorf("the response leaked a filesystem path: %s", body)
	}
}

func TestRepositoryInfo(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/info", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	info := decodeBody[view.Repository](t, recorder)
	if info.Name != "repo1" || info.HeadRevision != 7 {
		t.Errorf("info = %+v", info)
	}

	if info.LastChangedRev != 4 || info.LastChangedAuthor != "mixmedia" {
		t.Errorf("last changed = %+v", info)
	}

	if info.LastChangedAt == nil || !strings.HasPrefix(info.URL, "file://") {
		t.Errorf("info = %+v", info)
	}

	missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/nope/info", "127.0.0.1:5555", bearer(admin), "")
	if missing.Code != http.StatusNotFound {
		t.Errorf("missing repo status = %d, want 404", missing.Code)
	}
}

func TestRepositoryPermissions(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	// viewer holds the plain User role: repository:view only.
	viewer := env.mustToken(t, "viewer", testPassword)

	view := env.request(t, http.MethodGet, "/svnadmin/api/v1/repos/repo1/tree", "127.0.0.1:5555", bearer(viewer), "")
	if view.Code != http.StatusOK {
		t.Fatalf("view status = %d, body = %s", view.Code, view.Body.String())
	}

	create := env.request(t, http.MethodPost, "/svnadmin/api/v1/repos", "127.0.0.1:5555",
		bearer(viewer), `{"name":"sneaky"}`)
	if create.Code != http.StatusForbidden {
		t.Errorf("create status = %d, want 403", create.Code)
	}

	del := env.request(t, http.MethodDelete, "/svnadmin/api/v1/repos/repo1?confirm=repo1", "127.0.0.1:5555", bearer(viewer), "")
	if del.Code != http.StatusForbidden {
		t.Errorf("delete status = %d, want 403", del.Code)
	}
}

// lastCall returns the last recorded invocation of one svn operation.
func lastCall(calls []fakeCall, op string) *fakeCall {
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Name == "svn" && operation(calls[i].Args) == op {
			call := calls[i]

			return &call
		}
	}

	return nil
}
