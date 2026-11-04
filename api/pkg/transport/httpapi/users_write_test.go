package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/mmhk/svn-admin/pkg/store/htpasswd"
	"github.com/mmhk/svn-admin/pkg/view"
)

// fileHash returns the sha256 hex of a fixture file, used as If-Match.
func fileHash(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// credentialLine returns the htpasswd line of name, without its line ending.
func credentialLine(content, name string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, name+":") {
			return line
		}
	}

	return ""
}

func (env testEnv) rolesFile() string {
	return env.rolesPath
}

func TestCreateUser(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	token := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
		bearer(token), `{"username":"dave","password":"dave-secret"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	created := decodeBody[view.User](t, recorder)
	if created.Name != "dave" {
		t.Errorf("user = %+v", created)
	}

	if location := recorder.Header().Get("Location"); location != "/svnadmin/api/v1/users/dave" {
		t.Errorf("Location = %q", location)
	}

	if strings.Contains(recorder.Body.String(), "apr1") || strings.Contains(recorder.Body.String(), "dave-secret") {
		t.Error("the create response leaks the hash or the password")
	}

	// The new account can log in, which also proves the APR1 hash Apache will
	// read is valid.
	if login := env.login(t, "dave", "dave-secret"); login.Code != http.StatusOK {
		t.Fatalf("login as the new user = %d, body = %s", login.Code, login.Body.String())
	}

	// A duplicate is a conflict, not a silent overwrite.
	duplicate := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
		bearer(token), `{"username":"dave","password":"other"}`)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate status = %d, want 409", duplicate.Code)
	}

	if login := env.login(t, "dave", "other"); login.Code != http.StatusUnauthorized {
		t.Error("the duplicate request changed the password")
	}

	// The htpasswd line is APR1 and the other accounts are untouched.
	content, err := os.ReadFile(env.cfg.SVNAuthFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "dave:$apr1$") {
		t.Errorf("the new line is not APR1:\n%s", content)
	}

	if !strings.Contains(string(content), "viewer:"+testHash) {
		t.Error("an unrelated line changed")
	}
}

func TestCreateUserValidationAndForbidden(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	for _, body := range []string{
		`{"username":"","password":"x"}`,
		`{"username":"a b","password":"x"}`,
		`{"username":"a:b","password":"x"}`,
		`{"username":"dave","password":""}`,
		`{"username":"dave"}`,
		`not-json`,
	} {
		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(admin), body)
		if recorder.Code != http.StatusUnprocessableEntity {
			t.Errorf("body %s status = %d, want 422", body, recorder.Code)
		}
	}

	for _, user := range []string{"viewer", "norole"} {
		token := env.mustToken(t, user, testPassword)

		recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
			bearer(token), `{"username":"sneaky","password":"x"}`)
		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want 403 (body %s)", user, recorder.Code, recorder.Body.String())
		}
	}

	// No token at all is 401.
	if recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555", nil,
		`{"username":"x","password":"y"}`); recorder.Code != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", recorder.Code)
	}
}

func TestDeleteUserCascade(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	// Build a user that owns a group membership, a role and an ACL entry.
	if recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
		bearer(admin), `{"username":"temporary","password":"temp-secret"}`); recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/temporary/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":["devs"]}`); recorder.Code != http.StatusOK {
		t.Fatalf("groups status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/temporary/roles", "127.0.0.1:5555",
		bearer(admin), `{"roles":["User"]}`); recorder.Code != http.StatusOK {
		t.Fatalf("roles status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	accessBefore, err := os.ReadFile(env.cfg.SVNAccessFile)
	if err != nil {
		t.Fatalf("ReadFile(access) error = %v", err)
	}

	// Add an ACL entry by hand: the P3 endpoint does not exist yet, but the
	// delete cascade must strip it anyway.
	if err := os.WriteFile(env.cfg.SVNAccessFile,
		append(accessBefore, []byte("\n[repo3:/]\ntemporary = rw\n")...), 0o600); err != nil {
		t.Fatalf("WriteFile(access) error = %v", err)
	}

	recorder := env.request(t, http.MethodDelete, "/svnadmin/api/v1/users/temporary", "127.0.0.1:5555", bearer(admin), "")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if body := recorder.Body.String(); body != "" {
		t.Errorf("204 response has a body: %q", body)
	}

	// htpasswd
	authContent, err := os.ReadFile(env.cfg.SVNAuthFile)
	if err != nil {
		t.Fatalf("ReadFile(auth) error = %v", err)
	}

	if strings.Contains(string(authContent), "temporary:") {
		t.Errorf("the htpasswd line survived:\n%s", authContent)
	}

	// authz: groups and ACL
	accessContent, err := os.ReadFile(env.cfg.SVNAccessFile)
	if err != nil {
		t.Fatalf("ReadFile(access) error = %v", err)
	}

	if strings.Contains(string(accessContent), "temporary") {
		t.Errorf("the authz entry survived:\n%s", accessContent)
	}

	if !strings.Contains(string(accessContent), "devs = testuser, viewer") {
		t.Errorf("unrelated group members were lost:\n%s", accessContent)
	}

	// roles
	roleContent, err := os.ReadFile(env.rolesFile())
	if err != nil {
		t.Fatalf("ReadFile(roles) error = %v", err)
	}

	if strings.Contains(string(roleContent), "temporary") {
		t.Errorf("the role block survived:\n%s", roleContent)
	}

	if !strings.Contains(string(roleContent), "[testuser]") || !strings.Contains(string(roleContent), "[viewer]") {
		t.Errorf("unrelated role blocks were lost:\n%s", roleContent)
	}

	// The account is gone for the API too.
	if get := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/temporary", "127.0.0.1:5555", bearer(admin), ""); get.Code != http.StatusNotFound {
		t.Errorf("GET after delete = %d, want 404", get.Code)
	}

	// Deleted users must not be able to log in any more.
	if login := env.login(t, "temporary", "temp-secret"); login.Code != http.StatusUnauthorized {
		t.Errorf("login after delete = %d, want 401", login.Code)
	}

	// Deleting again is a 404.
	if again := env.request(t, http.MethodDelete, "/svnadmin/api/v1/users/temporary", "127.0.0.1:5555", bearer(admin), ""); again.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", again.Code)
	}

	// `*` is rejected before any store is touched.
	for _, path := range []string{"/svnadmin/api/v1/users/*", "/svnadmin/api/v1/users/%2A"} {
		if wildcard := env.request(t, http.MethodDelete, path, "127.0.0.1:5555", bearer(admin), ""); wildcard.Code != http.StatusUnprocessableEntity {
			t.Errorf("delete %s = %d, want 422", path, wildcard.Code)
		}
	}

	// A non-privileged token cannot delete.
	viewer := env.mustToken(t, "viewer", testPassword)

	if forbidden := env.request(t, http.MethodDelete, "/svnadmin/api/v1/users/viewer", "127.0.0.1:5555", bearer(viewer), ""); forbidden.Code != http.StatusForbidden {
		t.Errorf("viewer delete = %d, want 403", forbidden.Code)
	}
}

func TestPasswordChangeRevokesTokens(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	if recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
		bearer(admin), `{"username":"roger","password":"old-secret"}`); recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d", recorder.Code)
	}

	// roger has no roles yet: he cannot even rotate his own password. Grant
	// the plain User role, which is what user:changepass is scoped to.
	if recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/roger/roles", "127.0.0.1:5555",
		bearer(admin), `{"roles":["User"]}`); recorder.Code != http.StatusOK {
		t.Fatalf("grant role status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	userToken := env.mustToken(t, "roger", "old-secret")

	// A plain user cannot reset somebody else's password.
	if forbidden := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/password", "127.0.0.1:5555",
		bearer(userToken), `{"password":"hijacked"}`); forbidden.Code != http.StatusForbidden {
		t.Errorf("cross-user password change = %d, want 403", forbidden.Code)
	}

	// An administrator can.
	recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/roger/password", "127.0.0.1:5555",
		bearer(admin), `{"password":"new-secret"}`)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("password change = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	// Every token of the target is revoked, the old password stops working and
	// the new one works.
	if stale := env.request(t, http.MethodGet, "/svnadmin/api/v1/session", "127.0.0.1:5555", bearer(userToken), ""); stale.Code != http.StatusUnauthorized {
		t.Errorf("old token = %d, want 401", stale.Code)
	}

	if login := env.login(t, "roger", "old-secret"); login.Code != http.StatusUnauthorized {
		t.Errorf("old password login = %d, want 401", login.Code)
	}

	fresh := env.mustToken(t, "roger", "new-secret")

	// Self-service: roger rotates his own password and his token dies too.
	self := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/roger/password", "127.0.0.1:5555",
		bearer(fresh), `{"password":"third-secret"}`)
	if self.Code != http.StatusNoContent {
		t.Fatalf("self password change = %d, body = %s", self.Code, self.Body.String())
	}

	if stale := env.request(t, http.MethodGet, "/svnadmin/api/v1/session", "127.0.0.1:5555", bearer(fresh), ""); stale.Code != http.StatusUnauthorized {
		t.Errorf("self-changed token = %d, want 401", stale.Code)
	}

	env.mustToken(t, "roger", "third-secret")

	// Unknown account and weak password.
	if missing := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/nobody/password", "127.0.0.1:5555",
		bearer(admin), `{"password":"whatever"}`); missing.Code != http.StatusNotFound {
		t.Errorf("unknown account = %d, want 404", missing.Code)
	}

	for _, body := range []string{`{"password":""}`, `{"password":"line\nbreak"}`} {
		if weak := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/password", "127.0.0.1:5555",
			bearer(admin), body); weak.Code != http.StatusUnprocessableEntity {
			t.Errorf("body %q status = %d, want 422", body, weak.Code)
		}
	}

	// testuser's own token must still work (it was never the target).
	if ok := env.request(t, http.MethodGet, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(admin), ""); ok.Code != http.StatusOK {
		t.Errorf("admin token = %d, want 200", ok.Code)
	}
}

func TestSetRolesRoundTrip(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	before, err := os.ReadFile(env.rolesFile())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		bearer(admin), `{"roles":["User","Repository-Creator"]}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	roles := decodeBody[view.UserRoles](t, recorder)
	if len(roles.Roles) != 2 || roles.Roles[0] != "User" || roles.Roles[1] != "Repository-Creator" {
		t.Errorf("roles = %v", roles.Roles)
	}

	// The user detail reflects the change.
	detail := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/viewer", "127.0.0.1:5555", bearer(admin), "")
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d", detail.Code)
	}

	if user := decodeBody[view.User](t, detail); len(user.Roles) != 2 {
		t.Errorf("user roles = %v", user.Roles)
	}

	// Other accounts' lines are byte identical.
	after, err := os.ReadFile(env.rolesFile())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	for _, line := range []string{"[testuser]\nAdministrator=", "[viewer]\n"} {
		if !strings.Contains(string(before), line) || !strings.Contains(string(after), line) {
			t.Errorf("line %q changed", line)
		}
	}

	if !strings.Contains(string(after), "[testuser]\nAdministrator=\n") {
		t.Errorf("the unrelated block was rewritten:\n%s", after)
	}

	// Reading it back through the dedicated endpoint.
	read := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555", bearer(admin), "")
	if read.Code != http.StatusOK {
		t.Fatalf("GET roles = %d", read.Code)
	}

	if got := decodeBody[view.UserRoles](t, read); len(got.Roles) != 2 {
		t.Errorf("GET roles = %v", got.Roles)
	}

	// Unknown role and non-assigner are rejected.
	if invalid := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		bearer(admin), `{"roles":["Superuser"]}`); invalid.Code != http.StatusUnprocessableEntity {
		t.Errorf("unknown role = %d, want 422", invalid.Code)
	}

	viewer := env.mustToken(t, "viewer", testPassword)

	if forbidden := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		bearer(viewer), `{"roles":["Administrator"]}`); forbidden.Code != http.StatusForbidden {
		t.Errorf("viewer assigning roles = %d, want 403", forbidden.Code)
	}

	// An empty list clears the roles but keeps the block.
	cleared := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		bearer(admin), `{"roles":[]}`)
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear roles = %d, body = %s", cleared.Code, cleared.Body.String())
	}

	if got := decodeBody[view.UserRoles](t, cleared); len(got.Roles) != 0 {
		t.Errorf("roles after clear = %v", got.Roles)
	}
}

func TestSetGroupsRoundTrip(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":["devs"]}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	groups := decodeBody[view.UserGroups](t, recorder)
	if len(groups.Groups) != 1 || groups.Groups[0] != "devs" {
		t.Errorf("groups = %v", groups.Groups)
	}

	// Adding a second account to the same group keeps the first.
	if second := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/testuser/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":["devs"]}`); second.Code != http.StatusOK {
		t.Fatalf("second membership = %d, body = %s", second.Code, second.Body.String())
	}

	content, err := os.ReadFile(env.cfg.SVNAccessFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(content), "devs = testuser, viewer") {
		t.Errorf("the group line = %q", content)
	}

	// The ACL sections are untouched.
	if !strings.Contains(string(content), "[repo1:/]\n@devs = rw\ntestuser = r\n") {
		t.Errorf("the ACL section changed:\n%s", content)
	}

	// Unknown group, bad group name, non-assigner.
	if unknown := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":["nope"]}`); unknown.Code != http.StatusUnprocessableEntity {
		t.Errorf("unknown group = %d, want 422", unknown.Code)
	}

	if bad := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":["a b"]}`); bad.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad group name = %d, want 422", bad.Code)
	}

	viewer := env.mustToken(t, "viewer", testPassword)

	if forbidden := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		bearer(viewer), `{"groups":["devs"]}`); forbidden.Code != http.StatusForbidden {
		t.Errorf("viewer assigning groups = %d, want 403", forbidden.Code)
	}

	// Clearing the list removes only that member.
	if cleared := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		bearer(admin), `{"groups":[]}`); cleared.Code != http.StatusOK {
		t.Errorf("clear groups = %d", cleared.Code)
	}

	content, err = os.ReadFile(env.cfg.SVNAccessFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Only the [groups] line loses the member; the unrelated ACL entry that
	// names the same account stays untouched.
	if strings.Contains(string(content), "devs = testuser, viewer") || strings.Contains(string(content), "devs = viewer") {
		t.Errorf("viewer is still a member of devs:\n%s", content)
	}

	if !strings.Contains(string(content), "devs = testuser") {
		t.Errorf("the other member was removed too:\n%s", content)
	}

	if !strings.Contains(string(content), "[repo2:/trunk]\nviewer = r") {
		t.Errorf("the ACL entry changed:\n%s", content)
	}

	if missing := env.request(t, http.MethodGet, "/svnadmin/api/v1/users/nobody/groups", "127.0.0.1:5555", bearer(admin), ""); missing.Code != http.StatusNotFound {
		t.Errorf("unknown user = %d, want 404", missing.Code)
	}
}

func TestConcurrentWrites(t *testing.T) {
	t.Run("different users", func(t *testing.T) {
		env := newTestEnv(t, defaultRoleFile, nil)
		admin := env.mustToken(t, "testuser", testPassword)

		const count = 10

		var (
			wait   sync.WaitGroup
			mu     sync.Mutex
			status []int
		)

		for i := 0; i < count; i++ {
			wait.Add(1)

			go func(index int) {
				defer wait.Done()

				body := fmt.Sprintf(`{"username":"bulk%d","password":"bulk-secret"}`, index)
				recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555", bearer(admin), body)

				mu.Lock()
				status = append(status, recorder.Code)
				mu.Unlock()
			}(i)
		}

		wait.Wait()

		for index, code := range status {
			if code != http.StatusCreated {
				t.Errorf("create %d = %d, want 201", index, code)
			}
		}

		content, err := os.ReadFile(env.cfg.SVNAuthFile)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}

		for i := 0; i < count; i++ {
			if !strings.Contains(string(content), fmt.Sprintf("bulk%d:", i)) {
				t.Errorf("bulk%d is missing from the credentials file", i)
			}
		}

		// Every account must still verify: no line was corrupted. This checks
		// the hashes directly because the login rate limiter is per IP and
		// this test logs in ten times from the same address.
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("bulk%d", i)

			entry := credentialLine(string(content), name)
			if entry == "" {
				t.Errorf("%s has no credentials line", name)

				continue
			}

			hash := strings.TrimPrefix(entry, name+":")
			if err := htpasswd.Verify("bulk-secret", hash); err != nil {
				t.Errorf("Verify(%s) error = %v", name, err)
			}
		}
	})

	t.Run("same user with If-Match", func(t *testing.T) {
		env := newTestEnv(t, defaultRoleFile, nil)
		admin := env.mustToken(t, "testuser", testPassword)

		if recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
			bearer(admin), `{"username":"sam","password":"sam-secret"}`); recorder.Code != http.StatusCreated {
			t.Fatalf("create status = %d", recorder.Code)
		}

		hash := fileHash(t, env.rolesFile())
		headers := bearer(admin)
		headers["If-Match"] = hash

		bodies := []string{`{"roles":["User"]}`, `{"roles":["Repository-Creator"]}`}

		var (
			wait   sync.WaitGroup
			mu     sync.Mutex
			codes  []int
			values []string
		)

		for _, body := range bodies {
			wait.Add(1)

			go func(payload string) {
				defer wait.Done()

				recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/sam/roles", "127.0.0.1:5555", headers, payload)

				mu.Lock()
				codes = append(codes, recorder.Code)
				values = append(values, recorder.Body.String())
				mu.Unlock()
			}(body)
		}

		wait.Wait()

		successes := 0
		conflicts := 0

		for index, code := range codes {
			switch code {
			case http.StatusOK:
				successes++
			case http.StatusConflict:
				conflicts++
			default:
				t.Errorf("request %d = %d (%s)", index, code, values[index])
			}
		}

		if successes != 1 || conflicts != 1 {
			t.Fatalf("successes/conflicts = %d/%d, want 1/1 (codes %v)", successes, conflicts, codes)
		}
	})
}

func TestIfMatchPrecondition(t *testing.T) {
	env := newTestEnv(t, defaultRoleFile, nil)
	admin := env.mustToken(t, "testuser", testPassword)

	stale := strings.Repeat("0", 64)
	headers := bearer(admin)
	headers["If-Match"] = stale

	before, err := os.ReadFile(env.rolesFile())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	roles := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		headers, `{"roles":["Administrator"]}`)
	if roles.Code != http.StatusConflict {
		t.Errorf("stale If-Match roles = %d, want 409 (body %s)", roles.Code, roles.Body.String())
	}

	after, err := os.ReadFile(env.rolesFile())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(after) != string(before) {
		t.Error("a stale If-Match still modified the role file")
	}

	groups := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/groups", "127.0.0.1:5555",
		headers, `{"groups":["devs"]}`)
	if groups.Code != http.StatusConflict {
		t.Errorf("stale If-Match groups = %d, want 409", groups.Code)
	}

	password := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/password", "127.0.0.1:5555",
		headers, `{"password":"nope"}`)
	if password.Code != http.StatusConflict {
		t.Errorf("stale If-Match password = %d, want 409", password.Code)
	}

	// A malformed precondition is rejected instead of being ignored.
	bad := bearer(admin)
	bad["If-Match"] = "not-a-hash"

	if recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		bad, `{"roles":["User"]}`); recorder.Code != http.StatusUnprocessableEntity {
		t.Errorf("malformed If-Match = %d, want 422", recorder.Code)
	}

	// The correct hash is accepted.
	fresh := bearer(admin)
	fresh["If-Match"] = fileHash(t, env.rolesFile())

	if recorder := env.request(t, http.MethodPut, "/svnadmin/api/v1/users/viewer/roles", "127.0.0.1:5555",
		fresh, `{"roles":["User"]}`); recorder.Code != http.StatusOK {
		t.Errorf("matching If-Match = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
}

func TestWriteAuditLog(t *testing.T) {
	var logs bytes.Buffer

	env := newTestEnvWithLogs(t, defaultRoleFile, nil, &logs)

	admin := env.mustToken(t, "testuser", testPassword)

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/users", "127.0.0.1:5555",
		bearer(admin), `{"username":"audited","password":"audit-secret"}`)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	// A failed login must be recorded without the password.
	if login := env.login(t, "audited", "wrong-secret"); login.Code != http.StatusUnauthorized {
		t.Fatalf("login status = %d", login.Code)
	}

	output := logs.String()

	for _, field := range []string{
		`"action":"user.create"`,
		`"actor":"testuser"`,
		`"target":"audited"`,
		`"result":"success"`,
		`"action":"session.login"`,
		`"result":"failure"`,
	} {
		if !strings.Contains(output, field) {
			t.Errorf("audit log is missing %s:\n%s", field, output)
		}
	}

	for _, secret := range []string{"audit-secret", "wrong-secret", "apr1", admin} {
		if strings.Contains(output, secret) {
			t.Errorf("audit log leaks %q:\n%s", secret, output)
		}
	}
}
