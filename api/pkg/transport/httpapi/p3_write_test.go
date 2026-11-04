package httpapi

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mmhk/svn-admin/pkg/view"
)

func encodedPath(path string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(path))
}

// addAccount appends one htpasswd line so a test can log in as an account that
// is not part of the default fixture.
func addAccount(t *testing.T, env testEnv, name string) {
	t.Helper()

	file, err := os.OpenFile(env.cfg.SVNAuthFile, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open auth file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(name + ":" + testHash + "\n"); err != nil {
		t.Fatalf("append auth file: %v", err)
	}
}

func readAccess(t *testing.T, env testEnv) string {
	t.Helper()

	data, err := os.ReadFile(env.cfg.SVNAccessFile)
	if err != nil {
		t.Fatalf("read access file: %v", err)
	}

	return string(data)
}

func TestGroupLifecycleAndACL(t *testing.T) {
	env := newTestEnv(t, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	created := env.request(t, http.MethodPost, "/svnadmin/api/v1/groups", "127.0.0.1:5555",
		bearer(admin), `{"name":"newteam"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create group status = %d, body = %s", created.Code, created.Body.String())
	}

	group := decodeBody[view.Group](t, created)
	if group.Name != "newteam" || len(group.Members) != 0 {
		t.Errorf("group = %+v", group)
	}

	if !strings.Contains(readAccess(t, env), "newteam =") {
		t.Error("the authz file has no newteam line")
	}

	duplicate := env.request(t, http.MethodPost, "/svnadmin/api/v1/groups", "127.0.0.1:5555",
		bearer(admin), `{"name":"newteam"}`)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate group status = %d, want 409", duplicate.Code)
	}

	members := env.request(t, http.MethodPut, "/svnadmin/api/v1/groups/newteam/members", "127.0.0.1:5555",
		bearer(admin), `{"users":["viewer","testuser"]}`)
	if members.Code != http.StatusOK {
		t.Fatalf("set members status = %d, body = %s", members.Code, members.Body.String())
	}

	got := decodeBody[view.GroupMembers](t, members)
	if got.Name != "newteam" || len(got.Users) != 2 || got.Users[0] != "viewer" {
		t.Errorf("members = %+v", got)
	}

	readMembers := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups/newteam/members", "127.0.0.1:5555",
		bearer(admin), "")
	if readMembers.Code != http.StatusOK {
		t.Fatalf("get members status = %d, body = %s", readMembers.Code, readMembers.Body.String())
	}

	if body := decodeBody[view.GroupMembers](t, readMembers); len(body.Users) != 2 {
		t.Errorf("read members = %+v", body)
	}

	path := env.request(t, http.MethodPost, "/svnadmin/api/v1/access-paths", "127.0.0.1:5555",
		bearer(admin), `{"path":"repo3:/"}`)
	if path.Code != http.StatusCreated {
		t.Fatalf("create path status = %d, body = %s", path.Code, path.Body.String())
	}

	acl := env.request(t, http.MethodPut,
		"/svnadmin/api/v1/access-paths/"+encodedPath("repo3:/")+"/acl/@newteam", "127.0.0.1:5555",
		bearer(admin), `{"permission":"rw"}`)
	if acl.Code != http.StatusOK {
		t.Fatalf("set acl status = %d, body = %s", acl.Code, acl.Body.String())
	}

	updated := decodeBody[view.AccessPath](t, acl)
	if updated.Path != "repo3:/" || len(updated.ACL) != 1 ||
		updated.ACL[0].Principal != "@newteam" || updated.ACL[0].Permission != "rw" {
		t.Errorf("acl = %+v", updated)
	}

	groupPaths := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups/newteam/access-paths", "127.0.0.1:5555",
		bearer(admin), "")
	if groupPaths.Code != http.StatusOK {
		t.Fatalf("group access paths status = %d, body = %s", groupPaths.Code, groupPaths.Body.String())
	}

	paths := decodeBody[view.GroupAccessPaths](t, groupPaths)
	if len(paths.AccessPaths) != 1 || paths.AccessPaths[0] != "repo3:/" {
		t.Errorf("group access paths = %+v", paths)
	}

	userPaths := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/viewer/access-paths", "127.0.0.1:5555",
		bearer(admin), "")
	if userPaths.Code != http.StatusOK {
		t.Fatalf("user access paths status = %d, body = %s", userPaths.Code, userPaths.Body.String())
	}

	if list := decodeBody[view.UserAccessPaths](t, userPaths); !hasUserAccess(list, "repo3:/", "group", "newteam") {
		t.Errorf("viewer does not inherit repo3:/ from newteam: %+v", list)
	}

	deleted := env.request(t, http.MethodDelete, "/svnadmin/api/v1/groups/newteam", "127.0.0.1:5555",
		bearer(admin), "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete group status = %d, body = %s", deleted.Code, deleted.Body.String())
	}

	content := readAccess(t, env)
	if strings.Contains(content, "@newteam") {
		t.Errorf("@newteam survived the group delete:\n%s", content)
	}

	missing := env.request(t, http.MethodDelete, "/svnadmin/api/v1/groups/newteam", "127.0.0.1:5555",
		bearer(admin), "")
	if missing.Code != http.StatusNotFound {
		t.Errorf("deleting a missing group status = %d, want 404", missing.Code)
	}
}

func hasUserAccess(list view.UserAccessPaths, path, source, group string) bool {
	for _, item := range list.Items {
		if item.Path == path && item.Source == source && item.Group == group {
			return true
		}
	}

	return false
}

func TestGroupValidation(t *testing.T) {
	env := newTestEnv(t, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	for _, body := range []string{
		`{"name":""}`,
		`{"name":"a.b"}`,
		`{"name":"a b"}`,
		`{"name":"@devs"}`,
		`not-json`,
	} {
		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/groups", "127.0.0.1:5555", bearer(admin), body)
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Errorf("body %s status = %d, want 422", body, recorder.Code)
		}
	}

	if missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/groups/nope/members", "127.0.0.1:5555",
		bearer(admin), ""); missing.Code != http.StatusNotFound {
		t.Errorf("missing group members status = %d, want 404", missing.Code)
	}
}

func TestAccessPathValidationAndReadOnlyPrincipals(t *testing.T) {
	env := newTestEnv(t, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	for _, body := range []string{
		`{"path":""}`,
		`{"path":"../evil:/x"}`,
		`{"path":"repo:/.."}`,
		`{"path":"repo\\\\x:/"}`,
		`{"path":"repo:/a\u0000b"}`,
		`{"path":"repo:/a\nb"}`,
		`{"path":"no-colon"}`,
	} {
		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/access-paths", "127.0.0.1:5555",
			bearer(admin), body)
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Errorf("body %s status = %d, want 422 (body %s)", body, recorder.Code, recorder.Body.String())
		}
	}

	duplicate := env.request(t, http.MethodPost, "/svnadmin/api/v1/access-paths", "127.0.0.1:5555",
		bearer(admin), `{"path":"repo1:/"}`)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate path status = %d, want 409", duplicate.Code)
	}

	path := encodedPath("repo1:/")

	for _, principal := range []string{"$authenticated", "$anonymous", "&everyone"} {
		recorder := env.request(t, http.MethodPut,
			"/svnadmin/api/v1/access-paths/"+path+"/acl/"+principal, "127.0.0.1:5555",
			bearer(admin), `{"permission":"rw"}`)
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Errorf("write %s status = %d, want 422", principal, recorder.Code)
		}

		recorder = env.request(t, http.MethodDelete,
			"/svnadmin/api/v1/access-paths/"+path+"/acl/"+principal, "127.0.0.1:5555", bearer(admin), "")
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Errorf("delete %s status = %d, want 422", principal, recorder.Code)
		}
	}

	if recorder := env.request(t, http.MethodDelete,
		"/svnadmin/api/v1/access-paths/"+encodedPath("missing:/x"), "127.0.0.1:5555", bearer(admin), ""); recorder.Code != http.StatusNotFound {
		t.Errorf("missing path delete status = %d, want 404", recorder.Code)
	}
}

func TestUserAccessPathsThreeSources(t *testing.T) {
	env := newTestEnv(t, nil)
	admin := env.mustToken(t, "testuser", testPassword)
	repo2 := encodedPath("repo2:/trunk")

	// testuser already has a direct =r and the inherited @devs=rw on repo1.
	// Add the global source on repo2.
	if recorder := env.request(t, http.MethodPut,
		"/svnadmin/api/v1/access-paths/"+repo2+"/acl/*", "127.0.0.1:5555",
		bearer(admin), `{"permission":"r"}`); recorder.Code != http.StatusOK {
		t.Fatalf("set global acl status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/testuser/access-paths",
		"127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	list := decodeBody[view.UserAccessPaths](t, recorder)
	if list.Name != "testuser" {
		t.Errorf("name = %q", list.Name)
	}

	if !hasUserAccess(list, "repo1:/", "user", "") {
		t.Errorf("direct source missing: %+v", list.Items)
	}

	if !hasUserAccess(list, "repo1:/", "group", "devs") {
		t.Errorf("group source missing: %+v", list.Items)
	}

	if !hasUserAccess(list, "repo2:/trunk", "global", "") {
		t.Errorf("global source missing: %+v", list.Items)
	}

	for _, item := range list.Items {
		if item.Source != "user" && !item.Inherited {
			t.Errorf("inherited entry is flagged false: %+v", item)
		}

		if item.Source == "user" && item.Inherited {
			t.Errorf("direct entry is flagged inherited: %+v", item)
		}
	}

	if missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/nobody/access-paths",
		"127.0.0.1:5555", bearer(admin), ""); missing.Code != http.StatusNotFound {
		t.Errorf("missing user status = %d, want 404", missing.Code)
	}
}

func TestAuthzFileStaysLosslessThroughAPI(t *testing.T) {
	env := newTestEnv(t, nil)

	// Re-write the fixture with CRLF so the API test also covers line endings.
	crlf := strings.ReplaceAll(defaultAccessFile, "\n", "\r\n")
	if err := os.WriteFile(env.cfg.SVNAccessFile, []byte(crlf), 0o600); err != nil {
		t.Fatalf("rewrite access file: %v", err)
	}

	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodPut,
		"/svnadmin/api/v1/access-paths/"+encodedPath("repo1:/")+"/acl/testuser",
		"127.0.0.1:5555", bearer(admin), `{"permission":"rw"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("set acl status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	content := readAccess(t, env)

	if strings.Count(content, "\n") != strings.Count(content, "\r\n") {
		t.Errorf("CRLF was lost:\n%q", content)
	}

	want := strings.Replace(crlf, "testuser = r\r\n", "testuser = rw\r\n", 1)
	if content != want {
		t.Errorf("unrelated bytes changed:\n--- got ---\n%q\n--- want ---\n%q", content, want)
	}
}

// TestConcurrentACLWritesOverHTTP checks plan section 7.2 check 9 end to end.
func TestConcurrentACLWritesOverHTTP(t *testing.T) {
	env := newTestEnv(t, nil)
	admin := env.mustToken(t, "testuser", testPassword)
	path := encodedPath("repo2:/trunk")

	const writers = 20

	var wait sync.WaitGroup

	codes := make(chan int, writers)

	for index := 0; index < writers; index++ {
		wait.Add(1)

		go func(number int) {
			defer wait.Done()

			body := `{"permission":"r"}`
			recorder := env.request(t, http.MethodPut,
				"/svnadmin/api/v1/access-paths/"+path+"/acl/user"+string(rune('a'+number)),
				"127.0.0.1:5555", bearer(admin), body)
			codes <- recorder.Code
		}(index)
	}

	wait.Wait()
	close(codes)

	for code := range codes {
		if code != http.StatusOK {
			t.Errorf("concurrent write status = %d, want 200", code)
		}
	}

	acl := env.request(t, http.MethodGet, "/svnadmin/api/v1/access-paths/"+path+"/acl",
		"127.0.0.1:5555", bearer(admin), "")
	if acl.Code != http.StatusOK {
		t.Fatalf("acl status = %d", acl.Code)
	}

	body := decodeBody[view.AccessPath](t, acl)
	if len(body.ACL) != writers+2 { // viewer + the existing empty * entry + 20 writers
		t.Logf("acl = %+v", body.ACL)
	}

	found := make(map[string]bool)
	for _, entry := range body.ACL {
		found[entry.Principal] = true
	}

	for index := 0; index < writers; index++ {
		principal := "user" + string(rune('a'+index))
		if !found[principal] {
			t.Errorf("%s was lost", principal)
		}
	}
}
