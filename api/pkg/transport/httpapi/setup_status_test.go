package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mmhk/svn-admin/pkg/config"
	"github.com/mmhk/svn-admin/pkg/view"
)

func TestSetupStatusOnConfiguredDeployment(t *testing.T) {
	env := newTestEnv(t, nil)

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/setup/status", "127.0.0.1:5555", nil, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	status := decodeBody[view.SetupStatus](t, recorder)

	if status.SetupRequired || status.Reason != "" {
		t.Errorf("setupRequired/reason = %v/%q, want false/empty", status.SetupRequired, status.Reason)
	}

	if !status.AuthFile.Exists || status.AuthFile.UserCount != 3 {
		t.Errorf("authFile = %+v, want 3 accounts", status.AuthFile)
	}

	if !status.AdminFile.Exists || !status.AdminFile.Readable || !status.AdminFile.Writable {
		t.Errorf("adminFile = %+v, want exists/readable/writable", status.AdminFile)
	}

	if !status.CanCreateAccount {
		t.Error("canCreateAccount = false for a writable auth file")
	}

	if len(status.Users) != 0 {
		t.Errorf("users = %v, want empty once setup is done", status.Users)
	}

	if strings.Contains(recorder.Body.String(), env.dir) {
		t.Error("setup status leaks an absolute path")
	}
}

func TestSetupStatusReportsMissingFiles(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		cfg.SVNAuthFile = filepath.Join(cfg.SVNParentPath, "missing", "svn-auth-file")
		cfg.SVNAccessFile = filepath.Join(cfg.SVNParentPath, "missing", "svn-access-file")
	})

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/setup/status", "127.0.0.1:5555", nil, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	status := decodeBody[view.SetupStatus](t, recorder)

	if !status.SetupRequired || status.Reason != "missing_auth_file" {
		t.Errorf("setupRequired/reason = %v/%q, want true/missing_auth_file", status.SetupRequired, status.Reason)
	}

	if status.CanCreateAccount {
		t.Error("canCreateAccount = true when the auth directory does not exist")
	}

	if strings.Contains(recorder.Body.String(), env.dir) {
		t.Error("setup status leaks an absolute path")
	}
}

func TestSetupStatusReportsNoAdmin(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		// Keep the accounts but replace the marker with an empty file.
		emptyAdmin := filepath.Join(cfg.SVNParentPath, "empty-admin")
		if err := os.WriteFile(emptyAdmin, nil, 0o600); err != nil {
			t.Fatalf("write empty admin: %v", err)
		}
		cfg.AdminFile = emptyAdmin
	})

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/setup/status", "127.0.0.1:5555", nil, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	status := decodeBody[view.SetupStatus](t, recorder)

	if !status.SetupRequired || status.Reason != "no_admin" {
		t.Errorf("setupRequired/reason = %v/%q, want true/no_admin", status.SetupRequired, status.Reason)
	}

	if !status.AdminFile.Exists {
		t.Errorf("adminFile = %+v, want it to exist", status.AdminFile)
	}
}

func TestSetupStatusListsUsersWhileRequired(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		// Create an empty auth file so userCount == 0 → reason = "no_users".
		emptyAuth := filepath.Join(cfg.SVNParentPath, "empty-auth")
		if err := os.WriteFile(emptyAuth, nil, 0o600); err != nil {
			t.Fatalf("write empty auth: %v", err)
		}
		cfg.SVNAuthFile = emptyAuth
	})

	recorder := env.request(t, http.MethodGet, "/svnadmin/api/v1/setup/status", "127.0.0.1:5555", nil, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	status := decodeBody[view.SetupStatus](t, recorder)

	if !status.SetupRequired || status.Reason != "no_users" {
		t.Fatalf("setupRequired/reason = %v/%q, want true/no_users", status.SetupRequired, status.Reason)
	}

	if strings.Contains(recorder.Body.String(), "apr1") || strings.Contains(recorder.Body.String(), testHash) {
		t.Error("setup status leaks a password hash")
	}
}

func TestSetupAdminCreatesAccountAndFiles(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		cfg.SVNAuthFile = filepath.Join(cfg.SVNParentPath, "svn-auth-file")
		cfg.SVNAccessFile = filepath.Join(cfg.SVNParentPath, "svn-access-file")
	})

	body := `{"username":"admin","password":"Passw0rd!","token":"` + env.bootstrap + `"}`

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/setup/admin", "127.0.0.1:5555", nil, body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	result := decodeBody[view.SetupResult](t, recorder)
	if !result.AccountCreated {
		t.Errorf("result = %+v, want AccountCreated=true", result)
	}

	if len(result.InitializedFiles) != 2 {
		t.Errorf("initializedFiles = %v, want authFile, accessFile", result.InitializedFiles)
	}

	// The new account can sign in and the setup gate is now open.
	if login := env.login(t, "admin", "Passw0rd!"); login.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.Code, login.Body.String())
	}

	status := decodeBody[view.SetupStatus](t, env.request(
		t, http.MethodGet, "/svnadmin/api/v1/setup/status", "127.0.0.1:5555", nil, "",
	))

	if status.SetupRequired {
		t.Error("setupRequired is still true after bootstrap")
	}

	if len(status.Users) != 0 {
		t.Errorf("users = %v, want empty", status.Users)
	}
}

func TestSetupAdminRejectsMissingPassword(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		cfg.SVNAuthFile = filepath.Join(cfg.SVNParentPath, "empty-auth")
	})

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/setup/admin", "127.0.0.1:5555", nil,
		`{"username":"newuser","token":"`+env.bootstrap+`"}`)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body %s)", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "validation_error" {
		t.Errorf("error code = %q", envelope.Error.Code)
	}

	if !strings.Contains(envelope.Error.Message, "password") {
		t.Errorf("message = %q, want it to mention the password", envelope.Error.Message)
	}
}

func TestSetupAdminReportsStorageUnavailable(t *testing.T) {
	env := newTestEnv(t, func(cfg *config.Config) {
		cfg.SVNAuthFile = filepath.Join(cfg.SVNParentPath, "missing", "svn-auth-file")
		cfg.SVNAccessFile = filepath.Join(cfg.SVNParentPath, "missing", "svn-access-file")
	})

	recorder := env.request(t, http.MethodPost, "/svnadmin/api/v1/setup/admin", "127.0.0.1:5555", nil,
		`{"username":"admin","password":"Passw0rd!","token":"`+env.bootstrap+`"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body %s)", recorder.Code, recorder.Body.String())
	}

	envelope := decodeBody[view.ErrorEnvelope](t, recorder)
	if envelope.Error.Code != "storage_unavailable" {
		t.Errorf("error code = %q, want storage_unavailable", envelope.Error.Code)
	}

	if strings.Contains(envelope.Error.Message, env.dir) {
		t.Error("storage error leaks an absolute path")
	}
}
